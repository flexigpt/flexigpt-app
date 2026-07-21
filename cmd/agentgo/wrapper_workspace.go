package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/workspace"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
	"github.com/flexigpt/flexigpt-app/internal/workspace/provision"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

type WorkspaceWrapper struct {
	artifacts   *system.Components
	workspace   *workspace.Components
	provisioner *provision.Service
}

type WorkspaceDeleteRequest struct {
	RootID           artifactstore.RootID `json:"rootID"`
	ExpectedRevision uint64               `json:"expectedRevision"`
}

type WorkspaceRefreshRequest struct {
	RootID artifactstore.RootID `json:"rootID"`
}

type WorkspaceContextRequest struct {
	RootID    artifactstore.RootID     `json:"rootID"`
	RecordIDs []artifactstore.RecordID `json:"recordIDs,omitempty"`
}

type WorkspaceSkillLoadRequest struct {
	RootID    artifactstore.RootID     `json:"rootID"`
	RecordIDs []artifactstore.RecordID `json:"recordIDs"`
}

type WorkspaceRecordEnabledRequest struct {
	RecordID         artifactstore.RecordID `json:"recordID"`
	ExpectedRevision uint64                 `json:"expectedRevision"`
	Enabled          bool                   `json:"enabled"`
}

func InitWorkspaceWrapper(
	api *WorkspaceWrapper,
	baseDirectory string,
) error {
	config := workspace.DefaultConfig()
	artifacts, err := system.Open(
		context.Background(),
		system.Config{
			BaseDirectory: baseDirectory,
			Decoders:      workspace.BuiltinDecoders(),
		},
	)
	if err != nil {
		return err
	}

	components, err := workspace.NewComponents(artifacts, config)
	if err != nil {
		_ = artifacts.Close()
		return err
	}
	provisioner, err := provision.NewService(
		artifacts.Sources,
		components.Service,
	)
	if err != nil {
		_ = artifacts.Close()
		return err
	}

	api.artifacts = artifacts
	api.workspace = components
	api.provisioner = provisioner
	return nil
}

func (w *WorkspaceWrapper) CreateFilesystem(
	request *provision.Request,
) (*engine.Workspace, error) {
	return middleware.WithRecoveryResp(func() (*engine.Workspace, error) {
		value, err := w.provisioner.CreateFilesystem(
			context.Background(),
			*request,
		)
		return &value, err
	})
}

func (w *WorkspaceWrapper) CreateEmpty(
	request *engine.EmptyWorkspaceRequest,
) (*engine.Workspace, error) {
	return middleware.WithRecoveryResp(func() (*engine.Workspace, error) {
		value, err := w.workspace.Service.CreateEmpty(
			context.Background(),
			*request,
		)
		return &value, err
	})
}

func (w *WorkspaceWrapper) Get(
	rootID artifactstore.RootID,
) (*engine.Workspace, error) {
	return middleware.WithRecoveryResp(func() (*engine.Workspace, error) {
		value, err := w.workspace.Service.Get(context.Background(), rootID)
		return &value, err
	})
}

func (w *WorkspaceWrapper) List() ([]engine.Workspace, error) {
	return middleware.WithRecoveryResp(func() ([]engine.Workspace, error) {
		return w.workspace.Service.List(context.Background())
	})
}

func (w *WorkspaceWrapper) Update(
	request *engine.UpdateRequest,
) (*engine.Workspace, error) {
	return middleware.WithRecoveryResp(func() (*engine.Workspace, error) {
		value, err := w.workspace.Service.Update(
			context.Background(),
			*request,
		)
		return &value, err
	})
}

func (w *WorkspaceWrapper) Delete(
	request *WorkspaceDeleteRequest,
) (*catalog.Root, error) {
	return middleware.WithRecoveryResp(func() (*catalog.Root, error) {
		value, err := w.workspace.Service.Delete(
			context.Background(),
			request.RootID,
			request.ExpectedRevision,
		)
		return &value, err
	})
}

func (w *WorkspaceWrapper) Refresh(
	request *WorkspaceRefreshRequest,
) (*refresh.Result, error) {
	return middleware.WithRecoveryResp(func() (*refresh.Result, error) {
		value, err := w.workspace.Refresher.Refresh(
			context.Background(),
			request.RootID,
		)
		return &value, err
	})
}

func (w *WorkspaceWrapper) Catalog(
	rootID artifactstore.RootID,
) (*engine.CatalogView, error) {
	return middleware.WithRecoveryResp(func() (*engine.CatalogView, error) {
		value, err := w.workspace.Query.Catalog(
			context.Background(),
			rootID,
		)
		return &value, err
	})
}

func (w *WorkspaceWrapper) ComposeContext(
	request *WorkspaceContextRequest,
) (*contextadapter.ContextLoadPlan, error) {
	return middleware.WithRecoveryResp(func() (*contextadapter.ContextLoadPlan, error) {
		value, err := w.workspace.Context.Compose(
			context.Background(),
			request.RootID,
			request.RecordIDs,
		)
		return &value, err
	})
}

func (w *WorkspaceWrapper) ListWorkspaceSkills(
	rootID artifactstore.RootID,
) ([]skilladapter.WorkspaceSkill, error) {
	return middleware.WithRecoveryResp(func() ([]skilladapter.WorkspaceSkill, error) {
		return w.workspace.Skills.List(context.Background(), rootID)
	})
}

func (w *WorkspaceWrapper) LoadWorkspaceSkills(
	request *WorkspaceSkillLoadRequest,
) (*skilladapter.SkillLoadPlan, error) {
	return middleware.WithRecoveryResp(func() (*skilladapter.SkillLoadPlan, error) {
		value, err := w.workspace.Skills.Load(
			context.Background(),
			request.RootID,
			request.RecordIDs,
		)
		return &value, err
	})
}

func (w *WorkspaceWrapper) SetRecordEnabled(
	request *WorkspaceRecordEnabledRequest,
) (*record.Record, error) {
	return middleware.WithRecoveryResp(func() (*record.Record, error) {
		value, err := w.artifacts.Records.SetEnabled(
			context.Background(),
			request.RecordID,
			request.ExpectedRevision,
			request.Enabled,
		)
		return &value, err
	})
}

func (w *WorkspaceWrapper) close() {
	if w == nil || w.artifacts == nil {
		return
	}
	_ = w.artifacts.Close()
}
