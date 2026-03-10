package fakes

import (
	"context"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ port.ReceiptRepo = (*FakeReceiptRepo)(nil)

type FakeReceiptRepo struct {
	GetFn func(ctx context.Context, id bson.ObjectID) (document.Receipt, error)
	AddFn func(ctx context.Context, receipt document.Receipt) (bson.ObjectID, error)

	GetCallCount int
	AddCallCount int

	LastGetID      bson.ObjectID
	LastAddReceipt document.Receipt
}

func (f *FakeReceiptRepo) GetById(ctx context.Context, id bson.ObjectID) (document.Receipt, error) {
	f.GetCallCount++
	f.LastGetID = id

	if f.GetFn != nil {
		return f.GetFn(ctx, id)
	}

	return document.Receipt{}, nil
}

func (f *FakeReceiptRepo) Add(ctx context.Context, receipt document.Receipt) (bson.ObjectID, error) {
	f.AddCallCount++
	f.LastAddReceipt = receipt

	if f.AddFn != nil {
		return f.AddFn(ctx, receipt)
	}

	return bson.NilObjectID, nil
}
