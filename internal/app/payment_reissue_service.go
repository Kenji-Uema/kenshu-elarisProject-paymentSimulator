package app

import (
	"context"
	"errors"
	"strings"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc/payment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type paymentReissueService struct {
	payment.PaymentReissueServiceServer
	invoiceRepo port.InvoiceRepo
	paymentHost string
}

func NewPaymentReissueService(invoiceRepo port.InvoiceRepo, paymentHost string) (payment.PaymentReissueServiceServer, error) {
	if err := validation.New().
		NotNil("invoice_repo", invoiceRepo).
		NotBlank("payment_host", paymentHost).
		Validate(); err != nil {
		return nil, err
	}

	return &paymentReissueService{
		invoiceRepo: invoiceRepo,
		paymentHost: paymentHost,
	}, nil
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
	if strings.EqualFold(invoice.Status, document.InvoiceStatusPaid) {
		return nil, status.Error(codes.FailedPrecondition, "invoice is already paid")
	}

	paymentRequest := buildPaymentRequest(invoice, p.paymentHost)

	return paymentRequest, nil
}
