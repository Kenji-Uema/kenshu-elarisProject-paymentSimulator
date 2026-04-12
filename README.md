# Payment Simulator

Handles invoice creation, payment reissue, and card payment simulation.

## Main Docs

See the main project documentation: <https://kenji-uema.github.io/kenshu-elarisProject-docs/>

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

## RabbitMQ Specification

The service has one RabbitMQ consumer and two RabbitMQ publishers.

### Wire Format

Inbound invoice-generation messages are expected as protobuf binary.

Outbound messages are published as protobuf binary:

- `content_type`: `application/protobuf`
- `delivery_mode`: persistent
- header `message_type`: full protobuf name, for example `paymentSimulator.payment.v1.PaymentRequest`

JSON examples below are schema examples for readability. The actual published body is binary protobuf.

### Consumes

Invoice generation requests:

- queue: `INVOICE_QUEUE_NAME`
- binding exchange: `INVOICE_BINDING_EXCHANGE_NAME`
- binding routing key: `INVOICE_BINDING_ROUTING_KEY`
- README defaults:
  - queue: `q.invoice.worker`
  - exchange: `ex.invoice.generate`
  - routing key: `booking.create-invoice`
- ack mode: manual ack (`INVOICE_CONSUME_AUTO_ACK=false`)

Expected message type:

- protobuf message: `paymentSimulator.invoice.CreateInvoicePaymentRequest`
- `content_type`: `application/protobuf`
- header `message_type`: full protobuf name, for example `cottageManager.invoice.CreateInvoicePaymentRequest`

Example schema body:

```json
{
  "idempotencyKey": "idem-main-it-1",
  "bookingId": "booking-123",
  "payerId": "payer-123",
  "issuedAt": "2026-03-09T12:00:00Z",
  "dueAt": "2026-03-10T12:00:00Z",
  "payer": {
    "name": "John Doe",
    "email": "john@example.com",
    "documentNumber": "11122233344",
    "billingAddress": "Main St"
  },
  "booking": {
    "cottageName": "Cabin",
    "nights": 2,
    "numberOfGuests": 3,
    "valuePerNight": {
      "amount": "10000",
      "currency": "USD"
    }
  },
  "total": {
    "amount": "20000",
    "currency": "USD"
  },
  "taxTotal": {
    "amount": "1000",
    "currency": "USD"
  },
  "discountTotal": {
    "amount": "0",
    "currency": "USD"
  }
}
```

JSON example above is for readability. The actual consumed body is binary protobuf.

Delivery behavior:

- invalid protobuf payload: `Nack(requeue=false)`
- invoice save or publish retryable failure: `Nack(requeue=true)`
- success: `Ack`

### Produces

Guest-facing payment requests:

- exchange: `GUEST_COMMUNICATION_EXCHANGE_NAME`
- exchange kind: `GUEST_COMMUNICATION_EXCHANGE_KIND`
- routing key pattern: `guest.<payer_id>`
- protobuf message: `paymentSimulator.payment.v1.PaymentRequest`

Example schema body:

```json
{
  "invoiceNumber": "INV-2026-0001",
  "total": {
    "amount": "20000",
    "currency": "USD"
  },
  "issuedAt": "2026-03-09T12:00:00Z",
  "expiresAt": "2026-03-10T12:00:00Z",
  "booking": {
    "cottageName": "Cabin",
    "nights": 2,
    "numberOfGuests": 3
  },
  "payer": {
    "name": "John Doe",
    "email": "john@example.com"
  },
  "options": [
    {
      "method": "PAYMENT_METHOD_CREDIT_CARD",
      "paymentUrl": "http://payment-simulator:8080/v1/payments/invoice/INV-2026-0001",
      "instructions": "Please use the following url to pay for your booking"
    }
  ]
}
```

Payment confirmations:

- exchange: `PAYMENT_EXCHANGE_NAME`
- exchange kind: `PAYMENT_EXCHANGE_KIND`
- routing key pattern: `booking.<booking_id>.confirmation`
- protobuf message: `paymentSimulator.payment.v1.PaymentConfirmation`

Example schema body:

```json
{
  "id": "b52ce28d-6d4b-4e6d-9a85-3c8a4f955f53",
  "bookingId": "booking-123",
  "payerId": "payer-123",
  "invoiceNumber": "INV-2026-0001",
  "receiptNumber": "RCPT-2026-0001",
  "confirmedAt": "2026-03-09T12:05:00Z"
}
```

### Routing Summary

- inbound invoice request: `INVOICE_BINDING_EXCHANGE_NAME` -> `INVOICE_QUEUE_NAME` with `INVOICE_BINDING_ROUTING_KEY`
- outbound payment request: `GUEST_COMMUNICATION_EXCHANGE_NAME` with `guest.<payer_id>`
- outbound payment confirmation: `PAYMENT_EXCHANGE_NAME` with `booking.<booking_id>.confirmation`

### Notes

- The payment-request publisher uses `GUEST_COMMUNICATION_EXCHANGE_*` config and currently routes with `guest.<payer_id>`.
- The payment-confirmation publisher uses `PAYMENT_EXCHANGE_*` config and routes with `booking.<booking_id>.confirmation`.
- Some integration helpers in the repo still bind payment-request test queues using `payment.<payer_id>.request`; that does not match the current application routing key built by `internal/app/invoice_service.go`.

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

## HTTP Examples

Set a base URL first:

```sh
BASE_URL=http://localhost:8080
```

Create a card payment for an existing invoice:

```sh
curl -i \
  -X POST "$BASE_URL/v1/payments/invoice/INV-2026-0001" \
  -H 'Content-Type: application/json' \
  -d '{
    "brand": "VISA",
    "number": "4111111111111111",
    "expMonth": 12,
    "expYear": 2030,
    "cvv": "123",
    "holderName": "John Doe"
  }'
```

Example success response:

```json
{
  "receiptNumber": "RCPT-2026-0009",
  "invoiceNumber": "INV-2026-0001",
  "status": "PAYMENT_STATUS_SUCCEEDED",
  "card": {
    "brand": "VISA",
    "last4": "1111"
  },
  "processedAt": "2026-03-09T12:00:00Z"
}
```

Reissue a payment request by booking and payer document:

```sh
curl -i \
  -X POST "$BASE_URL/v1/payments/payment_request/reissue" \
  -H 'Content-Type: application/json' \
  -d '{
    "bookingNumber": "booking-123",
    "documentNumber": "11122233344"
  }'
```

Example success response:

```json
{
  "invoiceNumber": "INV-2026-0001",
  "total": {
    "amount": "20000",
    "currency": "USD"
  },
  "issuedAt": "2026-03-09T12:00:00Z",
  "expiresAt": "2026-03-10T12:00:00Z",
  "booking": {
    "cottageName": "Cabin",
    "nights": 2,
    "numberOfGuests": 3
  },
  "payer": {
    "name": "John Doe",
    "email": "john@example.com"
  },
  "options": [
    {
      "method": "PAYMENT_METHOD_CREDIT_CARD",
      "paymentUrl": "http://payment-simulator:8080/v1/payments/invoice/INV-2026-0001",
      "instructions": "Please use the following url to pay for your booking"
    }
  ]
}
```

Check liveness:

```sh
curl -i "$BASE_URL/healthz"
```

Check readiness:

```sh
curl -i "$BASE_URL/readyz"
```

## Key Files

- `internal/main.go`
- `internal/app/invoice_service.go`
- `internal/app/payment_making_service.go`
- `internal/app/payment_reissue_service.go`
- `internal/transport/init.go`
