package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/util"
	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type InvoiceService interface {
	StartInvoiceProcessing(ctx context.Context)
}

type invoiceService struct {
	clock               *grpc.Clock
	invoiceRepo         port.InvoiceRepo
	invoiceConsumer     port.MqConsumer
	paymentProducer     port.MqProducer
	paymentMakingConfig config.PaymentMakingCardConfig
}

func NewInvoiceService(invoiceRepo port.InvoiceRepo, clock *grpc.Clock,
	invoiceConsumer port.MqConsumer, paymentProducer port.MqProducer,
	paymentMakingConfig config.PaymentMakingCardConfig) InvoiceService {

	return &invoiceService{
		clock:               clock,
		invoiceRepo:         invoiceRepo,
		invoiceConsumer:     invoiceConsumer,
		paymentProducer:     paymentProducer,
		paymentMakingConfig: paymentMakingConfig,
	}
}

func (s *invoiceService) StartInvoiceProcessing(ctx context.Context) {
	deliveries, err := s.invoiceConsumer.Consume(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "consume invoice queue", "error", err)
		return
	}

	for delivery := range deliveries {
		if err := s.processInvoiceDelivery(ctx, delivery); err != nil {
			slog.ErrorContext(ctx, "process invoice delivery", "delivery_tag", delivery.DeliveryTag, "error", err)
		}
	}
}

func (s *invoiceService) processInvoiceDelivery(ctx context.Context, delivery amqp.Delivery) error {
	var createInvoiceRequest dto.CreateInvoicePaymentRequest
	if err := protojson.Unmarshal(delivery.Body, &createInvoiceRequest); err != nil {
		_ = delivery.Nack(false, false)
		return err
	}

	now, err := s.clock.Now(ctx)
	if err != nil {
		return err
	}

	invoiceNumber, err := util.GenerateHumanFriendlyId("INV", *now)
	if err != nil {
		return err
	}
	invoice, err := document.NewInvoiceFromProtoMessage(&createInvoiceRequest, invoiceNumber, *now)
	if err != nil {
		_ = delivery.Nack(false, true)
		return err
	}

	invoiceID, err := s.invoiceRepo.Add(ctx, invoice)
	if err != nil {
		_ = delivery.Nack(false, true)
		return err
	}
	slog.InfoContext(ctx, "invoice created", "invoice_id", invoiceID)

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
				PaymentUrl:   fmt.Sprintf("%s/v1/payments/invoice/%s", s.paymentMakingConfig.Host, invoice.InvoiceNumber),
				Instructions: "Please use the following url to pay for your booking",
			},
		},
	}

	if err := validatePaymentRequest(paymentRequest); err != nil {
		_ = delivery.Nack(false, false)
		return err
	}

	if err := s.paymentProducer.Publish(ctx, paymentRequest, fmt.Sprintf("payment.%s.request", invoice.PayerId)); err != nil {
		_ = delivery.Nack(false, true)
		return err
	}

	return delivery.Ack(false)
}

func validatePaymentRequest(paymentRequest *dto.PaymentRequest) error {
	issuedAt := paymentRequest.GetIssuedAt()
	expiresAt := paymentRequest.GetExpiresAt()
	booking := paymentRequest.GetBooking()
	payer := paymentRequest.GetPayer()
	total := paymentRequest.GetTotal()
	options := paymentRequest.GetOptions()

	validator := validation.New().
		NotNil("paymentRequest", paymentRequest).
		NotBlank("invoice_number", paymentRequest.GetInvoiceNumber()).
		NotNil("total", total).
		PositiveValue("total.amount", total.GetAmount()).
		NotBlank("total.currency", total.GetCurrency()).
		NotNil("issued_at", issuedAt).
		NotNil("expires_at", expiresAt).
		NotNil("booking", booking).
		NotBlank("booking.cottage_name", booking.GetCottageName()).
		PositiveValue("booking.nights", booking.GetNights()).
		PositiveValue("booking.number_of_guests", booking.GetNumberOfGuests()).
		NotNil("payer", payer).
		NotBlank("payer.name", payer.GetName()).
		NotBlank("payer.email", payer.GetEmail()).
		PositiveValue("options", len(options))

	for idx, option := range options {
		field := fmt.Sprintf("options[%d]", idx)
		validator.
			NotNil(field, option).
			PositiveValue(field+".method", int32(option.GetMethod())).
			NotBlank(field+".payment_url", option.GetPaymentUrl()).
			NotBlank(field+".instructions", option.GetInstructions())
	}

	if issuedAt != nil && expiresAt != nil {
		validator.ValidPrecedence(issuedAt.AsTime(), expiresAt.AsTime())
	}

	return validator.Validate()
}
