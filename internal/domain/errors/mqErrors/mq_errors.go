package mqErrors

import "fmt"

type UnexpectedErr struct {
	Msg string
	Err error
}

func (e *UnexpectedErr) Error() string {
	return fmt.Sprintf("unexpected error, %s: %v", e.Msg, e.Err)
}

func (e *UnexpectedErr) Unwrap() error {
	return e.Err
}
