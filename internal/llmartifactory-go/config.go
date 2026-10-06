package llmartifactory

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	corelocator "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

// Config contains only the generic capabilities needed to build LLM
// declaration composition plus separately named LLM-domain registrations. It deliberately
// does not combine source drivers, persistence, runtime execution, content,
// schemas, decoders, and locator behavior into a universal provider object.
type Config struct {
	Artifacts artifact.API
	Catalog   catalog.API
	Resources resourceFlow.API

	Interpretations  *coreinterpretation.Registry
	LocatorFactories []corelocator.Factory

	Scope composition.ScopeBinding

	DirectCapabilities           []composition.DirectCapabilityProvider
	ArtifactCapabilityProjectors map[declaration.Type]composition.ArtifactCapabilityProjector
}
