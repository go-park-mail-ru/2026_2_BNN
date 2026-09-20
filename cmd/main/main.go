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

	api.HandleFunc("/auth/signup", auth.SignUp).
		Methods(http.MethodPost)

	api.HandleFunc("/notes", note.ListNotes).
		Methods(http.MethodGet)

	server := http.Server{
		Addr:    ":5458",
		Handler: router,
	}

	slog.Info("запуск сервера", "address", "http://localhost:5458")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("ошибка запуска сервера", "error", err)
	}
}
