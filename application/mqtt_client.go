package application

import "time"

type MQTTStatus struct {
	MessageCount      uint64
	LastTimePublished time.Time
	Connected         bool
}

type MQTTMessage interface {
	Duplicate() bool
	Qos() byte
	Retained() bool
	Topic() string
	MessageID() uint16
	Payload() []byte
	Ack()
}

type MQTTClient interface {
	Publish(topic string, qos byte, retained bool, msg any) error
	Subscribe(topic string, qos byte, handler func(msg MQTTMessage)) error

	Connect() error
	IsConnected() bool
	Status() MQTTStatus
}
