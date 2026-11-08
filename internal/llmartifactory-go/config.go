package llmartifactory

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	corelocator "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/registration"
)

// Config contains generic Artifact Store capabilities and one declaration
// registration source. Application callers can pass either Declarations or
// the explicit interpretation and locator inputs, but not both.
type Config struct {
	Artifacts artifact.API
	Catalog   catalog.API
	Resources resourceFlow.API

	Declarations *registration.Bundle

	// Explicit declaration inputs remain supported for focused integration
	// assembly. Declarations is the preferred application production path.
	Interpretations  *coreinterpretation.Registry
	LocatorFactories []corelocator.Factory

	Scope composition.ScopeBinding

	DirectCapabilities           []composition.DirectCapabilityProvider
	ArtifactCapabilityProjectors map[declaration.Type]composition.ArtifactCapabilityProjector
}
