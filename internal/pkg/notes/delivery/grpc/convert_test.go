package grpc

import (
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/notes/delivery/grpc/gen"
	tagsgen "bnn/internal/pkg/tags/delivery/grpc/gen"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoteConvert(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	parentID := uuid.NewV4()
	userID := uuid.NewV4()
	note := models.Note{
		ID:           uuid.NewV4(),
		ParentNoteID: &parentID,
		Blocks:       []models.Block{{ID: uuid.NewV4(), Content: []byte(`{"type":"text"}`), CreatedAt: now, UpdatedAt: now}},
		Tags:         []models.Tag{{ID: uuid.NewV4(), CreatedBy: userID, Name: "учёба", Color: "#FF880080", CreatedAt: now, UpdatedAt: now}},
		CreatedBy:    userID,
		Title:        "Учёба",
		Header:       "header.png",
		Icon:         "📚",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	got, err := NotesFromProto([]*gen.Note{NoteToProto(note)})
	require.NoError(t, err)
	assert.Equal(t, []models.Note{note}, got)
}

func TestNoteConvertErrors(t *testing.T) {
	id := uuid.NewV4().String()
	bad := "bad"

	tests := []struct {
		name string
		note *gen.Note
	}{
		{name: "bad id", note: &gen.Note{Id: bad}},
		{name: "bad created by", note: &gen.Note{Id: id, CreatedBy: bad}},
		{name: "bad parent", note: &gen.Note{Id: id, CreatedBy: id, ParentNoteId: &bad}},
		{name: "bad block", note: &gen.Note{Id: id, CreatedBy: id, Blocks: []*gen.Block{{Id: bad}}}},
		{name: "bad tag", note: &gen.Note{Id: id, CreatedBy: id, Tags: []*tagsgen.Tag{{Id: bad}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NotesFromProto([]*gen.Note{tt.note})
			assert.ErrorIs(t, err, models.ErrValidation)
		})
	}
}
