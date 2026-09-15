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

		record, err := r.artifacts.Get(ctx, current)
		if err != nil {
			return artifact.ArtifactRef{}, err
		}
		if record.Ref() != current {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: Artifact reader returned another Artifact",
				basespec.ErrInvalid,
			)
		}
		if record.State != artifact.StateAvailable {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: Artifact %q is not available",
				basespec.ErrReferenceUnresolved,
				record.ID,
			)
		}
		if !r.options.IncludeDisabled && !record.Enabled {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: Artifact %q is disabled",
				basespec.ErrReferenceUnresolved,
				record.ID,
			)
		}

		declarationType := declaration.Type(record.Kind)
		if err := declarationType.Validate(); err != nil {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: Artifact %q has unsupported declaration type: %w",
				basespec.ErrReferenceUnresolved,
				record.ID,
				err,
			)
		}
		if expectedType != "" && declarationType != expectedType {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: Artifact %q has type %q, expected %q",
				basespec.ErrReferenceUnresolved,
				record.ID,
				declarationType,
				expectedType,
			)
		}
		if expectedName != "" && record.LogicalName != expectedName {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: located declaration %s/%s resolved to another logical name",
				basespec.ErrReferenceUnresolved,
				expectedType,
				expectedName,
			)
		}

		definitionValue, err := r.artifacts.GetDefinition(ctx, current)
		if err != nil {
			return artifact.ArtifactRef{}, err
		}
		if err := validateDefinitionContract(
			definitionValue,
			declarationType,
		); err != nil {
			return artifact.ArtifactRef{}, err
		}
		if definitionValue.Kind != record.Kind ||
			definitionValue.LogicalName != record.LogicalName ||
			definitionValue.LogicalVersion != record.LogicalVersion {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: Artifact Definition identity does not match Artifact state",
				basespec.ErrDigestMismatch,
			)
		}

		entry, err := declaration.DecodeCanonicalEntryJSON(
			definitionValue.Body,
		)
		if err != nil {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: Artifact Definition body is not a canonical declaration: %w",
				basespec.ErrReferenceUnresolved,
				err,
			)
		}
		header := entry.Header()
		if header.Type != declarationType ||
			header.Name != string(definitionValue.LogicalName) {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: Artifact Definition declaration identity differs from Artifact state",
				basespec.ErrDigestMismatch,
			)
		}
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
