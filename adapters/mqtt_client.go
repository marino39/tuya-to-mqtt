package adapters

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"tuya-to-mqtt/application"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog"
)

const (
	MQTTDefaultConnectTimeout = 30 * time.Second
	MQTTDefaultPublishTimeout = 5 * time.Second
)

var (
	ErrMQTTNotConnected   = fmt.Errorf("not connected")
	ErrMQTTConnectTimeout = fmt.Errorf("connect timeout")
	ErrMQTTPublishTimeout = fmt.Errorf("publish timeout")
)

type MQTTClientParams struct {
	ClientID string
	Username string
	Password string
	MQTTUrl  string

	ConnectTimeout time.Duration
	PublishTimeout time.Duration

	NewClientFunc func(options *mqtt.ClientOptions) mqtt.Client

	Log zerolog.Logger
}

func (m *MQTTClientParams) EnsureDefaults() {
	if m.ConnectTimeout == 0 {
		m.ConnectTimeout = MQTTDefaultConnectTimeout
	}

	if m.PublishTimeout == 0 {
		m.PublishTimeout = MQTTDefaultPublishTimeout
	}

	if m.NewClientFunc == nil {
		m.NewClientFunc = mqtt.NewClient
	}
}

type MQTTClient struct {
	params MQTTClientParams

	client mqtt.Client

	connected          uint64
	msgCount           uint64
	msgCountUpdateTime atomic.Pointer[time.Time]

	mu sync.RWMutex

	log zerolog.Logger
}

func NewMQTTClient(params MQTTClientParams) *MQTTClient {
	params.EnsureDefaults()

	m := &MQTTClient{params: params, log: params.Log}
	m.client = m.newMqttClient()

	t := time.Unix(0, 0)
	m.msgCountUpdateTime.Store(&t)

	return m
}

func (m *MQTTClient) Connect() error {
	if atomic.LoadUint64(&m.connected) == 1 {
		return nil
	}

	tc := time.NewTimer(m.params.ConnectTimeout)

	token := m.client.Connect()
	select {
	case <-tc.C:
		return ErrMQTTConnectTimeout
	case <-token.Done():
		if token.Error() != nil {
			return token.Error()
		}
	}

	atomic.StoreUint64(&m.connected, 1)
	return nil
}

func (m *MQTTClient) IsConnected() bool {
	if atomic.LoadUint64(&m.connected) == 0 {
		return false
	}
	return true
}

func (m *MQTTClient) Status() application.MQTTStatus {
	return application.MQTTStatus{
		MessageCount:      atomic.LoadUint64(&m.msgCount),
		LastTimePublished: *m.msgCountUpdateTime.Load(),
		Connected:         m.IsConnected(),
	}
}

func (m *MQTTClient) Publish(topic string, qos byte, retained bool, msg any) error {
	if !m.IsConnected() {
		return ErrMQTTNotConnected
	}

	tc := time.NewTimer(m.params.PublishTimeout)

	token := m.client.Publish(topic, qos, retained, msg)
	select {
	case <-tc.C:
		return ErrMQTTPublishTimeout
	case <-token.Done():
		if token.Error() != nil {
			return token.Error()
		}
	}

	t := time.Now()
	m.msgCountUpdateTime.Store(&t)
	atomic.AddUint64(&m.msgCount, 1)
	return nil
}

func (m *MQTTClient) Subscribe(topic string, qos byte, handler func(msg application.MQTTMessage)) error {
	token := m.client.Subscribe(topic, qos, func(client mqtt.Client, msg mqtt.Message) {
		handler(msg)
	})

	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (m *MQTTClient) AddRoute(topic string, handler func(msg application.MQTTMessage)) error {
	m.client.AddRoute(topic, func(client mqtt.Client, msg mqtt.Message) {
		handler(msg)
	})
	return nil
}

func (m *MQTTClient) PublishHandler(client mqtt.Client, msg mqtt.Message) {
	// do nothing
}

func (m *MQTTClient) OnConnect(client mqtt.Client) {
	m.log.Info().Msgf("connected")
	atomic.StoreUint64(&m.connected, 1)
}

func (m *MQTTClient) OnConnectionLost(client mqtt.Client, err error) {
	m.log.Info().Msgf("connect lost: %v", err)
	atomic.StoreUint64(&m.connected, 0)
}

func (m *MQTTClient) newMqttClient() mqtt.Client {
	opts := mqtt.NewClientOptions()

	opts.AddBroker(m.params.MQTTUrl)
	opts.SetClientID(m.params.ClientID)
	opts.SetUsername(m.params.Username)
	opts.SetPassword(m.params.Password)

	opts.SetDefaultPublishHandler(m.PublishHandler)
	opts.OnConnect = m.OnConnect
	opts.OnConnectionLost = m.OnConnectionLost

	return m.params.NewClientFunc(opts)
}

var _ application.MQTTClient = &MQTTClient{}
