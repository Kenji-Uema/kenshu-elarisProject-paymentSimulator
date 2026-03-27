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
	"github.com/Kenji-Uema/paymentSimulator/internal/infra"
	transport "github.com/Kenji-Uema/paymentSimulator/internal/transport"
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

	slog.InfoContext(ctx, "Payment Simulator Starting")

	components, err := infra.Init(ctx, configs)
	if err != nil {
		return err
	}

	services, err := app.Init(configs, components)
	if err != nil {
		return fmt.Errorf("init app services: %w", err)
	}
	services.Start(ctx)

	httpServer, err := transport.Init(ctx, configs, components, services)
	if err != nil {
		return err
	}
	go httpServer.Run(ctx)

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	shutdown(shutdownCtx, components, httpServer)

	return nil
}

func shutdown(ctx context.Context, components infra.Components, httpServer *transport.Server) {
	slog.InfoContext(ctx, "shutdown signal received; shutting down")

	if err := components.ShutdownTelemetry(ctx); err != nil {
		slog.ErrorContext(ctx, "telemetry shutdown", "error", err)
	}

	if err := components.RabbitMQ.Close(); err != nil {
		slog.ErrorContext(ctx, "close rabbitmq connection", "error", err)
	}

	if err := components.PaymentProducer.CloseChannel(); err != nil {
		slog.ErrorContext(ctx, "close payment producer", "error", err)
	}

	if err := components.InvoiceConsumer.CloseChannel(); err != nil {
		slog.ErrorContext(ctx, "close invoice consumer", "error", err)
	}

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.ErrorContext(ctx, "server shutdown failed", "err", err)
	}

	if err := components.Clock.Close(); err != nil {
		slog.ErrorContext(ctx, "close clock emu", "error", err)
	}
}
