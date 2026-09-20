package resolve

import (
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// TypeResolver is one registered Artifact-type resolver definition.
//
// Type-specific structure expansion remains private to this package. The
// registry is the single place that declares selector eligibility and fallback
// capability for each portable Artifact type.
type TypeResolver interface {
	Type() declaration.Type
	SupportsSelectors() bool
	SupportsMappedFallbackTargets() bool
	FallbackProvider() FallbackProvider
}

type registeredTypeResolver struct {
	declarationType declaration.Type
	selectors       bool
	mappedFallback  bool
	fallback        FallbackProvider
}

func (r registeredTypeResolver) Type() declaration.Type {
	return r.declarationType
}

func (r registeredTypeResolver) SupportsSelectors() bool {
	return r.selectors
}

func (r registeredTypeResolver) SupportsMappedFallbackTargets() bool {
	return r.mappedFallback
}

func (r registeredTypeResolver) FallbackProvider() FallbackProvider {
	return r.fallback
}

type fallbackTypeResolver struct {
	TypeResolver

	fallback FallbackProvider
}

func (r fallbackTypeResolver) FallbackProvider() FallbackProvider {
	return r.fallback
}

type Registry struct {
	resolvers map[declaration.Type]TypeResolver
}

func NewRegistry(resolvers ...TypeResolver) (*Registry, error) {
	output := &Registry{
		resolvers: make(map[declaration.Type]TypeResolver, len(resolvers)),
	}
	for _, resolver := range resolvers {
		if err := output.Register(resolver); err != nil {
			return nil, err
		}
	}
	return output, nil
}

func DefaultRegistry() *Registry {
	policies := documentTopology.ResolverTypePolicies()
	values := make([]TypeResolver, 0, len(policies))
	for _, policy := range policies {
		values = append(values, registeredTypeResolver{
			declarationType: policy.Type,
			selectors:       policy.SupportsSelectors,
			mappedFallback:  policy.SupportsMappedFallbackTargets,
		})
	}

	registry, err := NewRegistry(values...)
	if err != nil {
		panic(err)
	}
	return registry
}

func (r *Registry) Register(resolver TypeResolver) error {
	if r == nil {
		return fmt.Errorf(
			"%w: Artifact resolver registry is nil",
			basespec.ErrInvalid,
		)
	}
	if resolver == nil {
		return fmt.Errorf(
			"%w: Artifact type resolver is nil",
			basespec.ErrInvalid,
		)
	}
	if err := resolver.Type().Validate(); err != nil {
		return err
	}
	if _, duplicate := r.resolvers[resolver.Type()]; duplicate {
		return fmt.Errorf(
			"%w: Artifact resolver %q is already registered",
			basespec.ErrConflict,
			resolver.Type(),
		)
	}
	r.resolvers[resolver.Type()] = resolver
	return nil
}

func (r *Registry) Resolver(
	declarationType declaration.Type,
) (TypeResolver, bool) {
	if r == nil {
		return nil, false
	}
	value, found := r.resolvers[declarationType]
	return value, found
}

func (r *Registry) ValidateComplete() error {
	if r == nil {
		return fmt.Errorf(
			"%w: Artifact resolver registry is nil",
			basespec.ErrInvalid,
		)
	}
	for _, declarationType := range declaration.Types() {
		if _, found := r.Resolver(declarationType); !found {
			return fmt.Errorf(
				"%w: no resolver is registered for Artifact type %q",
				basespec.ErrInvalid,
				declarationType,
			)
		}
	}
	return nil
}

func (r *Registry) Clone() *Registry {
	if r == nil {
		return nil
	}
	output := &Registry{
		resolvers: make(map[declaration.Type]TypeResolver, len(r.resolvers)),
	}
	maps.Copy(output.resolvers, r.resolvers)
	return output
}

func (r *Registry) WithFallback(
	declarationType declaration.Type,
	provider FallbackProvider,
) (*Registry, error) {
	if r == nil || provider == nil {
		return nil, fmt.Errorf(
			"%w: Artifact fallback registration is incomplete",
			basespec.ErrInvalid,
		)
	}
	resolver, found := r.Resolver(declarationType)
	if !found {
		return nil, fmt.Errorf(
			"%w: no resolver is registered for fallback type %q",
			basespec.ErrUnsupported,
			declarationType,
		)
	}
	output := r.Clone()
	output.resolvers[declarationType] = fallbackTypeResolver{
		TypeResolver: resolver,
		fallback:     provider,
	}
	return output, nil
}
