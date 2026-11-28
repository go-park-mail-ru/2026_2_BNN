package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	authClient "bnn/internal/pkg/auth/client"
	"bnn/internal/pkg/database"
	"bnn/internal/pkg/logger"
	onlineconfGrpc "bnn/internal/pkg/onlineconf/delivery/grpc"
	onlineconfGen "bnn/internal/pkg/onlineconf/delivery/grpc/gen"
	onlineconfRepo "bnn/internal/pkg/onlineconf/repo/postgres"
	onlineconfUsecase "bnn/internal/pkg/onlineconf/usecase"
	"bnn/internal/pkg/utils/grpcutil"

	"github.com/joho/godotenv"
)

const (
	grpcAddr        = ":9004"
	defaultAuthAddr = "localhost:9001"
	startupTimeout  = 10 * time.Second
)

func main() {
	slog.SetDefault(logger.New())

	if err := godotenv.Load(".env"); err != nil {
		slog.Warn(
			"No .env file found, using system environment variables",
			"error", err,
		)
	}

	if err := run(); err != nil {
		slog.Error("OnlineConf service failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}

	ctx := context.Background()

	startupCtx, cancelStartup := context.WithTimeout(
		ctx, startupTimeout,
	)

	db, err := database.NewPool(
		startupCtx, databaseURL, slog.Default(),
	)
	cancelStartup()

	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()

	authAddr := os.Getenv("AUTH_GRPC_ADDR")
	if authAddr == "" {
		authAddr = defaultAuthAddr
	}

	authConn, err := grpcutil.NewClient(authAddr)
	if err != nil {
		return fmt.Errorf("create auth connection: %w", err)
	}
	defer authConn.Close()

	authService := authClient.NewClient(authConn)

	repo := onlineconfRepo.NewRepo(db)
	uc := onlineconfUsecase.NewUsecase(repo, authService)

	refreshCtx, cancelRefresh := context.WithCancel(ctx)
	refreshDone := make(chan struct{})
	rolloutDone := make(chan struct{})
	go func() {
		defer close(rolloutDone)
		uc.RunRollouts(refreshCtx)
	}()

	go func() {
		defer close(refreshDone)
		uc.Run(refreshCtx)
	}()

	defer func() {
		cancelRefresh()
		<-refreshDone
		<-rolloutDone
	}()

	server := grpcutil.NewServer()
	defer server.Stop()

	onlineconfGen.RegisterOnlineConfServer(
		server,
		onlineconfGrpc.NewHandler(uc),
	)

	if err := grpcutil.Serve(server, grpcAddr); err != nil {
		return fmt.Errorf("serve onlineconf grpc: %w", err)
	}

	return nil
}
