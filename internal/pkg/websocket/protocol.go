package ws

import (
	"encoding/json"

	uuid "github.com/satori/go.uuid"
)

const (
	EventCursorMove  = "cursor_move"
	EventNoteUpdated = "note_updated"
	EventUserJoined  = "user_joined"
	EventUserLeft    = "user_left"
)

type Message struct {
	Type    string          `json:"type"`
	NoteID  uuid.UUID       `json:"note_id"`
	UserID  uuid.UUID       `json:"user_id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type CursorPayload struct {
	BlockID   *uuid.UUID `json:"block_id,omitempty"`
	Offset    int        `json:"offset"`
	Selection *Selection `json:"selection,omitempty"`
	Color     string     `json:"color,omitempty"`
	UserName  string     `json:"user_name,omitempty"`
}

type Selection struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type BlockUpdatePayload struct {
	BlockID uuid.UUID       `json:"block_id"`
	Content json.RawMessage `json:"content"`
}
