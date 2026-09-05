package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync/atomic"

	"github.com/flexigpt/agentskills-go/document"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/skill/store/workspaceadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/artifactadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/provision"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

// API is the workspace aggregate boundary for HTTP, Wails, CLI, and other
// application transports. It owns API-safe projections and never exposes raw
// source configuration or artifact-store composition details.
type API struct {
	dependencies Dependencies
	workspace    *components
	provisioner  *provision.Service
	closed       atomic.Bool
}

func New(
	dependencies Dependencies,
	config Config,
) (*API, error) {
	if err := dependencies.Validate(); err != nil {
		return nil, err
	}
	config = config.normalized()

	workspaceComponents, err := newComponents(dependencies, config)
	if err != nil {
		return nil, err
	}
	provisioner, err := provision.NewService(
		workspaceComponents.service,
		dependencies.Store,
	)
	if err != nil {
		return nil, err
	}
	return &API{
		dependencies: dependencies,
		workspace:    workspaceComponents,
		provisioner:  provisioner,
	}, nil
}

func (a *API) Close() error {
	if a != nil {
		a.closed.Store(true)
	}
	return nil
}

// SkillAdapter returns the Workspace-owned Skill source adapter. Consumers may
// list or load Workspace Skills, but lifecycle policy remains outside workspace.
func (a *API) SkillAdapter() *workspaceadapter.Adapter {
	if a == nil ||
		a.closed.Load() ||
		a.workspace == nil {
		return nil
	}
	return a.workspace.skillAdapter
}

func (a *API) CreateFilesystemWorkspace(
	ctx context.Context,
	request *CreateFilesystemWorkspaceRequest,
) (*CreateFilesystemWorkspaceResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("filesystem workspace body is required")
	}

	// Workspace is composed with one application-owned Root. Root selection
	// is not a client-controlled part of Workspace creation.
	rootID := a.workspace.workspaceRootID
	value, err := a.provisioner.CreateFilesystem(ctx, provision.Request{
		DisplayName:      request.Body.DisplayName,
		Description:      request.Body.Description,
		RootPath:         request.Body.RootPath,
		CollectionID:     request.Body.WorkspaceID,
		SourceID:         request.Body.SourceID,
		SourceStorageKey: request.Body.SourceStorageKey,
		RootID:           rootID,
		Discovery:        discoveryPreferencesOf(request.Body.Discovery),
	})
	if err != nil {
		return nil, err
	}
	view, err := a.workspaceViewForAPI(ctx, value)
	if err != nil {
		return nil, err
	}
	return &CreateFilesystemWorkspaceResponse{Body: &view}, nil
}

func (a *API) CreateEmptyWorkspace(
	ctx context.Context,
	request *CreateEmptyWorkspaceRequest,
) (*CreateEmptyWorkspaceResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("empty workspace body is required")
	}

	// Keep empty and filesystem Workspaces in the same configured namespace,
	// independently of any Root currently selected by a client.
	rootID := a.workspace.workspaceRootID
	value, err := a.workspace.service.CreateEmpty(
		ctx,
		spec.EmptyWorkspaceRequest{
			CollectionID: request.Body.WorkspaceID,
			RootID:       rootID,
			DisplayName:  request.Body.DisplayName,
			Description:  request.Body.Description,
			Discovery:    discoveryPreferencesOf(request.Body.Discovery),
		},
	)
	if err != nil {
		return nil, err
	}
	view, err := a.workspaceViewForAPI(ctx, value)
	if err != nil {
		return nil, err
	}
	return &CreateEmptyWorkspaceResponse{Body: &view}, nil
}

func (a *API) GetWorkspace(
	ctx context.Context,
	request *GetWorkspaceRequest,
) (*GetWorkspaceResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace request is required")
	}
	value, err := a.workspace.service.Get(ctx, request.Workspace)
	if err != nil {
		return nil, err
	}
	view, err := a.workspaceViewForAPI(ctx, value)
	if err != nil {
		return nil, err
	}
	return &GetWorkspaceResponse{Body: &view}, nil
}

func (a *API) ListWorkspaces(
	ctx context.Context,
	request *ListWorkspacesRequest,
) (*ListWorkspacesResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace list request is required")
	}

	// Listing follows the same single-Root contract as creation. In
	// particular, a stale UI cannot enumerate the protected built-in Root.
	values, err := a.workspace.service.List(ctx, a.workspace.workspaceRootID)
	if err != nil {
		return nil, err
	}
	output := make([]WorkspaceView, 0, len(values))
	for _, value := range values {
		view, err := a.workspaceViewForAPI(ctx, value)
		if err != nil {
			return nil, err
		}
		output = append(output, view)
	}
	return &ListWorkspacesResponse{
		Body: &ListWorkspacesResponseBody{Workspaces: output},
	}, nil
}

func (a *API) UpdateWorkspace(
	ctx context.Context,
	request *UpdateWorkspaceRequest,
) (*UpdateWorkspaceResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace update body is required")
	}
	value, err := a.workspace.service.Update(ctx, spec.UpdateRequest{
		Workspace:        request.Workspace,
		ExpectedRevision: request.Body.ExpectedRevision,
		DisplayName:      request.Body.DisplayName,
		Description:      request.Body.Description,
		Enabled:          request.Body.Enabled,
		Discovery:        discoveryPreferencesOf(request.Body.Discovery),
	})
	if err != nil {
		return nil, err
	}
	view, err := a.workspaceViewForAPI(ctx, value)
	if err != nil {
		return nil, err
	}
	return &UpdateWorkspaceResponse{Body: &view}, nil
}

func (a *API) SetWorkspacePrimarySource(
	ctx context.Context,
	request *SetWorkspacePrimarySourceRequest,
) (*SetWorkspacePrimarySourceResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace primary-source body is required")
	}
	value, err := a.workspace.service.SetPrimary(
		ctx,
		spec.SetPrimaryRequest{
			Workspace:                  request.Workspace,
			ExpectedCollectionRevision: request.Body.ExpectedCollectionRevision,
			PreviousSourceID:           request.Body.PreviousSourceID,
			PreviousAttachmentRevision: request.Body.ExpectedPreviousAttachmentRevision,
			SourceID:                   request.Body.SourceID,
			Clear:                      request.Body.Clear,
		},
	)
	if err != nil {
		return nil, err
	}
	view, err := a.workspaceViewForAPI(ctx, value)
	if err != nil {
		return nil, err
	}
	return &SetWorkspacePrimarySourceResponse{Body: &view}, nil
}

func (a *API) RetireWorkspace(
	ctx context.Context,
	request *RetireWorkspaceRequest,
) (*RetireWorkspaceResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace retirement request is required")
	}
	value, err := a.workspace.service.Retire(
		ctx,
		request.Workspace,
		request.ExpectedRevision,
	)
	if err != nil {
		return nil, err
	}
	return &RetireWorkspaceResponse{
		Body: &RetireWorkspaceResponseBody{
			Workspace: value.Ref(),
			Revision:  value.Revision,
		},
	}, nil
}

func (a *API) PurgeWorkspace(
	ctx context.Context,
	request *PurgeWorkspaceRequest,
) (*PurgeWorkspaceResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace purge request is required")
	}
	if err := a.workspace.service.Purge(
		ctx,
		request.Workspace,
		request.ExpectedRevision,
	); err != nil {
		return nil, err
	}
	return &PurgeWorkspaceResponse{
		Body: &PurgeWorkspaceResponseBody{
			Workspace: request.Workspace,
		},
	}, nil
}

func (a *API) AttachWorkspaceSource(
	ctx context.Context,
	request *AttachWorkspaceSourceRequest,
) (*AttachWorkspaceSourceResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace attachment body is required")
	}
	value, err := a.workspace.service.Attach(ctx, spec.AttachRequest{
		Workspace:                  request.Workspace,
		ExpectedCollectionRevision: request.Body.ExpectedCollectionRevision,
		SourceID:                   request.Body.SourceID,
		Role:                       request.Body.Role,
		Enabled:                    request.Body.Enabled,
		Data:                       attachmentDataOf(request.Body.Settings),
	})
	if err != nil {
		return nil, err
	}
	view, err := a.workspaceViewForAPI(ctx, value)
	if err != nil {
		return nil, err
	}
	return &AttachWorkspaceSourceResponse{Body: &view}, nil
}

func (a *API) UpdateWorkspaceAttachment(
	ctx context.Context,
	request *UpdateWorkspaceAttachmentRequest,
) (*UpdateWorkspaceAttachmentResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace attachment update body is required")
	}
	value, err := a.workspace.service.UpdateAttachment(
		ctx,
		spec.UpdateAttachmentRequest{
			Workspace:                  request.Workspace,
			SourceID:                   request.SourceID,
			ExpectedCollectionRevision: request.Body.ExpectedCollectionRevision,
			ExpectedAttachmentRevision: request.Body.ExpectedAttachmentRevision,
			Role:                       request.Body.Role,
			Enabled:                    request.Body.Enabled,
			Data:                       attachmentDataOf(request.Body.Settings),
		},
	)
	if err != nil {
		return nil, err
	}
	view, err := a.workspaceViewForAPI(ctx, value)
	if err != nil {
		return nil, err
	}
	return &UpdateWorkspaceAttachmentResponse{Body: &view}, nil
}

func (a *API) DetachWorkspaceSource(
	ctx context.Context,
	request *DetachWorkspaceSourceRequest,
) (*DetachWorkspaceSourceResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace detach request is required")
	}
	value, err := a.workspace.service.Detach(
		ctx,
		request.Workspace,
		request.SourceID,
		request.ExpectedCollectionRevision,
		request.ExpectedAttachmentRevision,
	)
	if err != nil {
		return nil, err
	}
	view, err := a.workspaceViewForAPI(ctx, value)
	if err != nil {
		return nil, err
	}
	return &DetachWorkspaceSourceResponse{Body: &view}, nil
}

func (a *API) RefreshWorkspace(
	ctx context.Context,
	request *RefreshWorkspaceRequest,
) (*RefreshWorkspaceResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace refresh request is required")
	}
	if _, err := a.workspace.service.Get(ctx, request.Workspace); err != nil {
		return nil, err
	}
	value, err := a.dependencies.Store.RefreshCollection(
		ctx,
		request.Workspace,
	)
	if err != nil {
		return nil, err
	}
	output := WorkspaceRefreshResult{
		Workspace:       request.Workspace,
		CatalogRevision: value.Catalog.Revision,
		CreatedArtifacts: append(
			make([]artifact.ArtifactRef, 0, len(value.CreatedArtifacts)),
			artifactRefsOf(request.Workspace.RootID, value.CreatedArtifacts)...,
		),
		UpdatedArtifacts: append(
			make([]artifact.ArtifactRef, 0, len(value.UpdatedArtifacts)),
			artifactRefsOf(request.Workspace.RootID, value.UpdatedArtifacts)...,
		),
		Diagnostics: providerapi.CloneDiagnostics(value.Diagnostics),
		Candidates:  value.Candidates,
	}
	return &RefreshWorkspaceResponse{Body: &output}, nil
}

func (a *API) GetWorkspaceCatalog(
	ctx context.Context,
	request *GetWorkspaceCatalogRequest,
) (*GetWorkspaceCatalogResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace catalog request is required")
	}
	value, err := a.workspace.query.Catalog(ctx, request.Workspace)
	if err != nil {
		return nil, err
	}
	output, err := a.workspaceCatalogViewForAPI(ctx, value)
	if err != nil {
		return nil, err
	}
	return &GetWorkspaceCatalogResponse{Body: &output}, nil
}

func (a *API) GetWorkspaceArtifact(
	ctx context.Context,
	request *GetWorkspaceArtifactRequest,
) (*GetWorkspaceArtifactResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace Artifact request is required")
	}
	value, err := a.workspaceArtifact(
		ctx,
		request.Workspace,
		request.Artifact,
	)
	if err != nil {
		return nil, err
	}
	output := workspaceArtifactViewOf(value)
	return &GetWorkspaceArtifactResponse{Body: &output}, nil
}

func (a *API) ListWorkspaceArtifacts(
	ctx context.Context,
	request *ListWorkspaceArtifactsRequest,
) (*ListWorkspaceArtifactsResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace Artifact list request is required")
	}
	if _, err := a.workspace.service.Get(ctx, request.Workspace); err != nil {
		return nil, err
	}
	values, err := a.dependencies.Store.ListCollectionArtifacts(
		ctx,
		request.Workspace,
	)
	if err != nil {
		return nil, err
	}
	output := make([]WorkspaceArtifactView, 0, len(values))
	for _, value := range values {
		output = append(output, workspaceArtifactViewOf(value))
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Artifact.ArtifactID < output[right].Artifact.ArtifactID
	})
	return &ListWorkspaceArtifactsResponse{
		Body: &ListWorkspaceArtifactsResponseBody{Artifacts: output},
	}, nil
}

func (a *API) AdoptWorkspaceOccurrence(
	ctx context.Context,
	request *AdoptWorkspaceOccurrenceRequest,
) (*AdoptWorkspaceOccurrenceResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace occurrence adoption body is required")
	}
	if request.Body.ExpectedCatalogRevision == 0 {
		return nil, invalidAPIRequest("expected catalog revision is required")
	}

	key := catalog.OccurrenceKey{
		CollectionID:       request.Workspace.CollectionID,
		SourceID:           request.Body.Occurrence.SourceID,
		Locator:            request.Body.Occurrence.Locator,
		SubresourceLocator: request.Body.Occurrence.SubresourceLocator,
	}
	if err := key.Validate(); err != nil {
		return nil, err
	}
	occurrence, err := a.workspaceOccurrence(ctx, request.Workspace, key)
	if err != nil {
		return nil, err
	}
	if err := a.requireWorkspaceArtifactKind(occurrence.Kind); err != nil {
		return nil, err
	}

	data, err := workspaceArtifactDataOf(request.Body.Settings)
	if err != nil {
		return nil, err
	}
	value, err := a.dependencies.Store.AdoptArtifact(ctx, artifact.AdoptRequest{
		ArtifactID:              request.Body.ArtifactID,
		Collection:              request.Workspace,
		Occurrence:              key,
		ExpectedCatalogRevision: request.Body.ExpectedCatalogRevision,
		Name:                    request.Body.Name,
		Enabled:                 request.Body.Enabled,
		Data:                    data,
	})
	if err != nil {
		return nil, err
	}
	output := workspaceArtifactViewOf(value)
	return &AdoptWorkspaceOccurrenceResponse{Body: &output}, nil
}

func (a *API) PinWorkspaceArtifact(
	ctx context.Context,
	request *PinWorkspaceArtifactRequest,
) (*PinWorkspaceArtifactResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace artifact pin body is required")
	}
	if request.Body.ExpectedCollectionRevision == 0 {
		return nil, invalidAPIRequest("expected collection revision is required")
	}
	if _, err := a.workspace.service.Get(ctx, request.Workspace); err != nil {
		return nil, err
	}
	if err := a.requireWorkspaceArtifactKind(
		request.Body.Binding.ExpectedKind,
	); err != nil {
		return nil, err
	}
	data, err := workspaceArtifactDataOf(request.Body.Settings)
	if err != nil {
		return nil, err
	}
	value, err := a.dependencies.Store.PinArtifact(ctx, artifact.PinRequest{
		ArtifactID:                 request.Body.ArtifactID,
		Collection:                 request.Workspace,
		ExpectedCollectionRevision: request.Body.ExpectedCollectionRevision,
		Binding:                    request.Body.Binding,
		Name:                       request.Body.Name,
		Enabled:                    request.Body.Enabled,
		Data:                       data,
	})
	if err != nil {
		return nil, err
	}
	output := workspaceArtifactViewOf(value)
	return &PinWorkspaceArtifactResponse{Body: &output}, nil
}

func (a *API) ListWorkspaceSuppressions(
	ctx context.Context,
	request *ListWorkspaceSuppressionsRequest,
) (*ListWorkspaceSuppressionsResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace suppression list request is required")
	}
	if _, err := a.workspace.service.Get(ctx, request.Workspace); err != nil {
		return nil, err
	}
	values, err := a.dependencies.Store.ListCollectionSuppressions(
		ctx,
		request.Workspace,
	)
	if err != nil {
		return nil, err
	}
	output := make([]WorkspaceSuppressionView, 0, len(values))
	for _, value := range values {
		output = append(output, workspaceSuppressionViewOf(value))
	}
	return &ListWorkspaceSuppressionsResponse{
		Body: &ListWorkspaceSuppressionsResponseBody{Suppressions: output},
	}, nil
}

func (a *API) SuppressWorkspaceBinding(
	ctx context.Context,
	request *SuppressWorkspaceBindingRequest,
) (*SuppressWorkspaceBindingResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace suppression body is required")
	}
	if request.Body.ExpectedCollectionRevision == 0 {
		return nil, invalidAPIRequest("expected collection revision is required")
	}
	if _, err := a.workspace.service.Get(ctx, request.Workspace); err != nil {
		return nil, err
	}
	if err := a.requireWorkspaceArtifactKind(
		request.Body.Binding.ExpectedKind,
	); err != nil {
		return nil, err
	}
	value, err := a.dependencies.Store.SuppressBinding(ctx, artifact.SuppressRequest{
		Collection:                 request.Workspace,
		ExpectedCollectionRevision: request.Body.ExpectedCollectionRevision,
		Binding:                    request.Body.Binding,
	})
	if err != nil {
		return nil, err
	}
	output := workspaceSuppressionViewOf(value)
	return &SuppressWorkspaceBindingResponse{Body: &output}, nil
}

func (a *API) UnsuppressWorkspaceBinding(
	ctx context.Context,
	request *UnsuppressWorkspaceBindingRequest,
) (*UnsuppressWorkspaceBindingResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace unsuppression request is required")
	}
	if _, err := a.workspace.service.Get(ctx, request.Workspace); err != nil {
		return nil, err
	}
	if err := a.dependencies.Store.UnsuppressBinding(
		ctx,
		request.Workspace,
		request.Binding,
		request.ExpectedRevision,
	); err != nil {
		return nil, err
	}
	return &UnsuppressWorkspaceBindingResponse{
		Body: &UnsuppressWorkspaceBindingResponseBody{
			Workspace: request.Workspace,
			Binding:   request.Binding,
		},
	}, nil
}

func (a *API) ListWorkspaceContexts(
	ctx context.Context,
	request *ListWorkspaceContextsRequest,
) (*ListWorkspaceContextsResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace Context list request is required")
	}
	values, err := a.workspace.contextAdapter.List(ctx, request.Workspace)
	if err != nil {
		return nil, err
	}
	output := make([]WorkspaceContextView, 0, len(values))
	for _, value := range values {
		output = append(output, contextViewOf(value))
	}
	return &ListWorkspaceContextsResponse{
		Body: &ListWorkspaceContextsResponseBody{Contexts: output},
	}, nil
}

func (a *API) LoadWorkspaceContexts(
	ctx context.Context,
	request *LoadWorkspaceContextsRequest,
) (*LoadWorkspaceContextsResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace Context load body is required")
	}
	value, err := a.workspace.contextAdapter.Load(
		ctx,
		request.Workspace,
		request.Body.Artifacts,
	)
	if err != nil {
		return nil, err
	}
	output := WorkspaceContextInspectionView{
		Workspace:       value.Workspace,
		CatalogRevision: value.CatalogRevision,
		Diagnostics:     providerapi.CloneDiagnostics(value.Diagnostics),
		Contributions: make(
			[]WorkspaceContextContribution,
			0,
			len(value.Contributions),
		),
	}
	for _, contribution := range value.Contributions {
		output.Contributions = append(
			output.Contributions,
			contextContributionViewOf(contribution),
		)
	}
	return &LoadWorkspaceContextsResponse{Body: &output}, nil
}

func (a *API) ComposeWorkspaceContext(
	ctx context.Context,
	request *ComposeWorkspaceContextRequest,
) (*ComposeWorkspaceContextResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace context body is required")
	}
	value, err := a.workspace.contextAdapter.Compose(
		ctx,
		request.Workspace,
		request.Body.Artifacts,
	)
	if err != nil {
		return nil, err
	}
	output := contextLoadPlanViewOf(value)
	return &ComposeWorkspaceContextResponse{Body: &output}, nil
}

func (a *API) ListWorkspaceSkills(
	ctx context.Context,
	request *ListWorkspaceSkillsRequest,
) (*ListWorkspaceSkillsResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace skill list request is required")
	}
	values, err := a.workspace.skillAdapter.List(ctx, request.Workspace)
	if err != nil {
		return nil, err
	}
	output := make([]WorkspaceSkillView, 0, len(values))
	for _, value := range values {
		output = append(output, workspaceSkillViewOf(value))
	}
	return &ListWorkspaceSkillsResponse{
		Body: &ListWorkspaceSkillsResponseBody{Skills: output},
	}, nil
}

func (a *API) LoadWorkspaceSkills(
	ctx context.Context,
	request *LoadWorkspaceSkillsRequest,
) (*LoadWorkspaceSkillsResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace skill load body is required")
	}
	value, err := a.workspace.skillAdapter.Load(
		ctx,
		request.Workspace,
		request.Body.Artifacts,
	)
	if err != nil {
		return nil, err
	}
	output := workspaceSkillLoadViewOf(value)
	return &LoadWorkspaceSkillsResponse{Body: &output}, nil
}

func (a *API) SetWorkspaceArtifactEnabled(
	ctx context.Context,
	request *SetWorkspaceArtifactEnabledRequest,
) (*SetWorkspaceArtifactEnabledResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace Artifact update body is required")
	}
	if _, err := a.workspaceArtifact(ctx, request.Workspace, request.Artifact); err != nil {
		return nil, err
	}
	value, err := a.dependencies.Store.SetArtifactEnabled(
		ctx,
		request.Artifact,
		request.Body.ExpectedRevision,
		request.Body.Enabled,
	)
	if err != nil {
		return nil, err
	}
	output := workspaceArtifactViewOf(value)
	return &SetWorkspaceArtifactEnabledResponse{Body: &output}, nil
}

func (a *API) UnadoptWorkspaceArtifact(
	ctx context.Context,
	request *UnadoptWorkspaceArtifactRequest,
) (*UnadoptWorkspaceArtifactResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace Artifact unadopt request is required")
	}
	if _, err := a.workspaceArtifact(ctx, request.Workspace, request.Artifact); err != nil {
		return nil, err
	}
	if err := a.dependencies.Store.UnadoptArtifact(
		ctx,
		request.Artifact,
		request.ExpectedRevision,
		request.Suppress,
	); err != nil {
		return nil, err
	}
	return &UnadoptWorkspaceArtifactResponse{
		Body: &UnadoptWorkspaceArtifactResponseBody{
			Artifact: request.Artifact,
		},
	}, nil
}

func (a *API) PurgeWorkspaceArtifact(
	ctx context.Context,
	request *PurgeWorkspaceArtifactRequest,
) (*PurgeWorkspaceArtifactResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace Artifact purge request is required")
	}
	if _, err := a.workspaceArtifact(
		ctx,
		request.Workspace,
		request.Artifact,
	); err != nil {
		return nil, err
	}
	if err := a.dependencies.Store.PurgeArtifact(
		ctx,
		request.Artifact,
		request.ExpectedRevision,
	); err != nil {
		return nil, err
	}
	return &PurgeWorkspaceArtifactResponse{
		Body: &PurgeWorkspaceArtifactResponseBody{
			Artifact: request.Artifact,
		},
	}, nil
}

func (a *API) SetWorkspaceArtifactRuntimeDisabled(
	ctx context.Context,
	request *SetWorkspaceArtifactRuntimeDisabledRequest,
) (*SetWorkspaceArtifactRuntimeDisabledResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace Artifact data body is required")
	}
	current, err := a.workspaceArtifact(ctx, request.Workspace, request.Artifact)
	if err != nil {
		return nil, err
	}
	artifactData, err := artifactadapter.DecodeArtifactData(current.Data)
	if err != nil {
		return nil, err
	}
	artifactData.RuntimeDisabled = request.Body.RuntimeDisabled
	data, err := artifactadapter.EncodeArtifactData(artifactData)
	if err != nil {
		return nil, err
	}
	value, err := a.dependencies.Store.UpdateArtifactData(
		ctx,
		request.Artifact,
		request.Body.ExpectedRevision,
		data,
	)
	if err != nil {
		return nil, err
	}
	output := workspaceArtifactViewOf(value)
	return &SetWorkspaceArtifactRuntimeDisabledResponse{Body: &output}, nil
}

func (a *API) Ready() error {
	if a == nil ||
		a.closed.Load() ||
		a.workspace == nil ||
		a.provisioner == nil {
		return invalidAPIRequest("workspace API is not initialized")
	}
	return a.dependencies.Validate()
}

func (a *API) workspaceArtifact(
	ctx context.Context,
	workspace collection.CollectionRef,
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	if _, err := a.workspace.service.Get(ctx, workspace); err != nil {
		return artifact.Artifact{}, err
	}
	value, err := a.dependencies.Store.GetArtifact(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if value.RootID != workspace.RootID ||
		value.CollectionID != workspace.CollectionID {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: Artifact %q does not belong to Workspace %q",
			spec.ErrReferenceUnresolved,
			ref.ArtifactID,
			workspace.CollectionID,
		)
	}
	if err := a.requireWorkspaceArtifactKind(value.Kind); err != nil {
		return artifact.Artifact{}, err
	}
	return value, nil
}

func (a *API) workspaceOccurrence(
	ctx context.Context,
	workspace collection.CollectionRef,
	key catalog.OccurrenceKey,
) (catalog.Occurrence, error) {
	view, err := a.workspace.query.Catalog(ctx, workspace)
	if err != nil {
		return catalog.Occurrence{}, err
	}
	if !view.CatalogCurrent {
		return catalog.Occurrence{}, fmt.Errorf(
			"%w: Workspace catalog must be refreshed before an occurrence can be adopted",
			basespec.ErrCatalogStale,
		)
	}
	for _, occurrence := range view.Catalog.Occurrences {
		if occurrence.Key == key {
			return catalog.CloneOccurrence(occurrence), nil
		}
	}

	return catalog.Occurrence{}, fmt.Errorf(
		"%w: Workspace occurrence %q/%q is unavailable",
		spec.ErrReferenceUnresolved,
		key.SourceID,
		key.Locator,
	)
}

func (a *API) requireWorkspaceArtifactKind(
	kind basespec.ArtifactKind,
) error {
	if err := basespec.ValidateArtifactKind(kind); err != nil {
		return err
	}
	if a.workspace == nil {
		return invalidAPIRequest("workspace components are unavailable")
	}
	if _, supported := a.workspace.supportedKinds[kind]; !supported {
		return fmt.Errorf(
			"%w: Artifact kind %q is not supported by Workspace",
			spec.ErrInvalidWorkspace,
			kind,
		)
	}
	return nil
}

func artifactRefsOf(
	rootID basespec.RootID,
	ids []basespec.ArtifactID,
) []artifact.ArtifactRef {
	output := make([]artifact.ArtifactRef, 0, len(ids))
	for _, id := range ids {
		output = append(output, artifact.ArtifactRef{
			RootID:     rootID,
			ArtifactID: id,
		})
	}
	return output
}

func invalidAPIRequest(message string) error {
	return fmt.Errorf("%w: %s", spec.ErrInvalidWorkspace, message)
}

func (a *API) workspaceViewForAPI(
	ctx context.Context,
	value spec.Workspace,
) (WorkspaceView, error) {
	output, err := workspaceViewOf(value)
	if err != nil {
		return WorkspaceView{}, err
	}
	if err := a.enrichWorkspaceSourcePresentation(ctx, &output, value); err != nil {
		return WorkspaceView{}, err
	}
	return output, nil
}

func (a *API) workspaceCatalogViewForAPI(
	ctx context.Context,
	value spec.CatalogView,
) (WorkspaceCatalogView, error) {
	output, err := workspaceCatalogViewOf(value)
	if err != nil {
		return WorkspaceCatalogView{}, err
	}
	if err := a.enrichWorkspaceSourcePresentation(
		ctx,
		&output.Workspace,
		value.Workspace,
	); err != nil {
		return WorkspaceCatalogView{}, err
	}
	return output, nil
}

func (a *API) enrichWorkspaceSourcePresentation(
	ctx context.Context,
	output *WorkspaceView,
	value spec.Workspace,
) error {
	for index := range output.Attachments {
		attachment := &output.Attachments[index]
		var summaryFound bool

		for _, summary := range value.Sources {
			if summary.ID != attachment.SourceID {
				continue
			}
			attachment.SourceDisplayName = summary.DisplayName
			attachment.SourceKind = string(summary.Kind)
			summaryFound = true
			break
		}
		if !summaryFound {
			return fmt.Errorf(
				"%w: Workspace attachment source %q is unavailable",
				spec.ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}
		sourceKind := basespec.SourceKind(attachment.SourceKind)
		if !a.dependencies.Store.SupportsLocalPath(sourceKind) {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		pathValue, err := a.dependencies.Store.ResolveSourceLocalPath(
			ctx,
			value.Collection.RootID,
			attachment.SourceID,
			".",
		)
		if err != nil {
			attachment.Diagnostics = providerapi.AppendDiagnostics(
				attachment.Diagnostics,
				workspaceSourcePresentationDiagnostic(
					"workspace.source.path-unavailable",
					"the filesystem Source path is currently unavailable",
				),
			)
			continue
		}
		attachment.Path = pathValue
		if attachment.SourceID == output.PrimarySourceID {
			output.PrimaryPath = attachment.Path
		}
	}
	return nil
}

func workspaceSourcePresentationDiagnostic(
	code string,
	message string,
) providerapi.Diagnostic {
	return providerapi.Diagnostic{
		Severity: providerapi.DiagnosticWarning,
		Code:     code,
		Message:  message,
	}
}

func discoveryPreferencesOf(value WorkspaceDiscovery) spec.DiscoveryPreferences {
	output := spec.DiscoveryPreferences{
		AdditionalLocators: append(
			[]basespec.Locator(nil),
			value.AdditionalLocators...,
		),
		IncludeReadme: value.IncludeReadme,
	}
	for _, root := range value.AdditionalRoots {
		output.AdditionalRoots = append(output.AdditionalRoots, spec.DiscoveryRoot{
			Root:            root.Root,
			Recursive:       root.Recursive,
			IncludePatterns: append([]string(nil), root.IncludePatterns...),
		})
	}
	return output
}

func attachmentDataOf(value WorkspaceAttachmentSettings) spec.AttachmentData {
	return spec.AttachmentData{
		Recursive:     cloneBool(value.Recursive),
		Authoritative: cloneBool(value.Authoritative),
	}
}

func workspaceSuppressionViewOf(
	value artifact.Suppression,
) WorkspaceSuppressionView {
	return WorkspaceSuppressionView{
		Workspace: collection.CollectionRef{
			RootID: value.RootID, CollectionID: value.CollectionID,
		},
		Binding:    value.Binding,
		Revision:   value.Revision,
		CreatedAt:  value.CreatedAt,
		ModifiedAt: value.ModifiedAt,
	}
}

func workspaceCatalogViewOf(
	value spec.CatalogView,
) (WorkspaceCatalogView, error) {
	workspaceValue, err := workspaceViewOf(value.Workspace)
	if err != nil {
		return WorkspaceCatalogView{}, err
	}
	output := WorkspaceCatalogView{
		Workspace:       workspaceValue,
		CatalogRevision: value.Catalog.Revision,
		CatalogCurrent:  value.CatalogCurrent,
		Diagnostics: providerapi.AppendDiagnostics(
			value.Catalog.Diagnostics,
			value.FreshnessDiagnostics...,
		),
		Resources:             make([]WorkspaceResourceView, 0, len(value.Resources)),
		Groups:                make([]WorkspaceResourceGroupView, 0, len(value.Groups)),
		Occurrences:           make([]WorkspaceOccurrenceView, 0, len(value.Catalog.Occurrences)),
		ValidOccurrences:      make([]WorkspaceOccurrenceView, 0),
		InvalidOccurrences:    make([]WorkspaceOccurrenceView, 0),
		MissingOccurrences:    make([]WorkspaceOccurrenceView, 0),
		UnrecordedOccurrences: make([]WorkspaceOccurrenceView, 0),
		UnresolvedArtifacts:   make([]WorkspaceArtifactView, 0, len(value.UnresolvedArtifacts)),

		UnrecordedCount:         len(value.Unrecorded),
		UnresolvedArtifactCount: len(value.UnresolvedArtifacts),
	}
	artifactsByOccurrence := make(map[string]artifact.Artifact, len(value.Resources))
	for _, resourceValue := range value.Resources {
		artifactView := workspaceArtifactViewOf(resourceValue.Artifact)
		projected := WorkspaceResourceView{
			Artifact:         artifactView,
			DefinitionDigest: resourceValue.Definition.Digest,
			SourceID:         resourceValue.Source.ID,
			Locator:          resourceValue.Artifact.Binding.Locator,
			CatalogCurrent:   resourceValue.CatalogCurrent,
			ProjectionValid:  resourceValue.ProjectionValid,
			Diagnostics: providerapi.AppendDiagnostics(
				artifactView.Diagnostics,
				resourceValue.Diagnostics...,
			),
		}
		output.Resources = append(output.Resources, projected)
		artifactsByOccurrence[occurrenceViewKey(
			resourceValue.Artifact.Binding.SourceID,
			resourceValue.Artifact.Binding.Locator,
			resourceValue.Artifact.Binding.SubresourceLocator,
			resourceValue.Artifact.Kind,
		)] = resourceValue.Artifact
	}
	for _, localArtifact := range value.UnresolvedArtifacts {
		output.UnresolvedArtifacts = append(
			output.UnresolvedArtifacts,
			workspaceArtifactViewOf(localArtifact),
		)
		artifactsByOccurrence[occurrenceViewKey(
			localArtifact.Binding.SourceID,
			localArtifact.Binding.Locator,
			localArtifact.Binding.SubresourceLocator,
			localArtifact.Kind,
		)] = localArtifact
	}
	for _, occurrence := range value.Catalog.Occurrences {
		projected := workspaceOccurrenceViewOf(
			occurrence,
			artifactsByOccurrence,
		)
		output.Occurrences = append(output.Occurrences, projected)
		switch occurrence.State {
		case catalog.OccurrenceValid:
			output.ValidOccurrences = append(output.ValidOccurrences, projected)
		case catalog.OccurrenceInvalid:
			output.InvalidOccurrences = append(output.InvalidOccurrences, projected)
		case catalog.OccurrenceMissing:
			output.MissingOccurrences = append(output.MissingOccurrences, projected)
		default:
		}
		if !projected.Recorded {
			output.UnrecordedOccurrences = append(
				output.UnrecordedOccurrences,
				projected,
			)
		}
	}
	for _, group := range value.Groups {
		projected := WorkspaceResourceGroupView{
			Kind:       group.Kind,
			Resources:  make([]WorkspaceResourceView, 0, len(group.Resources)),
			Unrecorded: make([]WorkspaceOccurrenceView, 0, len(group.Unrecorded)),
		}
		for _, resourceValue := range group.Resources {
			artifactView := workspaceArtifactViewOf(resourceValue.Artifact)
			projected.Resources = append(
				projected.Resources,
				WorkspaceResourceView{
					Artifact:         artifactView,
					DefinitionDigest: resourceValue.Definition.Digest,
					SourceID:         resourceValue.Source.ID,
					Locator:          resourceValue.Artifact.Binding.Locator,
					CatalogCurrent:   resourceValue.CatalogCurrent,
					ProjectionValid:  resourceValue.ProjectionValid,
					Diagnostics: providerapi.AppendDiagnostics(
						artifactView.Diagnostics,
						resourceValue.Diagnostics...,
					),
				},
			)
		}
		for _, occurrence := range group.Unrecorded {
			projected.Unrecorded = append(
				projected.Unrecorded,
				workspaceOccurrenceViewOf(occurrence, artifactsByOccurrence),
			)
		}
		output.Groups = append(output.Groups, projected)
	}
	return output, nil
}

func workspaceViewOf(value spec.Workspace) (WorkspaceView, error) {
	output := WorkspaceView{
		Workspace:       value.Collection.Ref(),
		Revision:        value.Collection.Revision,
		DisplayName:     value.Collection.DisplayName,
		Description:     value.Collection.Description,
		Enabled:         value.Collection.Enabled,
		Mode:            value.Mode,
		PrimarySourceID: value.PrimarySourceID,
		Discovery:       workspaceDiscoveryOf(value.Data.Discovery),
		Attachments:     make([]WorkspaceAttachmentView, 0, len(value.Attachments)),
	}

	for _, attachment := range value.Attachments {
		settings, err := workspaceAttachmentSettingsOf(attachment.Data)
		if err != nil {
			return WorkspaceView{}, err
		}
		output.Attachments = append(output.Attachments, WorkspaceAttachmentView{
			SourceID: attachment.SourceID,
			Revision: attachment.Revision,
			Role:     attachment.Role,
			Enabled:  attachment.Enabled,
			Settings: settings,
		})
	}
	return output, nil
}

func workspaceAttachmentSettingsOf(
	raw json.RawMessage,
) (WorkspaceAttachmentSettings, error) {
	var value spec.AttachmentData
	if err := json.Unmarshal(raw, &value); err != nil {
		return WorkspaceAttachmentSettings{}, fmt.Errorf(
			"%w: decode workspace attachment settings: %w",
			spec.ErrInvalidWorkspace,
			err,
		)
	}
	return WorkspaceAttachmentSettings{
		Recursive:     cloneBool(value.Recursive),
		Authoritative: cloneBool(value.Authoritative),
	}, nil
}

func workspaceDiscoveryOf(value spec.DiscoveryPreferences) WorkspaceDiscovery {
	output := WorkspaceDiscovery{
		AdditionalLocators: append(
			[]basespec.Locator(nil),
			value.AdditionalLocators...,
		),
		IncludeReadme: value.IncludeReadme,
	}
	for _, root := range value.AdditionalRoots {
		output.AdditionalRoots = append(output.AdditionalRoots, WorkspaceDiscoveryRoot{
			Root:            root.Root,
			Recursive:       root.Recursive,
			IncludePatterns: append([]string(nil), root.IncludePatterns...),
		})
	}
	return output
}

func workspaceOccurrenceViewOf(
	value catalog.Occurrence,
	artifacts map[string]artifact.Artifact,
) WorkspaceOccurrenceView {
	output := WorkspaceOccurrenceView{
		SourceID:            value.Key.SourceID,
		Locator:             value.Key.Locator,
		SubresourceLocator:  value.Key.SubresourceLocator,
		Kind:                value.Kind,
		LogicalName:         value.LogicalName,
		LogicalVersion:      value.LogicalVersion,
		DefinitionDigest:    cryptoutil.CloneDigest(value.DefinitionDigest),
		SourceContentDigest: cryptoutil.CloneDigest(value.SourceContentDigest),
		State:               string(value.State),
		Diagnostics:         providerapi.CloneDiagnostics(value.Diagnostics),
	}
	if localArtifact, found := artifacts[occurrenceViewKey(
		value.Key.SourceID,
		value.Key.Locator,
		value.Key.SubresourceLocator,
		value.Kind,
	)]; found {
		artifactRef := localArtifact.Ref()
		output.Recorded = true
		output.Artifact = &artifactRef
	}
	return output
}

func contextLoadPlanViewOf(
	value contextadapter.ContextLoadPlan,
) WorkspaceContextLoadPlan {
	output := WorkspaceContextLoadPlan{
		Workspace:       value.Workspace,
		CatalogRevision: value.CatalogRevision,
		Prompt:          value.Prompt,
		Diagnostics:     providerapi.CloneDiagnostics(value.Diagnostics),
		Contributions:   make([]WorkspaceContextContribution, 0, len(value.Contributions)),
		Decisions:       make([]WorkspaceContextDecision, 0, len(value.Decisions)),
		PromptBytes:     value.PromptBytes,
	}
	for _, contribution := range value.Contributions {
		output.Contributions = append(
			output.Contributions,
			contextContributionViewOf(contribution),
		)
	}
	for _, decision := range value.Decisions {
		output.Decisions = append(output.Decisions, WorkspaceContextDecision{
			Artifact:      decision.Artifact,
			Status:        decision.Status,
			Code:          decision.Code,
			OriginalBytes: decision.OriginalBytes,
			IncludedBytes: decision.IncludedBytes,
		})
	}
	return output
}

func contextContributionViewOf(
	value contextadapter.ContextContribution,
) WorkspaceContextContribution {
	return WorkspaceContextContribution{
		Artifact:         value.Artifact,
		RecordRevision:   value.ArtifactRevision,
		DefinitionDigest: value.DefinitionDigest,
		SourceID:         value.SourceID,
		Locator:          value.Locator,
		Name:             value.Name,
		Role:             value.Role,
		MediaType:        value.MediaType,
		Content:          value.Content,
		ConventionOrder:  value.ConventionOrder,
		OriginalBytes:    value.OriginalBytes,
		IncludedBytes:    value.IncludedBytes,
		Truncated:        value.Truncated,
	}
}

func contextViewOf(value contextadapter.ContextDocument) WorkspaceContextView {
	return WorkspaceContextView{
		Artifact:         value.Artifact,
		RecordRevision:   value.ArtifactRevision,
		DefinitionDigest: value.DefinitionDigest,
		SourceID:         value.SourceID,
		Locator:          value.Locator,
		Name:             value.Name,
		Role:             value.Role,
		MediaType:        value.MediaType,
		Enabled:          value.Enabled,
		State:            value.State,
		CatalogCurrent:   value.CatalogCurrent,
		ProjectionValid:  value.ProjectionValid,
		RuntimeDisabled:  value.RuntimeDisabled,
		Diagnostics:      providerapi.CloneDiagnostics(value.Diagnostics),
	}
}

func workspaceSkillLoadViewOf(
	value workspaceadapter.SkillLoadPlan,
) WorkspaceSkillLoadView {
	output := WorkspaceSkillLoadView{
		Workspace:       value.Workspace,
		CatalogRevision: value.CatalogRevision,
		Diagnostics:     providerapi.CloneDiagnostics(value.Diagnostics),
		Skills:          make([]WorkspaceSkillView, 0, len(value.Skills)),
	}
	for _, skill := range value.Skills {
		output.Skills = append(output.Skills, workspaceSkillViewOf(skill))
	}
	return output
}

func workspaceSkillViewOf(value workspaceadapter.WorkspaceSkill) WorkspaceSkillView {
	summary := WorkspaceSkillSummary{
		SchemaVersion: value.Skill.SchemaVersion,
		ID:            value.Skill.ID,
		Slug:          value.Skill.Slug,
		Name:          value.Skill.Name,
		DisplayName:   value.Skill.DisplayName,
		Description:   value.Skill.Description,
		Tags:          append([]string(nil), value.Skill.Tags...),
		Insert:        document.SkillInsert(value.Skill.Insert),
		IsEnabled:     value.Skill.IsEnabled,
		CreatedAt:     value.Skill.CreatedAt,
		ModifiedAt:    value.Skill.ModifiedAt,
		Arguments:     make([]WorkspaceSkillArgument, 0, len(value.Skill.Arguments)),
	}
	for _, argument := range value.Skill.Arguments {
		summary.Arguments = append(summary.Arguments, WorkspaceSkillArgument{
			Name:        argument.Name,
			Description: argument.Description,
			Default:     argument.Default,
		})
	}
	return WorkspaceSkillView{
		Workspace:        value.Workspace,
		Artifact:         value.Artifact,
		DefinitionDigest: value.DefinitionDigest,
		SourceID:         value.SourceID,
		Locator:          value.Locator,
		Skill:            summary,
		MarkdownBody:     value.MarkdownBody,
		RecordRevision:   value.ArtifactRevision,
		State:            value.State,
		ProjectionValid:  value.ProjectionValid,
		CatalogCurrent:   value.CatalogCurrent,
		RuntimeDisabled:  value.RuntimeDisabled,
		Diagnostics:      providerapi.CloneDiagnostics(value.Diagnostics),
	}
}

func workspaceArtifactDataOf(
	value WorkspaceArtifactSettings,
) (json.RawMessage, error) {
	return artifactadapter.EncodeArtifactData(spec.ArtifactData{
		RuntimeDisabled: value.RuntimeDisabled,
	})
}

func workspaceArtifactViewOf(value artifact.Artifact) WorkspaceArtifactView {
	var digest *cryptoutil.Digest
	if value.ResolvedDefinition != nil {
		copyValue := *value.ResolvedDefinition
		digest = &copyValue
	}
	diagnostics := providerapi.CloneDiagnostics(value.Diagnostics)
	runtimeDisabled, dataErr := artifactadapter.ArtifactRuntimeDisabled(value)
	if dataErr != nil {
		diagnostics = providerapi.AppendDiagnostics(
			diagnostics,
			providerapi.Diagnostic{
				Severity: providerapi.DiagnosticError,
				Code:     artifactadapter.DiagnosticCodeProjectionInvalid,
				Message:  "the Workspace Artifact has invalid local runtime settings",
				Location: &providerapi.DiagnosticLocation{
					Locator:            value.Binding.Locator,
					SubresourceLocator: value.Binding.SubresourceLocator,
				},
			},
		)
	}
	return WorkspaceArtifactView{
		Artifact:           value.Ref(),
		Revision:           value.Revision,
		Name:               value.Name,
		Kind:               value.Kind,
		Enabled:            value.Enabled,
		State:              value.State,
		Adoption:           value.Adoption,
		ResolvedDefinition: digest,
		SourceID:           value.Binding.SourceID,
		Locator:            value.Binding.Locator,
		SubresourceLocator: value.Binding.SubresourceLocator,
		RuntimeDisabled:    runtimeDisabled,
		Diagnostics:        diagnostics,
	}
}

func occurrenceViewKey(
	sourceID basespec.SourceID,
	locator basespec.Locator,
	subresource basespec.SubresourceLocator,
	kind basespec.ArtifactKind,
) string {
	return string(sourceID) + "\x00" +
		string(locator) + "\x00" +
		string(subresource) + "\x00" +
		string(kind)
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
