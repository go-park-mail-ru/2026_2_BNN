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

	slog.Warn("Bad request", "error", err)
	w.WriteHeader(http.StatusBadRequest)
}

func ResponseError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("Request failed", "error", err)
	w.WriteHeader(http.StatusInternalServerError)
}
