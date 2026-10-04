package main

import (
	"log/slog"
	"net/http"
	"os"

	"bnn/internal/pkg/auth"
	"bnn/internal/pkg/note"
	"bnn/internal/pkg/user"

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

	auth.Init(jwtSecret)

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

		authRouter.HandleFunc("/signup", auth.SignUp).
			Methods(http.MethodPost)

		authRouter.HandleFunc("/signin", auth.SignIn).
			Methods(http.MethodPost)

		authRouter.HandleFunc("/logout", auth.Logout).
			Methods(http.MethodPost)
	}

	{
		notesRouter := api.PathPrefix("/notes").Subrouter()
		notesRouter.Use(auth.Middleware)

		notesRouter.HandleFunc("/getall", note.ListNotes).
			Methods(http.MethodGet)

		notesRouter.HandleFunc("/{id}", note.GetNote).
			Methods(http.MethodGet)
	}

	{
		usersRouter := api.PathPrefix("/users").Subrouter()
		usersRouter.Use(auth.Middleware)

		usersRouter.HandleFunc("/me", user.GetCurrentUser).
			Methods(http.MethodGet)
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
