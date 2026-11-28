package onlineconf

import (
	"context"
	"time"
)

const refreshInterval = 15 * time.Second

// runEvery вызывает fn сразу и потом раз в interval, пока жив ctx.
func runEvery(ctx context.Context, interval time.Duration, fn func(ctx context.Context)) {
	fn(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn(ctx)
		}
	}
}
