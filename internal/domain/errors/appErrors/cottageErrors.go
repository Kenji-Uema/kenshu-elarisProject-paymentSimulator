package appErrors

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type CorruptedDataError struct {
	Err error
}

func (e CorruptedDataError) Error() string {
	return fmt.Sprintf("database contains inconsistent cottage data: %v", e.Err)
}

type CottageNotFound struct {
	Err error
}

func (e CottageNotFound) Error() string {
	return fmt.Sprintf("Cottage not found: %v", e.Err)
}

type AddBookingToCottageError struct {
	Err error
}

func (e AddBookingToCottageError) Error() string {
	return fmt.Sprintf("Could not add booking: %v", e.Err)
}

type RemoveBookingFromCottageError struct {
	Err error
}

func (e RemoveBookingFromCottageError) Error() string {
	return fmt.Sprintf("Could not remove booking: %v", e.Err)
}

type BookingNotFound struct {
	BookingId bson.ObjectID
}

func (e BookingNotFound) Error() string {
	return fmt.Sprintf("Booking not found: %s", e.BookingId.Hex())
}

type UnexpectedError struct {
	Err error
}

func (e UnexpectedError) Error() string {
	return fmt.Sprintf("Unexpected error: %v", e.Err)
}
