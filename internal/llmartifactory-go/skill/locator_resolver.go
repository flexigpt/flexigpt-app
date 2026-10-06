package skill

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
)

type apiOptions struct {
	resolver *composition.Resolver
}

type Option func(*apiOptions)

// WithCompositionResolver supplies the shared LLM Artifactory composition
// resolver. Skill does not create a private locator registry or graph resolver.
func WithCompositionResolver(
	value *composition.Resolver,
) Option {
	return func(options *apiOptions) {
		options.resolver = value
	}
}

func requiredCompositionResolver(
	value *composition.Resolver,
) (*composition.Resolver, error) {
	if value == nil {
		return nil, fmt.Errorf(
			"%w: Skill composition resolver is required",
			spec.ErrInvalid,
		)
	}
	return value, nil
}
