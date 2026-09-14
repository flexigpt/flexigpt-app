package consumerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

func (a *StoreAPI) GetSource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) (source.Summary, error) {
	if a == nil || a.sources == nil {
		return source.Summary{}, basespec.ErrClosed
	}
	return a.sources.Get(ctx, rootID, sourceID)
}

func (a *StoreAPI) UpdateSource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	update source.Update,
) (source.Summary, error) {
	if a == nil || a.sources == nil {
		return source.Summary{}, basespec.ErrClosed
	}
	return a.sources.Update(ctx, rootID, sourceID, update)
}

func (a *StoreAPI) RefreshSource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) (refresh.RefreshSourceResult, error) {
	if a == nil || a.discovery == nil {
		return refresh.RefreshSourceResult{}, basespec.ErrClosed
	}
	return a.discovery.RefreshSource(ctx, rootID, sourceID)
}

func (a *StoreAPI) ListArtifactsBySource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) ([]artifact.Artifact, error) {
	if a == nil || a.artifacts == nil {
		return nil, basespec.ErrClosed
	}
	return a.artifacts.ListBySource(ctx, rootID, sourceID)
}
