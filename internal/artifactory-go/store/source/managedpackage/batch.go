package managedpackage

import (
	"context"

	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
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
