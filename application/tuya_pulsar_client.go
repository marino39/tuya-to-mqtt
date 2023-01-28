package application

import "context"

type Status struct {
	Code      string `json:"code"`
	Timestamp uint64 `json:"t"`
	Value     any    `json:"value"`
}

type Message struct {
	DataID     string   `json:"dataId"`
	DevID      string   `json:"devId"`
	ProductKey string   `json:"productKey"`
	Status     []Status `json:"status"`
}

type TuyaPulsarClient interface {
	Subscribe(ctx context.Context, handlerFunc func(ctx context.Context, msg Message) error) error
}
