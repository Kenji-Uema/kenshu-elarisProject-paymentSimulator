package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/app"
	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/clock"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/db"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/logging"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/mq"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc/payment"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.ErrorContext(ctx, "payment simulator failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	configs, err := config.LoadConfigs()
	if err != nil {
		return fmt.Errorf("load configs: %w", err)
	}

	slog.SetDefault(logging.NewLogger(configs.LogConfig))
	slog.InfoContext(ctx, "Payment Simulator Starting")

	shutdownTelemetry, err := telemetry.Init(ctx, configs.TelemetryConfig, configs.AppConfig)
	if err != nil {
		return fmt.Errorf("failed to setup telemetry: %w", err)
	}

	mongoDb, err := db.NewMongoDb(ctx, configs.MongoConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	invoiceRepo := db.NewInvoiceRepo(mongoDb.Database)
	receiptRepo := db.NewReceiptRepo(mongoDb.Database)

	rabbitMqClient, err := mq.NewRabbitMqConnection(ctx, configs.RabbitMqConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	paymentProducer, err := mq.NewRabbitmqProducer(rabbitMqClient, configs.PaymentPublisherConfig.Publish)
	if err != nil {
		return fmt.Errorf("failed to create payment producer: %w", err)
	}
	err = paymentProducer.DeclareExchange(configs.PaymentPublisherConfig.Exchange)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	invoiceConsumer, err := mq.NewRabbitmqConsumer(rabbitMqClient, configs.InvoiceConsumerConfig.Consume)
	if err != nil {
		return fmt.Errorf("failed to create invoice consumer: %w", err)
	}
	err = invoiceConsumer.DeclareQueue(ctx, configs.InvoiceConsumerConfig.Queue)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}
	err = invoiceConsumer.BindQueue(ctx, configs.InvoiceConsumerConfig.Binding)
	if err != nil {
		return fmt.Errorf("failed to bind invoice queue: %w", err)
	}
	clockEmu, err := clock.NewClock(configs.ClockEmuConfig)
	if err != nil {
		return fmt.Errorf("failed to create clock emu: %w", err)
	}

	paymentMakingService := app.NewPaymentMakingServer(configs.PaymentMakingCardConfig, clockEmu, invoiceRepo, receiptRepo, paymentProducer)
	invoiceService := app.NewInvoiceService(invoiceRepo, clockEmu, invoiceConsumer, paymentProducer, configs.PaymentMakingCardConfig)
	paymentReissueService := app.NewPaymentReissueService(invoiceRepo, configs.PaymentMakingCardConfig)

	go invoiceService.StartInvoiceProcessing(ctx)

	gwMux := runtime.NewServeMux()
	err = payment.RegisterPaymentMakingServiceHandlerServer(ctx, gwMux, paymentMakingService)
	if err != nil {
		return fmt.Errorf("register gateway handler: %w", err)
	}
	err = payment.RegisterPaymentReissueServiceHandlerServer(ctx, gwMux, paymentReissueService)
	if err != nil {
		return fmt.Errorf("register gateway handler: %w", err)
	}

	httpServer := http.NewHttpServer(configs.ServerConfig, configs.TelemetryConfig, gwMux, rabbitMqClient, mongoDb)
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

	if err := clockEmu.Close(); err != nil {
		slog.ErrorContext(ctx, "close clock emu", "error", err)
	}

	return nil
}
