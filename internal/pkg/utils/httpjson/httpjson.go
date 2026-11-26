package httpjson

import (
	"errors"
	"log/slog"
	"net/http"
)

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
