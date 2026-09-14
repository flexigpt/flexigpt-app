package source

import (
	"encoding/json"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

type Draft struct {
	ID          SourceID            `json:"id"`
	StorageKey  basespec.StorageKey `json:"storageKey"`
	Kind        SourceKind          `json:"kind"`
	DisplayName string              `json:"displayName"`
	Enabled     bool                `json:"enabled"`
	Config      json.RawMessage     `json:"config"`
	Discovery   DiscoverySpec       `json:"discovery"`
}

type Update struct {
	ExpectedRevision uint64 `json:"expectedRevision" required:"true"`
	DisplayName      string `json:"displayName"      required:"true"`
	Enabled          bool   `json:"enabled"`

	// Config is write-only replacement configuration. A nil value preserves
	// the current normalized configuration so public callers can update Source
	// metadata without reading or resending private Source configuration.
	Config json.RawMessage `json:"config,omitempty"`

	// Discovery is a complete replacement when non-nil. A nil value preserves
	// the current Store-owned declaration discovery configuration.
	Discovery *DiscoverySpec `json:"discovery,omitempty"`
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
