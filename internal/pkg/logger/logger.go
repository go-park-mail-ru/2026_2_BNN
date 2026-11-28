package logger

import (
	"log/slog"
	"os"
)

func New() *slog.Logger {
	base := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(ContextHandler{Handler: base})
}
