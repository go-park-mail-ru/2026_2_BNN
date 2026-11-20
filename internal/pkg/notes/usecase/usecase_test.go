package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/notes"
	"bnn/internal/pkg/notes/mocks"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var errDB = errors.New("db is down")

// Обновленный мок: теперь умеет искать по логину
type mockAuthClient struct {
	exists   bool
	err      error
	userID   uuid.UUID // ID, который вернется при поиске по логину
	loginErr error     // Ошибка, если пользователь не найден
}

func (m *mockAuthClient) CheckUserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	return m.exists, m.err
}

func (m *mockAuthClient) GetUserByLogin(ctx context.Context, login string) (uuid.UUID, error) {
	if m.loginErr != nil {
		return uuid.Nil, m.loginErr
	}
	return m.userID, nil
}

func TestGetNote(t *testing.T) {
	userID := uuid.NewV4()
	noteID := uuid.NewV4()
	own := models.Note{ID: noteID, CreatedBy: userID}
	blocks := []models.Block{{ID: uuid.NewV4()}}
	tags := []models.Tag{{ID: uuid.NewV4(), Name: "учёба"}}

	tests := []struct {
		name        string
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
	}{
		{
			name: "ok",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil)
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil)
				repo.EXPECT().ReadBlocks(gomock.Any(), noteID).Return(blocks, nil)
				repo.EXPECT().ReadNoteTags(gomock.Any(), noteID).Return(tags, nil)
			},
		},
		{
			name: "not found",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(models.Note{}, models.ErrNoteNotFound)
			},
			expectedErr: models.ErrNoteNotFound,
		},
		{
			name: "foreign note (IDOR)",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(models.Note{ID: noteID, CreatedBy: uuid.NewV4()}, nil)
				repo.EXPECT().ReadCollaboratorRole(gomock.Any(), noteID, userID).Return("", models.ErrNoteNotFound)
			},
			expectedErr: models.ErrNoteNotFound,
		},
		{
			name: "read blocks error",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil).Times(2)
				repo.EXPECT().ReadBlocks(gomock.Any(), noteID).Return(nil, errDB)
			},
			expectedErr: errDB,
		},
		{
			name: "read tags error",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil).Times(2)
				repo.EXPECT().ReadBlocks(gomock.Any(), noteID).Return(blocks, nil)
				repo.EXPECT().ReadNoteTags(gomock.Any(), noteID).Return(nil, errDB)
			},
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			note, err := NewUsecase(repo, &mockAuthClient{exists: true}).GetNote(context.Background(), userID, noteID)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, blocks, note.Blocks)
			assert.Equal(t, tags, note.Tags)
		})
	}
}

func TestGetNoteUpdatedAt(t *testing.T) {
	noteID := uuid.NewV4()
	now := time.Now().UTC()

	tests := []struct {
		name        string
		prepare     func(repo *mocks.MockRepo)
		expected    time.Time
		expectedErr error
	}{
		{
			name: "ok",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().GetNoteUpdatedAt(gomock.Any(), noteID).Return(now, nil)
			},
			expected: now,
		},
		{
			name: "error",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().GetNoteUpdatedAt(gomock.Any(), noteID).Return(time.Time{}, errDB)
			},
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			res, err := NewUsecase(repo, &mockAuthClient{}).GetNoteUpdatedAt(context.Background(), noteID)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, res)
		})
	}
}

func TestGetNotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRepo(ctrl)
	userID := uuid.NewV4()
	expected := []models.Note{{ID: uuid.NewV4(), CreatedBy: userID}}

	repo.EXPECT().ReadNotes(gomock.Any(), userID, int64(10), int64(5)).Return(expected, nil)

	list, err := NewUsecase(repo, &mockAuthClient{exists: true}).GetNotes(context.Background(), userID, 10, 5)

	require.NoError(t, err)
	assert.Equal(t, expected, list)
}

func TestCreateNote(t *testing.T) {
	userID := uuid.NewV4()
	parentID := uuid.NewV4()
	header := "Шапка"
	icon := "📒"

	tests := []struct {
		name        string
		req         notes.CreateNoteRequest
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
	}{
		{
			name: "ok",
			req:  notes.CreateNoteRequest{Title: "Заметка", Header: &header, Icon: &icon},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().CreateNote(gomock.Any(), gomock.Any(), gomock.Nil()).Return(nil)
			},
		},
		{
			name: "own parent",
			req:  notes.CreateNoteRequest{Title: "Заметка", ParentNoteID: &parentID},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), parentID).Return(models.Note{ID: parentID, CreatedBy: userID}, nil)
				repo.EXPECT().CreateNote(gomock.Any(), gomock.Any(), gomock.Nil()).Return(nil)
			},
		},
		{
			name: "parent not found",
			req:  notes.CreateNoteRequest{ParentNoteID: &parentID},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), parentID).Return(models.Note{}, models.ErrNoteNotFound)
			},
			expectedErr: models.ErrNoteNotFound,
		},
		{
			name: "foreign parent",
			req:  notes.CreateNoteRequest{ParentNoteID: &parentID},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), parentID).Return(models.Note{ID: parentID, CreatedBy: uuid.NewV4()}, nil)
			},
			expectedErr: models.ErrNoteNotFound,
		},
		{
			name: "read parent error",
			req:  notes.CreateNoteRequest{ParentNoteID: &parentID},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), parentID).Return(models.Note{}, errDB)
			},
			expectedErr: errDB,
		},
		{
			name: "create error",
			req:  notes.CreateNoteRequest{Title: "Заметка"},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().CreateNote(gomock.Any(), gomock.Any(), gomock.Nil()).Return(errDB)
			},
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			note, err := NewUsecase(repo, &mockAuthClient{exists: true}).CreateNote(context.Background(), userID, tt.req)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, note.ID)
			assert.Equal(t, userID, note.CreatedBy)
			assert.Equal(t, tt.req.Title, note.Title)
			assert.Equal(t, tt.req.ParentNoteID, note.ParentNoteID)
			if tt.req.Header != nil {
				assert.Equal(t, header, note.Header)
				assert.Equal(t, icon, note.Icon)
			}
		})
	}
}

func TestUpdateNote(t *testing.T) {
	userID := uuid.NewV4()
	noteID := uuid.NewV4()
	blockID := uuid.NewV4()
	tagID := uuid.NewV4()
	own := models.Note{ID: noteID, CreatedBy: userID}
	tag := models.Tag{ID: tagID, CreatedBy: userID, Name: "учёба"}

	tests := []struct {
		name        string
		req         notes.UpdateNoteRequest
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
		expectedTag []models.Tag
	}{
		{
			name: "set tags",
			req:  notes.UpdateNoteRequest{Title: "t", TagIDs: []uuid.UUID{tagID}},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil).Times(2)
				repo.EXPECT().ReadBlocks(gomock.Any(), noteID).Return(nil, nil)
				repo.EXPECT().ReadUserTags(gomock.Any(), userID, []uuid.UUID{tagID}).Return([]models.Tag{tag}, nil)
				repo.EXPECT().UpdateNote(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedTag: []models.Tag{tag},
		},
		{
			name: "tags not sent keep old",
			req:  notes.UpdateNoteRequest{Title: "t"},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil).Times(2)
				repo.EXPECT().ReadBlocks(gomock.Any(), noteID).Return(nil, nil)
				repo.EXPECT().ReadNoteTags(gomock.Any(), noteID).Return([]models.Tag{tag}, nil)
				repo.EXPECT().UpdateNote(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedTag: []models.Tag{tag},
		},
		{
			name: "foreign tag",
			req:  notes.UpdateNoteRequest{Title: "t", TagIDs: []uuid.UUID{tagID}},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil).Times(2)
				repo.EXPECT().ReadBlocks(gomock.Any(), noteID).Return(nil, nil)
				repo.EXPECT().ReadUserTags(gomock.Any(), userID, []uuid.UUID{tagID}).Return([]models.Tag{}, nil)
			},
			expectedErr: models.ErrTagNotFound,
		},
		{
			name: "read tags error",
			req:  notes.UpdateNoteRequest{Title: "t", TagIDs: []uuid.UUID{tagID}},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil).Times(2)
				repo.EXPECT().ReadBlocks(gomock.Any(), noteID).Return(nil, nil)
				repo.EXPECT().ReadUserTags(gomock.Any(), userID, []uuid.UUID{tagID}).Return(nil, errDB)
			},
			expectedErr: errDB,
		},
		{
			name: "foreign block",
			req:  notes.UpdateNoteRequest{Title: "t", Blocks: []notes.BlockInput{{ID: &blockID}}},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil).Times(2)
				repo.EXPECT().ReadBlocks(gomock.Any(), noteID).Return(nil, nil)
			},
			expectedErr: models.ErrBlockNotFound,
		},
		{
			name: "foreign note (IDOR)",
			req:  notes.UpdateNoteRequest{Title: "t"},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(models.Note{ID: noteID, CreatedBy: uuid.NewV4()}, nil)
				repo.EXPECT().ReadCollaboratorRole(gomock.Any(), noteID, userID).Return("", models.ErrNoteNotFound)
			},
			expectedErr: models.ErrNoteNotFound,
		},
		{
			name: "read note error",
			req:  notes.UpdateNoteRequest{Title: "t"},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(models.Note{}, errDB)
			},
			expectedErr: errDB,
		},
		{
			name: "read blocks error",
			req:  notes.UpdateNoteRequest{Title: "t"},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil).Times(2)
				repo.EXPECT().ReadBlocks(gomock.Any(), noteID).Return(nil, errDB)
			},
			expectedErr: errDB,
		},
		{
			name: "save error",
			req:  notes.UpdateNoteRequest{Title: "t"},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil).Times(2)
				repo.EXPECT().ReadBlocks(gomock.Any(), noteID).Return(nil, nil)
				repo.EXPECT().ReadNoteTags(gomock.Any(), noteID).Return(nil, nil)
				repo.EXPECT().UpdateNote(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errDB)
			},
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			note, err := NewUsecase(repo, &mockAuthClient{exists: true}).UpdateNote(context.Background(), userID, noteID, tt.req)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedTag, note.Tags)
		})
	}
}

func TestUpdateNoteBlocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRepo(ctrl)
	userID := uuid.NewV4()
	noteID := uuid.NewV4()
	keptID := uuid.NewV4()
	removedID := uuid.NewV4()
	header := "Шапка"
	icon := "📒"

	repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(models.Note{ID: noteID, CreatedBy: userID}, nil).Times(2)
	repo.EXPECT().ReadBlocks(gomock.Any(), noteID).Return([]models.Block{{ID: keptID}, {ID: removedID}}, nil)
	repo.EXPECT().ReadNoteTags(gomock.Any(), noteID).Return(nil, nil)
	repo.EXPECT().UpdateNote(gomock.Any(), gomock.Any(), gomock.Any(), []uuid.UUID{removedID}).
		DoAndReturn(func(ctx context.Context, note models.Note, blocks []models.Block, deleted []uuid.UUID) error {
			require.Len(t, blocks, 2)
			assert.Equal(t, keptID, blocks[0].ID)
			assert.NotEqual(t, uuid.Nil, blocks[1].ID)
			assert.Equal(t, "Новый заголовок", note.Title)
			assert.Equal(t, header, note.Header)
			assert.Equal(t, icon, note.Icon)
			return nil
		})

	req := notes.UpdateNoteRequest{
		Title:  "Новый заголовок",
		Header: &header,
		Icon:   &icon,
		Blocks: []notes.BlockInput{
			{ID: &keptID, Content: []byte(`{"text":"старый"}`)},
			{Content: []byte(`{"text":"новый"}`)},
		},
	}
	_, err := NewUsecase(repo, &mockAuthClient{exists: true}).UpdateNote(context.Background(), userID, noteID, req)

	require.NoError(t, err)
}

func TestDeleteNote(t *testing.T) {
	userID := uuid.NewV4()
	noteID := uuid.NewV4()

	tests := []struct {
		name        string
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
	}{
		{
			name: "ok",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(models.Note{ID: noteID, CreatedBy: userID}, nil)
				repo.EXPECT().DeleteNote(gomock.Any(), noteID).Return(nil)
			},
		},
		{
			name: "not found",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(models.Note{}, models.ErrNoteNotFound)
			},
			expectedErr: models.ErrNoteNotFound,
		},
		{
			name: "foreign note",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(models.Note{ID: noteID, CreatedBy: uuid.NewV4()}, nil)
				repo.EXPECT().ReadCollaboratorRole(gomock.Any(), noteID, userID).Return("", models.ErrNoteNotFound)
			},
			expectedErr: models.ErrNoteNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			err := NewUsecase(repo, &mockAuthClient{exists: true}).DeleteNote(context.Background(), userID, noteID)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestNoteTitleTooLong(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRepo(ctrl)
	u := NewUsecase(repo, &mockAuthClient{exists: true})
	long := strings.Repeat("я", maxTitleLength+1)
	userID := uuid.NewV4()
	noteID := uuid.NewV4()

	_, err := u.CreateNote(context.Background(), userID, notes.CreateNoteRequest{Title: long})
	assert.ErrorIs(t, err, models.ErrValidation)

	_, err = u.UpdateNote(context.Background(), userID, noteID, notes.UpdateNoteRequest{Title: long})
	assert.ErrorIs(t, err, models.ErrValidation)
}

func TestGetNote_CheckAccess(t *testing.T) {
	userID := uuid.NewV4()
	foreignUserID := uuid.NewV4()
	noteID := uuid.NewV4()
	own := models.Note{ID: noteID, CreatedBy: userID}

	tests := []struct {
		name        string
		userID      uuid.UUID
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
	}{
		{
			name:   "owner access",
			userID: userID,
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil).Times(2)
				repo.EXPECT().ReadBlocks(gomock.Any(), noteID).Return(nil, nil)
				repo.EXPECT().ReadNoteTags(gomock.Any(), noteID).Return(nil, nil)
			},
		},
		{
			name:   "collaborator editor access",
			userID: foreignUserID,
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil).Times(2)
				repo.EXPECT().ReadCollaboratorRole(gomock.Any(), noteID, foreignUserID).Return("editor", nil)
				repo.EXPECT().ReadBlocks(gomock.Any(), noteID).Return(nil, nil)
				repo.EXPECT().ReadNoteTags(gomock.Any(), noteID).Return(nil, nil)
			},
		},
		{
			name:   "collaborator viewer access",
			userID: foreignUserID,
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil).Times(2)
				repo.EXPECT().ReadCollaboratorRole(gomock.Any(), noteID, foreignUserID).Return("viewer", nil)
				repo.EXPECT().ReadBlocks(gomock.Any(), noteID).Return(nil, nil)
				repo.EXPECT().ReadNoteTags(gomock.Any(), noteID).Return(nil, nil)
			},
		},
		{
			name:   "no access (IDOR)",
			userID: foreignUserID,
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil).Times(1)
				repo.EXPECT().ReadCollaboratorRole(gomock.Any(), noteID, foreignUserID).Return("", models.ErrNoteNotFound)
			},
			expectedErr: models.ErrNoteNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			_, err := NewUsecase(repo, &mockAuthClient{exists: true}).GetNote(context.Background(), tt.userID, noteID)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpdateNote_ViewerCannotEdit(t *testing.T) {
	userID := uuid.NewV4()
	noteID := uuid.NewV4()
	own := models.Note{ID: noteID, CreatedBy: uuid.NewV4()}

	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRepo(ctrl)

	repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil).Times(1)
	repo.EXPECT().ReadCollaboratorRole(gomock.Any(), noteID, userID).Return("viewer", nil)

	_, err := NewUsecase(repo, &mockAuthClient{exists: true}).UpdateNote(context.Background(), userID, noteID, notes.UpdateNoteRequest{Title: "new"})

	assert.ErrorIs(t, err, models.ErrForbidden)
}

func TestAddCollaborator(t *testing.T) {
	ownerID := uuid.NewV4()
	noteID := uuid.NewV4()
	targetID := uuid.NewV4()
	own := models.Note{ID: noteID, CreatedBy: ownerID}

	tests := []struct {
		name        string
		userID      uuid.UUID
		login       string
		authMock    *mockAuthClient
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
	}{
		{
			name:   "owner adds collaborator",
			userID: ownerID,
			login:  "vasya",
			authMock: &mockAuthClient{
				userID: targetID, // auth вернет этот ID
			},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil)
				repo.EXPECT().AddCollaborator(gomock.Any(), noteID, targetID, "editor").Return(nil)
			},
		},
		{
			name:     "not owner tries to add",
			userID:   uuid.NewV4(),
			login:    "vasya",
			authMock: &mockAuthClient{},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil)
			},
			expectedErr: models.ErrForbidden,
		},
		{
			name:   "target user not found",
			userID: ownerID,
			login:  "unknown",
			authMock: &mockAuthClient{
				loginErr: models.ErrUserNotFound, // auth вернет ошибку
			},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil)
			},
			expectedErr: models.ErrUserNotFound,
		},
		{
			name:   "add self",
			userID: ownerID,
			login:  "owner_login",
			authMock: &mockAuthClient{
				userID: ownerID, // auth вернет ID самого владельца
			},
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadNote(gomock.Any(), noteID).Return(own, nil)
			},
			expectedErr: models.ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			err := NewUsecase(repo, tt.authMock).AddCollaborator(context.Background(), tt.userID, noteID, tt.login, "editor")

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
