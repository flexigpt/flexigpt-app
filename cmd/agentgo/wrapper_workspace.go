package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/workspace"
)

const workspaceBuiltInSkillsProviderKey = "flexigpt-builtin-skills"

type WorkspaceWrapper struct {
	artifactStore *artifactstore.Store
	service       *workspace.Service
}

func InitWorkspaceWrapper(wrapper *WorkspaceWrapper, baseDir string) error {
	if wrapper == nil {
		return errors.New("initializing WorkspaceWrapper on nil receiver")
	}

	store, err := artifactstore.NewStore(
		baseDir,
		artifactstore.WithEmbeddedFSProvider(
			workspaceBuiltInSkillsProviderKey,
			builtin.BuiltInSkillBundlesFS,
		),
	)
	if err != nil {
		return err
	}

	service, err := workspace.NewService(store)
	if err != nil {
		_ = store.Close()
		return err
	}

	wrapper.artifactStore = store
	wrapper.service = service
	return nil
}

func (w *WorkspaceWrapper) SelectFilesystemRoot(
	request workspace.FilesystemSelectionRequest,
) (workspace.Workspace, error) {
	return middleware.WithRecoveryResp(func() (workspace.Workspace, error) {
		return w.service.SelectFilesystemRoot(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) CreateEmptyWorkspace(
	request workspace.EmptyWorkspaceRequest,
) (workspace.Workspace, error) {
	return middleware.WithRecoveryResp(func() (workspace.Workspace, error) {
		return w.service.CreateEmptyWorkspace(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) GetWorkspace(
	rootID artifactstoreSpec.RootID,
) (workspace.Workspace, error) {
	return middleware.WithRecoveryResp(func() (workspace.Workspace, error) {
		return w.service.GetWorkspace(context.Background(), rootID)
	})
}

func (w *WorkspaceWrapper) ListWorkspaces() ([]workspace.Workspace, error) {
	return middleware.WithRecoveryResp(func() ([]workspace.Workspace, error) {
		return w.service.ListWorkspaces(context.Background())
	})
}

func (w *WorkspaceWrapper) Refresh(
	rootID artifactstoreSpec.RootID,
) (workspace.RefreshResult, error) {
	return middleware.WithRecoveryResp(func() (workspace.RefreshResult, error) {
		return w.service.Refresh(context.Background(), rootID)
	})
}

func (w *WorkspaceWrapper) Catalog(
	rootID artifactstoreSpec.RootID,
) (workspace.Catalog, error) {
	return middleware.WithRecoveryResp(func() (workspace.Catalog, error) {
		return w.service.Catalog(context.Background(), rootID)
	})
}

func (w *WorkspaceWrapper) close() {
	if w == nil || w.artifactStore == nil {
		return
	}
	if err := w.artifactStore.Close(); err != nil {
		slog.Error("failed to close Workspace Artifact Store", "error", err)
	}
	w.service = nil
	w.artifactStore = nil
}
