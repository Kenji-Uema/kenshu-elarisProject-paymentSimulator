package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	appfakes "github.com/Kenji-Uema/paymentSimulator/internal/app/fakes"
	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	paymentgw "github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc/payment"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestServer_PaymentMakingRoute_UsesFakeService(t *testing.T) {
	paymentMakingFake := &appfakes.FakePaymentMakingService{
		PayWithCardFn: func(ctx context.Context, req *dto.PayWithCardRequest) (*dto.PayWithCardResponse, error) {
			_ = ctx
			return &dto.PayWithCardResponse{
				ReceiptNumber: "RCPT-2026-0009",
				Status:        dto.PaymentStatus_PAYMENT_STATUS_SUCCEEDED,
				Card:          &dto.CardSummary{Brand: req.GetCard().GetBrand(), Last4: "1111"},
				ProcessedAt:   timestamppb.New(time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)),
			}, nil
		},
	}
	reissueFake := &appfakes.FakePaymentReissueService{}

	_, baseURL := startGatewayServer(t, paymentMakingFake, reissueFake)

	body := []byte(`{"number":"4111111111111111","brand":"VISA","expMonth":12,"expYear":2030,"cvv":"123","holderName":"John Doe"}`)
	resp, err := http.Post(baseURL+"/v1/payments/invoice/INV-2026-0001", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status: got=%d body=%s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	var payload dto.PayWithCardResponse
	if err := protojson.Unmarshal(respBody, &payload); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}
	if payload.GetReceiptNumber() != "RCPT-2026-0009" {
		t.Fatalf("unexpected receipt number: got=%q want=%q", payload.GetReceiptNumber(), "RCPT-2026-0009")
	}
}

func TestServer_PaymentReissueRoute_UsesFakeService(t *testing.T) {
	paymentMakingFake := &appfakes.FakePaymentMakingService{}
	reissueFake := &appfakes.FakePaymentReissueService{
		ReissueFn: func(ctx context.Context, req *dto.ReissuePaymentRequest) (*dto.PaymentRequest, error) {
			_ = ctx
			return &dto.PaymentRequest{
				InvoiceNumber: "INV-2026-0001",
				Payer:         &dto.PayerSummary{Name: "John Doe", Email: "john@example.com"},
			}, nil
		},
	}

	_, baseURL := startGatewayServer(t, paymentMakingFake, reissueFake)

	body := []byte(`{"bookingNumber":"booking-1001","documentNumber":"12345678900"}`)
	resp, err := http.Post(baseURL+"/v1/payments/payment_request/reissue", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status: got=%d body=%s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	var payload dto.PaymentRequest
	if err := protojson.Unmarshal(respBody, &payload); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}
	if payload.GetInvoiceNumber() != "INV-2026-0001" {
		t.Fatalf("unexpected invoice number: got=%q want=%q", payload.GetInvoiceNumber(), "INV-2026-0001")
	}
}

func startGatewayServer(t *testing.T, paymentMakingServer paymentgw.PaymentMakingServiceServer, paymentReissueServer paymentgw.PaymentReissueServiceServer) (*Server, string) {
	t.Helper()

	mux := runtime.NewServeMux()
	if err := paymentgw.RegisterPaymentMakingServiceHandlerServer(context.Background(), mux, paymentMakingServer); err != nil {
		t.Fatalf("register payment making handler: %v", err)
	}
	if err := paymentgw.RegisterPaymentReissueServiceHandlerServer(context.Background(), mux, paymentReissueServer); err != nil {
		t.Fatalf("register payment reissue handler: %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve free port: %v", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	_ = ln.Close()

	cfg := config.ServerConfig{
		Host:                       "127.0.0.1",
		Port:                       addr.Port,
		ReadHeaderTimeoutInSeconds: 1,
		ReadTimeoutInSeconds:       1,
		WriteTimeoutInSeconds:      1,
		IdleTimeoutInSeconds:       1,
	}
	s := NewHttpServer(cfg, config.TelemetryConfig{}, mux, nil, nil)
	s.SetServer()

	done := make(chan struct{})
	go func() {
		s.Run(context.Background())
		close(done)
	}()

	waitForHealthy(t, addr.Port)

	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := s.Shutdown(shutdownCtx); err != nil {
			t.Fatalf("shutdown server: %v", err)
		}

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("server run did not stop after shutdown")
		}
	})

	return s, fmt.Sprintf("http://127.0.0.1:%d", addr.Port)
}

func waitForHealthy(t *testing.T, port int) {
	t.Helper()

	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)
	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, reqErr := client.Get(url)
		if reqErr == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not become ready in time")
		}
		time.Sleep(25 * time.Millisecond)
	}
}
