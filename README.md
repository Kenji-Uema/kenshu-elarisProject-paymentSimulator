# paymentSimulator

Handles invoice creation, payment reissue, and card payment simulation.

## What It Does

- consumes invoice-generation requests from RabbitMQ
- persists invoices and receipts in MongoDB
- publishes guest-facing payment requests
- accepts card payments over the HTTP gateway
- transitions invoices from `pending` to `paid`
- publishes payment-confirmation events back to the platform

## Interfaces

- HTTP gateway for payment APIs plus `/healthz` and `/readyz`
- RabbitMQ consumer for invoice-generation requests
- RabbitMQ publishers for payment requests and payment confirmations
- gRPC client to `clockSimulator`

## Local Commands

```sh
go run ./internal
go build ./internal
make test
make generate
make docker-build
```

## Minimum Env To Start

Optional vars with defaults, such as `SERVICE_NAME`, `VERSION`, collection names, `LOG_LEVEL`, and timeout values, are omitted here.

```sh
SERVICE_HOST=0.0.0.0
SERVICE_PORT=8080

CLOCK_EMU_GRPC_URL=<clock host>
CLOCK_EMU_GRPC_PORT=50051
PAYMENT_MAKING_CARD_HOST=http://payment-simulator:8080
FAIL_CHANCE=10

MONGO_INITDB_ROOT_USERNAME=<mongo user>
MONGO_INITDB_ROOT_PASSWORD=<mongo password>
MONGO_HOST=<mongo host>
MONGO_DATABASE=cottages

RABBITMQ_USERNAME=<rabbit user>
RABBITMQ_PASSWORD=<rabbit password>
RABBITMQ_HOST=<rabbit host>
RABBITMQ_PORT=5672

INVOICE_QUEUE_NAME=q.invoice.worker
INVOICE_BINDING_EXCHANGE_NAME=ex.invoice.generate
INVOICE_BINDING_ROUTING_KEY=booking.create-invoice

PAYMENT_EXCHANGE_NAME=ex.payment
PAYMENT_EXCHANGE_KIND=topic
GUEST_COMMUNICATION_EXCHANGE_NAME=ex.communication
GUEST_COMMUNICATION_EXCHANGE_KIND=direct

OTEL_EXPORTER_OTLP_ENDPOINT=<otel host>
OTEL_EXPORTER_OTLP_GRPC_PORT=4317
OTEL_EXPORTER_OTLP_HEALTH_PORT=13133
OTEL_EXPORTER_OTLP_INSECURE=true
```

## Configuration

Configuration is environment-driven. Start with:

- `internal/config/config.go`
- `internal/config/rabbitmq_config.go`

Important groups:

- service HTTP and timeout settings: `SERVICE_*`
- clock client: `CLOCK_EMU_*`
- payment behavior: `PAYMENT_MAKING_CARD_HOST`, `FAIL_CHANCE`
- MongoDB: `MONGO_*`
- RabbitMQ connection plus invoice/payment settings: `RABBITMQ_*`, `INVOICE_*`, `PAYMENT_*`
- telemetry: `OTEL_EXPORTER_OTLP_*`

## Public Routes

- `POST /v1/payments/invoice/{invoice_number}`
- `POST /v1/payments/payment_request/reissue`
- `GET /healthz`
- `GET /readyz`

## Key Files

- `internal/main.go`
- `internal/app/invoice_service.go`
- `internal/app/payment_making_service.go`
- `internal/app/payment_reissue_service.go`
- `internal/transport/init.go`
