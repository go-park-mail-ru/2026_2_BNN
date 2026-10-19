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

//go:embed queries/read_block_owner.sql
var readBlockOwnerQuery string

//go:embed queries/create_attachment.sql
var createAttachmentQuery string

//go:embed queries/read_attachment.sql
var readAttachmentQuery string

//go:embed queries/delete_attachment.sql
var deleteAttachmentQuery string

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

func (r *Repo) ReadBlockOwner(ctx context.Context, blockID uuid.UUID) (uuid.UUID, error) {
	var ownerID uuid.UUID

	err := r.db.QueryRow(ctx, readBlockOwnerQuery, blockID).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, models.ErrBlockNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("read block owner: %w", err)
	}

	return ownerID, nil
}

func (r *Repo) CreateAttachment(ctx context.Context, attachment models.Attachment) error {
	_, err := r.db.Exec(ctx, createAttachmentQuery, attachment.ID, attachment.BlockID, attachment.Path)
	if err != nil {
		return fmt.Errorf("create attachment: %w", err)
	}

	return nil
}

func (r *Repo) ReadAttachment(ctx context.Context, id uuid.UUID) (models.Attachment, error) {
	var attachment models.Attachment

	err := r.db.QueryRow(ctx, readAttachmentQuery, id).Scan(
		&attachment.ID,
		&attachment.BlockID,
		&attachment.Path,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Attachment{}, models.ErrAttachmentNotFound
	}
	if err != nil {
		return models.Attachment{}, fmt.Errorf("read attachment: %w", err)
	}

	return attachment, nil
}

func (r *Repo) DeleteAttachment(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, deleteAttachmentQuery, id)
	if err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return models.ErrAttachmentNotFound
	}

	return nil
}
