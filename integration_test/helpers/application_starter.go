package helpers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

func ApplicationStart(t TestReporter, appPort int, clockHost string, clockPort int, mongoHost string, rabbitHost string, rabbitPort int) (stop func(), runErr <-chan error) {
	t.Helper()

	env := map[string]string{
		"SERVICE_NAME":                   "payment-simulator",
		"VERSION":                        "test",
		"SERVICE_HOST":                   "127.0.0.1",
		"SERVICE_PORT":                   fmt.Sprintf("%d", appPort),
		"READ_HEADER_TIMEOUT_IN_SECONDS": "1",
		"READ_TIMEOUT_IN_SECONDS":        "2",
		"WRITE_TIMEOUT_IN_SECONDS":       "2",
		"IDLE_TIMEOUT_IN_SECONDS":        "3",
		"CLOCK_EMU_GRPC_URL":             clockHost,
		"CLOCK_EMU_GRPC_PORT":            fmt.Sprintf("%d", clockPort),
		"MONGO_INITDB_ROOT_USERNAME":     "test_user",
		"MONGO_INITDB_ROOT_PASSWORD":     "test_pass",
		"MONGO_HOST":                     mongoHost,
		"MONGO_DATABASE":                 "test_db",
		"RABBITMQ_USERNAME":              "test_user",
		"RABBITMQ_PASSWORD":              "test_pass",
		"RABBITMQ_HOST":                  rabbitHost,
		"RABBITMQ_PORT":                  fmt.Sprintf("%d", rabbitPort),
		"PAYMENT_EXCHANGE_NAME":          "payment.events",
		"PAYMENT_EXCHANGE_KIND":          "direct",
		"PAYMENT_PUBLISH_MANDATORY":      "false",
		"PAYMENT_PUBLISH_IMMEDIATE":      "false",
		"INVOICE_QUEUE_NAME":             "invoice.requests",
		"INVOICE_QUEUE_DURABLE":          "false",
		"INVOICE_QUEUE_AUTO_DELETE":      "true",
		"INVOICE_BINDING_EXCHANGE_NAME":  "payment.events",
		"INVOICE_BINDING_ROUTING_KEY":    "invoice.request",
		"INVOICE_CONSUME_CONSUMER":       "invoice-main-integration",
		"INVOICE_CONSUME_AUTO_ACK":       "false",
		"PAYMENT_MAKING_CARD_HOST":       "http://127.0.0.1",
		"FAIL_CHANCE":                    "0",
		"OTEL_EXPORTER_OTLP_ENDPOINT":    "127.0.0.1",
		"OTEL_EXPORTER_OTLP_GRPC_PORT":   "4317",
		"OTEL_EXPORTER_OTLP_HEALTH_PORT": "13133",
		"OTEL_EXPORTER_OTLP_INSECURE":    "true",
		"LOG_LEVEL":                      "0",
	}
	runCtx, cancelRun := context.WithCancel(context.Background())
	runErrCh := make(chan error, 1)

	cmd := exec.CommandContext(runCtx, "go", "run", "../internal")
	cmd.Env = envWithOverrides(os.Environ(), env)

	go func() {
		err := cmd.Run()
		if runCtx.Err() != nil {
			runErrCh <- nil
			return
		}
		runErrCh <- err
	}()

	stop = func() {
		cancelRun()
	}

	return stop, runErrCh
}

func envWithOverrides(baseEnv []string, overrides map[string]string) []string {
	result := make([]string, 0, len(baseEnv)+len(overrides))
	seen := make(map[string]struct{}, len(baseEnv))

	for _, entry := range baseEnv {
		parts := bytes.SplitN([]byte(entry), []byte("="), 2)
		if len(parts) != 2 {
			result = append(result, entry)
			continue
		}

		key := string(parts[0])
		if value, ok := overrides[key]; ok {
			result = append(result, fmt.Sprintf("%s=%s", key, value))
		} else {
			result = append(result, entry)
		}
		seen[key] = struct{}{}
	}

	for key, value := range overrides {
		if _, ok := seen[key]; ok {
			continue
		}
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}

	return result
}
