package app

import (
	"fmt"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func buildPaymentRequest(invoice document.Invoice, host string) *dto.PaymentRequest {
	return &dto.PaymentRequest{
		InvoiceNumber: invoice.InvoiceNumber,
		Total: &dto.Money{
			Amount:   invoice.Total.Amount,
			Currency: invoice.Total.Currency,
		},
		IssuedAt:  timestamppb.New(invoice.IssuedAt),
		ExpiresAt: timestamppb.New(invoice.DueAt),
		Booking: &dto.BookingSummary{
			CottageName:    invoice.Booking.CottageName,
			Nights:         invoice.Booking.Nights,
			NumberOfGuests: invoice.Booking.NumberOfGuests,
		},
		Payer: &dto.PayerSummary{
			Name:  invoice.Payer.Name,
			Email: invoice.Payer.Email,
		},
		Options: []*dto.PaymentOption{
			{
				Method:       dto.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD,
				PaymentUrl:   fmt.Sprintf("%s/v1/payments/invoice/%s", host, invoice.InvoiceNumber),
				Instructions: "Please use the following url to pay for your booking",
			},
		},
	}

}
