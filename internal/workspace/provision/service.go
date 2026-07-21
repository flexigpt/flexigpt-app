package provision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/workspace"
)

type sourceManager interface {
	Create(
		ctx context.Context,
		draft source.Draft,
	) (source.Source, error)

	Delete(
		ctx context.Context,
		id artifactstore.SourceID,
		expectedRevision uint64,
	) error
}

type workspaceManager interface {
	CreateFilesystem(
		ctx context.Context,
		request workspace.FilesystemWorkspaceRequest,
	) (workspace.Workspace, error)
}

type Service struct {
	sources    sourceManager
	workspaces workspaceManager
}

func NewService(
	sources sourceManager,
	workspaces workspaceManager,
) (*Service, error) {
	if sources == nil || workspaces == nil {
		return nil, fmt.Errorf(
			"%w: Workspace provisioner dependencies are incomplete",
			workspace.ErrInvalidWorkspace,
		)
	}
	return &Service{
		sources:    sources,
		workspaces: workspaces,
	}, nil
}

type Request struct {
	DisplayName    string
	Description    string
	RootPath       string
	TrustReference string
	Discovery      workspace.DiscoveryPreferences
}

func (s *Service) CreateFilesystem(
	ctx context.Context,
	request Request,
) (workspace.Workspace, error) {
	config, err := json.Marshal(fsdir.Config{
		RootPath: request.RootPath,
	})
	if err != nil {
		return workspace.Workspace{}, err
	}
	sourceValue, err := s.sources.Create(
		ctx,
		source.Draft{
			Kind:        fsdir.Kind,
			DisplayName: request.DisplayName,
			Enabled:     true,
			Config:      config,
		},
	)
	if err != nil {
		return workspace.Workspace{}, err
	}

	value, createErr := s.workspaces.CreateFilesystem(
		ctx,
		workspace.FilesystemWorkspaceRequest{
			DisplayName:     request.DisplayName,
			Description:     request.Description,
			PrimarySourceID: sourceValue.ID,
			TrustReference:  request.TrustReference,
			Discovery:       request.Discovery,
		},
	)
	if createErr == nil {
		return value, nil
	}

	deleteErr := s.sources.Delete(
		context.WithoutCancel(ctx),
		sourceValue.ID,
		sourceValue.Revision,
	)
	return workspace.Workspace{}, errors.Join(createErr, deleteErr)
}
