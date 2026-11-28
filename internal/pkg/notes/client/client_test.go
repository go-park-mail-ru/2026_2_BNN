package client

import (
	"context"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/notes"
	notesgrpc "bnn/internal/pkg/notes/delivery/grpc"
	"bnn/internal/pkg/notes/delivery/grpc/gen"
	"bnn/internal/pkg/notes/mocks"
	"bnn/internal/pkg/utils/grpcutil/grpctest"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

func newClient(t *testing.T) (*Client, *mocks.MockUsecase) {
	uc := mocks.NewMockUsecase(gomock.NewController(t))
	conn := grpctest.Dial(t, func(server *grpc.Server) {
		gen.RegisterNotesServer(server, notesgrpc.NewHandler(uc))
	})
	return NewClient(conn), uc
}

func TestClient(t *testing.T) {
	ctx := context.Background()
	client, uc := newClient(t)

	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	userID := uuid.NewV4()
	parentID := uuid.NewV4()
	blockID := uuid.NewV4()
	tagID := uuid.NewV4()
	note := models.Note{ID: uuid.NewV4(), ParentNoteID: &parentID, CreatedBy: userID, Title: "Учёба", CreatedAt: now, UpdatedAt: now}
	fullNote := note
	fullNote.Blocks = []models.Block{}
	fullNote.Tags = []models.Tag{}

	uc.EXPECT().GetNote(gomock.Any(), userID, note.ID).Return(note, nil)
	got, err := client.GetNote(ctx, userID, note.ID)
	require.NoError(t, err)
	assert.Equal(t, fullNote, got)

	uc.EXPECT().GetNotes(gomock.Any(), userID, int64(100), int64(0)).Return([]models.Note{note}, nil)
	list, err := client.GetNotes(ctx, userID, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, []models.Note{note}, list)

	createReq := notes.CreateNoteRequest{Title: "Учёба", ParentNoteID: &parentID}
	uc.EXPECT().CreateNote(gomock.Any(), userID, createReq).Return(note, nil)
	got, err = client.CreateNote(ctx, userID, createReq)
	require.NoError(t, err)
	assert.Equal(t, note, got)

	updateReq := notes.UpdateNoteRequest{
		Title:  "Учёба",
		Blocks: []notes.BlockInput{{ID: &blockID, Content: []byte(`{"type":"text"}`)}},
		TagIDs: []uuid.UUID{tagID},
	}
	uc.EXPECT().UpdateNote(gomock.Any(), userID, note.ID, updateReq).Return(note, nil)
	got, err = client.UpdateNote(ctx, userID, note.ID, updateReq)
	require.NoError(t, err)
	assert.Equal(t, fullNote, got)

	uc.EXPECT().DeleteNote(gomock.Any(), userID, note.ID).Return(nil)
	assert.NoError(t, client.DeleteNote(ctx, userID, note.ID))

	uc.EXPECT().AddFavorite(gomock.Any(), userID, note.ID).Return(nil)
	assert.NoError(t, client.AddFavorite(ctx, userID, note.ID))

	uc.EXPECT().RemoveFavorite(gomock.Any(), userID, note.ID).Return(nil)
	assert.NoError(t, client.RemoveFavorite(ctx, userID, note.ID))

	uc.EXPECT().GetFavorites(gomock.Any(), userID).Return([]models.Note{note}, nil)
	list, err = client.GetFavorites(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, []models.Note{note}, list)

	uc.EXPECT().GetNoteUpdatedAt(gomock.Any(), note.ID).Return(now, nil)
	updatedAt, err := client.GetNoteUpdatedAt(ctx, note.ID)
	require.NoError(t, err)
	assert.Equal(t, now, updatedAt)

	uc.EXPECT().CheckAccess(gomock.Any(), userID, note.ID).Return("viewer", nil)
	role, err := client.CheckAccess(ctx, userID, note.ID)
	require.NoError(t, err)
	assert.Equal(t, "viewer", role)

	uc.EXPECT().AddCollaborator(gomock.Any(), userID, note.ID, "liza", "editor").Return(nil)
	assert.NoError(t, client.AddCollaborator(ctx, userID, note.ID, "liza", "editor"))

	uc.EXPECT().RemoveCollaborator(gomock.Any(), userID, note.ID, tagID).Return(nil)
	assert.NoError(t, client.RemoveCollaborator(ctx, userID, note.ID, tagID))
}

func TestClientErrors(t *testing.T) {
	ctx := context.Background()
	client, uc := newClient(t)
	userID := uuid.NewV4()
	noteID := uuid.NewV4()

	uc.EXPECT().GetNote(gomock.Any(), userID, noteID).Return(models.Note{}, models.ErrNoteNotFound)
	_, err := client.GetNote(ctx, userID, noteID)
	assert.ErrorIs(t, err, models.ErrNoteNotFound)

	uc.EXPECT().GetNotes(gomock.Any(), userID, int64(0), int64(0)).Return(nil, models.ErrValidation)
	_, err = client.GetNotes(ctx, userID, 0, 0)
	assert.ErrorIs(t, err, models.ErrValidation)

	uc.EXPECT().CreateNote(gomock.Any(), userID, notes.CreateNoteRequest{}).Return(models.Note{}, models.ErrValidation)
	_, err = client.CreateNote(ctx, userID, notes.CreateNoteRequest{})
	assert.ErrorIs(t, err, models.ErrValidation)

	uc.EXPECT().UpdateNote(gomock.Any(), userID, noteID, notes.UpdateNoteRequest{Blocks: []notes.BlockInput{}}).
		Return(models.Note{}, models.ErrForbidden)
	_, err = client.UpdateNote(ctx, userID, noteID, notes.UpdateNoteRequest{})
	assert.ErrorIs(t, err, models.ErrForbidden)

	uc.EXPECT().GetFavorites(gomock.Any(), userID).Return(nil, models.ErrValidation)
	_, err = client.GetFavorites(ctx, userID)
	assert.ErrorIs(t, err, models.ErrValidation)

	uc.EXPECT().GetNoteUpdatedAt(gomock.Any(), noteID).Return(time.Time{}, models.ErrNoteNotFound)
	_, err = client.GetNoteUpdatedAt(ctx, noteID)
	assert.ErrorIs(t, err, models.ErrNoteNotFound)

	uc.EXPECT().CheckAccess(gomock.Any(), userID, noteID).Return("", models.ErrForbidden)
	_, err = client.CheckAccess(ctx, userID, noteID)
	assert.ErrorIs(t, err, models.ErrForbidden)
}
