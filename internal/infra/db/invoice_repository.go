package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type invoiceRepo struct {
	collection *mongo.Collection
}

func NewInvoiceRepo(db *mongo.Database) (port.InvoiceRepo, error) {
	if err := validation.New().NotNil("mongo_db", db).Validate(); err != nil {
		return nil, err
	}
	return &invoiceRepo{collection: db.Collection("invoices")}, nil
}

func (r invoiceRepo) FindByInvoiceNumber(ctx context.Context, invoiceNumber string) (document.Invoice, error) {
	if err := validation.New().NotBlank("invoiceNumber", invoiceNumber).Validate(); err != nil {
		return document.Invoice{}, err
	}

	filter := bson.M{"invoice_number": invoiceNumber}

	invoice, err := r.find(ctx, filter)
	if err != nil {
		return document.Invoice{}, err
	}

	return invoice, nil
}

func (r invoiceRepo) FindByBookingNumberAndDocumentNumber(ctx context.Context, bookingNumber string, documentNumber string) (document.Invoice, error) {
	if err := validation.New().
		NotBlank("bookingNumber", bookingNumber).
		NotBlank("documentNumber", documentNumber).Validate(); err != nil {

		return document.Invoice{}, err
	}

	filter := bson.M{
		"booking_id":            bookingNumber,
		"payer.document_number": documentNumber,
	}

	invoice, err := r.find(ctx, filter)
	if err != nil {
		return invoice, err
	}

	return invoice, nil
}

func (r invoiceRepo) Add(ctx context.Context, invoice document.Invoice) (bson.ObjectID, error) {
	if err := validation.New().
		NotBlank("invoice.invoice_number", invoice.InvoiceNumber).
		NotBlank("invoice.status", invoice.Status).
		NotBlank("invoice.idempotency_id", invoice.IdempotencyId).
		NotBlank("invoice.booking_id", invoice.BookingId).
		NotBlank("invoice.payer_id", invoice.PayerId).
		NotBlank("invoice.payer.document_number", invoice.Payer.DocumentNumber).
		PositiveValue("invoice.total.amount", invoice.Total.Amount).
		NotBlank("invoice.total.currency", invoice.Total.Currency).
		NotZeroValue("invoice.issued_at", invoice.IssuedAt).
		NotZeroValue("invoice.due_at", invoice.DueAt).Validate(); err != nil {
		return bson.ObjectID{}, err
	}
	if !document.IsValidInvoiceStatus(invoice.Status) {
		return bson.ObjectID{}, &validationErrors.ErrValidationConstrain{
			Field:   "invoice.status",
			Message: "must be one of pending, paid",
		}
	}

	res, err := r.collection.InsertOne(ctx, invoice)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return bson.NilObjectID, &dbErrors.AlreadyExistsErr{Err: err}
		}

		return bson.ObjectID{}, &dbErrors.UnexpectedErr{Msg: "failed to insert invoice", Err: err}
	}

	invoiceId, ok := res.InsertedID.(bson.ObjectID)
	if !ok {
		slog.ErrorContext(ctx, "unexpected inserted id type", "type", fmt.Sprintf("%T", res.InsertedID))
		return bson.ObjectID{}, &dbErrors.UnexpectedErr{
			Msg: "unexpected inserted id type",
			Err: fmt.Errorf("unexpected inserted id type: %T", res.InsertedID),
		}
	}

	return invoiceId, nil
}

func (r invoiceRepo) UpdateStatus(ctx context.Context, invoiceNumber string, status string, updatedAt time.Time) error {
	if err := validation.New().
		NotBlank("invoiceNumber", invoiceNumber).
		NotBlank("status", status).
		NotZeroValue("updatedAt", updatedAt).Validate(); err != nil {
		return err
	}
	if !document.IsValidInvoiceStatus(status) {
		return &validationErrors.ErrValidationConstrain{
			Field:   "status",
			Message: "must be one of pending, paid",
		}
	}

	filter := bson.M{"invoice_number": invoiceNumber}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": updatedAt,
		},
	}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return &dbErrors.UnexpectedErr{Msg: "failed to update invoice status", Err: err}
	}

	if res.MatchedCount == 0 {
		return &dbErrors.InvoiceNotFoundErr{Err: mongo.ErrNoDocuments}
	}

	return nil
}

func (r invoiceRepo) find(ctx context.Context, filter bson.M) (document.Invoice, error) {
	var invoice document.Invoice
	if err := r.collection.FindOne(ctx, filter).Decode(&invoice); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			slog.WarnContext(ctx, "invoice not found", "filter", filter)
			return document.Invoice{}, &dbErrors.InvoiceNotFoundErr{Err: err}
		}

		slog.ErrorContext(ctx, "failed to decode invoice", "error", err, "filter", filter)
		return document.Invoice{}, &dbErrors.CorruptedDataErr{Err: err}
	}
	return invoice, nil
}
