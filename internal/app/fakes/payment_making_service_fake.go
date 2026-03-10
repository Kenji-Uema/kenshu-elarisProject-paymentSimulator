package fakes

import (
	"context"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc/payment"
)

var _ payment.PaymentMakingServiceServer = (*FakePaymentMakingService)(nil)

type FakePaymentMakingService struct {
	payment.UnimplementedPaymentMakingServiceServer

	PayWithCardFn func(ctx context.Context, req *dto.PayWithCardRequest) (*dto.PayWithCardResponse, error)

	PayWithCardCallCount int
	LastPayWithCardCtx   context.Context
	LastPayWithCardReq   *dto.PayWithCardRequest
}

func (f *FakePaymentMakingService) PayWithCard(ctx context.Context, req *dto.PayWithCardRequest) (*dto.PayWithCardResponse, error) {
	f.PayWithCardCallCount++
	f.LastPayWithCardCtx = ctx
	f.LastPayWithCardReq = req

	if f.PayWithCardFn != nil {
		return f.PayWithCardFn(ctx, req)
	}

	return &dto.PayWithCardResponse{}, nil
}
