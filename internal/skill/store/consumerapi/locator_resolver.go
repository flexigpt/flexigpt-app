package consumerapi

import (
	"context"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/composition/local/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/source"
)

type apiOptions struct {
	locatorResolvers  []providerapi.LocatorResolverFactory
	fallbackProviders map[declaration.Type]resolve.FallbackProvider
	targetMappers     map[declaration.Type]resolve.ArtifactTargetMapper
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

func WithTargetMappers(
	values map[declaration.Type]resolve.ArtifactTargetMapper,
) Option {
	return func(options *apiOptions) {
		if values == nil {
			options.targetMappers = nil
			return
		}

		options.targetMappers = make(
			map[declaration.Type]resolve.ArtifactTargetMapper,
			len(values),
		)
		maps.Copy(options.targetMappers, values)
	}
}

type skillLocatorRuntime struct {
	artifacts compositionapi.ArtifactAPI
}

func (r skillLocatorRuntime) ListArtifactsBySource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) ([]catalog.Entry, error) {
	if r.artifacts == nil {
		return nil, basespec.ErrClosed
	}
	return r.artifacts.ListBySource(ctx, rootID, sourceID, catalog.ListOptions{})
}
