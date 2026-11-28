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
	"bnn/internal/pkg/logger"
	bnnmcp "bnn/internal/pkg/mcp"
	"bnn/internal/pkg/metrics"
	"bnn/internal/pkg/middleware"
	notesClient "bnn/internal/pkg/notes/client"
	notesDelivery "bnn/internal/pkg/notes/delivery/http"
	notesapi "bnn/internal/pkg/notes/delivery/http/api"
	"bnn/internal/pkg/onlineconf"
	onlineconfClient "bnn/internal/pkg/onlineconf/client"
	onlineconfDelivery "bnn/internal/pkg/onlineconf/delivery/http"
	onlineconfapi "bnn/internal/pkg/onlineconf/delivery/http/api"
	"bnn/internal/pkg/onlineconf/delivery/http/runtimeapi"
	"bnn/internal/pkg/swagger"
	tagsClient "bnn/internal/pkg/tags/client"
	tagsDelivery "bnn/internal/pkg/tags/delivery/http"
	tagsapi "bnn/internal/pkg/tags/delivery/http/api"
	"bnn/internal/pkg/utils/cookies"
	"bnn/internal/pkg/utils/grpcutil"
	"bnn/internal/pkg/utils/httpjson"
	ws "bnn/internal/pkg/websocket"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/lo"
)

const (
	shutdownTimeout    = 5 * time.Second
	MaxRequestSize20MB = 20 << 20

	defaultFrontendURL    = "http://localhost:3000"
	defaultAuthAddr       = "localhost:9001"
	defaultNotesAddr      = "localhost:9002"
	metricsAddr           = ":9101"
	defaultOnlineConfAddr = "localhost:9004"
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

	onlineconfConn, err := grpcutil.NewClient(
		lo.CoalesceOrEmpty(
			os.Getenv("ONLINECONF_GRPC_ADDR"),
			defaultOnlineConfAddr,
		),
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create OnlineConf client", "error", err)
		os.Exit(1)
	}
	defer onlineconfConn.Close()

	onlineconfService := onlineconfClient.NewClient(onlineconfConn)

	authService := authClient.NewClient(authConn)
	notesService := notesClient.NewClient(notesConn)
	tagsService := tagsClient.NewClient(notesConn)
	attachmentsService := attachmentsClient.NewClient(notesConn)

	frontendURL := lo.CoalesceOrEmpty(os.Getenv("FRONTEND_URL"), defaultFrontendURL)

	authHandler := authDelivery.NewHandler(authService, frontendURL)
	notesHandler := notesDelivery.NewHandler(notesService)
	tagsHandler := tagsDelivery.NewHandler(tagsService)

	corsMiddleware, err := middleware.NewCORS(os.Getenv("FRONTEND_ORIGIN"))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create CORS middleware", "error", err)
		os.Exit(1)
	}

	wsHub := ws.NewHub(notesService)
	wsCtx, wsCancel := context.WithCancel(context.Background())
	defer wsCancel()
	go wsHub.Run(wsCtx)

	onlineconfCtx, onlineconfCancel := context.WithCancel(context.Background())
	defer onlineconfCancel()

	adminCache := onlineconf.NewAdminCache(authService)
	go adminCache.Run(onlineconfCtx)

	snapshotCache := onlineconf.NewSnapshotCache(onlineconfService)
	go snapshotCache.Run(onlineconfCtx)

	onlineconfHandler := onlineconfDelivery.NewHandler(
		authService, adminCache, onlineconfService,
	)
	runtimeHandler := onlineconfDelivery.NewRuntimeHandler(snapshotCache)
	attachmentsHandler := attachmentsDelivery.NewHandler(attachmentsService, snapshotCache)

	wsHandler := ws.NewHandler(wsHub, os.Getenv("FRONTEND_ORIGIN"), notesService)

	httpMetrics := metrics.NewHTTP(prometheus.DefaultRegisterer)

	router := mux.NewRouter()
	router.Use(cookies.Middleware)
	router.Use(middleware.RequestID)
	router.Use(middleware.AccessLog(baseLogger))
	router.Use(middleware.Metrics(httpMetrics))
	router.Use(middleware.Recovery)
	router.Use(onlineconfDelivery.NoStore)

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

	runtimeRouter := apiRouter.NewRoute().Subrouter()
	runtimeapi.HandlerFromMux(runtimeapi.NewStrictHandlerWithOptions(runtimeHandler, nil, runtimeapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  httpjson.RequestError,
		ResponseErrorHandlerFunc: httpjson.ResponseError,
	}), runtimeRouter)

	adminRouter := apiRouter.NewRoute().Subrouter()
	adminRouter.Use(onlineconfDelivery.AdminOnly(adminCache))

	strictOnlineconfHandler := onlineconfapi.NewStrictHandlerWithOptions(
		onlineconfHandler,
		nil,
		onlineconfapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  onlineconfDelivery.RequestError,
			ResponseErrorHandlerFunc: onlineconfDelivery.ResponseError,
		},
	)

	onlineconfapi.HandlerWithOptions(
		strictOnlineconfHandler,
		onlineconfapi.GorillaServerOptions{
			BaseRouter:       adminRouter,
			ErrorHandlerFunc: onlineconfDelivery.RequestError,
		},
	)

	router.PathPrefix("/mcp").Handler(bnnmcp.NewHandler(authService, notesService))

	swagger.Register(router)

	server := http.Server{
		Addr:    ":5458",
		Handler: corsMiddleware(router),
	}

	metricsServer := metrics.NewServer(metricsAddr)

	go func() {
		slog.InfoContext(ctx, "Starting metrics server", "address", metricsAddr)
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.ErrorContext(ctx, "Metrics server failed", "error", err)
			os.Exit(1)
		}
	}()

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

	onlineconfCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.ErrorContext(ctx, "Server shutdown failed", "error", err)
		return
	}

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		slog.ErrorContext(ctx, "Metrics server shutdown failed", "error", err)
		return
	}

	slog.InfoContext(ctx, "Server stopped")
}
