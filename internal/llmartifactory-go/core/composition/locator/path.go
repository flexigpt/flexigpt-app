package locator

import (
	"context"
	"fmt"
	"slices"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

const pathFactoryRevision = "artifact-path-locator/v1"

// PathFactory resolves portable scalar and typed path locators against the
// declaring Artifact's Source. It reads committed catalog state only.
type PathFactory struct {
	interpretations *coreinterpretation.Registry
	kinds           []artifactModel.ArtifactKind
}

func NewPathFactory(
	interpretations *coreinterpretation.Registry,
) (*PathFactory, error) {
	if interpretations == nil {
		return nil, fmt.Errorf(
			"%w: path locator interpretation registry is nil",
			spec.ErrInvalid,
		)
	}

	seen := make(map[artifactModel.ArtifactKind]struct{})
	kinds := make([]artifactModel.ArtifactKind, 0)
	for _, key := range interpretations.SchemaKeys() {
		kind := artifactModel.ArtifactKind(key.Kind)
		if err := kind.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[kind]; duplicate {
			continue
		}
		seen[kind] = struct{}{}
		kinds = append(kinds, kind)
	}
	if len(kinds) == 0 {
		return nil, fmt.Errorf(
			"%w: path locator has no supported Artifact kinds",
			spec.ErrInvalid,
		)
	}
	slices.Sort(kinds)

	return &PathFactory{
		interpretations: interpretations,
		kinds:           kinds,
	}, nil
}

func (*PathFactory) LocatorKind() string {
	return "path"
}

func (p *PathFactory) ArtifactKinds() []artifactModel.ArtifactKind {
	if p == nil {
		return nil
	}
	return append([]artifactModel.ArtifactKind(nil), p.kinds...)
}

func (*PathFactory) Revision() string {
	return pathFactoryRevision
}

func (p *PathFactory) Bind(
	runtime Runtime,
) (Resolver, error) {
	if p == nil || p.interpretations == nil {
		return nil, spec.ErrClosed
	}
	if runtime == nil {
		return nil, fmt.Errorf(
			"%w: path locator runtime is nil",
			spec.ErrInvalid,
		)
	}
	return &pathResolver{
		runtime:         runtime,
		interpretations: p.interpretations,
	}, nil
}

type pathResolver struct {
	runtime         Runtime
	interpretations *coreinterpretation.Registry
}

func (r *pathResolver) Resolve(
	ctx context.Context,
	request Request,
) (artifactModel.ArtifactRef, error) {
	if r == nil || r.runtime == nil || r.interpretations == nil {
		return artifactModel.ArtifactRef{}, spec.ErrClosed
	}
	if ctx == nil {
		return artifactModel.ArtifactRef{}, fmt.Errorf(
			"%w: path locator context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	if err := request.Validate(); err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	if request.From == nil {
		return artifactModel.ArtifactRef{}, fmt.Errorf(
			"%w: path locator requires a source-backed declaration origin",
			spec.ErrLocatorUnresolved,
		)
	}

	target, err := declaration.ResolveSourceRelativePathLocator(
		request.Locator,
		request.From.Binding.Locator,
	)
	if err != nil {
		return artifactModel.ArtifactRef{}, err
	}

	candidates, err := r.interpretations.LocatorCandidates(
		request.ExpectedKind,
		target,
	)
	if err != nil {
		return artifactModel.ArtifactRef{}, err
	}

	entries, err := r.runtime.ListBySource(
		ctx,
		request.RootID,
		request.From.Binding.SourceID,
		catalogModel.ListOptions{
			Kind: request.ExpectedKind,
		},
	)
	if err != nil {
		return artifactModel.ArtifactRef{}, err
	}

	seen := make(map[artifactModel.ArtifactRef]struct{})
	matches := make([]artifactModel.ArtifactRef, 0, 1)

	for _, candidate := range candidates {
		for _, entry := range entries {
			if entry.State != artifactModel.StateAvailable ||
				entry.Kind != request.ExpectedKind ||
				entry.Binding.Locator != candidate ||
				entry.Binding.SubresourceLocator != "" {
				continue
			}
			if request.ExpectedLogicalName != "" &&
				entry.LogicalName != request.ExpectedLogicalName {
				continue
			}

			ref := entry.Ref()
			if _, duplicate := seen[ref]; duplicate {
				continue
			}
			seen[ref] = struct{}{}
			matches = append(matches, ref)
		}
	}

	switch len(matches) {
	case 0:
		return artifactModel.ArtifactRef{}, fmt.Errorf(
			"%w: path locator %q did not resolve %q",
			spec.ErrLocatorUnresolved,
			request.Locator,
			request.ExpectedLogicalName,
		)

	case 1:
		return matches[0], nil

	default:
		return artifactModel.ArtifactRef{}, fmt.Errorf(
			"%w: path locator %q resolves to %d Artifacts",
			spec.ErrIdentityConflict,
			request.Locator,
			len(matches),
		)
	}
}
