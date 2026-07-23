package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/skillruntime"
	"github.com/flexigpt/flexigpt-app/internal/workspace"
)

type WorkspaceWrapper struct {
	api          *workspace.API
	skillRuntime *skillruntime.SkillRuntime
}

func InitWorkspaceWrapper(
	wrapper *WorkspaceWrapper,
	baseDirectory string,
) error {
	api, err := workspace.Open(context.Background(), workspace.OpenConfig{
		BaseDirectory:   baseDirectory,
		WorkspaceConfig: workspace.DefaultConfig(),
	})
	if err != nil {
		return err
	}
	wrapper.api = api
	return nil
}

// BindWorkspaceSkillRuntime is application composition. Workspace does not
// import or know about skillruntime; the application wrapper decides whether
// Workspace changes should be reflected in a running Skill runtime.
func BindWorkspaceSkillRuntime(
	wrapper *WorkspaceWrapper,
	runtime *skillruntime.SkillRuntime,
) error {
	if wrapper == nil || wrapper.api == nil {
		return errors.New("workspace wrapper is not initialized")
	}
	if runtime == nil {
		return errors.New("skill runtime is nil")
	}
	wrapper.skillRuntime = runtime
	return nil
}

func (w *WorkspaceWrapper) CreateFilesystemWorkspace(
	request *workspace.CreateFilesystemWorkspaceRequest,
) (*workspace.CreateFilesystemWorkspaceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.CreateFilesystemWorkspaceResponse, error) {
		ctx := context.Background()
		response, err := w.api.CreateFilesystemWorkspace(ctx, request)
		if err != nil {
			return nil, err
		}
		if response == nil || response.Body == nil {
			return nil, errors.New("create filesystem Workspace returned an empty response")
		}
		if err := w.syncWorkspaceSkills(ctx, response.Body.RootID); err != nil {
			return nil, err
		}
		return response, nil
	})
}

func (w *WorkspaceWrapper) CreateEmptyWorkspace(
	request *workspace.CreateEmptyWorkspaceRequest,
) (*workspace.CreateEmptyWorkspaceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.CreateEmptyWorkspaceResponse, error) {
		ctx := context.Background()
		response, err := w.api.CreateEmptyWorkspace(ctx, request)
		if err != nil {
			return nil, err
		}
		if response == nil || response.Body == nil {
			return nil, errors.New("create empty Workspace returned an empty response")
		}
		if err := w.syncWorkspaceSkills(ctx, response.Body.RootID); err != nil {
			return nil, err
		}
		return response, nil
	})
}

func (w *WorkspaceWrapper) GetWorkspace(
	request *workspace.GetWorkspaceRequest,
) (*workspace.GetWorkspaceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.GetWorkspaceResponse, error) {
		return w.api.GetWorkspace(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) ListWorkspaces(
	request *workspace.ListWorkspacesRequest,
) (*workspace.ListWorkspacesResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.ListWorkspacesResponse, error) {
		return w.api.ListWorkspaces(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) UpdateWorkspace(
	request *workspace.UpdateWorkspaceRequest,
) (*workspace.UpdateWorkspaceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.UpdateWorkspaceResponse, error) {
		ctx := context.Background()
		response, err := w.api.UpdateWorkspace(ctx, request)
		if err != nil {
			return nil, err
		}
		if err := w.syncWorkspaceSkills(ctx, request.RootID); err != nil {
			return nil, err
		}
		return response, nil
	})
}

func (w *WorkspaceWrapper) DeleteWorkspace(
	request *workspace.DeleteWorkspaceRequest,
) (*workspace.DeleteWorkspaceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.DeleteWorkspaceResponse, error) {
		ctx := context.Background()
		response, err := w.api.DeleteWorkspace(ctx, request)
		if err != nil {
			return nil, err
		}
		if err := w.removeWorkspaceSkills(ctx, request.RootID); err != nil {
			return nil, err
		}
		return response, nil
	})
}

func (w *WorkspaceWrapper) AttachWorkspaceSource(
	request *workspace.AttachWorkspaceSourceRequest,
) (*workspace.AttachWorkspaceSourceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.AttachWorkspaceSourceResponse, error) {
		ctx := context.Background()
		response, err := w.api.AttachWorkspaceSource(ctx, request)
		if err != nil {
			return nil, err
		}
		if err := w.syncWorkspaceSkills(ctx, request.RootID); err != nil {
			return nil, err
		}
		return response, nil
	})
}

func (w *WorkspaceWrapper) UpdateWorkspaceAttachment(
	request *workspace.UpdateWorkspaceAttachmentRequest,
) (*workspace.UpdateWorkspaceAttachmentResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.UpdateWorkspaceAttachmentResponse, error) {
		ctx := context.Background()
		response, err := w.api.UpdateWorkspaceAttachment(ctx, request)
		if err != nil {
			return nil, err
		}
		if err := w.syncWorkspaceSkills(ctx, request.RootID); err != nil {
			return nil, err
		}
		return response, nil
	})
}

func (w *WorkspaceWrapper) DetachWorkspaceSource(
	request *workspace.DetachWorkspaceSourceRequest,
) (*workspace.DetachWorkspaceSourceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.DetachWorkspaceSourceResponse, error) {
		ctx := context.Background()
		response, err := w.api.DetachWorkspaceSource(ctx, request)
		if err != nil {
			return nil, err
		}
		if err := w.syncWorkspaceSkills(ctx, request.RootID); err != nil {
			return nil, err
		}
		return response, nil
	})
}

func (w *WorkspaceWrapper) RefreshWorkspace(
	request *workspace.RefreshWorkspaceRequest,
) (*workspace.RefreshWorkspaceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.RefreshWorkspaceResponse, error) {
		ctx := context.Background()
		response, err := w.api.RefreshWorkspace(ctx, request)
		if err != nil {
			return nil, err
		}
		if err := w.syncWorkspaceSkills(ctx, request.RootID); err != nil {
			return nil, err
		}
		return response, nil
	})
}

func (w *WorkspaceWrapper) GetWorkspaceCatalog(
	request *workspace.GetWorkspaceCatalogRequest,
) (*workspace.GetWorkspaceCatalogResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.GetWorkspaceCatalogResponse, error) {
		return w.api.GetWorkspaceCatalog(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) GetWorkspaceRecord(
	request *workspace.GetWorkspaceRecordRequest,
) (*workspace.GetWorkspaceRecordResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.GetWorkspaceRecordResponse, error) {
		return w.api.GetWorkspaceRecord(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) ListWorkspaceContexts(
	request *workspace.ListWorkspaceContextsRequest,
) (*workspace.ListWorkspaceContextsResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.ListWorkspaceContextsResponse, error) {
		return w.api.ListWorkspaceContexts(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) LoadWorkspaceContexts(
	request *workspace.LoadWorkspaceContextsRequest,
) (*workspace.LoadWorkspaceContextsResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.LoadWorkspaceContextsResponse, error) {
		return w.api.LoadWorkspaceContexts(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) ComposeWorkspaceContext(
	request *workspace.ComposeWorkspaceContextRequest,
) (*workspace.ComposeWorkspaceContextResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.ComposeWorkspaceContextResponse, error) {
		return w.api.ComposeWorkspaceContext(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) ListWorkspaceSkills(
	request *workspace.ListWorkspaceSkillsRequest,
) (*workspace.ListWorkspaceSkillsResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.ListWorkspaceSkillsResponse, error) {
		return w.api.ListWorkspaceSkills(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) LoadWorkspaceSkills(
	request *workspace.LoadWorkspaceSkillsRequest,
) (*workspace.LoadWorkspaceSkillsResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.LoadWorkspaceSkillsResponse, error) {
		return w.api.LoadWorkspaceSkills(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) SetWorkspaceRecordEnabled(
	request *workspace.SetWorkspaceRecordEnabledRequest,
) (*workspace.SetWorkspaceRecordEnabledResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.SetWorkspaceRecordEnabledResponse, error) {
		ctx := context.Background()
		response, err := w.api.SetWorkspaceRecordEnabled(ctx, request)
		if err != nil {
			return nil, err
		}
		if err := w.syncWorkspaceSkills(ctx, request.RootID); err != nil {
			return nil, err
		}
		return response, nil
	})
}

func (w *WorkspaceWrapper) PinWorkspaceRecord(
	request *workspace.PinWorkspaceRecordRequest,
) (*workspace.PinWorkspaceRecordResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.PinWorkspaceRecordResponse, error) {
		ctx := context.Background()
		response, err := w.api.PinWorkspaceRecord(ctx, request)
		if err != nil {
			return nil, err
		}
		if err := w.syncWorkspaceSkills(ctx, request.RootID); err != nil {
			return nil, err
		}
		return response, nil
	})
}

func (w *WorkspaceWrapper) FollowWorkspaceRecord(
	request *workspace.FollowWorkspaceRecordRequest,
) (*workspace.FollowWorkspaceRecordResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.FollowWorkspaceRecordResponse, error) {
		ctx := context.Background()
		response, err := w.api.FollowWorkspaceRecord(ctx, request)
		if err != nil {
			return nil, err
		}
		if err := w.syncWorkspaceSkills(ctx, request.RootID); err != nil {
			return nil, err
		}
		return response, nil
	})
}

func (w *WorkspaceWrapper) DeleteWorkspaceRecord(
	request *workspace.DeleteWorkspaceRecordRequest,
) (*workspace.DeleteWorkspaceRecordResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.DeleteWorkspaceRecordResponse, error) {
		ctx := context.Background()
		response, err := w.api.DeleteWorkspaceRecord(ctx, request)
		if err != nil {
			return nil, err
		}
		if err := w.syncWorkspaceSkills(ctx, request.RootID); err != nil {
			return nil, err
		}
		return response, nil
	})
}

func (w *WorkspaceWrapper) UpdateWorkspaceRecordData(
	request *workspace.UpdateWorkspaceRecordDataRequest,
) (*workspace.UpdateWorkspaceRecordDataResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.UpdateWorkspaceRecordDataResponse, error) {
		ctx := context.Background()
		response, err := w.api.UpdateWorkspaceRecordData(ctx, request)
		if err != nil {
			return nil, err
		}
		if err := w.syncWorkspaceSkills(ctx, request.RootID); err != nil {
			return nil, err
		}
		return response, nil
	})
}

func (w *WorkspaceWrapper) syncWorkspaceSkills(
	ctx context.Context,
	rootID artifactstore.RootID,
) error {
	if w == nil || w.skillRuntime == nil {
		// Workspace remains usable by itself in tests, tools, and future hosts
		// that intentionally do not configure an Agent Skills runtime.
		return nil
	}
	if err := w.skillRuntime.ResyncWorkspace(ctx, rootID); err != nil {
		return fmt.Errorf("sync Workspace Skills: %w", err)
	}
	return nil
}

func (w *WorkspaceWrapper) removeWorkspaceSkills(
	ctx context.Context,
	rootID artifactstore.RootID,
) error {
	if w == nil || w.skillRuntime == nil {
		return nil
	}
	if err := w.skillRuntime.RemoveWorkspace(ctx, rootID); err != nil {
		return fmt.Errorf("remove Workspace Skills: %w", err)
	}
	return nil
}

func (w *WorkspaceWrapper) close() {
	if w == nil || w.api == nil {
		return
	}
	_ = w.api.Close()
	w.skillRuntime = nil
	w.api = nil
}
