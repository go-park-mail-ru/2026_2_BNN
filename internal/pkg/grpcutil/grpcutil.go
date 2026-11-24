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
	"time"

	"bnn/internal/pkg/metrics"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const maxMessageSize = 20 << 20

func NewServer(grpcMetrics ...metrics.GRPC) *grpc.Server {
	interceptors := []grpc.UnaryServerInterceptor{
		loggingInterceptor,
	}

	if len(grpcMetrics) > 0 && grpcMetrics[0] != nil {
		interceptors = append(interceptors, metricsInterceptor(grpcMetrics[0]))
	}

	interceptors = append(interceptors, recoveryInterceptor)

	return grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMessageSize),
		grpc.MaxSendMsgSize(maxMessageSize),
		grpc.ChainUnaryInterceptor(interceptors...),
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

func loggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)
	code := status.Code(err)

	if err != nil {
		slog.WarnContext(ctx, "gRPC request failed",
			"method", info.FullMethod,
			"code", code.String(),
			"duration", duration,
			"error", err,
		)
	} else {
		slog.InfoContext(ctx, "gRPC request handled",
			"method", info.FullMethod,
			"code", code.String(),
			"duration", duration,
		)
	}

	return resp, err
}

func metricsInterceptor(m metrics.GRPC) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		m.IncreaseHits(info.FullMethod)
		m.ObserveDuration(info.FullMethod, duration)
		if err != nil {
			m.IncreaseErr(info.FullMethod, status.Code(err).String())
		}

		return resp, err
	}
}
