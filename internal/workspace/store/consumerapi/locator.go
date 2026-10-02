package consumerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/composition/local/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/source"
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
		return nil, basespec.ErrClosed
	}
	return r.artifacts.ListBySource(
		ctx,
		rootID,
		sourceID,
		catalog.ListOptions{},
	)
}
