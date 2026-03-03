package validation

import (
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
)

func (v *Validator) ValidPeriod(start time.Time, end time.Time) *Validator {
	v.steps = append(v.steps, func() error {
		if start.After(end) || start.Equal(end) {
			return &validationErrors.ErrValidationConstrain{
				Field: "start", Message: "start date must be before end date"}
		}
		return nil
	})
	return v
}
