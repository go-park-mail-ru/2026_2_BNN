package models

import (
	"encoding/json"
	"time"

	uuid "github.com/satori/go.uuid"
)

const (
	OnlineConfScopeFrontend = "frontend"
	OnlineConfScopeBackend  = "backend"
	OnlineConfScopeAuth     = "auth"
	OnlineConfScopeNotes    = "notes"
)

const (
	OnlineConfKindConfig = "config"
	OnlineConfKindFlag   = "flag"
)

const (
	OnlineConfValueString  = "string"
	OnlineConfValueNumber  = "number"
	OnlineConfValueBoolean = "boolean"
	OnlineConfValueObject  = "object"
	OnlineConfValueArray   = "array"
)

type OnlineConfUser struct {
	ID    uuid.UUID `json:"id"`
	Login string    `json:"login"`
}

type OnlineConfConfig struct {
	ValueType string          `json:"value_type"`
	Value     json.RawMessage `json:"value"`
}

type OnlineConfFlag struct {
	Enabled        bool             `json:"enabled"`
	RolloutPercent int32            `json:"rollout_percent"`
	IncludeUsers   []OnlineConfUser `json:"include_users"`
	ExcludeUsers   []OnlineConfUser `json:"exclude_users"`
}

type OnlineConfKey struct {
	ID          uuid.UUID         `json:"id"`
	Scope       string            `json:"scope"`
	Key         string            `json:"key"`
	Kind        string            `json:"kind"`
	Description string            `json:"description"`
	IsPublic    bool              `json:"is_public"`
	Config      *OnlineConfConfig `json:"config,omitempty"`
	Flag        *OnlineConfFlag   `json:"flag,omitempty"`
	Version     int32             `json:"version"`
	UpdatedBy   OnlineConfUser    `json:"updated_by"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type OnlineConfOverride struct {
	KeyID   uuid.UUID
	UserID  uuid.UUID
	Login   string
	Enabled bool
}

type OnlineConfHistory struct {
	ID        uuid.UUID      `json:"id"`
	KeyID     uuid.UUID      `json:"-"`
	Action    string         `json:"action"`
	OldValue  *OnlineConfKey `json:"old_value"`
	NewValue  *OnlineConfKey `json:"new_value"`
	User      OnlineConfUser `json:"user"`
	CreatedAt time.Time      `json:"created_at"`
}

type Snapshot struct {
	Revision int64           `json:"revision"`
	Keys     []OnlineConfKey `json:"keys"`
}
