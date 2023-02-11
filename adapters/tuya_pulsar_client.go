package adapters

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"tuya-to-mqtt/application"

	"github.com/davecgh/go-spew/spew"
	"github.com/rs/zerolog"
	pulsar "github.com/tuya/tuya-pulsar-sdk-go"
	"github.com/tuya/tuya-pulsar-sdk-go/pkg/tylog"
	"github.com/tuya/tuya-pulsar-sdk-go/pkg/tyutils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	tylog.SetGlobalLog("tuya-to-mqtt", true, tylog.WithLevelOption(zap.NewAtomicLevelAt(zapcore.FatalLevel)))
}

type PulsarEnvironment int

const (
	PulsarEnvironmentProd PulsarEnvironment = 0
	PulsarEnvironmentTest                   = 1
)

type messageHandler struct {
	aesSecret     string
	clientHandler func(ctx context.Context, msg *application.Message) error

	log zerolog.Logger
}

func (h *messageHandler) HandlePayload(ctx context.Context, _ *pulsar.Message, payload []byte) error {
	// let's decode the payload with AES
	m := map[string]interface{}{}
	err := json.Unmarshal(payload, &m)
	if err != nil {
		h.log.Warn().Msg("failed to parse pulsar message payload")
		return err
	}
	bs := m["data"].(string)
	de, err := base64.StdEncoding.DecodeString(bs)
	if err != nil {
		h.log.Warn().Msg("failed to decode message data")
		return err
	}
	decode := tyutils.EcbDecrypt(de, []byte(h.aesSecret))

	spew.Dump(m)

	// build message
	var appMsg application.Message
	err = json.Unmarshal(decode, &appMsg)
	if err != nil {
		h.log.Warn().Msg("failed to parse message data")
		return err
	}

	if msgType, ok := m["protocol"].(float64); ok {
		appMsg.Type = application.MessageType(msgType)
	}

	return h.clientHandler(ctx, &appMsg)
}

type TuyaPulsarClientParams struct {
	AccessID  string
	AccessKey string

	PulsarClient      pulsar.Client
	PulsarEnvironment PulsarEnvironment

	Log zerolog.Logger
}

type TuyaPulsarClient struct {
	accessKey string

	client      pulsar.Client
	consumerCfg pulsar.ConsumerConfig

	log zerolog.Logger
}

func NewTuyaPulsarClient(params TuyaPulsarClientParams) (*TuyaPulsarClient, error) {
	if params.PulsarClient == nil {
		return nil, fmt.Errorf("pulsar client is required")
	}

	if len(params.AccessKey) < 24 {
		return nil, fmt.Errorf("access key needs to be at least 24 characters long")
	}

	topic := pulsar.TopicForAccessID(params.AccessID)
	if params.PulsarEnvironment == PulsarEnvironmentTest {
		topic = strings.Replace(topic, "event", "event-test", 1)
	}

	return &TuyaPulsarClient{
		accessKey: params.AccessKey,
		client:    params.PulsarClient,
		consumerCfg: pulsar.ConsumerConfig{
			Topic: topic,
			Auth:  pulsar.NewAuthProvider(params.AccessID, params.AccessKey),
		},
		log: params.Log,
	}, nil
}

func (t *TuyaPulsarClient) Subscribe(ctx context.Context, handlerFunc func(ctx context.Context, msg *application.Message) error) error {
	c, err := t.client.NewConsumer(t.consumerCfg)
	if err != nil {
		return err
	}
	defer c.Stop()

	c.ReceiveAndHandle(ctx, &messageHandler{aesSecret: t.accessKey[8:24], clientHandler: handlerFunc, log: t.log})
	return nil
}

var _ application.TuyaPulsarClient = &TuyaPulsarClient{}
