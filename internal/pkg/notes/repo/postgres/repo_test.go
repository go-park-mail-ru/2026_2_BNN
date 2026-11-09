package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/notes/repo/postgres"

	"github.com/pashagolub/pgxmock/v3"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateNote(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockDB.Close()

	repo := postgres.NewRepo(mockDB)
	noteID := uuid.NewV4()
	blockID := uuid.NewV4()
	userID := uuid.NewV4()

	now := time.Now().UTC()

	note := models.Note{
		ID:        noteID,
		CreatedBy: userID,
		Title:     "New Note",
		CreatedAt: now,
		UpdatedAt: now,
	}
	blocks := []models.Block{{ID: blockID, Content: []byte(`{"text":"hello"}`), CreatedAt: now, UpdatedAt: now}}

	t.Run("Success", func(t *testing.T) {
		var nilParent *uuid.UUID = nil

		mockDB.ExpectExec(`INSERT INTO notes`).
			WithArgs(noteID, nilParent, userID, note.Title, "", "", []uuid.UUID{blockID}, now, now).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.CreateNote(context.Background(), note, blocks)
		require.NoError(t, err)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("Error Insert Note", func(t *testing.T) {
		var nilParent *uuid.UUID = nil

		mockDB.ExpectExec(`INSERT INTO notes`).
			WithArgs(noteID, nilParent, userID, note.Title, "", "", []uuid.UUID{blockID}, now, now).
			WillReturnError(errors.New("db error"))

		err := repo.CreateNote(context.Background(), note, blocks)
		assert.Error(t, err)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})
}
