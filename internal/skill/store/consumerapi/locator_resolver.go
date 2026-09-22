package consumerapi

import (
	"context"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

type apiOptions struct {
	locatorResolvers  []providerapi.LocatorResolverFactory
	fallbackProviders map[declaration.Type]resolve.FallbackProvider
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

func WithFallbackProviders(
	values map[declaration.Type]resolve.FallbackProvider,
) Option {
	return func(options *apiOptions) {
		if values == nil {
			options.fallbackProviders = nil
			return
		}

		options.fallbackProviders = make(
			map[declaration.Type]resolve.FallbackProvider,
			len(values),
		)
		maps.Copy(options.fallbackProviders, values)
	}
}

type skillLocatorRuntime struct {
	artifacts compositionapi.ArtifactAPI
}

func (r skillLocatorRuntime) ListArtifactsBySource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) ([]artifact.Artifact, error) {
	if r.artifacts == nil {
		return nil, basespec.ErrClosed
	}
	return r.artifacts.ListBySource(ctx, rootID, sourceID)
}
