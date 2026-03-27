package app

import (
	"context"
	"errors"
	"testing"
	"time"

	dbfakes "github.com/Kenji-Uema/paymentSimulator/internal/infra/db/fakes"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewPaymentReissueService(t *testing.T) {
	t.Run("invalid dependencies", func(t *testing.T) {
		_, err := NewPaymentReissueService(nil, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		service, err := NewPaymentReissueService(&dbfakes.FakeInvoiceRepo{}, "https://pay.local")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if service == nil {
			t.Fatal("expected non-nil service")
		}
	})
}

func TestPaymentReissueService_Reissue(t *testing.T) {
	t.Run("invalid request", func(t *testing.T) {
		service := paymentReissueService{invoiceRepo: &dbfakes.FakeInvoiceRepo{}}

		_, err := service.Reissue(context.Background(), &dto.ReissuePaymentRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
		}
	})

	t.Run("invoice not found", func(t *testing.T) {
		repo := &dbfakes.FakeInvoiceRepo{
			FindByBookingNumberAndDocumentNumberFn: func(context.Context, string, string) (document.Invoice, error) {
				return document.Invoice{}, &dbErrors.InvoiceNotFoundErr{Err: errors.New("not found")}
			},
		}
		service := paymentReissueService{invoiceRepo: repo}

		_, err := service.Reissue(context.Background(), validReissueRequest())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if status.Code(err) != codes.NotFound {
			t.Fatalf("expected NotFound, got %v", status.Code(err))
		}
	})

	t.Run("corrupted invoice", func(t *testing.T) {
		repo := &dbfakes.FakeInvoiceRepo{
			FindByBookingNumberAndDocumentNumberFn: func(context.Context, string, string) (document.Invoice, error) {
				return document.Invoice{}, &dbErrors.CorruptedDataErr{Err: errors.New("bad data")}
			},
		}
		service := paymentReissueService{invoiceRepo: repo}

		_, err := service.Reissue(context.Background(), validReissueRequest())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if status.Code(err) != codes.Internal {
			t.Fatalf("expected Internal, got %v", status.Code(err))
		}
	})

	t.Run("generic repository error", func(t *testing.T) {
		repo := &dbfakes.FakeInvoiceRepo{
			FindByBookingNumberAndDocumentNumberFn: func(context.Context, string, string) (document.Invoice, error) {
				return document.Invoice{}, errors.New("db down")
			},
		}
		service := paymentReissueService{invoiceRepo: repo}

		_, err := service.Reissue(context.Background(), validReissueRequest())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if status.Code(err) != codes.Internal {
			t.Fatalf("expected Internal, got %v", status.Code(err))
		}
	})

	t.Run("success returns payment request", func(t *testing.T) {
		invoice := validInvoiceForReissue()
		repo := &dbfakes.FakeInvoiceRepo{
			FindByBookingNumberAndDocumentNumberFn: func(context.Context, string, string) (document.Invoice, error) {
				return invoice, nil
			},
		}
		service := paymentReissueService{
			invoiceRepo: repo,
			paymentHost: "https://pay.local",
		}

		resp, err := service.Reissue(context.Background(), validReissueRequest())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.GetInvoiceNumber() != invoice.InvoiceNumber {
			t.Fatalf("expected invoice number %q, got %q", invoice.InvoiceNumber, resp.GetInvoiceNumber())
		}
		if len(resp.GetOptions()) != 1 {
			t.Fatalf("expected 1 payment option, got %d", len(resp.GetOptions()))
		}
		expectedURL := "https://pay.local/v1/payments/invoice/" + invoice.InvoiceNumber
		if resp.GetOptions()[0].GetPaymentUrl() != expectedURL {
			t.Fatalf("expected payment url %q, got %q", expectedURL, resp.GetOptions()[0].GetPaymentUrl())
		}
		if repo.FindCallCount != 1 {
			t.Fatalf("expected FindByBookingNumberAndDocumentNumber call count 1, got %d", repo.FindCallCount)
		}
	})

	t.Run("already paid invoice does not reissue payment request", func(t *testing.T) {
		invoice := validInvoiceForReissue()
		invoice.Status = "paid"
		repo := &dbfakes.FakeInvoiceRepo{
			FindByBookingNumberAndDocumentNumberFn: func(context.Context, string, string) (document.Invoice, error) {
				return invoice, nil
			},
		}
		service := paymentReissueService{
			invoiceRepo: repo,
			paymentHost: "https://pay.local",
		}

		_, err := service.Reissue(context.Background(), validReissueRequest())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("expected FailedPrecondition, got %v", status.Code(err))
		}
	})
}

func validReissueRequest() *dto.ReissuePaymentRequest {
	return &dto.ReissuePaymentRequest{
		BookingNumber:  "booking-123",
		DocumentNumber: "11122233344",
	}
}

func validInvoiceForReissue() document.Invoice {
	now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
	return document.Invoice{
		InvoiceNumber: "INV-20260309-ABC123",
		Status:        "pending",
		IssuedAt:      now,
		DueAt:         now.Add(24 * time.Hour),
		Total:         document.Money{Amount: 15000, Currency: "USD"},
		Booking: document.BookingSnapshot{
			CottageName:    "Cabin",
			Nights:         2,
			NumberOfGuests: 3,
		},
		Payer: document.Payer{Name: "John", Email: "john@example.com"},
	}
}
