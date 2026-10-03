package source

import (
	"context"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// ContentMutation is the trusted Source capability that acknowledges a
// successful source-side mutation by advancing the Source revision.
//
// Physical writes belong to source/managedpackage.Runtime. This capability
// publishes only the corresponding Source metadata transition.
type ContentMutation interface {
	MarkContentChanged(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		expectedRevision uint64,
	) (sourceModel.Summary, error)
}

type Reader interface {
	Get(
		ctx context.Context,
		rootID rootModel.RootID,
		id sourceModel.SourceID,
	) (sourceModel.Source, error)

	List(
		ctx context.Context,
		rootID rootModel.RootID,
	) ([]sourceModel.Source, error)
}

type Repository interface {
	Reader

	FindByStorageKey(
		ctx context.Context,
		rootID rootModel.RootID,
		storageKey spec.StorageKey,
	) (sourceModel.Source, error)

	Create(
		ctx context.Context,
		value sourceModel.Source,
	) error

	Update(
		ctx context.Context,
		value sourceModel.Source,
		expectedRevision uint64,
	) error

	Retire(
		ctx context.Context,
		value sourceModel.Source,
		expectedRevision uint64,
	) error

	Discard(
		ctx context.Context,
		rootID rootModel.RootID,
		id sourceModel.SourceID,
		expectedRevision uint64,
	) error

	Purge(
		ctx context.Context,
		rootID rootModel.RootID,
		id sourceModel.SourceID,
		expectedRevision uint64,
	) error
}
