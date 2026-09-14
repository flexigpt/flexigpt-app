package resolve

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

// ProviderLocatorResolver adapts provider-owned generic locator factories to
// the typed Artifact Resolver boundary.
type ProviderLocatorResolver struct {
	resolvers map[providerapi.LocatorResolverKey]providerapi.BoundLocatorResolver
}

func NewProviderLocatorResolver(
	factories []providerapi.LocatorResolverFactory,
	runtime providerapi.LocatorRuntime,
) (*ProviderLocatorResolver, error) {
	if runtime == nil {
		return nil, fmt.Errorf(
			"%w: provider locator resolver runtime is nil",
			basespec.ErrInvalid,
		)
	}

	output := &ProviderLocatorResolver{
		resolvers: make(
			map[providerapi.LocatorResolverKey]providerapi.BoundLocatorResolver,
		),
	}
	for index, factory := range factories {
		if err := providerapi.ValidateLocatorResolverFactory(factory); err != nil {
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
				basespec.ErrInvalid,
				factory.LocatorKind(),
			)
		}
		for _, artifactKind := range factory.ArtifactKinds() {
			key := providerapi.LocatorResolverKey{
				LocatorKind:  factory.LocatorKind(),
				ArtifactKind: artifactKind,
			}
			if _, duplicate := output.resolvers[key]; duplicate {
				return nil, fmt.Errorf(
					"%w: duplicate bound locator resolver %q for %q",
					basespec.ErrConflict,
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
		return artifact.ArtifactRef{}, basespec.ErrClosed
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
	key := providerapi.LocatorResolverKey{
		LocatorKind:  locatorKind,
		ArtifactKind: artifact.ArtifactKind(request.ExpectedType),
	}
	resolver, found := r.resolvers[key]
	if !found {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: no %q locator resolver is registered for Artifact kind %q",
			basespec.ErrLocatorUnresolved,
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
		providerapi.LocatorResolutionRequest{
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
			basespec.ErrInvalid,
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
