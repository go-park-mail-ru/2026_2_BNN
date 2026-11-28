package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	authGrpc "bnn/internal/pkg/auth/delivery/grpc"
	authGen "bnn/internal/pkg/auth/delivery/grpc/gen"
	authGithub "bnn/internal/pkg/auth/repo/github"
	authRepo "bnn/internal/pkg/auth/repo/postgres"
	authFiles "bnn/internal/pkg/auth/repo/s3"
	authUsecase "bnn/internal/pkg/auth/usecase"
	"bnn/internal/pkg/database"
	"bnn/internal/pkg/logger"
	"bnn/internal/pkg/metrics"
	"bnn/internal/pkg/s3"
	"bnn/internal/pkg/utils/env"
	"bnn/internal/pkg/utils/grpcutil"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/lo"
)

const (
	grpcAddr          = ":9001"
	metricsAddr       = ":9102"
	shutdownTimeout   = 5 * time.Second
	defaultAppBaseURL = "http://localhost:5458"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		slog.Warn("No .env file found, using system environment variables", "error", err)
	}

	ctx := context.Background()
	log := logger.New()
	slog.SetDefault(log)

	db, err := database.NewPool(ctx, env.Must("DATABASE_URL"), log)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	minioClient, err := s3.Connect(s3.Config{
		Endpoint:  env.Must("S3_ENDPOINT"),
		AccessKey: env.Must("S3_ACCESS_KEY"),
		SecretKey: env.Must("S3_SECRET_KEY"),
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create S3 client", "error", err)
		os.Exit(1)
	}

	avatarStorage, err := authFiles.NewRepo(ctx, minioClient, env.Must("S3_AVATARS_BUCKET"))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create avatars storage", "error", err)
		os.Exit(1)
	}

	appBaseURL := lo.CoalesceOrEmpty(os.Getenv("APP_BASE_URL"), defaultAppBaseURL)
	githubRepo, err := authGithub.NewRepo(os.Getenv("GITHUB_CLIENT_ID"), os.Getenv("GITHUB_CLIENT_SECRET"), appBaseURL)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create GitHub repo", "error", err)
		os.Exit(1)
	}

	uc := authUsecase.NewUsecase(authRepo.NewRepo(db), avatarStorage, githubRepo, env.Must("JWT_SECRET"))

	grpcMetrics := metrics.NewGRPC(prometheus.DefaultRegisterer)
	server := grpcutil.NewServer(grpcMetrics)
	authGen.RegisterAuthServer(server, authGrpc.NewHandler(uc))

	metricsServer := metrics.NewServer(metricsAddr)
	go func() {
		slog.InfoContext(ctx, "Starting metrics server", "address", metricsAddr)
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.ErrorContext(ctx, "Metrics server failed", "error", err)
			os.Exit(1)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = metricsServer.Shutdown(shutdownCtx)
	}()

	if err := grpcutil.Serve(server, grpcAddr); err != nil {
		slog.ErrorContext(ctx, "gRPC server failed", "error", err)
		os.Exit(1)
	}
}
