package document

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Invoice struct {
	Id            bson.ObjectID   `bson:"_id,omitempty"`
	BookingId     bson.ObjectID   `bson:"booking_id"`
	PayerId       bson.ObjectID   `bson:"payer_id"`
	Payer         Payer           `bson:"payer"`
	Booking       BookingSnapshot `bson:"booking"`
	Total         Money           `bson:"total"`
	TaxTotal      Money           `bson:"tax_total"`
	DiscountTotal Money           `bson:"discount_total"`
}

type Money struct {
	Amount   int64  `bson:"amount"`
	Currency string `bson:"currency""`
}

type Payer struct {
	Name           string `bson:"name"`
	Email          string `bson:"email"`
	DocumentNumber string `bson:"document_number"`
	BillingAddress string `bson:"billing_address"`
}

type BookingSnapshot struct {
	CottageName    string `bson:"cottage_name"`
	Nights         int32  `bson:"nights"`
	NumberOfGuests int32  `bson:"number_of_guests"`
	ValuePerNight  Money  `bson:"value_per_night"`
}
