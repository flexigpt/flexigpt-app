package mcp

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
)

type apiOptions struct {
	resolver *composition.Resolver
	support  Support
}

type Option func(*apiOptions)

// WithCompositionResolver supplies the single LLM Artifactory composition
// owner. MCP does not bind an independent locator registry or graph resolver.
func WithCompositionResolver(
	value *composition.Resolver,
) Option {
	return func(options *apiOptions) {
		options.resolver = value
	}
}

func WithSupport(value Support) Option {
	return func(options *apiOptions) {
		options.support = value
	}
}
