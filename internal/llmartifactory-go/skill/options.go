package skill

import (
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
)

type apiOptions struct {
	resolver        *composition.Resolver
	support         Support
	nativeResources resourceFlow.NativePathAPI
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

// WithNativeResources enables verified, read-only local package inspection.
func WithNativeResources(value resourceFlow.NativePathAPI) Option {
	return func(options *apiOptions) {
		options.nativeResources = value
	}
}
