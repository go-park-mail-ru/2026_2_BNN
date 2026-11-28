package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"bnn/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errDB      = errors.New("db is down")
	keyColumns = []string{
		"id", "scope", "key", "kind", "description", "is_public",
		"value_type", "value", "enabled", "rollout_percent", "version",
		"updated_by", "login", "updated_at", "included", "excluded",
	}
	historyColumns = []string{"id", "key_id", "action", "old_value", "new_value", "user_id", "login", "created_at"}
)

func newMock(t *testing.T) (pgxmock.PgxPoolIface, *Repo) {
	t.Helper()
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
		mock.Close()
	})
	return mock, NewRepo(mock)
}

func newConfig() models.OnlineConfKey {
	return models.OnlineConfKey{
		ID: uuid.NewV4(), Scope: models.OnlineConfScopeBackend,
		Key: "attachments.max_file_size_bytes", Kind: models.OnlineConfKindConfig,
		Description: "Лимит вложений", IsPublic: true, Version: 1,
		Config:    &models.OnlineConfConfig{ValueType: models.OnlineConfValueNumber, Value: json.RawMessage("10485760")},
		UpdatedBy: models.OnlineConfUser{ID: uuid.NewV4(), Login: "liza"},
		UpdatedAt: time.Date(2026, 11, 28, 10, 0, 0, 0, time.UTC),
	}
}

func newFlag() models.OnlineConfKey {
	key := newConfig()
	key.Key = "attachments.upload_enabled"
	key.Kind = models.OnlineConfKindFlag
	key.Config = nil
	key.Flag = &models.OnlineConfFlag{
		Enabled: true, RolloutPercent: 25,
		IncludeUsers: []models.OnlineConfUser{{ID: uuid.NewV4(), Login: "included"}},
		ExcludeUsers: []models.OnlineConfUser{{ID: uuid.NewV4(), Login: "excluded"}},
	}
	return key
}

func keyRow(key models.OnlineConfKey) []any {
	var valueType, value, enabled, percent any
	included, excluded := []byte("[]"), []byte("[]")
	if key.Config != nil {
		valueType = &key.Config.ValueType
		value = []byte(key.Config.Value)
	}
	if key.Flag != nil {
		enabled = &key.Flag.Enabled
		percent = &key.Flag.RolloutPercent
		included, _ = json.Marshal(key.Flag.IncludeUsers)
		excluded, _ = json.Marshal(key.Flag.ExcludeUsers)
	}
	return []any{
		key.ID.String(), key.Scope, key.Key, key.Kind, key.Description, key.IsPublic,
		valueType, value, enabled, percent, key.Version,
		key.UpdatedBy.ID.String(), key.UpdatedBy.Login, key.UpdatedAt, included, excluded,
	}
}

func expectRead(mock pgxmock.PgxPoolIface, key models.OnlineConfKey) {
	mock.ExpectQuery(readKeyQuery).WithArgs(key.ID).
		WillReturnRows(pgxmock.NewRows(keyColumns).AddRow(keyRow(key)...))
}

func expectLock(mock pgxmock.PgxPoolIface, key models.OnlineConfKey) {
	mock.ExpectQuery(lockKeyQuery).WithArgs(key.ID).
		WillReturnRows(pgxmock.NewRows([]string{"version"}).AddRow(key.Version))
}

func expectCreate(mock pgxmock.PgxPoolIface, key models.OnlineConfKey) *pgxmock.ExpectedExec {
	var valueType, value, enabled, percent any
	if key.Config != nil {
		valueType, value = key.Config.ValueType, []byte(key.Config.Value)
	}
	if key.Flag != nil {
		enabled, percent = key.Flag.Enabled, key.Flag.RolloutPercent
	}
	return mock.ExpectExec(createKeyQuery).WithArgs(
		key.ID, key.Scope, key.Key, key.Kind, key.Description, key.IsPublic,
		valueType, value, enabled, percent, key.UpdatedBy.ID,
	)
}

func expectUpdate(mock pgxmock.PgxPoolIface, key models.OnlineConfKey) *pgxmock.ExpectedExec {
	var valueType, value, enabled, percent any
	if key.Config != nil {
		valueType, value = key.Config.ValueType, []byte(key.Config.Value)
	}
	if key.Flag != nil {
		enabled, percent = key.Flag.Enabled, key.Flag.RolloutPercent
	}
	return mock.ExpectExec(updateKeyQuery).WithArgs(
		key.Scope, key.Key, key.Description, key.IsPublic, valueType, value, enabled, percent,
		key.UpdatedBy.ID, key.ID, key.Version,
	)
}

type keyJSONArg struct {
	key *models.OnlineConfKey
}

func (arg keyJSONArg) Match(value any) bool {
	data, ok := value.([]byte)
	if !ok {
		return false
	}
	if arg.key == nil {
		return data == nil
	}
	var actual models.OnlineConfKey
	return json.Unmarshal(data, &actual) == nil && reflect.DeepEqual(*arg.key, actual)
}

func expectHistory(mock pgxmock.PgxPoolIface, id uuid.UUID, action string, old, updated *models.OnlineConfKey, actor uuid.UUID) *pgxmock.ExpectedExec {
	return mock.ExpectExec(createHistoryQuery).
		WithArgs(pgxmock.AnyArg(), id, action, keyJSONArg{old}, keyJSONArg{updated}, actor)
}

func expectRevision(mock pgxmock.PgxPoolIface) {
	mock.ExpectExec(updateRevisionQuery).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
}

func TestCreateKey(t *testing.T) {
	for _, key := range []models.OnlineConfKey{newConfig(), newFlag()} {
		t.Run(key.Kind, func(t *testing.T) {
			mock, repo := newMock(t)
			mock.ExpectBegin()
			expectCreate(mock, key).WillReturnResult(pgxmock.NewResult("INSERT", 1))
			if key.Flag != nil {
				mock.ExpectExec(createOverridesQuery).WithArgs(key.ID,
					[]string{key.Flag.IncludeUsers[0].ID.String(), key.Flag.ExcludeUsers[0].ID.String()},
					[]string{"included", "excluded"}, []bool{true, false},
				).WillReturnResult(pgxmock.NewResult("INSERT", 2))
			}
			expectRead(mock, key)
			expectHistory(mock, key.ID, "create", nil, &key, key.UpdatedBy.ID).
				WillReturnResult(pgxmock.NewResult("INSERT", 1))
			expectRevision(mock)
			mock.ExpectCommit()

			got, err := repo.CreateKey(context.Background(), key)

			require.NoError(t, err)
			assert.Equal(t, key, got)
		})
	}
}

func TestCreateKeyRollback(t *testing.T) {
	key := newConfig()
	duplicate := &pgconn.PgError{Code: "23505", ConstraintName: keyNameConstraint}
	otherDuplicate := &pgconn.PgError{Code: "23505", ConstraintName: "onlineconf_keys_pkey"}
	tests := []struct {
		name    string
		prepare func(pgxmock.PgxPoolIface)
		wantErr error
	}{
		{"duplicate name", func(mock pgxmock.PgxPoolIface) {
			expectCreate(mock, key).WillReturnError(duplicate)
		}, models.ErrOnlineConfKeyExists},
		{"other unique violation", func(mock pgxmock.PgxPoolIface) {
			expectCreate(mock, key).WillReturnError(otherDuplicate)
		}, otherDuplicate},
		{"readback error", func(mock pgxmock.PgxPoolIface) {
			expectCreate(mock, key).WillReturnResult(pgxmock.NewResult("INSERT", 1))
			mock.ExpectQuery(readKeyQuery).WithArgs(key.ID).WillReturnError(errDB)
		}, errDB},
		{"history error", func(mock pgxmock.PgxPoolIface) {
			expectCreate(mock, key).WillReturnResult(pgxmock.NewResult("INSERT", 1))
			expectRead(mock, key)
			expectHistory(mock, key.ID, "create", nil, &key, key.UpdatedBy.ID).WillReturnError(errDB)
		}, errDB},
		{"revision error", func(mock pgxmock.PgxPoolIface) {
			expectCreate(mock, key).WillReturnResult(pgxmock.NewResult("INSERT", 1))
			expectRead(mock, key)
			expectHistory(mock, key.ID, "create", nil, &key, key.UpdatedBy.ID).
				WillReturnResult(pgxmock.NewResult("INSERT", 1))
			mock.ExpectExec(updateRevisionQuery).WillReturnError(errDB)
		}, errDB},
		{"missing revision row", func(mock pgxmock.PgxPoolIface) {
			expectCreate(mock, key).WillReturnResult(pgxmock.NewResult("INSERT", 1))
			expectRead(mock, key)
			expectHistory(mock, key.ID, "create", nil, &key, key.UpdatedBy.ID).
				WillReturnResult(pgxmock.NewResult("INSERT", 1))
			mock.ExpectExec(updateRevisionQuery).WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		}, nil},
		{"commit error", func(mock pgxmock.PgxPoolIface) {
			expectCreate(mock, key).WillReturnResult(pgxmock.NewResult("INSERT", 1))
			expectRead(mock, key)
			expectHistory(mock, key.ID, "create", nil, &key, key.UpdatedBy.ID).
				WillReturnResult(pgxmock.NewResult("INSERT", 1))
			expectRevision(mock)
			mock.ExpectCommit().WillReturnError(errDB)
		}, errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, repo := newMock(t)
			mock.ExpectBegin()
			tt.prepare(mock)
			mock.ExpectRollback()

			got, err := repo.CreateKey(context.Background(), key)

			require.Error(t, err)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			}
			assert.Empty(t, got)
		})
	}
}

func TestCreateKeyOverridesFailure(t *testing.T) {
	key := newFlag()
	mock, repo := newMock(t)
	mock.ExpectBegin()
	expectCreate(mock, key).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(createOverridesQuery).
		WithArgs(key.ID, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errDB)
	mock.ExpectRollback()

	_, err := repo.CreateKey(context.Background(), key)

	assert.ErrorIs(t, err, errDB)
}

func TestCreateKeyBeginFailure(t *testing.T) {
	mock, repo := newMock(t)
	mock.ExpectBegin().WillReturnError(errDB)

	_, err := repo.CreateKey(context.Background(), newConfig())

	assert.ErrorIs(t, err, errDB)
}

func TestUpdateKey(t *testing.T) {
	old := newFlag()
	updated := old
	updated.Flag = &models.OnlineConfFlag{
		Enabled: true, RolloutPercent: 50,
		IncludeUsers: []models.OnlineConfUser{}, ExcludeUsers: []models.OnlineConfUser{},
	}
	updated.UpdatedBy = models.OnlineConfUser{ID: uuid.NewV4(), Login: "seyran"}
	saved := updated
	saved.Version++
	saved.UpdatedAt = saved.UpdatedAt.Add(time.Minute)
	mock, repo := newMock(t)
	mock.ExpectBegin()
	expectLock(mock, old)
	expectRead(mock, old)
	expectUpdate(mock, updated).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(deleteOverridesQuery).WithArgs(old.ID).WillReturnResult(pgxmock.NewResult("DELETE", 2))
	expectRead(mock, saved)
	expectHistory(mock, old.ID, "update", &old, &saved, updated.UpdatedBy.ID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	expectRevision(mock)
	mock.ExpectCommit()

	got, err := repo.UpdateKey(context.Background(), updated)

	require.NoError(t, err)
	assert.Equal(t, saved, got)
	assert.Empty(t, got.Flag.IncludeUsers)
	assert.Empty(t, got.Flag.ExcludeUsers)
}

func TestUpdateKeyReplacesOverrides(t *testing.T) {
	old := newFlag()
	updated := old
	updated.Flag = &models.OnlineConfFlag{
		Enabled: false, RolloutPercent: 0,
		IncludeUsers: old.Flag.ExcludeUsers, ExcludeUsers: old.Flag.IncludeUsers,
	}
	saved := updated
	saved.Version++
	mock, repo := newMock(t)
	mock.ExpectBegin()
	expectLock(mock, old)
	expectRead(mock, old)
	expectUpdate(mock, updated).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(deleteOverridesQuery).WithArgs(old.ID).WillReturnResult(pgxmock.NewResult("DELETE", 2))
	mock.ExpectExec(createOverridesQuery).WithArgs(old.ID,
		[]string{old.Flag.ExcludeUsers[0].ID.String(), old.Flag.IncludeUsers[0].ID.String()},
		[]string{"excluded", "included"}, []bool{true, false},
	).WillReturnResult(pgxmock.NewResult("INSERT", 2))
	expectRead(mock, saved)
	expectHistory(mock, old.ID, "update", &old, &saved, updated.UpdatedBy.ID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	expectRevision(mock)
	mock.ExpectCommit()

	got, err := repo.UpdateKey(context.Background(), updated)

	require.NoError(t, err)
	assert.Equal(t, saved, got)
}

func TestVersionConflict(t *testing.T) {
	for _, operation := range []string{"update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			key := newConfig()
			current := key
			current.Version++
			mock, repo := newMock(t)
			mock.ExpectBegin()
			expectLock(mock, current)
			mock.ExpectRollback()

			var err error
			if operation == "update" {
				_, err = repo.UpdateKey(context.Background(), key)
			} else {
				err = repo.DeleteKey(context.Background(), key.ID, key.Version, key.UpdatedBy.ID)
			}

			assert.ErrorIs(t, err, models.ErrOnlineConfVersionConflict)
		})
	}
}

func TestWriteMissingKey(t *testing.T) {
	for _, operation := range []string{"update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			key := newConfig()
			mock, repo := newMock(t)
			mock.ExpectBegin()
			mock.ExpectQuery(lockKeyQuery).WithArgs(key.ID).WillReturnError(pgx.ErrNoRows)
			mock.ExpectRollback()

			var err error
			if operation == "update" {
				_, err = repo.UpdateKey(context.Background(), key)
			} else {
				err = repo.DeleteKey(context.Background(), key.ID, key.Version, key.UpdatedBy.ID)
			}

			assert.ErrorIs(t, err, models.ErrOnlineConfKeyNotFound)
		})
	}
}

func TestUpdateKeyCannotChangeKind(t *testing.T) {
	old := newConfig()
	updated := newFlag()
	updated.ID = old.ID
	mock, repo := newMock(t)
	mock.ExpectBegin()
	expectLock(mock, old)
	expectRead(mock, old)
	mock.ExpectRollback()

	_, err := repo.UpdateKey(context.Background(), updated)

	assert.ErrorIs(t, err, models.ErrValidation)
}

func TestUpdateKeyRollback(t *testing.T) {
	old := newConfig()
	updated := old
	updated.Config = &models.OnlineConfConfig{ValueType: models.OnlineConfValueNumber, Value: json.RawMessage("2097152")}
	saved := updated
	saved.Version++
	for _, stage := range []string{"duplicate name", "no updated row", "delete overrides", "history"} {
		t.Run(stage, func(t *testing.T) {
			mock, repo := newMock(t)
			mock.ExpectBegin()
			expectLock(mock, old)
			expectRead(mock, old)
			var wantErr error
			switch stage {
			case "duplicate name":
				expectUpdate(mock, updated).WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: keyNameConstraint})
				wantErr = models.ErrOnlineConfKeyExists
			case "no updated row":
				expectUpdate(mock, updated).WillReturnResult(pgxmock.NewResult("UPDATE", 0))
				wantErr = models.ErrOnlineConfVersionConflict
			case "delete overrides":
				expectUpdate(mock, updated).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mock.ExpectExec(deleteOverridesQuery).WithArgs(old.ID).WillReturnError(errDB)
				wantErr = errDB
			case "history":
				expectUpdate(mock, updated).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mock.ExpectExec(deleteOverridesQuery).WithArgs(old.ID).WillReturnResult(pgxmock.NewResult("DELETE", 0))
				expectRead(mock, saved)
				expectHistory(mock, old.ID, "update", &old, &saved, updated.UpdatedBy.ID).WillReturnError(errDB)
				wantErr = errDB
			}
			mock.ExpectRollback()

			got, err := repo.UpdateKey(context.Background(), updated)

			assert.ErrorIs(t, err, wantErr)
			assert.Empty(t, got)
		})
	}
}

func TestDeleteKey(t *testing.T) {
	for _, failHistory := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "history error"}[failHistory], func(t *testing.T) {
			key := newFlag()
			actor := uuid.NewV4()
			mock, repo := newMock(t)
			mock.ExpectBegin()
			expectLock(mock, key)
			expectRead(mock, key)
			mock.ExpectExec(deleteKeyQuery).WithArgs(key.ID, key.Version).WillReturnResult(pgxmock.NewResult("DELETE", 1))
			history := expectHistory(mock, key.ID, "delete", &key, nil, actor)
			if failHistory {
				history.WillReturnError(errDB)
				mock.ExpectRollback()
			} else {
				history.WillReturnResult(pgxmock.NewResult("INSERT", 1))
				expectRevision(mock)
				mock.ExpectCommit()
			}

			err := repo.DeleteKey(context.Background(), key.ID, key.Version, actor)

			if failHistory {
				assert.ErrorIs(t, err, errDB)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestReadKeys(t *testing.T) {
	mock, repo := newMock(t)
	config, flag := newConfig(), newFlag()
	mock.ExpectQuery(readKeysQuery).WithArgs("attachments", "backend", "").
		WillReturnRows(pgxmock.NewRows(keyColumns).AddRow(keyRow(config)...).AddRow(keyRow(flag)...)).RowsWillBeClosed()

	keys, err := repo.ReadKeys(context.Background(), "attachments", "backend", "")

	require.NoError(t, err)
	assert.Equal(t, []models.OnlineConfKey{config, flag}, keys)
}

func TestReadKeyNotFound(t *testing.T) {
	mock, repo := newMock(t)
	id := uuid.NewV4()
	mock.ExpectQuery(readKeyQuery).WithArgs(id).WillReturnError(pgx.ErrNoRows)

	_, err := repo.ReadKey(context.Background(), id)

	assert.ErrorIs(t, err, models.ErrOnlineConfKeyNotFound)
}

func TestReadSnapshot(t *testing.T) {
	for _, count := range []int{0, 201} {
		name := "empty revision zero"
		if count > 0 {
			name = "all keys beyond catalog limit"
		}
		t.Run(name, func(t *testing.T) {
			mock, repo := newMock(t)
			rows := pgxmock.NewRows(keyColumns)
			want := make([]models.OnlineConfKey, 0, count)
			for i := 0; i < count; i++ {
				key := newConfig()
				want = append(want, key)
				rows.AddRow(keyRow(key)...)
			}
			mock.ExpectBegin()
			mock.ExpectExec(setSnapshotTransactionQuery).WillReturnResult(pgxmock.NewResult("SET", 0))
			mock.ExpectQuery(readRevisionQuery).WillReturnRows(pgxmock.NewRows([]string{"revision"}).AddRow(int64(count)))
			mock.ExpectQuery(readSnapshotKeysQuery).WillReturnRows(rows).RowsWillBeClosed()
			mock.ExpectCommit()

			snapshot, err := repo.ReadSnapshot(context.Background())

			require.NoError(t, err)
			assert.Equal(t, models.Snapshot{Revision: int64(count), Keys: want}, snapshot)
		})
	}
}

func TestReadSnapshotRollback(t *testing.T) {
	for _, stage := range []string{"isolation", "revision", "keys", "rows"} {
		t.Run(stage, func(t *testing.T) {
			mock, repo := newMock(t)
			mock.ExpectBegin()
			if stage == "isolation" {
				mock.ExpectExec(setSnapshotTransactionQuery).WillReturnError(errDB)
			} else {
				mock.ExpectExec(setSnapshotTransactionQuery).WillReturnResult(pgxmock.NewResult("SET", 0))
				if stage == "revision" {
					mock.ExpectQuery(readRevisionQuery).WillReturnError(errDB)
				} else {
					mock.ExpectQuery(readRevisionQuery).WillReturnRows(pgxmock.NewRows([]string{"revision"}).AddRow(int64(42)))
					if stage == "keys" {
						mock.ExpectQuery(readSnapshotKeysQuery).WillReturnError(errDB)
					} else {
						mock.ExpectQuery(readSnapshotKeysQuery).
							WillReturnRows(pgxmock.NewRows(keyColumns).AddRow(keyRow(newConfig())...).RowError(0, errDB)).RowsWillBeClosed()
					}
				}
			}
			mock.ExpectRollback()

			snapshot, err := repo.ReadSnapshot(context.Background())

			assert.ErrorIs(t, err, errDB)
			assert.Empty(t, snapshot)
		})
	}
}

func TestReadHistory(t *testing.T) {
	key := newFlag()
	keyJSON, err := json.Marshal(key)
	require.NoError(t, err)
	actor := models.OnlineConfUser{ID: uuid.NewV4(), Login: "seyran"}
	createdID, deletedID := uuid.NewV4(), uuid.NewV4()
	deletedAt := key.UpdatedAt.Add(time.Hour)
	mock, repo := newMock(t)
	mock.ExpectQuery(readHistoryQuery).WithArgs(key.ID).WillReturnRows(pgxmock.NewRows(historyColumns).
		AddRow(deletedID.String(), key.ID.String(), "delete", keyJSON, nil, actor.ID.String(), actor.Login, deletedAt).
		AddRow(createdID.String(), key.ID.String(), "create", nil, keyJSON, key.UpdatedBy.ID.String(), key.UpdatedBy.Login, key.UpdatedAt)).RowsWillBeClosed()

	history, err := repo.ReadHistory(context.Background(), key.ID)

	require.NoError(t, err)
	assert.Equal(t, []models.OnlineConfHistory{
		{ID: deletedID, KeyID: key.ID, Action: "delete", OldValue: &key, User: actor, CreatedAt: deletedAt},
		{ID: createdID, KeyID: key.ID, Action: "create", NewValue: &key, User: key.UpdatedBy, CreatedAt: key.UpdatedAt},
	}, history)
}

func TestReadEmptyCollections(t *testing.T) {
	mock, repo := newMock(t)
	id := uuid.NewV4()
	mock.ExpectQuery(readKeysQuery).WithArgs("", "", "").WillReturnRows(pgxmock.NewRows(keyColumns))
	mock.ExpectQuery(readHistoryQuery).WithArgs(id).WillReturnRows(pgxmock.NewRows(historyColumns))

	keys, err := repo.ReadKeys(context.Background(), "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, keys)
	assert.Empty(t, keys)
	history, err := repo.ReadHistory(context.Background(), id)
	require.NoError(t, err)
	assert.NotNil(t, history)
	assert.Empty(t, history)
}
