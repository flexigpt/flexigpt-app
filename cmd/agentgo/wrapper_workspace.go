package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/workspace"
)

type WorkspaceWrapper struct {
	api *workspace.API
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

func (w *WorkspaceWrapper) CreateFilesystemWorkspace(
	request *workspace.CreateFilesystemWorkspaceRequest,
) (*workspace.CreateFilesystemWorkspaceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.CreateFilesystemWorkspaceResponse, error) {
		return w.api.CreateFilesystemWorkspace(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) CreateEmptyWorkspace(
	request *workspace.CreateEmptyWorkspaceRequest,
) (*workspace.CreateEmptyWorkspaceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.CreateEmptyWorkspaceResponse, error) {
		return w.api.CreateEmptyWorkspace(context.Background(), request)
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
		return w.api.UpdateWorkspace(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) DeleteWorkspace(
	request *workspace.DeleteWorkspaceRequest,
) (*workspace.DeleteWorkspaceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.DeleteWorkspaceResponse, error) {
		return w.api.DeleteWorkspace(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) AttachWorkspaceSource(
	request *workspace.AttachWorkspaceSourceRequest,
) (*workspace.AttachWorkspaceSourceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.AttachWorkspaceSourceResponse, error) {
		return w.api.AttachWorkspaceSource(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) UpdateWorkspaceAttachment(
	request *workspace.UpdateWorkspaceAttachmentRequest,
) (*workspace.UpdateWorkspaceAttachmentResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.UpdateWorkspaceAttachmentResponse, error) {
		return w.api.UpdateWorkspaceAttachment(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) DetachWorkspaceSource(
	request *workspace.DetachWorkspaceSourceRequest,
) (*workspace.DetachWorkspaceSourceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.DetachWorkspaceSourceResponse, error) {
		return w.api.DetachWorkspaceSource(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) RefreshWorkspace(
	request *workspace.RefreshWorkspaceRequest,
) (*workspace.RefreshWorkspaceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.RefreshWorkspaceResponse, error) {
		return w.api.RefreshWorkspace(context.Background(), request)
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
		return w.api.SetWorkspaceRecordEnabled(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) PinWorkspaceRecord(
	request *workspace.PinWorkspaceRecordRequest,
) (*workspace.PinWorkspaceRecordResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.PinWorkspaceRecordResponse, error) {
		return w.api.PinWorkspaceRecord(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) FollowWorkspaceRecord(
	request *workspace.FollowWorkspaceRecordRequest,
) (*workspace.FollowWorkspaceRecordResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.FollowWorkspaceRecordResponse, error) {
		return w.api.FollowWorkspaceRecord(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) DeleteWorkspaceRecord(
	request *workspace.DeleteWorkspaceRecordRequest,
) (*workspace.DeleteWorkspaceRecordResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.DeleteWorkspaceRecordResponse, error) {
		return w.api.DeleteWorkspaceRecord(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) UpdateWorkspaceRecordData(
	request *workspace.UpdateWorkspaceRecordDataRequest,
) (*workspace.UpdateWorkspaceRecordDataResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.UpdateWorkspaceRecordDataResponse, error) {
		return w.api.UpdateWorkspaceRecordData(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) close() {
	if w == nil || w.api == nil {
		return
	}
	_ = w.api.Close()
}
