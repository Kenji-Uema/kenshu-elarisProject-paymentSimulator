package config

import (
	"context"
	"log/slog"

	"github.com/caarlos0/env/v11"
)

type Secret string

func (s Secret) String() string {
	return "REDACTED"
}

type Configs struct {
	AppConfig
	Services
	MongoConfig
	RabbitMqConfig
	TelemetryConfig
}

type AppConfig struct {
	ServiceName string `env:"SERVICE_NAME" envDefault:"kenshu-elarisProject-paymentSimulator"`
	Version     string `env:"VERSION"`
	LogConfig   struct {
		Level int `env:"LOG_LEVEL,required" envDefault:"0"`
	}
	ServerConfig struct {
		Host                       string `env:"SERVICE_HOST,required"`
		Port                       int    `env:"SERVICE_PORT,required"`
		ReadHeaderTimeoutInSeconds int    `env:"READ_HEADER_TIMEOUT_IN_SECONDS,required" envDefault:"5"`
		ReadTimeoutInSeconds       int    `env:"READ_TIMEOUT_IN_SECONDS,required" envDefault:"10"`
		WriteTimeoutInSeconds      int    `env:"WRITE_TIMEOUT_IN_SECONDS,required" envDefault:"15"`
		IdleTimeoutInSeconds       int    `env:"IDLE_TIMEOUT_IN_SECONDS,required" envDefault:"60"`
	}
}

type Services struct {
	ClockSimulatorConfig struct {
		GrpcHost string `env:"CLOCK_EMU_GRPC_URL,required"`
		GrpcPort int    `env:"CLOCK_EMU_GRPC_PORT,required"`
	}
	PaymentMakingCardConfig struct {
		Host       string `env:"PAYMENT_MAKING_CARD_HOST,required"`
		FailChance int    `env:"FAIL_CHANCE,required"`
	}
}

type MongoConfig struct {
	Username                        Secret `env:"MONGO_INITDB_ROOT_USERNAME,required"`
	Password                        Secret `env:"MONGO_INITDB_ROOT_PASSWORD,required"`
	Host                            string `env:"MONGO_HOST,required"`
	Database                        string `env:"MONGO_DATABASE,required"`
	ReplicaSet                      string `env:"MONGO_REPLICA_SET" envDefault:""`
	ConnectionTimeoutInSeconds      int    `env:"MONGO_CONNECTION_TIMEOUT_IN_SECONDS" envDefault:"10"`
	ServerSelectionTimeoutInSeconds int    `env:"MONGO_SERVER_SELECTION_TIMEOUT_IN_SECONDS" envDefault:"10"`
	PingTimeoutInSeconds            int    `env:"MONGO_PING_TIMEOUT_IN_SECONDS" envDefault:"5"`
	StartupTimeoutInSeconds         int    `env:"MONGO_STARTUP_TIMEOUT_IN_SECONDS" envDefault:"10"`
	MaxConnIdleTimeInSeconds        int    `env:"MONGO_MAX_CONN_IDLE_TIME_IN_SECONDS" envDefault:"60"`
	MaxPoolSize                     uint64 `env:"MONGO_MAX_POOL_SIZE" envDefault:"100"`
	MinPoolSize                     uint64 `env:"MONGO_MIN_POOL_SIZE" envDefault:"0"`
	RetryWrites                     bool   `env:"MONGO_RETRY_WRITES" envDefault:"true"`
	Collections                     struct {
		InvoiceCollection string `env:"INVOICE_COLLECTION" envDefault:"Invoice"`
		ReceiptCollection string `env:"RECEIPT_COLLECTION" envDefault:"Receipt"`
	}
}

type RabbitMqConfig struct {
	Username   Secret `env:"RABBITMQ_USERNAME,required"`
	Password   Secret `env:"RABBITMQ_PASSWORD,required"`
	Host       string `env:"RABBITMQ_HOST,required"`
	Port       int    `env:"RABBITMQ_PORT,required"`
	Publishers struct {
		PaymentExchange            ExchangeConfig `envPrefix:"PAYMENT_EXCHANGE_"`
		PaymentPublish             PublishConfig  `envPrefix:"PAYMENT_PUBLISH_"`
		GuestCommunicationExchange ExchangeConfig `envPrefix:"GUEST_COMMUNICATION_EXCHANGE_"`
		GuestCommunicationPublish  PublishConfig  `envPrefix:"GUEST_COMMUNICATION_PUBLISH_"`
	}
	Consumers struct {
		InvoiceQueue   QueueConfig   `envPrefix:"INVOICE_QUEUE_"`
		InvoiceBinding BindingConfig `envPrefix:"INVOICE_BINDING_"`
		InvoiceConsume ConsumeConfig `envPrefix:"INVOICE_CONSUME_"`
	}
}

type TelemetryConfig struct {
	OTLPEndpoint             string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,required"`
	OTLPGrpcPort             int    `env:"OTEL_EXPORTER_OTLP_GRPC_PORT,required"`
	OTLPHealthPort           int    `env:"OTEL_EXPORTER_OTLP_HEALTH_PORT,required"`
	OTLPInsecure             bool   `env:"OTEL_EXPORTER_OTLP_INSECURE,required"`
	ShutdownTimeoutInSeconds int    `env:"OTEL_SHUTDOWN_TIMEOUT_IN_SECONDS" envDefault:"15"`
}

func LoadConfigs() (Configs, error) {
	var cfg Configs
	if err := env.Parse(&cfg); err != nil {
		return cfg, err
	}

	slog.InfoContext(context.Background(), "config loaded", "config", cfg)

	return cfg, nil
}
