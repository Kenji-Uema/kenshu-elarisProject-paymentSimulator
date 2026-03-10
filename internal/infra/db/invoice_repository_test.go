package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestInvoiceRepo_FindByInvoiceNumber(t *testing.T) {
	setupAndRun("TestInvoiceRepo_FindByInvoiceNumber", t, func(t *testing.T, invoiceCollection *mongo.Collection, _ *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		repo := &invoiceRepo{collection: invoiceCollection}

		t.Run("returns invoice when it exists", func(t *testing.T) {
			got, err := repo.FindByInvoiceNumber(ctx, "INV-2026-0001")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if got == (document.Invoice{}) {
				t.Fatal("expected non-empty invoice")
			}
		})

		t.Run("returns validation error when invoice number is blank", func(t *testing.T) {
			_, err := repo.FindByInvoiceNumber(ctx, "   ")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var validationErr *validationErrors.ErrValidationConstrain
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
			}
		})

		t.Run("returns not found when invoice does not exist", func(t *testing.T) {
			_, err := repo.FindByInvoiceNumber(ctx, "INV-DOES-NOT-EXIST")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var notFoundErr *dbErrors.InvoiceNotFoundErr
			if !errors.As(err, &notFoundErr) {
				t.Fatalf("expected *dbErrors.InvoiceNotFoundErr, got %T", err)
			}
		})

		t.Run("returns corrupted data when stored document shape is invalid", func(t *testing.T) {
			_, err := repo.FindByInvoiceNumber(ctx, "INV-2026-CORRUPT1")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var corruptedErr *dbErrors.CorruptedDataErr
			if !errors.As(err, &corruptedErr) {
				t.Fatalf("expected *dbErrors.CorruptedDataErr, got %T", err)
			}
		})
	})
}

func TestInvoiceRepo_FindByBookingNumberAndDocumentNumber(t *testing.T) {
	setupAndRun("TestInvoiceRepo_FindByBookingNumberAndDocumentNumber", t, func(t *testing.T, invoiceCollection *mongo.Collection, _ *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		repo := &invoiceRepo{collection: invoiceCollection}

		t.Run("returns invoice when booking and document numbers match", func(t *testing.T) {
			got, err := repo.FindByBookingNumberAndDocumentNumber(ctx, "booking-1001", "12345678900")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if got == (document.Invoice{}) {
				t.Fatal("expected non-empty invoice")
			}
		})

		t.Run("returns validation error when booking number is blank", func(t *testing.T) {
			_, err := repo.FindByBookingNumberAndDocumentNumber(ctx, "", "12345678900")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var validationErr *validationErrors.ErrValidationConstrain
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
			}
		})

		t.Run("returns not found when no invoice matches", func(t *testing.T) {
			_, err := repo.FindByBookingNumberAndDocumentNumber(ctx, "booking-9999", "00000000000")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var notFoundErr *dbErrors.InvoiceNotFoundErr
			if !errors.As(err, &notFoundErr) {
				t.Fatalf("expected *dbErrors.InvoiceNotFoundErr, got %T", err)
			}
		})

		t.Run("returns corrupted data when stored document shape is invalid", func(t *testing.T) {
			_, err := repo.FindByBookingNumberAndDocumentNumber(ctx, "booking-corrupt-1", "00000000000")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var corruptedErr *dbErrors.CorruptedDataErr
			if !errors.As(err, &corruptedErr) {
				t.Fatalf("expected *dbErrors.CorruptedDataErr, got %T", err)
			}
		})
	})
}

func TestInvoiceRepo_Add(t *testing.T) {
	setupAndRun("TestInvoiceRepo_Add", t, func(t *testing.T, invoiceCollection *mongo.Collection, _ *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		repo := &invoiceRepo{collection: invoiceCollection}

		t.Run("inserts invoice and returns object id", func(t *testing.T) {
			now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
			invoice := document.Invoice{
				InvoiceNumber: "INV-2026-NEW1",
				Status:        "pending",
				IssuedAt:      now,
				DueAt:         now.Add(24 * time.Hour),
				IdempotencyId: "idem-new-1",
				BookingId:     "booking-new-1",
				PayerId:       "payer-new-1",
				Payer: document.Payer{
					Name:           "Test User",
					Email:          "test.user@example.com",
					DocumentNumber: "11122233344",
					BillingAddress: "Main St",
				},
				Booking: document.BookingSnapshot{
					CottageName:    "Cottage Test",
					Nights:         2,
					NumberOfGuests: 2,
					ValuePerNight:  document.Money{Amount: 10000, Currency: "USD"},
				},
				Total:         document.Money{Amount: 20000, Currency: "USD"},
				TaxTotal:      document.Money{Amount: 2000, Currency: "USD"},
				DiscountTotal: document.Money{Amount: 0, Currency: "USD"},
				CreatedAt:     now,
				UpdatedAt:     now,
			}

			gotID, err := repo.Add(ctx, invoice)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if gotID == bson.NilObjectID {
				t.Fatal("expected non-nil inserted id")
			}
		})

		t.Run("returns validation error when invoice payload is invalid", func(t *testing.T) {
			invalid := document.Invoice{
				InvoiceNumber: "",
				BookingId:     "",
				PayerId:       "",
				Total:         document.Money{Amount: 0, Currency: ""},
			}

			_, err := repo.Add(ctx, invalid)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var validationErr *validationErrors.ErrValidationConstrain
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
			}
		})

		t.Run("returns already exists when _id is duplicated", func(t *testing.T) {
			dupID, err := bson.ObjectIDFromHex("65f000000000000000000001")
			if err != nil {
				t.Fatalf("parse object id: %v", err)
			}

			now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
			invoice := document.Invoice{
				Id:            dupID,
				InvoiceNumber: "INV-2026-DUP1",
				Status:        "pending",
				IssuedAt:      now,
				DueAt:         now.Add(24 * time.Hour),
				IdempotencyId: "idem-dup-1",
				BookingId:     "booking-dup-1",
				PayerId:       "payer-dup-1",
				Payer: document.Payer{
					Name:           "Duplicate User",
					Email:          "duplicate.user@example.com",
					DocumentNumber: "99988877766",
					BillingAddress: "Dup St",
				},
				Booking: document.BookingSnapshot{
					CottageName:    "Duplicate Cottage",
					Nights:         1,
					NumberOfGuests: 1,
					ValuePerNight:  document.Money{Amount: 5000, Currency: "USD"},
				},
				Total:         document.Money{Amount: 5000, Currency: "USD"},
				TaxTotal:      document.Money{Amount: 500, Currency: "USD"},
				DiscountTotal: document.Money{Amount: 0, Currency: "USD"},
				CreatedAt:     now,
				UpdatedAt:     now,
			}

			_, err = repo.Add(ctx, invoice)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var alreadyExistsErr *dbErrors.AlreadyExistsErr
			if !errors.As(err, &alreadyExistsErr) {
				t.Fatalf("expected *dbErrors.AlreadyExistsErr, got %T", err)
			}
		})
	})
}
