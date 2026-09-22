package consumerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
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
) ([]artifact.Artifact, error) {
	if r.artifacts == nil {
		return nil, basespec.ErrClosed
	}
	return r.artifacts.ListBySource(ctx, rootID, sourceID)
}
