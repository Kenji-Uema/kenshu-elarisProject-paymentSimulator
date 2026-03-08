package port

import (
	"context"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type InvoiceRepo interface {
	Get(ctx context.Context, invoiceNumber string) (document.Invoice, error)
	Add(ctx context.Context, invoice document.Invoice) (bson.ObjectID, error)
	FindByBookingNumberAndDocumentNumber(ctx context.Context, bookingNumber string, documentNumber string) (document.Invoice, error)
}

type ReceiptRepo interface {
	Get(ctx context.Context, id bson.ObjectID) (document.Receipt, error)
	Add(ctx context.Context, receipt document.Receipt) (bson.ObjectID, error)
}
