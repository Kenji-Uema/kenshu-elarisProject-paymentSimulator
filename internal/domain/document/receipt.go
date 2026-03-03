package document

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Receipt struct {
	Id              bson.ObjectID `bson:"_id,omitempty"`
	PaymentIntentId bson.ObjectID `bson:"payment_intent_id"`
	BookingId       bson.ObjectID `bson:"booking_id"`
	Amount          Money         `bson:"amount"`
	Method          PaymentMethod `bson:"method"`
	ReceiptId       bson.ObjectID `bson:"receipt_id"`
	Card            CardSummary   `bson:"card"`
	ConfirmedAt     *time.Time    `bson:"confirmed_at"`
}

type PaymentMethod string

type CardSummary struct {
	Brand string `bson:"brand"`
	Last4 string `bson:"last4"`
}
