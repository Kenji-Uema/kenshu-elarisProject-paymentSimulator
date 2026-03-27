package app

import (
	"context"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc/payment"
)

type Services struct {
	PaymentMaking  payment.PaymentMakingServiceServer
	PaymentReissue payment.PaymentReissueServiceServer
	Invoice        InvoiceService
}

func Init(configs config.Configs, components infra.Components) (Services, error) {
	paymentMaking, err := NewPaymentMakingServer(
		configs.Services.PaymentMakingCardConfig.FailChance,
		components.Clock,
		components.InvoiceRepo,
		components.ReceiptRepo,
		components.PaymentProducer,
	)
	if err != nil {
		return Services{}, err
	}

	paymentReissue, err := NewPaymentReissueService(
		components.InvoiceRepo,
		configs.Services.PaymentMakingCardConfig.Host,
	)
	if err != nil {
		return Services{}, err
	}

	invoice, err := NewInvoiceService(
		components.InvoiceRepo,
		components.Clock,
		components.InvoiceConsumer,
		components.PaymentProducer,
		configs.Services.PaymentMakingCardConfig.Host,
	)
	if err != nil {
		return Services{}, err
	}

	return Services{
		PaymentMaking:  paymentMaking,
		PaymentReissue: paymentReissue,
		Invoice:        invoice,
	}, nil
}

func (s Services) Start(ctx context.Context) {
	go s.Invoice.StartInvoiceProcessing(ctx)
}
