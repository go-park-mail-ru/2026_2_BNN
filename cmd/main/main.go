package main

import (
	"log/slog"
	"net/http"

	"bnn/internal/pkg/auth"
	"bnn/internal/pkg/note"

	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()

	router.PathPrefix("/static/").Handler(
		http.StripPrefix(
			"/static/",
			http.FileServer(http.Dir("static")),
		),
	)

	api := router.PathPrefix("/api").Subrouter()

	authRouter := api.PathPrefix("/auth").Subrouter()
	notesRouter := api.PathPrefix("/notes").Subrouter()

	authRouter.HandleFunc("/signup", auth.SignUp).
		Methods(http.MethodPost)

	notesRouter.HandleFunc("/getall", note.ListNotes).
		Methods(http.MethodGet)

	server := http.Server{
		Addr:    ":5458",
		Handler: router,
	}

	slog.Info("Starting server", "address", "http://localhost:5458")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed", "error", err)
	}
}
