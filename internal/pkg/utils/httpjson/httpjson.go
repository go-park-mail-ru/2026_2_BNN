package httpjson

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func RequestError(w http.ResponseWriter, r *http.Request, err error) {
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	slog.WarnContext(r.Context(), "Bad request", "error", err)
	w.WriteHeader(http.StatusBadRequest)
}

func ResponseError(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "Request failed", "error", err)
	w.WriteHeader(http.StatusInternalServerError)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorBody{Code: code, Message: message})
}
