package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/util"
	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/appErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/encoding/protojson"
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
		if err := s.processInvoiceDelivery(ctx, delivery); err != nil {
			slog.ErrorContext(ctx, "process invoice delivery", "delivery_tag", delivery.DeliveryTag, "error", err)
		}
	}
}

func (s *invoiceService) processInvoiceDelivery(ctx context.Context, delivery amqp.Delivery) error {
	createInvoiceRequest, err := s.unmarshalRequest(delivery)
	if err != nil {
		s.nack(ctx, delivery, false, false)
		return err
	}

	invoice, err := s.generateInvoiceDoc(ctx, createInvoiceRequest)
	if err != nil {
		s.nack(ctx, delivery, false, false)
		return err
	}

	invoiceID, err := s.saveInvoice(ctx, invoice)
	if err != nil {
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
	if err := protojson.Unmarshal(delivery.Body, &createInvoiceRequest); err != nil {
		return nil, &appErrors.CorruptedDataError{Msg: "could not unmarshal message to dto.CreateInvoicePaymentRequest", Err: err}
	}
	return &createInvoiceRequest, nil
}

func (s *invoiceService) generateInvoiceDoc(ctx context.Context, createInvoiceRequest *dto.CreateInvoicePaymentRequest) (document.Invoice, error) {
	now, err := s.clock.Now(ctx)
	if err != nil {
		return document.Invoice{}, &appErrors.UnexpectedErr{Msg: "failed to get current time", Err: err}
	}

	invoiceNumber, err := util.GenerateHumanFriendlyId("INV", *now)
	if err != nil {
		return document.Invoice{}, &appErrors.UnexpectedErr{Msg: "failed to generate invoice number", Err: err}
	}

	invoice, err := document.NewInvoiceFromProtoMessage(createInvoiceRequest, invoiceNumber, *now)
	if err != nil {
		return document.Invoice{}, &appErrors.CorruptedDataError{Msg: "could not create invoice document from request", Err: err}
	}

	return invoice, nil
}

func (s *invoiceService) saveInvoice(ctx context.Context, invoice document.Invoice) (bson.ObjectID, error) {
	invoiceID, err := s.invoiceRepo.Add(ctx, invoice)
	if err != nil {
		var alreadyExistsErr *dbErrors.AlreadyExistsErr
		if errors.As(err, &alreadyExistsErr) {
			return bson.ObjectID{}, &appErrors.AlreadyExistsErr{Err: err}
		}

		return bson.ObjectID{}, &appErrors.UnexpectedErr{Msg: "failed to add invoice", Err: err}
	}
	return invoiceID, nil
}

func (s *invoiceService) publishPaymentRequest(ctx context.Context, invoice document.Invoice) error {
	paymentRequest := buildPaymentRequest(invoice, s.paymentHost)

	if err := s.validatePaymentRequest(paymentRequest); err != nil {
		return err
	}
	routingKey := fmt.Sprintf("payment.%s.request", invoice.PayerId)
	if err := s.paymentProducer.Publish(ctx, paymentRequest, routingKey); err != nil {
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
