package main

import (
	"context"
	"log/slog"
	"os"

	attachmentsGrpc "bnn/internal/pkg/attachments/delivery/grpc"
	attachmentsGen "bnn/internal/pkg/attachments/delivery/grpc/gen"
	attachmentsRepo "bnn/internal/pkg/attachments/repo/postgres"
	attachmentsFiles "bnn/internal/pkg/attachments/repo/s3"
	attachmentsUsecase "bnn/internal/pkg/attachments/usecase"
	"bnn/internal/pkg/database"
	"bnn/internal/pkg/env"
	"bnn/internal/pkg/grpcutil"
	"bnn/internal/pkg/logger"
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
)

const grpcAddr = ":9002"

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

	attachmentsStorage, err := attachmentsFiles.NewRepo(ctx, minioClient, env.Must("S3_ATTACHMENTS_BUCKET"))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create attachments storage", "error", err)
		os.Exit(1)
	}

	server := grpcutil.NewServer()
	notesGen.RegisterNotesServer(server, notesGrpc.NewHandler(
		notesUsecase.NewUsecase(notesRepo.NewRepo(db)),
	))
	tagsGen.RegisterTagsServer(server, tagsGrpc.NewHandler(
		tagsUsecase.NewUsecase(tagsRepo.NewRepo(db)),
	))
	attachmentsGen.RegisterAttachmentsServer(server, attachmentsGrpc.NewHandler(
		attachmentsUsecase.NewUsecase(attachmentsRepo.NewRepo(db), attachmentsStorage),
	))

	if err := grpcutil.Serve(server, grpcAddr); err != nil {
		slog.ErrorContext(ctx, "gRPC server failed", "error", err)
		os.Exit(1)
	}
}
