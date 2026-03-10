package integration_test

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Kenji-Uema/paymentSimulator/integration_test/helpers"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	. "github.com/onsi/ginkgo/v2"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ = Describe("reissue repayment flow", Ordered, func() {
	t := GinkgoT()

	var reissueResp dto.PaymentRequest
	var payResp dto.PayWithCardResponse

	It("reissues payment request for a seeded invoice", func() {
		reissueRespBody := helpers.PostJSON(
			t,
			fmt.Sprintf("http://127.0.0.1:%d/v1/payments/payment_request/reissue", suiteAppPort),
			[]byte(`{"bookingNumber":"booking-1001","documentNumber":"12345678900"}`),
		)

		if err := protojson.Unmarshal(reissueRespBody, &reissueResp); err != nil {
			t.Fatalf("unmarshal reissue response: %v", err)
		}
		if reissueResp.GetInvoiceNumber() != "INV-2026-0001" {
			t.Fatalf("unexpected reissued invoice number: got=%q want=%q", reissueResp.GetInvoiceNumber(), "INV-2026-0001")
		}
		if reissueResp.GetPayer().GetEmail() != "alice.johnson@example.com" {
			t.Fatalf("unexpected reissued payer email: %q", reissueResp.GetPayer().GetEmail())
		}
	})

	It("customer effectivates payment", func() {
		payRespBody := helpers.PostJSON(
			t,
			fmt.Sprintf("http://127.0.0.1:%d/v1/payments/invoice/%s", suiteAppPort, reissueResp.GetInvoiceNumber()),
			[]byte(`{"number":"4111111111111111","brand":"VISA","expMonth":12,"expYear":2030,"cvv":"123","holderName":"Alice Johnson"}`),
		)

		if err := protojson.Unmarshal(payRespBody, &payResp); err != nil {
			t.Fatalf("unmarshal pay response: %v", err)
		}
		if payResp.GetReceiptNumber() == "" {
			t.Fatalf("expected non-empty receipt number")
		}
	})

	It("does not reissue after invoice is paid", func() {
		respBody := helpers.PostJSON(
			t,
			fmt.Sprintf("http://127.0.0.1:%d/v1/payments/payment_request/reissue", suiteAppPort),
			[]byte(`{"bookingNumber":"booking-1001","documentNumber":"12345678900"}`),
			http.StatusBadRequest,
		)
		body := string(respBody)
		if !strings.Contains(body, "invoice is already paid") {
			t.Fatalf("expected already-paid message, got body=%s", body)
		}
	})

	It("does not pay again after invoice is paid", func() {
		respBody := helpers.PostJSON(
			t,
			fmt.Sprintf("http://127.0.0.1:%d/v1/payments/invoice/%s", suiteAppPort, reissueResp.GetInvoiceNumber()),
			[]byte(`{"number":"4111111111111111","brand":"VISA","expMonth":12,"expYear":2030,"cvv":"123","holderName":"Alice Johnson"}`),
			http.StatusBadRequest,
		)
		body := string(respBody)
		if !strings.Contains(body, "invoice is already paid") {
			t.Fatalf("expected already-paid message, got body=%s", body)
		}
	})
})
