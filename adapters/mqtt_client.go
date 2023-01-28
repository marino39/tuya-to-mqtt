package adapters

import (
	"sync"
	"sync/atomic"
	"time"
	"tuya-to-mqtt/application"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog"
)

type MQTTClientParams struct {
	ClientID string
	Username string
	Password string
	MQTTUrl  string

	Log zerolog.Logger
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
	mqtt := &MQTTClient{
		params: params,
		log:    params.Log,
	}

	t := time.Now()
	mqtt.msgCountUpdateTime.Store(&t)
	return mqtt
}

func (m *MQTTClient) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := mqtt.NewClientOptions()

	opts.AddBroker(m.params.MQTTUrl)
	opts.SetClientID(m.params.ClientID)
	opts.SetUsername(m.params.Username)
	opts.SetPassword(m.params.Password)

	opts.SetDefaultPublishHandler(m.PublishHandler)
	opts.OnConnect = m.OnConnect
	opts.OnConnectionLost = m.OnConnectionLost

	m.client = mqtt.NewClient(opts)
	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		token.Wait()
		return token.Error()
	}
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
	token := m.client.Publish(topic, qos, retained, msg)
	token.Wait()
	return token.Error()
}

func (m *MQTTClient) PublishHandler(client mqtt.Client, msg mqtt.Message) {
	t := time.Now()
	m.msgCountUpdateTime.Store(&t)
	atomic.AddUint64(&m.msgCount, 1)
}

func (m *MQTTClient) OnConnect(client mqtt.Client) {
	m.log.Info().Msgf("connected")
	atomic.StoreUint64(&m.connected, 1)
}

func (m *MQTTClient) OnConnectionLost(client mqtt.Client, err error) {
	m.log.Info().Msgf("connect lost: %v", err)
	atomic.StoreUint64(&m.connected, 0)
}

var _ application.MQTTClient = &MQTTClient{}
