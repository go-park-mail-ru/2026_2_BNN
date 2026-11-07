package logger

import (
	"bnn/internal/pkg/middleware"
	"context"
	"log/slog"
)

func WithContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	reqID, ok := ctx.Value(middleware.RequestIDKey).(string)
	if ok {
		return logger.With("request_id", reqID)
	}
	return logger
}
