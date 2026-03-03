package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	invoiceConsumerTag        = "invoice-service"
	paymentRequestRoutingKey  = "payment.request.create"
	paymentRequestExpiryDelta = 24 * time.Hour
)

type InvoiceService interface {
	StartInvoiceProcessing(ctx context.Context)
}

type invoiceService struct {
	invoiceRepo     port.InvoiceRepo
	invoiceConsumer port.MqConsumer
	paymentProducer port.MqProducer
}

func NewInvoiceService(invoiceRepo port.InvoiceRepo, invoiceConsumer port.MqConsumer, paymentProducer port.MqProducer) InvoiceService {
	return &invoiceService{
		invoiceRepo:     invoiceRepo,
		invoiceConsumer: invoiceConsumer,
		paymentProducer: paymentProducer,
	}
}

func (s invoiceService) StartInvoiceProcessing(ctx context.Context) {
	deliveries, err := s.invoiceConsumer.Consume(ctx, config.ConsumeConfig{
		Consumer:  invoiceConsumerTag,
		AutoAck:   false,
		Exclusive: false,
		NoLocal:   false,
		NoWait:    false,
		Args:      nil,
	})
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

func (s invoiceService) processInvoiceDelivery(ctx context.Context, delivery amqp.Delivery) error {
	var createInvoiceRequest dto.CreateInvoicePaymentRequest
	if err := protojson.Unmarshal(delivery.Body, &createInvoiceRequest); err != nil {
		_ = delivery.Nack(false, false)
		return err
	}

	bookingID, err := bson.ObjectIDFromHex(createInvoiceRequest.GetBookingId())
	if err != nil {
		_ = delivery.Nack(false, false)
		return err
	}

	payerID, err := bson.ObjectIDFromHex(createInvoiceRequest.GetPayerId())
	if err != nil {
		_ = delivery.Nack(false, false)
		return err
	}

	invoice := document.Invoice{
		BookingId: bookingID,
		PayerId:   payerID,
		Payer: document.Payer{
			Name:           createInvoiceRequest.GetPayer().GetName(),
			Email:          createInvoiceRequest.GetPayer().GetEmail(),
			DocumentNumber: createInvoiceRequest.GetPayer().GetDocumentNumber(),
			BillingAddress: createInvoiceRequest.GetPayer().GetBillingAddress(),
		},
		Booking: document.BookingSnapshot{
			CottageName:    createInvoiceRequest.GetBooking().GetCottageName(),
			Nights:         createInvoiceRequest.GetBooking().GetNights(),
			NumberOfGuests: createInvoiceRequest.GetBooking().GetNumberOfGuests(),
			ValuePerNight: document.Money{
				Amount:   createInvoiceRequest.GetBooking().GetValuePerNight().GetAmount(),
				Currency: createInvoiceRequest.GetBooking().GetValuePerNight().GetCurrency(),
			},
		},
		Total: document.Money{
			Amount:   createInvoiceRequest.GetTotal().GetAmount(),
			Currency: createInvoiceRequest.GetTotal().GetCurrency(),
		},
		TaxTotal: document.Money{
			Amount:   createInvoiceRequest.GetTaxTotal().GetAmount(),
			Currency: createInvoiceRequest.GetTaxTotal().GetCurrency(),
		},
		DiscountTotal: document.Money{
			Amount:   createInvoiceRequest.GetDiscountTotal().GetAmount(),
			Currency: createInvoiceRequest.GetDiscountTotal().GetCurrency(),
		},
	}

	invoiceID, err := s.invoiceRepo.Add(ctx, invoice)
	if err != nil {
		_ = delivery.Nack(false, true)
		return err
	}

	paymentRequest := &dto.PaymentRequest{
		PaymentRequestId: uuid.NewString(),
		InvoiceNumber:    invoiceID.Hex(),
		Total: &dto.Money{
			Amount:   createInvoiceRequest.GetTotal().GetAmount(),
			Currency: createInvoiceRequest.GetTotal().GetCurrency(),
		},
		IssuedAt:  timestamppb.Now(),
		ExpiresAt: timestamppb.New(time.Now().Add(paymentRequestExpiryDelta)),
		Booking: &dto.BookingSummary{
			CottageName:    createInvoiceRequest.GetBooking().GetCottageName(),
			Nights:         createInvoiceRequest.GetBooking().GetNights(),
			NumberOfGuests: createInvoiceRequest.GetBooking().GetNumberOfGuests(),
		},
		Payer: &dto.PayerSummary{
			Name:  createInvoiceRequest.GetPayer().GetName(),
			Email: createInvoiceRequest.GetPayer().GetEmail(),
		},
		Options: []*dto.PaymentOption{
			{
				Method: dto.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD,
			},
		},
	}

	if err := s.paymentProducer.Publish(ctx, paymentRequest, config.PublishConfig{
		RoutingKey: paymentRequestRoutingKey,
		Mandatory:  false,
		Immediate:  false,
	}); err != nil {
		_ = delivery.Nack(false, true)
		return err
	}

	return delivery.Ack(false)
}
