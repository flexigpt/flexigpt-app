package provision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type sourceManager interface {
	CreateWithStatus(
		ctx context.Context,
		rootID basespec.RootID,
		draft source.Draft,
	) (source.Summary, bool, error)

	Discard(
		ctx context.Context,
		rootID basespec.RootID,
		id basespec.SourceID,
		expectedRevision uint64,
	) error
}

type workspaceManager interface {
	ValidateFilesystemCreate(
		request spec.FilesystemWorkspaceRequest,
	) error

	CreateFilesystem(
		ctx context.Context,
		request spec.FilesystemWorkspaceRequest,
	) (spec.Workspace, error)
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
			spec.ErrInvalidWorkspace,
		)
	}
	return &Service{
		sources:    sources,
		workspaces: workspaces,
	}, nil
}

type Request struct {
	RootID       basespec.RootID
	CollectionID basespec.CollectionID
	SourceID     basespec.SourceID
	DisplayName  string
	Description  string
	RootPath     string
	Discovery    spec.DiscoveryPreferences
}

func (s *Service) CreateFilesystem(
	ctx context.Context,
	request Request,
) (spec.Workspace, error) {
	workspaceRequest := spec.FilesystemWorkspaceRequest{
		CollectionID:    request.CollectionID,
		RootID:          request.RootID,
		DisplayName:     request.DisplayName,
		Description:     request.Description,
		PrimarySourceID: request.SourceID,
		Discovery:       request.Discovery,
	}
	if err := s.workspaces.ValidateFilesystemCreate(workspaceRequest); err != nil {
		return spec.Workspace{}, err
	}

	config, err := json.Marshal(fsdir.Config{
		RootPath: request.RootPath,
	})
	if err != nil {
		return spec.Workspace{}, err
	}
	sourceValue, sourceCreated, err := s.sources.CreateWithStatus(
		ctx,
		request.RootID,
		source.Draft{
			ID:          request.SourceID,
			Kind:        fsdir.Kind,
			DisplayName: request.DisplayName,
			Enabled:     true,
			Config:      config,
		},
	)
	if err != nil {
		return spec.Workspace{}, err
	}

	workspaceRequest.PrimarySourceID = sourceValue.ID
	value, createErr := s.workspaces.CreateFilesystem(ctx, workspaceRequest)
	if createErr == nil {
		return value, nil
	}

	if !sourceCreated {
		return spec.Workspace{}, createErr
	}

	discardErr := s.sources.Discard(
		context.WithoutCancel(ctx),
		request.RootID,
		sourceValue.ID,
		sourceValue.Revision,
	)
	return spec.Workspace{}, errors.Join(createErr, discardErr)
}
