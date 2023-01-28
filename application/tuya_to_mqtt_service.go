package application

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/sync/errgroup"
)

type TuyaToMQTTService interface {
	Run(ctx context.Context) error
}

type TuyaToMQTTServiceParams struct {
	TuyaPulsarClient TuyaPulsarClient
}

type tuyaToMQTTService struct {
	params TuyaToMQTTServiceParams
}

func NewTuyaToMQTTService(params TuyaToMQTTServiceParams) (TuyaToMQTTService, error) {
	if params.TuyaPulsarClient == nil {
		return nil, fmt.Errorf("misconfigured")
	}
	return &tuyaToMQTTService{params: params}, nil
}

func (t tuyaToMQTTService) Run(ctx context.Context) error {
	g := errgroup.Group{}

	g.Go(func() error {
		return t.params.TuyaPulsarClient.Subscribe(ctx, func(ctx context.Context, m Message) error {
			spew.Dump(m)
			return nil
		})
	})

	return g.Wait()
}
