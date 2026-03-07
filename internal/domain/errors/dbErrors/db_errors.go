package dbErrors

import (
	"fmt"
)

type UnexpectedErr struct {
	Msg string
	Err error
}

func (e *UnexpectedErr) Error() string {
	return fmt.Sprintf("unexpected error, %s: %v", e.Msg, e.Err)
}

type CorruptedDataErr struct {
	Err error
}

func (e CorruptedDataErr) Error() string {
	return fmt.Sprintf("database contains inconsistent cottage data: %v", e.Err)
}

type AlreadyExistsErr struct {
	Err error
}

func (e AlreadyExistsErr) Error() string {
	return fmt.Sprintf("element already exists: %v", e.Err)
}

type InvoiceNotFoundErr struct {
	Err error
}

func (e InvoiceNotFoundErr) Error() string {
	return fmt.Sprintf("receipt not found: %v", e.Err)
}

type ReceiptNotFoundErr struct {
	Err error
}

func (e ReceiptNotFoundErr) Error() string {
	return fmt.Sprintf("receipt not found: %v", e.Err)
}
