package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
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
	serverConfig     httpServerConfig
	telemetryConfig  config.TelemetryConfig
}

type httpServerConfig struct {
	host                       string
	port                       int
	readHeaderTimeoutInSeconds int
	readTimeoutInSeconds       int
	writeTimeoutInSeconds      int
	idleTimeoutInSeconds       int
}

func NewHttpServer(host string, port int, readHeaderTimeoutInSeconds int, readTimeoutInSeconds int,
	writeTimeoutInSeconds int, idleTimeoutInSeconds int, telemetryConfig config.TelemetryConfig,
	paymentServerMux *runtime.ServeMux, rabbitmqClient *mq.RabbitMqConnection, mongoClient *db.Db) (*Server, error) {
	if err := validation.New().
		NotBlank("http.host", host).
		PositiveValue("http.port", port).
		PositiveValue("http.read_header_timeout_in_seconds", readHeaderTimeoutInSeconds).
		PositiveValue("http.read_timeout_in_seconds", readTimeoutInSeconds).
		PositiveValue("http.write_timeout_in_seconds", writeTimeoutInSeconds).
		PositiveValue("http.idle_timeout_in_seconds", idleTimeoutInSeconds).
		NotNil("payment_server_mux", paymentServerMux).
		Validate(); err != nil {
		return nil, err
	}

	return &Server{serverConfig: httpServerConfig{
		host:                       host,
		port:                       port,
		readHeaderTimeoutInSeconds: readHeaderTimeoutInSeconds,
		readTimeoutInSeconds:       readTimeoutInSeconds,
		writeTimeoutInSeconds:      writeTimeoutInSeconds,
		idleTimeoutInSeconds:       idleTimeoutInSeconds,
	}, telemetryConfig: telemetryConfig,
		paymentServerMux: paymentServerMux, rabbitmqClient: rabbitmqClient, mongoClient: mongoClient}, nil
}

func (s *Server) SetServer() {
	rootMux := http.NewServeMux()
	rootMux.HandleFunc("/healthz", probe.HealthHandler)
	rootMux.HandleFunc("/readyz", probe.ReadinessHandler(s.rabbitmqClient, s.mongoClient))
	rootMux.Handle("/", s.traceContextMiddleware(otelhttp.NewHandler(s.paymentServerMux, "payment-http-gateway")))

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", s.serverConfig.host, s.serverConfig.port),
		Handler:           rootMux,
		ReadHeaderTimeout: time.Duration(s.serverConfig.readHeaderTimeoutInSeconds) * time.Second,
		ReadTimeout:       time.Duration(s.serverConfig.readTimeoutInSeconds) * time.Second,
		WriteTimeout:      time.Duration(s.serverConfig.writeTimeoutInSeconds) * time.Second,
		IdleTimeout:       time.Duration(s.serverConfig.idleTimeoutInSeconds) * time.Second,
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
