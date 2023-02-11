package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sourcegraph/conc/pool"
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

	rCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// tuya message handler
	g.Go(func() error {
		t.log.Info().Msgf("start publishing on topic: %s", t.params.MQTTTopic)
		defer t.log.Info().Msg("stop publishing")
		defer cancel()

		task := pool.New().WithContext(rCtx).WithMaxGoroutines(10)
		defer func() { _ = task.Wait() }()

		return t.params.TuyaPulsarClient.Subscribe(rCtx, func(ctx context.Context, msg *Message) error {
			task.Go(func(ctx context.Context) error {
				switch msg.Type {
				case MessageTypeStatus:
					for _, status := range msg.Status {
						topic := BuildMQTTTopicForStatus(t.params.MQTTTopic, msg, status)
						value := strings.ReplaceAll(fmt.Sprintf("%#v", status.Value), "\"", "")
						err := t.params.MQTTClient.Publish(topic, 2, true, []byte(value))
						if err != nil {
							return err
						}
					}
				case MessageTypeNameModify:
					if msg.BizData != nil && msg.BizData["name"] != nil {
						if value, ok := msg.BizData["name"].(string); ok {
							topic := BuildMQTTTopicForNameChange(t.params.MQTTTopic, msg)
							err := t.params.MQTTClient.Publish(topic, 2, true, []byte(value))
							if err != nil {
								return err
							}
						}
					}
				default:
					log.Warn().Fields(map[string]any{"msg": msg}).Msg("Unknown message")
				}
				return nil
			})
			return nil
		})
	})

	// mqtt publish reported
	g.Go(func() error {
		ticker := time.NewTicker(60 * time.Second)
		lastStatus := MQTTStatus{}

	ReporterLoop:
		for {
			select {
			case <-rCtx.Done():
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

func BuildMQTTTopicForStatus(prefix string, msg *Message, status Status) string {
	return fmt.Sprintf("%s/tuya/%s/%s/status/%s", prefix, msg.ProductKey, msg.DevID, status.Code)
}

func BuildMQTTTopicForNameChange(prefix string, msg *Message) string {
	return fmt.Sprintf("%s/tuya/%s/%s/status/name", prefix, msg.ProductKey, msg.DevID)
}
