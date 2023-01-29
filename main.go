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
			FlagTuyaAccessID,
			FlagTuyaAccessKey,
			FlagTuyaPulsarRegion,
			FlagTuyaPulsarEnv,
			FlagMQTTUrl,
			FlagMQTTClientID,
			FlagMQTTUsername,
			FlagMQTTPassword,
			FlagMQTTTopic,
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

			err := mqttClient.Connect()
			if err != nil {
				return err
			}

			var pulsarAddress string
			switch ctx.String(FlagTuyaPulsarRegion.Name) {
			case "US":
				pulsarAddress = pulsar.PulsarAddrUS
			case "EU":
				pulsarAddress = pulsar.PulsarAddrEU
			case "CN":
				pulsarAddress = pulsar.PulsarAddrCN
			default:
				return fmt.Errorf("invalid tuya pulsar region")
			}

			pulsarEnv := adapters.PulsarEnvironmentProd
			if ctx.String(FlagTuyaPulsarEnv.Name) == "TEST" {
				pulsarEnv = adapters.PulsarEnvironmentTest
			}

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
				TuyaPulsarClient: tuyaPulsarClient,
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
		DisableSliceFlagSeparator: true,
		Commands: []*cli.Command{
			{
				Name: "publish",
				Flags: []cli.Flag{
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

						msgData, err := json.Marshal(msg)
						if err != nil {
							return err
						}

						err = mqttClient.Publish(application.BuildMQTTTopicForMessage(topicPrefix, msg), 2, false, []byte(msgData))
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

					err = mqttClient.Subscribe(ctx.String(FlagMQTTTopic.Name), 2, func(msg application.MQTTMessage) {
						fmt.Println(msg.Topic(), string(msg.Payload()))
						msg.Ack()
					})
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
