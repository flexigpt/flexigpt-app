package driver

import (
	"context"

	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// ManagedSourceBootstrapper is implemented by writable managed drivers that
// must provision physical storage before Source metadata is committed.
type ManagedSourceBootstrapper interface {
	BootstrapManagedSource(
		ctx context.Context,
		value sourceModel.Source,
	) error

	DiscardBootstrappedManagedSource(
		ctx context.Context,
		value sourceModel.Source,
	) error
}

// ManagedRootRemover is a trusted topology-reset capability. It must never be
// exposed through source.API.
type ManagedRootRemover interface {
	RemoveManagedRoot(
		ctx context.Context,
		rootStorageKey spec.StorageKey,
	) error
}
