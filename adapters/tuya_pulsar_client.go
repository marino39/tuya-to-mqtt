package adapters

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"tuya-to-mqtt/application"

	pulsar "github.com/tuya/tuya-pulsar-sdk-go"
	"github.com/tuya/tuya-pulsar-sdk-go/pkg/tylog"
	"github.com/tuya/tuya-pulsar-sdk-go/pkg/tyutils"
)

func init() {
	tylog.SetGlobalLog("tuya-to-mqtt", true)
}

type messageHandler struct {
	AesSecret     string
	ClientHandler func(ctx context.Context, m application.Message) error
}

func (h *messageHandler) HandlePayload(ctx context.Context, msg *pulsar.Message, payload []byte) error {
	// let's decode the payload with AES
	m := map[string]interface{}{}
	err := json.Unmarshal(payload, &m)
	if err != nil {
		return err
	}
	bs := m["data"].(string)
	de, err := base64.StdEncoding.DecodeString(bs)
	if err != nil {
		return err
	}
	decode := tyutils.EcbDecrypt(de, []byte(h.AesSecret))

	// build message
	var appMsg application.Message
	err = json.Unmarshal(decode, &msg)
	if err != nil {
		return err
	}

	return h.ClientHandler(ctx, appMsg)
}

type TuyaPulsarClientParams struct {
	AccessID      string
	AccessKey     string
	PulsarAddress string
}

type TuyaPulsarClient struct {
	accessKey string

	client      pulsar.Client
	consumerCfg pulsar.ConsumerConfig
}

func NewTuyaPulsarClient(params TuyaPulsarClientParams) *TuyaPulsarClient {
	return &TuyaPulsarClient{
		accessKey: params.AccessKey,
		client: pulsar.NewClient(pulsar.ClientConfig{
			PulsarAddr: params.PulsarAddress,
		}),
		consumerCfg: pulsar.ConsumerConfig{
			Topic: pulsar.TopicForAccessID(params.AccessID),
			Auth:  pulsar.NewAuthProvider(params.AccessID, params.AccessKey),
		},
	}
}

func (t *TuyaPulsarClient) Subscribe(ctx context.Context, handlerFunc func(ctx context.Context, m application.Message) error) error {
	c, err := t.client.NewConsumer(t.consumerCfg)
	if err != nil {
		return err
	}
	defer c.Stop()

	c.ReceiveAndHandle(ctx, &messageHandler{AesSecret: t.accessKey[8:24], ClientHandler: handlerFunc})
	return nil
}

var _ application.TuyaPulsarClient = &TuyaPulsarClient{}
