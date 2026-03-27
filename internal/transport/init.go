package http

import (
	"context"
	"fmt"

	"github.com/Kenji-Uema/paymentSimulator/internal/app"
	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc/payment"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

func Init(ctx context.Context, configs config.Configs, components infra.Components, services app.Services) (*Server, error) {
	gwMux := runtime.NewServeMux()

	if err := payment.RegisterPaymentMakingServiceHandlerServer(ctx, gwMux, services.PaymentMaking); err != nil {
		return nil, fmt.Errorf("register payment making gateway handler: %w", err)
	}
	if err := payment.RegisterPaymentReissueServiceHandlerServer(ctx, gwMux, services.PaymentReissue); err != nil {
		return nil, fmt.Errorf("register payment reissue gateway handler: %w", err)
	}

	httpServer, err := NewHttpServer(
		configs.AppConfig.ServerConfig.Host,
		configs.AppConfig.ServerConfig.Port,
		configs.AppConfig.ServerConfig.ReadHeaderTimeoutInSeconds,
		configs.AppConfig.ServerConfig.ReadTimeoutInSeconds,
		configs.AppConfig.ServerConfig.WriteTimeoutInSeconds,
		configs.AppConfig.ServerConfig.IdleTimeoutInSeconds,
		configs.TelemetryConfig,
		gwMux,
		components.RabbitMQ,
		components.MongoDB,
	)
	if err != nil {
		return nil, fmt.Errorf("create http server: %w", err)
	}
	httpServer.SetServer()

	return httpServer, nil
}
