package validation

import (
	"errors"
	"testing"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
)

func assertValidationConstrainErr(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var validationErr *validationErrors.ErrValidationConstrain
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
	}
}
