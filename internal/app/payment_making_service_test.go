package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
	clockfakes "github.com/Kenji-Uema/paymentSimulator/internal/infra/clock/fakes"
	dbfakes "github.com/Kenji-Uema/paymentSimulator/internal/infra/db/fakes"
	mqfakes "github.com/Kenji-Uema/paymentSimulator/internal/infra/mq/fakes"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/appErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/dbErrors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestPaymentMakingService_BuildResponse(t *testing.T) {
	s := &paymentMakingService{}
	now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)

	resp, err := s.buildResponse(validPayWithCardRequest(), dto.PaymentStatus_PAYMENT_STATUS_SUCCEEDED, now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp == (&dto.PayWithCardResponse{}) {
		t.Fatal("expected non-empty response")
	}
}

func TestPaymentMakingService_SaveReceipt(t *testing.T) {
	t.Run("invalid response returns validation error", func(t *testing.T) {
		s := &paymentMakingService{receiptRepo: &dbfakes.FakeReceiptRepo{}}
		_, err := s.saveReceipt(context.Background(), &dto.PayWithCardResponse{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
		}
	})

	t.Run("already exists", func(t *testing.T) {
		s := &paymentMakingService{receiptRepo: &dbfakes.FakeReceiptRepo{AddFn: func(context.Context, document.Receipt) (bson.ObjectID, error) {
			return bson.NilObjectID, &dbErrors.AlreadyExistsErr{Err: errors.New("duplicate")}
		}}}
		_, err := s.saveReceipt(context.Background(), validPayWithCardResponse())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var alreadyExistsErr *appErrors.AlreadyExistsErr
		if !errors.As(err, &alreadyExistsErr) {
			t.Fatalf("expected *appErrors.AlreadyExistsErr, got %T", err)
		}
	})

	t.Run("corrupted data", func(t *testing.T) {
		s := &paymentMakingService{receiptRepo: &dbfakes.FakeReceiptRepo{AddFn: func(context.Context, document.Receipt) (bson.ObjectID, error) {
			return bson.NilObjectID, &dbErrors.CorruptedDataErr{Err: errors.New("bad data")}
		}}}
		_, err := s.saveReceipt(context.Background(), validPayWithCardResponse())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var corruptedErr *appErrors.CorruptedDataError
		if !errors.As(err, &corruptedErr) {
			t.Fatalf("expected *appErrors.CorruptedDataError, got %T", err)
		}
	})

	t.Run("generic db error maps to app unexpected", func(t *testing.T) {
		s := &paymentMakingService{receiptRepo: &dbfakes.FakeReceiptRepo{AddFn: func(context.Context, document.Receipt) (bson.ObjectID, error) {
			return bson.NilObjectID, errors.New("db down")
		}}}
		_, err := s.saveReceipt(context.Background(), validPayWithCardResponse())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var unexpectedErr *appErrors.UnexpectedErr
		if !errors.As(err, &unexpectedErr) {
			t.Fatalf("expected *appErrors.UnexpectedErr, got %T", err)
		}
	})

	t.Run("success stores and returns receipt", func(t *testing.T) {
		repo := &dbfakes.FakeReceiptRepo{AddFn: func(context.Context, document.Receipt) (bson.ObjectID, error) {
			return bson.NewObjectID(), nil
		}}
		s := &paymentMakingService{receiptRepo: repo}
		receipt, err := s.saveReceipt(context.Background(), validPayWithCardResponse())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if receipt == (document.Receipt{}) {
			t.Fatal("expected non-empty receipt")
		}
	})
}

func TestPaymentMakingService_SendConfirmationMessage(t *testing.T) {
	now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
	resp := validPayWithCardResponse()

	t.Run("invoice not found", func(t *testing.T) {
		s := &paymentMakingService{invoiceRepo: &dbfakes.FakeInvoiceRepo{
			GetFn: func(context.Context, string) (document.Invoice, error) {
				return document.Invoice{}, &dbErrors.InvoiceNotFoundErr{Err: errors.New("not found")}
			}}}

		err := s.sendConfirmationMessage(context.Background(), &now, resp)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var notFoundErr *appErrors.InvoiceNotFoundErr
		if !errors.As(err, &notFoundErr) {
			t.Fatalf("expected *appErrors.InvoiceNotFoundErr, got %T", err)
		}
	})

	t.Run("corrupted invoice", func(t *testing.T) {
		s := &paymentMakingService{invoiceRepo: &dbfakes.FakeInvoiceRepo{GetFn: func(context.Context, string) (document.Invoice, error) {
			return document.Invoice{}, &dbErrors.CorruptedDataErr{Err: errors.New("bad")}
		}}}
		err := s.sendConfirmationMessage(context.Background(), &now, resp)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var corruptedErr *appErrors.CorruptedDataError
		if !errors.As(err, &corruptedErr) {
			t.Fatalf("expected *appErrors.CorruptedDataError, got %T", err)
		}
	})

	t.Run("generic invoice error", func(t *testing.T) {
		s := &paymentMakingService{invoiceRepo: &dbfakes.FakeInvoiceRepo{GetFn: func(context.Context, string) (document.Invoice, error) {
			return document.Invoice{}, errors.New("db down")
		}}}
		err := s.sendConfirmationMessage(context.Background(), &now, resp)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var unexpectedErr *appErrors.UnexpectedErr
		if !errors.As(err, &unexpectedErr) {
			t.Fatalf("expected *appErrors.UnexpectedErr, got %T", err)
		}
	})

	t.Run("publish error", func(t *testing.T) {
		s := &paymentMakingService{
			invoiceRepo: &dbfakes.FakeInvoiceRepo{GetFn: func(context.Context, string) (document.Invoice, error) {
				return validInvoiceForConfirmation(), nil
			}},
			paymentProducer: &mqfakes.FakeMqProducer{PublishFn: func(context.Context, proto.Message, string) error {
				return errors.New("mq down")
			}},
		}
		err := s.sendConfirmationMessage(context.Background(), &now, resp)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var unexpectedErr *appErrors.UnexpectedErr
		if !errors.As(err, &unexpectedErr) {
			t.Fatalf("expected *appErrors.UnexpectedErr, got %T", err)
		}
	})

	t.Run("success publishes confirmation", func(t *testing.T) {
		producer := &mqfakes.FakeMqProducer{}
		s := &paymentMakingService{
			invoiceRepo: &dbfakes.FakeInvoiceRepo{GetFn: func(context.Context, string) (document.Invoice, error) {
				return validInvoiceForConfirmation(), nil
			}},
			paymentProducer: producer,
		}
		err := s.sendConfirmationMessage(context.Background(), &now, resp)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if producer.PublishCallCount != 1 {
			t.Fatalf("expected publish call count 1, got %d", producer.PublishCallCount)
		}
		if producer.LastPublishedRoutingKey != "booking.booking-123.confirmation" {
			t.Fatalf("unexpected routing key: %q", producer.LastPublishedRoutingKey)
		}
		confirmation, ok := producer.LastPublishedMessage.(*dto.PaymentConfirmation)
		if !ok {
			t.Fatalf("expected *dto.PaymentConfirmation, got %T", producer.LastPublishedMessage)
		}
		if confirmation.GetInvoiceNumber() != "INV-20260309-ABC123" {
			t.Fatalf("unexpected confirmation invoice number: %q", confirmation.GetInvoiceNumber())
		}
	})
}

func TestPaymentMakingService_PayWithCard(t *testing.T) {
	t.Run("invalid request returns invalid argument", func(t *testing.T) {
		s := &paymentMakingService{}
		_, err := s.PayWithCard(context.Background(), &dto.PayWithCardRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
		}
	})

	t.Run("clock error", func(t *testing.T) {
		s := &paymentMakingService{
			clock: &clockfakes.FakeClock{NowFn: func(context.Context) (*time.Time, error) {
				return nil, errors.New("clock down")
			}},
		}

		_, err := s.PayWithCard(context.Background(), validPayWithCardRequest())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if status.Code(err) != codes.Internal {
			t.Fatalf("expected Internal, got %v", status.Code(err))
		}
	})

	t.Run("failure chance branch returns failed status", func(t *testing.T) {
		now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
		s := &paymentMakingService{
			config: config.PaymentMakingCardConfig{FailChance: int(^uint(0) >> 1)},
			clock:  &clockfakes.FakeClock{NowFn: func(context.Context) (*time.Time, error) { return &now, nil }},
			invoiceRepo: &dbfakes.FakeInvoiceRepo{GetFn: func(context.Context, string) (document.Invoice, error) {
				return validInvoiceForConfirmation(), nil
			}},
		}

		resp, err := s.PayWithCard(context.Background(), validPayWithCardRequest())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.GetStatus() != dto.PaymentStatus_PAYMENT_STATUS_FAILED {
			t.Fatalf("expected FAILED status, got %v", resp.GetStatus())
		}
	})

	t.Run("success branch returns succeeded status", func(t *testing.T) {
		now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
		invoiceRepo := &dbfakes.FakeInvoiceRepo{GetFn: func(context.Context, string) (document.Invoice, error) {
			return validInvoiceForConfirmation(), nil
		}}
		s := &paymentMakingService{
			config:          config.PaymentMakingCardConfig{FailChance: 0},
			clock:           &clockfakes.FakeClock{NowFn: func(context.Context) (*time.Time, error) { return &now, nil }},
			receiptRepo:     &dbfakes.FakeReceiptRepo{},
			invoiceRepo:     invoiceRepo,
			paymentProducer: &mqfakes.FakeMqProducer{},
		}

		resp, err := s.PayWithCard(context.Background(), validPayWithCardRequest())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.GetStatus() != dto.PaymentStatus_PAYMENT_STATUS_SUCCEEDED {
			t.Fatalf("expected SUCCEEDED status, got %v", resp.GetStatus())
		}
		if invoiceRepo.UpdateStatusCallCount != 1 {
			t.Fatalf("expected invoice status update call count 1, got %d", invoiceRepo.UpdateStatusCallCount)
		}
		if invoiceRepo.LastUpdatedStatus != "paid" {
			t.Fatalf("expected invoice status to be updated to paid, got %q", invoiceRepo.LastUpdatedStatus)
		}
	})

	t.Run("already paid invoice returns failed precondition", func(t *testing.T) {
		now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
		invoiceRepo := &dbfakes.FakeInvoiceRepo{GetFn: func(context.Context, string) (document.Invoice, error) {
			invoice := validInvoiceForConfirmation()
			invoice.Status = "paid"
			return invoice, nil
		}}
		s := &paymentMakingService{
			config:      config.PaymentMakingCardConfig{FailChance: 0},
			clock:       &clockfakes.FakeClock{NowFn: func(context.Context) (*time.Time, error) { return &now, nil }},
			invoiceRepo: invoiceRepo,
		}

		_, err := s.PayWithCard(context.Background(), validPayWithCardRequest())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("expected FailedPrecondition, got %v", status.Code(err))
		}
		if invoiceRepo.UpdateStatusCallCount != 0 {
			t.Fatalf("expected no invoice status update, got %d", invoiceRepo.UpdateStatusCallCount)
		}
	})
}

func validPayWithCardRequest() *dto.PayWithCardRequest {
	return &dto.PayWithCardRequest{
		InvoiceNumber: "INV-20260309-ABC123",
		Card: &dto.Card{
			Brand:      "VISA",
			Number:     "4111111111111111",
			ExpMonth:   12,
			ExpYear:    2030,
			Cvv:        "123",
			HolderName: "John Doe",
		},
	}
}

func validPayWithCardResponse() *dto.PayWithCardResponse {
	now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
	return &dto.PayWithCardResponse{
		ReceiptNumber: "RCPT-20260309-ABC123",
		InvoiceNumber: "INV-20260309-ABC123",
		Card: &dto.CardSummary{
			Brand: "VISA",
			Last4: "1111",
		},
		ProcessedAt: timestamppb.New(now),
	}
}

func validInvoiceForConfirmation() document.Invoice {
	return document.Invoice{
		BookingId:     "booking-123",
		PayerId:       "payer-123",
		InvoiceNumber: "INV-20260309-ABC123",
		IdempotencyId: "idem-1",
		Status:        "pending",
		Total:         document.Money{Amount: 10000, Currency: "USD"},
		TaxTotal:      document.Money{Amount: 1000, Currency: "USD"},
		DiscountTotal: document.Money{Amount: 0, Currency: "USD"},
		Booking:       document.BookingSnapshot{CottageName: "Cabin", Nights: 2, NumberOfGuests: 3, ValuePerNight: document.Money{Amount: 5000, Currency: "USD"}},
		Payer:         document.Payer{Name: "John", Email: "john@example.com", DocumentNumber: "111", BillingAddress: "Main St"},
		IssuedAt:      time.Date(2026, time.March, 9, 10, 0, 0, 0, time.UTC),
		DueAt:         time.Date(2026, time.March, 10, 10, 0, 0, 0, time.UTC),
		CreatedAt:     time.Date(2026, time.March, 9, 10, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, time.March, 9, 10, 0, 0, 0, time.UTC),
	}
}
