package consumerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

type apiOptions struct {
	locatorResolvers []providerapi.LocatorResolverFactory
}

type Option func(*apiOptions)

func WithLocatorResolvers(
	values []providerapi.LocatorResolverFactory,
) Option {
	return func(options *apiOptions) {
		options.locatorResolvers = append(
			[]providerapi.LocatorResolverFactory(nil),
			values...,
		)
	}
}

type mcpLocatorRuntime struct {
	api *API
}

var _ providerapi.LocatorRuntime = mcpLocatorRuntime{}

func (r mcpLocatorRuntime) GetSource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) (source.Summary, error) {
	if r.api == nil || r.api.sources == nil {
		return source.Summary{}, basespec.ErrClosed
	}
	return r.api.sources.Get(ctx, rootID, sourceID)
}

func (r mcpLocatorRuntime) UpdateSource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	update source.Update,
) (source.Summary, error) {
	if r.api == nil || r.api.sources == nil {
		return source.Summary{}, basespec.ErrClosed
	}
	return r.api.sources.Update(ctx, rootID, sourceID, update)
}

func (r mcpLocatorRuntime) RefreshSource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) (refresh.RefreshSourceResult, error) {
	if r.api == nil || r.api.discovery == nil {
		return refresh.RefreshSourceResult{}, basespec.ErrClosed
	}
	return r.api.discovery.RefreshSource(ctx, rootID, sourceID)
}

func (r mcpLocatorRuntime) ListArtifactsBySource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) ([]artifact.Artifact, error) {
	if r.api == nil || r.api.artifacts == nil {
		return nil, basespec.ErrClosed
	}
	return r.api.artifacts.ListBySource(ctx, rootID, sourceID)
}
