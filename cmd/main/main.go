package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	authdelivery "bnn/internal/pkg/auth/delivery/http"
	"bnn/internal/pkg/auth/repo/postgres"
	"bnn/internal/pkg/auth/usecase"
	notesdelivery "bnn/internal/pkg/notes/delivery/http"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	jwtSecret := os.Getenv("JWT_SECRET")
	dsn := os.Getenv("DATABASE_URL")
	if jwtSecret == "" || dsn == "" {
		slog.Error("JWT_SECRET or DATABASE_URL is not set")
		os.Exit(1)
	}

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		slog.Error("Failed to connect to DB", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	authRepo := postgres.NewRepo(db)
	authUC := usecase.NewUsecase(authRepo, jwtSecret)
	authHandler := authdelivery.NewHandler(authUC)

	// notes — точные имена уточним по notes/usecase
	//notesUC := notesusecase.NewUsecase( /* repo */ )
	notesHandler := notesdelivery.NewHandler(notesdelivery.DemoNotes())

	router := mux.NewRouter()
	api := router.PathPrefix("/api").Subrouter()

	api.HandleFunc("/auth/signup", authHandler.SignUp).Methods(http.MethodPost)
	api.HandleFunc("/auth/signin", authHandler.SignIn).Methods(http.MethodPost)
	api.HandleFunc("/auth/refresh", authHandler.Refresh).Methods(http.MethodPost)
	api.HandleFunc("/auth/logout", authHandler.Logout).Methods(http.MethodPost)

	protected := api.NewRoute().Subrouter()
	protected.Use(authHandler.Middleware)

	protected.HandleFunc("/auth/me", authHandler.GetCurrentUser).Methods(http.MethodGet)
	protected.HandleFunc("/notes/getall", notesHandler.ListNotes).Methods(http.MethodGet)
	protected.HandleFunc("/notes/{id}", notesHandler.GetNote).Methods(http.MethodGet)

	server := http.Server{
		Addr:    ":5458",
		Handler: router,
	}

	slog.Info("Starting server", "address", "http://localhost:5458")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed", "error", err)
	}
}
