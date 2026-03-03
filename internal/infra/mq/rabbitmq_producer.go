package mq

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type rabbitmqProducer struct {
	*RabbitMqConnection
	channel      *amqp.Channel
	exchangeName string
	exchangeKind string
}

func NewRabbitmqProducer(rabbitmqConnection *RabbitMqConnection) (port.MqProducer, error) {
	paymentProducer := rabbitmqProducer{
		RabbitMqConnection: rabbitmqConnection,
	}

	channel, err := rabbitmqConnection.Channel()
	if err != nil {
		return nil, err
	}
	paymentProducer.channel = channel

	return &paymentProducer, nil
}

func (p *rabbitmqProducer) DeclareExchange(config config.ExchangeConfig) error {
	p.exchangeName = config.Name
	if config.Kind == "" {
		slog.Warn("exchange kind not specified, defaulting to 'direct'")
		p.exchangeKind = "direct"
	}

	if p.channel == nil || p.channel.IsClosed() {
		ch, err := p.Channel()
		if err != nil {
			return fmt.Errorf("open channel: %w", err)
		}
		p.channel = ch
	}

	if err := p.channel.ExchangeDeclare(p.exchangeName, p.exchangeKind,
		config.Durable, config.AutoDelete, config.Internal,
		config.NoWait, config.Args); err != nil {

		return fmt.Errorf("declare exchange %q: %w", config.Name, err)
	}

	return nil
}

func (p *rabbitmqProducer) Publish(ctx context.Context, message proto.Message, config config.PublishConfig) error {
	if p.channel == nil || p.channel.IsClosed() {
		ch, err := p.Channel()
		if err != nil {
			return fmt.Errorf("open channel: %w", err)
		}
		p.channel = ch
	}

	payload, err := protojson.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal payment confirmation: %w", err)
	}

	headers := amqp.Table{}
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
		config.RoutingKey,
		config.Mandatory,
		config.Immediate,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         payload,
			DeliveryMode: amqp.Persistent,
			Headers:      headers,
			Timestamp:    time.Now(),
		},
	); err != nil {
		return fmt.Errorf("publish payment confirmation: %w", err)
	}

	return nil
}

func (p *rabbitmqProducer) CloseChannel() error {
	return p.channel.Close()
}
