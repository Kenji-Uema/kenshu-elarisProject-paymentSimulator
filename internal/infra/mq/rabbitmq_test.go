package mq

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRabbitmqConsumer_BindQueue(t *testing.T) {
	setupAndRun("edge cases", t, func(t *testing.T, consumer port.MqConsumer, producer port.MqProducer) {
		t.Run("returns error when queue is not declared", func(t *testing.T) {
			err := consumer.BindQueue(context.Background(), config.BindingConfig{
				ExchangeName: "exchange.does.not.matter",
				RoutingKey:   "rk.test",
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("returns error when exchange name is empty", func(t *testing.T) {
			queueName := fmt.Sprintf("consumer.bind.queue.%d", time.Now().UnixNano())
			if err := consumer.DeclareQueue(context.Background(), config.QueueConfig{Name: queueName, AutoDelete: true}); err != nil {
				t.Fatalf("declare queue: %v", err)
			}

			err := consumer.BindQueue(context.Background(), config.BindingConfig{
				ExchangeName: "",
				RoutingKey:   "rk.test",
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}

func TestRabbitmqProducer_Publish(t *testing.T) {
	setupAndRun("publish flow", t, func(t *testing.T, consumer port.MqConsumer, producer port.MqProducer) {
		exchangeName := fmt.Sprintf("producer.publish.exchange.%d", time.Now().UnixNano())
		queueName := fmt.Sprintf("producer.publish.queue.%d", time.Now().UnixNano())
		routingKey := "payment.created"

		if err := producer.DeclareExchange(config.ExchangeConfig{Name: exchangeName, Kind: "direct"}); err != nil {
			t.Fatalf("declare exchange: %v", err)
		}
		if err := consumer.DeclareQueue(context.Background(), config.QueueConfig{Name: queueName, AutoDelete: true}); err != nil {
			t.Fatalf("declare queue: %v", err)
		}
		if err := consumer.BindQueue(context.Background(), config.BindingConfig{ExchangeName: exchangeName, RoutingKey: routingKey}); err != nil {
			t.Fatalf("bind queue: %v", err)
		}

		consumeCtx, cancelConsume := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelConsume()
		deliveries, err := consumer.Consume(consumeCtx)
		if err != nil {
			t.Fatalf("consume: %v", err)
		}

		msg := &dto.PaymentConfirmation{
			Id:            "id-1",
			BookingId:     "booking-1",
			PayerId:       "payer-1",
			InvoiceNumber: "INV-2026-0001",
			ReceiptNumber: "RCPT-2026-0001",
			ConfirmedAt:   timestamppb.Now(),
		}

		if err := producer.Publish(context.Background(), msg, routingKey); err != nil {
			t.Fatalf("publish: %v", err)
		}

		select {
		case d, ok := <-deliveries:
			if !ok {
				t.Fatal("deliveries channel closed before receiving message")
			}
			if d.ContentType != "application/protobuf" {
				t.Fatalf("unexpected content type: got=%q want=%q", d.ContentType, "application/protobuf")
			}
			if got := d.Headers["message_type"]; got != "paymentSimulator.payment.v1.PaymentConfirmation" {
				t.Fatalf("unexpected message_type header: got=%v", got)
			}
			if err := d.Ack(false); err != nil {
				t.Fatalf("ack: %v", err)
			}
		case <-consumeCtx.Done():
			t.Fatalf("timed out waiting for delivery: %v", consumeCtx.Err())
		}
	})
}

func TestRabbitmqConsumer_Consume(t *testing.T) {
	setupAndRun("consume flow", t, func(t *testing.T, consumer port.MqConsumer, producer port.MqProducer) {
		exchangeName := fmt.Sprintf("consumer.consume.exchange.%d", time.Now().UnixNano())
		queueName := fmt.Sprintf("consumer.consume.queue.%d", time.Now().UnixNano())
		routingKey := "payment.created"

		if err := producer.DeclareExchange(config.ExchangeConfig{Name: exchangeName, Kind: "direct"}); err != nil {
			t.Fatalf("declare exchange: %v", err)
		}
		if err := consumer.DeclareQueue(context.Background(), config.QueueConfig{Name: queueName, AutoDelete: true}); err != nil {
			t.Fatalf("declare queue: %v", err)
		}
		if err := consumer.BindQueue(context.Background(), config.BindingConfig{ExchangeName: exchangeName, RoutingKey: routingKey}); err != nil {
			t.Fatalf("bind queue: %v", err)
		}

		consumeCtx, cancelConsume := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelConsume()
		deliveries, err := consumer.Consume(consumeCtx)
		if err != nil {
			t.Fatalf("consume: %v", err)
		}

		msg := &dto.PaymentConfirmation{
			Id:            "id-1",
			BookingId:     "booking-1",
			PayerId:       "payer-1",
			InvoiceNumber: "INV-2026-0001",
			ReceiptNumber: "RCPT-2026-0001",
			ConfirmedAt:   timestamppb.Now(),
		}

		if err := producer.Publish(context.Background(), msg, routingKey); err != nil {
			t.Fatalf("publish: %v", err)
		}

		select {
		case d, ok := <-deliveries:
			if !ok {
				t.Fatal("deliveries channel closed before receiving message")
			}
			if d.RoutingKey != routingKey {
				t.Fatalf("unexpected routing key: got=%q want=%q", d.RoutingKey, routingKey)
			}
			if err := d.Ack(false); err != nil {
				t.Fatalf("ack: %v", err)
			}
		case <-consumeCtx.Done():
			t.Fatalf("timed out waiting for delivery: %v", consumeCtx.Err())
		}

		// Explicitly stop consuming and wait for the channel to close to avoid
		// blocking on channel teardown in parent test cleanup.
		cancelConsume()
		select {
		case _, ok := <-deliveries:
			if ok {
				t.Fatal("expected deliveries channel to close after cancel")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting deliveries channel to close after cancel")
		}
	})
}
