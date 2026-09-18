package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

func (a *StoreAPI) LoadWorkspace(
	ctx context.Context,
	ref WorkspaceRef,
) (WorkspaceLoad, error) {
	workspace, resolved, err := a.resolveCurrentWorkspace(ctx, ref)
	if err != nil {
		return WorkspaceLoad{}, err
	}

	members := make([]declaration.Entry, len(workspace.Document.Members))
	for index, value := range workspace.Document.Members {
		members[index] = value.Clone()
	}

	return WorkspaceLoad{
		Workspace: workspace,
		Members:   members,
		resolved:  resolved.Workspace,
	}, nil
}

func (a *StoreAPI) RefreshWorkspace(
	ctx context.Context,
	ref WorkspaceRef,
) (WorkspaceRefresh, error) {
	if a == nil || a.resolver == nil {
		return WorkspaceRefresh{}, basespec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return WorkspaceRefresh{}, err
	}
	if err := a.resolver.RefreshWorkspace(ctx, ref); err != nil {
		return WorkspaceRefresh{}, err
	}
	workspace, err := a.GetWorkspace(ctx, ref)
	if err != nil {
		return WorkspaceRefresh{}, err
	}
	return WorkspaceRefresh{
		Workspace: workspace.Ref(),
	}, nil
}

func (a *StoreAPI) resolveCurrentWorkspace(
	ctx context.Context,
	ref WorkspaceRef,
) (
	workspaceDomain.Workspace,
	*resolve.ResolvedEntry,
	error,
) {
	if a == nil || a.resolver == nil {
		return workspaceDomain.Workspace{},
			nil,
			basespec.ErrClosed
	}
	workspace, err := a.GetWorkspace(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{}, nil, err
	}
	resolved, err := a.resolver.ResolveWorkspaceEntry(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{}, nil, err
	}
	if resolved == nil ||
		resolved.Type != declaration.TypeWorkspace ||
		resolved.Workspace == nil {
		return workspaceDomain.Workspace{}, nil, fmt.Errorf(
			"%w: Artifact %q did not resolve as a Workspace",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	return workspace, resolved, nil
}

func (a *StoreAPI) workspaceForRef(
	ctx context.Context,
	ref WorkspaceRef,
) (workspaceDomain.Workspace, error) {
	return a.workspaceAt(ctx, ref)
}

func (a *StoreAPI) workspaceAt(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (workspaceDomain.Workspace, error) {
	if a.resolver != nil {
		terminal, err := a.resolver.ResolveTerminalArtifact(ctx, ref)
		if err != nil {
			return workspaceDomain.Workspace{}, err
		}
		ref = terminal
	}

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
