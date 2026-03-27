# paymentSimulator

Handles invoice creation, payment reissue, and card payment simulation.

## Responsibilities

- consume invoice creation requests
- persist invoices
- reissue payment requests for existing unpaid invoices
- process card payments
- transition invoice state from `pending` to `paid`
- persist receipts
- publish payment confirmation events

## Interfaces

- HTTP gateway for payment endpoints
- RabbitMQ consumer for invoice requests
- RabbitMQ publisher for payment confirmations and payment requests
- gRPC client to `clockSimulator`

## Run

```sh
go run ./internal
```

## Build and test

```sh
make build
make test
make docker-build
```

## Configuration

Configuration is environment-driven. See:

- `internal/config/config.go`
- `internal/config/rabbitmq_config.go`

Important families:

- service HTTP: `SERVICE_*`, timeout settings
- clock client: `CLOCK_EMU_*`
- payment behavior: `PAYMENT_MAKING_CARD_HOST`, `FAIL_CHANCE`
- MongoDB: `MONGO_*`
- RabbitMQ: `RABBITMQ_*`, `PAYMENT_*`, `INVOICE_*`
- telemetry: `OTEL_EXPORTER_OTLP_*`

## Entry points

- `internal/main.go`
- `internal/app/invoice_service.go`
- `internal/app/payment_making_service.go`
- `internal/app/payment_reissue_service.go`
