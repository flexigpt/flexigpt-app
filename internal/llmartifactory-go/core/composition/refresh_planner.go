package composition

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	corerefresh "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/refresh"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

// RefreshRootResolver resolves the family-selected root declaration for an
// explicit refresh operation. Workspace supplies composition-source binding
// through this function without giving normal resolution refresh authority.
type RefreshRootResolver func(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (*ResolvedEntry, error)

type SelectorDiscoveryRequest struct {
	Parent   artifactModel.Artifact
	Selector declaration.Selector
}

func (r SelectorDiscoveryRequest) Validate() error {
	if err := r.Parent.ValidateRead(); err != nil {
		return err
	}
	return r.Selector.Validate()
}

type LocatedMemberDiscoveryRequest struct {
	Parent artifactModel.Artifact
	Member declaration.Entry
}

func (r LocatedMemberDiscoveryRequest) Validate() error {
	if err := r.Parent.ValidateRead(); err != nil {
		return err
	}
	if err := r.Member.Validate(); err != nil {
		return err
	}
	if r.Member.Header().Locator == nil {
		return fmt.Errorf(
			"%w: located-member discovery requires a declaration locator",
			spec.ErrInvalid,
		)
	}
	return nil
}

// DiscoveryRequirementProvider is family-owned interpretation of which Source
// discovery requirement one selector or local locator needs. It cannot refresh
// a Source or mutate generic Store state itself.
type DiscoveryRequirementProvider interface {
	RequirementsForSelector(
		ctx context.Context,
		request SelectorDiscoveryRequest,
	) ([]corerefresh.Requirement, error)

	RequirementsForLocatedMember(
		ctx context.Context,
		request LocatedMemberDiscoveryRequest,
	) ([]corerefresh.Requirement, error)
}

// ReachableDiscoveryPlanner owns generic traversal of an already resolved
// declaration graph. The child refresh package owns Source preparation and
// refresh closure execution.
type ReachableDiscoveryPlanner struct {
	resolveRoot  RefreshRootResolver
	requirements DiscoveryRequirementProvider
}

func NewReachableDiscoveryPlanner(
	resolveRoot RefreshRootResolver,
	requirements DiscoveryRequirementProvider,
) (*ReachableDiscoveryPlanner, error) {
	if resolveRoot == nil || requirements == nil {
		return nil, fmt.Errorf(
			"%w: composition refresh planner dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	return &ReachableDiscoveryPlanner{
		resolveRoot:  resolveRoot,
		requirements: requirements,
	}, nil
}

func (p *ReachableDiscoveryPlanner) ReachableRequirements(
	ctx context.Context,
	root artifactModel.ArtifactRef,
) ([]corerefresh.Requirement, error) {
	if err := root.Validate(); err != nil {
		return nil, err
	}

	entry, err := p.resolveRoot(ctx, root)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, fmt.Errorf(
			"%w: composition refresh root resolved to nil",
			spec.ErrInvalid,
		)
	}
	rootID, found := entry.RootID()
	if !found {
		return nil, fmt.Errorf(
			"%w: composition refresh root has no Root identity",
			spec.ErrInvalid,
		)
	}

	walker := reachableDiscoveryWalker{
		rootID:       rootID,
		requirements: p.requirements,
		visited:      make(map[artifactModel.ArtifactRef]struct{}),
	}
	if err := walker.visitEntry(ctx, entry); err != nil {
		return nil, err
	}
	return walker.values, nil
}

type reachableDiscoveryWalker struct {
	rootID       rootModel.RootID
	requirements DiscoveryRequirementProvider
	visited      map[artifactModel.ArtifactRef]struct{}
	values       []corerefresh.Requirement
}

func (w *reachableDiscoveryWalker) visitEntry(
	ctx context.Context,
	entry *ResolvedEntry,
) error {
	if entry == nil {
		return nil
	}
	if ref, found := entry.ArtifactRef(); found {
		if _, visited := w.visited[ref]; visited {
			return nil
		}
		w.visited[ref] = struct{}{}
	}

	for _, relationship := range entry.Relationships {
		if err := w.visitRelationship(ctx, entry, relationship); err != nil {
			return err
		}
	}
	return nil
}

func (w *reachableDiscoveryWalker) visitRelationship(
	ctx context.Context,
	parent *ResolvedEntry,
	relationship ResolvedRelationship,
) error {
	if parent == nil || parent.DeclarationOrigin == nil {
		return nil
	}

	switch relationship.Form {
	case declaration.MemberSelector:
		selector, err := relationship.Declared.Selector()
		if err != nil {
			return err
		}
		values, err := w.requirements.RequirementsForSelector(
			ctx,
			SelectorDiscoveryRequest{
				Parent:   parent.DeclarationOrigin.Clone(),
				Selector: selector,
			},
		)
		if err != nil {
			return err
		}
		if err := w.appendRequirements(values); err != nil {
			return err
		}

	case declaration.MemberNamed:
		if relationship.Declared.Header().Locator != nil {
			values, err := w.requirements.RequirementsForLocatedMember(
				ctx,
				LocatedMemberDiscoveryRequest{
					Parent: parent.DeclarationOrigin.Clone(),
					Member: relationship.Declared.Clone(),
				},
			)
			if err != nil {
				return err
			}
			if err := w.appendRequirements(values); err != nil {
				return err
			}
		}
	default:
	}

	if relationship.Resolved != nil {
		if err := w.visitEntry(ctx, relationship.Resolved); err != nil {
			return err
		}
	}
	if relationship.Selector != nil {
		for _, match := range relationship.Selector.Matches {
			if err := w.visitEntry(ctx, match.Resolved); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *reachableDiscoveryWalker) appendRequirements(
	values []corerefresh.Requirement,
) error {
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return err
		}
		if value.RootID != w.rootID {
			return fmt.Errorf(
				"%w: composition refresh requirement escaped its Root",
				spec.ErrInvalid,
			)
		}
		copyValue := value
		copyValue.Discovery = value.Discovery.Clone()
		w.values = append(w.values, copyValue)
	}
	return nil
}
