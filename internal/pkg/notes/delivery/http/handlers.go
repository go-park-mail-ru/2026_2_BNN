package delivery

import (
	"context"
	"encoding/json"
	"errors"

	"bnn/internal/models"
	"bnn/internal/pkg/ctxutil"
	"bnn/internal/pkg/notes"
	"bnn/internal/pkg/notes/delivery/http/api"

	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
)

type Handler struct {
	uc notes.Usecase
}

func NewHandler(uc notes.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) ListNotes(ctx context.Context, req api.ListNotesRequestObject) (api.ListNotesResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.ListNotes401Response{}, nil
	}

	limit := int64(10)
	if req.Params.Limit != nil {
		limit = int64(*req.Params.Limit)
		if limit < 1 || limit > 100 {
			return api.ListNotes400Response{}, nil
		}
	}

	offset := int64(0)
	if req.Params.Offset != nil {
		offset = int64(*req.Params.Offset)
		if offset < 0 {
			return api.ListNotes400Response{}, nil
		}
	}

	notesList, err := h.uc.GetNotes(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	apiNotes := make([]api.Note, 0, len(notesList))
	for _, n := range notesList {
		apiNotes = append(apiNotes, toAPINote(n))
	}
	return api.ListNotes200JSONResponse(apiNotes), nil
}

func (h *Handler) GetNote(ctx context.Context, req api.GetNoteRequestObject) (api.GetNoteResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.GetNote401Response{}, nil
	}

	note, err := h.uc.GetNote(ctx, userID, uuid.UUID(req.Id))
	if errors.Is(err, models.ErrNoteNotFound) {
		return api.GetNote404Response{}, nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetNote200JSONResponse(toAPINote(note)), nil
}

func (h *Handler) CreateNote(ctx context.Context, req api.CreateNoteRequestObject) (api.CreateNoteResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.CreateNote401Response{}, nil
	}

	var parentNoteID *uuid.UUID
	if req.Body.ParentNoteId != nil {
		id := uuid.UUID(*req.Body.ParentNoteId)
		parentNoteID = &id
	}

	title := ""
	if req.Body.Title != nil {
		title = *req.Body.Title
	}

	note, err := h.uc.CreateNote(ctx, userID, notes.CreateNoteRequest{
		Title:        title,
		ParentNoteID: parentNoteID,
		Header:       req.Body.Header,
		Icon:         req.Body.Icon,
	})
	if errors.Is(err, models.ErrNoteNotFound) {
		return api.CreateNote404Response{}, nil
	}
	if errors.Is(err, models.ErrValidation) {
		return api.CreateNote400Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.CreateNote201JSONResponse(toAPINote(note)), nil
}

func (h *Handler) UpdateNote(ctx context.Context, req api.UpdateNoteRequestObject) (api.UpdateNoteResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.UpdateNote401Response{}, nil
	}

	blocks := make([]notes.BlockInput, 0, len(req.Body.Blocks))
	for _, b := range req.Body.Blocks {
		var id *uuid.UUID
		if b.Id != nil {
			uid := uuid.UUID(*b.Id)
			id = &uid
		}
		contentBytes, _ := json.Marshal(b.Content)
		blocks = append(blocks, notes.BlockInput{ID: id, Content: contentBytes})
	}

	note, err := h.uc.UpdateNote(ctx, userID, uuid.UUID(req.Id), notes.UpdateNoteRequest{
		Title:  req.Body.Title,
		Header: req.Body.Header,
		Icon:   req.Body.Icon,
		Blocks: blocks,
	})
	if err != nil {
		return nil, err
	}
	return api.UpdateNote200JSONResponse(toAPINote(note)), nil
}

func (h *Handler) DeleteNote(ctx context.Context, req api.DeleteNoteRequestObject) (api.DeleteNoteResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.DeleteNote401Response{}, nil
	}

	err := h.uc.DeleteNote(ctx, userID, uuid.UUID(req.Id))
	if errors.Is(err, models.ErrNoteNotFound) {
		return api.DeleteNote404Response{}, nil
	}
	if err != nil {
		return nil, err
	}
	return api.DeleteNote204Response{}, nil
}

func toAPINote(n models.Note) api.Note {
	var blocks *[]api.Block
	if n.Blocks != nil {
		apiBlocks := make([]api.Block, 0, len(n.Blocks))
		for _, b := range n.Blocks {
			apiBlocks = append(apiBlocks, toAPIBlock(b))
		}
		blocks = &apiBlocks
	}

	var parentNoteID *openapi_types.UUID
	if n.ParentNoteID != nil {
		id := openapi_types.UUID(*n.ParentNoteID)
		parentNoteID = &id
	}

	var header *string
	if n.Header != "" {
		header = &n.Header
	}
	var icon *string
	if n.Icon != "" {
		icon = &n.Icon
	}

	return api.Note{
		Id:           openapi_types.UUID(n.ID),
		ParentNoteId: parentNoteID,
		CreatedBy:    openapi_types.UUID(n.CreatedBy),
		Title:        n.Title,
		Header:       header,
		Icon:         icon,
		Blocks:       blocks,
		CreatedAt:    n.CreatedAt,
		UpdatedAt:    n.UpdatedAt,
	}
}

func toAPIBlock(b models.Block) api.Block {
	var content map[string]interface{}
	_ = json.Unmarshal(b.Content, &content)
	return api.Block{
		Id:        openapi_types.UUID(b.ID),
		Content:   content,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
}
