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
	*RabbitMqChannel
	queue         *amqp.Queue
	consumeConfig config.ConsumeConfig
}

func NewRabbitmqConsumer(rabbitMqConnection *RabbitMqConnection, consumeConfig config.ConsumeConfig) (port.MqConsumer, error) {
	rabbitmqConsumer := rabbitmqConsumer{
		RabbitMqChannel: NewRabbitMqChannel(rabbitMqConnection),
		consumeConfig:   consumeConfig,
	}

	if err := rabbitmqConsumer.openChannel(); err != nil {
		return nil, err
	}

	return &rabbitmqConsumer, nil
}

func (c *rabbitmqConsumer) DeclareQueue(ctx context.Context, config config.QueueConfig) error {
	if c.channel == nil || c.channel.IsClosed() {
		if err := c.reopenChannel(ctx); err != nil {
			return err
		}
	}

	q, err := c.channel.QueueDeclare(
		config.Name,
		config.Durable,
		config.AutoDelete,
		config.Exclusive,
		config.NoWait,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare queue %q: %w", config.Name, err)
	}

	c.queue = &q
	return nil
}

func (c *rabbitmqConsumer) BindQueue(ctx context.Context, config config.BindingConfig) error {
	if c.channel == nil || c.channel.IsClosed() {
		if err := c.reopenChannel(ctx); err != nil {
			return err
		}
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
		nil,
	); err != nil {
		return fmt.Errorf(
			"bind queue %q to exchange %q with routing key %q: %w",
			c.queue.Name,
			config.ExchangeName,
			config.RoutingKey,
			err,
		)
	}

	return nil
}

func (c *rabbitmqConsumer) Consume(ctx context.Context) (<-chan amqp.Delivery, error) {
	if c.channel == nil {
		if err := c.reopenChannel(ctx); err != nil {
			return nil, err
		}
	}

	deliveries, err := c.channel.ConsumeWithContext(
		ctx,
		c.queue.Name,
		c.consumeConfig.Consumer,
		c.consumeConfig.AutoAck,
		c.consumeConfig.Exclusive,
		c.consumeConfig.NoLocal,
		c.consumeConfig.NoWait,
		nil,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to consume queue", "error", err)
		return nil, &mqErrors.UnexpectedErr{Msg: fmt.Sprintf("start consuming queue %q", c.queue.Name), Err: err}
	}

	return deliveries, nil
}
