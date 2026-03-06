package app

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/util"
	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc/payment"
	"github.com/google/uuid"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type paymentMakingCardService struct {
	payment.PaymentMakingCardServiceServer
	config          config.PaymentMakingCardConfig
	clock           *grpc.Clock
	invoiceRepo     port.InvoiceRepo
	receiptRepo     port.ReceiptRepo
	paymentProducer port.MqProducer
}

func NewPaymentMakingCardServer(config config.PaymentMakingCardConfig, clock *grpc.Clock,
	invoiceRepo port.InvoiceRepo, receiptRepo port.ReceiptRepo, producer port.MqProducer) payment.PaymentMakingCardServiceServer {
	return &paymentMakingCardService{
		config:          config,
		clock:           clock,
		invoiceRepo:     invoiceRepo,
		receiptRepo:     receiptRepo,
		paymentProducer: producer,
	}
}

func (s *paymentMakingCardService) PayWithCard(ctx context.Context, req *dto.PayWithCardRequest) (*dto.PayWithCardResponse, error) {
	slog.DebugContext(ctx, "card information")

	if err := validation.New().
		NotBlank("invoice_number", req.InvoiceNumber).
		NotBlank("card.number", req.Card.Number).
		NotZeroValue("card.exp_month", req.Card.ExpMonth).
		NotZeroValue("card.exp_year", req.Card.ExpYear).
		NotBlank("card.cvv", req.Card.Cvv).
		NotBlank("card.holder_name", req.Card.HolderName).Validate(); err != nil {

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	now, err := s.clock.Now(ctx)
	if err != nil {
		return nil, err
	}

	if rand.Int() < s.config.FailChance {
		return s.buildResponse(req, dto.PaymentStatus_PAYMENT_STATUS_FAILED, *now)
	}
	resp, err := s.buildResponse(req, dto.PaymentStatus_PAYMENT_STATUS_SUCCEEDED, *now)
	if err != nil {
		return nil, err
	}

	go func() {
		if _, err := s.saveReceipt(ctx, resp); err != nil {
			slog.ErrorContext(ctx, "save receipt", "error", err)
			return
		}
		if err := s.sendConfirmationMessage(ctx, now, resp); err != nil {
			slog.ErrorContext(ctx, "send confirmation message", "error", err)
			return
		}
	}()

	return resp, nil
}

func (s *paymentMakingCardService) buildResponse(req *dto.PayWithCardRequest, status dto.PaymentStatus, now time.Time) (*dto.PayWithCardResponse, error) {
	cardNumber := req.GetCard().GetNumber()
	cardNumberLast4 := cardNumber[len(cardNumber)-4:]

	cardSummary := &dto.CardSummary{
		Brand: req.GetCard().GetBrand(),
		Last4: cardNumberLast4,
	}

	receiptNumber, err := util.GenerateHumanFriendlyId("RCPT", now)
	if err != nil {
		return nil, err
	}

	return &dto.PayWithCardResponse{
		ReceiptNumber: receiptNumber,
		Status:        status,
		ProcessedAt:   timestamppb.Now(),
		Card:          cardSummary,
	}, nil

}

func (s *paymentMakingCardService) saveReceipt(ctx context.Context, resp *dto.PayWithCardResponse) (document.Receipt, error) {
	receipt, err := document.NewReceipt(resp)
	if err != nil {
		return document.Receipt{}, status.Error(codes.Internal, fmt.Sprintf("create receipt: %v", err))
	}

	if _, err := s.receiptRepo.Add(ctx, receipt); err != nil {
		return document.Receipt{}, status.Error(codes.Internal, fmt.Sprintf("save receipt: %v", err))
	}
	return receipt, nil
}

func (s *paymentMakingCardService) sendConfirmationMessage(ctx context.Context, now *time.Time, resp *dto.PayWithCardResponse) error {
	invoice, err := s.invoiceRepo.Get(ctx, resp.GetInvoiceNumber())
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("get invoice: %v", err))
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
		return status.Error(codes.Internal, fmt.Sprintf("publish payment confirmation: %v", err))
	}
	return nil
}
