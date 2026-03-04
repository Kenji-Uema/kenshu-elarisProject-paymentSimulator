package validation

import (
	"reflect"
	"strings"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
)

func (v *Validator) NotBlank(field, value string) *Validator {
	v.steps = append(v.steps, func() error {
		if strings.TrimSpace(value) == "" {
			return &validationErrors.ErrValidationConstrain{Field: field, Message: "must not be blank"}
		}
		return nil
	})
	return v
}

func (v *Validator) NotZeroValue(field string, value any) *Validator {
	v.steps = append(v.steps, func() error {
		if reflect.ValueOf(value).IsZero() {
			return &validationErrors.ErrValidationConstrain{Field: field, Message: "must not be zero value"}
		}
		return nil
	})
	return v
}

func (v *Validator) PositiveValue(field string, value any) *Validator {
	v.steps = append(v.steps, func() error {
		rv := reflect.ValueOf(value)
		if !rv.IsValid() {
			return &validationErrors.ErrValidationConstrain{Field: field, Message: "must be a number greater than 0"}
		}

		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if rv.Int() > 0 {
				return nil
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			if rv.Uint() > 0 {
				return nil
			}
		case reflect.Float32, reflect.Float64:
			if rv.Float() > 0 {
				return nil
			}
		default:
			return &validationErrors.ErrValidationConstrain{Field: field, Message: "must be a number greater than 0"}
		}

		return &validationErrors.ErrValidationConstrain{Field: field, Message: "must be greater than 0"}
	})

	return v
}

func (v *Validator) NotNil(field string, value any) *Validator {
	v.steps = append(v.steps, func() error {
		if value == nil {
			return &validationErrors.ErrValidationConstrain{Field: field, Message: "must not be nil"}
		}
		return nil
	})

	return v
}
