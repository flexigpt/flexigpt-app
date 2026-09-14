// Package locatorpath implements portable source-relative path declaration
// loading. It intentionally does not resolve URL, Git, package, archive, or
// command locators.
package locatorpath

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

const locatorKindPath = "path"

// Planner adds the discovery state needed to load a path target and returns
// the source locators that can emit the requested Artifact.
type Planner interface {
	PlanPath(
		target basespec.Locator,
		discovery source.DiscoverySpec,
	) (
		next source.DiscoverySpec,
		candidates []basespec.Locator,
		err error,
	)
}

type Factory struct {
	artifactKinds []artifact.ArtifactKind
	planner       Planner
}

func NewFactory(
	artifactKinds []artifact.ArtifactKind,
	planner Planner,
) *Factory {
	return &Factory{
		artifactKinds: append(
			[]artifact.ArtifactKind(nil),
			artifactKinds...,
		),
		planner: planner,
	}
}

func NewCanonicalDeclarationFactory() *Factory {
	return NewFactory(
		[]artifact.ArtifactKind{
			artifact.ArtifactKind(declaration.TypeCollection),
			artifact.ArtifactKind(declaration.TypeAgent),
			artifact.ArtifactKind(declaration.TypeTeam),
			artifact.ArtifactKind(declaration.TypeLoop),
			artifact.ArtifactKind(declaration.TypeWorkflow),
			artifact.ArtifactKind(declaration.TypeWorkspace),
			artifact.ArtifactKind(declaration.TypeMCPPolicy),
		},
		canonicalDeclarationPlanner{},
	)
}

func (*Factory) LocatorKind() string {
	return locatorKindPath
}

func (f *Factory) ArtifactKinds() []artifact.ArtifactKind {
	if f == nil {
		return nil
	}
	return append([]artifact.ArtifactKind(nil), f.artifactKinds...)
}

func (*Factory) Revision() string {
	return "artifact-declaration-path/v1"
}

func (f *Factory) BindLocatorRuntime(
	runtime providerapi.LocatorRuntime,
) (providerapi.BoundLocatorResolver, error) {
	if f == nil || f.planner == nil || runtime == nil {
		return nil, fmt.Errorf(
			"%w: path locator resolver dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &boundResolver{
		artifactKinds: f.ArtifactKinds(),
		planner:       f.planner,
		runtime:       runtime,
	}, nil
}

type boundResolver struct {
	artifactKinds []artifact.ArtifactKind
	planner       Planner
	runtime       providerapi.LocatorRuntime
}

func (r *boundResolver) ResolveLocator(
	ctx context.Context,
	request providerapi.LocatorResolutionRequest,
) (artifact.ArtifactRef, error) {
	if r == nil || r.runtime == nil || r.planner == nil {
		return artifact.ArtifactRef{}, basespec.ErrClosed
	}
	if ctx == nil {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: path locator resolution context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return artifact.ArtifactRef{}, err
	}
	if err := request.Validate(); err != nil {
		return artifact.ArtifactRef{}, err
	}
	if request.From == nil {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: path declaration locator requires a source-backed origin Artifact",
			basespec.ErrLocatorUnresolved,
		)
	}
	if !slices.Contains(r.artifactKinds, request.ExpectedKind) {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: path locator resolver does not support Artifact kind %q",
			basespec.ErrUnsupported,
			request.ExpectedKind,
		)
	}

	var locator declaration.Locator
	if err := json.Unmarshal(request.LocatorJSON, &locator); err != nil {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: decode path locator: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	target, err := declaration.ResolveSourceRelativePathLocator(
		locator,
		request.From.Binding.Locator,
	)
	if err != nil {
		return artifact.ArtifactRef{}, err
	}

	candidates, err := r.ensureDiscovered(
		ctx,
		request.RootID,
		*request.From,
		target,
	)
	if err != nil {
		return artifact.ArtifactRef{}, err
	}
	return r.selectArtifact(ctx, request, candidates)
}

func (r *boundResolver) ensureDiscovered(
	ctx context.Context,
	rootID root.RootID,
	origin artifact.Artifact,
	target basespec.Locator,
) ([]basespec.Locator, error) {
	for range 3 {
		current, err := r.runtime.GetSource(
			ctx,
			rootID,
			origin.Binding.SourceID,
		)
		if err != nil {
			return nil, err
		}
		if !current.Enabled {
			return nil, fmt.Errorf(
				"%w: declaration Source %q is disabled",
				basespec.ErrSourceUnavailable,
				current.ID,
			)
		}

		next, candidates, err := r.planner.PlanPath(
			target,
			current.Discovery,
		)
		if err != nil {
			return nil, err
		}
		next = next.Normalized()
		if err := next.Validate(); err != nil {
			return nil, err
		}

		if !current.Discovery.Equal(next) {
			current, err = r.runtime.UpdateSource(
				ctx,
				rootID,
				current.ID,
				source.Update{
					ExpectedRevision: current.Revision,
					DisplayName:      current.DisplayName,
					Enabled:          current.Enabled,
					Discovery:        &next,
				},
			)
			if errors.Is(err, basespec.ErrConflict) {
				continue
			}
			if err != nil {
				return nil, err
			}
		}

		if _, err := r.runtime.RefreshSource(
			ctx,
			rootID,
			current.ID,
		); err != nil {
			return nil, err
		}
		return candidates, nil
	}
	return nil, fmt.Errorf(
		"%w: declaration Source changed during path locator resolution",
		basespec.ErrConflict,
	)
}

func (r *boundResolver) selectArtifact(
	ctx context.Context,
	request providerapi.LocatorResolutionRequest,
	locators []basespec.Locator,
) (artifact.ArtifactRef, error) {
	records, err := r.runtime.ListArtifactsBySource(
		ctx,
		request.RootID,
		request.From.Binding.SourceID,
	)
	if err != nil {
		return artifact.ArtifactRef{}, err
	}

	selectedLocators := make(map[basespec.Locator]struct{}, len(locators))
	for _, locator := range locators {
		selectedLocators[locator] = struct{}{}
	}

	candidates := make([]artifact.Artifact, 0)
	for _, record := range records {
		if record.Kind != request.ExpectedKind ||
			record.State != artifact.StateAvailable {
			continue
		}
		if _, found := selectedLocators[record.Binding.Locator]; !found {
			continue
		}
		if request.ExpectedLogicalName != "" &&
			record.LogicalName != request.ExpectedLogicalName {
			continue
		}
		candidates = append(candidates, record)
	}

	switch len(candidates) {
	case 0:
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: path locator did not produce %s/%s",
			basespec.ErrReferenceUnresolved,
			request.ExpectedKind,
			request.ExpectedLogicalName,
		)
	case 1:
		return candidates[0].Ref(), nil
	default:
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: path locator produced %d %s Artifacts",
			basespec.ErrIdentityConflict,
			len(candidates),
			request.ExpectedKind,
		)
	}
}

type canonicalDeclarationPlanner struct{}

func (canonicalDeclarationPlanner) PlanPath(
	target basespec.Locator,
	discovery source.DiscoverySpec,
) (source.DiscoverySpec, []basespec.Locator, error) {
	if target == "." {
		return source.DiscoverySpec{}, nil, fmt.Errorf(
			"%w: declaration path locator must identify a file",
			basespec.ErrInvalid,
		)
	}

	output := discovery.Clone()
	inScope, err := output.InScope(target)
	if err != nil {
		return source.DiscoverySpec{}, nil, err
	}
	if !inScope {
		output.ExplicitLocators = append(
			output.ExplicitLocators,
			target,
		)
	}
	return output, []basespec.Locator{target}, nil
}
