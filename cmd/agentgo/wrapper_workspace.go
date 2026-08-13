package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/workspace"
)

type WorkspaceWrapper struct {
	api *workspace.API
}

func InitWorkspaceWrapper(
	wrapper *WorkspaceWrapper,
	artifacts *system.Components,
) error {
	if wrapper == nil {
		return errors.New("workspace wrapper is nil")
	}
	if artifacts == nil {
		return errors.New("artifact store components are nil")
	}
	config := workspace.DefaultConfig()
	config.WorkspaceRootID = artifactbuiltin.WorkspaceRootID
	config.AutoAdoptionIDProvider = artifact.UUIDArtifactIDProvider()

	api, err := workspace.New(workspace.Dependencies{
		Roots:                  artifacts.Roots,
		Sources:                artifacts.Sources,
		Collections:            artifacts.Collections,
		Artifacts:              artifacts.Artifacts,
		Refresh:                artifacts.Refresh,
		Catalogs:               artifacts.Catalogs,
		ShareableCanonicalizer: artifacts.ShareableSchemas,
		SourceRuntime:          artifacts.SourceRuntime,
		HasDecoder:             artifacts.HasDecoder,
		DecoderFingerprint:     artifacts.DecoderFingerprint,
		RootMutationPolicy:     artifacts.RootMutationPolicy(),
	}, config)
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
		ctx := context.Background()
		response, err := w.api.CreateFilesystemWorkspace(ctx, request)
		if err != nil {
			return nil, err
		}
		if response == nil || response.Body == nil {
			return nil, errors.New("create filesystem Workspace returned an empty response")
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

		return response, nil
	})
}

func (w *WorkspaceWrapper) ReplaceWorkspacePrimarySource(
	request *workspace.ReplaceWorkspacePrimarySourceRequest,
) (*workspace.ReplaceWorkspacePrimarySourceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.ReplaceWorkspacePrimarySourceResponse, error) {
		ctx := context.Background()
		response, err := w.api.ReplaceWorkspacePrimarySource(ctx, request)
		if err != nil {
			return nil, err
		}

		return response, nil
	})
}

func (w *WorkspaceWrapper) SetWorkspacePrimarySource(
	request *workspace.SetWorkspacePrimarySourceRequest,
) (*workspace.SetWorkspacePrimarySourceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.SetWorkspacePrimarySourceResponse, error) {
		ctx := context.Background()
		response, err := w.api.SetWorkspacePrimarySource(ctx, request)
		if err != nil {
			return nil, err
		}

		return response, nil
	})
}

func (w *WorkspaceWrapper) RetireWorkspace(
	request *workspace.RetireWorkspaceRequest,
) (*workspace.RetireWorkspaceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.RetireWorkspaceResponse, error) {
		ctx := context.Background()
		response, err := w.api.RetireWorkspace(ctx, request)
		if err != nil {
			return nil, err
		}

		return response, nil
	})
}

func (w *WorkspaceWrapper) PurgeWorkspace(
	request *workspace.PurgeWorkspaceRequest,
) (*workspace.PurgeWorkspaceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.PurgeWorkspaceResponse, error) {
		ctx := context.Background()
		response, err := w.api.PurgeWorkspace(ctx, request)
		if err != nil {
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

func (w *WorkspaceWrapper) ComposeWorkspaceLoadPlan(
	request *workspace.ComposeWorkspaceLoadPlanRequest,
) (*workspace.ComposeWorkspaceLoadPlanResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.ComposeWorkspaceLoadPlanResponse, error) {
		return w.api.ComposeWorkspaceLoadPlan(
			context.Background(),
			request,
		)
	})
}

func (w *WorkspaceWrapper) ResolveWorkspaceResource(
	request *workspace.ResolveWorkspaceResourceRequest,
) (*workspace.ResolveWorkspaceResourceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.ResolveWorkspaceResourceResponse, error) {
		return w.api.ResolveWorkspaceResource(
			context.Background(),
			request,
		)
	})
}

func (w *WorkspaceWrapper) GetWorkspaceArtifact(
	request *workspace.GetWorkspaceArtifactRequest,
) (*workspace.GetWorkspaceArtifactResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.GetWorkspaceArtifactResponse, error) {
		return w.api.GetWorkspaceArtifact(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) ListWorkspaceArtifacts(
	request *workspace.ListWorkspaceArtifactsRequest,
) (*workspace.ListWorkspaceArtifactsResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.ListWorkspaceArtifactsResponse, error) {
		return w.api.ListWorkspaceArtifacts(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) AdoptWorkspaceOccurrence(
	request *workspace.AdoptWorkspaceOccurrenceRequest,
) (*workspace.AdoptWorkspaceOccurrenceResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.AdoptWorkspaceOccurrenceResponse, error) {
		ctx := context.Background()
		response, err := w.api.AdoptWorkspaceOccurrence(ctx, request)
		if err != nil {
			return nil, err
		}

		return response, nil
	})
}

func (w *WorkspaceWrapper) PinWorkspaceArtifact(
	request *workspace.PinWorkspaceArtifactRequest,
) (*workspace.PinWorkspaceArtifactResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.PinWorkspaceArtifactResponse, error) {
		ctx := context.Background()
		response, err := w.api.PinWorkspaceArtifact(ctx, request)
		if err != nil {
			return nil, err
		}

		return response, nil
	})
}

func (w *WorkspaceWrapper) ListWorkspaceSuppressions(
	request *workspace.ListWorkspaceSuppressionsRequest,
) (*workspace.ListWorkspaceSuppressionsResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.ListWorkspaceSuppressionsResponse, error) {
		return w.api.ListWorkspaceSuppressions(context.Background(), request)
	})
}

func (w *WorkspaceWrapper) SuppressWorkspaceBinding(
	request *workspace.SuppressWorkspaceBindingRequest,
) (*workspace.SuppressWorkspaceBindingResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.SuppressWorkspaceBindingResponse, error) {
		ctx := context.Background()
		response, err := w.api.SuppressWorkspaceBinding(ctx, request)
		if err != nil {
			return nil, err
		}

		return response, nil
	})
}

func (w *WorkspaceWrapper) UnsuppressWorkspaceBinding(
	request *workspace.UnsuppressWorkspaceBindingRequest,
) (*workspace.UnsuppressWorkspaceBindingResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.UnsuppressWorkspaceBindingResponse, error) {
		ctx := context.Background()
		response, err := w.api.UnsuppressWorkspaceBinding(ctx, request)
		if err != nil {
			return nil, err
		}

		return response, nil
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

func (w *WorkspaceWrapper) SetWorkspaceArtifactEnabled(
	request *workspace.SetWorkspaceArtifactEnabledRequest,
) (*workspace.SetWorkspaceArtifactEnabledResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.SetWorkspaceArtifactEnabledResponse, error) {
		ctx := context.Background()
		response, err := w.api.SetWorkspaceArtifactEnabled(ctx, request)
		if err != nil {
			return nil, err
		}

		return response, nil
	})
}

func (w *WorkspaceWrapper) UnadoptWorkspaceArtifact(
	request *workspace.UnadoptWorkspaceArtifactRequest,
) (*workspace.UnadoptWorkspaceArtifactResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.UnadoptWorkspaceArtifactResponse, error) {
		ctx := context.Background()
		response, err := w.api.UnadoptWorkspaceArtifact(ctx, request)
		if err != nil {
			return nil, err
		}

		return response, nil
	})
}

func (w *WorkspaceWrapper) PurgeWorkspaceArtifact(
	request *workspace.PurgeWorkspaceArtifactRequest,
) (*workspace.PurgeWorkspaceArtifactResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.PurgeWorkspaceArtifactResponse, error) {
		ctx := context.Background()
		response, err := w.api.PurgeWorkspaceArtifact(ctx, request)
		if err != nil {
			return nil, err
		}

		return response, nil
	})
}

func (w *WorkspaceWrapper) SetWorkspaceArtifactRuntimeDisabled(
	request *workspace.SetWorkspaceArtifactRuntimeDisabledRequest,
) (*workspace.SetWorkspaceArtifactRuntimeDisabledResponse, error) {
	return middleware.WithRecoveryResp(func() (*workspace.SetWorkspaceArtifactRuntimeDisabledResponse, error) {
		ctx := context.Background()
		response, err := w.api.SetWorkspaceArtifactRuntimeDisabled(ctx, request)
		if err != nil {
			return nil, err
		}

		return response, nil
	})
}

func (w *WorkspaceWrapper) close() {
	if w == nil {
		return
	}
	api := w.api
	w.api = nil
	if api != nil {
		_ = api.Close()
	}
}
