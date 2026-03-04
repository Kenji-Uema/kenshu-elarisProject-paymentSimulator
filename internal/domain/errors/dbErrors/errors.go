package dbErrors

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
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

type MissingBookingsErr struct {
	Missing []bson.ObjectID
}

func (e *MissingBookingsErr) Error() string {
	return fmt.Sprintf("not all booking ids found, missing: %v", e.Missing)
}

type BookingsNotUpdatedErr struct {
	CottageName string
	BookingId   bson.ObjectID
}

func (e *BookingsNotUpdatedErr) Error() string {
	return fmt.Sprintf("could not update bookings for cottage %s, booking %s", e.CottageName, e.BookingId.Hex())
}

type CottageNotFoundErr struct {
	CottageName string
}

func (e *CottageNotFoundErr) Error() string {
	return fmt.Sprintf("cottage %s not found", e.CottageName)
}

type BookingNotFoundErr struct {
	BookingId bson.ObjectID
}

func (e *BookingNotFoundErr) Error() string {
	return fmt.Sprintf("booking %s not found", e.BookingId.Hex())
}
