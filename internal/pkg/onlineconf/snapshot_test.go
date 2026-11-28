package onlineconf_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/mocks"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func refreshOnce(
	t *testing.T,
	cache *onlineconf.SnapshotCache,
	source *mocks.MockSnapshotSource,
	revision *int64,
	next models.Snapshot,
	changed bool,
	sourceErr error,
) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	matcher := gomock.Nil()
	if revision != nil {
		matcher = gomock.Eq(revision)
	}

	source.EXPECT().
		GetSnapshot(gomock.Any(), matcher).
		DoAndReturn(func(
			context.Context,
			*int64,
		) (models.Snapshot, bool, error) {
			cancel()
			return next, changed, sourceErr
		})

	done := make(chan struct{})
	go func() {
		cache.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("snapshot cache did not stop")
	}
}

func TestSnapshotCache(t *testing.T) {
	source := mocks.NewMockSnapshotSource(gomock.NewController(t))
	cache := onlineconf.NewSnapshotCache(source)

	limit := models.OnlineConfKey{
		Scope: onlineconf.ScopeBackend,
		Key:   "attachments.max_file_size_bytes",
		Kind:  onlineconf.KindConfig,
	}
	first := models.Snapshot{
		Revision: 5,
		Keys:     []models.OnlineConfKey{limit},
	}

	_, ok := cache.Key(limit.Scope, limit.Key)
	assert.False(t, ok)

	refreshOnce(t, cache, source, nil, first, true, nil)
	assert.Equal(t, first, cache.Get())

	got, ok := cache.Key(limit.Scope, limit.Key)
	assert.True(t, ok)
	assert.Equal(t, limit, got)

	known := int64(5)

	refreshOnce(
		t, cache, source, &known,
		models.Snapshot{}, false, nil,
	)
	assert.Equal(t, first, cache.Get())

	refreshOnce(
		t, cache, source, &known,
		models.Snapshot{}, false, errors.New("unavailable"),
	)
	assert.Equal(t, first, cache.Get())

	second := models.Snapshot{Revision: 6}
	refreshOnce(t, cache, source, &known, second, true, nil)
	assert.Equal(t, second, cache.Get())

	_, ok = cache.Key(limit.Scope, limit.Key)
	assert.False(t, ok)
}

func TestSnapshotCacheInitialRevisionZero(t *testing.T) {
	source := mocks.NewMockSnapshotSource(gomock.NewController(t))
	cache := onlineconf.NewSnapshotCache(source)

	empty := models.Snapshot{
		Revision: 0,
		Keys:     []models.OnlineConfKey{},
	}

	refreshOnce(t, cache, source, nil, empty, true, nil)
	assert.Equal(t, empty, cache.Get())

	zero := int64(0)
	refreshOnce(
		t, cache, source, &zero,
		models.Snapshot{}, false, nil,
	)
	assert.Equal(t, empty, cache.Get())
}

func TestSnapshotCacheValues(t *testing.T) {
	userID := uuid.NewV4()
	source := mocks.NewMockSnapshotSource(gomock.NewController(t))
	cache := onlineconf.NewSnapshotCache(source)

	// до первого снимка — значения по умолчанию
	assert.Equal(t, int64(10), cache.ConfigInt(onlineconf.ScopeBackend, "limit", 10))
	assert.True(t, cache.FlagEnabled(onlineconf.ScopeBackend, "upload", userID, true))

	snapshot := models.Snapshot{Revision: 1, Keys: []models.OnlineConfKey{
		{Scope: onlineconf.ScopeBackend, Key: "limit", Config: &models.OnlineConfConfig{ValueType: onlineconf.ValueNumber, Value: json.RawMessage("2048")}},
		{Scope: onlineconf.ScopeBackend, Key: "broken", Config: &models.OnlineConfConfig{ValueType: onlineconf.ValueNumber, Value: json.RawMessage("1.5")}},
		{Scope: onlineconf.ScopeBackend, Key: "upload", Flag: &models.OnlineConfFlag{Enabled: false}},
	}}
	refreshOnce(t, cache, source, nil, snapshot, true, nil)

	assert.Equal(t, int64(2048), cache.ConfigInt(onlineconf.ScopeBackend, "limit", 10))
	assert.Equal(t, int64(10), cache.ConfigInt(onlineconf.ScopeBackend, "broken", 10))
	assert.Equal(t, int64(10), cache.ConfigInt(onlineconf.ScopeBackend, "upload", 10))
	assert.False(t, cache.FlagEnabled(onlineconf.ScopeBackend, "upload", userID, true))
	assert.True(t, cache.FlagEnabled(onlineconf.ScopeBackend, "limit", userID, true))
}
