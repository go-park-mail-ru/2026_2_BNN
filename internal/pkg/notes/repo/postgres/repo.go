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

//go:embed queries/delete_blocks.sql
var deleteBlocksQuery string

//go:embed queries/create_block.sql
var createBlockQuery string

//go:embed queries/update_user_avatar.sql
var updateUserAvatarQuery string

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

// --- Read ---

func (r *Repo) ReadNote(ctx context.Context, id uuid.UUID) (models.Note, error) {
	var note models.Note

	err := r.db.QueryRow(ctx, readNoteQuery, id).Scan(
		&note.ID,
		&note.ParentNoteID,
		&note.CreatedBy,
		&note.Title,
		&note.Header,
		&note.Icon,
		&note.CreatedAt,
		&note.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Note{}, models.ErrNoteNotFound
	}
	if err != nil {
		return models.Note{}, fmt.Errorf("read note: %w", err)
	}

	return note, nil
}

func (r *Repo) ReadNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error) {
	rows, err := r.db.Query(ctx, readNotesQuery, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("read notes: %w", err)
	}
	defer rows.Close()

	notes := make([]models.Note, 0)

	for rows.Next() {
		var note models.Note

		err := rows.Scan(
			&note.ID,
			&note.ParentNoteID,
			&note.CreatedBy,
			&note.Title,
			&note.Header,
			&note.Icon,
			&note.CreatedAt,
			&note.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}

		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read notes: %w", err)
	}

	return notes, nil
}

func (r *Repo) ReadBlocks(ctx context.Context, noteID uuid.UUID) ([]models.Block, error) {
	rows, err := r.db.Query(ctx, readBlocksQuery, noteID)
	if err != nil {
		return nil, fmt.Errorf("read blocks: %w", err)
	}
	defer rows.Close()

	blocks := make([]models.Block, 0)

	for rows.Next() {
		var block models.Block

		err := rows.Scan(
			&block.ID,
			&block.Content,
			&block.CreatedAt,
			&block.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan block: %w", err)
		}

		blocks = append(blocks, block)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read blocks: %w", err)
	}

	return blocks, nil
}

// --- Create ---

func (r *Repo) CreateNote(ctx context.Context, note models.Note, blocks []models.Block) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, createNoteQuery,
		note.ID,
		note.ParentNoteID,
		note.CreatedBy,
		note.Title,
		note.Header,
		note.Icon,
		note.CreatedAt,
		note.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create note: %w", err)
	}

	for _, b := range blocks {
		_, err = tx.Exec(ctx, createBlockQuery,
			b.ID,
			note.ID,
			b.Content,
			b.CreatedAt,
			b.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("create block: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// --- Update ---

func (r *Repo) UpdateNote(ctx context.Context, note models.Note, blocks []models.Block) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, updateNoteQuery,
		note.Title,
		note.Header,
		note.Icon,
		note.UpdatedAt,
		note.ID,
	)
	if err != nil {
		return fmt.Errorf("update note: %w", err)
	}

	// Если blocks == nil — блоки не трогаем.
	// Если blocks != nil — полностью заменяем.
	if blocks != nil {
		_, err = tx.Exec(ctx, deleteBlocksQuery, note.ID)
		if err != nil {
			return fmt.Errorf("delete old blocks: %w", err)
		}

		for _, b := range blocks {
			_, err = tx.Exec(ctx, createBlockQuery,
				b.ID,
				note.ID,
				b.Content,
				b.CreatedAt,
				b.UpdatedAt,
			)
			if err != nil {
				return fmt.Errorf("create block: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}

// --- Delete ---

func (r *Repo) DeleteNote(ctx context.Context, noteID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, deleteBlocksQuery, noteID)
	if err != nil {
		return fmt.Errorf("delete blocks: %w", err)
	}

	_, err = tx.Exec(ctx, deleteNoteQuery, noteID)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}

	return tx.Commit(ctx)
}
