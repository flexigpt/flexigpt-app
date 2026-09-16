package resolve

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
)

// ResolveDeclarationArtifact follows only located composite declaration
// edges and returns the terminal source-backed ArtifactRef. It deliberately
// does not expand Collection members, Workspace roots, Agent members, or
// other graph structure.
func (r *Resolver) ResolveDeclarationArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.ArtifactRef, error) {
	if r == nil || r.artifacts == nil {
		return artifact.ArtifactRef{}, basespec.ErrClosed
	}
	if err := validateResolutionContext(ctx); err != nil {
		return artifact.ArtifactRef{}, err
	}
	if err := ref.Validate(); err != nil {
		return artifact.ArtifactRef{}, err
	}

	current := ref
	expectedType := declaration.Type("")
	expectedName := basespec.LogicalName("")
	seen := make(map[artifact.ArtifactRef]struct{})

	for depth := 0; depth <= r.limits.MaxDepth; depth++ {
		if _, duplicate := seen[current]; duplicate {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: located declaration cycle at Artifact %q",
				basespec.ErrReferenceUnresolved,
				current.ArtifactID,
			)
		}
		if len(seen) >= r.limits.MaxNodes {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: located declaration resolution exceeds %d Artifacts",
				basespec.ErrLocatorLimitExceeded,
				r.limits.MaxNodes,
			)
		}
		seen[current] = struct{}{}

		loaded, err := r.loadAvailableDeclarationArtifact(
			ctx,
			current,
			expectedType,
		)
		if err != nil {
			return artifact.ArtifactRef{}, err
		}
		record := loaded.record
		if expectedName != "" && record.LogicalName != expectedName {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: located declaration %s/%s resolved to another logical name",
				basespec.ErrReferenceUnresolved,
				expectedType,
				expectedName,
			)
		}

		entry := loaded.entry
		header := entry.Header()
		if !shouldResolveDeclarationLocator(entry, header.Locator) {
			return record.Ref(), nil
		}
		if r.locators == nil {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: declaration locator requires a locator resolver",
				basespec.ErrLocatorUnresolved,
			)
		}

		next, err := r.locators.ResolveArtifactLocator(
			ctx,
			LocatorRequest{
				RootID:              record.RootID,
				From:                pointerArtifact(record),
				Entry:               entry.Clone(),
				Locator:             header.Locator.Clone(),
				ExpectedType:        header.Type,
				ExpectedLogicalName: basespec.LogicalName(header.Name),
			},
		)
		if err != nil {
			return artifact.ArtifactRef{}, err
		}
		if err := next.Validate(); err != nil {
			return artifact.ArtifactRef{}, err
		}
		if next.RootID != record.RootID {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: locator resolver returned an Artifact from another Root",
				basespec.ErrInvalid,
			)
		}

		expectedType = header.Type
		expectedName = basespec.LogicalName(header.Name)
		current = next
	}

	return artifact.ArtifactRef{}, fmt.Errorf(
		"%w: located declaration chain exceeds depth %d",
		basespec.ErrLocatorLimitExceeded,
		r.limits.MaxDepth,
	)
}
