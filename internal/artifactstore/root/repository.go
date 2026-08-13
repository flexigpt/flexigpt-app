package root

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

type RootDraft struct {
	ID          basespec.RootID     `json:"id"                    required:"true"`
	StorageKey  basespec.StorageKey `json:"storageKey"            required:"true"`
	DisplayName string              `json:"displayName"           required:"true"`
	Description string              `json:"description,omitempty"`
}

type RootUpdate struct {
	ExpectedRevision uint64 `json:"expectedRevision"`
	DisplayName      string `json:"displayName"`
	Description      string `json:"description,omitempty"`
}

type Repository interface {
	Create(
		ctx context.Context,
		value Root,
	) error

	Get(
		ctx context.Context,
		id basespec.RootID,
	) (Root, error)

	List(ctx context.Context) ([]Root, error)

	Update(
		ctx context.Context,
		value Root,
		expectedRevision uint64,
	) error

	Retire(
		ctx context.Context,
		value Root,
		expectedRevision uint64,
	) error

	Purge(
		ctx context.Context,
		id basespec.RootID,
		expectedRevision uint64,
	) error
}
