package consumerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
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

type skillLocatorRuntime struct {
	api *API
}

var _ providerapi.LocatorRuntime = skillLocatorRuntime{}

func (r skillLocatorRuntime) ListArtifactsBySource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) ([]artifact.Artifact, error) {
	if r.api == nil || r.api.artifacts == nil {
		return nil, basespec.ErrClosed
	}
	return r.api.artifacts.ListBySource(ctx, rootID, sourceID)
}
