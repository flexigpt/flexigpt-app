package consumerapi

import (
	"context"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type apiOptions struct {
	locatorResolvers  []locator.Factory
	fallbackProviders map[declaration.Type]composition.FallbackProvider
	targetMappers     map[declaration.Type]composition.ArtifactTargetMapper
}

type Option func(*apiOptions)

func WithLocatorResolvers(
	values []locator.Factory,
) Option {
	return func(options *apiOptions) {
		options.locatorResolvers = append(
			[]locator.Factory(nil),
			values...,
		)
	}
}

func WithFallbackProviders(
	values map[declaration.Type]composition.FallbackProvider,
) Option {
	return func(options *apiOptions) {
		if values == nil {
			options.fallbackProviders = nil
			return
		}

		options.fallbackProviders = make(
			map[declaration.Type]composition.FallbackProvider,
			len(values),
		)
		maps.Copy(options.fallbackProviders, values)
	}
}

func WithTargetMappers(
	values map[declaration.Type]composition.ArtifactTargetMapper,
) Option {
	return func(options *apiOptions) {
		if values == nil {
			options.targetMappers = nil
			return
		}

		options.targetMappers = make(
			map[declaration.Type]composition.ArtifactTargetMapper,
			len(values),
		)
		maps.Copy(options.targetMappers, values)
	}
}

type skillLocatorRuntime struct {
	cat catalog.API
}

func (r skillLocatorRuntime) ListBySource(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	options catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
	return r.cat.ListBySource(ctx, rootID, sourceID, options)
}
