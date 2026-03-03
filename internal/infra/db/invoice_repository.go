package db

import (
	"context"
	"fmt"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
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

func (i invoiceRepo) Get(ctx context.Context, id bson.ObjectID) (document.Invoice, error) {
	var invoice document.Invoice
	if err := i.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&invoice); err != nil {
		return document.Invoice{}, err
	}
	return invoice, nil
}

func (i invoiceRepo) Add(ctx context.Context, invoice document.Invoice) (bson.ObjectID, error) {
	result, err := i.collection.InsertOne(ctx, invoice)
	if err != nil {
		return bson.ObjectID{}, err
	}

	insertedID, ok := result.InsertedID.(bson.ObjectID)
	if !ok {
		return bson.ObjectID{}, fmt.Errorf("unexpected inserted id type %T", result.InsertedID)
	}

	return insertedID, nil
}
