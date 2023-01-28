package main

import "github.com/urfave/cli/v2"

var FlagLogLevel = &cli.StringFlag{
	Name:     "log-level",
	EnvVars:  []string{"LOG_LEVEL"},
	Value:    "info",
	Required: false,
}

var FlagLogWriter = &cli.StringFlag{
	Name:     "log-writer",
	EnvVars:  []string{"LOG_WRITER"},
	Value:    "console",
	Required: false,
}

var FlagTuyaAccessID = &cli.StringFlag{
	Name:     "tuya-access-id",
	Usage:    "tuya cloud access id",
	EnvVars:  []string{"TUYA_ACCESS_ID"},
	Required: true,
}

var FlagTuyaAccessKey = &cli.StringFlag{
	Name:     "tuya-access-key",
	Usage:    "tuya cloud access key",
	EnvVars:  []string{"TUYA_ACCESS_KEY"},
	Required: true,
}

var FlagTuyaPulsarRegion = &cli.StringFlag{
	Name:     "tuya-pulsar-region",
	Usage:    "one of: [EU, US, CN]",
	EnvVars:  []string{"TUYA_ACCESS_REGION"},
	Value:    "EU",
	Required: false,
}

var FlagMQTTUrl = &cli.StringFlag{
	Name:     "mqtt-url",
	Usage:    "tcp://broker:port",
	EnvVars:  []string{"MQTT_URL"},
	Required: true,
}

var FlagMQTTClientID = &cli.StringFlag{
	Name:     "mqtt-client-id",
	EnvVars:  []string{"MQTT_CLIENT_ID"},
	Required: true,
}

var FlagMQTTUsername = &cli.StringFlag{
	Name:     "mqtt-username",
	EnvVars:  []string{"MQTT_USERNAME"},
	Required: true,
}

var FlagMQTTPassword = &cli.StringFlag{
	Name:     "mqtt-password",
	EnvVars:  []string{"MQTT_PASSWORD"},
	Required: true,
}

var FlagMQTTTopic = &cli.StringFlag{
	Name:     "mqtt-topic",
	EnvVars:  []string{"MQTT_TOPIC"},
	Value:    "tuya-devices",
	Required: false,
}
