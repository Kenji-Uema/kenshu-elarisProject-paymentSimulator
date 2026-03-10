package helpers

import (
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ValidCreateInvoiceRequest() *dto.CreateInvoicePaymentRequest {
	return &dto.CreateInvoicePaymentRequest{
		IdempotencyKey: "idem-main-it-1",
		BookingId:      "booking-123",
		PayerId:        "payer-123",
		IssuedAt:       timestamppb.New(time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)),
		DueAt:          timestamppb.New(time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)),
		Payer: &dto.Payer{
			Name:           "John Doe",
			Email:          "john@example.com",
			DocumentNumber: "11122233344",
			BillingAddress: "Main St",
		},
		Booking: &dto.BookingSnapshot{
			CottageName:    "Cabin",
			Nights:         2,
			NumberOfGuests: 3,
			ValuePerNight: &dto.Money{
				Amount:   10000,
				Currency: "USD",
			},
		},
		Total:         &dto.Money{Amount: 20000, Currency: "USD"},
		TaxTotal:      &dto.Money{Amount: 1000, Currency: "USD"},
		DiscountTotal: &dto.Money{Amount: 0, Currency: "USD"},
	}
}
