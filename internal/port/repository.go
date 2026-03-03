package port

import (
	"context"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type InvoiceRepo interface {
	Get(ctx context.Context, id bson.ObjectID) (document.Invoice, error)
	Add(ctx context.Context, booking document.Invoice) (bson.ObjectID, error)
}

type ReceiptRepo interface {
	Get(ctx context.Context, id bson.ObjectID) (document.Receipt, error)
	Add(ctx context.Context, receipt document.Receipt) (bson.ObjectID, error)
}
