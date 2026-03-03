package probe

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/mq"
)

const otelHealthTimeout = 2 * time.Second

var otelHealthClient = &http.Client{Timeout: otelHealthTimeout}

func ReadinessHandler(rabbitMqClient *mq.RabbitMqConnection, telemetryCfg config.TelemetryConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rabbitMqClient.IsConnectionOpen() {
			slog.ErrorContext(r.Context(), "rabbitmq connection is not open")
			http.Error(w, "rabbitmq connection closed", http.StatusServiceUnavailable)
			return
		}
		if err := checkOtelCollector(r.Context(), telemetryCfg); err != nil {
			slog.ErrorContext(r.Context(), "otel-collector check failed", "error", err)
			http.Error(w, "otel collector unreachable", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func checkOtelCollector(ctx context.Context, cfg config.TelemetryConfig) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s:%d/health", cfg.OTLPEndpoint, cfg.OTLPHealthPort), nil)
	if err != nil {
		return err
	}

	resp, err := otelHealthClient.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.ErrorContext(ctx, "close response body", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("otel collector health check failed: status %d", resp.StatusCode)
	}
	return nil
}
