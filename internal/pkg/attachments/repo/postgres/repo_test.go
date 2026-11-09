package postgres_test

import (
	"context"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/attachments/repo/postgres"

	"github.com/pashagolub/pgxmock/v3"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachmentRepo(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockDB.Close()

	repo := postgres.NewRepo(mockDB)
	attID := uuid.NewV4()
	blockID := uuid.NewV4()
	userID := uuid.NewV4()

	t.Run("ReadBlockOwner", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"created_by"}).AddRow(userID)
		mockDB.ExpectQuery(`SELECT n.created_by FROM blocks b JOIN notes n`).
			WithArgs(blockID).
			WillReturnRows(rows)

		owner, err := repo.ReadBlockOwner(context.Background(), blockID)
		require.NoError(t, err)
		assert.Equal(t, userID, owner)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("CreateAttachment", func(t *testing.T) {
		att := models.Attachment{ID: attID, BlockID: blockID, Path: "/path"}
		mockDB.ExpectExec(`INSERT INTO attachments`).
			WithArgs(attID, blockID, "/path").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.CreateAttachment(context.Background(), att)
		require.NoError(t, err)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("ReadAttachment", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"id", "block_id", "path"}).AddRow(attID, blockID, "/path")
		mockDB.ExpectQuery(`SELECT id, block_id, path FROM attachments`).
			WithArgs(attID).
			WillReturnRows(rows)

		att, err := repo.ReadAttachment(context.Background(), attID)
		require.NoError(t, err)
		assert.Equal(t, "/path", att.Path)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("DeleteAttachment", func(t *testing.T) {
		mockDB.ExpectExec(`DELETE FROM attachments`).
			WithArgs(attID).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.DeleteAttachment(context.Background(), attID)
		require.NoError(t, err)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})
}
