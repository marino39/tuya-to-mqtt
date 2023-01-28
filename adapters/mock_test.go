package adapters

import (
	"context"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/mock"
	pulsar "github.com/tuya/tuya-pulsar-sdk-go"
)

type MockPulsarClient struct {
	mock.Mock
}

func (m *MockPulsarClient) NewConsumer(config pulsar.ConsumerConfig) (pulsar.Consumer, error) {
	args := m.Called(config)

	var err error
	var consumer pulsar.Consumer
	if consInt := args.Get(0); consInt != nil {
		consumer = consInt.(pulsar.Consumer)
	}
	if errInt := args.Get(1); errInt != nil {
		err = errInt.(error)
	}
	return consumer, err
}

var _ pulsar.Client = &MockPulsarClient{}

type MockPulsarConsumer struct {
	mock.Mock
}

func (m *MockPulsarConsumer) ReceiveAndHandle(ctx context.Context, handler pulsar.PayloadHandler) {
	m.Called(ctx, handler)
}

func (m *MockPulsarConsumer) Stop() {
	m.Called()
}

var _ pulsar.Consumer = &MockPulsarConsumer{}

type MockMQTTClient struct {
	mock.Mock
}

func (m *MockMQTTClient) IsConnected() bool {
	args := m.Called()
	return args.Get(0).(bool)
}

func (m *MockMQTTClient) IsConnectionOpen() bool {
	args := m.Called()
	return args.Get(0).(bool)
}

func (m *MockMQTTClient) Connect() mqtt.Token {
	args := m.Called()
	return args.Get(0).(mqtt.Token)
}

func (m *MockMQTTClient) Disconnect(quiesce uint) {
	m.Called(quiesce)
}

func (m *MockMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	args := m.Called(topic, qos, retained, payload)
	return args.Get(0).(mqtt.Token)
}

func (m *MockMQTTClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	//TODO implement me
	panic("implement me")
}

func (m *MockMQTTClient) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	//TODO implement me
	panic("implement me")
}

func (m *MockMQTTClient) Unsubscribe(topics ...string) mqtt.Token {
	//TODO implement me
	panic("implement me")
}

func (m *MockMQTTClient) AddRoute(topic string, callback mqtt.MessageHandler) {
	//TODO implement me
	panic("implement me")
}

func (m *MockMQTTClient) OptionsReader() mqtt.ClientOptionsReader {
	//TODO implement me
	panic("implement me")
}

var _ mqtt.Client = &MockMQTTClient{}

type MockToken struct {
	mock.Mock
}

func (m *MockToken) Wait() bool {
	args := m.Called()
	return args.Get(0).(bool)
}

func (m *MockToken) WaitTimeout(duration time.Duration) bool {
	args := m.Called(duration)
	return args.Get(0).(bool)
}

func (m *MockToken) Done() <-chan struct{} {
	args := m.Called()
	return args.Get(0).(<-chan struct{})
}

func (m *MockToken) Error() error {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(error)
}

var _ mqtt.Token = &MockToken{}
