package app

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	clockfakes "github.com/Kenji-Uema/paymentSimulator/internal/infra/clock/fakes"
	dbfakes "github.com/Kenji-Uema/paymentSimulator/internal/infra/db/fakes"
	mqfakes "github.com/Kenji-Uema/paymentSimulator/internal/infra/mq/fakes"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/appErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestInvoiceService_GenerateInvoiceDoc(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
		service := &invoiceService{
			clock: &clockfakes.FakeClock{
				NowFn: func(context.Context) (*time.Time, error) {
					return &now, nil
				},
			},
		}

		req := validCreateInvoiceRequest()
		invoice, err := service.generateInvoiceDoc(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if invoice == (document.Invoice{}) {
			t.Fatal("expected non-empty invoice")
		}
	})

	t.Run("clock error returns unexpected", func(t *testing.T) {
		service := &invoiceService{
			clock: &clockfakes.FakeClock{
				NowFn: func(context.Context) (*time.Time, error) {
					return nil, errors.New("clock down")
				},
			},
		}

		_, err := service.generateInvoiceDoc(context.Background(), validCreateInvoiceRequest())
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var unexpectedErr *appErrors.UnexpectedErr
		if !errors.As(err, &unexpectedErr) {
			t.Fatalf("expected *appErrors.UnexpectedErr, got %T", err)
		}
	})

	t.Run("invalid request returns corrupted data error", func(t *testing.T) {
		now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
		service := &invoiceService{
			clock: &clockfakes.FakeClock{
				NowFn: func(context.Context) (*time.Time, error) {
					return &now, nil
				},
			},
		}

		req := validCreateInvoiceRequest()
		req.Payer.Name = ""

		_, err := service.generateInvoiceDoc(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var corruptedErr *appErrors.CorruptedDataError
		if !errors.As(err, &corruptedErr) {
			t.Fatalf("expected *appErrors.CorruptedDataError, got %T", err)
		}
	})
}

func TestNewInvoiceService(t *testing.T) {
	t.Run("invalid dependencies", func(t *testing.T) {
		_, err := NewInvoiceService(nil, nil, nil, nil, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		service, err := NewInvoiceService(
			&dbfakes.FakeInvoiceRepo{},
			&clockfakes.FakeClock{},
			&mqfakes.FakeMqConsumer{},
			&mqfakes.FakeMqProducer{},
			"https://pay.local",
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if service == nil {
			t.Fatal("expected non-nil service")
		}
	})
}

func TestInvoiceService_SaveInvoice(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		insertedId := bson.NewObjectID()
		repo := &dbfakes.FakeInvoiceRepo{
			AddFn: func(context.Context, document.Invoice) (bson.ObjectID, error) {
				return insertedId, nil
			},
		}
		service := &invoiceService{invoiceRepo: repo}

		gotID, err := service.saveInvoice(context.Background(), validInvoiceDocument())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if gotID != insertedId {
			t.Fatalf("expected %s got %s", insertedId.Hex(), gotID.Hex())
		}
	})

	t.Run("duplicate key", func(t *testing.T) {
		repo := &dbfakes.FakeInvoiceRepo{
			AddFn: func(context.Context, document.Invoice) (bson.ObjectID, error) {
				return bson.NilObjectID, &dbErrors.AlreadyExistsErr{Err: errors.New("duplicate key")}
			},
		}
		service := &invoiceService{invoiceRepo: repo}

		_, err := service.saveInvoice(context.Background(), validInvoiceDocument())
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var alreadyExistsErr *appErrors.AlreadyExistsErr
		if !errors.As(err, &alreadyExistsErr) {
			t.Fatalf("expected *appErrors.AlreadyExistsErr, got %T", err)
		}
	})

	t.Run("unexpected db error", func(t *testing.T) {
		repo := &dbfakes.FakeInvoiceRepo{
			AddFn: func(context.Context, document.Invoice) (bson.ObjectID, error) {
				return bson.NilObjectID, &dbErrors.UnexpectedErr{Msg: "db failure", Err: errors.New("boom")}
			},
		}
		service := &invoiceService{invoiceRepo: repo}

		_, err := service.saveInvoice(context.Background(), validInvoiceDocument())
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var unexpectedErr *appErrors.UnexpectedErr
		if !errors.As(err, &unexpectedErr) {
			t.Fatalf("expected *appErrors.UnexpectedErr, got %T", err)
		}
	})
}

func TestInvoiceService_PublishPaymentRequest(t *testing.T) {
	paymentHost := "https://pay.local"

	t.Run("success", func(t *testing.T) {
		paymentProducer := &mqfakes.FakeMqProducer{}
		service := &invoiceService{
			paymentProducer: paymentProducer,
			paymentHost:     paymentHost,
		}

		invoice := validInvoiceDocument()
		err := service.publishPaymentRequest(context.Background(), invoice)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if paymentProducer.PublishCallCount != 1 {
			t.Fatalf("expected Publish to be called once, got %d", paymentProducer.PublishCallCount)
		}
		expectedRoutingKey := fmt.Sprintf("guest.%s", invoice.PayerId)
		if paymentProducer.LastPublishedRoutingKey != expectedRoutingKey {
			t.Fatalf("unexpected routing key: %q", paymentProducer.LastPublishedRoutingKey)
		}
	})

	t.Run("invalid request", func(t *testing.T) {
		paymentProducer := &mqfakes.FakeMqProducer{}
		service := &invoiceService{
			paymentProducer: paymentProducer,
			paymentHost:     paymentHost,
		}

		invoice := validInvoiceDocument()
		invoice.Total.Amount = 0

		err := service.publishPaymentRequest(context.Background(), invoice)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
		}
		if paymentProducer.PublishCallCount != 0 {
			t.Fatalf("expected Publish not to be called, got %d", paymentProducer.PublishCallCount)
		}
	})

	t.Run("publish error", func(t *testing.T) {
		paymentProducer := &mqfakes.FakeMqProducer{
			PublishFn: func(ctx context.Context, message proto.Message, routingKey string) error {
				return errors.New("publish failed")
			},
		}
		service := &invoiceService{
			paymentProducer: paymentProducer,
			paymentHost:     paymentHost,
		}

		invoice := validInvoiceDocument()

		err := service.publishPaymentRequest(context.Background(), invoice)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var publishErr *appErrors.PublishErr
		if !errors.As(err, &publishErr) {
			t.Fatalf("expected *appErrors.PublishErr, got %T", err)
		}
	})
}

func TestInvoiceService_ProcessInvoiceDelivery(t *testing.T) {
	t.Run("invalid payload nacks without requeue", func(t *testing.T) {
		ack := &mqfakes.FakeAcknowledger{}
		delivery := amqp.Delivery{Acknowledger: ack, DeliveryTag: 99, Body: []byte("{")}

		service := &invoiceService{}
		err := service.processInvoiceDelivery(context.Background(), delivery)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var corruptedErr *appErrors.CorruptedDataError
		if !errors.As(err, &corruptedErr) {
			t.Fatalf("expected *appErrors.CorruptedDataError, got %T", err)
		}
		if ack.NackCalls != 1 || ack.LastNackRequeue {
			t.Fatalf("expected one nack with requeue=false, got calls=%d requeue=%v", ack.NackCalls, ack.LastNackRequeue)
		}
	})

	t.Run("generateInvoiceDoc err nacks without requeue", func(t *testing.T) {
		ack := &mqfakes.FakeAcknowledger{}
		delivery := validDelivery(t, ack)
		service := &invoiceService{clock: &clockfakes.FakeClock{NowFn: func(context.Context) (*time.Time, error) {
			return nil, errors.New("clock down")
		}}}

		err := service.processInvoiceDelivery(context.Background(), delivery)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var unexpectedErr *appErrors.UnexpectedErr
		if !errors.As(err, &unexpectedErr) {
			t.Fatalf("expected *appErrors.UnexpectedErr, got %T", err)
		}
		if ack.NackCalls != 1 || ack.LastNackRequeue {
			t.Fatalf("expected one nack with requeue=false, got calls=%d requeue=%v", ack.NackCalls, ack.LastNackRequeue)
		}
	})

	t.Run("save invoice duplicate key nacks without requeue", func(t *testing.T) {
		ack := &mqfakes.FakeAcknowledger{}
		delivery := validDelivery(t, ack)
		now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
		service := &invoiceService{
			clock: &clockfakes.FakeClock{NowFn: func(context.Context) (*time.Time, error) { return &now, nil }},
			invoiceRepo: &dbfakes.FakeInvoiceRepo{AddFn: func(context.Context, document.Invoice) (bson.ObjectID, error) {
				return bson.NilObjectID, &dbErrors.AlreadyExistsErr{Err: errors.New("duplicate")}
			}},
		}

		err := service.processInvoiceDelivery(context.Background(), delivery)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var alreadyExistsErr *appErrors.AlreadyExistsErr
		if !errors.As(err, &alreadyExistsErr) {
			t.Fatalf("expected *appErrors.AlreadyExistsErr, got %T", err)
		}
		if ack.NackCalls != 1 || ack.LastNackRequeue {
			t.Fatalf("expected one nack with requeue=false, got calls=%d requeue=%v", ack.NackCalls, ack.LastNackRequeue)
		}
	})

	t.Run("save invoice unexpected nacks with requeue", func(t *testing.T) {
		ack := &mqfakes.FakeAcknowledger{}
		delivery := validDelivery(t, ack)
		now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
		service := &invoiceService{
			clock: &clockfakes.FakeClock{NowFn: func(context.Context) (*time.Time, error) { return &now, nil }},
			invoiceRepo: &dbfakes.FakeInvoiceRepo{AddFn: func(context.Context, document.Invoice) (bson.ObjectID, error) {
				return bson.NilObjectID, &dbErrors.UnexpectedErr{Msg: "db", Err: errors.New("down")}
			}},
		}

		err := service.processInvoiceDelivery(context.Background(), delivery)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if ack.NackCalls != 1 || !ack.LastNackRequeue {
			t.Fatalf("expected one nack with requeue=true, got calls=%d requeue=%v", ack.NackCalls, ack.LastNackRequeue)
		}
	})

	t.Run("publish validation error nacks without requeue", func(t *testing.T) {
		ack := &mqfakes.FakeAcknowledger{}
		delivery := validDelivery(t, ack)
		now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
		service := &invoiceService{
			clock: &clockfakes.FakeClock{NowFn: func(context.Context) (*time.Time, error) { return &now, nil }},
			invoiceRepo: &dbfakes.FakeInvoiceRepo{AddFn: func(context.Context, document.Invoice) (bson.ObjectID, error) {
				return bson.NewObjectID(), nil
			}},
			paymentProducer: &mqfakes.FakeMqProducer{PublishFn: func(ctx context.Context, message proto.Message, routingKey string) error {
				_ = ctx
				_ = message
				_ = routingKey
				return &validationErrors.ErrValidationConstrain{Field: "x", Message: "bad"}
			}},
			paymentHost: "https://pay.local",
		}

		err := service.processInvoiceDelivery(context.Background(), delivery)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
		}
		if ack.NackCalls != 1 || ack.LastNackRequeue {
			t.Fatalf("expected one nack with requeue=false, got calls=%d requeue=%v", ack.NackCalls, ack.LastNackRequeue)
		}
	})

	t.Run("publish unexpected error nacks with requeue", func(t *testing.T) {
		ack := &mqfakes.FakeAcknowledger{}
		delivery := validDelivery(t, ack)
		now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
		service := &invoiceService{
			clock: &clockfakes.FakeClock{NowFn: func(context.Context) (*time.Time, error) { return &now, nil }},
			invoiceRepo: &dbfakes.FakeInvoiceRepo{AddFn: func(context.Context, document.Invoice) (bson.ObjectID, error) {
				return bson.NewObjectID(), nil
			}},
			paymentProducer: &mqfakes.FakeMqProducer{PublishFn: func(ctx context.Context, message proto.Message, routingKey string) error {
				_ = ctx
				_ = message
				_ = routingKey
				return errors.New("publish failed")
			}},
			paymentHost: "https://pay.local",
		}

		err := service.processInvoiceDelivery(context.Background(), delivery)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if ack.NackCalls != 1 || !ack.LastNackRequeue {
			t.Fatalf("expected one nack with requeue=true, got calls=%d requeue=%v", ack.NackCalls, ack.LastNackRequeue)
		}
	})

	t.Run("success acks", func(t *testing.T) {
		ack := &mqfakes.FakeAcknowledger{}
		delivery := validDelivery(t, ack)
		now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
		service := &invoiceService{
			clock: &clockfakes.FakeClock{NowFn: func(context.Context) (*time.Time, error) { return &now, nil }},
			invoiceRepo: &dbfakes.FakeInvoiceRepo{AddFn: func(context.Context, document.Invoice) (bson.ObjectID, error) {
				return bson.NewObjectID(), nil
			}},
			paymentProducer: &mqfakes.FakeMqProducer{},
			paymentHost:     "https://pay.local",
		}

		err := service.processInvoiceDelivery(context.Background(), delivery)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if ack.AckCalls != 1 || ack.NackCalls != 0 {
			t.Fatalf("expected ack once and no nack, got ack=%d nack=%d", ack.AckCalls, ack.NackCalls)
		}
	})
}

func TestInvoiceService_StartInvoiceProcessing(t *testing.T) {
	t.Run("consume error returns without processing", func(t *testing.T) {
		consumeErr := errors.New("mq down")
		invoiceConsumer := &mqfakes.FakeMqConsumer{
			ConsumeFn: func(context.Context) (<-chan amqp.Delivery, error) {
				return nil, consumeErr
			},
		}
		service := &invoiceService{invoiceConsumer: invoiceConsumer}

		service.StartInvoiceProcessing(context.Background())

		if invoiceConsumer.ConsumeCallCount != 1 {
			t.Fatalf("expected Consume to be called once, got %d", invoiceConsumer.ConsumeCallCount)
		}
	})

	t.Run("processes deliveries from channel", func(t *testing.T) {
		ack := &mqfakes.FakeAcknowledger{}
		deliveries := make(chan amqp.Delivery, 1)
		deliveries <- validDelivery(t, ack)
		close(deliveries)

		invoiceConsumer := &mqfakes.FakeMqConsumer{
			ConsumeFn: func(context.Context) (<-chan amqp.Delivery, error) { return deliveries, nil },
		}
		now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
		repo := &dbfakes.FakeInvoiceRepo{AddFn: func(context.Context, document.Invoice) (bson.ObjectID, error) {
			return bson.NewObjectID(), nil
		}}
		producer := &mqfakes.FakeMqProducer{}
		service := &invoiceService{
			clock:           &clockfakes.FakeClock{NowFn: func(context.Context) (*time.Time, error) { return &now, nil }},
			invoiceRepo:     repo,
			invoiceConsumer: invoiceConsumer,
			paymentProducer: producer,
			paymentHost:     "https://pay.local",
		}

		service.StartInvoiceProcessing(context.Background())

		if ack.AckCalls != 1 {
			t.Fatalf("expected ack to be called once, got %d", ack.AckCalls)
		}
		if repo.AddCallCount != 1 {
			t.Fatalf("expected add to be called once, got %d", repo.AddCallCount)
		}
		if producer.PublishCallCount != 1 {
			t.Fatalf("expected publish to be called once, got %d", producer.PublishCallCount)
		}
	})
}

func validDelivery(t *testing.T, ack *mqfakes.FakeAcknowledger) amqp.Delivery {
	t.Helper()

	body, err := protojson.Marshal(validCreateInvoiceRequest())
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	return amqp.Delivery{Acknowledger: ack, DeliveryTag: 1, Body: body}
}

func validCreateInvoiceRequest() *dto.CreateInvoicePaymentRequest {
	now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
	return &dto.CreateInvoicePaymentRequest{
		IdempotencyKey: "idem-1",
		BookingId:      "booking-123",
		PayerId:        "payer-123",
		IssuedAt:       timestamppb.New(now),
		DueAt:          timestamppb.New(now.Add(24 * time.Hour)),
		Payer: &dto.Payer{
			Name:           "John Doe",
			Email:          "john@example.com",
			DocumentNumber: "11122233344",
			BillingAddress: "Main St",
		},
		Booking: &dto.BookingSnapshot{
			CottageName:    "Cabin",
			Nights:         2,
			NumberOfGuests: 3,
			ValuePerNight: &dto.Money{
				Amount:   10000,
				Currency: "USD",
			},
		},
		Total:         &dto.Money{Amount: 20000, Currency: "USD"},
		TaxTotal:      &dto.Money{Amount: 1000, Currency: "USD"},
		DiscountTotal: &dto.Money{Amount: 0, Currency: "USD"},
	}
}

func validInvoiceDocument() document.Invoice {
	now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)

	return document.Invoice{
		InvoiceNumber: "INV-20260309-ABC123",
		IssuedAt:      now,
		DueAt:         now.Add(24 * time.Hour),
		PayerId:       "payer-123",
		Total: document.Money{
			Amount:   25000,
			Currency: "USD",
		},
		Booking: document.BookingSnapshot{
			CottageName:    "Mountain View Cabin",
			Nights:         2,
			NumberOfGuests: 4,
		},
		Payer: document.Payer{
			Name:  "John Doe",
			Email: "john@example.com",
		},
	}
}
