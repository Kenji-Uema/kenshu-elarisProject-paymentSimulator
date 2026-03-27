package clock

import (
	"errors"
	"testing"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
)

func TestNewClock(t *testing.T) {
	t.Run("invalid config", func(t *testing.T) {
		_, err := NewClock(config.Services{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
		}
	})
}
