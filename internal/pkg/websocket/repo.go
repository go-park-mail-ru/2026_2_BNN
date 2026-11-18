package ws

import (
	"context"
	"time"

	"bnn/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
	uuid "github.com/satori/go.uuid"
)

type PostgresRepo struct {
	db *pgxpool.Pool
}

func NewPostgresRepo(db *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) GetUpdatedBlocks(ctx context.Context, noteID uuid.UUID, since time.Time) ([]models.Block, error) {
	const query = `
		SELECT id, content, created_at, updated_at
		FROM blocks
		WHERE note_id = $1 AND updated_at > $2
		ORDER BY updated_at ASC
	`

	rows, err := r.db.Query(ctx, query, noteID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []models.Block
	for rows.Next() {
		var b models.Block
		if err := rows.Scan(&b.ID, &b.Content, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
	}

	return blocks, rows.Err()
}
