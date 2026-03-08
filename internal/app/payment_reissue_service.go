package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc/payment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type paymentReissueService struct {
	payment.PaymentRequestServiceServer
	invoiceRepo         port.InvoiceRepo
	paymentMakingConfig config.PaymentMakingCardConfig
}

func NewPaymentReissueService(invoiceRepo port.InvoiceRepo, paymentMakingConfig config.PaymentMakingCardConfig) payment.PaymentRequestServiceServer {
	return &paymentReissueService{
		invoiceRepo:         invoiceRepo,
		paymentMakingConfig: paymentMakingConfig,
	}
}

func (p paymentReissueService) Reissue(ctx context.Context, r *dto.ReissuePaymentRequest) (*dto.PaymentRequest, error) {
	if err := validation.New().
		NotBlank("booking_number", r.GetBookingNumber()).
		NotBlank("document_number", r.GetDocumentNumber()).Validate(); err != nil {

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	invoice, err := p.invoiceRepo.FindByBookingNumberAndDocumentNumber(ctx, r.GetBookingNumber(), r.GetDocumentNumber())
	if err != nil {
		var notFoundErr *dbErrors.InvoiceNotFoundErr
		if errors.As(err, &notFoundErr) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		var corruptedDataErr *dbErrors.CorruptedDataErr
		if errors.As(err, &corruptedDataErr) {
			return nil, status.Error(codes.Internal, "database returned corrupted invoice data")
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	paymentRequest := &dto.PaymentRequest{
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
				PaymentUrl:   fmt.Sprintf("%s/v1/payments/invoice/%s", p.paymentMakingConfig.Host, invoice.InvoiceNumber),
				Instructions: "Please use the following url to pay for your booking",
			},
		},
	}

	return paymentRequest, nil
}
