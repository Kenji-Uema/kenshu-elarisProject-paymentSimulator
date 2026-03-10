package validation

import (
	"testing"
	"time"
)

func TestValidator_ValidPrecedence(t *testing.T) {
	now := time.Now().UTC()

	t.Run("valid precedence", func(t *testing.T) {
		start := now
		end := now.Add(24 * time.Hour)

		err := New().ValidPrecedence(start, end).Validate()

		if err != nil {
			t.Fatalf("expected no error for start=%v end=%v, got %v", start, end, err)
		}
	})

	t.Run("start and end equals", func(t *testing.T) {
		start := now
		end := now

		err := New().ValidPrecedence(start, end).Validate()
		assertValidationConstrainErr(t, err)
	})

	t.Run("end before start", func(t *testing.T) {
		start := now
		end := now.Add(-24 * time.Hour)

		err := New().ValidPrecedence(start, end).Validate()
		assertValidationConstrainErr(t, err)
	})
}
