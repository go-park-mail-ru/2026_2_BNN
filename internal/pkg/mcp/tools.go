package mcp

import (
	"context"
	"errors"
	"log/slog"

	"bnn/internal/models"
	"bnn/internal/pkg/notes"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/samber/lo"
	uuid "github.com/satori/go.uuid"
)

const (
	defaultNotesLimit = 20
	maxNotesLimit     = 100
)

var (
	errUnauthenticated = errors.New("пользователь не аутентифицирован")
	errInvalidArgs     = errors.New("неверные параметры")
	errInvalidPage     = errors.New("limit должен быть от 1 до 100, offset не меньше 0")
	errInvalidNoteID   = errors.New("note_id должен быть uuid заметки из list_notes")
	errNoteNotFound    = errors.New("заметка не найдена")
	errForbidden       = errors.New("нет доступа к заметке")
	errInternal        = errors.New("внутренняя ошибка, попробуйте позже")
)

type listNotesArgs struct {
	Limit  int64 `json:"limit,omitempty" jsonschema:"сколько заметок вернуть, от 1 до 100, по умолчанию 20"`
	Offset int64 `json:"offset,omitempty" jsonschema:"сколько заметок пропустить с начала списка, по умолчанию 0"`
}

type getNoteArgs struct {
	NoteID string `json:"note_id" jsonschema:"id заметки из list_notes"`
}

func registerNoteTools(server *mcp.Server, uc notes.Usecase) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_notes",
		Description: "список заметок пользователя: название, id, id родителя и дата изменения. Сначала самые свежие. Используй, чтобы найти нужную заметку, а потом прочитать её через get_note",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, listNotes(uc))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_note",
		Description: "содержимое заметки: название, теги и блоки по порядку. Пункты чек-листа показаны как [ ] (не сделано) и [x] (сделано), у каждого указан id блока",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, getNote(uc))
}

func listNotes(uc notes.Usecase) mcp.ToolHandlerFor[listNotesArgs, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, args listNotesArgs) (*mcp.CallToolResult, any, error) {
		userID, err := userIDFromRequest(req)
		if err != nil {
			return nil, nil, err
		}

		limit := lo.CoalesceOrEmpty(args.Limit, defaultNotesLimit)
		if limit < 1 || limit > maxNotesLimit || args.Offset < 0 {
			return nil, nil, errInvalidPage
		}

		list, err := uc.GetNotes(ctx, userID, limit, args.Offset)
		if err != nil {
			return nil, nil, toolError(ctx, err)
		}

		return textResult(formatNotes(list)), nil, nil
	}
}

func getNote(uc notes.Usecase) mcp.ToolHandlerFor[getNoteArgs, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, args getNoteArgs) (*mcp.CallToolResult, any, error) {
		userID, err := userIDFromRequest(req)
		if err != nil {
			return nil, nil, err
		}

		noteID, err := uuid.FromString(args.NoteID)
		if err != nil {
			return nil, nil, errInvalidNoteID
		}

		note, err := uc.GetNote(ctx, userID, noteID)
		if err != nil {
			return nil, nil, toolError(ctx, err)
		}

		return textResult(formatNote(note)), nil, nil
	}
}

func userIDFromRequest(req *mcp.CallToolRequest) (uuid.UUID, error) {
	if req.Extra == nil || req.Extra.TokenInfo == nil {
		return uuid.Nil, errUnauthenticated
	}

	userID, err := uuid.FromString(req.Extra.TokenInfo.UserID)
	if err != nil {
		return uuid.Nil, errUnauthenticated
	}

	return userID, nil
}

func toolError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, models.ErrNoteNotFound):
		return errNoteNotFound
	case errors.Is(err, models.ErrForbidden):
		return errForbidden
	case errors.Is(err, models.ErrValidation):
		return errInvalidArgs
	default:
		slog.ErrorContext(ctx, "MCP tool failed", "error", err)
		return errInternal
	}
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}
