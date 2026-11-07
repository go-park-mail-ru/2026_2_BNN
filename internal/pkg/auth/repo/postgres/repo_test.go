package postgres_test

import (
	"context"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/auth/repo/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v3"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthRepo(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockDB.Close()

	repo := postgres.NewRepo(mockDB)
	userID := uuid.NewV4()
	now := time.Now().UTC()

	t.Run("CreateUser - Success", func(t *testing.T) {
		user := models.User{
			ID: userID, Login: "test", PasswordHash: []byte("hash"),
			Avatar: "avatar.png", Email: "test@test.com", IsExternal: false,
			TOTPSecret: "", Is2FAEnabled: false, CreatedAt: now, UpdatedAt: now,
		}

		mockDB.ExpectExec(`INSERT INTO users`).
			WithArgs(
				user.ID, user.Login, user.PasswordHash, user.Avatar, user.Email,
				user.IsExternal, user.TOTPSecret, user.Is2FAEnabled,
				user.CreatedAt, user.UpdatedAt,
			).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.CreateUser(context.Background(), user)
		assert.NoError(t, err)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("CreateUser - Duplicate (409)", func(t *testing.T) {
		user := models.User{ID: userID, Login: "test"}
		pgErr := &pgconn.PgError{Code: "23505"}

		mockDB.ExpectExec(`INSERT INTO users`).
			WithArgs(
				user.ID, user.Login, user.PasswordHash, user.Avatar, user.Email,
				user.IsExternal, user.TOTPSecret, user.Is2FAEnabled,
				user.CreatedAt, user.UpdatedAt,
			).
			WillReturnError(pgErr)

		err := repo.CreateUser(context.Background(), user)
		assert.ErrorIs(t, err, models.ErrUserExists)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("ReadUserByID - Not Found", func(t *testing.T) {
		mockDB.ExpectQuery(`SELECT`).
			WithArgs(userID).
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.ReadUserByID(context.Background(), userID)
		assert.ErrorIs(t, err, models.ErrUserNotFound)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("Set2FAStatus", func(t *testing.T) {
		mockDB.ExpectExec(`UPDATE users SET is_2fa_enabled`).
			WithArgs(true, userID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Set2FAStatus(context.Background(), userID, true)
		assert.NoError(t, err)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("UpdateAvatar", func(t *testing.T) {
		user := models.User{ID: userID, Avatar: "/new.png", UpdatedAt: now}
		mockDB.ExpectExec(`UPDATE users SET avatar`).
			WithArgs("/new.png", now, userID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.UpdateAvatar(context.Background(), user)
		assert.NoError(t, err)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})
}
