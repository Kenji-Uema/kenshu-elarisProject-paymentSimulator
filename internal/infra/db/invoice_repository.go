package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type invoiceRepo struct {
	collection *mongo.Collection
}

func NewInvoiceRepo(db *mongo.Database) port.InvoiceRepo {
	return &invoiceRepo{collection: db.Collection("invoices")}
}

func (r invoiceRepo) Get(ctx context.Context, invoiceNumber string) (document.Invoice, error) {
	if err := validation.New().NotBlank("invoiceNumber", invoiceNumber).Validate(); err != nil {
		return document.Invoice{}, err
	}

	filter := bson.M{"invoice_number": invoiceNumber}

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

func (r invoiceRepo) Add(ctx context.Context, invoice document.Invoice) (bson.ObjectID, error) {
	if err := validation.New().NotNil("invoice", invoice).Validate(); err != nil {
		return bson.ObjectID{}, err
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
