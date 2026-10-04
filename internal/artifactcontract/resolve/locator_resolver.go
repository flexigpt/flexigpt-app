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

type ProviderLocatorResolver struct {
	resolvers map[locator.FactoryKey]locator.Resolver
}

func NewProviderLocatorResolver(
	factories []locator.Factory,
	runtime locator.Runtime,
) (*ProviderLocatorResolver, error) {
	if runtime == nil {
		return nil, fmt.Errorf("%w: locator catalog runtime is nil", spec.ErrInvalid)
	}
	output := &ProviderLocatorResolver{
		resolvers: make(map[locator.FactoryKey]locator.Resolver),
	}
	for index, factory := range factories {
		if err := locator.ValidateFactory(factory); err != nil {
			return nil, fmt.Errorf("locator resolver factory %d: %w", index, err)
		}
		bound, err := factory.Bind(runtime)
		if err != nil {
			return nil, fmt.Errorf("bind locator resolver %q: %w", factory.LocatorKind(), err)
		}
		if bound == nil {
			return nil, fmt.Errorf("%w: locator factory bound a nil resolver", spec.ErrInvalid)
		}
		for _, kind := range factory.ArtifactKinds() {
			key := locator.FactoryKey{LocatorKind: factory.LocatorKind(), ArtifactKind: kind}
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
	if err := validateResolutionContext(ctx); err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	if err := request.ExpectedType.Validate(); err != nil {
		return artifactModel.ArtifactRef{}, err
	}

	entry := request.Entry.Clone()
	input := locator.Request{
		RootID:              request.RootID,
		From:                cloneArtifactPointer(request.From),
		Locator:             request.Locator.Clone(),
		Entry:               &entry,
		ExpectedKind:        artifactModel.ArtifactKind(request.ExpectedType),
		ExpectedLogicalName: request.ExpectedLogicalName,
	}
	if err := input.Validate(); err != nil {
		return artifactModel.ArtifactRef{}, err
	}

	kind, err := providerLocatorKind(input.Locator)
	if err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	key := locator.FactoryKey{LocatorKind: kind, ArtifactKind: input.ExpectedKind}
	resolver, found := r.resolvers[key]
	if !found {
		return artifactModel.ArtifactRef{}, fmt.Errorf(
			"%w: no %q locator resolver is registered for %q",
			spec.ErrLocatorUnresolved,
			kind,
			input.ExpectedKind,
		)
	}
	ref, err := resolver.Resolve(ctx, input)
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

func providerLocatorKind(loc declaration.Locator) (string, error) {
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
