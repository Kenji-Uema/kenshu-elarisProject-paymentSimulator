package clock

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Clock struct {
	conn   *grpc.ClientConn
	client ClockServiceClient
}

func NewClock(cfg config.ClockEmuConfig) (port.Clock, error) {
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", cfg.ClockEmuGrpcUrl, cfg.ClockEmuGrpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))

	if err != nil {
		return nil, err
	}

	return &Clock{conn: conn, client: NewClockServiceClient(conn)}, nil
}

func (e *Clock) Now(ctx context.Context) (*time.Time, error) {
	createTime, err := e.client.Now(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	createdTimestamp := createTime.Time.AsTime()

	return &createdTimestamp, nil
}

func (e *Clock) Close() error {
	return e.conn.Close()
}
