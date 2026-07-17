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

const (
	workspaceBuiltInSkillsProviderKey = "flexigpt-builtin-skills"
	workspaceBuiltInSkillsDisplayName = "Built-in Skills"
)

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

func (w *WorkspaceWrapper) AttachSource(
	request workspace.AttachSourceRequest,
) (workspace.Workspace, error) {
	return middleware.WithRecoveryResp(func() (workspace.Workspace, error) {
		return w.service.AttachSource(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) MountEmbeddedSource(
	request workspace.EmbeddedSourceAttachmentRequest,
) (workspace.Workspace, error) {
	return middleware.WithRecoveryResp(func() (workspace.Workspace, error) {
		return w.service.MountEmbeddedSource(context.Background(), request)
	})
}

// MountBuiltInSkills attaches FlexiGPT's registered embedded skill packages to
// a Workspace without requiring a frontend client to know the provider key.
func (w *WorkspaceWrapper) MountBuiltInSkills(
	rootID artifactstoreSpec.RootID,
	priority int,
	discoverImmediately bool,
) (workspace.Workspace, error) {
	return middleware.WithRecoveryResp(func() (workspace.Workspace, error) {
		recursive := true
		return w.service.MountEmbeddedSource(
			context.Background(),
			workspace.EmbeddedSourceAttachmentRequest{
				RootID:              rootID,
				DisplayName:         workspaceBuiltInSkillsDisplayName,
				ProviderKey:         workspaceBuiltInSkillsProviderKey,
				RootLocator:         artifactstoreSpec.SourceLocator(builtin.BuiltInSkillBundlesRootDir),
				Role:                workspace.RoleBuiltIn,
				Priority:            priority,
				AttachmentData:      workspace.AttachmentData{Recursive: &recursive},
				DiscoverImmediately: discoverImmediately,
			},
		)
	})
}

func (w *WorkspaceWrapper) DetachSource(
	rootID artifactstoreSpec.RootID,
	sourceID artifactstoreSpec.SourceID,
	discoverImmediately bool,
) (workspace.Workspace, error) {
	return middleware.WithRecoveryResp(func() (workspace.Workspace, error) {
		return w.service.DetachSource(
			context.Background(),
			rootID,
			sourceID,
			discoverImmediately,
		)
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

func (w *WorkspaceWrapper) Project(
	recordID artifactstoreSpec.RecordID,
) (workspace.Projection, error) {
	return middleware.WithRecoveryResp(func() (workspace.Projection, error) {
		return w.service.Project(context.Background(), recordID)
	})
}

func (w *WorkspaceWrapper) ResolveReference(
	rootID artifactstoreSpec.RootID,
	reference workspace.Reference,
) (workspace.CatalogResource, error) {
	return middleware.WithRecoveryResp(func() (workspace.CatalogResource, error) {
		return w.service.ResolveReference(context.Background(), rootID, reference)
	})
}

func (w *WorkspaceWrapper) ComposeLoadPlan(
	rootID artifactstoreSpec.RootID,
	recordIDs []artifactstoreSpec.RecordID,
) (workspace.LoadPlan, error) {
	return middleware.WithRecoveryResp(func() (workspace.LoadPlan, error) {
		return w.service.ComposeLoadPlan(context.Background(), rootID, recordIDs)
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
