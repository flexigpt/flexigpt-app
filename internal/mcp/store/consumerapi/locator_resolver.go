package consumerapi

import (
	"context"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
)

type apiOptions struct {
	locatorResolvers  []provider.LocatorResolverFactory
	fallbackProviders map[declaration.Type]resolve.FallbackProvider
	targetMappers     map[declaration.Type]resolve.ArtifactTargetMapper
}

type Option func(*apiOptions)

func WithLocatorResolvers(
	values []provider.LocatorResolverFactory,
) Option {
	return func(options *apiOptions) {
		options.locatorResolvers = append(
			[]provider.LocatorResolverFactory(nil),
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

type mcpLocatorRuntime struct {
	cat catalog.API
}

func (r mcpLocatorRuntime) ListArtifactsBySource(
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
