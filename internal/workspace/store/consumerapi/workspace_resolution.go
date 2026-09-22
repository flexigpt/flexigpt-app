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

func (a *StoreAPI) resolveCurrentWorkspace(
	ctx context.Context,
	ref artifact.ArtifactRef,
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
	workspace, err := a.ResolveWorkspace(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{}, nil, err
	}
	resolved, err := a.resolver.ResolveWorkspaceWithCompositionSource(
		ctx,
		workspace.Ref(),
		workspace.CompositionSourceID,
	)
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
	ref artifact.ArtifactRef,
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
	value, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}

	workspace, err := workspaceDomain.NewWorkspace(record, value)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}
	workspace.CompositionSourceID = record.Binding.SourceID

	sources, err := a.loadWorkspaceSources(ctx, record.RootID)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}
	if sources.HasDirectory &&
		(record.Binding.SourceID == sources.Directory.ID ||
			(sources.HasPolicy &&
				record.Binding.SourceID == sources.Policy.ID)) {
		workspace.CompositionSourceID = sources.Directory.ID
	}
	if err := workspace.CompositionSourceID.Validate(); err != nil {
		return workspaceDomain.Workspace{}, err
	}
	return workspace, nil
}
