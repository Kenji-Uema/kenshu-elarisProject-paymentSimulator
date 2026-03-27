package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/util"
	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/appErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc/payment"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type paymentMakingService struct {
	payment.PaymentMakingServiceServer
	failChance      int
	clock           port.Clock
	invoiceRepo     port.InvoiceRepo
	receiptRepo     port.ReceiptRepo
	paymentProducer port.MqProducer
}

func NewPaymentMakingServer(failChance int, clock port.Clock,
	invoiceRepo port.InvoiceRepo, receiptRepo port.ReceiptRepo, producer port.MqProducer) (payment.PaymentMakingServiceServer, error) {
	if err := validation.New().
		NotNil("clock", clock).
		NotNil("invoice_repo", invoiceRepo).
		NotNil("receipt_repo", receiptRepo).
		NotNil("payment_producer", producer).
		Validate(); err != nil {
		return nil, err
	}
	if failChance < 0 {
		return nil, &validationErrors.ErrValidationConstrain{Field: "fail_chance", Message: "must be greater than or equal to 0"}
	}

	return &paymentMakingService{
		failChance:      failChance,
		clock:           clock,
		invoiceRepo:     invoiceRepo,
		receiptRepo:     receiptRepo,
		paymentProducer: producer,
	}, nil
}

func (s *paymentMakingService) PayWithCard(ctx context.Context, req *dto.PayWithCardRequest) (*dto.PayWithCardResponse, error) {
	slog.DebugContext(ctx, "card information")

	if err := validation.New().
		NotBlank("invoice_number", req.GetInvoiceNumber()).
		NotBlank("card.number", req.GetCard().GetNumber()).
		NotZeroValue("card.exp_month", req.GetCard().GetExpMonth()).
		NotZeroValue("card.exp_year", req.GetCard().GetExpYear()).
		NotBlank("card.cvv", req.GetCard().GetCvv()).
		NotBlank("card.holder_name", req.GetCard().GetHolderName()).Validate(); err != nil {

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	now, err := s.clock.Now(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "get current time", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	invoice, err := s.invoiceRepo.FindByInvoiceNumber(ctx, req.GetInvoiceNumber())
	if err != nil {
		return s.mapError(err)
	}
	if strings.EqualFold(invoice.Status, document.InvoiceStatusPaid) {
		return nil, status.Error(codes.FailedPrecondition, "invoice is already paid")
	}

	if n := rand.Int(); n < s.failChance {
		slog.InfoContext(ctx, "payment failed; generated random number below threshold", "number", n, "chance", s.failChance)
		return s.buildResponse(req, dto.PaymentStatus_PAYMENT_STATUS_FAILED, *now)
	}

	if err := s.invoiceRepo.UpdateStatus(ctx, req.GetInvoiceNumber(), document.InvoiceStatusPaid, *now); err != nil {
		return s.mapError(err)
	}

	resp, err := s.buildResponse(req, dto.PaymentStatus_PAYMENT_STATUS_SUCCEEDED, *now)
	if err != nil {
		slog.ErrorContext(ctx, "build response", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	go func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if _, err := s.saveReceipt(asyncCtx, resp); err != nil {
			slog.ErrorContext(ctx, "save receipt", "error", err)
			return
		}
		if err := s.sendConfirmationMessage(asyncCtx, now, resp); err != nil {
			slog.ErrorContext(ctx, "send confirmation message", "error", err)
			return
		}
	}()

	return resp, nil
}

func (s *paymentMakingService) mapError(err error) (*dto.PayWithCardResponse, error) {
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

func (s *paymentMakingService) buildResponse(req *dto.PayWithCardRequest, status dto.PaymentStatus, now time.Time) (*dto.PayWithCardResponse, error) {
	cardNumber := req.GetCard().GetNumber()
	cardNumberLast4 := cardNumber[len(cardNumber)-4:]

	cardSummary := &dto.CardSummary{
		Brand: req.GetCard().GetBrand(),
		Last4: cardNumberLast4,
	}

	receiptNumber, err := util.GenerateHumanFriendlyId("RCPT", now)
	if err != nil {
		return nil, &appErrors.UnexpectedErr{Msg: "failed to generate receipt number", Err: err}
	}

	return &dto.PayWithCardResponse{
		ReceiptNumber: receiptNumber,
		InvoiceNumber: req.GetInvoiceNumber(),
		Status:        status,
		ProcessedAt:   timestamppb.Now(),
		Card:          cardSummary,
	}, nil

}

func (s *paymentMakingService) saveReceipt(ctx context.Context, resp *dto.PayWithCardResponse) (document.Receipt, error) {
	receipt, err := document.NewReceiptFromProtMessage(resp)
	if err != nil {
		return document.Receipt{}, err
	}

	receiptId, err := s.receiptRepo.Add(ctx, receipt)
	if err != nil {
		var alreadyExistsErr *dbErrors.AlreadyExistsErr
		if errors.As(err, &alreadyExistsErr) {
			return document.Receipt{}, &appErrors.AlreadyExistsErr{Err: err}
		}

		var corruptedDataErr *dbErrors.CorruptedDataErr
		if errors.As(err, &corruptedDataErr) {
			return document.Receipt{}, &appErrors.CorruptedDataError{Msg: "database returned corrupted receipt data", Err: err}
		}

		return document.Receipt{}, &appErrors.UnexpectedErr{Msg: "unexpect error happened while saving receipt", Err: err}
	}

	slog.InfoContext(ctx, "receipt saved", "receipt_id", receiptId)

	return receipt, nil
}

func (s *paymentMakingService) sendConfirmationMessage(ctx context.Context, now *time.Time, resp *dto.PayWithCardResponse) error {
	invoice, err := s.invoiceRepo.FindByInvoiceNumber(ctx, resp.GetInvoiceNumber())
	if err != nil {
		var notFoundErr *dbErrors.InvoiceNotFoundErr
		if errors.As(err, &notFoundErr) {
			return &appErrors.InvoiceNotFoundErr{Err: err}
		}

		var corruptedDataErr *dbErrors.CorruptedDataErr
		if errors.As(err, &corruptedDataErr) {
			return &appErrors.CorruptedDataError{Err: err}
		}

		return &appErrors.UnexpectedErr{Msg: "unexpected error happened while getting invoice", Err: err}
	}

	confirmation := &dto.PaymentConfirmation{
		Id:            uuid.New().String(),
		BookingId:     invoice.BookingId,
		PayerId:       invoice.PayerId,
		InvoiceNumber: invoice.InvoiceNumber,
		ReceiptNumber: resp.GetReceiptNumber(),
		ConfirmedAt:   timestamppb.New(*now),
	}

	routingKey := fmt.Sprintf("booking.%s.confirmation", invoice.BookingId)

	if err := s.paymentProducer.Publish(ctx, confirmation, routingKey); err != nil {
		return &appErrors.UnexpectedErr{
			Msg: "unexpected error happened while publishing payment confirmation message",
			Err: err,
		}
	}
	return nil
}
