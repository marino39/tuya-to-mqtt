package application

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type TuyaToMQTTService interface {
	Run(ctx context.Context) error
}

type TuyaToMQTTServiceParams struct {
	TuyaPulsarClient TuyaPulsarClient
	MQTTClient       MQTTClient

	MQTTTopic string

	Log zerolog.Logger
}

type tuyaToMQTTService struct {
	params TuyaToMQTTServiceParams

	log zerolog.Logger
}

func NewTuyaToMQTTService(params TuyaToMQTTServiceParams) (TuyaToMQTTService, error) {
	if params.TuyaPulsarClient == nil {
		return nil, fmt.Errorf("TuyaPulsarClient is nil")
	}
	if params.MQTTClient == nil {
		return nil, fmt.Errorf("MQTTClient is nil")
	}
	return &tuyaToMQTTService{params: params, log: params.Log}, nil
}

func (t tuyaToMQTTService) Run(ctx context.Context) error {
	g := errgroup.Group{}

	// tuya message handler
	g.Go(func() error {
		t.log.Info().Msgf("start publishing on topic: %s", t.params.MQTTTopic)
		defer t.log.Info().Msg("stop publishing")

		return t.params.TuyaPulsarClient.Subscribe(ctx, func(ctx context.Context, m *Message) error {
			return t.params.MQTTClient.Publish(t.params.MQTTTopic, 0, true, m)
		})
	})

	// mqtt publish reported
	g.Go(func() error {
		ticker := time.NewTicker(30 * time.Second)
		lastStatus := MQTTStatus{}

	ReporterLoop:
		for {
			select {
			case <-ctx.Done():
				break ReporterLoop
			case <-ticker.C:
				newStatus := t.params.MQTTClient.Status()
				if lastStatus.Connected != false {
					msgCountDiff := newStatus.MessageCount - lastStatus.MessageCount
					timeDiff := newStatus.LastTimePublished.Unix() - lastStatus.LastTimePublished.Unix()

					msgPerSec := uint64(0)
					if timeDiff != 0 {
						msgPerSec = msgCountDiff / uint64(timeDiff*60)
					}

					t.log.Info().
						Uint64("msg_per_min", msgPerSec).
						Bool("is_connected", newStatus.Connected).
						Time("last_time_published", newStatus.LastTimePublished).
						Msg("publish report")
				}
				lastStatus = newStatus
			}
		}

		return nil
	})

	return g.Wait()
}
