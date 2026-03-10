package validation

import (
	"testing"
)

func TestValidator_NotBlank_PositiveCases(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "valid string", value: "value"},
		{name: "valid string with trailing spaces", value: " value "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().NotBlank("field", tt.value).Validate()
			if err != nil {
				t.Fatalf("expected no error for %q, got %v", tt.value, err)
			}
		})
	}
}

func TestValidator_NotBlank_NegativeCases(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "empty", value: ""},
		{name: "spaces only", value: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().NotBlank("field", tt.value).Validate()
			assertValidationConstrainErr(t, err)
		})
	}
}

func TestValidator_NotZeroValue_PositiveCases(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{name: "int positive", value: int64(1)},
		{name: "int negative", value: int64(-1)},
		{name: "string value", value: "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().NotZeroValue("field", tt.value).Validate()
			if err != nil {
				t.Fatalf("expected no error for value=%v, got %v", tt.value, err)
			}
		})
	}
}

func TestValidator_NotZeroValue_NegativeCases(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{name: "int zero", value: int64(0)},
		{name: "string empty", value: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().NotZeroValue("field", tt.value).Validate()
			assertValidationConstrainErr(t, err)
		})
	}
}

func TestValidator_PositiveValue_PositiveCases(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{name: "int positive", value: int64(10)},
		{name: "uint positive", value: uint64(3)},
		{name: "float positive", value: 2.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().PositiveValue("fieldName", tt.value).Validate()
			if err != nil {
				t.Fatalf("expected no error for value=%v, got %v", tt.value, err)
			}
		})
	}
}

func TestValidator_PositiveValue_NegativeCases(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{name: "int zero", value: int64(0)},
		{name: "int negative", value: int64(-5)},
		{name: "uint zero", value: uint64(0)},
		{name: "float negative", value: -1.5},
		{name: "float zero", value: float64(0)},
		{name: "string invalid type", value: "10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().PositiveValue("fieldName", tt.value).Validate()
			assertValidationConstrainErr(t, err)
		})
	}
}

func TestValidator_NotNil_PositiveCases(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{name: "string value", value: "value"},
		{name: "int value", value: 1},
		{name: "struct value", value: struct{ Name string }{Name: "test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().NotNil("field", tt.value).Validate()
			if err != nil {
				t.Fatalf("expected no error for value=%v, got %v", tt.value, err)
			}
		})
	}
}

func TestValidator_NotNil_NegativeCases(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{name: "nil value", value: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().NotNil("field", tt.value).Validate()
			assertValidationConstrainErr(t, err)
		})
	}
}
