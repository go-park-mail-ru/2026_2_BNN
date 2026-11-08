package database

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

type queryStartKey struct{}

type queryInfo struct {
	start time.Time
	sql   string
}

type Tracer struct {
	logger *slog.Logger
}

func NewTracer(logger *slog.Logger) *Tracer {
	return &Tracer{logger: logger}
}

func (t *Tracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	info := queryInfo{
		start: time.Now(),
		sql:   data.SQL,
	}
	return context.WithValue(ctx, queryStartKey{}, info)
}

func (t *Tracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	info, ok := ctx.Value(queryStartKey{}).(queryInfo)
	if !ok {
		return
	}

	duration := time.Since(info.start)

	t.logger.Info("DB Query",
		"sql", info.sql,
		"duration", duration.String(),
		"err", data.Err,
	)
}
