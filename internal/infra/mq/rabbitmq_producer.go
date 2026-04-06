package mq

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/mqErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

type rabbitmqProducer struct {
	*RabbitMqChannel
	exchangeName  string
	exchangeKind  string
	publishConfig config.PublishConfig
}

func NewRabbitmqProducer(rabbitmqConnection *RabbitMqConnection, publishConfig config.PublishConfig) (port.MqProducer, error) {
	channel, err := NewRabbitMqChannel(rabbitmqConnection)
	if err != nil {
		return nil, err
	}

	producer := rabbitmqProducer{
		RabbitMqChannel: channel,
		publishConfig:   publishConfig,
	}

	if err := producer.openChannel(); err != nil {
		return nil, err
	}

	return &producer, nil
}

func (p *rabbitmqProducer) DeclareExchange(config config.ExchangeConfig) error {
	p.exchangeName = config.Name
	p.exchangeKind = config.Kind
	if config.Kind == "" {
		slog.Warn("exchange kind not specified, defaulting to 'direct'")
		p.exchangeKind = "direct"
	}

	if p.channel == nil || p.channel.IsClosed() {
		if err := p.reopenChannel(context.Background()); err != nil {
			return err
		}
	}

	if err := p.channel.ExchangeDeclare(
		p.exchangeName,
		p.exchangeKind,
		config.Durable,
		config.AutoDelete,
		config.Internal,
		config.NoWait,
		nil,
	); err != nil {
		return fmt.Errorf("declare exchange %q: %w", config.Name, err)
	}

	return nil
}

func (p *rabbitmqProducer) Publish(ctx context.Context, message proto.Message, routingKey string) error {
	if p.channel == nil || p.channel.IsClosed() {
		if err := p.reopenChannel(ctx); err != nil {
			return err
		}
	}

	payload, err := proto.Marshal(message)
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal protobuf message", "error", err)
		return &mqErrors.UnexpectedErr{Msg: "failed to marshal protobuf message", Err: err}
	}

	headers := amqp.Table{
		"message_type": string(message.ProtoReflect().Descriptor().FullName()),
	}
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		carrier := propagation.MapCarrier{}
		otel.GetTextMapPropagator().Inject(ctx, carrier)
		for k, v := range carrier {
			headers[k] = v
		}
	}

	if err := p.channel.PublishWithContext(
		ctx,
		p.exchangeName,
		routingKey,
		p.publishConfig.Mandatory,
		p.publishConfig.Immediate,
		amqp.Publishing{
			ContentType:  "application/protobuf",
			Body:         payload,
			DeliveryMode: amqp.Persistent,
			Headers:      headers,
			Timestamp:    time.Now(),
		},
	); err != nil {
		slog.ErrorContext(ctx, "failed to publish protobuf message", "error", err)
		return &mqErrors.UnexpectedErr{Msg: "failed to publish protobuf message", Err: err}
	}

	return nil
}
