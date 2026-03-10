package fakes

import (
	"context"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ port.InvoiceRepo = (*FakeInvoiceRepo)(nil)

type FakeInvoiceRepo struct {
	GetFn                                  func(ctx context.Context, invoiceNumber string) (document.Invoice, error)
	AddFn                                  func(ctx context.Context, invoice document.Invoice) (bson.ObjectID, error)
	FindByBookingNumberAndDocumentNumberFn func(ctx context.Context, bookingNumber string, documentNumber string) (document.Invoice, error)
	UpdateStatusFn                         func(ctx context.Context, invoiceNumber string, status string, updatedAt time.Time) error

	GetCallCount          int
	AddCallCount          int
	FindCallCount         int
	UpdateStatusCallCount int

	LastGetInvoiceNumber   string
	LastAddedInvoice       document.Invoice
	LastFindBookingNumber  string
	LastFindDocumentNumber string
	LastUpdatedInvoice     string
	LastUpdatedStatus      string
	LastUpdatedAt          time.Time
}

func (f *FakeInvoiceRepo) FindByInvoiceNumber(ctx context.Context, invoiceNumber string) (document.Invoice, error) {
	f.GetCallCount++
	f.LastGetInvoiceNumber = invoiceNumber

	if f.GetFn != nil {
		return f.GetFn(ctx, invoiceNumber)
	}

	return document.Invoice{}, nil
}

func (f *FakeInvoiceRepo) Add(ctx context.Context, invoice document.Invoice) (bson.ObjectID, error) {
	f.AddCallCount++
	f.LastAddedInvoice = invoice

	if f.AddFn != nil {
		return f.AddFn(ctx, invoice)
	}

	return bson.NilObjectID, nil
}

func (f *FakeInvoiceRepo) FindByBookingNumberAndDocumentNumber(ctx context.Context, bookingNumber string, documentNumber string) (document.Invoice, error) {
	f.FindCallCount++
	f.LastFindBookingNumber = bookingNumber
	f.LastFindDocumentNumber = documentNumber

	if f.FindByBookingNumberAndDocumentNumberFn != nil {
		return f.FindByBookingNumberAndDocumentNumberFn(ctx, bookingNumber, documentNumber)
	}

	return document.Invoice{}, nil
}

func (f *FakeInvoiceRepo) UpdateStatus(ctx context.Context, invoiceNumber string, status string, updatedAt time.Time) error {
	f.UpdateStatusCallCount++
	f.LastUpdatedInvoice = invoiceNumber
	f.LastUpdatedStatus = status
	f.LastUpdatedAt = updatedAt

	if f.UpdateStatusFn != nil {
		return f.UpdateStatusFn(ctx, invoiceNumber, status, updatedAt)
	}

	return nil
}
