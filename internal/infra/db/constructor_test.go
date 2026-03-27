package db

import (
	"errors"
	"testing"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
)

func TestNewMongoDb_InvalidConfig(t *testing.T) {
	_, err := NewMongoDb(t.Context(), config.MongoConfig{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var validationErr *validationErrors.ErrValidationConstrain
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
	}
}

func TestNewInvoiceRepo_InvalidDB(t *testing.T) {
	_, err := NewInvoiceRepo(nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var validationErr *validationErrors.ErrValidationConstrain
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
	}
}

func TestNewReceiptRepo_InvalidDB(t *testing.T) {
	_, err := NewReceiptRepo(nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var validationErr *validationErrors.ErrValidationConstrain
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
	}
}
