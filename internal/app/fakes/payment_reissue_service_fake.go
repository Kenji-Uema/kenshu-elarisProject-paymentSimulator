package fakes

import (
	"context"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc/payment"
)

var _ payment.PaymentReissueServiceServer = (*FakePaymentReissueService)(nil)

type FakePaymentReissueService struct {
	payment.UnimplementedPaymentReissueServiceServer

	ReissueFn func(ctx context.Context, req *dto.ReissuePaymentRequest) (*dto.PaymentRequest, error)

	ReissueCallCount int
	LastReissueCtx   context.Context
	LastReissueReq   *dto.ReissuePaymentRequest
}

func (f *FakePaymentReissueService) Reissue(ctx context.Context, req *dto.ReissuePaymentRequest) (*dto.PaymentRequest, error) {
	f.ReissueCallCount++
	f.LastReissueCtx = ctx
	f.LastReissueReq = req

	if f.ReissueFn != nil {
		return f.ReissueFn(ctx, req)
	}

	return &dto.PaymentRequest{}, nil
}
