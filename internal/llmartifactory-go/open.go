package llmartifactory

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	corelocator "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
)

// Open attaches the LLM artifact-domain registration set to an already opened
// generic Artifact Store capability set. Generic Store ownership remains with
// deployment assembly; this constructor does not open or close provider
// resources and does not receive the broad compose.Store aggregate.
func Open(ctx context.Context, config Config) (*Artifactory, error) {
	if config.Artifacts == nil ||
		config.Catalog == nil ||
		config.Resources == nil {
		return nil, fmt.Errorf(
			"%w: LLM Artifactory generic composition dependencies are incomplete",
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
		config.Catalog,
	)
	if err != nil {
		return nil, err
	}
	resolver, err := composition.New(composition.ResolverOptions{
		Artifacts:                    config.Artifacts,
		Catalog:                      config.Catalog,
		SourceEntries:                config.Resources,
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
