package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	attachmentsGrpc "bnn/internal/pkg/attachments/delivery/grpc"
	attachmentsGen "bnn/internal/pkg/attachments/delivery/grpc/gen"
	attachmentsRepo "bnn/internal/pkg/attachments/repo/postgres"
	attachmentsFiles "bnn/internal/pkg/attachments/repo/s3"
	attachmentsUsecase "bnn/internal/pkg/attachments/usecase"
	authClient "bnn/internal/pkg/auth/client"
	"bnn/internal/pkg/database"
	"bnn/internal/pkg/env"
	"bnn/internal/pkg/grpcutil"
	"bnn/internal/pkg/logger"
	"bnn/internal/pkg/metrics"
	notesGrpc "bnn/internal/pkg/notes/delivery/grpc"
	notesGen "bnn/internal/pkg/notes/delivery/grpc/gen"
	notesRepo "bnn/internal/pkg/notes/repo/postgres"
	notesUsecase "bnn/internal/pkg/notes/usecase"
	"bnn/internal/pkg/s3"
	tagsGrpc "bnn/internal/pkg/tags/delivery/grpc"
	tagsGen "bnn/internal/pkg/tags/delivery/grpc/gen"
	tagsRepo "bnn/internal/pkg/tags/repo/postgres"
	tagsUsecase "bnn/internal/pkg/tags/usecase"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/lo"
)

const (
	grpcAddr        = ":9002"
	metricsAddr     = ":9103"
	shutdownTimeout = 5 * time.Second
	defaultAuthAddr = "localhost:9001"
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

	authAddr := lo.CoalesceOrEmpty(os.Getenv("AUTH_GRPC_ADDR"), defaultAuthAddr)
	authConn, err := grpcutil.NewClient(authAddr)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to connect to auth service", "error", err)
		os.Exit(1)
	}
	defer authConn.Close()

	authService := authClient.NewClient(authConn)

	minioClient, err := s3.Connect(s3.Config{
		Endpoint:  env.Must("S3_ENDPOINT"),
		AccessKey: env.Must("S3_ACCESS_KEY"),
		SecretKey: env.Must("S3_SECRET_KEY"),
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create S3 client", "error", err)
		os.Exit(1)
	}

	attachmentsStorage, err := attachmentsFiles.NewRepo(ctx, minioClient, env.Must("S3_ATTACHMENTS_BUCKET"))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create attachments storage", "error", err)
		os.Exit(1)
	}

	grpcMetrics := metrics.NewGRPC(prometheus.DefaultRegisterer)
	server := grpcutil.NewServer(grpcMetrics)

	notesUC := notesUsecase.NewUsecase(notesRepo.NewRepo(db), authService)
	notesGen.RegisterNotesServer(server, notesGrpc.NewHandler(notesUC))

	attachmentsUC := attachmentsUsecase.NewUsecase(attachmentsRepo.NewRepo(db), attachmentsStorage, notesUC)
	attachmentsGen.RegisterAttachmentsServer(server, attachmentsGrpc.NewHandler(attachmentsUC))

	tagsGen.RegisterTagsServer(server, tagsGrpc.NewHandler(
		tagsUsecase.NewUsecase(tagsRepo.NewRepo(db)),
	))

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
