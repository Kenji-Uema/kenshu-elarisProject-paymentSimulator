package fakes

import (
	"context"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"google.golang.org/protobuf/proto"
)

var _ port.MqProducer = (*FakeMqProducer)(nil)

type FakeMqProducer struct {
	DeclareExchangeFn func(config config.ExchangeConfig) error
	PublishFn         func(ctx context.Context, message proto.Message, routingKey string) error
	CloseChannelFn    func() error

	DeclareExchangeCallCount int
	PublishCallCount         int
	CloseChannelCallCount    int

	LastDeclaredExchange    config.ExchangeConfig
	LastPublishedCtx        context.Context
	LastPublishedMessage    proto.Message
	LastPublishedRoutingKey string
}

func (f *FakeMqProducer) DeclareExchange(cfg config.ExchangeConfig) error {
	f.DeclareExchangeCallCount++
	f.LastDeclaredExchange = cfg

	if f.DeclareExchangeFn != nil {
		return f.DeclareExchangeFn(cfg)
	}

	return nil
}

func (f *FakeMqProducer) Publish(ctx context.Context, message proto.Message, routingKey string) error {
	f.PublishCallCount++
	f.LastPublishedCtx = ctx
	f.LastPublishedMessage = message
	f.LastPublishedRoutingKey = routingKey

	if f.PublishFn != nil {
		return f.PublishFn(ctx, message, routingKey)
	}

	return nil
}

func (f *FakeMqProducer) CloseChannel() error {
	f.CloseChannelCallCount++

	if f.CloseChannelFn != nil {
		return f.CloseChannelFn()
	}

	return nil
}
