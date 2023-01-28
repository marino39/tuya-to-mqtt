package adapters

import (
	"context"

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
