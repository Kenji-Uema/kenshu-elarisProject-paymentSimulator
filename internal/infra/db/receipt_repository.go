package db

import (
	"context"
	"fmt"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
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
	var receipt document.Receipt
	if err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&receipt); err != nil {
		return document.Receipt{}, err
	}
	return receipt, nil
}

func (r receiptRepository) Add(ctx context.Context, receipt document.Receipt) (bson.ObjectID, error) {
	result, err := r.collection.InsertOne(ctx, receipt)
	if err != nil {
		return bson.ObjectID{}, err
	}

	insertedID, ok := result.InsertedID.(bson.ObjectID)
	if !ok {
		return bson.ObjectID{}, fmt.Errorf("unexpected inserted id type %T", result.InsertedID)
	}

	return insertedID, nil
}
