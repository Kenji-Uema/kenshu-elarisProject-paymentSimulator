package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/util"
	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/appErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type InvoiceService interface {
	StartInvoiceProcessing(ctx context.Context)
}

type invoiceService struct {
	clock           port.Clock
	invoiceRepo     port.InvoiceRepo
	invoiceConsumer port.MqConsumer
	paymentProducer port.MqProducer
	paymentHost     string
}

func NewInvoiceService(invoiceRepo port.InvoiceRepo, clock port.Clock,
	invoiceConsumer port.MqConsumer, paymentProducer port.MqProducer,
	paymentHost string) (InvoiceService, error) {
	if err := validation.New().
		NotNil("invoice_repo", invoiceRepo).
		NotNil("clock", clock).
		NotNil("invoice_consumer", invoiceConsumer).
		NotNil("payment_producer", paymentProducer).
		NotBlank("payment_host", paymentHost).
		Validate(); err != nil {
		return nil, err
	}

	return &invoiceService{
		clock:           clock,
		invoiceRepo:     invoiceRepo,
		invoiceConsumer: invoiceConsumer,
		paymentProducer: paymentProducer,
		paymentHost:     paymentHost,
	}, nil
}

func (s *invoiceService) StartInvoiceProcessing(ctx context.Context) {
	deliveries, err := s.invoiceConsumer.Consume(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "consume invoice queue", "error", err)
		return
	}

	for delivery := range deliveries {
		deliveryCtx, span := telemetry.StartConsumerSpan(ctx, delivery, "invoice.requests")
		if err := s.processInvoiceDelivery(deliveryCtx, delivery); err != nil {
			telemetry.RecordDeliveryError(span, err)
			slog.ErrorContext(deliveryCtx, "process invoice delivery", "delivery_tag", delivery.DeliveryTag, "error", err)
		}
		span.End()
	}
}

func (s *invoiceService) processInvoiceDelivery(ctx context.Context, delivery amqp.Delivery) error {
	ctx, span := telemetry.StartAppSpan(ctx, "payment.invoice.process_delivery",
		attribute.Int64("messaging.rabbitmq.delivery_tag", int64(delivery.DeliveryTag)),
		attribute.String("messaging.rabbitmq.routing_key", delivery.RoutingKey),
	)
	defer span.End()

	createInvoiceRequest, err := s.unmarshalRequest(delivery)
	if err != nil {
		telemetry.RecordSpanError(span, err)
		s.nack(ctx, delivery, false, false)
		return err
	}

	invoice, err := s.generateInvoiceDoc(ctx, createInvoiceRequest)
	if err != nil {
		telemetry.RecordSpanError(span, err)
		s.nack(ctx, delivery, false, false)
		return err
	}

	invoiceID, err := s.saveInvoice(ctx, invoice)
	if err != nil {
		telemetry.RecordSpanError(span, err)
		var alreadyExistsErr *appErrors.AlreadyExistsErr
		if errors.As(err, &alreadyExistsErr) {
			s.nack(ctx, delivery, false, false)
			return err
		}

		s.nack(ctx, delivery, false, true)
		return err
	}
	slog.InfoContext(ctx, "invoice created", "invoice_id", invoiceID)

	if err := s.publishPaymentRequest(ctx, invoice); err != nil {
		telemetry.RecordSpanError(span, err)
		var validationErr *validationErrors.ErrValidationConstrain
		if errors.As(err, &validationErr) {
			s.nack(ctx, delivery, false, false)
			return err
		}

		s.nack(ctx, delivery, false, true)
		return err
	}

	return delivery.Ack(false)
}

func (s *invoiceService) unmarshalRequest(delivery amqp.Delivery) (*dto.CreateInvoicePaymentRequest, error) {
	var createInvoiceRequest dto.CreateInvoicePaymentRequest
	contentType := strings.ToLower(strings.TrimSpace(delivery.ContentType))

	switch contentType {
	case "", "application/protobuf":
		if err := proto.Unmarshal(delivery.Body, &createInvoiceRequest); err == nil {
			return &createInvoiceRequest, nil
		} else if contentType == "application/protobuf" {
			return nil, &appErrors.CorruptedDataError{Msg: "could not unmarshal protobuf message to dto.CreateInvoicePaymentRequest", Err: err}
		}
	case "application/json":
		if err := protojson.Unmarshal(delivery.Body, &createInvoiceRequest); err == nil {
			return &createInvoiceRequest, nil
		} else {
			return nil, &appErrors.CorruptedDataError{Msg: "could not unmarshal json message to dto.CreateInvoicePaymentRequest", Err: err}
		}
	}

	if err := proto.Unmarshal(delivery.Body, &createInvoiceRequest); err == nil {
		return &createInvoiceRequest, nil
	}
	if err := protojson.Unmarshal(delivery.Body, &createInvoiceRequest); err == nil {
		return &createInvoiceRequest, nil
	}

	return nil, &appErrors.CorruptedDataError{
		Msg: fmt.Sprintf("could not unmarshal message to dto.CreateInvoicePaymentRequest; unsupported content_type=%q", delivery.ContentType),
		Err: errors.New("message body is neither valid protobuf nor valid protobuf json"),
	}
}

func (s *invoiceService) generateInvoiceDoc(ctx context.Context, createInvoiceRequest *dto.CreateInvoicePaymentRequest) (document.Invoice, error) {
	ctx, span := telemetry.StartAppSpan(ctx, "payment.invoice.generate",
		attribute.String("payment.booking_id", createInvoiceRequest.GetBookingId()),
		attribute.String("payment.payer_id", createInvoiceRequest.GetPayerId()),
	)
	defer span.End()

	now, err := s.clock.Now(ctx)
	if err != nil {
		telemetry.RecordSpanError(span, err)
		return document.Invoice{}, &appErrors.UnexpectedErr{Msg: "failed to get current time", Err: err}
	}

	invoiceNumber, err := util.GenerateHumanFriendlyId("INV", *now)
	if err != nil {
		telemetry.RecordSpanError(span, err)
		return document.Invoice{}, &appErrors.UnexpectedErr{Msg: "failed to generate invoice number", Err: err}
	}

	invoice, err := document.NewInvoiceFromProtoMessage(createInvoiceRequest, invoiceNumber, *now)
	if err != nil {
		telemetry.RecordSpanError(span, err)
		return document.Invoice{}, &appErrors.CorruptedDataError{Msg: "could not create invoice document from request", Err: err}
	}

	return invoice, nil
}

func (s *invoiceService) saveInvoice(ctx context.Context, invoice document.Invoice) (bson.ObjectID, error) {
	ctx, span := telemetry.StartAppSpan(ctx, "payment.invoice.save",
		attribute.String("payment.invoice_number", invoice.InvoiceNumber),
		attribute.String("payment.booking_id", invoice.BookingId),
	)
	defer span.End()

	invoiceID, err := s.invoiceRepo.Add(ctx, invoice)
	if err != nil {
		telemetry.RecordSpanError(span, err)
		var alreadyExistsErr *dbErrors.AlreadyExistsErr
		if errors.As(err, &alreadyExistsErr) {
			return bson.ObjectID{}, &appErrors.AlreadyExistsErr{Err: err}
		}

		return bson.ObjectID{}, &appErrors.UnexpectedErr{Msg: "failed to add invoice", Err: err}
	}
	return invoiceID, nil
}

func (s *invoiceService) publishPaymentRequest(ctx context.Context, invoice document.Invoice) error {
	ctx, span := telemetry.StartAppSpan(ctx, "payment.invoice.publish_request",
		attribute.String("payment.invoice_number", invoice.InvoiceNumber),
		attribute.String("payment.booking_id", invoice.BookingId),
		attribute.String("payment.payer_id", invoice.PayerId),
	)
	defer span.End()

	paymentRequest := buildPaymentRequest(invoice, s.paymentHost)

	if err := s.validatePaymentRequest(paymentRequest); err != nil {
		telemetry.RecordSpanError(span, err)
		return err
	}
	routingKey := fmt.Sprintf("%s%s", "guest.", invoice.PayerId)
	if err := s.paymentProducer.Publish(ctx, paymentRequest, routingKey); err != nil {
		telemetry.RecordSpanError(span, err)
		return &appErrors.PublishErr{
			Msg: fmt.Sprintf("failed to publish paymentRequest; routingKey: %s; invoiceNumber: %s", routingKey, paymentRequest.InvoiceNumber),
			Err: err}
	}

	return nil
}

func (s *invoiceService) nack(ctx context.Context, delivery amqp.Delivery, multiple bool, requeue bool) {
	if err := delivery.Nack(multiple, requeue); err != nil {
		slog.ErrorContext(ctx, "nack delivery", "delivery_tag", delivery.DeliveryTag, "error", err)
	}
}

func (s *invoiceService) validatePaymentRequest(paymentRequest *dto.PaymentRequest) error {
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
