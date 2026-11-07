package database

import (
	"context"
	"log/slog"
	"time"

	"bnn/internal/pkg/middleware"

	"github.com/jackc/pgx/v5"
)

type Tracer struct {
	logger *slog.Logger
}

func NewTracer(logger *slog.Logger) *Tracer {
	return &Tracer{logger: logger}
}

func (t *Tracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, "query_start", time.Now())
}

func (t *Tracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	start, _ := ctx.Value("query_start").(time.Time)
	reqID, _ := ctx.Value(middleware.RequestIDKey).(string)

	t.logger.Info("DB Query",
		"request_id", reqID,
		"sql", data.CommandTag.String(),
		"duration", time.Since(start).String(),
		"err", data.Err,
	)
}
