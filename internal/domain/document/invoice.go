package document

import (
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	InvoiceStatusPending = "pending"
	InvoiceStatusPaid    = "paid"
)

type Invoice struct {
	Id            bson.ObjectID `bson:"_id,omitempty"`
	InvoiceNumber string        `bson:"invoice_number"`
	Status        string        `bson:"status"`

	IssuedAt time.Time `bson:"issued_at"`
	DueAt    time.Time `bson:"due_at"`

	IdempotencyId string `bson:"idempotency_id"`

	BookingId string `bson:"booking_id"`
	PayerId   string `bson:"payer_id"`
	Payer     Payer  `bson:"payer"`

	Booking       BookingSnapshot `bson:"booking"`
	Total         Money           `bson:"total"`
	TaxTotal      Money           `bson:"tax_total"`
	DiscountTotal Money           `bson:"discount_total"`

	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

type Money struct {
	Amount   int64  `bson:"amount"`
	Currency string `bson:"currency"`
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

func NewInvoiceFromProtoMessage(invoiceProto *dto.CreateInvoicePaymentRequest, invoiceNumber string, now time.Time) (Invoice, error) {
	if err := validation.New().NotNil("invoiceProto", invoiceProto).Validate(); err != nil {
		return Invoice{}, err
	}

	payer := invoiceProto.GetPayer()
	booking := invoiceProto.GetBooking()
	total := invoiceProto.GetTotal()
	taxTotal := invoiceProto.GetTaxTotal()
	discountTotal := invoiceProto.GetDiscountTotal()
	issuedAt := invoiceProto.GetIssuedAt()
	dueAt := invoiceProto.GetDueAt()
	valuePerNight := booking.GetValuePerNight()

	validator := validation.New().
		NotBlank("invoiceProto.idempotency_key", invoiceProto.GetIdempotencyKey()).
		NotBlank("invoiceProto.booking_id", invoiceProto.GetBookingId()).
		NotBlank("invoiceProto.payer_id", invoiceProto.GetPayerId()).
		NotNil("invoiceProto.issuedAt", issuedAt).
		NotNil("invoiceProto.dueAt", dueAt).
		NotNil("invoiceProto.payer", payer).
		NotNil("invoiceProto.booking", booking).
		NotNil("invoiceProto.total", total).
		NotNil("invoiceProto.tax_total", taxTotal).
		NotNil("invoiceProto.discount_total", discountTotal).
		NotNil("invoiceProto.booking.value_per_night", valuePerNight).
		NotBlank("invoiceProto.payer.name", payer.GetName()).
		NotBlank("invoiceProto.payer.email", payer.GetEmail()).
		NotBlank("invoiceProto.payer.document_number", payer.GetDocumentNumber()).
		NotBlank("invoiceProto.payer.billing_address", payer.GetBillingAddress()).
		NotBlank("invoiceProto.booking.cottage_name", booking.GetCottageName()).
		PositiveValue("invoiceProto.booking.nights", booking.GetNights()).
		PositiveValue("invoiceProto.booking.number_of_guests", booking.GetNumberOfGuests()).
		PositiveValue("invoiceProto.booking.value_per_night.amount", valuePerNight.GetAmount()).
		NotBlank("invoiceProto.booking.value_per_night.currency", valuePerNight.GetCurrency()).
		PositiveValue("invoiceProto.total.amount", total.GetAmount()).
		NotBlank("invoiceProto.total.currency", total.GetCurrency()).
		NotBlank("invoiceProto.tax_total.currency", taxTotal.GetCurrency()).
		NotBlank("invoiceProto.discount_total.currency", discountTotal.GetCurrency()).
		ValidPrecedence(issuedAt.AsTime(), dueAt.AsTime())

	if err := validator.Validate(); err != nil {
		return Invoice{}, err
	}

	return Invoice{
		InvoiceNumber: invoiceNumber,
		Status:        InvoiceStatusPending,
		IssuedAt:      issuedAt.AsTime(),
		DueAt:         dueAt.AsTime(),
		IdempotencyId: invoiceProto.GetIdempotencyKey(),
		BookingId:     invoiceProto.GetBookingId(),
		PayerId:       invoiceProto.GetPayerId(),
		Payer: Payer{
			Name:           payer.GetName(),
			Email:          payer.GetEmail(),
			DocumentNumber: payer.GetDocumentNumber(),
			BillingAddress: payer.GetBillingAddress(),
		},
		Booking: BookingSnapshot{
			CottageName:    booking.GetCottageName(),
			Nights:         booking.GetNights(),
			NumberOfGuests: booking.GetNumberOfGuests(),
			ValuePerNight: Money{
				Amount:   valuePerNight.GetAmount(),
				Currency: valuePerNight.GetCurrency(),
			},
		},
		Total: Money{
			Amount:   total.GetAmount(),
			Currency: total.GetCurrency(),
		},
		TaxTotal: Money{
			Amount:   taxTotal.GetAmount(),
			Currency: taxTotal.GetCurrency(),
		},
		DiscountTotal: Money{
			Amount:   discountTotal.GetAmount(),
			Currency: discountTotal.GetCurrency(),
		},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
