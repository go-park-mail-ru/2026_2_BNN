package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/auth/repo/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessions(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockDB.Close()

	repo := postgres.NewRepo(mockDB)
	ctx := context.Background()
	sessionID := uuid.NewV4()
	userID := uuid.NewV4()
	hash := []byte("hash")
	now := time.Now().UTC()
	errDB := errors.New("db is down")

	t.Run("CreateRefreshToken", func(t *testing.T) {
		mockDB.ExpectExec(`INSERT INTO refresh_tokens`).
			WithArgs(sessionID, hash, userID, now, now).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.CreateRefreshToken(ctx, models.RefreshToken{ID: sessionID, TokenHash: hash, UserID: userID, ExpiresAt: now, CreatedAt: now})
		assert.NoError(t, err)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("RevokeRefreshToken", func(t *testing.T) {
		mockDB.ExpectQuery(`UPDATE refresh_tokens`).
			WithArgs(hash).
			WillReturnRows(pgxmock.NewRows([]string{"id", "token_hash", "user_id", "expires_at", "created_at"}).
				AddRow(sessionID, hash, userID, now, now))

		token, err := repo.RevokeRefreshToken(ctx, hash)
		require.NoError(t, err)
		assert.Equal(t, sessionID, token.ID)
		assert.Equal(t, userID, token.UserID)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("RevokeRefreshToken - not found or already revoked", func(t *testing.T) {
		mockDB.ExpectQuery(`UPDATE refresh_tokens`).WithArgs(hash).WillReturnError(pgx.ErrNoRows)

		_, err := repo.RevokeRefreshToken(ctx, hash)
		assert.ErrorIs(t, err, models.ErrRefreshTokenNotFound)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("RevokeRefreshToken - db error", func(t *testing.T) {
		mockDB.ExpectQuery(`UPDATE refresh_tokens`).WithArgs(hash).WillReturnError(errDB)

		_, err := repo.RevokeRefreshToken(ctx, hash)
		assert.ErrorIs(t, err, errDB)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("ReadRefreshToken - revoked", func(t *testing.T) {
		revokedAt := now
		mockDB.ExpectQuery(`SELECT id, token_hash`).
			WithArgs(hash).
			WillReturnRows(pgxmock.NewRows([]string{"id", "token_hash", "user_id", "expires_at", "created_at", "revoked_at"}).
				AddRow(sessionID, hash, userID, now, now, &revokedAt))

		token, err := repo.ReadRefreshToken(ctx, hash)
		require.NoError(t, err)
		require.NotNil(t, token.RevokedAt)
		assert.Equal(t, userID, token.UserID)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("ReadRefreshToken - not found", func(t *testing.T) {
		mockDB.ExpectQuery(`SELECT id, token_hash`).WithArgs(hash).WillReturnError(pgx.ErrNoRows)

		_, err := repo.ReadRefreshToken(ctx, hash)
		assert.ErrorIs(t, err, models.ErrRefreshTokenNotFound)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("ReadRefreshToken - db error", func(t *testing.T) {
		mockDB.ExpectQuery(`SELECT id, token_hash`).WithArgs(hash).WillReturnError(errDB)

		_, err := repo.ReadRefreshToken(ctx, hash)
		assert.ErrorIs(t, err, errDB)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("IsSessionActive", func(t *testing.T) {
		mockDB.ExpectQuery(`SELECT EXISTS`).
			WithArgs(sessionID).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

		active, err := repo.IsSessionActive(ctx, sessionID)
		require.NoError(t, err)
		assert.True(t, active)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("IsSessionActive - db error", func(t *testing.T) {
		mockDB.ExpectQuery(`SELECT EXISTS`).WithArgs(sessionID).WillReturnError(errDB)

		_, err := repo.IsSessionActive(ctx, sessionID)
		assert.ErrorIs(t, err, errDB)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})
}
