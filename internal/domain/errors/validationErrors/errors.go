package validationErrors

import "fmt"

type ErrValidationConstrain struct {
	Field   string
	Message string
}

func (e *ErrValidationConstrain) Error() string {
	return fmt.Sprintf("%s has violations; %s", e.Field, e.Message)
}
