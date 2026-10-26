package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"bnn/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	uuid "github.com/satori/go.uuid"
)

const tagNameConstraint = "tags_created_by_name_key"

//go:embed queries/read_tags.sql
var readTagsQuery string

//go:embed queries/read_tag.sql
var readTagQuery string

//go:embed queries/create_tag.sql
var createTagQuery string

//go:embed queries/update_tag.sql
var updateTagQuery string

//go:embed queries/delete_tag.sql
var deleteTagQuery string

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

func (r *Repo) ReadTags(ctx context.Context, userID uuid.UUID) ([]models.Tag, error) {
	rows, err := r.db.Query(ctx, readTagsQuery, userID)
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

func (r *Repo) ReadTag(ctx context.Context, id uuid.UUID) (models.Tag, error) {
	var tag models.Tag

	err := r.db.QueryRow(ctx, readTagQuery, id).Scan(&tag.ID, &tag.CreatedBy, &tag.Name, &tag.Color, &tag.CreatedAt, &tag.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Tag{}, models.ErrTagNotFound
	}
	if err != nil {
		return models.Tag{}, fmt.Errorf("read tag: %w", err)
	}

	return tag, nil
}

func (r *Repo) CreateTag(ctx context.Context, tag models.Tag) error {
	_, err := r.db.Exec(ctx, createTagQuery, tag.ID, tag.CreatedBy, tag.Name, tag.Color, tag.CreatedAt, tag.UpdatedAt)
	if isNameTaken(err) {
		return models.ErrTagExists
	}
	if err != nil {
		return fmt.Errorf("create tag: %w", err)
	}

	return nil
}

func (r *Repo) UpdateTag(ctx context.Context, tag models.Tag) error {
	_, err := r.db.Exec(ctx, updateTagQuery, tag.Name, tag.Color, tag.UpdatedAt, tag.ID)
	if isNameTaken(err) {
		return models.ErrTagExists
	}
	if err != nil {
		return fmt.Errorf("update tag: %w", err)
	}

	return nil
}

func (r *Repo) DeleteTag(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, deleteTagQuery, id)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}

	return nil
}

func isNameTaken(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.ConstraintName == tagNameConstraint
}
