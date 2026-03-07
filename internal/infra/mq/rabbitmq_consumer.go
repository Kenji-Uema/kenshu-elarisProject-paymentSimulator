package mq

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/mqErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitmqConsumer struct {
	*RabbitMqConnection
	channel       *amqp.Channel
	queue         *amqp.Queue
	consumeConfig config.ConsumeConfig
}

func NewRabbitmqConsumer(rabbitMqConnection *RabbitMqConnection, consumeConfig config.ConsumeConfig) (port.MqConsumer, error) {
	rabbitmqConsumer := rabbitmqConsumer{
		RabbitMqConnection: rabbitMqConnection,
		consumeConfig:      consumeConfig,
	}

	channel, err := rabbitMqConnection.Channel()
	if err != nil {
		return nil, err
	}
	rabbitmqConsumer.channel = channel

	return &rabbitmqConsumer, err
}

func (c *rabbitmqConsumer) DeclareQueue(config config.QueueConfig) error {
	if c.channel == nil || c.channel.IsClosed() {
		ch, err := c.Channel()
		if err != nil {
			return fmt.Errorf("open channel: %w", err)
		}
		c.channel = ch
	}

	q, err := c.channel.QueueDeclare(
		config.Name,
		config.Durable,
		config.AutoDelete,
		config.Exclusive,
		config.NoWait,
		config.Args,
	)
	if err != nil {
		return fmt.Errorf("declare queue %q: %w", config.Name, err)
	}

	c.queue = &q
	return nil
}

func (c *rabbitmqConsumer) BindQueue(config config.BindingConfig) error {
	if c.channel == nil || c.channel.IsClosed() {
		ch, err := c.Channel()
		if err != nil {
			return fmt.Errorf("open channel: %w", err)
		}
		c.channel = ch
	}

	if c.queue == nil {
		return fmt.Errorf("queue not declared")
	}

	if config.ExchangeName == "" {
		return fmt.Errorf("exchange name is required")
	}

	if err := c.channel.QueueBind(
		c.queue.Name,
		config.RoutingKey,
		config.ExchangeName,
		config.NoWait,
		config.Args,
	); err != nil {
		return fmt.Errorf(
			"bind queue %q to exchange %q with routing key %q: %w",
			c.queue.Name,
			c.queue.Name,
			config.ExchangeName,
			err,
		)
	}

	return nil
}

func (c *rabbitmqConsumer) Consume(ctx context.Context) (<-chan amqp.Delivery, error) {
	if c.channel == nil {
		slog.InfoContext(ctx, "channel not opened, opening channel")

		ch, err := c.Channel()
		if err != nil {
			slog.ErrorContext(ctx, "failed to open channel", "error", err)
			return nil, &mqErrors.UnexpectedErr{Msg: "unexpected error when reopening channel", Err: err}
		}
		c.channel = ch
	}

	deliveries, err := c.channel.ConsumeWithContext(
		ctx,
		c.queue.Name,
		c.consumeConfig.Consumer,
		c.consumeConfig.AutoAck,
		c.consumeConfig.Exclusive,
		c.consumeConfig.AutoAck,
		c.consumeConfig.NoLocal,
		c.consumeConfig.Args,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to consume queue", "error", err)
		return nil, &mqErrors.UnexpectedErr{Msg: fmt.Sprintf("start consuming queue %q", c.queue.Name), Err: err}
	}

	return deliveries, nil
}

func (c *rabbitmqConsumer) CloseChannel() error {
	return c.channel.Close()
}
