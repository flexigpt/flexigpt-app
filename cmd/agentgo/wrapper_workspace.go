package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/materializer"
	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/workspace"
)

const (
	workspaceBuiltInSkillsProviderKey           = "flexigpt-builtin-skills"
	workspaceBuiltInSkillsDisplayName           = "Built-in Skills"
	workspaceBuiltInToolsProviderKey            = "flexigpt-builtin-tools"
	workspaceBuiltInToolsDisplayName            = "Built-in Tools"
	workspaceBuiltInMCPProviderKey              = "flexigpt-builtin-mcp"
	workspaceBuiltInMCPDisplayName              = "Built-in MCP Servers"
	workspaceBuiltInAssistantPresetsProviderKey = "flexigpt-builtin-assistant-presets"
	workspaceBuiltInAssistantPresetsDisplayName = "Built-in Assistant Presets"
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
		artifactstore.WithDefinitionMaterializer(
			materializer.NewFSDefinitionMaterializer(),
		),
		artifactstore.WithEmbeddedFSProvider(
			workspaceBuiltInSkillsProviderKey,
			builtin.BuiltInSkillBundlesFS,
		),
		artifactstore.WithEmbeddedFSProvider(
			workspaceBuiltInToolsProviderKey,
			builtin.BuiltInToolBundlesFS,
		),
		artifactstore.WithEmbeddedFSProvider(
			workspaceBuiltInMCPProviderKey,
			builtin.BuiltInMCPBundlesFS,
		),
		artifactstore.WithEmbeddedFSProvider(
			workspaceBuiltInAssistantPresetsProviderKey,
			builtin.BuiltInAssistantPresetBundlesFS,
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

func (w *WorkspaceWrapper) UpdateWorkspace(
	request workspace.UpdateWorkspaceRequest,
) (workspace.Workspace, error) {
	return middleware.WithRecoveryResp(func() (workspace.Workspace, error) {
		return w.service.UpdateWorkspace(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) DeleteWorkspace(
	request workspace.DeleteWorkspaceRequest,
) (artifactstoreSpec.ArtifactRoot, error) {
	return middleware.WithRecoveryResp(func() (artifactstoreSpec.ArtifactRoot, error) {
		return w.service.DeleteWorkspace(context.Background(), request)
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

func (w *WorkspaceWrapper) MountBuiltInSkills(
	rootID artifactstoreSpec.RootID,
	priority int,
	discoverImmediately bool,
) (workspace.Workspace, error) {
	return w.mountBuiltInSource(
		rootID,
		priority,
		discoverImmediately,
		workspaceBuiltInSkillsProviderKey,
		workspaceBuiltInSkillsDisplayName,
		builtin.BuiltInSkillBundlesRootDir,
	)
}

func (w *WorkspaceWrapper) MountBuiltInTools(
	rootID artifactstoreSpec.RootID,
	priority int,
	discoverImmediately bool,
) (workspace.Workspace, error) {
	return w.mountBuiltInSource(
		rootID,
		priority,
		discoverImmediately,
		workspaceBuiltInToolsProviderKey,
		workspaceBuiltInToolsDisplayName,
		builtin.BuiltInToolBundlesRootDir,
	)
}

func (w *WorkspaceWrapper) MountBuiltInMCPServers(
	rootID artifactstoreSpec.RootID,
	priority int,
	discoverImmediately bool,
) (workspace.Workspace, error) {
	return w.mountBuiltInSource(
		rootID,
		priority,
		discoverImmediately,
		workspaceBuiltInMCPProviderKey,
		workspaceBuiltInMCPDisplayName,
		builtin.BuiltInMCPBundlesRootDir,
	)
}

func (w *WorkspaceWrapper) MountBuiltInAssistantPresets(
	rootID artifactstoreSpec.RootID,
	priority int,
	discoverImmediately bool,
) (workspace.Workspace, error) {
	return w.mountBuiltInSource(
		rootID,
		priority,
		discoverImmediately,
		workspaceBuiltInAssistantPresetsProviderKey,
		workspaceBuiltInAssistantPresetsDisplayName,
		builtin.BuiltInAssistantPresetBundlesRootDir,
	)
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

func (w *WorkspaceWrapper) ExportWorkspaceRecord(
	recordID artifactstoreSpec.RecordID,
) (artifactstoreSpec.ExportedRecord, error) {
	return middleware.WithRecoveryResp(func() (artifactstoreSpec.ExportedRecord, error) {
		return w.service.ExportRecord(context.Background(), recordID)
	})
}

func (w *WorkspaceWrapper) ImportWorkspaceDefinition(
	request artifactstoreSpec.ImportDefinitionRequest,
	discoverImmediately bool,
) (workspace.TransferResult, error) {
	return middleware.WithRecoveryResp(func() (workspace.TransferResult, error) {
		return w.service.ImportDefinition(context.Background(), request, discoverImmediately)
	})
}

func (w *WorkspaceWrapper) CaptureWorkspaceRecord(
	request artifactstoreSpec.CaptureRecordRequest,
	discoverImmediately bool,
) (workspace.TransferResult, error) {
	return middleware.WithRecoveryResp(func() (workspace.TransferResult, error) {
		return w.service.CaptureRecord(context.Background(), request, discoverImmediately)
	})
}

func (w *WorkspaceWrapper) ForkWorkspaceRecord(
	request artifactstoreSpec.ForkRecordRequest,
	discoverImmediately bool,
) (workspace.TransferResult, error) {
	return middleware.WithRecoveryResp(func() (workspace.TransferResult, error) {
		return w.service.ForkRecord(context.Background(), request, discoverImmediately)
	})
}

func (w *WorkspaceWrapper) mountBuiltInSource(
	rootID artifactstoreSpec.RootID,
	priority int,
	discoverImmediately bool,
	providerKey string,
	displayName string,
	rootLocator string,
) (workspace.Workspace, error) {
	return middleware.WithRecoveryResp(func() (workspace.Workspace, error) {
		recursive := true
		return w.service.MountEmbeddedSource(
			context.Background(),
			workspace.EmbeddedSourceAttachmentRequest{
				RootID:              rootID,
				DisplayName:         displayName,
				ProviderKey:         providerKey,
				RootLocator:         artifactstoreSpec.SourceLocator(rootLocator),
				Role:                workspace.RoleBuiltIn,
				Priority:            priority,
				AttachmentData:      workspace.AttachmentData{Recursive: &recursive},
				DiscoverImmediately: discoverImmediately,
			},
		)
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
