package fakes

import (
	"context"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/document"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ port.InvoiceRepo = (*FakeInvoiceRepo)(nil)

type FakeInvoiceRepo struct {
	GetFn                                  func(ctx context.Context, invoiceNumber string) (document.Invoice, error)
	AddFn                                  func(ctx context.Context, invoice document.Invoice) (bson.ObjectID, error)
	FindByBookingNumberAndDocumentNumberFn func(ctx context.Context, bookingNumber string, documentNumber string) (document.Invoice, error)

	GetCallCount  int
	AddCallCount  int
	FindCallCount int

	LastGetInvoiceNumber   string
	LastAddedInvoice       document.Invoice
	LastFindBookingNumber  string
	LastFindDocumentNumber string
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
