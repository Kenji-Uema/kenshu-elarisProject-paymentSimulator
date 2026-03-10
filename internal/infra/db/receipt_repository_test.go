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

func TestReceiptRepo_GetById(t *testing.T) {
	setupAndRun("TestReceiptRepo_GetById", t, func(t *testing.T, _ *mongo.Collection, receiptCollection *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		repo := &receiptRepo{collection: receiptCollection}

		t.Run("returns receipt when it exists", func(t *testing.T) {
			id, err := bson.ObjectIDFromHex("65f100000000000000000001")
			if err != nil {
				t.Fatalf("parse object id: %v", err)
			}

			got, err := repo.GetById(ctx, id)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if got == (document.Receipt{}) {
				t.Fatal("expected non-empty receipt")
			}
		})

		t.Run("returns validation error when id is nil", func(t *testing.T) {
			_, err := repo.GetById(ctx, bson.NilObjectID)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var validationErr *validationErrors.ErrValidationConstrain
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
			}
		})

		t.Run("returns not found when receipt does not exist", func(t *testing.T) {
			_, err := repo.GetById(ctx, bson.NewObjectID())
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var notFoundErr *dbErrors.ReceiptNotFoundErr
			if !errors.As(err, &notFoundErr) {
				t.Fatalf("expected *dbErrors.ReceiptNotFoundErr, got %T", err)
			}
		})

		t.Run("returns corrupted data when stored document shape is invalid", func(t *testing.T) {
			id, err := bson.ObjectIDFromHex("65f300000000000000000001")
			if err != nil {
				t.Fatalf("parse object id: %v", err)
			}

			_, err = repo.GetById(ctx, id)
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

func TestReceiptRepo_Add(t *testing.T) {
	setupAndRun("TestReceiptRepo_Add", t, func(t *testing.T, _ *mongo.Collection, receiptCollection *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		repo := &receiptRepo{collection: receiptCollection}

		t.Run("inserts receipt and returns object id", func(t *testing.T) {
			now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
			receipt := document.Receipt{
				ReceiptNumber: "RCPT-2026-NEW1",
				InvoiceNumber: "INV-2026-NEW1",
				Card: document.CardSummary{
					Brand: "VISA",
					Last4: "1111",
				},
				ProcessedAt: now,
			}

			gotID, err := repo.Add(ctx, receipt)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if gotID == bson.NilObjectID {
				t.Fatal("expected non-nil inserted id")
			}
		})

		t.Run("returns validation error when receipt payload is invalid", func(t *testing.T) {
			invalid := document.Receipt{
				ReceiptNumber: "",
				InvoiceNumber: "",
				Card:          document.CardSummary{},
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
			dupID, err := bson.ObjectIDFromHex("65f100000000000000000001")
			if err != nil {
				t.Fatalf("parse object id: %v", err)
			}

			now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
			receipt := document.Receipt{
				Id:            dupID,
				ReceiptNumber: "RCPT-2026-DUP1",
				InvoiceNumber: "INV-2026-DUP1",
				Card: document.CardSummary{
					Brand: "MASTERCARD",
					Last4: "2222",
				},
				ProcessedAt: now,
			}

			_, err = repo.Add(ctx, receipt)
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
