package managedpackage

import (
	"context"

	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// BatchWriter is the trusted physical package hydration capability used by
// generated protected-topology installation.
type BatchWriter interface {
	ApplyPackageBatch(
		ctx context.Context,
		value sourceModel.Source,
		publications []managedpackageModel.ManagedPackagePublication,
		removals []managedpackageModel.ManagedPackageAddress,
	) error
}

// Runtime is the trusted Source driver-dispatch capability used by
// ManagePackage and Install flows.
//
// It is deliberately not part of source.API.
type Runtime interface {
	Writer
	BatchWriter

	SupportsManagedPackages(
		kind sourceModel.SourceKind,
	) bool

	RemoveManagedRoot(
		ctx context.Context,
		rootStorageKey spec.StorageKey,
	) error
}
