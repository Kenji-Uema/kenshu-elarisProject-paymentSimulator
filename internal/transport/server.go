package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/db"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/mq"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/http/probe"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type Server struct {
	server           *http.Server
	paymentServerMux *runtime.ServeMux
	rabbitmqClient   *mq.RabbitMqConnection
	mongoClient      *db.Db
	serverConfig     config.ServerConfig
	telemetryConfig  config.TelemetryConfig
}

func NewHttpServer(config config.ServerConfig, telemetryConfig config.TelemetryConfig,
	paymentServerMux *runtime.ServeMux, rabbitmqClient *mq.RabbitMqConnection, mongoClient *db.Db) *Server {

	return &Server{serverConfig: config, telemetryConfig: telemetryConfig,
		paymentServerMux: paymentServerMux, rabbitmqClient: rabbitmqClient, mongoClient: mongoClient}
}

func (s *Server) SetServer() {
	rootMux := http.NewServeMux()
	rootMux.HandleFunc("/healthz", probe.HealthHandler)
	rootMux.HandleFunc("/readyz", probe.ReadinessHandler(s.rabbitmqClient, s.mongoClient))
	rootMux.Handle("/", s.traceContextMiddleware(otelhttp.NewHandler(s.paymentServerMux, "payment-http-gateway")))

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", s.serverConfig.Host, s.serverConfig.Port),
		Handler:           rootMux,
		ReadHeaderTimeout: time.Duration(s.serverConfig.ReadHeaderTimeoutInSeconds) * time.Second,
		ReadTimeout:       time.Duration(s.serverConfig.ReadTimeoutInSeconds) * time.Second,
		WriteTimeout:      time.Duration(s.serverConfig.WriteTimeoutInSeconds) * time.Second,
		IdleTimeout:       time.Duration(s.serverConfig.IdleTimeoutInSeconds) * time.Second,
	}
	s.server = server
}

func (s *Server) traceContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) Run(ctx context.Context) {
	slog.InfoContext(ctx, "http server listening", "addr", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.ErrorContext(ctx, "server error", "err", err)
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
