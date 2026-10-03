package source

import (
	"context"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

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
