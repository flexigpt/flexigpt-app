package llmartifactory

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

// Artifactory owns immutable LLM declaration interpretation and composition
// state attached to an already-open generic Artifact Store.
//
// It deliberately exposes only the two shared LLM capabilities required by
// family construction. It does not mirror generic Store APIs, application
// topology, runtime adapters, content inventory, or Wails transport.
type Artifactory struct {
	interpretations *coreinterpretation.Registry
	composition     *composition.Resolver
}

func (a *Artifactory) Interpretations() *coreinterpretation.Registry {
	return a.interpretations
}

// Composition returns the one composition owner assembled for this LLM
// attachment. Families consume this resolver rather than creating private
// locator registries or graph resolvers.
func (a *Artifactory) Composition() *composition.Resolver {
	return a.composition
}
