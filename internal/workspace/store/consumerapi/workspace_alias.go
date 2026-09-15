package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

func (a *StoreAPI) workspaceForRef(
	ctx context.Context,
	ref WorkspaceRef,
) (workspaceDomain.Workspace, error) {
	if a == nil || a.artifacts == nil {
		return workspaceDomain.Workspace{}, basespec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return workspaceDomain.Workspace{}, err
	}

	candidate, err := a.workspaceAt(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}
	if candidate.Document.Locator == nil {
		return candidate, nil
	}
	if a.resolver == nil {
		return workspaceDomain.Workspace{}, basespec.ErrClosed
	}

	terminal, err := a.resolver.ResolveDeclarationArtifact(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}
	if terminal == ref {
		return workspaceDomain.Workspace{}, fmt.Errorf(
			"%w: Workspace locator did not resolve to another Artifact",
			basespec.ErrReferenceUnresolved,
		)
	}
	return a.workspaceAt(ctx, terminal)
}

func (a *StoreAPI) workspaceAt(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (workspaceDomain.Workspace, error) {
	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}
	if record.Kind != workspaceDomain.WorkspaceArtifactKind {
		return workspaceDomain.Workspace{}, fmt.Errorf(
			"%w: Artifact %q has kind %q",
			workspaceDomain.ErrNotWorkspace,
			record.ID,
			record.Kind,
		)
	}

	value, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}
	return workspaceDomain.NewWorkspace(record, value)
}
