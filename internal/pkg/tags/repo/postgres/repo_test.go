package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"bnn/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errDB      = errors.New("db is down")
	tagColumns = []string{"id", "created_by", "name", "color", "created_at", "updated_at"}
)

func newMock(t *testing.T) (pgxmock.PgxPoolIface, *Repo) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	return mock, NewRepo(mock)
}

func newTag() models.Tag {
	now := time.Now().UTC()
	return models.Tag{
		ID:        uuid.NewV4(),
		CreatedBy: uuid.NewV4(),
		Name:      "учёба",
		Color:     "#FF880080",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func tagRow(tag models.Tag) []any {
	return []any{tag.ID.String(), tag.CreatedBy.String(), tag.Name, tag.Color, tag.CreatedAt, tag.UpdatedAt}
}

func TestReadTags(t *testing.T) {
	tag := newTag()

	t.Run("ok", func(t *testing.T) {
		mock, repo := newMock(t)
		mock.ExpectQuery(readTagsQuery).
			WithArgs(tag.CreatedBy).
			WillReturnRows(pgxmock.NewRows(tagColumns).AddRow(tagRow(tag)...))

		tags, err := repo.ReadTags(context.Background(), tag.CreatedBy)

		require.NoError(t, err)
		assert.Equal(t, []models.Tag{tag}, tags)
	})

	t.Run("no tags", func(t *testing.T) {
		mock, repo := newMock(t)
		mock.ExpectQuery(readTagsQuery).
			WithArgs(tag.CreatedBy).
			WillReturnRows(pgxmock.NewRows(tagColumns))

		tags, err := repo.ReadTags(context.Background(), tag.CreatedBy)

		require.NoError(t, err)
		assert.NotNil(t, tags)
		assert.Empty(t, tags)
	})

	t.Run("query error", func(t *testing.T) {
		mock, repo := newMock(t)
		mock.ExpectQuery(readTagsQuery).WithArgs(tag.CreatedBy).WillReturnError(errDB)

		_, err := repo.ReadTags(context.Background(), tag.CreatedBy)

		assert.ErrorIs(t, err, errDB)
	})

	t.Run("scan error", func(t *testing.T) {
		mock, repo := newMock(t)
		mock.ExpectQuery(readTagsQuery).
			WithArgs(tag.CreatedBy).
			WillReturnRows(pgxmock.NewRows(tagColumns).AddRow("not-uuid", tag.CreatedBy.String(), tag.Name, tag.Color, tag.CreatedAt, tag.UpdatedAt))

		_, err := repo.ReadTags(context.Background(), tag.CreatedBy)

		assert.Error(t, err)
	})

	t.Run("rows error", func(t *testing.T) {
		mock, repo := newMock(t)
		mock.ExpectQuery(readTagsQuery).
			WithArgs(tag.CreatedBy).
			WillReturnRows(pgxmock.NewRows(tagColumns).AddRow(tagRow(tag)...).RowError(0, errDB))

		_, err := repo.ReadTags(context.Background(), tag.CreatedBy)

		assert.ErrorIs(t, err, errDB)
	})
}

func TestReadTag(t *testing.T) {
	tag := newTag()

	tests := []struct {
		name        string
		prepare     func(mock pgxmock.PgxPoolIface)
		expectedErr error
	}{
		{
			name: "ok",
			prepare: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(readTagQuery).
					WithArgs(tag.ID).
					WillReturnRows(pgxmock.NewRows(tagColumns).AddRow(tagRow(tag)...))
			},
		},
		{
			name: "not found",
			prepare: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(readTagQuery).WithArgs(tag.ID).WillReturnRows(pgxmock.NewRows(tagColumns))
			},
			expectedErr: models.ErrTagNotFound,
		},
		{
			name: "db error",
			prepare: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(readTagQuery).WithArgs(tag.ID).WillReturnError(errDB)
			},
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, repo := newMock(t)
			tt.prepare(mock)

			got, err := repo.ReadTag(context.Background(), tag.ID)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tag, got)
		})
	}
}

func TestCreateTag(t *testing.T) {
	tag := newTag()

	tests := []struct {
		name        string
		dbErr       error
		expectedErr error
	}{
		{name: "ok"},
		{name: "name taken", dbErr: &pgconn.PgError{Code: "23505", ConstraintName: tagNameConstraint}, expectedErr: models.ErrTagExists},
		{name: "db error", dbErr: errDB, expectedErr: errDB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, repo := newMock(t)
			exec := mock.ExpectExec(createTagQuery).
				WithArgs(tag.ID, tag.CreatedBy, tag.Name, tag.Color, tag.CreatedAt, tag.UpdatedAt)
			if tt.dbErr != nil {
				exec.WillReturnError(tt.dbErr)
			} else {
				exec.WillReturnResult(pgxmock.NewResult("INSERT", 1))
			}

			err := repo.CreateTag(context.Background(), tag)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestUpdateTag(t *testing.T) {
	tag := newTag()

	tests := []struct {
		name        string
		dbErr       error
		expectedErr error
	}{
		{name: "ok"},
		{name: "name taken", dbErr: &pgconn.PgError{Code: "23505", ConstraintName: tagNameConstraint}, expectedErr: models.ErrTagExists},
		{name: "other unique error", dbErr: &pgconn.PgError{Code: "23505", ConstraintName: "tags_pkey"}},
		{name: "db error", dbErr: errDB, expectedErr: errDB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, repo := newMock(t)
			exec := mock.ExpectExec(updateTagQuery).WithArgs(tag.Name, tag.Color, tag.UpdatedAt, tag.ID)
			if tt.dbErr != nil {
				exec.WillReturnError(tt.dbErr)
			} else {
				exec.WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			}

			err := repo.UpdateTag(context.Background(), tag)

			switch {
			case tt.expectedErr != nil:
				assert.ErrorIs(t, err, tt.expectedErr)
			case tt.dbErr != nil:
				assert.Error(t, err)
				assert.NotErrorIs(t, err, models.ErrTagExists)
			default:
				require.NoError(t, err)
			}
		})
	}
}

func TestDeleteTag(t *testing.T) {
	id := uuid.NewV4()

	t.Run("ok", func(t *testing.T) {
		mock, repo := newMock(t)
		mock.ExpectExec(deleteTagQuery).WithArgs(id).WillReturnResult(pgxmock.NewResult("DELETE", 1))

		require.NoError(t, repo.DeleteTag(context.Background(), id))
	})

	t.Run("db error", func(t *testing.T) {
		mock, repo := newMock(t)
		mock.ExpectExec(deleteTagQuery).WithArgs(id).WillReturnError(errDB)

		assert.ErrorIs(t, repo.DeleteTag(context.Background(), id), errDB)
	})
}
