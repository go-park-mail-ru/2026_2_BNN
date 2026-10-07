package models

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type Note struct {
	ID           uuid.UUID  `json:"id"`
	ParentNoteID *uuid.UUID `json:"parent_note_id,omitempty"`
	Blocks       []Block    `json:"blocks"`
	CreatedBy    uuid.UUID  `json:"created_by"`
	Title        string     `json:"title"`
	Header       string     `json:"header,omitempty"`
	Icon         string     `json:"icon,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
