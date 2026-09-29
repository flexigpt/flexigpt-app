package overlay

import (
	"encoding/json"
	"time"
)

// GroupID is the compile-time type for group names stored on disk.
type GroupID string

// KeyID is the compile-time type for individual entry names.
type KeyID string

// Flag represents one feature-flag record.
type Flag struct {
	Value      json.RawMessage `json:"value"`
	CreatedAt  time.Time       `json:"created_at"`  //nolint:tagliatelle // JSON file.
	ModifiedAt time.Time       `json:"modified_at"` //nolint:tagliatelle // JSON file.
}

type TypedFlag[ValT any] struct {
	Value      ValT      `json:"value"`
	CreatedAt  time.Time `json:"created_at"`  //nolint:tagliatelle // JSON file.
	ModifiedAt time.Time `json:"modified_at"` //nolint:tagliatelle // JSON file.
}

// Root is the complete in-memory image of the file.
type Root map[GroupID]map[KeyID]Flag

// Key identifies a single flag entry.
type Key interface {
	Group() GroupID
	ID() KeyID
}
