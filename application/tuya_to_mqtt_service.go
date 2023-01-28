package application

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
		return t.params.TuyaPulsarClient.Subscribe(ctx, func(ctx context.Context, m *Message) error {
			return t.params.MQTTClient.Publish(t.params.MQTTTopic, 0, false, m)
		})
	})

	// mqtt publish reported
	g.Go(func() error {
		ticker := time.NewTicker(30 * time.Second)

	ReporterLoop:
		for {
			var lastStatus MQTTStatus
			select {
			case <-ctx.Done():
				break ReporterLoop
			case <-ticker.C:
				newStatus := t.params.MQTTClient.Status()
				if lastStatus.MessageCount != 0 {
					msgCountDiff := newStatus.MessageCount - lastStatus.MessageCount
					timeDiff := newStatus.LastTimePublished.Unix() - lastStatus.LastTimePublished.Unix()

					log.Info().
						Uint64("msgs_per_sec", msgCountDiff/uint64(timeDiff)).
						Bool("is_connected", newStatus.Connected).
						Time("last_time_published", newStatus.LastTimePublished)
				}
				lastStatus = newStatus
			}
		}

		return nil
	})

	return g.Wait()
}
