package grpctest

import (
	"context"
	"net"
	"testing"

	"bnn/internal/pkg/grpcutil"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufferSize = 1 << 20

func Dial(t *testing.T, register func(server *grpc.Server)) *grpc.ClientConn {
	listener := bufconn.Listen(bufferSize)
	server := grpcutil.NewServer()
	register(server)

	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return conn
}
