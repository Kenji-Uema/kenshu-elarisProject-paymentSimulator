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

type receiptRepository struct {
	collection *mongo.Collection
}

func NewReceiptRepo(db *mongo.Database) port.ReceiptRepo {
	return &receiptRepository{collection: db.Collection("receipts")}
}

func (r receiptRepository) Get(ctx context.Context, id bson.ObjectID) (document.Receipt, error) {
	if err := validation.New().NotNilObjectID("id", id).Validate(); err != nil {
		return document.Receipt{}, err
	}

	filter := bson.M{"_id": id}

	var receipt document.Receipt
	if err := r.collection.FindOne(ctx, filter).Decode(&receipt); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			slog.WarnContext(ctx, "receipt not found", "filter", filter)
			return document.Receipt{}, &dbErrors.ReceiptNotFoundErr{Err: err}
		}

		slog.ErrorContext(ctx, "failed to decode receipt", "error", err, "filter", filter)
		return document.Receipt{}, &dbErrors.CorruptedDataErr{Err: err}
	}

	return receipt, nil
}

func (r receiptRepository) Add(ctx context.Context, receipt document.Receipt) (bson.ObjectID, error) {
	if err := validation.New().NotNil("receipt", receipt).Validate(); err != nil {
		return bson.ObjectID{}, err
	}

	result, err := r.collection.InsertOne(ctx, receipt)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return bson.NilObjectID, &dbErrors.AlreadyExistsErr{Err: err}
		}

		return bson.NilObjectID, &dbErrors.UnexpectedErr{Msg: "failed to insert receipt", Err: err}
	}

	insertedID, ok := result.InsertedID.(bson.ObjectID)
	if !ok {
		slog.ErrorContext(ctx, "unexpected inserted id type", "type", fmt.Sprintf("%T", result.InsertedID))
		return bson.NilObjectID, &dbErrors.UnexpectedErr{
			Msg: "unexpected inserted id type",
			Err: fmt.Errorf("unexpected inserted id type: %T", result.InsertedID),
		}
	}

	return insertedID, nil
}
