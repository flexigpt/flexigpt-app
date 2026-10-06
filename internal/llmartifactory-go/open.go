package llmartifactory

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	corelocator "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
)

// Open attaches the LLM artifact-domain registration set to an already opened
// generic Artifact Store. Generic Store ownership remains with deployment
// assembly; this constructor does not open or close provider resources.
func Open(ctx context.Context, config Config) (*Artifactory, error) {
	if config.Store == nil {
		return nil, fmt.Errorf(
			"%w: LLM Artifactory generic Store is nil",
			spec.ErrInvalid,
		)
	}
	if config.Interpretations == nil {
		return nil, fmt.Errorf(
			"%w: LLM Artifactory declaration interpretation registry is nil",
			spec.ErrInvalid,
		)
	}

	if err := config.Scope.Validate(); err != nil {
		return nil, err
	}

	locators, err := corelocator.NewRegistry(config.LocatorFactories...)
	if err != nil {
		return nil, err
	}

	locatorResolver, err := composition.NewProviderLocatorResolver(
		locators.Factories(),
		config.Store.Catalog,
	)
	if err != nil {
		return nil, err
	}
	resolver, err := composition.New(composition.ResolverOptions{
		Artifacts:                    config.Store.Artifacts,
		Catalog:                      config.Store.Catalog,
		SourceEntries:                config.Store.Resources,
		Locators:                     locatorResolver,
		Interpretations:              config.Interpretations,
		Scope:                        config.Scope,
		DirectCapabilities:           config.DirectCapabilities,
		ArtifactCapabilityProjectors: config.ArtifactCapabilityProjectors,
		Limits:                       composition.DefaultLimits(),
	})
	if err != nil {
		return nil, err
	}
	return &Artifactory{
		interpretations: config.Interpretations,
		composition:     resolver,
	}, nil
}
