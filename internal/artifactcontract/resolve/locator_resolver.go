package resolve

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/locator"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// ProviderLocatorResolver adapts provider-owned locator resolver factories to
// the typed contract resolver locator boundary.
type ProviderLocatorResolver struct {
	resolvers map[locator.FactoryKey]locator.Resolver
}

func NewProviderLocatorResolver(
	factories []locator.Factory,
	runtime locator.Runtime,
) (*ProviderLocatorResolver, error) {
	if runtime == nil {
		return nil, fmt.Errorf(
			"%w: provider locator resolver runtime is nil",
			spec.ErrInvalid,
		)
	}

	output := &ProviderLocatorResolver{
		resolvers: make(
			map[locator.FactoryKey]locator.Resolver,
		),
	}
	for index, factory := range factories {
		if err := locator.ValidateFactory(factory); err != nil {
			return nil, fmt.Errorf(
				"locator resolver factory %d: %w",
				index,
				err,
			)
		}
		bound, err := factory.Bind(runtime)
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
			key := locator.FactoryKey{
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
) (artifactModel.ArtifactRef, error) {
	if r == nil {
		return artifactModel.ArtifactRef{}, spec.ErrClosed
	}
	if err := request.RootID.Validate(); err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	if err := request.ExpectedType.Validate(); err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	if request.ExpectedLogicalName != "" {
		if err := request.ExpectedLogicalName.Validate(); err != nil {
			return artifactModel.ArtifactRef{}, err
		}
	}

	locatorKind, err := providerLocatorKind(request.Locator)
	if err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	key := locator.FactoryKey{
		LocatorKind:  locatorKind,
		ArtifactKind: artifactModel.ArtifactKind(request.ExpectedType),
	}
	resolver, found := r.resolvers[key]
	if !found {
		return artifactModel.ArtifactRef{}, fmt.Errorf(
			"%w: no %q locator resolver is registered for Artifact kind %q",
			spec.ErrLocatorUnresolved,
			locatorKind,
			key.ArtifactKind,
		)
	}

	locatorJSON, err := request.Locator.MarshalJSON()
	if err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	entryJSON, err := request.Entry.CanonicalJSON()
	if err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	ref, err := resolver.Resolve(
		ctx,
		locator.Request{
			RootID:              request.RootID,
			From:                cloneArtifactPointer(request.From),
			LocatorJSON:         locatorJSON,
			EntryJSON:           entryJSON,
			ExpectedKind:        key.ArtifactKind,
			ExpectedLogicalName: request.ExpectedLogicalName,
		},
	)
	if err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	if err := ref.Validate(); err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	if ref.RootID != request.RootID {
		return artifactModel.ArtifactRef{}, fmt.Errorf(
			"%w: locator resolver returned an Artifact from another Root",
			spec.ErrInvalid,
		)
	}
	return ref, nil
}

func providerLocatorKind(
	loc declaration.Locator,
) (string, error) {
	if err := loc.Validate(); err != nil {
		return "", err
	}
	if loc.Kind != "" {
		return string(loc.Kind), nil
	}

	parsed, err := url.Parse(loc.Scalar)
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
