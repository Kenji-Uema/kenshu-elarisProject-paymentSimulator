package config

import (
	"log/slog"

	"github.com/caarlos0/env/v11"
)

type Secret string

func (s Secret) String() string {
	return "REDACTED"
}

type Configs struct {
	AppConfig
	ServerConfig
	MongoConfig
	RabbitMqConfig
	PaymentExchangeConfig
	TelemetryConfig
	LogConfig
}

type AppConfig struct {
	ServiceName string `env:"SERVICE_NAME"`
	Version     string `env:"VERSION"`
}

type ServerConfig struct {
	Host                       string `env:"SERVICE_HOST,required"`
	Port                       int    `env:"SERVICE_PORT,required"`
	ReadHeaderTimeoutInSeconds int    `env:"READ_HEADER_TIMEOUT_IN_SECONDS,required" envDefault:"5"`
	ReadTimeoutInSeconds       int    `env:"READ_TIMEOUT_IN_SECONDS,required" envDefault:"10"`
	WriteTimeoutInSeconds      int    `env:"WRITE_TIMEOUT_IN_SECONDS,required" envDefault:"15"`
	IdleTimeoutInSeconds       int    `env:"IDLE_TIMEOUT_IN_SECONDS,required" envDefault:"60"`
}

type PaymentMakingCardConfig struct {
	FailChance int `env:"FAIL_CHANCE,required"`
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
}

type InvoiceCollectionConfig struct {
	Name string `env:"INVOICE_COLLECTION" envDefault:"Invoice"`
}

type ReceiptCollectionConfig struct {
	Name string `env:"RECEIPT_COLLECTION" envDefault:"Receipt"`
}

type RabbitMqConfig struct {
	Username Secret `env:"RABBITMQ_USERNAME,required"`
	Password Secret `env:"RABBITMQ_PASSWORD,required"`
	Host     string `env:"RABBITMQ_HOST,required"`
	Port     int    `env:"RABBITMQ_PORT,required"`
}

type PaymentExchangeConfig struct {
	Name           string `env:"PAYMENT_EXCHANGE_NAME,required" envDefault:"ex.payment"`
	Kind           string `env:"PAYMENT_EXCHANGE_KIND,required" envDefault:"topic"`
	Durable        bool   `env:"PAYMENT_EXCHANGE_DURABLE,required"`
	AutoDelete     bool   `env:"PAYMENT_EXCHANGE_AUTO_DELETE,required"`
	Internal       bool   `env:"PAYMENT_EXCHANGE_INTERNAL,required"`
	NoWait         bool   `env:"PAYMENT_EXCHANGE_NO_WAIT,required"`
	ConfirmNotWait bool   `env:"PAYMENT_EXCHANGE_CONFIRM_NOT_WAIT,required"`
}

type InvoiceQueueConfig struct {
	Name       string `env:"INVOICE_QUEUE_NAME,required" envDefault:"q.generate_invoice.worker"`
	Durable    bool   `env:"INVOICE_QUEUE_DURABLE,required" envDefault:"true"`
	AutoDelete bool   `env:"INVOICE_QUEUE_AUTO_DELETE,required" envDefault:"false"`
	Exclusive  bool   `env:"INVOICE_QUEUE_EXCLUSIVE,required" envDefault:"false"`
	NoWait     bool   `env:"INVOICE_QUEUE_NO_WAIT,required" envDefault:"false"`
}

type TelemetryConfig struct {
	OTLPEndpoint   string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,required"`
	OTLPGrpcPort   int    `env:"OTEL_EXPORTER_OTLP_GRPC_PORT,required"`
	OTLPHealthPort int    `env:"OTEL_EXPORTER_OTLP_HEALTH_PORT,required"`
	OTLPInsecure   bool   `env:"OTEL_EXPORTER_OTLP_INSECURE,required"`
}

type LogConfig struct {
	Level int `env:"LOG_LEVEL,required" envDefault:"0"`
}

func LoadConfigs() (Configs, error) {
	var cfg Configs
	if err := env.Parse(&cfg); err != nil {
		return cfg, err
	}

	slog.Info("config loaded", "config", cfg)

	return cfg, nil
}
