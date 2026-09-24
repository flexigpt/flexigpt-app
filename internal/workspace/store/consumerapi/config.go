package consumerapi

import (
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	workspaceRuntime "github.com/flexigpt/flexigpt-app/internal/workspace/runtime"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/mcp"
)

type Config struct {
	ContextComposition workspaceRuntime.CompositionPolicy
	LocatorResolvers   []providerapi.LocatorResolverFactory
	ResolverLimits     resolve.Limits
	FallbackProviders  map[declaration.Type]resolve.FallbackProvider
	TargetMappers      map[declaration.Type]resolve.ArtifactTargetMapper
	MCPServers         mcp.ServerResolver

	// AdditionalDecoderHints lets application composition add dedicated
	// providers such as MCP or future YAML adapters without making Workspace
	// import those consumer domains.
	AdditionalDecoderHints []source.DecoderHint
}

func (c Config) normalized() Config {
	output := c
	output.ContextComposition = output.ContextComposition.Normalized()
	output.LocatorResolvers = append(
		[]providerapi.LocatorResolverFactory(nil),
		c.LocatorResolvers...,
	)
	if c.FallbackProviders != nil {
		output.FallbackProviders = make(
			map[declaration.Type]resolve.FallbackProvider,
			len(c.FallbackProviders),
		)
		maps.Copy(output.FallbackProviders, c.FallbackProviders)
	}
	if c.TargetMappers != nil {
		output.TargetMappers = make(
			map[declaration.Type]resolve.ArtifactTargetMapper,
			len(c.TargetMappers),
		)
		maps.Copy(output.TargetMappers, c.TargetMappers)
	}

	output.AdditionalDecoderHints = make(
		[]source.DecoderHint,
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
