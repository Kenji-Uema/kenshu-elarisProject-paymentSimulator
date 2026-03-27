package config

import "testing"

func TestLoadConfigs(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := LoadConfigs()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.AppConfig.ServerConfig.Host != "127.0.0.1" {
		t.Fatalf("unexpected SERVICE_HOST: got=%q", cfg.AppConfig.ServerConfig.Host)
	}
	if cfg.AppConfig.ServerConfig.Port != 8080 {
		t.Fatalf("unexpected SERVICE_PORT: got=%d", cfg.AppConfig.ServerConfig.Port)
	}
	if cfg.MongoConfig.Database != "test_db" {
		t.Fatalf("unexpected MONGO_DATABASE: got=%q", cfg.MongoConfig.Database)
	}
	if cfg.RabbitMqConfig.Publishers.PaymentExchange.Name != "payment.events" {
		t.Fatalf("unexpected PAYMENT_EXCHANGE_NAME: got=%q", cfg.RabbitMqConfig.Publishers.PaymentExchange.Name)
	}
	if cfg.RabbitMqConfig.Consumers.InvoiceQueue.Name != "invoice.requests" {
		t.Fatalf("unexpected INVOICE_QUEUE_NAME: got=%q", cfg.RabbitMqConfig.Consumers.InvoiceQueue.Name)
	}
}

func TestLoadConfigs_MissingRequiredEnv(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("SERVICE_PORT", "not-a-number")

	_, err := LoadConfigs()
	if err == nil {
		t.Fatal("expected error when required env var has invalid value")
	}
}

func TestSecretString_RedactsValue(t *testing.T) {
	if got := Secret("s3cr3t").String(); got != "REDACTED" {
		t.Fatalf("unexpected secret string representation: got=%q", got)
	}
}

func setRequiredEnv(t *testing.T) {
	t.Helper()

	t.Setenv("SERVICE_NAME", "payment-simulator")
	t.Setenv("VERSION", "test")
	t.Setenv("SERVICE_HOST", "127.0.0.1")
	t.Setenv("SERVICE_PORT", "8080")
	t.Setenv("READ_HEADER_TIMEOUT_IN_SECONDS", "1")
	t.Setenv("READ_TIMEOUT_IN_SECONDS", "2")
	t.Setenv("WRITE_TIMEOUT_IN_SECONDS", "3")
	t.Setenv("IDLE_TIMEOUT_IN_SECONDS", "4")

	t.Setenv("CLOCK_EMU_GRPC_URL", "127.0.0.1")
	t.Setenv("CLOCK_EMU_GRPC_PORT", "50051")

	t.Setenv("MONGO_INITDB_ROOT_USERNAME", "user")
	t.Setenv("MONGO_INITDB_ROOT_PASSWORD", "pass")
	t.Setenv("MONGO_HOST", "127.0.0.1:27017")
	t.Setenv("MONGO_DATABASE", "test_db")

	t.Setenv("RABBITMQ_USERNAME", "user")
	t.Setenv("RABBITMQ_PASSWORD", "pass")
	t.Setenv("RABBITMQ_HOST", "127.0.0.1")
	t.Setenv("RABBITMQ_PORT", "5672")

	t.Setenv("PAYMENT_EXCHANGE_NAME", "payment.events")
	t.Setenv("PAYMENT_EXCHANGE_KIND", "direct")
	t.Setenv("PAYMENT_PUBLISH_MANDATORY", "false")
	t.Setenv("PAYMENT_PUBLISH_IMMEDIATE", "false")

	t.Setenv("INVOICE_QUEUE_NAME", "invoice.requests")
	t.Setenv("INVOICE_BINDING_EXCHANGE_NAME", "payment.events")
	t.Setenv("INVOICE_BINDING_ROUTING_KEY", "invoice.request")
	t.Setenv("INVOICE_CONSUME_CONSUMER", "invoice-consumer")

	t.Setenv("PAYMENT_MAKING_CARD_HOST", "http://127.0.0.1")
	t.Setenv("FAIL_CHANCE", "0")

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "127.0.0.1")
	t.Setenv("OTEL_EXPORTER_OTLP_GRPC_PORT", "4317")
	t.Setenv("OTEL_EXPORTER_OTLP_HEALTH_PORT", "13133")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")

	t.Setenv("LOG_LEVEL", "0")
}
