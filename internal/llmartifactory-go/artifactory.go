package llmartifactory

import (
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

// Artifactory owns only LLM in-memory composition and interpretation state.
// Generic Store ownership remains with deployment assembly.
type Artifactory struct {
	interpretations *coreinterpretation.Registry
	composition     *composition.Resolver
	mu              sync.RWMutex
	closed          bool
}

func (a *Artifactory) Interpretations() *coreinterpretation.Registry {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return nil
	}
	return a.interpretations
}

// Composition exposes the one LLM composition owner assembled for this
// Artifactory attachment. Families consume this shared resolver rather than
// constructing private graph resolvers.
func (a *Artifactory) Composition() *composition.Resolver {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return nil
	}
	return a.composition
}
