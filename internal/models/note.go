package models

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type Note struct {
	ID           uuid.UUID   `json:"id"`
	ParentNoteID *uuid.UUID  `json:"parent_note_id,omitempty"`
	BlocksID     []uuid.UUID `json:"blocks_id"`
	CreatedBy    uuid.UUID   `json:"created_by"`
	Title        string      `json:"title"`
	Header       string      `json:"header"`
	Icon         string      `json:"icon"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}
