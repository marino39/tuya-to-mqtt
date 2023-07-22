package application

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/sourcegraph/conc/pool"
	"golang.org/x/sync/errgroup"
)

type TuyaToMQTTService interface {
	Run(ctx context.Context) error
}

type TuyaToMQTTServiceParams struct {
	TuyaClient       TuyaClient
	TuyaPulsarClient TuyaPulsarClient
	TuyaUser         string
	MQTTClient       MQTTClient

	MQTTTopic string

	Log zerolog.Logger
}

type tuyaToMQTTService struct {
	params TuyaToMQTTServiceParams

	mqttCurrentState      map[string]string
	mqttCurrentStateMutex sync.RWMutex

	log zerolog.Logger
}

func NewTuyaToMQTTService(params TuyaToMQTTServiceParams) (TuyaToMQTTService, error) {
	if params.TuyaPulsarClient == nil {
		return nil, fmt.Errorf("TuyaPulsarClient is nil")
	}
	if params.MQTTClient == nil {
		return nil, fmt.Errorf("MQTTClient is nil")
	}
	return &tuyaToMQTTService{params: params, mqttCurrentState: make(map[string]string), log: params.Log}, nil
}

func (t *tuyaToMQTTService) Run(ctx context.Context) error {
	g := errgroup.Group{}

	rCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	initWaitGroup := sync.WaitGroup{}
	initWaitGroup.Add(1)

	// tuya load mqtt current state
	g.Go(func() error {
		err := t.params.MQTTClient.Subscribe(fmt.Sprintf("%s/#", t.params.MQTTTopic), 2, func(msg MQTTMessage) {
			t.mqttCurrentStateMutex.Lock()
			t.mqttCurrentState[msg.Topic()] = string(msg.Payload())
			t.mqttCurrentStateMutex.Unlock()
			msg.Ack()
		})
		if err != nil {
			return err
		}
		return nil
	})

	// tuya message handler
	g.Go(func() error {
		t.log.Info().Msgf("start publishing on topic: %s", t.params.MQTTTopic)
		defer t.log.Info().Msg("stop publishing")
		defer cancel()

		task := pool.New().WithContext(rCtx).WithMaxGoroutines(10)
		defer func() { _ = task.Wait() }()

		return t.params.TuyaPulsarClient.Subscribe(rCtx, func(ctx context.Context, msg *Message) error {
			task.Go(func(ctx context.Context) error {
				initWaitGroup.Wait()

				switch msg.Type {
				case MessageTypeStatus:
					return t.handleMessageStatus(msg)
				case MessageTypeDeviceManagment:
					return t.handleMessageDeviceManagement(msg)
				default:
					t.log.Warn().Fields(map[string]any{"msg": msg}).Msg("unknown message")
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

	// get current device status
	devices, err := t.params.TuyaClient.UserDevices(ctx, t.params.TuyaUser)
	if err != nil {
		return err
	}

	for _, device := range devices {
		var statusList []Status
		statusList, err = t.params.TuyaClient.DeviceStatus(ctx, device.ID)
		if err != nil {
			return err
		}

		for _, status := range statusList {
			// build topic and value
			topic := BuildMQTTTopicForStatusRaw(
				t.params.MQTTTopic, "tuya", device.ProductID, device.ID, status.Code,
			)
			value := strings.ReplaceAll(fmt.Sprintf("%#v", status.Value), "\"", "")

			// publish
			err := t.handlePublish(topic, value)
			if err != nil {
				return err
			}
		}
	}

	initWaitGroup.Done()

	return g.Wait()
}

func (t *tuyaToMQTTService) handleMessageStatus(msg *Message) error {
	for _, status := range msg.Status {
		// build topic and value
		topic := BuildMQTTTopicForStatus(t.params.MQTTTopic, msg, status)
		value := strings.ReplaceAll(fmt.Sprintf("%#v", status.Value), "\"", "")

		// publish
		err := t.handlePublish(topic, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *tuyaToMQTTService) handleMessageDeviceManagement(msg *Message) error {
	switch msg.BizCode {
	case "nameUpdate":
		if msg.BizData != nil && msg.BizData["name"] != nil {
			if value, ok := msg.BizData["name"].(string); ok {
				// build topic
				topic := BuildMQTTTopicForStatusProperty(t.params.MQTTTopic, msg, "name")

				// publish
				err := t.handlePublish(topic, value)
				if err != nil {
					return err
				}
			}
		}
	case "online", "offline":
		// build topic and value
		topic := BuildMQTTTopicForStatusProperty(t.params.MQTTTopic, msg, "network")
		value := msg.BizCode

		// publish
		err := t.handlePublish(topic, value)
		if err != nil {
			return err
		}
	default:
		t.log.Warn().Fields(map[string]any{"msg": msg}).Msg("unknown message")
	}
	return nil
}

func (t *tuyaToMQTTService) handlePublish(topic, value string) error {
	if value == "" {
		return nil
	}

	// check if value is the same as last time
	t.mqttCurrentStateMutex.RLock()
	currentState, ok := t.mqttCurrentState[topic]
	t.mqttCurrentStateMutex.RUnlock()

	if ok && currentState == value {
		return nil
	}

	// publish
	t.log.Debug().Str("topic", topic).Str("value", value).Msg("publishing")
	err := t.params.MQTTClient.Publish(topic, 2, true, []byte(value))
	if err != nil {
		return err
	}
	return nil
}

func BuildMQTTTopicForStatus(prefix string, msg *Message, status Status) string {
	return fmt.Sprintf("%s/tuya/%s/%s/status/%s", prefix, msg.ProductKey, msg.DevID, status.Code)
}

func BuildMQTTTopicForStatusProperty(prefix string, msg *Message, propertyName string) string {
	return fmt.Sprintf("%s/tuya/%s/%s/status/%s", prefix, msg.ProductKey, msg.DevID, propertyName)
}

func BuildMQTTTopicForStatusRaw(prefix, company, product, device, propertyName string) string {
	return fmt.Sprintf("%s/%s/%s/%s/status/%s", prefix, company, product, device, propertyName)
}
