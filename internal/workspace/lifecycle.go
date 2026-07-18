package workspace

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

func (s *Service) UpdateWorkspace(
	ctx context.Context,
	request UpdateWorkspaceRequest,
) (Workspace, error) {
	current, err := s.GetWorkspace(ctx, request.RootID)
	if err != nil {
		return Workspace{}, err
	}
	root := current.Root
	data := current.Data

	if request.DisplayName != nil {
		root.DisplayName = *request.DisplayName
	}
	if request.Description != nil {
		root.Description = *request.Description
	}
	if request.Enabled != nil {
		root.Enabled = *request.Enabled
	}
	if request.TrustReference != nil {
		data.RootTrustReference = *request.TrustReference
	}
	if request.Discovery != nil {
		data.DiscoveryPreferences = *request.Discovery
	}
	if request.AttachedPackages != nil {
		data.AttachedPackagePreferences = *request.AttachedPackages
	}
	if request.DisplayPreferences != nil {
		data.DisplayPreferences = *request.DisplayPreferences
	}

	raw, err := encodeRootData(data)
	if err != nil {
		return Workspace{}, fmt.Errorf("%w: %w", ErrInvalidWorkspace, err)
	}
	updated, err := s.store.UpdateRoot(ctx, request.RootID, artifactstore.RootUpdate{
		ExpectedModifiedAt: request.ExpectedModifiedAt,
		DisplayName:        root.DisplayName,
		Description:        root.Description,
		Enabled:            root.Enabled,
		DataSchemaID:       RootDataSchemaID,
		Data:               raw,
	})
	if err != nil {
		return Workspace{}, err
	}
	workspace, err := s.GetWorkspace(ctx, updated.RootID)
	if err != nil {
		return Workspace{}, err
	}
	if !request.DiscoverImmediately || !updated.Enabled {
		return workspace, nil
	}
	if _, err := s.Refresh(ctx, updated.RootID); err != nil {
		return workspace, err
	}
	return s.GetWorkspace(ctx, updated.RootID)
}

func (s *Service) DeleteWorkspace(
	ctx context.Context,
	request DeleteWorkspaceRequest,
) (artifactstoreSpec.ArtifactRoot, error) {
	if _, err := s.GetWorkspace(ctx, request.RootID); err != nil {
		return artifactstoreSpec.ArtifactRoot{}, err
	}
	return s.store.DeleteRoot(
		ctx,
		request.RootID,
		request.ExpectedModifiedAt,
	)
}
