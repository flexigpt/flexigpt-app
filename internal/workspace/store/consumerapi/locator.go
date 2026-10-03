package consumerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
)

// workspaceLocatorRuntime keeps the generic provider runtime port out of the
// Workspace consumer API surface.
type workspaceLocatorRuntime struct {
	cat catalog.API
}

func (r workspaceLocatorRuntime) ListArtifactsBySource(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) ([]catalogModel.Entry, error) {
	return r.cat.ListBySource(
		ctx,
		rootID,
		sourceID,
		catalogModel.ListOptions{},
	)
}
