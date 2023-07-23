package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"tuya-to-mqtt/adapters"
	"tuya-to-mqtt/application"

	"github.com/rs/zerolog"
	"github.com/tuya/tuya-connector-go/connector"
	"github.com/tuya/tuya-connector-go/connector/env"
	"github.com/tuya/tuya-connector-go/connector/httplib"
	pulsar "github.com/tuya/tuya-pulsar-sdk-go"
	"github.com/urfave/cli/v2"
)

func main() {
	var logger zerolog.Logger

	app := cli.App{
		Name:    "tuya-to-mqtt",
		Version: "v0.0.1",
		Flags: []cli.Flag{
			FlagLogLevel,
			FlagLogWriter,
		},
		Before: func(ctx *cli.Context) error {
			var logWriter io.Writer
			if ctx.String(FlagLogWriter.Name) == "console" {
				logWriter = zerolog.ConsoleWriter{
					Out:        os.Stderr,
					TimeFormat: time.RFC3339Nano,
				}
			} else if ctx.String(FlagLogWriter.Name) == "json" {
				logWriter = os.Stderr
			}

			logger = zerolog.New(logWriter).With().Timestamp().
				Str("service", "tuya-to-mqtt").
				Str("module", "main").
				Logger()

			level, err := zerolog.ParseLevel(ctx.String(FlagLogLevel.Name))
			if err != nil {
				return err
			}

			zerolog.SetGlobalLevel(level)

			return nil
		},
		DisableSliceFlagSeparator: true,
		Commands: []*cli.Command{
			{
				Name: "run",
				Flags: []cli.Flag{
					FlagTuyaAccessID,
					FlagTuyaAccessKey,
					FlagTuyaPulsarRegion,
					FlagTuyaPulsarEnv,
					FlagTuyaUser,
					FlagMQTTUrl,
					FlagMQTTClientID,
					FlagMQTTUsername,
					FlagMQTTPassword,
					FlagMQTTTopic,
				},
				Action: func(ctx *cli.Context) error {
					logger.Info().Msg("service starting...")

					appCtx, cancel := context.WithCancel(logger.WithContext(context.Background()))
					go func() {
						c := make(chan os.Signal)
						signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

						<-c

						logger.Warn().Msg("interrupt signal received")
						cancel()
					}()

					logger.Info().Msgf("mqtt broker: %s, client_id: %s", ctx.String(FlagMQTTUrl.Name), ctx.String(FlagMQTTClientID.Name))
					mqttClient := adapters.NewMQTTClient(adapters.MQTTClientParams{
						ClientID: ctx.String(FlagMQTTClientID.Name),
						Username: ctx.String(FlagMQTTUsername.Name),
						Password: ctx.String(FlagMQTTPassword.Name),
						MQTTUrl:  ctx.String(FlagMQTTUrl.Name),
						Log:      logger.With().Str("module", "mqtt-client").Logger(),
					})

					var tuyaAddress string
					var pulsarAddress string
					switch ctx.String(FlagTuyaPulsarRegion.Name) {
					case "US":
						tuyaAddress = httplib.URL_US
						pulsarAddress = pulsar.PulsarAddrUS
					case "EU":
						tuyaAddress = httplib.URL_EU
						pulsarAddress = pulsar.PulsarAddrEU
					case "CN":
						tuyaAddress = httplib.URL_CN
						pulsarAddress = pulsar.PulsarAddrCN
					default:
						return fmt.Errorf("invalid tuya pulsar region")
					}

					pulsarEnv := adapters.PulsarEnvironmentProd
					if ctx.String(FlagTuyaPulsarEnv.Name) == "TEST" {
						pulsarEnv = adapters.PulsarEnvironmentTest
					}

					logger.Info().Msgf("tuya client endpoint: %s env: %s", tuyaAddress, ctx.String(FlagTuyaPulsarEnv.Name))
					connector.InitWithOptions(
						env.WithApiHost(tuyaAddress),
						env.WithMsgHost(pulsarAddress),
						env.WithAccessID(ctx.String(FlagTuyaAccessID.Name)),
						env.WithAccessKey(ctx.String(FlagTuyaAccessKey.Name)),
						env.WithAppName("tuya-to-mqtt"),
					)

					tuyaClient := adapters.NewTuyaClient()

					logger.Info().Msgf("pulsar endpoint: %s env: %s", pulsarAddress, ctx.String(FlagTuyaPulsarEnv.Name))
					pulsarClient := pulsar.NewClient(pulsar.ClientConfig{
						PulsarAddr: pulsarAddress,
					})

					tuyaPulsarClient, err := adapters.NewTuyaPulsarClient(adapters.TuyaPulsarClientParams{
						AccessID:          ctx.String(FlagTuyaAccessID.Name),
						AccessKey:         ctx.String(FlagTuyaAccessKey.Name),
						PulsarClient:      pulsarClient,
						PulsarEnvironment: pulsarEnv,
						Log:               logger.With().Str("module", "pulsar-client").Logger(),
					})
					if err != nil {
						return err
					}

					tuyaToMQTTService, err := application.NewTuyaToMQTTService(application.TuyaToMQTTServiceParams{
						TuyaClient:       tuyaClient,
						TuyaPulsarClient: tuyaPulsarClient,
						TuyaUser:         ctx.String(FlagTuyaUser.Name),
						MQTTClient:       mqttClient,
						MQTTTopic:        ctx.String(FlagMQTTTopic.Name),
						Log:              logger.With().Str("module", "tuya_to_mqtt_service").Logger(),
					})
					if err != nil {
						return err
					}

					logger.Info().Msg("service started")
					err = tuyaToMQTTService.Run(appCtx)
					if err != nil {
						return err
					}

					logger.Info().Msg("service terminating...")
					return nil
				},
			},
			{
				Name: "publish",
				Flags: []cli.Flag{
					FlagMQTTUrl,
					FlagMQTTClientID,
					FlagMQTTUsername,
					FlagMQTTPassword,
					FlagMQTTTopic,
					FlagPublishJSON,
					FlagPublishText,
				},
				Action: func(ctx *cli.Context) error {
					mqttClient := adapters.NewMQTTClient(adapters.MQTTClientParams{
						ClientID: ctx.String(FlagMQTTClientID.Name),
						Username: ctx.String(FlagMQTTUsername.Name),
						Password: ctx.String(FlagMQTTPassword.Name),
						MQTTUrl:  ctx.String(FlagMQTTUrl.Name),
					})

					err := mqttClient.Connect()
					if err != nil {
						return err
					}

					topicPrefix := ctx.String(FlagMQTTTopic.Name)
					for _, payload := range ctx.StringSlice(FlagPublishJSON.Name) {
						var msg *application.Message
						err = json.Unmarshal([]byte(payload), &msg)
						if err != nil {
							return err
						}

						for _, status := range msg.Status {
							topic := application.BuildMQTTTopicForStatus(topicPrefix, msg, status)
							value := strings.ReplaceAll(fmt.Sprintf("%#v", status.Value), "\"", "")
							err = mqttClient.Publish(topic, 2, false, []byte(value))
							if err != nil {
								return err
							}
						}
					}

					for _, payload := range ctx.StringSlice(FlagPublishText.Name) {
						err = mqttClient.Publish(topicPrefix, 2, false, []byte(payload))
						if err != nil {
							return err
						}
					}

					return nil
				},
			},
			{
				Name: "subscribe",
				Flags: []cli.Flag{
					FlagMQTTUrl,
					FlagMQTTClientID,
					FlagMQTTUsername,
					FlagMQTTPassword,
					FlagMQTTTopic,
				},
				Action: func(ctx *cli.Context) error {
					mqttClient := adapters.NewMQTTClient(adapters.MQTTClientParams{
						ClientID: ctx.String(FlagMQTTClientID.Name),
						Username: ctx.String(FlagMQTTUsername.Name),
						Password: ctx.String(FlagMQTTPassword.Name),
						MQTTUrl:  ctx.String(FlagMQTTUrl.Name),
					})

					mqttUpdateHandler := func(msg application.MQTTMessage) {
						fmt.Println(msg.Topic(), string(msg.Payload()))
						msg.Ack()
					}

					err := mqttClient.AddRoute(ctx.String(FlagMQTTTopic.Name), mqttUpdateHandler)
					if err != nil {
						return err
					}

					err = mqttClient.Connect()
					if err != nil {
						return err
					}

					err = mqttClient.Subscribe(ctx.String(FlagMQTTTopic.Name), 2, mqttUpdateHandler)
					if err != nil {
						return err
					}

					wg := sync.WaitGroup{}
					wg.Add(1)
					go func() {
						defer wg.Done()
						c := make(chan os.Signal)
						signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

						<-c

						logger.Warn().Msg("interrupt signal received")
					}()

					wg.Wait()
					return nil
				},
			},
			{},
		},
		Authors: []*cli.Author{
			{
				Name:  "Marcin Gorzynski",
				Email: "marcin@gorzynski.me",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Printf(err.Error())
	}
}
