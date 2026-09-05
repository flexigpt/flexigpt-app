package provision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	artifactAPI "github.com/flexigpt/flexigpt-app/internal/artifactstore/api"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type filesystemSourceConfig struct {
	RootPath string `json:"rootPath"`
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
	store      artifactAPI.ConsumerAPI
	workspaces workspaceManager
}

func NewService(
	workspaces workspaceManager,
	store artifactAPI.ConsumerAPI,
) (*Service, error) {
	if store == nil || workspaces == nil {
		return nil, fmt.Errorf(
			"%w: Workspace provisioner dependencies are incomplete",
			spec.ErrInvalidWorkspace,
		)
	}
	return &Service{
		store:      store,
		workspaces: workspaces,
	}, nil
}

type Request struct {
	RootID           basespec.RootID
	CollectionID     basespec.CollectionID
	SourceID         basespec.SourceID
	SourceStorageKey basespec.StorageKey
	DisplayName      string
	Description      string
	RootPath         string
	Discovery        spec.DiscoveryPreferences
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

	config, err := json.Marshal(filesystemSourceConfig{
		RootPath: request.RootPath,
	})
	if err != nil {
		return spec.Workspace{}, err
	}

	sourceValue, sourceCreated, err := s.store.CreateSourceWithStatus(
		ctx,
		request.RootID,
		source.Draft{
			ID:          request.SourceID,
			StorageKey:  request.SourceStorageKey,
			Kind:        basespec.SourceKindFilesystemDirectory,
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

	discardErr := s.store.DiscardSource(
		context.WithoutCancel(ctx),
		request.RootID,
		sourceValue.ID,
		sourceValue.Revision,
	)
	return spec.Workspace{}, errors.Join(createErr, discardErr)
}
