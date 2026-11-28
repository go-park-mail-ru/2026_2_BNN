package grpc

import (
	"context"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/notes"
	"bnn/internal/pkg/notes/delivery/grpc/gen"
	"bnn/internal/pkg/notes/mocks"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandler(t *testing.T) {
	ctx := context.Background()
	userID := uuid.NewV4()
	noteID := uuid.NewV4()
	blockID := uuid.NewV4()
	tagID := uuid.NewV4()
	note := models.Note{ID: noteID, CreatedBy: userID, Title: "Учёба"}
	noteReq := &gen.NoteRequest{UserId: userID.String(), NoteId: noteID.String()}
	badNoteReq := &gen.NoteRequest{UserId: userID.String(), NoteId: "bad"}
	rawBlockID := blockID.String()
	bad := "bad"

	tests := []struct {
		name    string
		prepare func(uc *mocks.MockUsecase)
		call    func(h *Handler) error
		code    codes.Code
	}{
		{
			name: "get note ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetNote(gomock.Any(), userID, noteID).Return(note, nil)
			},
			call: func(h *Handler) error {
				_, err := h.GetNote(ctx, noteReq)
				return err
			},
		},
		{
			name:    "get note bad user",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.GetNote(ctx, &gen.NoteRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "get note not found",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetNote(gomock.Any(), userID, noteID).Return(models.Note{}, models.ErrNoteNotFound)
			},
			call: func(h *Handler) error {
				_, err := h.GetNote(ctx, noteReq)
				return err
			},
			code: codes.NotFound,
		},
		{
			name: "get notes ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetNotes(gomock.Any(), userID, int64(100), int64(0)).Return([]models.Note{note}, nil)
			},
			call: func(h *Handler) error {
				_, err := h.GetNotes(ctx, &gen.GetNotesRequest{UserId: userID.String(), Limit: 100})
				return err
			},
		},
		{
			name:    "get notes bad user",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.GetNotes(ctx, &gen.GetNotesRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "get notes internal",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetNotes(gomock.Any(), userID, int64(100), int64(0)).Return(nil, assert.AnError)
			},
			call: func(h *Handler) error {
				_, err := h.GetNotes(ctx, &gen.GetNotesRequest{UserId: userID.String(), Limit: 100})
				return err
			},
			code: codes.Internal,
		},
		{
			name: "create note ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().CreateNote(gomock.Any(), userID, notes.CreateNoteRequest{Title: "Учёба"}).Return(note, nil)
			},
			call: func(h *Handler) error {
				_, err := h.CreateNote(ctx, &gen.CreateNoteRequest{UserId: userID.String(), Title: "Учёба"})
				return err
			},
		},
		{
			name:    "create note bad user",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.CreateNote(ctx, &gen.CreateNoteRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name:    "create note bad parent",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.CreateNote(ctx, &gen.CreateNoteRequest{UserId: userID.String(), ParentNoteId: &bad})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "create note invalid title",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().CreateNote(gomock.Any(), userID, notes.CreateNoteRequest{}).Return(models.Note{}, models.ErrValidation)
			},
			call: func(h *Handler) error {
				_, err := h.CreateNote(ctx, &gen.CreateNoteRequest{UserId: userID.String()})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "update note ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UpdateNote(gomock.Any(), userID, noteID, notes.UpdateNoteRequest{
					Title:  "Учёба",
					Blocks: []notes.BlockInput{{ID: &blockID, Content: []byte(`{"type":"text"}`)}},
					TagIDs: []uuid.UUID{tagID},
				}).Return(note, nil)
			},
			call: func(h *Handler) error {
				_, err := h.UpdateNote(ctx, &gen.UpdateNoteRequest{
					UserId: userID.String(),
					NoteId: noteID.String(),
					Title:  "Учёба",
					Blocks: []*gen.BlockInput{{Id: &rawBlockID, Content: []byte(`{"type":"text"}`)}},
					TagIds: &gen.TagIDs{Ids: []string{tagID.String()}},
				})
				return err
			},
		},
		{
			name:    "update note bad user",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.UpdateNote(ctx, &gen.UpdateNoteRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name:    "update note bad note",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.UpdateNote(ctx, &gen.UpdateNoteRequest{UserId: userID.String(), NoteId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name:    "update note bad block",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.UpdateNote(ctx, &gen.UpdateNoteRequest{
					UserId: userID.String(),
					NoteId: noteID.String(),
					Blocks: []*gen.BlockInput{{Id: &bad}},
				})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name:    "update note bad tag",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.UpdateNote(ctx, &gen.UpdateNoteRequest{
					UserId: userID.String(),
					NoteId: noteID.String(),
					TagIds: &gen.TagIDs{Ids: []string{"bad"}},
				})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "update note forbidden",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UpdateNote(gomock.Any(), userID, noteID, notes.UpdateNoteRequest{Blocks: []notes.BlockInput{}}).
					Return(models.Note{}, models.ErrForbidden)
			},
			call: func(h *Handler) error {
				_, err := h.UpdateNote(ctx, &gen.UpdateNoteRequest{UserId: userID.String(), NoteId: noteID.String()})
				return err
			},
			code: codes.PermissionDenied,
		},
		{
			name: "delete note ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().DeleteNote(gomock.Any(), userID, noteID).Return(nil)
			},
			call: func(h *Handler) error {
				_, err := h.DeleteNote(ctx, noteReq)
				return err
			},
		},
		{
			name:    "delete note bad note",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.DeleteNote(ctx, badNoteReq)
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "delete note forbidden",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().DeleteNote(gomock.Any(), userID, noteID).Return(models.ErrForbidden)
			},
			call: func(h *Handler) error {
				_, err := h.DeleteNote(ctx, noteReq)
				return err
			},
			code: codes.PermissionDenied,
		},
		{
			name: "add favorite ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().AddFavorite(gomock.Any(), userID, noteID).Return(nil)
			},
			call: func(h *Handler) error {
				_, err := h.AddFavorite(ctx, noteReq)
				return err
			},
		},
		{
			name:    "add favorite bad note",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.AddFavorite(ctx, badNoteReq)
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "add favorite not found",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().AddFavorite(gomock.Any(), userID, noteID).Return(models.ErrNoteNotFound)
			},
			call: func(h *Handler) error {
				_, err := h.AddFavorite(ctx, noteReq)
				return err
			},
			code: codes.NotFound,
		},
		{
			name: "remove favorite ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().RemoveFavorite(gomock.Any(), userID, noteID).Return(nil)
			},
			call: func(h *Handler) error {
				_, err := h.RemoveFavorite(ctx, noteReq)
				return err
			},
		},
		{
			name:    "remove favorite bad note",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.RemoveFavorite(ctx, badNoteReq)
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "remove favorite internal",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().RemoveFavorite(gomock.Any(), userID, noteID).Return(assert.AnError)
			},
			call: func(h *Handler) error {
				_, err := h.RemoveFavorite(ctx, noteReq)
				return err
			},
			code: codes.Internal,
		},
		{
			name: "get favorites ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetFavorites(gomock.Any(), userID).Return([]models.Note{note}, nil)
			},
			call: func(h *Handler) error {
				_, err := h.GetFavorites(ctx, &gen.UserRequest{UserId: userID.String()})
				return err
			},
		},
		{
			name:    "get favorites bad user",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.GetFavorites(ctx, &gen.UserRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "get favorites internal",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetFavorites(gomock.Any(), userID).Return(nil, assert.AnError)
			},
			call: func(h *Handler) error {
				_, err := h.GetFavorites(ctx, &gen.UserRequest{UserId: userID.String()})
				return err
			},
			code: codes.Internal,
		},
		{
			name: "updated at ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetNoteUpdatedAt(gomock.Any(), noteID).Return(time.Now(), nil)
			},
			call: func(h *Handler) error {
				_, err := h.GetNoteUpdatedAt(ctx, &gen.GetNoteUpdatedAtRequest{NoteId: noteID.String()})
				return err
			},
		},
		{
			name:    "updated at bad note",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.GetNoteUpdatedAt(ctx, &gen.GetNoteUpdatedAtRequest{NoteId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "updated at not found",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetNoteUpdatedAt(gomock.Any(), noteID).Return(time.Time{}, models.ErrNoteNotFound)
			},
			call: func(h *Handler) error {
				_, err := h.GetNoteUpdatedAt(ctx, &gen.GetNoteUpdatedAtRequest{NoteId: noteID.String()})
				return err
			},
			code: codes.NotFound,
		},
		{
			name: "check access ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().CheckAccess(gomock.Any(), userID, noteID).Return("editor", nil)
			},
			call: func(h *Handler) error {
				_, err := h.CheckAccess(ctx, &gen.CheckAccessRequest{UserId: userID.String(), NoteId: noteID.String()})
				return err
			},
		},
		{
			name:    "check access bad user",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.CheckAccess(ctx, &gen.CheckAccessRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name:    "check access bad note",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.CheckAccess(ctx, &gen.CheckAccessRequest{UserId: userID.String(), NoteId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "check access forbidden",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().CheckAccess(gomock.Any(), userID, noteID).Return("", models.ErrForbidden)
			},
			call: func(h *Handler) error {
				_, err := h.CheckAccess(ctx, &gen.CheckAccessRequest{UserId: userID.String(), NoteId: noteID.String()})
				return err
			},
			code: codes.PermissionDenied,
		},
		{
			name: "add collaborator ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().AddCollaborator(gomock.Any(), userID, noteID, "liza", "editor").Return(nil)
			},
			call: func(h *Handler) error {
				_, err := h.AddCollaborator(ctx, &gen.AddCollaboratorRequest{UserId: userID.String(), NoteId: noteID.String(), Login: "liza", Role: "editor"})
				return err
			},
		},
		{
			name:    "add collaborator bad user",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.AddCollaborator(ctx, &gen.AddCollaboratorRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name:    "add collaborator bad note",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.AddCollaborator(ctx, &gen.AddCollaboratorRequest{UserId: userID.String(), NoteId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "add collaborator bad role",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().AddCollaborator(gomock.Any(), userID, noteID, "liza", "owner").Return(models.ErrValidation)
			},
			call: func(h *Handler) error {
				_, err := h.AddCollaborator(ctx, &gen.AddCollaboratorRequest{UserId: userID.String(), NoteId: noteID.String(), Login: "liza", Role: "owner"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "remove collaborator ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().RemoveCollaborator(gomock.Any(), userID, noteID, tagID).Return(nil)
			},
			call: func(h *Handler) error {
				_, err := h.RemoveCollaborator(ctx, &gen.RemoveCollaboratorRequest{UserId: userID.String(), NoteId: noteID.String(), TargetUserId: tagID.String()})
				return err
			},
		},
		{
			name:    "remove collaborator bad user",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.RemoveCollaborator(ctx, &gen.RemoveCollaboratorRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name:    "remove collaborator bad note",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.RemoveCollaborator(ctx, &gen.RemoveCollaboratorRequest{UserId: userID.String(), NoteId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name:    "remove collaborator bad target",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.RemoveCollaborator(ctx, &gen.RemoveCollaboratorRequest{UserId: userID.String(), NoteId: noteID.String(), TargetUserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "remove collaborator forbidden",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().RemoveCollaborator(gomock.Any(), userID, noteID, tagID).Return(models.ErrForbidden)
			},
			call: func(h *Handler) error {
				_, err := h.RemoveCollaborator(ctx, &gen.RemoveCollaboratorRequest{UserId: userID.String(), NoteId: noteID.String(), TargetUserId: tagID.String()})
				return err
			},
			code: codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			err := tt.call(NewHandler(uc))
			assert.Equal(t, tt.code, status.Code(err))
		})
	}
}
