package adapters

import (
	"fmt"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMQTTClient_Connect(t *testing.T) {
	mClient := &MockMQTTClient{}
	mToken := &MockToken{}

	mqttClient := NewMQTTClient(MQTTClientParams{
		ClientID: "test",
		Username: "admin",
		Password: "password",
		MQTTUrl:  "tcp://localhost:1883",
		// for testing
		NewClientFunc: func(options *mqtt.ClientOptions) mqtt.Client {
			return mClient
		},
	})

	mClient.On("Connect").Return(mToken).Once()
	mToken.On("Wait").Return(true).Once()
	mToken.On("Error").Return(nil).Once()

	err := mqttClient.Connect()
	require.NoError(t, err)
	assert.Equal(t, true, mqttClient.IsConnected())

	status := mqttClient.Status()
	assert.Equal(t, uint64(0), status.MessageCount)
	assert.Equal(t, time.Unix(0, 0), status.LastTimePublished)
	assert.Equal(t, true, status.Connected)

	err = mqttClient.Connect()
	require.NoError(t, err)

	mClient.AssertExpectations(t)
	mToken.AssertExpectations(t)
}

func TestMQTTClient_Connect_Error(t *testing.T) {
	mClient := &MockMQTTClient{}
	mToken := &MockToken{}

	mqttClient := NewMQTTClient(MQTTClientParams{
		ClientID: "test",
		Username: "admin",
		Password: "password",
		MQTTUrl:  "tcp://localhost:1883",
		// for testing
		NewClientFunc: func(options *mqtt.ClientOptions) mqtt.Client {
			return mClient
		},
	})

	mClient.On("Connect").Return(mToken).Once()
	mToken.On("Wait").Return(true).Once()
	mToken.On("Error").Return(fmt.Errorf("internal")).Twice()

	err := mqttClient.Connect()
	require.Error(t, err)
	assert.Equal(t, false, mqttClient.IsConnected())

	status := mqttClient.Status()
	assert.Equal(t, uint64(0), status.MessageCount)
	assert.Equal(t, time.Unix(0, 0), status.LastTimePublished)
	assert.Equal(t, false, status.Connected)

	mClient.AssertExpectations(t)
	mToken.AssertExpectations(t)
}

func TestMQTTClient_OnConnectionLost(t *testing.T) {
	mClient := &MockMQTTClient{}
	mToken := &MockToken{}

	mqttClient := NewMQTTClient(MQTTClientParams{
		ClientID: "test",
		Username: "admin",
		Password: "password",
		MQTTUrl:  "tcp://localhost:1883",
		// for testing
		NewClientFunc: func(options *mqtt.ClientOptions) mqtt.Client {
			return mClient
		},
	})

	mClient.On("Connect").Return(mToken).Once()
	mToken.On("Wait").Return(true).Once()
	mToken.On("Error").Return(nil).Once()

	err := mqttClient.Connect()
	require.NoError(t, err)
	assert.Equal(t, true, mqttClient.IsConnected())

	status := mqttClient.Status()
	assert.Equal(t, uint64(0), status.MessageCount)
	assert.Equal(t, time.Unix(0, 0), status.LastTimePublished)
	assert.Equal(t, true, status.Connected)

	mqttClient.OnConnectionLost(mClient, fmt.Errorf("connection lost"))
	assert.Equal(t, false, mqttClient.IsConnected())

	status = mqttClient.Status()
	assert.Equal(t, uint64(0), status.MessageCount)
	assert.Equal(t, time.Unix(0, 0), status.LastTimePublished)
	assert.Equal(t, false, status.Connected)

	mClient.AssertExpectations(t)
	mToken.AssertExpectations(t)
}

func TestMQTTClient_Publish(t *testing.T) {
	mClient := &MockMQTTClient{}
	mToken := &MockToken{}

	mqttClient := NewMQTTClient(MQTTClientParams{
		ClientID: "test",
		Username: "admin",
		Password: "password",
		MQTTUrl:  "tcp://localhost:1883",
		// for testing
		NewClientFunc: func(options *mqtt.ClientOptions) mqtt.Client {
			return mClient
		},
	})

	mClient.On("Connect").Run(func(args mock.Arguments) {
		mqttClient.OnConnect(mClient)
	}).Return(mToken).Once()
	mToken.On("Wait").Return(true).Once()
	mToken.On("Error").Return(nil).Once()

	err := mqttClient.Connect()
	require.NoError(t, err)
	assert.Equal(t, true, mqttClient.IsConnected())

	topic := "testTopic"
	qos := byte(0)
	retained := true
	payload := []byte("test_payload")

	mClient.On("Publish", topic, qos, retained, payload).Return(mToken).Once()
	mToken.On("Wait").Return(true).Once()
	mToken.On("Error").Return(nil).Once()

	err = mqttClient.Publish(topic, qos, retained, payload)
	require.NoError(t, err)

	status := mqttClient.Status()
	assert.Equal(t, uint64(1), status.MessageCount)
	assert.True(t, time.Now().After(status.LastTimePublished))
	assert.Equal(t, true, status.Connected)

	mClient.AssertExpectations(t)
	mToken.AssertExpectations(t)
}

func TestMQTTClient_Publish_NotConnected(t *testing.T) {
	mClient := &MockMQTTClient{}
	mToken := &MockToken{}

	mqttClient := NewMQTTClient(MQTTClientParams{
		ClientID: "test",
		Username: "admin",
		Password: "password",
		MQTTUrl:  "tcp://localhost:1883",
		// for testing
		NewClientFunc: func(options *mqtt.ClientOptions) mqtt.Client {
			return mClient
		},
	})

	topic := "testTopic"
	qos := byte(0)
	retained := true
	payload := []byte("test_payload")

	err := mqttClient.Publish(topic, qos, retained, payload)
	require.Error(t, err)
	require.Equal(t, ErrMQTTNotConnected, err)

	status := mqttClient.Status()
	assert.Equal(t, uint64(0), status.MessageCount)
	assert.True(t, time.Now().After(status.LastTimePublished))
	assert.Equal(t, false, status.Connected)

	mClient.AssertExpectations(t)
	mToken.AssertExpectations(t)
}

func TestMQTTClient_Publish_Error(t *testing.T) {
	mClient := &MockMQTTClient{}
	mToken := &MockToken{}

	mqttClient := NewMQTTClient(MQTTClientParams{
		ClientID: "test",
		Username: "admin",
		Password: "password",
		MQTTUrl:  "tcp://localhost:1883",
		// for testing
		NewClientFunc: func(options *mqtt.ClientOptions) mqtt.Client {
			return mClient
		},
	})

	mClient.On("Connect").Run(func(args mock.Arguments) {
		mqttClient.OnConnect(mClient)
	}).Return(mToken).Once()
	mToken.On("Wait").Return(true).Once()
	mToken.On("Error").Return(nil).Once()

	err := mqttClient.Connect()
	require.NoError(t, err)
	assert.Equal(t, true, mqttClient.IsConnected())

	topic := "testTopic"
	qos := byte(0)
	retained := true
	payload := []byte("test_payload")

	mClient.On("Publish", topic, qos, retained, payload).Return(mToken).Once()
	mToken.On("Wait").Return(true).Once()
	mToken.On("Error").Return(fmt.Errorf("internal")).Twice()

	err = mqttClient.Publish(topic, qos, retained, payload)
	require.Error(t, err)

	status := mqttClient.Status()
	assert.Equal(t, uint64(0), status.MessageCount)
	assert.True(t, time.Now().After(status.LastTimePublished))
	assert.Equal(t, true, status.Connected)

	mClient.AssertExpectations(t)
	mToken.AssertExpectations(t)
}
