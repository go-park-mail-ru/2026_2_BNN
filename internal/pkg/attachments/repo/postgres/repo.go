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

//go:embed queries/can_edit_block.sql
var canEditBlockQuery string

//go:embed queries/lock_editable_block.sql
var lockEditableBlockQuery string

//go:embed queries/create_attachment.sql
var createAttachmentQuery string

//go:embed queries/create_block_attachment.sql
var createBlockAttachmentQuery string

//go:embed queries/read_attachment_path.sql
var readAttachmentPathQuery string

//go:embed queries/delete_attachment.sql
var deleteAttachmentQuery string

// Транзакция создания начинается с блокировки строки блока,
// поэтому READ COMMITTED достаточно.
var txOptions = pgx.TxOptions{IsoLevel: pgx.ReadCommitted}

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

func (r *Repo) CanEditBlock(ctx context.Context, userID, blockID uuid.UUID) (bool, error) {
	var ok bool

	if err := r.db.QueryRow(ctx, canEditBlockQuery, blockID, userID).Scan(&ok); err != nil {
		return false, fmt.Errorf("can edit block: %w", err)
	}

	return ok, nil
}

// CreateAttachment создаёт вложение и привязывает его к блоку последним.
// FOR NO KEY UPDATE ставит параллельные загрузки в один блок в очередь:
// иначе обе посчитают одну и ту же следующую position и упрутся в UNIQUE.
// Права проверяются здесь же, под блокировкой.
func (r *Repo) CreateAttachment(ctx context.Context, userID uuid.UUID, attachment models.Attachment) error {
	err := pgx.BeginTxFunc(ctx, r.db, txOptions, func(tx pgx.Tx) error {
		var blockID uuid.UUID
		if err := tx.QueryRow(ctx, lockEditableBlockQuery, attachment.BlockID, userID).Scan(&blockID); err != nil {
			return err
		}

		batch := &pgx.Batch{}
		batch.Queue(createAttachmentQuery, attachment.ID, attachment.Path)
		batch.Queue(createBlockAttachmentQuery, attachment.BlockID, attachment.ID)

		return tx.SendBatch(ctx, batch).Close()
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ErrBlockNotFound
	}
	if err != nil {
		return fmt.Errorf("create attachment: %w", err)
	}

	return nil
}

func (r *Repo) ReadAttachmentPath(ctx context.Context, userID, attachmentID uuid.UUID) (string, error) {
	var path string

	err := r.db.QueryRow(ctx, readAttachmentPathQuery, attachmentID, userID).Scan(&path)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", models.ErrAttachmentNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read attachment: %w", err)
	}

	return path, nil
}

// DeleteAttachment удаляет вложение, если пользователь редактор заметки с этим блоком.
// Проверка и удаление — один запрос. Возвращает путь файла в S3.
func (r *Repo) DeleteAttachment(ctx context.Context, userID, attachmentID uuid.UUID) (string, error) {
	var path string

	err := r.db.QueryRow(ctx, deleteAttachmentQuery, attachmentID, userID).Scan(&path)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", models.ErrAttachmentNotFound
	}
	if err != nil {
		return "", fmt.Errorf("delete attachment: %w", err)
	}

	return path, nil
}
