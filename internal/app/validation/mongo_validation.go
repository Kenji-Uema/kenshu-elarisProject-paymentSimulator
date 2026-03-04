package validation

import (
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func (v *Validator) NotNilObjectID(field string, id bson.ObjectID) *Validator {
	v.steps = append(v.steps, func() error {
		if id == bson.NilObjectID {
			return &validationErrors.ErrValidationConstrain{Field: field, Message: "must not be nil"}
		}
		return nil
	})
	return v
}
