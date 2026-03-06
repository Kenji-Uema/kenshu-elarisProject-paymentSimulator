package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/transport/grpc/clock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

type Clock struct {
	conn   *grpc.ClientConn
	client clock.ClockServiceClient
}

func NewClockEmu(cfg config.ClockEmuConfig) (*Clock, error) {
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", cfg.ClockEmuGrpcUrl, cfg.ClockEmuGrpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))

	if err != nil {
		return nil, err
	}

	return &Clock{conn: conn, client: clock.NewClockServiceClient(conn)}, nil
}

func (e *Clock) Close() error {
	return e.conn.Close()
}

func (e *Clock) Now(ctx context.Context) (*time.Time, error) {
	createTime, err := e.client.Now(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	createdTimestamp := createTime.Time.AsTime()

	return &createdTimestamp, nil
}
