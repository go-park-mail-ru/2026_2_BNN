package postgres

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/database"
	"bnn/internal/pkg/onlineconf"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	uuid "github.com/satori/go.uuid"
)

const keyNameConstraint = "onlineconf_keys_scope_key_key"

//go:embed queries/read_keys.sql
var readKeysQuery string

//go:embed queries/read_key.sql
var readKeyQuery string

//go:embed queries/read_snapshot_keys.sql
var readSnapshotKeysQuery string

//go:embed queries/lock_key.sql
var lockKeyQuery string

//go:embed queries/create_key.sql
var createKeyQuery string

//go:embed queries/update_key.sql
var updateKeyQuery string

//go:embed queries/delete_key.sql
var deleteKeyQuery string

//go:embed queries/delete_overrides.sql
var deleteOverridesQuery string

//go:embed queries/create_overrides.sql
var createOverridesQuery string

//go:embed queries/create_history.sql
var createHistoryQuery string

//go:embed queries/read_history.sql
var readHistoryQuery string

//go:embed queries/read_revision.sql
var readRevisionQuery string

//go:embed queries/update_revision.sql
var updateRevisionQuery string

//go:embed queries/set_snapshot_transaction.sql
var setSnapshotTransactionQuery string

var _ onlineconf.Repo = (*Repo)(nil)

type Repo struct {
	db database.DB
}

func NewRepo(db database.DB) *Repo {
	return &Repo{db: db}
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *Repo) ReadKeys(ctx context.Context, query, scope, kind string) ([]models.OnlineConfKey, error) {
	return readKeys(ctx, r.db, readKeysQuery, query, scope, kind)
}

func (r *Repo) ReadKey(ctx context.Context, id uuid.UUID) (models.OnlineConfKey, error) {
	return readKey(ctx, r.db, id)
}

func readKey(ctx context.Context, db querier, id uuid.UUID) (models.OnlineConfKey, error) {
	key, err := scanKey(db.QueryRow(ctx, readKeyQuery, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return models.OnlineConfKey{}, models.ErrOnlineConfKeyNotFound
	}
	return key, err
}

func readKeys(ctx context.Context, db querier, query string, args ...any) ([]models.OnlineConfKey, error) {
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read onlineconf keys: %w", err)
	}
	defer rows.Close()

	keys := make([]models.OnlineConfKey, 0)
	for rows.Next() {
		key, err := scanKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read onlineconf key rows: %w", err)
	}
	return keys, nil
}

func scanKey(row pgx.Row) (models.OnlineConfKey, error) {
	var key models.OnlineConfKey
	var valueType *string
	var value, included, excluded []byte
	var enabled *bool
	var rolloutPercent *int32

	err := row.Scan(
		&key.ID, &key.Scope, &key.Key, &key.Kind, &key.Description, &key.IsPublic,
		&valueType, &value, &enabled, &rolloutPercent, &key.Version,
		&key.UpdatedBy.ID, &key.UpdatedBy.Login, &key.UpdatedAt, &included, &excluded,
	)
	if err != nil {
		return models.OnlineConfKey{}, fmt.Errorf("scan onlineconf key: %w", err)
	}

	switch key.Kind {
	case onlineconf.KindConfig:
		if valueType == nil || !json.Valid(value) {
			return models.OnlineConfKey{}, fmt.Errorf("invalid stored onlineconf config %s", key.ID)
		}
		key.Config = &models.OnlineConfConfig{ValueType: *valueType, Value: value}
	case onlineconf.KindFlag:
		if enabled == nil || rolloutPercent == nil {
			return models.OnlineConfKey{}, fmt.Errorf("invalid stored onlineconf flag %s", key.ID)
		}
		key.Flag = &models.OnlineConfFlag{
			Enabled: *enabled, RolloutPercent: *rolloutPercent,
			IncludeUsers: make([]models.OnlineConfUser, 0),
			ExcludeUsers: make([]models.OnlineConfUser, 0),
		}
		if err := json.Unmarshal(included, &key.Flag.IncludeUsers); err != nil {
			return models.OnlineConfKey{}, fmt.Errorf("decode included users: %w", err)
		}
		if err := json.Unmarshal(excluded, &key.Flag.ExcludeUsers); err != nil {
			return models.OnlineConfKey{}, fmt.Errorf("decode excluded users: %w", err)
		}
	default:
		return models.OnlineConfKey{}, fmt.Errorf("unknown stored onlineconf kind %q", key.Kind)
	}
	return key, nil
}

type keyValues struct {
	valueType      any
	value          any
	enabled        any
	rolloutPercent any
}

func valuesForKey(key models.OnlineConfKey) (keyValues, error) {
	switch key.Kind {
	case onlineconf.KindConfig:
		if key.Config == nil || key.Flag != nil || !json.Valid(key.Config.Value) {
			return keyValues{}, models.ErrValidation
		}
		return keyValues{valueType: key.Config.ValueType, value: []byte(key.Config.Value)}, nil
	case onlineconf.KindFlag:
		if key.Flag == nil || key.Config != nil {
			return keyValues{}, models.ErrValidation
		}
		return keyValues{enabled: key.Flag.Enabled, rolloutPercent: key.Flag.RolloutPercent}, nil
	default:
		return keyValues{}, models.ErrValidation
	}
}

func (r *Repo) CreateKey(ctx context.Context, key models.OnlineConfKey) (models.OnlineConfKey, error) {
	values, err := valuesForKey(key)
	if err != nil {
		return models.OnlineConfKey{}, err
	}
	var saved models.OnlineConfKey
	err = r.withTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, createKeyQuery,
			key.ID, key.Scope, key.Key, key.Kind, key.Description, key.IsPublic,
			values.valueType, values.value, values.enabled, values.rolloutPercent, key.UpdatedBy.ID,
		)
		if err != nil {
			return keyWriteError("create onlineconf key", err)
		}
		if err := createOverrides(ctx, tx, key); err != nil {
			return err
		}
		saved, err = readKey(ctx, tx, key.ID)
		if err != nil {
			return err
		}
		return recordChange(ctx, tx, key.ID, "create", nil, &saved, key.UpdatedBy.ID)
	})
	if err != nil {
		return models.OnlineConfKey{}, err
	}
	return saved, nil
}

func (r *Repo) UpdateKey(ctx context.Context, key models.OnlineConfKey) (models.OnlineConfKey, error) {
	values, err := valuesForKey(key)
	if err != nil {
		return models.OnlineConfKey{}, err
	}
	var saved models.OnlineConfKey
	err = r.withTx(ctx, func(tx pgx.Tx) error {
		old, err := lockAndReadKey(ctx, tx, key.ID, key.Version)
		if err != nil {
			return err
		}
		if old.Kind != key.Kind {
			return models.ErrValidation
		}
		tag, err := tx.Exec(ctx, updateKeyQuery,
			key.Scope, key.Key, key.Description, key.IsPublic,
			values.valueType, values.value, values.enabled, values.rolloutPercent,
			key.UpdatedBy.ID, key.ID, key.Version,
		)
		if err != nil {
			return keyWriteError("update onlineconf key", err)
		}
		if tag.RowsAffected() != 1 {
			return models.ErrOnlineConfVersionConflict
		}
		if _, err := tx.Exec(ctx, deleteOverridesQuery, key.ID); err != nil {
			return fmt.Errorf("delete onlineconf overrides: %w", err)
		}
		if err := createOverrides(ctx, tx, key); err != nil {
			return err
		}
		saved, err = readKey(ctx, tx, key.ID)
		if err != nil {
			return err
		}
		return recordChange(ctx, tx, key.ID, "update", &old, &saved, key.UpdatedBy.ID)
	})
	if err != nil {
		return models.OnlineConfKey{}, err
	}
	return saved, nil
}

func (r *Repo) DeleteKey(ctx context.Context, id uuid.UUID, version int32, userID uuid.UUID) error {
	return r.withTx(ctx, func(tx pgx.Tx) error {
		old, err := lockAndReadKey(ctx, tx, id, version)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, deleteKeyQuery, id, version)
		if err != nil {
			return fmt.Errorf("delete onlineconf key: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return models.ErrOnlineConfVersionConflict
		}
		return recordChange(ctx, tx, id, "delete", &old, nil, userID)
	})
}

func lockAndReadKey(ctx context.Context, tx pgx.Tx, id uuid.UUID, version int32) (models.OnlineConfKey, error) {
	var currentVersion int32
	err := tx.QueryRow(ctx, lockKeyQuery, id).Scan(&currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.OnlineConfKey{}, models.ErrOnlineConfKeyNotFound
	}
	if err != nil {
		return models.OnlineConfKey{}, fmt.Errorf("lock onlineconf key: %w", err)
	}
	if currentVersion != version {
		return models.OnlineConfKey{}, models.ErrOnlineConfVersionConflict
	}
	// Read overrides after acquiring the lock: a waiting writer needs a fresh snapshot.
	return readKey(ctx, tx, id)
}

func createOverrides(ctx context.Context, tx pgx.Tx, key models.OnlineConfKey) error {
	if key.Flag == nil || len(key.Flag.IncludeUsers)+len(key.Flag.ExcludeUsers) == 0 {
		return nil
	}
	var ids, logins []string
	var enabled []bool
	for _, user := range key.Flag.IncludeUsers {
		ids = append(ids, user.ID.String())
		logins = append(logins, user.Login)
		enabled = append(enabled, true)
	}
	for _, user := range key.Flag.ExcludeUsers {
		ids = append(ids, user.ID.String())
		logins = append(logins, user.Login)
		enabled = append(enabled, false)
	}
	if _, err := tx.Exec(ctx, createOverridesQuery, key.ID, ids, logins, enabled); err != nil {
		return fmt.Errorf("create onlineconf overrides: %w", err)
	}
	return nil
}

func recordChange(ctx context.Context, tx pgx.Tx, keyID uuid.UUID, action string, old, updated *models.OnlineConfKey, userID uuid.UUID) error {
	var oldJSON, newJSON []byte
	var err error
	if old != nil {
		oldJSON, err = json.Marshal(old)
		if err != nil {
			return fmt.Errorf("encode old onlineconf key: %w", err)
		}
	}
	if updated != nil {
		newJSON, err = json.Marshal(updated)
		if err != nil {
			return fmt.Errorf("encode new onlineconf key: %w", err)
		}
	}
	_, err = tx.Exec(ctx, createHistoryQuery, uuid.NewV4(), keyID, action, oldJSON, newJSON, userID)
	if err != nil {
		return fmt.Errorf("create onlineconf history: %w", err)
	}
	tag, err := tx.Exec(ctx, updateRevisionQuery)
	if err != nil {
		return fmt.Errorf("update onlineconf revision: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("onlineconf revision row is missing")
	}
	return nil
}

func (r *Repo) ReadHistory(ctx context.Context, keyID uuid.UUID) ([]models.OnlineConfHistory, error) {
	rows, err := r.db.Query(ctx, readHistoryQuery, keyID)
	if err != nil {
		return nil, fmt.Errorf("read onlineconf history: %w", err)
	}
	defer rows.Close()

	history := make([]models.OnlineConfHistory, 0)
	for rows.Next() {
		var entry models.OnlineConfHistory
		var oldJSON, newJSON []byte
		if err := rows.Scan(&entry.ID, &entry.KeyID, &entry.Action, &oldJSON, &newJSON,
			&entry.User.ID, &entry.User.Login, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan onlineconf history: %w", err)
		}
		if len(oldJSON) > 0 {
			if err := json.Unmarshal(oldJSON, &entry.OldValue); err != nil {
				return nil, fmt.Errorf("decode old onlineconf key: %w", err)
			}
		}
		if len(newJSON) > 0 {
			if err := json.Unmarshal(newJSON, &entry.NewValue); err != nil {
				return nil, fmt.Errorf("decode new onlineconf key: %w", err)
			}
		}
		history = append(history, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read onlineconf history rows: %w", err)
	}
	return history, nil
}

func (r *Repo) ReadSnapshot(ctx context.Context) (models.Snapshot, error) {
	var snapshot models.Snapshot
	err := r.withTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, setSnapshotTransactionQuery); err != nil {
			return fmt.Errorf("set onlineconf snapshot isolation: %w", err)
		}
		if err := tx.QueryRow(ctx, readRevisionQuery).Scan(&snapshot.Revision); err != nil {
			return fmt.Errorf("read onlineconf revision: %w", err)
		}
		var err error
		snapshot.Keys, err = readKeys(ctx, tx, readSnapshotKeysQuery)
		return err
	})
	if err != nil {
		return models.Snapshot{}, err
	}
	return snapshot, nil
}

func (r *Repo) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin onlineconf transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_ = tx.Rollback(rollbackCtx)
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit onlineconf transaction: %w", err)
	}
	committed = true
	return nil
}

func keyWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == keyNameConstraint {
		return models.ErrOnlineConfKeyExists
	}
	return fmt.Errorf("%s: %w", operation, err)
}
