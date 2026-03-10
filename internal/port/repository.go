package port

import (
	"context"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type InvoiceRepo interface {
	FindByInvoiceNumber(ctx context.Context, invoiceNumber string) (document.Invoice, error)
	FindByBookingNumberAndDocumentNumber(ctx context.Context, bookingNumber string, documentNumber string) (document.Invoice, error)
	UpdateStatus(ctx context.Context, invoiceNumber string, status string, updatedAt time.Time) error
	Add(ctx context.Context, invoice document.Invoice) (bson.ObjectID, error)
}

type ReceiptRepo interface {
	GetById(ctx context.Context, id bson.ObjectID) (document.Receipt, error)
	Add(ctx context.Context, receipt document.Receipt) (bson.ObjectID, error)
}
