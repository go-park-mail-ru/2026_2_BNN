package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"bnn/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	uuid "github.com/satori/go.uuid"
)

//go:embed queries/read_note.sql
var readNoteQuery string

//go:embed queries/read_notes.sql
var readNotesQuery string

//go:embed queries/read_blocks.sql
var readBlocksQuery string

//go:embed queries/create_note.sql
var createNoteQuery string

//go:embed queries/update_note.sql
var updateNoteQuery string

//go:embed queries/delete_note.sql
var deleteNoteQuery string

//go:embed queries/create_block.sql
var createBlockQuery string

//go:embed queries/update_block.sql
var updateBlockQuery string

//go:embed queries/delete_blocks.sql
var deleteBlocksQuery string

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

func (r *Repo) ReadNote(ctx context.Context, id uuid.UUID) (models.Note, error) {
	var note models.Note
	err := r.db.QueryRow(ctx, readNoteQuery, id).Scan(
		&note.ID, &note.ParentNoteID, &note.CreatedBy, &note.Title,
		&note.Header, &note.Icon, &note.CreatedAt, &note.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Note{}, models.ErrNoteNotFound
	}
	return note, err
}

func (r *Repo) ReadNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error) {
	rows, err := r.db.Query(ctx, readNotesQuery, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := make([]models.Note, 0)
	for rows.Next() {
		var note models.Note
		err := rows.Scan(&note.ID, &note.ParentNoteID, &note.CreatedBy, &note.Title, &note.Header, &note.Icon, &note.CreatedAt, &note.UpdatedAt)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

func (r *Repo) ReadBlocks(ctx context.Context, noteID uuid.UUID) ([]models.Block, error) {
	rows, err := r.db.Query(ctx, readBlocksQuery, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := make([]models.Block, 0)
	for rows.Next() {
		var block models.Block
		err := rows.Scan(&block.ID, &block.Content, &block.CreatedAt, &block.UpdatedAt)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, rows.Err()
}

func (r *Repo) CreateNote(ctx context.Context, note models.Note, blocks []models.Block) error {
	// Собираем ID блоков в массив для колонки block_ids
	blockIDs := make([]uuid.UUID, len(blocks))
	for i, b := range blocks {
		blockIDs[i] = b.ID
	}

	_, err := r.db.Exec(ctx, createNoteQuery, note.ID, note.ParentNoteID, note.CreatedBy, note.Title, note.Header, note.Icon, blockIDs, note.CreatedAt, note.UpdatedAt)
	return err
}

func (r *Repo) UpdateNote(ctx context.Context, note models.Note, blocks []models.Block, deletedBlockIDs []uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Обновляем блоки (вставляем новые или обновляем существующие)
	blockIDs := make([]uuid.UUID, len(blocks))
	for i, b := range blocks {
		blockIDs[i] = b.ID

		// Используем ON CONFLICT для upsert'а (вставить или обновить)
		_, err = tx.Exec(ctx, `
			INSERT INTO blocks (id, note_id, content, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO UPDATE SET content = EXCLUDED.content, updated_at = EXCLUDED.updated_at
		`, b.ID, note.ID, b.Content, b.CreatedAt, b.UpdatedAt)
		if err != nil {
			return fmt.Errorf("upsert block: %w", err)
		}
	}

	// 2. Обновляем саму заметку (с новым порядком в block_ids)
	_, err = tx.Exec(ctx, updateNoteQuery, note.Title, note.Header, note.Icon, blockIDs, note.UpdatedAt, note.ID)
	if err != nil {
		return fmt.Errorf("update note: %w", err)
	}

	// 3. Удаляем блоки, которых больше нет в массиве
	if len(deletedBlockIDs) > 0 {
		_, err = tx.Exec(ctx, deleteBlocksQuery, deletedBlockIDs)
		if err != nil {
			return fmt.Errorf("delete removed blocks: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *Repo) DeleteNote(ctx context.Context, noteID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, deleteBlocksQuery, "DELETE FROM blocks WHERE note_id = $1", noteID) // Костыль, если удаляем всё
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, deleteNoteQuery, noteID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
