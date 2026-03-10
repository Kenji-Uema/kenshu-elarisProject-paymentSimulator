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

var _ = Describe("default flow", Ordered, func() {
	t := GinkgoT()

	var amqpCh *amqp.Channel
	var paymentRequestDeliveries, confirmationDeliveries <-chan amqp.Delivery

	BeforeAll(func() {
		amqpCh = helpers.SetupRabbitmqChannel(t, suiteRabbitHost, suiteRabbitPort)
		paymentRequestDeliveries, confirmationDeliveries = helpers.SetupRabbitmqExchangesAndQueues(t, amqpCh)
	})

	It("receives an request to generate an invoice", func() {
		createReq := helpers.ValidCreateInvoiceRequest()
		createReqBody, err := protojson.Marshal(createReq)
		if err != nil {
			t.Fatalf("marshal create invoice request: %v", err)
		}

		if err := amqpCh.PublishWithContext(context.Background(), "", "invoice.requests", false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        createReqBody,
		}); err != nil {
			t.Fatalf("publish invoice creation request: %v", err)
		}
	})

	var paymentReq dto.PaymentRequest
	It("customer receives an payment request", func() {
		paymentRequestMsg := helpers.WaitForDelivery(t, paymentRequestDeliveries, 20*time.Second)

		if err := protojson.Unmarshal(paymentRequestMsg.Body, &paymentReq); err != nil {
			t.Fatalf("unmarshal payment request: %v", err)
		}
		if paymentReq.GetInvoiceNumber() == "" {
			t.Fatalf("expected non-empty invoice number in payment request")
		}
		if paymentReq.GetPayer().GetEmail() != "john@example.com" {
			t.Fatalf("unexpected payer email: %q", paymentReq.GetPayer().GetEmail())
		}
	})

	var payResp dto.PayWithCardResponse
	It("customer effectivate payment", func() {
		payBody := []byte(`{"number":"4111111111111111","brand":"VISA","expMonth":12,"expYear":2030,"cvv":"123","holderName":"John Doe"}`)
		payRespBody := helpers.PostJSON(t, fmt.Sprintf("http://127.0.0.1:%d/v1/payments/invoice/%s", suiteAppPort, paymentReq.GetInvoiceNumber()), payBody)

		if err := protojson.Unmarshal(payRespBody, &payResp); err != nil {
			t.Fatalf("unmarshal pay response: %v", err)
		}
		if payResp.GetReceiptNumber() == "" {
			t.Fatalf("expected non-empty receipt number")
		}
	})

	It("paymentSimulator notifies other system about payment confirmation", func() {
		confirmationMsg := helpers.WaitForDelivery(t, confirmationDeliveries, 20*time.Second)
		var confirmation dto.PaymentConfirmation
		if err := protojson.Unmarshal(confirmationMsg.Body, &confirmation); err != nil {
			t.Fatalf("unmarshal confirmation: %v", err)
		}
		if confirmation.GetInvoiceNumber() != paymentReq.GetInvoiceNumber() {
			t.Fatalf("unexpected confirmation invoice number: got=%q want=%q", confirmation.GetInvoiceNumber(), paymentReq.GetInvoiceNumber())
		}
		if confirmation.GetReceiptNumber() != payResp.GetReceiptNumber() {
			t.Fatalf("unexpected confirmation receipt number: got=%q want=%q", confirmation.GetReceiptNumber(), payResp.GetReceiptNumber())
		}
	})

	It("reissue payment request", func() {
		reissueBody := []byte(`{"bookingNumber":"booking-123","documentNumber":"11122233344"}`)
		reissueRespBody := helpers.PostJSON(t, fmt.Sprintf("http://127.0.0.1:%d/v1/payments/payment_request/reissue", suiteAppPort), reissueBody)

		var reissueResp dto.PaymentRequest
		if err := protojson.Unmarshal(reissueRespBody, &reissueResp); err != nil {
			t.Fatalf("unmarshal reissue response: %v", err)
		}
		if reissueResp.GetInvoiceNumber() != paymentReq.GetInvoiceNumber() {
			t.Fatalf("unexpected reissued invoice number: got=%q want=%q", reissueResp.GetInvoiceNumber(), paymentReq.GetInvoiceNumber())
		}
		if reissueResp.GetPayer().GetEmail() != "john@example.com" {
			t.Fatalf("unexpected reissued payer email: %q", reissueResp.GetPayer().GetEmail())
		}
	})
})
