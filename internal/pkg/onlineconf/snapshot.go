package onlineconf

import (
	"context"
	"log/slog"
	"sync/atomic"

	"bnn/internal/models"

	"github.com/samber/lo"
)

type SnapshotCache struct {
	source   SnapshotSource
	snapshot atomic.Pointer[models.Snapshot]
}

func NewSnapshotCache(source SnapshotSource) *SnapshotCache {
	return &SnapshotCache{source: source}
}

func (c *SnapshotCache) Run(ctx context.Context) {
	runEvery(ctx, refreshInterval, c.refresh)
}

func (c *SnapshotCache) Get() models.Snapshot {
	snapshot := c.snapshot.Load()
	if snapshot == nil {
		return models.Snapshot{}
	}

	return *snapshot
}

func (c *SnapshotCache) Key(scope, key string) (models.OnlineConfKey, bool) {
	return lo.Find(c.Get().Keys, func(k models.OnlineConfKey) bool {
		return k.Scope == scope && k.Key == key
	})
}

func (c *SnapshotCache) refresh(ctx context.Context) {
	var revision *int64
	if current := c.snapshot.Load(); current != nil {
		known := current.Revision
		revision = &known
	}

	snapshot, changed, err := c.source.GetSnapshot(ctx, revision)
	if err != nil {
		slog.WarnContext(ctx, "Failed to refresh onlineconf snapshot", "error", err)
		return
	}

	if !changed {
		return
	}

	c.snapshot.Store(&snapshot)
}
