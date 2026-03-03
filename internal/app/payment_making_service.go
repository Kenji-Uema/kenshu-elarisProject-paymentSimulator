package app

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	paymentgrpc "github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc/payment"
	"go.mongodb.org/mongo-driver/v2/bson"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type paymentMakingCardService struct {
	paymentgrpc.PaymentMakingCardServiceServer
	config          config.PaymentMakingCardConfig
	invoiceRepo     port.InvoiceRepo
	receiptRepo     port.ReceiptRepo
	paymentProducer port.MqProducer
}

func NewPaymentMakingCardServer(config config.PaymentMakingCardConfig,
	invoiceRepo port.InvoiceRepo, receiptRepo port.ReceiptRepo, paymentProducer port.MqProducer) paymentgrpc.PaymentMakingCardServiceServer {
	return &paymentMakingCardService{
		config:          config,
		invoiceRepo:     invoiceRepo,
		receiptRepo:     receiptRepo,
		paymentProducer: paymentProducer,
	}
}

func (s *paymentMakingCardService) PayWithCard(ctx context.Context,
	req *dto.PayWithCardRequest) (*dto.PayWithCardResponse, error) {

	slog.DebugContext(ctx, "card information")

	if err := validation.New().
		NotBlank("card.number", req.Card.Number).
		NotZeroValue("card.exp_month", req.Card.ExpMonth).
		NotZeroValue("card.exp_year", req.Card.ExpYear).
		NotBlank("card.cvv", req.Card.Cvv).
		NotBlank("card.holder_name", req.Card.HolderName).
		NotBlank("invoice_id", req.InvoiceId).
		Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	invoiceID, err := bson.ObjectIDFromHex(req.GetInvoiceId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid invoice_id: %v", err))
	}

	invoice, err := s.invoiceRepo.Get(ctx, invoiceID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("load invoice: %v", err))
	}

	n := req.GetCard().GetNumber()
	last4 := n[len(n)-4:]
	paymentIntentID := bson.NewObjectID()
	receiptBusinessID := bson.NewObjectID()

	resp := dto.PayWithCardResponse{
		PaymentIntentId: paymentIntentID.Hex(),
		Status:          dto.PaymentStatus_PAYMENT_STATUS_SUCCEEDED,
		ReceiptId:       receiptBusinessID.Hex(),
		ProcessedAt:     timestamppb.Now(),
		Card: &dto.CardSummary{
			Brand: req.Card.Brand,
			Last4: last4,
		},
	}

	if rand.Int() < s.config.FailChance {
		resp.Status = dto.PaymentStatus_PAYMENT_STATUS_FAILED

		return &resp, nil
	}

	confirmedAt := time.Now()
	receipt := document.Receipt{
		PaymentIntentId: paymentIntentID,
		BookingId:       invoice.BookingId,
		Amount: document.Money{
			Amount:   invoice.Total.Amount,
			Currency: invoice.Total.Currency,
		},
		Method:    document.PaymentMethod(dto.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD.String()),
		ReceiptId: receiptBusinessID,
		Card: document.CardSummary{
			Brand: req.GetCard().GetBrand(),
			Last4: last4,
		},
		ConfirmedAt: &confirmedAt,
	}

	if _, err := s.receiptRepo.Add(ctx, receipt); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("save receipt: %v", err))
	}

	confirmation := &dto.PaymentConfirmation{
		Id:              bson.NewObjectID().Hex(),
		PaymentIntentId: paymentIntentID.Hex(),
		BookingId:       invoice.BookingId.Hex(),
		Amount: &dto.Money{
			Amount:   invoice.Total.Amount,
			Currency: invoice.Total.Currency,
		},
		Method:    dto.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD,
		ReceiptId: receiptBusinessID.Hex(),
		Card: &dto.CardSummary{
			Brand: req.GetCard().GetBrand(),
			Last4: last4,
		},
		ConfirmedAt: timestamppb.New(confirmedAt),
	}

	if err := s.paymentProducer.Publish(ctx, confirmation, config.PublishConfig{
		RoutingKey: "payment.confirmation",
		Mandatory:  false,
		Immediate:  false,
	}); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("publish payment confirmation: %v", err))
	}

	return &resp, nil
}
