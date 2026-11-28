package onlineconf

import (
	"context"
	"errors"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSnapshotCache(t *testing.T) {
	ctx := context.Background()
	source := mocks.NewMockSnapshotSource(gomock.NewController(t))
	cache := NewSnapshotCache(source)

	limit := models.OnlineConfKey{Scope: "backend", Key: "attachments.max_file_size_bytes", Kind: "config"}
	first := models.Snapshot{Revision: 5, Keys: []models.OnlineConfKey{limit}}

	// до первого снимка — пусто
	_, ok := cache.Key("backend", "attachments.max_file_size_bytes")
	assert.False(t, ok)

	// первый снимок
	source.EXPECT().GetSnapshot(gomock.Any(), int64(0)).Return(first, nil)
	cache.refresh(ctx)
	assert.Equal(t, int64(5), cache.Get().Revision)

	got, ok := cache.Key("backend", "attachments.max_file_size_bytes")
	assert.True(t, ok)
	assert.Equal(t, limit, got)

	// ревизия не изменилась — сервис отдал пустой ответ, снимок остаётся
	source.EXPECT().GetSnapshot(gomock.Any(), int64(5)).Return(models.Snapshot{Revision: 5}, nil)
	cache.refresh(ctx)
	assert.Equal(t, first, cache.Get())

	// сервис недоступен — снимок остаётся
	source.EXPECT().GetSnapshot(gomock.Any(), int64(5)).Return(models.Snapshot{}, errors.New("unavailable"))
	cache.refresh(ctx)
	assert.Equal(t, first, cache.Get())

	// новая ревизия — снимок заменяется
	second := models.Snapshot{Revision: 6}
	source.EXPECT().GetSnapshot(gomock.Any(), int64(5)).Return(second, nil)
	cache.refresh(ctx)
	assert.Equal(t, second, cache.Get())

	_, ok = cache.Key("backend", "attachments.max_file_size_bytes")
	assert.False(t, ok)
}

func TestSnapshotCacheRunStopsOnCancel(t *testing.T) {
	source := mocks.NewMockSnapshotSource(gomock.NewController(t))
	cache := NewSnapshotCache(source)

	ctx, cancel := context.WithCancel(context.Background())
	source.EXPECT().GetSnapshot(gomock.Any(), int64(0)).DoAndReturn(func(context.Context, int64) (models.Snapshot, error) {
		cancel()
		return models.Snapshot{Revision: 1}, nil
	})

	cache.Run(ctx)
	assert.Equal(t, int64(1), cache.Get().Revision)
}
