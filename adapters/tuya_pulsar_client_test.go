package adapters

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"
	"tuya-to-mqtt/application"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	pulsar "github.com/tuya/tuya-pulsar-sdk-go"
	"github.com/tuya/tuya-pulsar-sdk-go/pkg/tyutils"
)

func TestNewTuyaPulsarClient(t *testing.T) {
	tuyaPulsarClient, err := NewTuyaPulsarClient(TuyaPulsarClientParams{
		AccessID:     "access_id",
		AccessKey:    "123456789012345678901234",
		PulsarClient: &MockPulsarClient{},
	})
	require.NoError(t, err)
	require.NotNil(t, tuyaPulsarClient)
}

func TestNewTuyaPulsarClient_NoPulsarClient(t *testing.T) {
	tuyaPulsarClient, err := NewTuyaPulsarClient(TuyaPulsarClientParams{
		AccessID:     "access_id",
		AccessKey:    "access_key",
		PulsarClient: &MockPulsarClient{},
	})
	require.Error(t, err)
	require.Nil(t, tuyaPulsarClient)
}

func TestNewTuyaPulsarClient_TooShortAccessKey(t *testing.T) {
	tuyaPulsarClient, err := NewTuyaPulsarClient(TuyaPulsarClientParams{
		AccessID:     "access_id",
		AccessKey:    "123456789012345678901234",
		PulsarClient: nil,
	})
	require.Error(t, err)
	require.Nil(t, tuyaPulsarClient)
}

func TestTuyaPulsarClient_Subscribe(t *testing.T) {
	accessID := "access_id"
	accessKey := "123456789012345678901234"

	mPulsarClient := &MockPulsarClient{}
	mPulsarConsumer := &MockPulsarConsumer{}

	tuyaPulsarClient, err := NewTuyaPulsarClient(TuyaPulsarClientParams{
		AccessID:     accessID,
		AccessKey:    accessKey,
		PulsarClient: mPulsarClient,
	})
	require.NoError(t, err)
	require.NotNil(t, tuyaPulsarClient)

	expectedMessage := &application.Message{
		DataID:     "1",
		DevID:      "2",
		ProductKey: "ab1",
		Status: []application.Status{
			{
				Code:      "switch",
				Timestamp: uint64(time.Now().Unix()),
				Value:     false,
			},
		},
	}

	mPulsarClient.On("NewConsumer", mock.Anything).Return(mPulsarConsumer, nil).Once()
	mPulsarConsumer.On("ReceiveAndHandle", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		f := args.Get(1).(pulsar.PayloadHandler)
		err := f.HandlePayload(context.Background(), nil, buildPulsarPayload(t, expectedMessage, accessKey))
		require.NoError(t, err)
	}).Return().Once()
	mPulsarConsumer.On("Stop").Return().Once()

	var receivedMessage *application.Message
	err = tuyaPulsarClient.Subscribe(context.Background(), func(ctx context.Context, msg *application.Message) error {
		receivedMessage = msg
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, expectedMessage, receivedMessage)

	mPulsarClient.AssertExpectations(t)
	mPulsarConsumer.AssertExpectations(t)
}

func TestTuyaPulsarClient_Subscribe_FailedToCreateConsumer(t *testing.T) {
	accessID := "access_id"
	accessKey := "123456789012345678901234"

	mPulsarClient := &MockPulsarClient{}

	tuyaPulsarClient, err := NewTuyaPulsarClient(TuyaPulsarClientParams{
		AccessID:     accessID,
		AccessKey:    accessKey,
		PulsarClient: mPulsarClient,
	})
	require.NoError(t, err)
	require.NotNil(t, tuyaPulsarClient)

	mPulsarClient.On("NewConsumer", mock.Anything).Return(nil, fmt.Errorf("fail")).Once()

	err = tuyaPulsarClient.Subscribe(context.Background(), func(ctx context.Context, msg *application.Message) error {
		return nil
	})
	assert.Error(t, err)

	mPulsarClient.AssertExpectations(t)
}

func TestTuyaPulsarClient_Subscribe_WrongAccessKey(t *testing.T) {
	accessID := "access_id"
	accessKey := "123456789012345678901234"
	accessKey2 := "098765432109876543210987"

	mPulsarClient := &MockPulsarClient{}
	mPulsarConsumer := &MockPulsarConsumer{}

	tuyaPulsarClient, err := NewTuyaPulsarClient(TuyaPulsarClientParams{
		AccessID:     accessID,
		AccessKey:    accessKey,
		PulsarClient: mPulsarClient,
	})
	require.NoError(t, err)
	require.NotNil(t, tuyaPulsarClient)

	messageToSend := &application.Message{
		DataID:     "1",
		DevID:      "2",
		ProductKey: "ab1",
		Status: []application.Status{
			{
				Code:      "switch",
				Timestamp: uint64(time.Now().Unix()),
				Value:     false,
			},
		},
	}

	mPulsarClient.On("NewConsumer", mock.Anything).Return(mPulsarConsumer, nil).Once()
	mPulsarConsumer.On("ReceiveAndHandle", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		f := args.Get(1).(pulsar.PayloadHandler)
		err := f.HandlePayload(context.Background(), nil, buildPulsarPayload(t, messageToSend, accessKey2))
		require.Error(t, err)
	}).Return().Once()
	mPulsarConsumer.On("Stop").Return().Once()

	var receivedMessage *application.Message
	err = tuyaPulsarClient.Subscribe(context.Background(), func(ctx context.Context, msg *application.Message) error {
		receivedMessage = msg
		return nil
	})
	assert.NoError(t, err)
	assert.Nil(t, receivedMessage)

	mPulsarClient.AssertExpectations(t)
	mPulsarConsumer.AssertExpectations(t)
}

func TestTuyaPulsarClient_Subscribe_MalformedPayload(t *testing.T) {
	makePayloadFuncs := map[string]func(t *testing.T, message *application.Message, accessKey string) []byte{
		"MalformedPayload": buildPulsarMalformedPayload,
		"MalformedData":    buildPulsarMalformedData,
		"MalformedBase64":  buildPulsarMalformedBase64,
	}

	for testCase, makePayloadFunc := range makePayloadFuncs {
		t.Run(testCase, func(t *testing.T) {
			accessID := "access_id"
			accessKey := "123456789012345678901234"

			mPulsarClient := &MockPulsarClient{}
			mPulsarConsumer := &MockPulsarConsumer{}

			tuyaPulsarClient, err := NewTuyaPulsarClient(TuyaPulsarClientParams{
				AccessID:     accessID,
				AccessKey:    accessKey,
				PulsarClient: mPulsarClient,
			})
			require.NoError(t, err)
			require.NotNil(t, tuyaPulsarClient)

			messageToSend := &application.Message{
				DataID:     "1",
				DevID:      "2",
				ProductKey: "ab1",
				Status: []application.Status{
					{
						Code:      "switch",
						Timestamp: uint64(time.Now().Unix()),
						Value:     false,
					},
				},
			}

			mPulsarClient.On("NewConsumer", mock.Anything).Return(mPulsarConsumer, nil).Once()
			mPulsarConsumer.On("ReceiveAndHandle", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				f := args.Get(1).(pulsar.PayloadHandler)
				err := f.HandlePayload(context.Background(), nil, makePayloadFunc(t, messageToSend, accessKey))
				require.Error(t, err)
			}).Return().Once()
			mPulsarConsumer.On("Stop").Return().Once()

			var receivedMessage *application.Message
			err = tuyaPulsarClient.Subscribe(context.Background(), func(ctx context.Context, msg *application.Message) error {
				receivedMessage = msg
				return nil
			})
			assert.NoError(t, err)
			assert.Nil(t, receivedMessage)

			mPulsarClient.AssertExpectations(t)
			mPulsarConsumer.AssertExpectations(t)
		})
	}
}

func buildPulsarPayload(t *testing.T, message *application.Message, accessKey string) []byte {
	data, err := json.Marshal(message)
	require.NoError(t, err)

	encData := tyutils.EcbEncrypt(data, []byte(accessKey[8:24]))

	dataBase64 := base64.StdEncoding.EncodeToString(encData)

	payload, err := json.Marshal(map[string]interface{}{"data": dataBase64})
	require.NoError(t, err)

	return payload
}

func buildPulsarMalformedBase64(t *testing.T, message *application.Message, accessKey string) []byte {
	data, err := json.Marshal(message)
	require.NoError(t, err)

	encData := tyutils.EcbEncrypt(data, []byte(accessKey[8:24]))

	dataBase64 := base64.StdEncoding.EncodeToString(encData)
	dataBase64 += "Ó"

	payload, err := json.Marshal(map[string]interface{}{"data": dataBase64})
	require.NoError(t, err)

	return payload
}

func buildPulsarMalformedData(t *testing.T, message *application.Message, accessKey string) []byte {
	data, err := json.Marshal(message)
	require.NoError(t, err)

	encData := tyutils.EcbEncrypt(data[len(data)/2:], []byte(accessKey[8:24]))

	dataBase64 := base64.StdEncoding.EncodeToString(encData)

	payload, err := json.Marshal(map[string]interface{}{"data": dataBase64})
	require.NoError(t, err)

	return payload
}

func buildPulsarMalformedPayload(t *testing.T, message *application.Message, accessKey string) []byte {
	data, err := json.Marshal(message)
	require.NoError(t, err)

	encData := tyutils.EcbEncrypt(data, []byte(accessKey[8:24]))

	dataBase64 := base64.StdEncoding.EncodeToString(encData)

	payload, err := json.Marshal(map[string]interface{}{"data": dataBase64})
	require.NoError(t, err)

	return payload[len(payload)/2:]
}
