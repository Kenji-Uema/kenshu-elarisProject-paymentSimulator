package validation

import (
	"log/slog"
	"testing"
)

func TestValidator_NotBlank(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{name: "empty", value: "", shouldErr: true},
		{name: "spaces only", value: "   ", shouldErr: true},
		{name: "value", value: "value", shouldErr: false},
		{name: "value with spaces", value: " value ", shouldErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().NotBlank("field", tt.value).Validate()
			slog.Info("returned err", "err", err)

			if tt.shouldErr && err == nil {
				t.Fatalf("expected validation error for %q, got nil", tt.value)
			}
			if !tt.shouldErr && err != nil {
				t.Fatalf("expected no error for %q, got %v", tt.value, err)
			}
		})
	}
}

func TestValidator_NotZeroValue(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		shouldErr bool
	}{
		{name: "int zero", value: int64(0), shouldErr: true},
		{name: "int positive", value: int64(1), shouldErr: false},
		{name: "int negative", value: int64(-1), shouldErr: false},
		{name: "string empty", value: "", shouldErr: true},
		{name: "string value", value: "text", shouldErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().NotZeroValue("field", tt.value).Validate()
			slog.Info("returned err", "err", err)

			if tt.shouldErr && err == nil {
				t.Fatalf("expected validation error for value=%v, got nil", tt.value)
			}
			if !tt.shouldErr && err != nil {
				t.Fatalf("expected no error for value=%v, got %v", tt.value, err)
			}
		})
	}
}

func TestValidator_PositiveValue(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		shouldErr bool
	}{
		{name: "int zero", value: int64(0), shouldErr: true},
		{name: "int negative", value: int64(-5), shouldErr: true},
		{name: "int positive", value: int64(10), shouldErr: false},
		{name: "uint zero", value: uint64(0), shouldErr: true},
		{name: "uint positive", value: uint64(3), shouldErr: false},
		{name: "float negative", value: float64(-1.5), shouldErr: true},
		{name: "float zero", value: float64(0), shouldErr: true},
		{name: "float positive", value: float64(2.5), shouldErr: false},
		{name: "string invalid type", value: "10", shouldErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().PositiveValue("fieldName", tt.value).Validate()
			slog.Info("returned err", "err", err)

			if tt.shouldErr && err == nil {
				t.Fatalf("expected validation error for value=%v, got nil", tt.value)
			}
			if !tt.shouldErr && err != nil {
				t.Fatalf("expected no error for value=%v, got %v", tt.value, err)
			}
		})
	}
}
