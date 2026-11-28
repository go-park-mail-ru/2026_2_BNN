package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const snapshotTestTimeout = 5 * time.Second

func snapshotFixture(revision int64) models.Snapshot {
	return models.Snapshot{
		Revision: revision,
		Keys: []models.OnlineConfKey{
			{
				ID: uuid.NewV4(), Scope: "backend", Key: "test.value", Kind: "config", Version: 1,
				Config: &models.OnlineConfConfig{ValueType: "number", Value: json.RawMessage(strconv.FormatInt(revision, 10))},
			},
			{
				ID: uuid.NewV4(), Scope: "backend", Key: "test.flag", Kind: "flag", Version: 1,
				Flag: &models.OnlineConfFlag{
					Enabled: true, RolloutPercent: 25,
					IncludeUsers: []models.OnlineConfUser{{ID: uuid.NewV4(), Login: "included"}},
					ExcludeUsers: []models.OnlineConfUser{{ID: uuid.NewV4(), Login: "excluded"}},
				},
			},
		},
	}
}

func waitSnapshotEvent(t *testing.T, event <-chan struct{}) {
	t.Helper()
	select {
	case <-event:
	case <-time.After(snapshotTestTimeout):
		t.Fatal("snapshot operation did not finish")
	}
}

func waitSnapshotRevision(t *testing.T, uc *Usecase, revision int64) {
	t.Helper()
	require.Eventually(t, func() bool {
		snapshot, changed, err := uc.GetSnapshot(context.Background(), nil)
		return err == nil && changed && snapshot.Revision == revision
	}, snapshotTestTimeout, time.Millisecond)
}

func TestGetSnapshotBeforeLoad(t *testing.T) {
	_, _, uc := newUsecase(t)
	zero := int64(0)
	for _, revision := range []*int64{nil, &zero} {
		snapshot, changed, err := uc.GetSnapshot(context.Background(), revision)

		assert.ErrorIs(t, err, models.ErrOnlineConfSnapshotNotReady)
		assert.False(t, changed)
		assert.Empty(t, snapshot)
	}
}

func TestGetEmptySnapshotAtRevisionZero(t *testing.T) {
	repo, _, uc := newUsecase(t)
	repo.EXPECT().ReadSnapshot(gomock.Any()).Return(models.Snapshot{Revision: 0}, nil)
	require.NoError(t, uc.RefreshSnapshot(context.Background()))

	snapshot, changed, err := uc.GetSnapshot(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Zero(t, snapshot.Revision)
	assert.NotNil(t, snapshot.Keys)
	assert.Empty(t, snapshot.Keys)

	zero := int64(0)
	snapshot, changed, err = uc.GetSnapshot(context.Background(), &zero)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Empty(t, snapshot)
}

func TestGetSnapshotRevision(t *testing.T) {
	repo, _, uc := newUsecase(t)
	source := snapshotFixture(3)
	repo.EXPECT().ReadSnapshot(gomock.Any()).Return(source, nil)
	require.NoError(t, uc.RefreshSnapshot(context.Background()))

	for _, revision := range []int64{0, 2, 3, 4} {
		t.Run(strconv.FormatInt(revision, 10), func(t *testing.T) {
			snapshot, changed, err := uc.GetSnapshot(context.Background(), &revision)

			require.NoError(t, err)
			assert.Equal(t, revision != source.Revision, changed)
			if changed {
				assert.Equal(t, source, snapshot)
			} else {
				assert.Empty(t, snapshot)
			}
		})
	}
}

func TestGetSnapshotInvalidRequest(t *testing.T) {
	_, _, uc := newUsecase(t)
	negative := int64(-1)
	_, _, err := uc.GetSnapshot(context.Background(), &negative)
	assert.ErrorIs(t, err, models.ErrValidation)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = uc.GetSnapshot(ctx, nil)
	assert.ErrorIs(t, err, context.Canceled)
	assert.ErrorIs(t, uc.RefreshSnapshot(ctx), context.Canceled)
}

func TestSnapshotDoesNotShareMutableData(t *testing.T) {
	repo, _, uc := newUsecase(t)
	source := snapshotFixture(1)
	originalJSON, err := json.Marshal(source)
	require.NoError(t, err)
	repo.EXPECT().ReadSnapshot(gomock.Any()).Return(source, nil)
	require.NoError(t, uc.RefreshSnapshot(context.Background()))

	source.Keys[0].Config.Value[0] = '9'
	source.Keys[0].Description = "changed source"
	source.Keys[1].Flag.Enabled = false
	source.Keys[1].Flag.IncludeUsers[0].Login = "changed source"
	source.Keys[1].Flag.ExcludeUsers[0].ID = uuid.NewV4()

	snapshot, changed, err := uc.GetSnapshot(context.Background(), nil)
	require.NoError(t, err)
	require.True(t, changed)
	actualJSON, err := json.Marshal(snapshot)
	require.NoError(t, err)
	assert.JSONEq(t, string(originalJSON), string(actualJSON))

	snapshot.Keys[0].Config.Value[0] = '!'
	snapshot.Keys[0].Key = "changed response"
	snapshot.Keys[1].Flag.Enabled = false
	snapshot.Keys[1].Flag.IncludeUsers[0].ID = uuid.NewV4()
	snapshot.Keys[1].Flag.ExcludeUsers[0].Login = "changed response"
	unchanged, _, err := uc.GetSnapshot(context.Background(), nil)
	require.NoError(t, err)
	actualJSON, err = json.Marshal(unchanged)
	require.NoError(t, err)
	assert.JSONEq(t, string(originalJSON), string(actualJSON))
}

func TestRefreshSnapshotKeepsLastSuccess(t *testing.T) {
	for _, invalidRevision := range []bool{false, true} {
		t.Run(fmt.Sprint(invalidRevision), func(t *testing.T) {
			repo, _, uc := newUsecase(t)
			source := snapshotFixture(7)
			repo.EXPECT().ReadSnapshot(gomock.Any()).Return(source, nil)
			require.NoError(t, uc.RefreshSnapshot(context.Background()))
			if invalidRevision {
				repo.EXPECT().ReadSnapshot(gomock.Any()).Return(models.Snapshot{Revision: -1}, nil)
			} else {
				repo.EXPECT().ReadSnapshot(gomock.Any()).Return(models.Snapshot{}, errDependency)
			}

			err := uc.RefreshSnapshot(context.Background())
			require.Error(t, err)
			if !invalidRevision {
				assert.ErrorIs(t, err, errDependency)
			}
			snapshot, changed, err := uc.GetSnapshot(context.Background(), nil)
			require.NoError(t, err)
			assert.True(t, changed)
			assert.Equal(t, source, snapshot)
		})
	}
}

func TestRefreshDoesNotPublishAfterCancellation(t *testing.T) {
	repo, _, uc := newUsecase(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo.EXPECT().ReadSnapshot(gomock.Any()).DoAndReturn(func(queryCtx context.Context) (models.Snapshot, error) {
		deadline, ok := queryCtx.Deadline()
		assert.True(t, ok)
		assert.LessOrEqual(t, time.Until(deadline), snapshotRefreshTimeout)
		cancel()
		return snapshotFixture(1), nil
	})

	assert.ErrorIs(t, uc.RefreshSnapshot(ctx), context.Canceled)
	_, _, err := uc.GetSnapshot(context.Background(), nil)
	assert.ErrorIs(t, err, models.ErrOnlineConfSnapshotNotReady)
}

func TestSuccessfulChangesRefreshSnapshot(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete"} {
		for _, refreshFails := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/refresh_error=%t", operation, refreshFails), func(t *testing.T) {
				repo, _, uc := newUsecase(t)
				before := snapshotFixture(1)
				if operation == "delete" {
					before.Keys = before.Keys[:1]
				}
				repo.EXPECT().ReadSnapshot(gomock.Any()).Return(before, nil)
				require.NoError(t, uc.RefreshSnapshot(context.Background()))
				actor := uuid.NewV4()
				old := before.Keys[0]
				saved := old
				saved.Version = 2
				saved.Config = &models.OnlineConfConfig{ValueType: "number", Value: json.RawMessage("2")}
				after := models.Snapshot{Revision: 2, Keys: append([]models.OnlineConfKey{}, before.Keys...)}
				after.Keys[0] = saved
				input := configInput("number", "2")
				switch operation {
				case "create":
					saved.ID = uuid.NewV4()
					after.Keys = append([]models.OnlineConfKey{saved}, before.Keys...)
					repo.EXPECT().CreateKey(gomock.Any(), gomock.Any()).Return(saved, nil)
				case "update":
					repo.EXPECT().ReadKey(gomock.Any(), old.ID).Return(old, nil)
					repo.EXPECT().UpdateKey(gomock.Any(), gomock.Any()).Return(saved, nil)
				case "delete":
					after.Keys = []models.OnlineConfKey{}
					repo.EXPECT().DeleteKey(gomock.Any(), old.ID, old.Version, actor).Return(nil)
				}
				if refreshFails {
					repo.EXPECT().ReadSnapshot(gomock.Any()).Return(models.Snapshot{}, errDependency)
				} else {
					repo.EXPECT().ReadSnapshot(gomock.Any()).Return(after, nil)
				}

				var err error
				switch operation {
				case "create":
					var got models.OnlineConfKey
					got, err = uc.CreateKey(context.Background(), actor, input)
					assert.Equal(t, saved, got)
				case "update":
					var got models.OnlineConfKey
					got, err = uc.UpdateKey(context.Background(), actor, old.ID, old.Version, input)
					assert.Equal(t, saved, got)
				case "delete":
					err = uc.DeleteKey(context.Background(), actor, old.ID, old.Version)
				}
				require.NoError(t, err)
				if refreshFails {
					snapshot, _, err := uc.GetSnapshot(context.Background(), nil)
					require.NoError(t, err)
					assert.Equal(t, before, snapshot)
					repo.EXPECT().ReadSnapshot(gomock.Any()).Return(after, nil)
					require.NoError(t, uc.RefreshSnapshot(context.Background()))
				}
				snapshot, changed, err := uc.GetSnapshot(context.Background(), &before.Revision)
				require.NoError(t, err)
				assert.True(t, changed)
				assert.Equal(t, after, snapshot)
			})
		}
	}
}

func TestRefreshWaitCanBeCancelled(t *testing.T) {
	repo, _, uc := newUsecase(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	releaseQuery := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseQuery)
	repo.EXPECT().ReadSnapshot(gomock.Any()).DoAndReturn(func(ctx context.Context) (models.Snapshot, error) {
		close(entered)
		select {
		case <-release:
			return snapshotFixture(1), nil
		case <-ctx.Done():
			return models.Snapshot{}, ctx.Err()
		}
	})
	firstDone := make(chan error, 1)
	go func() { firstDone <- uc.RefreshSnapshot(context.Background()) }()
	waitSnapshotEvent(t, entered)

	// Readers do not wait for the database refresh.
	_, _, err := uc.GetSnapshot(context.Background(), nil)
	assert.ErrorIs(t, err, models.ErrOnlineConfSnapshotNotReady)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	secondDone := make(chan struct{})
	var secondErr error
	go func() {
		secondErr = uc.RefreshSnapshot(ctx)
		close(secondDone)
	}()
	cancel()
	waitSnapshotEvent(t, secondDone)
	assert.ErrorIs(t, secondErr, context.Canceled)
	releaseQuery()
	select {
	case err := <-firstDone:
		require.NoError(t, err)
	case <-time.After(snapshotTestTimeout):
		t.Fatal("first refresh did not finish")
	}
}

func TestConcurrentSnapshotAccess(t *testing.T) {
	repo, _, uc := newUsecase(t)
	var revision, active, peak atomic.Int64
	repo.EXPECT().ReadSnapshot(gomock.Any()).AnyTimes().DoAndReturn(func(context.Context) (models.Snapshot, error) {
		count := active.Add(1)
		defer active.Add(-1)
		for previous := peak.Load(); count > previous; previous = peak.Load() {
			if peak.CompareAndSwap(previous, count) {
				break
			}
		}
		next := revision.Add(1)
		runtime.Gosched()
		return snapshotFixture(next), nil
	})
	require.NoError(t, uc.RefreshSnapshot(context.Background()))
	start := make(chan struct{})
	errors := make(chan error, 8)
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(2)
		go func() {
			defer workers.Done()
			<-start
			for j := 0; j < 25; j++ {
				if err := uc.RefreshSnapshot(context.Background()); err != nil {
					errors <- err
					return
				}
			}
		}()
		go func() {
			defer workers.Done()
			<-start
			for j := 0; j < 200; j++ {
				snapshot, changed, err := uc.GetSnapshot(context.Background(), nil)
				if err != nil || !changed || len(snapshot.Keys) != 2 {
					errors <- fmt.Errorf("incomplete snapshot: changed=%t, error=%v", changed, err)
					return
				}
				if string(snapshot.Keys[0].Config.Value) != strconv.FormatInt(snapshot.Revision, 10) ||
					!snapshot.Keys[1].Flag.Enabled || snapshot.Keys[1].Flag.IncludeUsers[0].Login != "included" {
					errors <- fmt.Errorf("inconsistent snapshot at revision %d", snapshot.Revision)
					return
				}
				snapshot.Keys[0].Config.Value[0] = '!'
				snapshot.Keys[1].Flag.Enabled = false
				snapshot.Keys[1].Flag.IncludeUsers[0].Login = "changed"
			}
		}()
	}
	close(start)
	workers.Wait()
	close(errors)
	for err := range errors {
		assert.NoError(t, err)
	}
	assert.Equal(t, int64(1), peak.Load())
	assert.Equal(t, int64(101), revision.Load())
	snapshot, _, err := uc.GetSnapshot(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, revision.Load(), snapshot.Revision)
}

func TestSnapshotRunRetriesAndStops(t *testing.T) {
	repo, _, uc := newUsecase(t)
	initialAttempt := make(chan struct{})
	gomock.InOrder(
		repo.EXPECT().ReadSnapshot(gomock.Any()).DoAndReturn(func(context.Context) (models.Snapshot, error) {
			close(initialAttempt)
			return models.Snapshot{}, errDependency
		}),
		repo.EXPECT().ReadSnapshot(gomock.Any()).Return(snapshotFixture(1), nil),
		repo.EXPECT().ReadSnapshot(gomock.Any()).Return(models.Snapshot{}, errDependency),
		repo.EXPECT().ReadSnapshot(gomock.Any()).Return(snapshotFixture(2), nil),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ticks := make(chan time.Time)
	done := make(chan struct{})
	go func() { uc.run(ctx, ticks); close(done) }()
	waitSnapshotEvent(t, initialAttempt)
	_, _, err := uc.GetSnapshot(context.Background(), nil)
	assert.ErrorIs(t, err, models.ErrOnlineConfSnapshotNotReady)
	for _, revision := range []int64{1, 1, 2} {
		select {
		case ticks <- time.Now():
		case <-time.After(snapshotTestTimeout):
			t.Fatal("snapshot loop did not accept tick")
		}
		waitSnapshotRevision(t, uc, revision)
	}
	cancel()
	waitSnapshotEvent(t, done)
}

func TestSnapshotRunAlreadyCancelled(t *testing.T) {
	_, _, uc := newUsecase(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() { uc.Run(ctx); close(done) }()
	waitSnapshotEvent(t, done)
}
