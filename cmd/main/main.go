package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bnn/docs"
	attachmentsDelivery "bnn/internal/pkg/attachments/delivery/http"
	attachmentsapi "bnn/internal/pkg/attachments/delivery/http/api"
	attachmentsRepo "bnn/internal/pkg/attachments/repo/postgres"
	attachmentsFiles "bnn/internal/pkg/attachments/repo/s3"
	attachmentsUsecase "bnn/internal/pkg/attachments/usecase"

	authDelivery "bnn/internal/pkg/auth/delivery/http"
	authapi "bnn/internal/pkg/auth/delivery/http/api"
	authRepo "bnn/internal/pkg/auth/repo/postgres"
	authFiles "bnn/internal/pkg/auth/repo/s3"
	authUsecase "bnn/internal/pkg/auth/usecase"
	"bnn/internal/pkg/cookies"
	"bnn/internal/pkg/httpjson"
	"bnn/internal/pkg/middleware"
	notesDelivery "bnn/internal/pkg/notes/delivery/http"
	notesapi "bnn/internal/pkg/notes/delivery/http/api"
	notesRepo "bnn/internal/pkg/notes/repo/postgres"
	notesUsecase "bnn/internal/pkg/notes/usecase"
	"bnn/internal/pkg/s3"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

const (
	shutdownTimeout    = 5 * time.Second
	MaxRequestSize20MB = 20 << 20
)

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

	minioClient, err := s3.Connect(s3.Config{
		Endpoint:  mustEnv("S3_ENDPOINT"),
		AccessKey: mustEnv("S3_ACCESS_KEY"),
		SecretKey: mustEnv("S3_SECRET_KEY"),
	})
	if err != nil {
		slog.Error("Failed to create S3 client", "error", err)
		os.Exit(1)
	}

	avatarStorage, err := authFiles.NewRepo(ctx, minioClient, mustEnv("S3_AVATARS_BUCKET"))
	if err != nil {
		slog.Error("Failed to create avatars storage", "error", err)
		os.Exit(1)
	}

	authHandler := authDelivery.NewHandler(
		authUsecase.NewUsecase(authRepo.NewRepo(db), avatarStorage, mustEnv("JWT_SECRET")),
	)

	notesHandler := notesDelivery.NewHandler(
		notesUsecase.NewUsecase(notesRepo.NewRepo(db)),
	)

	attachmentsStorage, err := attachmentsFiles.NewRepo(ctx, minioClient, mustEnv("S3_ATTACHMENTS_BUCKET"))
	if err != nil {
		slog.Error("Failed to create attachments storage", "error", err)
		os.Exit(1)
	}

	attachmentsHandler := attachmentsDelivery.NewHandler(
		attachmentsUsecase.NewUsecase(attachmentsRepo.NewRepo(db), attachmentsStorage),
	)

	corsMiddleware, err := middleware.NewCORS(os.Getenv("FRONTEND_ORIGIN"))
	if err != nil {
		slog.Error("Failed to create CORS middleware", "error", err)
		os.Exit(1)
	}

	router := mux.NewRouter()
	router.Use(cookies.Middleware)

	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.Use(authHandler.Middleware)
	apiRouter.Use(middleware.MaxBytesMiddleware(MaxRequestSize20MB))

	authapi.HandlerFromMux(authapi.NewStrictHandlerWithOptions(authHandler, nil, authapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  httpjson.RequestError,
		ResponseErrorHandlerFunc: httpjson.ResponseError,
	}), apiRouter)

	notesapi.HandlerFromMux(notesapi.NewStrictHandlerWithOptions(notesHandler, nil, notesapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  httpjson.RequestError,
		ResponseErrorHandlerFunc: httpjson.ResponseError,
	}), apiRouter)

	attachmentsapi.HandlerFromMux(attachmentsapi.NewStrictHandlerWithOptions(attachmentsHandler, nil, attachmentsapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  httpjson.RequestError,
		ResponseErrorHandlerFunc: httpjson.ResponseError,
	}), apiRouter)

	router.HandleFunc("/swagger/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		if _, err := w.Write(docs.OpenAPI); err != nil {
			slog.Error("Failed to write spec", "error", err)
		}
	}).Methods(http.MethodGet)

	router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger/openapi.yaml"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
	)).Methods(http.MethodGet)

	server := http.Server{
		Addr:    ":5458",
		Handler: corsMiddleware(router),
	}

	go func() {
		slog.Info("Starting server", "address", "http://localhost:5458")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-signalCh
	slog.Info("Received signal, shutting down", "signal", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
		return
	}

	slog.Info("Server stopped")
}

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		slog.Error("Environment variable is not set", "key", key)
		os.Exit(1)
	}

	return value
}
