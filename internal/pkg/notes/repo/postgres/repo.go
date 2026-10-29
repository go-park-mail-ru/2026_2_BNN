package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"bnn/internal/models"
	"bnn/internal/pkg/notes"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	uuid "github.com/satori/go.uuid"
)

const (
	foreignKeyViolationCode = "23503"
	parentNoteForeignKey    = "note_parent_note_id_fkey"
)

//go:embed queries/read_note.sql
var readNoteQuery string

//go:embed queries/read_notes.sql
var readNotesQuery string

//go:embed queries/read_blocks.sql
var readBlocksQuery string

//go:embed queries/read_note_block_ids.sql
var readNoteBlockIDsQuery string

//go:embed queries/read_editable_note.sql
var readEditableNoteQuery string

//go:embed queries/create_note.sql
var createNoteQuery string

//go:embed queries/create_owner_member.sql
var createOwnerMemberQuery string

//go:embed queries/create_chat.sql
var createChatQuery string

//go:embed queries/update_note.sql
var updateNoteQuery string

//go:embed queries/create_block.sql
var createBlockQuery string

//go:embed queries/update_block.sql
var updateBlockQuery string

//go:embed queries/delete_note_blocks.sql
var deleteNoteBlocksQuery string

//go:embed queries/create_note_block.sql
var createNoteBlockQuery string

//go:embed queries/delete_orphan_blocks.sql
var deleteOrphanBlocksQuery string

//go:embed queries/lock_own_subtree.sql
var lockOwnSubtreeQuery string

//go:embed queries/read_notes_block_ids.sql
var readNotesBlockIDsQuery string

//go:embed queries/delete_note.sql
var deleteNoteQuery string

// Заметка и её блоки читаются двумя запросами. В одном снимке REPEATABLE READ
// они согласованы: иначе между запросами может пройти чужое сохранение,
// и клиент получит блоки одной версии вместе с номером другой.
var readTxOptions = pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}

// Запись начинается с блокировки строки заметки (UPDATE или SELECT FOR UPDATE),
// поэтому все изменения одной заметки идут по очереди. READ COMMITTED достаточно.
var writeTxOptions = pgx.TxOptions{IsoLevel: pgx.ReadCommitted}

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

func (r *Repo) ReadNoteWithBlocks(ctx context.Context, userID, noteID uuid.UUID) (models.Note, error) {
	var note models.Note

	err := pgx.BeginTxFunc(ctx, r.db, readTxOptions, func(tx pgx.Tx) error {
		var err error

		note, err = scanNote(tx.QueryRow(ctx, readNoteQuery, noteID, userID))
		if err != nil {
			return err
		}

		note.Blocks, err = readBlocks(ctx, tx, noteID)
		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Note{}, models.ErrNoteNotFound
	}
	if err != nil {
		return models.Note{}, fmt.Errorf("read note with blocks: %w", err)
	}

	return note, nil
}

func (r *Repo) ReadNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error) {
	rows, err := r.db.Query(ctx, readNotesQuery, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("read notes: %w", err)
	}
	defer rows.Close()

	notesList := make([]models.Note, 0)
	for rows.Next() {
		note, err := scanNote(rows)
		if err != nil {
			return nil, fmt.Errorf("read notes: %w", err)
		}
		notesList = append(notesList, note)
	}

	return notesList, rows.Err()
}

// CreateNote создаёт заметку, членство автора с ролью editor и чат заметки
// одной транзакцией, как требует relations.md. Если родитель указан,
// пользователь должен быть его editor, иначе ErrNoteNotFound.
func (r *Repo) CreateNote(ctx context.Context, note models.Note) error {
	err := pgx.BeginTxFunc(ctx, r.db, writeTxOptions, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, createNoteQuery,
			note.ID, note.ParentNoteID, note.CreatedBy,
			note.Title, note.Header, note.Icon,
			note.CreatedAt, note.UpdatedAt,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return models.ErrNoteNotFound
		}

		batch := &pgx.Batch{}
		batch.Queue(createOwnerMemberQuery, note.ID, note.CreatedBy)
		batch.Queue(createChatQuery, note.ID)

		return tx.SendBatch(ctx, batch).Close()
	})
	if errors.Is(err, models.ErrNoteNotFound) {
		return err
	}
	if err != nil {
		// Родителя удалили во время вставки.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolationCode && pgErr.ConstraintName == parentNoteForeignKey {
			return models.ErrNoteNotFound
		}

		return fmt.Errorf("create note: %w", err)
	}

	return nil
}

// UpdateNote сохраняет заметку целиком одной транзакцией.
// req.Version — версия, которую видел клиент: если в базе уже другая, вернётся ErrNoteConflict.
// Блок без ID создаётся, с ID обновляется, блоки заметки, которых нет в req.Blocks, удаляются.
func (r *Repo) UpdateNote(ctx context.Context, userID, noteID uuid.UUID, req notes.UpdateNoteRequest) (models.Note, error) {
	var note models.Note

	err := pgx.BeginTxFunc(ctx, r.db, writeTxOptions, func(tx pgx.Tx) error {
		var err error

		// Оптимистическая блокировка: проверяет версию и права и блокирует строку заметки.
		note, err = scanNote(tx.QueryRow(ctx, updateNoteQuery,
			req.Title, req.Header, req.Icon,
			noteID, req.Version, userID,
		))
		if errors.Is(err, pgx.ErrNoRows) {
			return conflictOrNotFound(ctx, tx, userID, noteID)
		}
		if err != nil {
			return err
		}

		// Строка заметки заблокирована, состав её блоков никто не меняет.
		oldIDs, err := readNoteBlockIDs(ctx, tx, note.ID)
		if err != nil {
			return err
		}

		batch := &pgx.Batch{}
		blockIDs := make([]uuid.UUID, 0, len(req.Blocks))

		for _, b := range req.Blocks {
			if b.ID == nil {
				id := uuid.NewV4()
				batch.Queue(createBlockQuery, id, b.Content, userID)
				blockIDs = append(blockIDs, id)
				continue
			}

			// Чужой блок или повтор id в запросе.
			if _, ok := oldIDs[*b.ID]; !ok {
				return fmt.Errorf("%w: block %s is not in note", models.ErrValidation, *b.ID)
			}
			delete(oldIDs, *b.ID)

			batch.Queue(updateBlockQuery, *b.ID, b.Content, userID)
			blockIDs = append(blockIDs, *b.ID)
		}

		// Порядок переписывается целиком: по одному обновлять position
		// не даёт UNIQUE (note_id, position).
		batch.Queue(deleteNoteBlocksQuery, note.ID)
		for position, id := range blockIDs {
			batch.Queue(createNoteBlockQuery, note.ID, id, position)
		}

		if len(oldIDs) > 0 {
			removedIDs := make([]uuid.UUID, 0, len(oldIDs))
			for id := range oldIDs {
				removedIDs = append(removedIDs, id)
			}
			batch.Queue(deleteOrphanBlocksQuery, removedIDs)
		}

		if err := tx.SendBatch(ctx, batch).Close(); err != nil {
			return err
		}

		// Время создания и изменения блоков ставит БД, поэтому в ответ — то, что записано.
		note.Blocks, err = readBlocks(ctx, tx, note.ID)
		return err
	})
	if errors.Is(err, models.ErrNoteNotFound) || errors.Is(err, models.ErrNoteConflict) || errors.Is(err, models.ErrValidation) {
		return models.Note{}, err
	}
	if err != nil {
		return models.Note{}, fmt.Errorf("update note: %w", err)
	}

	return note, nil
}

// DeleteNote удаляет заметку владельца вместе со всеми вложенными заметками
// (ON DELETE CASCADE) и блоки поддерева, которые больше нигде не используются.
func (r *Repo) DeleteNote(ctx context.Context, userID, noteID uuid.UUID) error {
	err := pgx.BeginTxFunc(ctx, r.db, writeTxOptions, func(tx pgx.Tx) error {
		// Блокируем всё поддерево: иначе параллельное сохранение вложенной
		// заметки добавит блоки, пока мы собираем их список.
		noteIDs, err := lockOwnSubtree(ctx, tx, userID, noteID)
		if err != nil {
			return err
		}
		if len(noteIDs) == 0 {
			return models.ErrNoteNotFound
		}

		blockIDs, err := readNotesBlockIDs(ctx, tx, noteIDs)
		if err != nil {
			return err
		}

		batch := &pgx.Batch{}
		batch.Queue(deleteNoteQuery, noteID, userID)
		batch.Queue(deleteOrphanBlocksQuery, blockIDs)

		return tx.SendBatch(ctx, batch).Close()
	})
	if errors.Is(err, models.ErrNoteNotFound) {
		return err
	}
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}

	return nil
}

func lockOwnSubtree(ctx context.Context, tx pgx.Tx, userID, noteID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := tx.Query(ctx, lockOwnSubtreeQuery, noteID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func readNotesBlockIDs(ctx context.Context, tx pgx.Tx, noteIDs []uuid.UUID) ([]uuid.UUID, error) {
	rows, err := tx.Query(ctx, readNotesBlockIDsQuery, noteIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func conflictOrNotFound(ctx context.Context, tx pgx.Tx, userID, noteID uuid.UUID) error {
	var version int

	err := tx.QueryRow(ctx, readEditableNoteQuery, noteID, userID).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ErrNoteNotFound
	}
	if err != nil {
		return err
	}

	return models.ErrNoteConflict
}

func readNoteBlockIDs(ctx context.Context, tx pgx.Tx, noteID uuid.UUID) (map[uuid.UUID]struct{}, error) {
	rows, err := tx.Query(ctx, readNoteBlockIDsQuery, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[uuid.UUID]struct{})
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = struct{}{}
	}

	return ids, rows.Err()
}

func readBlocks(ctx context.Context, tx pgx.Tx, noteID uuid.UUID) ([]models.Block, error) {
	rows, err := tx.Query(ctx, readBlocksQuery, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := make([]models.Block, 0)
	for rows.Next() {
		var block models.Block
		if err := rows.Scan(&block.ID, &block.Content, &block.CreatedAt, &block.UpdatedAt); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, rows.Err()
}

func scanNote(row pgx.Row) (models.Note, error) {
	var note models.Note

	err := row.Scan(
		&note.ID, &note.ParentNoteID, &note.CreatedBy,
		&note.Title, &note.Header, &note.Icon,
		&note.Version, &note.CreatedAt, &note.UpdatedAt,
	)

	return note, err
}
