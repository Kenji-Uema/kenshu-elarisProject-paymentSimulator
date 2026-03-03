package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/app"
	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/db"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/logging"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/mq"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/telemetry"
	http "github.com/Kenji-Uema/paymentSimulator/internal/transport"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc/payment"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	configs, err := config.LoadConfigs()
	exitOnError(ctx, "load configs", err)

	slog.SetDefault(logging.NewLogger(configs.LogConfig))
	slog.InfoContext(ctx, "Clock Emulation Starting")

	shutdownTelemetry, err := telemetry.Init(ctx, configs.TelemetryConfig, configs.AppConfig)
	exitOnError(ctx, "failed to setup telemetry", err)

	mongoDb, err := db.NewMongoDbFromConfig(ctx, configs.MongoConfig)
	exitOnError(ctx, "failed to connect to MongoDB", err)

	invoiceRepo := db.NewInvoiceRepo(mongoDb.Database)
	receiptRepo := db.NewReceiptRepo(mongoDb.Database)

	rabbitMqClient, err := mq.NewRabbitMqConnection(configs.RabbitMqConfig)
	exitOnError(ctx, "failed to connect to RabbitMQ", err)

	paymentProducer, err := mq.NewRabbitmqProducer(rabbitMqClient)
	exitOnError(ctx, "failed to create payment producer", err)
	err = paymentProducer.DeclareExchange(config.ExchangeConfig{})
	exitOnError(ctx, "failed to declare exchange", err)

	invoiceConsumer, err := mq.NewRabbitmqConsumer(rabbitMqClient)
	exitOnError(ctx, "failed to create invoice consumer", err)
	err = invoiceConsumer.DeclareQueue(config.QueueConfig{})
	exitOnError(ctx, "failed to declare queue", err)

	paymentMakingCardService := app.NewPaymentMakingCardServer(config.PaymentMakingCardConfig{}, invoiceRepo, receiptRepo, paymentProducer)
	invoiceService := app.NewInvoiceService(invoiceRepo, invoiceConsumer, paymentProducer)

	invoiceService.StartInvoiceProcessing(ctx)

	gwMux := runtime.NewServeMux()
	err = payment.RegisterPaymentMakingCardServiceHandlerServer(ctx, gwMux, paymentMakingCardService)
	exitOnError(ctx, "register gateway handler", err)

	httpServer := http.NewHttpServer(configs.ServerConfig, configs.TelemetryConfig, gwMux, rabbitMqClient)
	httpServer.SetServer()
	go httpServer.Run(ctx)

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slog.InfoContext(shutdownCtx, "shutdown signal received; shutting down")

	if err := shutdownTelemetry(shutdownCtx); err != nil {
		slog.ErrorContext(shutdownCtx, "telemetry shutdown", "error", err)
	}

	if err := rabbitMqClient.Close(); err != nil {
		slog.ErrorContext(ctx, "close rabbitmq connection", "error", err)
	}

	if err := paymentProducer.CloseChannel(); err != nil {
		slog.ErrorContext(ctx, "close payment producer", "error", err)
	}

	if err := invoiceConsumer.CloseChannel(); err != nil {
		slog.ErrorContext(ctx, "close invoice consumer", "error", err)
	}

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.ErrorContext(shutdownCtx, "server shutdown failed", "err", err)
	}
}

func exitOnError(ctx context.Context, errMsg string, err error) {
	if err != nil {
		slog.ErrorContext(ctx, errMsg, "error", err)
		os.Exit(1)
	}
}
