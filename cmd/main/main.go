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

	attachmentsClient "bnn/internal/pkg/attachments/client"
	attachmentsDelivery "bnn/internal/pkg/attachments/delivery/http"
	attachmentsapi "bnn/internal/pkg/attachments/delivery/http/api"
	authClient "bnn/internal/pkg/auth/client"
	authDelivery "bnn/internal/pkg/auth/delivery/http"
	authapi "bnn/internal/pkg/auth/delivery/http/api"
	"bnn/internal/pkg/cookies"
	"bnn/internal/pkg/database"
	"bnn/internal/pkg/env"
	"bnn/internal/pkg/grpcutil"
	"bnn/internal/pkg/httpjson"
	"bnn/internal/pkg/logger"
	"bnn/internal/pkg/middleware"
	notesClient "bnn/internal/pkg/notes/client"
	notesDelivery "bnn/internal/pkg/notes/delivery/http"
	notesapi "bnn/internal/pkg/notes/delivery/http/api"
	"bnn/internal/pkg/swagger"
	tagsClient "bnn/internal/pkg/tags/client"
	tagsDelivery "bnn/internal/pkg/tags/delivery/http"
	tagsapi "bnn/internal/pkg/tags/delivery/http/api"
	ws "bnn/internal/pkg/websocket"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/samber/lo"
)

const (
	shutdownTimeout    = 5 * time.Second
	MaxRequestSize20MB = 20 << 20

	defaultFrontendURL = "http://localhost:3000"
	defaultAuthAddr    = "localhost:9001"
	defaultNotesAddr   = "localhost:9002"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		slog.Warn("No .env file found, using system environment variables", "error", err)
	}

	ctx := context.Background()
	baseLogger := logger.New()
	slog.SetDefault(baseLogger)

	authConn, err := grpcutil.NewClient(lo.CoalesceOrEmpty(os.Getenv("AUTH_GRPC_ADDR"), defaultAuthAddr))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create auth client", "error", err)
		os.Exit(1)
	}
	defer authConn.Close()

	notesConn, err := grpcutil.NewClient(lo.CoalesceOrEmpty(os.Getenv("NOTES_GRPC_ADDR"), defaultNotesAddr))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create notes client", "error", err)
		os.Exit(1)
	}
	defer notesConn.Close()

	authService := authClient.NewClient(authConn)
	frontendURL := lo.CoalesceOrEmpty(os.Getenv("FRONTEND_URL"), defaultFrontendURL)

	authHandler := authDelivery.NewHandler(authService, frontendURL)
	notesHandler := notesDelivery.NewHandler(notesClient.NewClient(notesConn))
	tagsHandler := tagsDelivery.NewHandler(tagsClient.NewClient(notesConn))
	attachmentsHandler := attachmentsDelivery.NewHandler(attachmentsClient.NewClient(notesConn))

	corsMiddleware, err := middleware.NewCORS(os.Getenv("FRONTEND_ORIGIN"))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create CORS middleware", "error", err)
		os.Exit(1)
	}

	// WebSocket читает блоки напрямую из БД, сделать для gRPC сервиса notes
	db, err := database.NewPool(ctx, env.Must("DATABASE_URL"), baseLogger)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	wsHub := ws.NewHub(ws.NewPostgresRepo(db))
	wsCtx, wsCancel := context.WithCancel(context.Background())
	defer wsCancel()
	go wsHub.Run(wsCtx)

	wsHandler := ws.NewHandler(wsHub, os.Getenv("FRONTEND_ORIGIN"), authService)

	router := mux.NewRouter()
	router.Use(cookies.Middleware)
	router.Use(middleware.RequestID)
	router.Use(middleware.AccessLog(baseLogger))
	router.Use(middleware.Recovery)

	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.Use(authHandler.Middleware)
	apiRouter.Use(middleware.MaxBytesMiddleware(MaxRequestSize20MB))

	apiRouter.HandleFunc("/ws/notes/{noteID}", wsHandler.ServeNoteWS).Methods(http.MethodGet)

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

	wsCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.ErrorContext(ctx, "Server shutdown failed", "error", err)
		return
	}

	slog.InfoContext(ctx, "Server stopped")
}
