package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/util"
	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/appErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc/payment"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type paymentMakingService struct {
	payment.PaymentMakingServiceServer
	config          config.PaymentMakingCardConfig
	clock           port.Clock
	invoiceRepo     port.InvoiceRepo
	receiptRepo     port.ReceiptRepo
	paymentProducer port.MqProducer
}

func NewPaymentMakingServer(config config.PaymentMakingCardConfig, clock port.Clock,
	invoiceRepo port.InvoiceRepo, receiptRepo port.ReceiptRepo, producer port.MqProducer) payment.PaymentMakingServiceServer {
	return &paymentMakingService{
		config:          config,
		clock:           clock,
		invoiceRepo:     invoiceRepo,
		receiptRepo:     receiptRepo,
		paymentProducer: producer,
	}
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

	if n := rand.Int(); n < s.config.FailChance {
		slog.InfoContext(ctx, "payment failed; generated random number below threshold", "number", n, "chance", s.config.FailChance)
		return s.buildResponse(req, dto.PaymentStatus_PAYMENT_STATUS_FAILED, *now)
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
