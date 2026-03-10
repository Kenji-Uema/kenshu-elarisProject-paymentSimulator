package validation

import (
	"errors"
	"testing"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestValidator_NotNilObjectID(t *testing.T) {
	t.Run("nil objectId, should return error", func(t *testing.T) {
		err := New().NotNilObjectID("field", bson.NilObjectID).Validate()

		if err == nil {
			t.Fatalf("expected validation error for nil objectId, got nil")
		}

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("not nil objectId, should return nil", func(t *testing.T) {
		err := New().NotNilObjectID("field", bson.NewObjectID()).Validate()

		if err != nil {
			t.Fatalf("expected no error for not nil objectId, got %v", err)
		}
	})
}
