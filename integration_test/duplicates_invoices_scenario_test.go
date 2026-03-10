package integration_test

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/integration_test/helpers"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	. "github.com/onsi/ginkgo/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ = Describe("duplicates flow", Ordered, func() {
	t := GinkgoT()

	var amqpCh *amqp.Channel
	var paymentRequestDeliveries <-chan amqp.Delivery
	var firstReq, secondReq dto.PaymentRequest

	BeforeAll(func() {
		amqpCh = helpers.SetupRabbitmqChannel(t, suiteRabbitHost, suiteRabbitPort)

		paymentRequestQueue := fmt.Sprintf("it.payment.dup.request.%d", time.Now().UnixNano())
		helpers.MustDeclareExchange(t, amqpCh, "payment.events")
		helpers.MustDeclareAndBindQueue(t, amqpCh, paymentRequestQueue, "payment.events", "payment.payer-123.request")

		var err error
		paymentRequestDeliveries, err = amqpCh.Consume(paymentRequestQueue, "", true, false, false, false, nil)
		if err != nil {
			t.Fatalf("consume payment request queue: %v", err)
		}
	})

	It("receives duplicated requests to generate invoices", func() {
		createReq := helpers.ValidCreateInvoiceRequest()
		createReq.IdempotencyKey = "idem-dup-it-1"
		createReq.BookingId = "booking-dup-123"
		createReq.Payer.DocumentNumber = "99988877766"

		createReqBody, err := protojson.Marshal(createReq)
		if err != nil {
			t.Fatalf("marshal create invoice request: %v", err)
		}

		for i := 0; i < 2; i++ {
			if err := amqpCh.PublishWithContext(context.Background(), "", "invoice.requests", false, false, amqp.Publishing{
				ContentType: "application/json",
				Body:        createReqBody,
			}); err != nil {
				t.Fatalf("publish invoice creation request: %v", err)
			}
		}
	})

	It("customer receives two payment requests", func() {
		firstMsg := helpers.WaitForDelivery(t, paymentRequestDeliveries, 20*time.Second)
		secondMsg := helpers.WaitForDelivery(t, paymentRequestDeliveries, 20*time.Second)

		if err := protojson.Unmarshal(firstMsg.Body, &firstReq); err != nil {
			t.Fatalf("unmarshal first payment request: %v", err)
		}
		if err := protojson.Unmarshal(secondMsg.Body, &secondReq); err != nil {
			t.Fatalf("unmarshal second payment request: %v", err)
		}
	})

	It("processes duplicated idempotency requests with current behavior", func() {
		if firstReq.GetInvoiceNumber() == "" || secondReq.GetInvoiceNumber() == "" {
			t.Fatal("expected non-empty invoice numbers for duplicated requests")
		}
		if firstReq.GetInvoiceNumber() == secondReq.GetInvoiceNumber() {
			t.Fatalf("expected different invoice numbers, got=%q", firstReq.GetInvoiceNumber())
		}
		if firstReq.GetPayer().GetEmail() != "john@example.com" || secondReq.GetPayer().GetEmail() != "john@example.com" {
			t.Fatalf("unexpected payer email(s): first=%q second=%q", firstReq.GetPayer().GetEmail(), secondReq.GetPayer().GetEmail())
		}
	})
})
