package grpc

import (
	"context"

	"bnn/internal/models"
	"bnn/internal/pkg/grpcconv"
	"bnn/internal/pkg/grpcerr"
	"bnn/internal/pkg/notes"
	"bnn/internal/pkg/notes/delivery/grpc/gen"

	"github.com/samber/lo"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	gen.UnimplementedNotesServer
	uc notes.Usecase
}

func NewHandler(uc notes.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) GetNote(ctx context.Context, in *gen.NoteRequest) (*gen.Note, error) {
	userID, noteID, err := parseNoteRequest(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	note, err := h.uc.GetNote(ctx, userID, noteID)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return NoteToProto(note), nil
}

func (h *Handler) GetNotes(ctx context.Context, in *gen.GetNotesRequest) (*gen.NotesResponse, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	list, err := h.uc.GetNotes(ctx, userID, in.GetLimit(), in.GetOffset())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return toNotesResponse(list), nil
}

func (h *Handler) CreateNote(ctx context.Context, in *gen.CreateNoteRequest) (*gen.Note, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	parentNoteID, err := grpcconv.ParseOptionalUUID(in.ParentNoteId)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	note, err := h.uc.CreateNote(ctx, userID, notes.CreateNoteRequest{
		Title:        in.GetTitle(),
		ParentNoteID: parentNoteID,
		Header:       in.Header,
		Icon:         in.Icon,
	})
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return NoteToProto(note), nil
}

func (h *Handler) UpdateNote(ctx context.Context, in *gen.UpdateNoteRequest) (*gen.Note, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	noteID, err := grpcconv.ParseUUID(in.GetNoteId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	blocks := make([]notes.BlockInput, 0, len(in.GetBlocks()))
	for _, b := range in.GetBlocks() {
		id, err := grpcconv.ParseOptionalUUID(b.Id)
		if err != nil {
			return nil, grpcerr.ToStatus(err)
		}
		blocks = append(blocks, notes.BlockInput{ID: id, Content: b.GetContent()})
	}

	var tagIDs []uuid.UUID
	if in.GetTagIds() != nil {
		tagIDs = make([]uuid.UUID, 0, len(in.GetTagIds().GetIds()))
		for _, raw := range in.GetTagIds().GetIds() {
			id, err := grpcconv.ParseUUID(raw)
			if err != nil {
				return nil, grpcerr.ToStatus(err)
			}
			tagIDs = append(tagIDs, id)
		}
	}

	note, err := h.uc.UpdateNote(ctx, userID, noteID, notes.UpdateNoteRequest{
		Title:  in.GetTitle(),
		Header: in.Header,
		Icon:   in.Icon,
		Blocks: blocks,
		TagIDs: tagIDs,
	})
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return NoteToProto(note), nil
}

func (h *Handler) DeleteNote(ctx context.Context, in *gen.NoteRequest) (*emptypb.Empty, error) {
	userID, noteID, err := parseNoteRequest(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	if err := h.uc.DeleteNote(ctx, userID, noteID); err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) AddFavorite(ctx context.Context, in *gen.NoteRequest) (*emptypb.Empty, error) {
	userID, noteID, err := parseNoteRequest(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	if err := h.uc.AddFavorite(ctx, userID, noteID); err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) RemoveFavorite(ctx context.Context, in *gen.NoteRequest) (*emptypb.Empty, error) {
	userID, noteID, err := parseNoteRequest(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	if err := h.uc.RemoveFavorite(ctx, userID, noteID); err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) GetFavorites(ctx context.Context, in *gen.UserRequest) (*gen.NotesResponse, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	list, err := h.uc.GetFavorites(ctx, userID)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return toNotesResponse(list), nil
}

func parseNoteRequest(in *gen.NoteRequest) (uuid.UUID, uuid.UUID, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	noteID, err := grpcconv.ParseUUID(in.GetNoteId())
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	return userID, noteID, nil
}

func toNotesResponse(list []models.Note) *gen.NotesResponse {
	return &gen.NotesResponse{
		Notes: lo.Map(list, func(n models.Note, _ int) *gen.Note { return NoteToProto(n) }),
	}
}

func (h *Handler) GetNoteUpdatedAt(ctx context.Context, req *gen.GetNoteUpdatedAtRequest) (*gen.GetNoteUpdatedAtResponse, error) {
	noteID, err := uuid.FromString(req.NoteId)
	if err != nil {
		return nil, grpcerr.ToStatus(models.ErrValidation)
	}

	updatedAt, err := h.uc.GetNoteUpdatedAt(ctx, noteID)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &gen.GetNoteUpdatedAtResponse{
		UpdatedAt: timestamppb.New(updatedAt),
	}, nil
}

func (h *Handler) CheckAccess(ctx context.Context, req *gen.CheckAccessRequest) (*gen.CheckAccessResponse, error) {
	userID, err := uuid.FromString(req.UserId)
	if err != nil {
		return nil, grpcerr.ToStatus(models.ErrValidation)
	}
	noteID, err := uuid.FromString(req.NoteId)
	if err != nil {
		return nil, grpcerr.ToStatus(models.ErrValidation)
	}

	role, err := h.uc.CheckAccess(ctx, userID, noteID)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &gen.CheckAccessResponse{Role: role}, nil
}

func (h *Handler) AddCollaborator(ctx context.Context, req *gen.AddCollaboratorRequest) (*emptypb.Empty, error) {
	userID, err := uuid.FromString(req.UserId)
	if err != nil {
		return nil, grpcerr.ToStatus(models.ErrValidation)
	}
	noteID, err := uuid.FromString(req.NoteId)
	if err != nil {
		return nil, grpcerr.ToStatus(models.ErrValidation)
	}

	err = h.uc.AddCollaborator(ctx, userID, noteID, req.TargetUserId, req.Role)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) RemoveCollaborator(ctx context.Context, req *gen.RemoveCollaboratorRequest) (*emptypb.Empty, error) {
	userID, err := uuid.FromString(req.UserId)
	if err != nil {
		return nil, grpcerr.ToStatus(models.ErrValidation)
	}
	noteID, err := uuid.FromString(req.NoteId)
	if err != nil {
		return nil, grpcerr.ToStatus(models.ErrValidation)
	}
	targetID, err := uuid.FromString(req.TargetUserId)
	if err != nil {
		return nil, grpcerr.ToStatus(models.ErrValidation)
	}

	err = h.uc.RemoveCollaborator(ctx, userID, noteID, targetID)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &emptypb.Empty{}, nil
}
