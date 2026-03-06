package logging

import (
	"context"
	"log/slog"
	"os"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"

	"go.opentelemetry.io/otel/trace"
)

type TraceHandler struct {
	base slog.Handler
}

func (h *TraceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.base.Handle(ctx, r)
}

func (h *TraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TraceHandler{base: h.base.WithAttrs(attrs)}
}

func (h *TraceHandler) WithGroup(name string) slog.Handler {
	return &TraceHandler{base: h.base.WithGroup(name)}
}

func NewLogger(config config.LogConfig) *slog.Logger {
	base := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.Level(config.Level),
		AddSource: true,
	})

	traceHandler := &TraceHandler{base: base}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return slog.New(traceHandler).With(
		"app", hostname,
	)
}
