package models

import "time"

type Note struct {
	ID           string    `json:"id"`
	ParentNoteID *string   `json:"parent_note_id,omitempty"`
	BlocksID     []string  `json:"blocks_id"`
	CreatedBy    string    `json:"created_by"`
	Title        string    `json:"title"`
	Header       string    `json:"header"`
	Icon         string    `json:"icon"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
