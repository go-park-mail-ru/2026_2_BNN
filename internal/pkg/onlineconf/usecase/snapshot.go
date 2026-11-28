package usecase

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"bnn/internal/models"
)

const (
	snapshotRefreshInterval = 10 * time.Second
	snapshotRefreshTimeout  = 5 * time.Second
)

func (u *Usecase) GetSnapshot(ctx context.Context, revision *int64) (models.Snapshot, bool, error) {
	if err := ctx.Err(); err != nil {
		return models.Snapshot{}, false, err
	}
	if revision != nil && *revision < 0 {
		return models.Snapshot{}, false, fmt.Errorf("%w: revision cannot be negative", models.ErrValidation)
	}
	snapshot := u.snapshot.Load()
	if snapshot == nil {
		return models.Snapshot{}, false, models.ErrOnlineConfSnapshotNotReady
	}
	if revision != nil && *revision == snapshot.Revision {
		return models.Snapshot{}, false, nil
	}
	return cloneSnapshot(*snapshot), true, nil
}

func (u *Usecase) RefreshSnapshot(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, snapshotRefreshTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}

	// Serialize reads and publication so an older refresh cannot overwrite a newer one.
	select {
	case u.refreshGate <- struct{}{}:
		defer func() { <-u.refreshGate }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	snapshot, err := u.repo.ReadSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("refresh onlineconf snapshot: %w", err)
	}
	if snapshot.Revision < 0 {
		return fmt.Errorf("stored onlineconf revision is negative")
	}
	snapshot = cloneSnapshot(snapshot)
	if err := ctx.Err(); err != nil {
		return err
	}
	u.snapshot.Store(&snapshot)
	return nil
}

func (u *Usecase) Run(ctx context.Context) {
	ticker := time.NewTicker(snapshotRefreshInterval)
	defer ticker.Stop()
	u.run(ctx, ticker.C)
}

func (u *Usecase) run(ctx context.Context, ticks <-chan time.Time) {
	u.refreshSnapshot(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ticks:
			if !ok {
				return
			}
			u.refreshSnapshot(ctx)
		}
	}
}

func (u *Usecase) refreshSnapshot(ctx context.Context) {
	if err := u.RefreshSnapshot(ctx); err != nil && ctx.Err() == nil {
		slog.WarnContext(ctx, "Failed to refresh OnlineConf snapshot", "error", err)
	}
}

func cloneSnapshot(snapshot models.Snapshot) models.Snapshot {
	cloned := models.Snapshot{
		Revision: snapshot.Revision,
		Keys:     make([]models.OnlineConfKey, len(snapshot.Keys)),
	}
	for i, key := range snapshot.Keys {
		cloned.Keys[i] = key
		cloned.Keys[i].RolloutPlanID = clonePointer(key.RolloutPlanID)
		cloned.Keys[i].RolloutStep = clonePointer(key.RolloutStep)
		cloned.Keys[i].StepReadyAt = clonePointer(key.StepReadyAt)
		if key.Config != nil {
			config := *key.Config
			config.Value = bytes.Clone(config.Value)
			cloned.Keys[i].Config = &config
		}
		if key.Flag != nil {
			flag := *key.Flag
			flag.IncludeUsers = append([]models.OnlineConfUser{}, flag.IncludeUsers...)
			flag.ExcludeUsers = append([]models.OnlineConfUser{}, flag.ExcludeUsers...)
			cloned.Keys[i].Flag = &flag
		}
	}
	return cloned
}

func clonePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
