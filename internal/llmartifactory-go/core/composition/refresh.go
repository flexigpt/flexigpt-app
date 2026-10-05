package composition

import (
	"context"
	"fmt"
	"sort"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

func (r *Resolver) RefreshPlugin(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	return r.refreshTyped(ctx, ref, declaration.TypePlugin)
}

func (r *Resolver) RefreshAgent(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	return r.refreshTyped(ctx, ref, declaration.TypeAgent)
}

func (r *Resolver) RefreshTeam(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	return r.refreshTyped(ctx, ref, declaration.TypeTeam)
}

func (r *Resolver) RefreshWorkspace(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	return r.refreshTyped(ctx, ref, declaration.TypeWorkspace)
}

func (r *Resolver) RefreshWorkspaceWithCompositionSource(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	compositionSourceID sourceModel.SourceID,
) error {
	if err := compositionSourceID.Validate(); err != nil {
		return err
	}
	return r.refreshTypedWithCompositionSource(
		ctx,
		ref,
		declaration.TypeWorkspace,
		compositionSourceID,
	)
}

func (r *Resolver) refreshTyped(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expected declaration.Type,
) error {
	return r.refreshTypedWithCompositionSource(ctx, ref, expected, "")
}

func (r *Resolver) refreshTypedWithCompositionSource(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expected declaration.Type,
	compositionSourceID sourceModel.SourceID,
) error {
	if r == nil || r.refresh == nil {
		return fmt.Errorf(
			"%w: Artifact composition refresh coordinator is unavailable",
			spec.ErrUnsupported,
		)
	}
	if err := validateResolutionContext(ctx); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}

	refreshed := make(map[RefreshTarget]struct{})
	for pass := 0; pass <= r.limits.MaxDepth; pass++ {
		var (
			rootEntry *ResolvedEntry
			err       error
		)
		if compositionSourceID != "" {
			rootEntry, err = r.ResolveWorkspaceWithCompositionSource(
				ctx,
				ref,
				compositionSourceID,
			)
		} else {
			rootEntry, err = r.resolveTyped(ctx, ref, expected)
		}
		if err != nil {
			return err
		}

		rootID, found := rootEntry.RootID()
		if !found {
			return fmt.Errorf(
				"%w: composition refresh root has no Root identity",
				spec.ErrInvalid,
			)
		}

		walker := refreshWalker{
			resolver:   r,
			rootID:     rootID,
			directives: make(map[RefreshTarget]bool),
			visited:    make(map[artifactModel.ArtifactRef]struct{}),
		}
		if err := walker.visitEntry(ctx, rootEntry); err != nil {
			return err
		}

		targets := make([]RefreshTarget, 0, len(walker.directives))
		for target, changed := range walker.directives {
			if target.RootID != rootID {
				return fmt.Errorf(
					"%w: refresh coordinator returned another Root",
					spec.ErrInvalid,
				)
			}
			if _, done := refreshed[target]; !done || changed {
				targets = append(targets, target)
			}
		}
		if len(targets) == 0 {
			return nil
		}

		sort.Slice(targets, func(left, right int) bool {
			if targets[left].RootID != targets[right].RootID {
				return targets[left].RootID < targets[right].RootID
			}
			return targets[left].SourceID < targets[right].SourceID
		})
		for _, target := range targets {
			if err := r.refresh.RefreshSource(ctx, target); err != nil {
				return fmt.Errorf(
					"refresh declaration Source %q: %w",
					target.SourceID,
					err,
				)
			}
			refreshed[target] = struct{}{}
		}
	}

	return fmt.Errorf(
		"%w: Artifact composition refresh closure exceeds depth %d",
		spec.ErrLocatorLimitExceeded,
		r.limits.MaxDepth,
	)
}

type refreshWalker struct {
	resolver *Resolver
	rootID   rootModel.RootID

	directives map[RefreshTarget]bool
	visited    map[artifactModel.ArtifactRef]struct{}
}

func (w *refreshWalker) visitEntry(
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

func (w *refreshWalker) visitRelationship(
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
		directives, err := w.resolver.refresh.PrepareSelectorDiscovery(
			ctx,
			SelectorRefreshRequest{
				Parent:   parent.DeclarationOrigin.Clone(),
				Selector: selector,
			},
		)
		if err != nil {
			return err
		}
		if err := w.addDirectives(directives); err != nil {
			return err
		}

	case declaration.MemberNamed:
		if relationship.Declared.Header().Locator != nil {
			directives, err := w.resolver.refresh.PrepareLocatedMemberDiscovery(
				ctx,
				LocatedMemberRefreshRequest{
					Parent: parent.DeclarationOrigin.Clone(),
					Member: relationship.Declared.Clone(),
				},
			)
			if err != nil {
				return err
			}
			if err := w.addDirectives(directives); err != nil {
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

func (w *refreshWalker) addDirectives(
	directives []RefreshDirective,
) error {
	for _, directive := range directives {
		if err := directive.Target.Validate(); err != nil {
			return err
		}
		w.directives[directive.Target] =
			w.directives[directive.Target] || directive.Changed
	}
	return nil
}
