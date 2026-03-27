package http

import (
	"errors"
	"testing"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
)

func TestNewHttpServer_InvalidConfig(t *testing.T) {
	_, err := NewHttpServer("", 0, 0, 0, 0, 0, config.TelemetryConfig{}, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var validationErr *validationErrors.ErrValidationConstrain
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
	}
}
