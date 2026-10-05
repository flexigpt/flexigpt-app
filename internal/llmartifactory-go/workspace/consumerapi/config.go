package consumerapi

import (
	"maps"

	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/mcp"
	workspaceRuntime "github.com/flexigpt/flexigpt-app/internal/workspace/runtime"
)

type Config struct {
	ContextComposition workspaceRuntime.CompositionPolicy
	LocatorResolvers   []locator.Factory
	ResolverLimits     composition.Limits
	FallbackProviders  map[declaration.Type]composition.FallbackProvider
	TargetMappers      map[declaration.Type]composition.ArtifactTargetMapper
	MCPServers         mcp.ServerResolver

	// AdditionalDecoderHints lets application composition add dedicated
	// providers such as MCP or future YAML adapters without making Workspace
	// import those consumer domains.
	AdditionalDecoderHints []sourceModel.DecoderHint
}

func (c Config) normalized() Config {
	output := c
	output.ContextComposition = output.ContextComposition.Normalized()
	output.LocatorResolvers = append(
		[]locator.Factory(nil),
		c.LocatorResolvers...,
	)
	if c.FallbackProviders != nil {
		output.FallbackProviders = make(
			map[declaration.Type]composition.FallbackProvider,
			len(c.FallbackProviders),
		)
		maps.Copy(output.FallbackProviders, c.FallbackProviders)
	}
	if c.TargetMappers != nil {
		output.TargetMappers = make(
			map[declaration.Type]composition.ArtifactTargetMapper,
			len(c.TargetMappers),
		)
		maps.Copy(output.TargetMappers, c.TargetMappers)
	}

	output.AdditionalDecoderHints = make(
		[]sourceModel.DecoderHint,
		len(c.AdditionalDecoderHints),
	)
	for index, hint := range c.AdditionalDecoderHints {
		output.AdditionalDecoderHints[index] = hint.Clone()
	}
	return output
}

func DefaultConfig() Config {
	return Config{
		ContextComposition: workspaceRuntime.DefaultCompositionPolicy(),
	}
}
