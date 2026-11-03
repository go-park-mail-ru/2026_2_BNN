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

//go:embed queries/upsert_block.sql
var upsertBlockQuery string

//go:embed queries/read_note_tags.sql
var readNoteTagsQuery string

//go:embed queries/read_user_tags.sql
var readUserTagsQuery string

//go:embed queries/delete_note_tags.sql
var deleteNoteTagsQuery string

//go:embed queries/create_note_tags.sql
var createNoteTagsQuery string

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
	blockIDs := make([]uuid.UUID, len(blocks))
	for i, b := range blocks {
		blockIDs[i] = b.ID
	}

	_, err := r.db.Exec(ctx, createNoteQuery, note.ID, note.ParentNoteID, note.CreatedBy, note.Title, note.Header, note.Icon, blockIDs, note.CreatedAt, note.UpdatedAt)
	return err
}

func (r *Repo) ReadNoteTags(ctx context.Context, noteID uuid.UUID) ([]models.Tag, error) {
	return r.readTags(ctx, readNoteTagsQuery, noteID)
}

func (r *Repo) ReadUserTags(ctx context.Context, userID uuid.UUID, tagIDs []uuid.UUID) ([]models.Tag, error) {
	return r.readTags(ctx, readUserTagsQuery, userID, tagIDs)
}

func (r *Repo) readTags(ctx context.Context, query string, args ...any) ([]models.Tag, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read tags: %w", err)
	}
	defer rows.Close()

	tags := make([]models.Tag, 0)
	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(&tag.ID, &tag.CreatedBy, &tag.Name, &tag.Color, &tag.CreatedAt, &tag.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

func (r *Repo) UpdateNote(ctx context.Context, note models.Note, blocks []models.Block, deletedBlockIDs []uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	blockIDs := make([]uuid.UUID, len(blocks))
	for i, b := range blocks {
		blockIDs[i] = b.ID
		_, err = tx.Exec(ctx, upsertBlockQuery, b.ID, note.ID, b.Content, b.CreatedAt, b.UpdatedAt)
		if err != nil {
			return fmt.Errorf("upsert block: %w", err)
		}
	}

	_, err = tx.Exec(ctx, updateNoteQuery, note.Title, note.Header, note.Icon, blockIDs, note.UpdatedAt, note.ID)
	if err != nil {
		return fmt.Errorf("update note: %w", err)
	}

	if len(deletedBlockIDs) > 0 {
		_, err = tx.Exec(ctx, deleteBlocksQuery, deletedBlockIDs)
		if err != nil {
			return fmt.Errorf("delete removed blocks: %w", err)
		}
	}

	tagIDs := make([]uuid.UUID, len(note.Tags))
	for i, t := range note.Tags {
		tagIDs[i] = t.ID
	}

	_, err = tx.Exec(ctx, deleteNoteTagsQuery, note.ID)
	if err != nil {
		return fmt.Errorf("delete note tags: %w", err)
	}

	_, err = tx.Exec(ctx, createNoteTagsQuery, note.ID, tagIDs)
	if err != nil {
		return fmt.Errorf("create note tags: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *Repo) DeleteNote(ctx context.Context, noteID uuid.UUID) error {
	_, err := r.db.Exec(ctx, deleteNoteQuery, noteID)
	return err
}
