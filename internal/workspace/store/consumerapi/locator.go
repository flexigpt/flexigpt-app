package consumerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	catalog "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	root "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// workspaceLocatorRuntime keeps the generic provider runtime port out of the
// Workspace consumer API surface.
type workspaceLocatorRuntime struct {
	artifacts local.ArtifactAPI
}

func (r workspaceLocatorRuntime) ListArtifactsBySource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) ([]catalog.Entry, error) {
	if r.artifacts == nil {
		return nil, spec.ErrClosed
	}
	return r.artifacts.ListBySource(
		ctx,
		rootID,
		sourceID,
		catalog.ListOptions{},
	)
}
