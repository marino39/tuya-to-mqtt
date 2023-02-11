package application

import "context"

type Status struct {
	Code      string `json:"code"`
	Timestamp uint64 `json:"t"`
	Value     any    `json:"value"`
}

type MessageType int

const (
	MessageTypeStatus     = 4
	MessageTypeNameModify = 20
)

type Message struct {
	Type       MessageType
	DataID     string   `json:"dataId"`
	DevID      string   `json:"devId"`
	ProductKey string   `json:"productKey"`
	Status     []Status `json:"status"`

	BizCode string         `json:"bizCode"`
	BizData map[string]any `json:"bizData"`
}

type TuyaPulsarClient interface {
	Subscribe(ctx context.Context, handlerFunc func(ctx context.Context, msg *Message) error) error
}
