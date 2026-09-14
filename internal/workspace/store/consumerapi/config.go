package consumerapi

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	workspaceRuntime "github.com/flexigpt/flexigpt-app/internal/workspace/runtime"
)

type Config struct {
	ContextComposition workspaceRuntime.CompositionPolicy

	// AdditionalDecoderHints lets application composition add dedicated
	// providers such as MCP or future YAML adapters without making Workspace
	// import those consumer domains.
	AdditionalDecoderHints []source.DecoderHint
}

func (c Config) normalized() Config {
	output := c
	output.ContextComposition = output.ContextComposition.Normalized()
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
