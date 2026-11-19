package grpcutil

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const maxMessageSize = 20 << 20

func NewServer() *grpc.Server {
	return grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMessageSize),
		grpc.MaxSendMsgSize(maxMessageSize),
		grpc.UnaryInterceptor(recoveryInterceptor),
	)
}

func NewClient(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMessageSize),
			grpc.MaxCallSendMsgSize(maxMessageSize),
		),
	)
}

func Serve(server *grpc.Server, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("Starting gRPC server", "address", addr)
		errCh <- server.Serve(listener)
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("serve grpc: %w", err)
	case sig := <-signalCh:
		slog.Info("Received signal, shutting down", "signal", sig)
		server.GracefulStop()
		return nil
	}
}

func recoveryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.ErrorContext(ctx, "Panic recovered", "method", info.FullMethod, "error", rec, "stack", string(debug.Stack()))
			err = status.Error(codes.Internal, "internal error")
		}
	}()

	return handler(ctx, req)
}
