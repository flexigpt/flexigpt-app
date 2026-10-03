package managedpackage

import (
	"context"

	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
)

// Writer is the optional physical managed-package capability of a Source
// driver. It writes complete package directories, never arbitrary files.
type Writer interface {
	PublishPackage(
		ctx context.Context,
		value sourceModel.Source,
		publication managedpackageModel.ManagedPackagePublication,
	) (generation string, err error)

	RemovePackage(
		ctx context.Context,
		value sourceModel.Source,
		address managedpackageModel.ManagedPackageAddress,
		expectedGeneration string,
	) error
}
