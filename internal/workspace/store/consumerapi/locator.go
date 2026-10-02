package consumerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/composition/local/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
)

// workspaceLocatorRuntime keeps the generic provider runtime port out of the
// Workspace consumer API surface.
type workspaceLocatorRuntime struct {
	artifacts compositionapi.ArtifactAPI
}

func (r workspaceLocatorRuntime) ListArtifactsBySource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) ([]catalog.Entry, error) {
	if r.artifacts == nil {
		return nil, model.ErrClosed
	}
	return r.artifacts.ListBySource(
		ctx,
		rootID,
		sourceID,
		catalog.ListOptions{},
	)
}
