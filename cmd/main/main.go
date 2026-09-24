package main

import (
	"os"
	"log/slog"
	"net/http"

	"bnn/internal/pkg/auth"
	"bnn/internal/pkg/note"

	"github.com/gorilla/mux"
)

func main() {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-me"
		slog.Warn("JWT_SECRET is not set, using dev default")
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

	api.HandleFunc("/auth/signup", auth.SignUp).Methods(http.MethodPost)
	api.HandleFunc("/auth/signin", auth.SignIn).Methods(http.MethodPost)

	protected := api.NewRoute().Subrouter()
	protected.Use(auth.Middleware)

	protected.HandleFunc("/notes", note.ListNotes).Methods(http.MethodGet)
	protected.HandleFunc("/notes/{id}", note.GetNote).Methods(http.MethodGet)

	server := http.Server{
		Addr:    ":5458",
		Handler: router,
	}

	slog.Info("Starting server", "address", "http://localhost:5458")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed", "error", err)
	}
}