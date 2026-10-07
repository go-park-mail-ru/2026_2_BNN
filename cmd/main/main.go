package main

import (
	"log/slog"
	"net/http"
	"os"

	"bnn/internal/pkg/auth"
	"bnn/internal/pkg/middleware"
	"bnn/internal/pkg/note"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, using system environment")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		slog.Error("JWT_SECRET is not set")
		os.Exit(1)
	}

	authService := auth.NewService(jwtSecret)
	noteHandler := note.NewHandler(note.DemoNotes())

	corsMiddleware, err := middleware.NewCORS(os.Getenv("FRONTEND_ORIGIN"))
	if err != nil {
		slog.Error("Failed to configure CORS", "error", err)
		os.Exit(1)
	}

	router := mux.NewRouter()
	api := router.PathPrefix("/api").Subrouter()

	authRouter := api.PathPrefix("/auth").Subrouter()
	{
		authRouter.HandleFunc("/signup", authService.SignUp).
			Methods(http.MethodPost)

		authRouter.HandleFunc("/signin", authService.SignIn).
			Methods(http.MethodPost)
	}

	authProtectedRouter := api.PathPrefix("/auth").Subrouter()
	authProtectedRouter.Use(authService.Middleware)
	{
		authProtectedRouter.HandleFunc("/me", authService.GetCurrentUser).
			Methods(http.MethodGet)

		authProtectedRouter.HandleFunc("/logout", authService.Logout).
			Methods(http.MethodPost)
	}

	notesRouter := api.PathPrefix("/notes").Subrouter()
	notesRouter.Use(authService.Middleware)
	{
		notesRouter.HandleFunc("/getall", noteHandler.ListNotes).
			Methods(http.MethodGet)

		notesRouter.HandleFunc("/{id}", noteHandler.GetNote).
			Methods(http.MethodGet)
	}

	server := http.Server{
		Addr:    ":5458",
		Handler: corsMiddleware(router),
	}

	slog.Info("Starting server", "address", "http://localhost:5458")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed", "error", err)
	}
}
