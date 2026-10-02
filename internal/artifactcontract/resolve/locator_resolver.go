package resolve

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// ProviderLocatorResolver adapts provider-owned locator resolver factories to
// the typed contract resolver locator boundary.
type ProviderLocatorResolver struct {
	resolvers map[provider.LocatorResolverKey]provider.BoundLocatorResolver
}

func NewProviderLocatorResolver(
	factories []provider.LocatorResolverFactory,
	runtime provider.LocatorRuntime,
) (*ProviderLocatorResolver, error) {
	if runtime == nil {
		return nil, fmt.Errorf(
			"%w: provider locator resolver runtime is nil",
			spec.ErrInvalid,
		)
	}

	output := &ProviderLocatorResolver{
		resolvers: make(
			map[provider.LocatorResolverKey]provider.BoundLocatorResolver,
		),
	}
	for index, factory := range factories {
		if err := provider.ValidateLocatorResolverFactory(factory); err != nil {
			return nil, fmt.Errorf(
				"locator resolver factory %d: %w",
				index,
				err,
			)
		}
		bound, err := factory.BindLocatorRuntime(runtime)
		if err != nil {
			return nil, fmt.Errorf(
				"bind locator resolver %q: %w",
				factory.LocatorKind(),
				err,
			)
		}
		if bound == nil {
			return nil, fmt.Errorf(
				"%w: locator resolver %q bound to nil",
				spec.ErrInvalid,
				factory.LocatorKind(),
			)
		}
		for _, artifactKind := range factory.ArtifactKinds() {
			key := provider.LocatorResolverKey{
				LocatorKind:  factory.LocatorKind(),
				ArtifactKind: artifactKind,
			}
			if _, duplicate := output.resolvers[key]; duplicate {
				return nil, fmt.Errorf(
					"%w: duplicate locator resolver %q for %q",
					spec.ErrConflict,
					key.LocatorKind,
					key.ArtifactKind,
				)
			}
			output.resolvers[key] = bound
		}
	}
	return output, nil
}

func (r *ProviderLocatorResolver) ResolveArtifactLocator(
	ctx context.Context,
	request LocatorRequest,
) (artifact.ArtifactRef, error) {
	if r == nil {
		return artifact.ArtifactRef{}, spec.ErrClosed
	}
	if err := request.RootID.Validate(); err != nil {
		return artifact.ArtifactRef{}, err
	}
	if err := request.ExpectedType.Validate(); err != nil {
		return artifact.ArtifactRef{}, err
	}
	if request.ExpectedLogicalName != "" {
		if err := request.ExpectedLogicalName.Validate(); err != nil {
			return artifact.ArtifactRef{}, err
		}
	}

	locatorKind, err := providerLocatorKind(request.Locator)
	if err != nil {
		return artifact.ArtifactRef{}, err
	}
	key := provider.LocatorResolverKey{
		LocatorKind:  locatorKind,
		ArtifactKind: artifact.ArtifactKind(request.ExpectedType),
	}
	resolver, found := r.resolvers[key]
	if !found {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: no %q locator resolver is registered for Artifact kind %q",
			spec.ErrLocatorUnresolved,
			locatorKind,
			key.ArtifactKind,
		)
	}

	locatorJSON, err := request.Locator.MarshalJSON()
	if err != nil {
		return artifact.ArtifactRef{}, err
	}
	entryJSON, err := request.Entry.CanonicalJSON()
	if err != nil {
		return artifact.ArtifactRef{}, err
	}
	ref, err := resolver.ResolveLocator(
		ctx,
		provider.LocatorResolutionRequest{
			RootID:              request.RootID,
			From:                cloneArtifactPointer(request.From),
			LocatorJSON:         locatorJSON,
			EntryJSON:           entryJSON,
			ExpectedKind:        key.ArtifactKind,
			ExpectedLogicalName: request.ExpectedLogicalName,
		},
	)
	if err != nil {
		return artifact.ArtifactRef{}, err
	}
	if err := ref.Validate(); err != nil {
		return artifact.ArtifactRef{}, err
	}
	if ref.RootID != request.RootID {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: locator resolver returned an Artifact from another Root",
			spec.ErrInvalid,
		)
	}
	return ref, nil
}

func providerLocatorKind(
	locator declaration.Locator,
) (string, error) {
	if err := locator.Validate(); err != nil {
		return "", err
	}
	if locator.Kind != "" {
		return string(locator.Kind), nil
	}

	parsed, err := url.Parse(locator.Scalar)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" {
		return "path", nil
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return "url", nil
	default:
		return strings.ToLower(parsed.Scheme), nil
	}
}
