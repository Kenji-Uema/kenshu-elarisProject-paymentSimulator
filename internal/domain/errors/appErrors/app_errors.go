package appErrors

import (
	"fmt"
)

type CorruptedDataError struct {
	Msg string
	Err error
}

func (e CorruptedDataError) Error() string {
	return fmt.Sprintf("corrupted data error, %s: %v", e.Msg, e.Err)
}

func (e CorruptedDataError) Unwrap() error {
	return e.Err
}

type AlreadyExistsErr struct {
	Err error
}

func (e AlreadyExistsErr) Error() string {
	return fmt.Sprintf("element already exists: %v", e.Err)
}

func (e AlreadyExistsErr) Unwrap() error {
	return e.Err
}

type InvoiceNotFoundErr struct {
	Err error
}

func (e InvoiceNotFoundErr) Error() string {
	return fmt.Sprintf("invoice not found: %v", e.Err)
}

func (e InvoiceNotFoundErr) Unwrap() error {
	return e.Err
}

type ReceiptNotFoundErr struct {
	Err error
}

func (e ReceiptNotFoundErr) Error() string {
	return fmt.Sprintf("receipt not found: %v", e.Err)
}

func (e ReceiptNotFoundErr) Unwrap() error {
	return e.Err
}

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
