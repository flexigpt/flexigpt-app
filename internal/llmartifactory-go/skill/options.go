package skill

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
)

type apiOptions struct {
	resolver *composition.Resolver
	support  Support
}

type Option func(*apiOptions)

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
