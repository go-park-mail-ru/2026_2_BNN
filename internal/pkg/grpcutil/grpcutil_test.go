package grpcutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewServer(t *testing.T) {
	assert.NotNil(t, NewServer())
}

func TestNewClient(t *testing.T) {
	conn, err := NewClient("localhost:9001")
	require.NoError(t, err)
	assert.NoError(t, conn.Close())
}

func TestServeListenError(t *testing.T) {
	err := Serve(NewServer(), "bad address")
	assert.Error(t, err)
}

func TestRecoveryInterceptor(t *testing.T) {
	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}

	resp, err := recoveryInterceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	_, err = recoveryInterceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
		panic("boom")
	})
	assert.Equal(t, codes.Internal, status.Code(err))
}
