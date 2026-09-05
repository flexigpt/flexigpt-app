package root

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"

// RootDraft is the caller-owned input for Root creation.
type RootDraft struct {
	ID          basespec.RootID     `json:"id"                    required:"true"`
	StorageKey  basespec.StorageKey `json:"storageKey"            required:"true"`
	DisplayName string              `json:"displayName"           required:"true"`
	Description string              `json:"description,omitempty"`
}

// RootUpdate is the caller-owned input for a Root metadata update.
type RootUpdate struct {
	ExpectedRevision uint64 `json:"expectedRevision"`
	DisplayName      string `json:"displayName"`
	Description      string `json:"description,omitempty"`
}
