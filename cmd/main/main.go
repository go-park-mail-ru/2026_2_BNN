package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

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

	frontendOrigin := os.Getenv("FRONTEND_ORIGIN")
	if frontendOrigin == "" {
		slog.Error("FRONTEND_ORIGIN is not set")
		os.Exit(1)
	}

	allowedOrigins := make([]string, 0, 2)

	for _, origin := range strings.Split(frontendOrigin, ",") {
		origin = strings.TrimSpace(origin)
		origin = strings.TrimSuffix(origin, "/")

		if origin != "" {
			allowedOrigins = append(allowedOrigins, origin)
		}
	}

	if len(allowedOrigins) == 0 {
		slog.Error("FRONTEND_ORIGIN contains no valid origins")
		os.Exit(1)
	}

	slog.Info("CORS configured", "allowed_origins", allowedOrigins)

	router := mux.NewRouter()

	router.PathPrefix("/static/").Handler(
		http.StripPrefix(
			"/static/",
			http.FileServer(http.Dir("static")),
		),
	)

	api := router.PathPrefix("/api").Subrouter()

	{
		authRouter := api.PathPrefix("/auth").Subrouter()

		authRouter.HandleFunc("/signup", authService.SignUp).
			Methods(http.MethodPost)

		authRouter.HandleFunc("/signin", authService.SignIn).
			Methods(http.MethodPost)

	}
	{
		authProtected := api.PathPrefix("/auth").Subrouter()
		authProtected.Use(authService.Middleware)

		authProtected.HandleFunc("/me", authService.GetCurrentUser).
			Methods(http.MethodGet)

		authProtected.HandleFunc("/logout", authService.Logout).
			Methods(http.MethodPost)

	}

	{
		notesRouter := api.PathPrefix("/notes").Subrouter()
		notesRouter.Use(authService.Middleware)

		notesRouter.HandleFunc("/getall", noteHandler.ListNotes).
			Methods(http.MethodGet)

		notesRouter.HandleFunc("/{id}", noteHandler.GetNote).
			Methods(http.MethodGet)
	}

	corsMiddleware := middleware.CORS(allowedOrigins)

	server := http.Server{
		Addr:    ":5458",
		Handler: corsMiddleware(router),
	}

	slog.Info("Starting server", "address", "http://localhost:5458")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed", "error", err)
	}
}
