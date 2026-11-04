package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	attachmentsDelivery "bnn/internal/pkg/attachments/delivery/http"
	attachmentsapi "bnn/internal/pkg/attachments/delivery/http/api"
	attachmentsRepo "bnn/internal/pkg/attachments/repo/postgres"
	attachmentsFiles "bnn/internal/pkg/attachments/repo/s3"
	attachmentsUsecase "bnn/internal/pkg/attachments/usecase"
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
	githubCallbackPath = "/api/auth/github/callback"
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

	appBaseURL := lo.CoalesceOrEmpty(os.Getenv("APP_BASE_URL"), defaultAppBaseURL)
	frontendURL := lo.CoalesceOrEmpty(os.Getenv("FRONTEND_URL"), defaultFrontendURL)

	githubCallbackURL, err := url.JoinPath(appBaseURL, githubCallbackPath)
	if err != nil {
		slog.Error("Failed to build GitHub callback URL", "error", err)
		os.Exit(1)
	}

	githubRepo := authGithub.NewRepo(os.Getenv("GITHUB_CLIENT_ID"), os.Getenv("GITHUB_CLIENT_SECRET"), githubCallbackURL)

	authHandler := authDelivery.NewHandler(
		authUsecase.NewUsecase(authRepo.NewRepo(db), avatarStorage, githubRepo, mustEnv("JWT_SECRET")),
		frontendURL,
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

	tagsHandler := tagsDelivery.NewHandler(
		tagsUsecase.NewUsecase(tagsRepo.NewRepo(db)),
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
