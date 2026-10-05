package consumerapi

import (
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/mcp"
	workspaceRuntime "github.com/flexigpt/flexigpt-app/internal/workspace/runtime"
)

type Config struct {
	ContextComposition workspaceRuntime.CompositionPolicy
	Composition        *composition.Resolver
	MCPServers         mcp.ServerResolver

	// AdditionalDecoderHints lets application composition add dedicated
	// decoders without making Workspace import unrelated family services.
	AdditionalDecoderHints []sourceModel.DecoderHint
}

func (c Config) normalized() Config {
	output := c
	output.ContextComposition = output.ContextComposition.Normalized()
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
