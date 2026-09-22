package resolve

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

func (r *Resolver) RefreshPlugin(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	return r.refreshTyped(ctx, ref, declaration.TypePlugin)
}

func (r *Resolver) RefreshAgent(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	return r.refreshTyped(ctx, ref, declaration.TypeAgent)
}

func (r *Resolver) RefreshTeam(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	return r.refreshTyped(ctx, ref, declaration.TypeTeam)
}

func (r *Resolver) RefreshWorkspace(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	return r.refreshTyped(ctx, ref, declaration.TypeWorkspace)
}

func (r *Resolver) RefreshWorkspaceWithCompositionSource(
	ctx context.Context,
	ref artifact.ArtifactRef,
	compositionSourceID source.SourceID,
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
	ref artifact.ArtifactRef,
	expected declaration.Type,
) error {
	return r.refreshTypedWithCompositionSource(ctx, ref, expected, "")
}

func (r *Resolver) refreshTypedWithCompositionSource(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expected declaration.Type,
	compositionSourceID source.SourceID,
) error {
	if r == nil || r.refresh == nil {
		return fmt.Errorf(
			"%w: Artifact refresh coordinator is unavailable",
			basespec.ErrUnsupported,
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
				"%w: refresh root has no Root identity",
				basespec.ErrInvalid,
			)
		}

		walker := refreshWalker{
			resolver:   r,
			rootID:     rootID,
			directives: make(map[RefreshTarget]bool),
			visited:    make(map[artifact.ArtifactRef]struct{}),
		}
		if err := walker.visitEntry(ctx, rootEntry); err != nil {
			return err
		}

		targets := make([]RefreshTarget, 0, len(walker.directives))
		for target, changed := range walker.directives {
			if target.RootID != rootID {
				return fmt.Errorf(
					"%w: refresh coordinator returned another Root",
					basespec.ErrInvalid,
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
		"%w: Artifact refresh closure exceeds depth %d",
		basespec.ErrLocatorLimitExceeded,
		r.limits.MaxDepth,
	)
}

type refreshWalker struct {
	resolver   *Resolver
	rootID     root.RootID
	directives map[RefreshTarget]bool
	visited    map[artifact.ArtifactRef]struct{}
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

	switch entry.Type {
	case declaration.TypePlugin,
		declaration.TypeAgent,
		declaration.TypeTeam:
		for _, relationship := range entry.MemberResults {
			if err := w.visitRelationship(ctx, entry, relationship); err != nil {
				return err
			}
		}
		if entry.DirectLoopResult != nil {
			if err := w.visitRelationship(
				ctx,
				entry,
				*entry.DirectLoopResult,
			); err != nil {
				return err
			}
		}
		if entry.DirectWorkflowResult != nil {
			if err := w.visitRelationship(
				ctx,
				entry,
				*entry.DirectWorkflowResult,
			); err != nil {
				return err
			}
		}

	case declaration.TypeSkill:
		for _, relationship := range entry.AllowedToolResults {
			if err := w.visitRelationship(ctx, entry, relationship); err != nil {
				return err
			}
		}

	case declaration.TypeMCP:
		if entry.MCP != nil && entry.MCP.PolicyResult != nil {
			return w.visitRelationship(ctx, entry, *entry.MCP.PolicyResult)
		}

	case declaration.TypeLoop:
		if entry.Loop != nil && entry.Loop.BodyResult != nil {
			return w.visitRelationship(ctx, entry, *entry.Loop.BodyResult)
		}

	case declaration.TypeWorkflow:
		if entry.Workflow == nil {
			return nil
		}
		for _, node := range entry.Workflow.Nodes {
			if node.MemberResult == nil {
				continue
			}
			if err := w.visitRelationship(ctx, entry, *node.MemberResult); err != nil {
				return err
			}
		}

	case declaration.TypeWorkspace:
		if entry.Workspace == nil {
			return nil
		}
		for _, relationship := range entry.Workspace.MemberResults {
			if err := w.visitRelationship(ctx, entry, relationship); err != nil {
				return err
			}
		}
	default:
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
