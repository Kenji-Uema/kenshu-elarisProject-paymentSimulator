package fakes

import (
	"context"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
)

var _ port.MqConsumer = (*FakeMqConsumer)(nil)

type FakeMqConsumer struct {
	DeclareQueueFn func(ctx context.Context, config config.QueueConfig) error
	BindQueueFn    func(ctx context.Context, config config.BindingConfig) error
	ConsumeFn      func(ctx context.Context) (<-chan amqp.Delivery, error)
	CloseChannelFn func() error

	DeclareQueueCallCount int
	BindQueueCallCount    int
	ConsumeCallCount      int
	CloseChannelCallCount int

	LastDeclaredQueue config.QueueConfig
	LastBoundQueue    config.BindingConfig
	LastDeclareCtx    context.Context
	LastBindCtx       context.Context
	LastConsumeCtx    context.Context
}

func (f *FakeMqConsumer) DeclareQueue(ctx context.Context, cfg config.QueueConfig) error {
	f.DeclareQueueCallCount++
	f.LastDeclareCtx = ctx
	f.LastDeclaredQueue = cfg

	if f.DeclareQueueFn != nil {
		return f.DeclareQueueFn(ctx, cfg)
	}

	return nil
}

func (f *FakeMqConsumer) BindQueue(ctx context.Context, cfg config.BindingConfig) error {
	f.BindQueueCallCount++
	f.LastBindCtx = ctx
	f.LastBoundQueue = cfg

	if f.BindQueueFn != nil {
		return f.BindQueueFn(ctx, cfg)
	}

	return nil
}

func (f *FakeMqConsumer) Consume(ctx context.Context) (<-chan amqp.Delivery, error) {
	f.ConsumeCallCount++
	f.LastConsumeCtx = ctx

	if f.ConsumeFn != nil {
		return f.ConsumeFn(ctx)
	}

	return nil, nil
}

func (f *FakeMqConsumer) CloseChannel() error {
	f.CloseChannelCallCount++

	if f.CloseChannelFn != nil {
		return f.CloseChannelFn()
	}

	return nil
}
