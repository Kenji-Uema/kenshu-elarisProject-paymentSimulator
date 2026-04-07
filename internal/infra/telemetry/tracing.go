package telemetry

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	appTracerName      = "github.com/Kenji-Uema/paymentSimulator/internal/app"
	consumerTracerName = "github.com/Kenji-Uema/paymentSimulator/internal/infra/mq"
)

func StartAppSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return otel.Tracer(appTracerName).Start(ctx, name, trace.WithAttributes(attrs...))
}

func RecordSpanError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func StartConsumerSpan(ctx context.Context, delivery amqp.Delivery, queueName string) (context.Context, trace.Span) {
	ctx = otel.GetTextMapPropagator().Extract(ctx, deliveryHeadersCarrier(delivery.Headers))

	attrs := []attribute.KeyValue{
		attribute.String("messaging.system", "rabbitmq"),
		attribute.String("messaging.operation", "process"),
		attribute.String("messaging.destination.name", queueName),
		attribute.String("messaging.rabbitmq.routing_key", delivery.RoutingKey),
		attribute.Int64("messaging.message.body.size", int64(len(delivery.Body))),
		attribute.Int64("messaging.rabbitmq.delivery_tag", int64(delivery.DeliveryTag)),
	}
	if delivery.Exchange != "" {
		attrs = append(attrs, attribute.String("messaging.rabbitmq.exchange", delivery.Exchange))
	}
	if delivery.MessageId != "" {
		attrs = append(attrs, attribute.String("messaging.message.id", delivery.MessageId))
	}
	if delivery.CorrelationId != "" {
		attrs = append(attrs, attribute.String("messaging.message.conversation_id", delivery.CorrelationId))
	}
	if messageType := fmt.Sprint(delivery.Headers["message_type"]); messageType != "" && messageType != "<nil>" {
		attrs = append(attrs, attribute.String("messaging.message.type", messageType))
	}

	return otel.Tracer(consumerTracerName).Start(
		ctx,
		consumerSpanName(queueName, delivery),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attrs...),
	)
}

func RecordDeliveryError(span trace.Span, err error) {
	RecordSpanError(span, err)
}

func consumerSpanName(queueName string, delivery amqp.Delivery) string {
	if delivery.RoutingKey != "" {
		return fmt.Sprintf("%s receive %s", queueName, delivery.RoutingKey)
	}
	return fmt.Sprintf("%s receive", queueName)
}

type deliveryHeadersCarrier amqp.Table

func (c deliveryHeadersCarrier) Get(key string) string {
	value, ok := c[key]
	if !ok {
		return ""
	}
	return fmt.Sprint(value)
}

func (c deliveryHeadersCarrier) Set(key string, value string) {
	c[key] = value
}

func (c deliveryHeadersCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	return keys
}

var _ propagation.TextMapCarrier = deliveryHeadersCarrier{}
