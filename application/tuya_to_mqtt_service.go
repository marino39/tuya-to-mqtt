package application

import (
	"context"
	"encoding/json"
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

		return t.params.TuyaPulsarClient.Subscribe(ctx, func(ctx context.Context, msg *Message) error {
			msgData, err := json.Marshal(msg)
			if err != nil {
				return err
			}
			return t.params.MQTTClient.Publish(t.params.MQTTTopic, 0, true, msgData)
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

					msgPerMin := uint64(0)
					if timeDiff != 0 {
						msgPerMin = (msgCountDiff * 60) / uint64(timeDiff)
					}

					t.log.Info().
						Uint64("mqtt_msg_per_min", msgPerMin).
						Uint64("mqtt_msg_total", newStatus.MessageCount).
						Bool("mqtt_is_connected", newStatus.Connected).
						Time("mqtt_last_published", newStatus.LastTimePublished).
						Msg("mqtt publish report")
				}
				lastStatus = newStatus
			}
		}

		return nil
	})

	return g.Wait()
}
