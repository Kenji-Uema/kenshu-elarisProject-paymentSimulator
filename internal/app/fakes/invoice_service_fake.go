package fakes

import (
	"context"
)

type FakeInvoiceService struct {
	StartInvoiceProcessingFn func(ctx context.Context)

	StartInvoiceProcessingCallCount int
	LastStartInvoiceProcessingCtx   context.Context
}

func (f *FakeInvoiceService) StartInvoiceProcessing(ctx context.Context) {
	f.StartInvoiceProcessingCallCount++
	f.LastStartInvoiceProcessingCtx = ctx

	if f.StartInvoiceProcessingFn != nil {
		f.StartInvoiceProcessingFn(ctx)
	}
}
