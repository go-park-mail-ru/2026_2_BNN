package main

import (
	"bnn/internal/pkg/logger"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	attachmentsDelivery "bnn/internal/pkg/attachments/delivery/http"
	attachmentsapi "bnn/internal/pkg/attachments/delivery/http/api"
	attachmentsRepo "bnn/internal/pkg/attachments/repo/postgres"
	attachmentsFiles "bnn/internal/pkg/attachments/repo/s3"
	attachmentsUsecase "bnn/internal/pkg/attachments/usecase"
	"bnn/internal/pkg/database"
	"bnn/internal/pkg/swagger"

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

	authGithub "bnn/internal/pkg/auth/repo/github"
	tagsDelivery "bnn/internal/pkg/tags/delivery/http"
	tagsapi "bnn/internal/pkg/tags/delivery/http/api"
	tagsRepo "bnn/internal/pkg/tags/repo/postgres"
	tagsUsecase "bnn/internal/pkg/tags/usecase"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/samber/lo"
)

const (
	shutdownTimeout    = 5 * time.Second
	MaxRequestSize20MB = 20 << 20

	defaultAppBaseURL  = "http://localhost:5458"
	defaultFrontendURL = "http://localhost:3000"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		slog.Warn("No .env file found, using system environment variables", "error", err)
	}

	ctx := context.Background()

	baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	contextHandler := logger.ContextHandler{Handler: baseHandler}

	baseLogger := slog.New(contextHandler)
	slog.SetDefault(baseLogger)

	dbTracer := database.NewTracer(baseLogger)

	poolConfig, err := pgxpool.ParseConfig(mustEnv("DATABASE_URL"))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to parse DB config", "error", err)
		os.Exit(1)
	}

	poolConfig.ConnConfig.Tracer = dbTracer

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		slog.ErrorContext(ctx, "Failed to connect to database", "error", err)
		os.Exit(1)
	}

	minioClient, err := s3.Connect(s3.Config{
		Endpoint:  mustEnv("S3_ENDPOINT"),
		AccessKey: mustEnv("S3_ACCESS_KEY"),
		SecretKey: mustEnv("S3_SECRET_KEY"),
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create S3 client", "error", err)
		os.Exit(1)
	}

	avatarStorage, err := authFiles.NewRepo(ctx, minioClient, mustEnv("S3_AVATARS_BUCKET"))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create avatars storage", "error", err)
		os.Exit(1)
	}

	appBaseURL := lo.CoalesceOrEmpty(os.Getenv("APP_BASE_URL"), defaultAppBaseURL)
	frontendURL := lo.CoalesceOrEmpty(os.Getenv("FRONTEND_URL"), defaultFrontendURL)

	githubRepo := authGithub.NewRepo(os.Getenv("GITHUB_CLIENT_ID"), os.Getenv("GITHUB_CLIENT_SECRET"), appBaseURL)

	authHandler := authDelivery.NewHandler(
		authUsecase.NewUsecase(authRepo.NewRepo(db), avatarStorage, githubRepo, mustEnv("JWT_SECRET")),
		frontendURL,
	)

	notesHandler := notesDelivery.NewHandler(
		notesUsecase.NewUsecase(notesRepo.NewRepo(db)),
	)

	attachmentsStorage, err := attachmentsFiles.NewRepo(ctx, minioClient, mustEnv("S3_ATTACHMENTS_BUCKET"))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create attachments storage", "error", err)
		os.Exit(1)
	}

	attachmentsHandler := attachmentsDelivery.NewHandler(
		attachmentsUsecase.NewUsecase(attachmentsRepo.NewRepo(db), attachmentsStorage),
	)

	tagsHandler := tagsDelivery.NewHandler(
		tagsUsecase.NewUsecase(tagsRepo.NewRepo(db)),
	)

	corsMiddleware, err := middleware.NewCORS(os.Getenv("FRONTEND_ORIGIN"))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create CORS middleware", "error", err)
		os.Exit(1)
	}

	router := mux.NewRouter()
	router.Use(cookies.Middleware)
	router.Use(middleware.RequestID)
	router.Use(middleware.AccessLog(baseLogger))

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

	tagsapi.HandlerFromMux(tagsapi.NewStrictHandlerWithOptions(tagsHandler, nil, tagsapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  httpjson.RequestError,
		ResponseErrorHandlerFunc: httpjson.ResponseError,
	}), apiRouter)

	swagger.Register(router)

	server := http.Server{
		Addr:    ":5458",
		Handler: corsMiddleware(router),
	}

	go func() {
		slog.InfoContext(ctx, "Starting server", "address", "http://localhost:5458")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.ErrorContext(ctx, "Server failed", "error", err)
			os.Exit(1)
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-signalCh
	slog.InfoContext(ctx, "Received signal, shutting down", "signal", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.ErrorContext(ctx, "Server shutdown failed", "error", err)
		return
	}

	slog.InfoContext(ctx, "Server stopped")
}

func mustEnv(key string) string {
	ctx := context.Background()
	value := os.Getenv(key)
	if value == "" {
		slog.ErrorContext(ctx, "Environment variable is not set", "key", key)
		os.Exit(1)
	}

	return value
}
