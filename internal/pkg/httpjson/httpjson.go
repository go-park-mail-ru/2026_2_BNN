package httpjson

import (
	"context"
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
	ctx := context.Background()

	slog.ErrorContext(ctx, "Request failed", "error", err)
	w.WriteHeader(http.StatusInternalServerError)
}
