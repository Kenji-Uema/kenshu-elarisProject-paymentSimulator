package infra

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	grpcclock "github.com/Kenji-Uema/paymentSimulator/internal/infra/clock"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/db"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/logging"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/mq"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
)

type Components struct {
	ShutdownTelemetry func(context.Context) error
	MongoDB           *db.Db
	InvoiceRepo       port.InvoiceRepo
	ReceiptRepo       port.ReceiptRepo
	RabbitMQ          *mq.RabbitMqConnection
	PaymentProducer   port.MqProducer
	GuestCommProducer port.MqProducer
	InvoiceConsumer   port.MqConsumer
	Clock             port.Clock
}

func Init(ctx context.Context, configs config.Configs) (Components, error) {
	slog.SetDefault(logging.NewLogger(configs.AppConfig.LogConfig.Level))

	shutdownTelemetry, err := telemetry.Init(ctx, configs.TelemetryConfig, configs.AppConfig)
	if err != nil {
		return Components{}, fmt.Errorf("failed to setup telemetry: %w", err)
	}

	mongoDB, err := db.NewMongoDb(ctx, configs.MongoConfig)
	if err != nil {
		return Components{}, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	invoiceRepo, err := db.NewInvoiceRepo(mongoDB.Database)
	if err != nil {
		return Components{}, fmt.Errorf("failed to create invoice repo: %w", err)
	}

	receiptRepo, err := db.NewReceiptRepo(mongoDB.Database)
	if err != nil {
		return Components{}, fmt.Errorf("failed to create receipt repo: %w", err)
	}

	rabbitMQ, err := mq.NewRabbitMqConnection(ctx, configs.RabbitMqConfig)
	if err != nil {
		return Components{}, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	paymentProducer, err := mq.NewRabbitmqProducer(rabbitMQ, configs.RabbitMqConfig.Publishers.PaymentPublish)
	if err != nil {
		return Components{}, fmt.Errorf("failed to create payment producer: %w", err)
	}
	if err := paymentProducer.DeclareExchange(configs.RabbitMqConfig.Publishers.PaymentExchange); err != nil {
		return Components{}, fmt.Errorf("failed to declare exchange: %w", err)
	}

	guestCommProducer, err := mq.NewRabbitmqProducer(rabbitMQ, configs.RabbitMqConfig.Publishers.GuestCommunicationPublish)
	if err != nil {
		return Components{}, fmt.Errorf("failed to create guest communication producer: %w", err)
	}
	if err := guestCommProducer.DeclareExchange(configs.RabbitMqConfig.Publishers.GuestCommunicationExchange); err != nil {
		return Components{}, fmt.Errorf("failed to declare guest communication exchange: %w", err)
	}

	invoiceConsumer, err := mq.NewRabbitmqConsumer(rabbitMQ, configs.RabbitMqConfig.Consumers.InvoiceConsume)
	if err != nil {
		return Components{}, fmt.Errorf("failed to create invoice consumer: %w", err)
	}
	if err := invoiceConsumer.DeclareQueue(ctx, configs.RabbitMqConfig.Consumers.InvoiceQueue); err != nil {
		return Components{}, fmt.Errorf("failed to declare queue: %w", err)
	}
	if err := invoiceConsumer.BindQueue(ctx, configs.RabbitMqConfig.Consumers.InvoiceBinding); err != nil {
		return Components{}, fmt.Errorf("failed to bind invoice queue: %w", err)
	}

	clockClient, err := grpcclock.NewClock(configs.Services)
	if err != nil {
		return Components{}, fmt.Errorf("failed to create clock emu: %w", err)
	}

	return Components{
		ShutdownTelemetry: shutdownTelemetry,
		MongoDB:           mongoDB,
		InvoiceRepo:       invoiceRepo,
		ReceiptRepo:       receiptRepo,
		RabbitMQ:          rabbitMQ,
		PaymentProducer:   paymentProducer,
		GuestCommProducer: guestCommProducer,
		InvoiceConsumer:   invoiceConsumer,
		Clock:             clockClient,
	}, nil
}
