package model

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// RootDraft is the caller-owned input for Root creation.
type RootDraft struct {
	ID          RootID          `json:"id"                    required:"true"`
	StorageKey  spec.StorageKey `json:"storageKey"            required:"true"`
	DisplayName string          `json:"displayName"           required:"true"`
	Description string          `json:"description,omitempty"`
}

// RootUpdate is the caller-owned input for a Root metadata update.
type RootUpdate struct {
	ExpectedRevision uint64 `json:"expectedRevision"`
	DisplayName      string `json:"displayName"`
	Description      string `json:"description,omitempty"`
}
