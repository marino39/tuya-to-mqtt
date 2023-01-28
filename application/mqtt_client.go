package application

import "time"

type MQTTStatus struct {
	MessageCount      uint64
	LastTimePublished time.Time
	Connected         bool
}

type MQTTClient interface {
	Publish(topic string, qos byte, retained bool, msg any) error

	Connect() error
	IsConnected() bool
	Status() MQTTStatus
}
