package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/ctxutil"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

type mockNotesGetter struct {
	note    *models.Note
	err     error
	role    string
	roleErr error
}

func (m *mockNotesGetter) GetNote(ctx context.Context, userID, noteID uuid.UUID) (models.Note, error) {
	return *m.note, m.err
}

func (m *mockNotesGetter) CheckAccess(ctx context.Context, userID, noteID uuid.UUID) (string, error) {
	return m.role, m.roleErr
}

func TestServeNoteWS(t *testing.T) {
	noteID := uuid.NewV4()
	userID := uuid.NewV4()

	tests := []struct {
		name         string
		userID       *uuid.UUID
		noteIDStr    string
		mockNote     *models.Note
		mockErr      error
		role         string
		roleErr      error
		expectedCode int
	}{
		{
			name:         "unauthorized (401)",
			userID:       nil,
			noteIDStr:    noteID.String(),
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "invalid note id (400)",
			userID:       &userID,
			noteIDStr:    "not-a-uuid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "note not found (404 - IDOR)",
			userID:       &userID,
			noteIDStr:    noteID.String(),
			mockNote:     &models.Note{},
			mockErr:      models.ErrNoteNotFound,
			roleErr:      models.ErrNoteNotFound,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "forbidden (403 - no access)",
			userID:       &userID,
			noteIDStr:    noteID.String(),
			mockNote:     &models.Note{},
			role:         "",
			roleErr:      models.ErrForbidden,
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "internal server error (500)",
			userID:       &userID,
			noteIDStr:    noteID.String(),
			mockNote:     &models.Note{},
			mockErr:      assert.AnError,
			roleErr:      assert.AnError,
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGetter := &mockNotesGetter{
				note:    tt.mockNote,
				err:     tt.mockErr,
				role:    tt.role,
				roleErr: tt.roleErr,
			}

			h := NewHandler(nil, "*", mockGetter)

			req := httptest.NewRequest(http.MethodGet, "/api/ws/notes/"+tt.noteIDStr, nil)
			req = mux.SetURLVars(req, map[string]string{"noteID": tt.noteIDStr})

			if tt.userID != nil {
				req = req.WithContext(ctxutil.WithUserID(req.Context(), *tt.userID))
			}

			w := httptest.NewRecorder()
			h.ServeNoteWS(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}
