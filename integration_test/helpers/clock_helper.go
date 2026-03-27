package helpers

import (
	"context"
	"net"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/paymentSimulator/internal/infra/clock"
	. "github.com/onsi/ginkgo/v2"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeClockServer struct {
	clock.UnimplementedClockServiceServer
}

func (fakeClockServer) Now(context.Context, *emptypb.Empty) (*dto.TimeEvent, error) {
	return &dto.TimeEvent{Time: timestamppb.Now()}, nil
}

func StartClockEmulator(t FullGinkgoTInterface) (clockHost string, clockPort int) {
	clockHost, clockPort, stopClock := startFakeClockServer(t)
	DeferCleanup(stopClock)

	return
}

func startFakeClockServer(t TestReporter) (host string, port int, stop func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen fake clock: %v", err)
	}
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected addr type: %T", ln.Addr())
	}

	grpcServer := grpc.NewServer()
	clock.RegisterClockServiceServer(grpcServer, fakeClockServer{})
	go func() {
		_ = grpcServer.Serve(ln)
	}()

	return "127.0.0.1", tcpAddr.Port, func() {
		grpcServer.GracefulStop()
		_ = ln.Close()
	}
}
