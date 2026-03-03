package validation

import "errors"

type Validator struct {
	steps []func() error
}

func New() *Validator {
	return &Validator{
		steps: make([]func() error, 0),
	}
}

func (v *Validator) Validate() (err error) {
	for _, step := range v.steps {
		if stepErr := step(); stepErr != nil {
			err = errors.Join(err, stepErr)
		}
	}
	return
}
