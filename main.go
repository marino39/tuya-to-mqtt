package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
	"tuya-to-mqtt/adapters"
	"tuya-to-mqtt/application"

	"github.com/rs/zerolog"
	pulsar "github.com/tuya/tuya-pulsar-sdk-go"
	"github.com/urfave/cli/v2"
)

var Flags = []cli.Flag{
	FlagLogLevel,
	FlagLogWriter,
	FlagTuyaAccessID,
	FlagTuyaAccessKey,
	FlagTuyaPulsarRegion,
}

func main() {
	var logger zerolog.Logger

	app := cli.App{
		Name:    "tuya-to-mqtt",
		Version: "v0.0.1",
		Flags:   Flags,
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

			logger.Info().Msgf("pulsar endpoint: %s", pulsarAddress)
			pulsarClient := pulsar.NewClient(pulsar.ClientConfig{
				PulsarAddr: pulsarAddress,
			})

			tuyaPulsarClient, err := adapters.NewTuyaPulsarClient(adapters.TuyaPulsarClientParams{
				AccessID:     ctx.String(FlagTuyaAccessID.Name),
				AccessKey:    ctx.String(FlagTuyaAccessKey.Name),
				PulsarClient: pulsarClient,
				Log:          logger.With().Str("module", "pulsar-client").Logger(),
			})
			if err != nil {
				return err
			}

			tuyaToMQTTService, err := application.NewTuyaToMQTTService(application.TuyaToMQTTServiceParams{
				TuyaPulsarClient: tuyaPulsarClient,
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
		Authors: []*cli.Author{
			{
				Name:  "Marcin Gorzynski",
				Email: "marcin@gorzynski.me",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Err(err).Msg("service terminated")
	}
}
