package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	authDelivery "bnn/internal/pkg/auth/delivery/http"
	authRepo "bnn/internal/pkg/auth/repo/postgres"
	authUsecase "bnn/internal/pkg/auth/usecase"
	"bnn/internal/pkg/middleware"
	notesDelivery "bnn/internal/pkg/notes/delivery/http"
	notesRepo "bnn/internal/pkg/notes/repo/postgres"
	notesUsecase "bnn/internal/pkg/notes/usecase"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var buckets = []string{"avatars", "attachments"}

func main() {
	_ = godotenv.Load()

	ctx := context.Background()

	db, err := pgxpool.New(ctx, mustEnv("DATABASE_URL"))
	if err != nil {
		slog.Error("Failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	s3, err := minio.New(mustEnv("S3_ENDPOINT"), &minio.Options{
		Creds: credentials.NewStaticV4(mustEnv("S3_ACCESS_KEY"), mustEnv("S3_SECRET_KEY"), ""),
	})
	if err != nil {
		slog.Error("Failed to create S3 client", "error", err)
		os.Exit(1)
	}

	if err := createBuckets(ctx, s3); err != nil {
		slog.Error("Failed to create S3 buckets", "error", err)
		os.Exit(1)
	}

	authHandler := authDelivery.NewHandler(
		authUsecase.NewUsecase(authRepo.NewRepo(db), mustEnv("JWT_SECRET")),
	)
	notesHandler := notesDelivery.NewHandler(
		notesUsecase.NewUsecase(notesRepo.NewRepo(db)),
	)

	corsMiddleware, err := middleware.NewCORS(os.Getenv("FRONTEND_ORIGIN"))
	if err != nil {
		slog.Error("Failed to create CORS middleware", "error", err)
		os.Exit(1)
	}

	router := mux.NewRouter()
	router.Use(corsMiddleware)

	api := router.PathPrefix("/api").Subrouter()

	authRouter := api.PathPrefix("/auth").Subrouter()
	{
		authRouter.HandleFunc("/signup", authHandler.SignUp).Methods(http.MethodPost)
		authRouter.HandleFunc("/signin", authHandler.SignIn).Methods(http.MethodPost)
		authRouter.HandleFunc("/refresh", authHandler.Refresh).Methods(http.MethodPost)
		authRouter.HandleFunc("/logout", authHandler.Logout).Methods(http.MethodPost)
	}

	authProtectedRouter := api.PathPrefix("/auth").Subrouter()
	authProtectedRouter.Use(authHandler.Middleware)
	{
		authProtectedRouter.HandleFunc("/me", authHandler.GetCurrentUser).Methods(http.MethodGet)
	}

	notesRouter := api.PathPrefix("/notes").Subrouter()
	notesRouter.Use(authHandler.Middleware)
	{
		notesRouter.HandleFunc("/getall", notesHandler.ListNotes).Methods(http.MethodGet)
		notesRouter.HandleFunc("/{id}", notesHandler.GetNote).Methods(http.MethodGet)
	}

	server := http.Server{
		Addr:    ":5458",
		Handler: router,
	}

	slog.Info("Starting server", "address", "http://localhost:5458")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed", "error", err)
	}
}

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		slog.Error("Environment variable is not set", "key", key)
		os.Exit(1)
	}

	return value
}

func createBuckets(ctx context.Context, s3 *minio.Client) error {
	for _, bucket := range buckets {
		exists, err := s3.BucketExists(ctx, bucket)
		if err != nil {
			return fmt.Errorf("check bucket %s: %w", bucket, err)
		}

		if exists {
			continue
		}

		if err := s3.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("create bucket %s: %w", bucket, err)
		}
	}

	return nil
}
