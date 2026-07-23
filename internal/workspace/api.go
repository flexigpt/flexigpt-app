package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"

	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
	"github.com/flexigpt/flexigpt-app/internal/workspace/provision"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

// OpenConfig configures the workspace aggregate and its private artifact-store
// runtime. Additional adapters and decoders are extension seams, not API
// transport contracts.
type OpenConfig struct {
	BaseDirectory string

	EmbeddedProviders        map[string]fs.FS
	AdditionalSourceAdapters []source.Adapter
	AdditionalDecoders       []discovery.Decoder

	Clock       artifactstore.Clock
	IDGenerator artifactstore.IDGenerator

	WorkspaceConfig Config
}

// API is the workspace aggregate boundary for HTTP, Wails, CLI, and other
// application transports. It owns API-safe projections and never exposes raw
// source configuration or artifact-store composition details.
type API struct {
	artifacts   *system.Components
	workspace   *components
	provisioner *provision.Service
}

func Open(
	ctx context.Context,
	config OpenConfig,
) (*API, error) {
	workspaceConfig := config.WorkspaceConfig
	if len(workspaceConfig.Supports) == 0 {
		workspaceConfig.Supports = BuiltinArtifactSupports()
	}
	skillConventions, err := workspaceConfig.skillConventions()
	if err != nil {
		return nil, err
	}
	skillDecoder, err := skilladapter.NewSkillDecoderWithConventions(
		skillConventions,
	)
	if err != nil {
		return nil, err
	}

	decoders := make(
		[]discovery.Decoder,
		0,
		len(config.AdditionalDecoders)+3,
	)
	decoders = append(
		decoders,
		engine.NewDefinitionDecoder(),
		contextadapter.NewContextDecoder(),
		skillDecoder,
	)
	decoders = append(decoders, config.AdditionalDecoders...)

	artifacts, err := system.Open(ctx, system.Config{
		BaseDirectory:     config.BaseDirectory,
		EmbeddedProviders: config.EmbeddedProviders,
		AdditionalSources: config.AdditionalSourceAdapters,
		Decoders:          decoders,
		Clock:             config.Clock,
		IDGenerator:       config.IDGenerator,
	})
	if err != nil {
		return nil, err
	}

	workspaceComponents, err := newComponents(artifacts, workspaceConfig)
	if err != nil {
		_ = artifacts.Close()
		return nil, err
	}
	provisioner, err := provision.NewService(
		artifacts.Sources,
		workspaceComponents.service,
	)
	if err != nil {
		_ = artifacts.Close()
		return nil, err
	}
	return &API{
		artifacts:   artifacts,
		workspace:   workspaceComponents,
		provisioner: provisioner,
	}, nil
}

func (a *API) Close() error {
	if a == nil || a.artifacts == nil {
		return nil
	}
	return a.artifacts.Close()
}

// SkillAdapter returns the Workspace-owned Skill source adapter. Consumers may
// list or load Workspace Skills, but lifecycle policy remains outside workspace.
func (a *API) SkillAdapter() *skilladapter.Adapter {
	if a == nil || a.workspace == nil {
		return nil
	}
	return a.workspace.skillAdapter
}

func (a *API) CreateFilesystemWorkspace(
	ctx context.Context,
	request *CreateFilesystemWorkspaceRequest,
) (*CreateFilesystemWorkspaceResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("filesystem workspace body is required")
	}
	value, err := a.provisioner.CreateFilesystem(ctx, provision.Request{
		DisplayName:    request.Body.DisplayName,
		Description:    request.Body.Description,
		RootPath:       request.Body.RootPath,
		TrustReference: request.Body.TrustReference,
		Discovery:      discoveryPreferencesOf(request.Body.Discovery),
	})
	if err != nil {
		return nil, err
	}
	view, err := workspaceViewOf(value)
	if err != nil {
		return nil, err
	}
	return &CreateFilesystemWorkspaceResponse{Body: &view}, nil
}

func (a *API) CreateEmptyWorkspace(
	ctx context.Context,
	request *CreateEmptyWorkspaceRequest,
) (*CreateEmptyWorkspaceResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("empty workspace body is required")
	}
	value, err := a.workspace.service.CreateEmpty(
		ctx,
		engine.EmptyWorkspaceRequest{
			DisplayName:    request.Body.DisplayName,
			Description:    request.Body.Description,
			TrustReference: request.Body.TrustReference,
			Discovery:      discoveryPreferencesOf(request.Body.Discovery),
		},
	)
	if err != nil {
		return nil, err
	}
	view, err := workspaceViewOf(value)
	if err != nil {
		return nil, err
	}
	return &CreateEmptyWorkspaceResponse{Body: &view}, nil
}

func (a *API) GetWorkspace(
	ctx context.Context,
	request *GetWorkspaceRequest,
) (*GetWorkspaceResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace request is required")
	}
	value, err := a.workspace.service.Get(ctx, request.RootID)
	if err != nil {
		return nil, err
	}
	view, err := workspaceViewOf(value)
	if err != nil {
		return nil, err
	}
	return &GetWorkspaceResponse{Body: &view}, nil
}

func (a *API) ListWorkspaces(
	ctx context.Context,
	_ *ListWorkspacesRequest,
) (*ListWorkspacesResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	values, err := a.workspace.service.List(ctx)
	if err != nil {
		return nil, err
	}
	output := make([]WorkspaceView, 0, len(values))
	for _, value := range values {
		view, err := workspaceViewOf(value)
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
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace update body is required")
	}
	value, err := a.workspace.service.Update(ctx, engine.UpdateRequest{
		RootID:           request.RootID,
		ExpectedRevision: request.Body.ExpectedRevision,
		DisplayName:      request.Body.DisplayName,
		Description:      request.Body.Description,
		Enabled:          request.Body.Enabled,
		TrustReference:   request.Body.TrustReference,
		Discovery:        discoveryPreferencesOf(request.Body.Discovery),
	})
	if err != nil {
		return nil, err
	}
	view, err := workspaceViewOf(value)
	if err != nil {
		return nil, err
	}
	return &UpdateWorkspaceResponse{Body: &view}, nil
}

func (a *API) DeleteWorkspace(
	ctx context.Context,
	request *DeleteWorkspaceRequest,
) (*DeleteWorkspaceResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace delete request is required")
	}
	value, err := a.workspace.service.Delete(
		ctx,
		request.RootID,
		request.ExpectedRevision,
	)
	if err != nil {
		return nil, err
	}
	return &DeleteWorkspaceResponse{
		Body: &DeleteWorkspaceResponseBody{
			RootID:   value.ID,
			Revision: value.Revision,
		},
	}, nil
}

func (a *API) AttachWorkspaceSource(
	ctx context.Context,
	request *AttachWorkspaceSourceRequest,
) (*AttachWorkspaceSourceResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace attachment body is required")
	}
	value, err := a.workspace.service.Attach(ctx, engine.AttachRequest{
		RootID:               request.RootID,
		ExpectedRootRevision: request.Body.ExpectedRootRevision,
		SourceID:             request.Body.SourceID,
		Role:                 request.Body.Role,
		Priority:             request.Body.Priority,
		Enabled:              request.Body.Enabled,
		Data:                 attachmentDataOf(request.Body.Settings),
	})
	if err != nil {
		return nil, err
	}
	view, err := workspaceViewOf(value)
	if err != nil {
		return nil, err
	}
	return &AttachWorkspaceSourceResponse{Body: &view}, nil
}

func (a *API) UpdateWorkspaceAttachment(
	ctx context.Context,
	request *UpdateWorkspaceAttachmentRequest,
) (*UpdateWorkspaceAttachmentResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace attachment update body is required")
	}
	value, err := a.workspace.service.UpdateAttachment(
		ctx,
		engine.UpdateAttachmentRequest{
			RootID:                     request.RootID,
			SourceID:                   request.SourceID,
			ExpectedRootRevision:       request.Body.ExpectedRootRevision,
			ExpectedAttachmentRevision: request.Body.ExpectedAttachmentRevision,
			Role:                       request.Body.Role,
			Priority:                   request.Body.Priority,
			Enabled:                    request.Body.Enabled,
			Data:                       attachmentDataOf(request.Body.Settings),
		},
	)
	if err != nil {
		return nil, err
	}
	view, err := workspaceViewOf(value)
	if err != nil {
		return nil, err
	}
	return &UpdateWorkspaceAttachmentResponse{Body: &view}, nil
}

func (a *API) DetachWorkspaceSource(
	ctx context.Context,
	request *DetachWorkspaceSourceRequest,
) (*DetachWorkspaceSourceResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace detach request is required")
	}
	value, err := a.workspace.service.Detach(
		ctx,
		request.RootID,
		request.SourceID,
		request.ExpectedRootRevision,
		request.ExpectedAttachmentRevision,
	)
	if err != nil {
		return nil, err
	}
	view, err := workspaceViewOf(value)
	if err != nil {
		return nil, err
	}
	return &DetachWorkspaceSourceResponse{Body: &view}, nil
}

func (a *API) RefreshWorkspace(
	ctx context.Context,
	request *RefreshWorkspaceRequest,
) (*RefreshWorkspaceResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace refresh request is required")
	}
	value, err := a.workspace.refresher.Refresh(ctx, request.RootID)
	if err != nil {
		return nil, err
	}
	output := WorkspaceRefreshResult{
		RootID:          value.Catalog.RootID,
		CatalogRevision: value.Catalog.Revision,
		CreatedRecords:  append([]artifactstore.RecordID(nil), value.CreatedRecords...),
		UpdatedRecords:  append([]artifactstore.RecordID(nil), value.UpdatedRecords...),
		Diagnostics:     artifactstore.CloneDiagnostics(value.Diagnostics),
		Candidates:      value.Candidates,
	}
	return &RefreshWorkspaceResponse{Body: &output}, nil
}

func (a *API) GetWorkspaceCatalog(
	ctx context.Context,
	request *GetWorkspaceCatalogRequest,
) (*GetWorkspaceCatalogResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace catalog request is required")
	}
	value, err := a.workspace.query.Catalog(ctx, request.RootID)
	if err != nil {
		return nil, err
	}
	output, err := workspaceCatalogViewOf(value)
	if err != nil {
		return nil, err
	}
	return &GetWorkspaceCatalogResponse{Body: &output}, nil
}

func (a *API) GetWorkspaceRecord(
	ctx context.Context,
	request *GetWorkspaceRecordRequest,
) (*GetWorkspaceRecordResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace record request is required")
	}
	value, err := a.workspaceRecord(ctx, request.RootID, request.RecordID)
	if err != nil {
		return nil, err
	}
	output := workspaceRecordViewOf(value)
	return &GetWorkspaceRecordResponse{Body: &output}, nil
}

func (a *API) ListWorkspaceContexts(
	ctx context.Context,
	request *ListWorkspaceContextsRequest,
) (*ListWorkspaceContextsResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace Context list request is required")
	}
	values, err := a.workspace.contextAdapter.List(ctx, request.RootID)
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
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace Context load body is required")
	}
	value, err := a.workspace.contextAdapter.Load(
		ctx,
		request.RootID,
		request.Body.RecordIDs,
	)
	if err != nil {
		return nil, err
	}
	output := WorkspaceContextInspectionView{
		RootID:          value.RootID,
		CatalogRevision: value.CatalogRevision,
		Diagnostics:     artifactstore.CloneDiagnostics(value.Diagnostics),
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
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace context body is required")
	}
	value, err := a.workspace.contextAdapter.Compose(
		ctx,
		request.RootID,
		request.Body.RecordIDs,
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
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace skill list request is required")
	}
	values, err := a.workspace.skillAdapter.List(ctx, request.RootID)
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
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace skill load body is required")
	}
	value, err := a.workspace.skillAdapter.Load(
		ctx,
		request.RootID,
		request.Body.RecordIDs,
	)
	if err != nil {
		return nil, err
	}
	output := workspaceSkillLoadViewOf(value)
	return &LoadWorkspaceSkillsResponse{Body: &output}, nil
}

func (a *API) SetWorkspaceRecordEnabled(
	ctx context.Context,
	request *SetWorkspaceRecordEnabledRequest,
) (*SetWorkspaceRecordEnabledResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace record update body is required")
	}
	if _, err := a.workspaceRecord(ctx, request.RootID, request.RecordID); err != nil {
		return nil, err
	}
	value, err := a.artifacts.Records.SetEnabled(
		ctx,
		request.RecordID,
		request.Body.ExpectedRevision,
		request.Body.Enabled,
	)
	if err != nil {
		return nil, err
	}
	output := workspaceRecordViewOf(value)
	return &SetWorkspaceRecordEnabledResponse{Body: &output}, nil
}

func (a *API) PinWorkspaceRecord(
	ctx context.Context,
	request *PinWorkspaceRecordRequest,
) (*PinWorkspaceRecordResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace record pin body is required")
	}
	if _, err := a.workspaceRecord(ctx, request.RootID, request.RecordID); err != nil {
		return nil, err
	}
	value, err := a.artifacts.Records.Pin(
		ctx,
		request.RecordID,
		request.Body.ExpectedRevision,
		request.Body.DefinitionDigest,
	)
	if err != nil {
		return nil, err
	}
	output := workspaceRecordViewOf(value)
	return &PinWorkspaceRecordResponse{Body: &output}, nil
}

func (a *API) FollowWorkspaceRecord(
	ctx context.Context,
	request *FollowWorkspaceRecordRequest,
) (*FollowWorkspaceRecordResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace record follow body is required")
	}
	if _, err := a.workspaceRecord(ctx, request.RootID, request.RecordID); err != nil {
		return nil, err
	}
	value, err := a.artifacts.Records.Follow(
		ctx,
		request.RecordID,
		request.Body.ExpectedRevision,
	)
	if err != nil {
		return nil, err
	}
	output := workspaceRecordViewOf(value)
	return &FollowWorkspaceRecordResponse{Body: &output}, nil
}

func (a *API) DeleteWorkspaceRecord(
	ctx context.Context,
	request *DeleteWorkspaceRecordRequest,
) (*DeleteWorkspaceRecordResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest("workspace record delete request is required")
	}
	if _, err := a.workspaceRecord(ctx, request.RootID, request.RecordID); err != nil {
		return nil, err
	}
	if err := a.artifacts.Records.Delete(
		ctx,
		request.RecordID,
		request.ExpectedRevision,
	); err != nil {
		return nil, err
	}
	return &DeleteWorkspaceRecordResponse{
		Body: &DeleteWorkspaceRecordResponseBody{
			RecordID: request.RecordID,
		},
	}, nil
}

func (a *API) UpdateWorkspaceRecordData(
	ctx context.Context,
	request *UpdateWorkspaceRecordDataRequest,
) (*UpdateWorkspaceRecordDataResponse, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("workspace record data body is required")
	}
	if _, err := a.workspaceRecord(ctx, request.RootID, request.RecordID); err != nil {
		return nil, err
	}
	data, err := engine.EncodeRecordData(engine.RecordData{
		RuntimeAllowed: request.Body.RuntimeAllowed,
	})
	if err != nil {
		return nil, err
	}
	value, err := a.artifacts.Records.UpdateData(
		ctx,
		request.RecordID,
		request.Body.ExpectedRevision,
		data,
	)
	if err != nil {
		return nil, err
	}
	output := workspaceRecordViewOf(value)
	return &UpdateWorkspaceRecordDataResponse{Body: &output}, nil
}

func (a *API) workspaceRecord(
	ctx context.Context,
	rootID artifactstore.RootID,
	recordID artifactstore.RecordID,
) (record.Record, error) {
	if _, err := a.workspace.service.Get(ctx, rootID); err != nil {
		return record.Record{}, err
	}
	value, err := a.artifacts.Records.Get(ctx, recordID)
	if err != nil {
		return record.Record{}, err
	}
	if value.RootID != rootID {
		return record.Record{}, fmt.Errorf(
			"%w: record %q does not belong to workspace %q",
			engine.ErrReferenceUnresolved,
			recordID,
			rootID,
		)
	}
	return value, nil
}

func (a *API) ready() error {
	if a == nil ||
		a.artifacts == nil ||
		a.workspace == nil ||
		a.provisioner == nil {
		return invalidAPIRequest("workspace API is not initialized")
	}
	return nil
}

func invalidAPIRequest(message string) error {
	return fmt.Errorf("%w: %s", engine.ErrInvalidWorkspace, message)
}

func workspaceViewOf(value engine.Workspace) (WorkspaceView, error) {
	output := WorkspaceView{
		RootID:            value.Root.ID,
		Revision:          value.Root.Revision,
		DisplayName:       value.Root.DisplayName,
		Description:       value.Root.Description,
		Enabled:           value.Root.Enabled,
		Mode:              string(value.Data.Mode),
		PrimarySourceID:   value.Data.PrimarySourceID,
		HasTrustReference: value.Data.TrustReference != "",
		Discovery:         workspaceDiscoveryOf(value.Data.Discovery),
		Attachments:       make([]WorkspaceAttachmentView, 0, len(value.Attachments)),
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
			Priority: attachment.Priority,
			Enabled:  attachment.Enabled,
			Settings: settings,
		})
	}
	return output, nil
}

func workspaceDiscoveryOf(value engine.DiscoveryPreferences) WorkspaceDiscovery {
	output := WorkspaceDiscovery{
		AdditionalLocators: append(
			[]artifactstore.Locator(nil),
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

func discoveryPreferencesOf(value WorkspaceDiscovery) engine.DiscoveryPreferences {
	output := engine.DiscoveryPreferences{
		AdditionalLocators: append(
			[]artifactstore.Locator(nil),
			value.AdditionalLocators...,
		),
		IncludeReadme: value.IncludeReadme,
	}
	for _, root := range value.AdditionalRoots {
		output.AdditionalRoots = append(output.AdditionalRoots, engine.DiscoveryRoot{
			Root:            root.Root,
			Recursive:       root.Recursive,
			IncludePatterns: append([]string(nil), root.IncludePatterns...),
		})
	}
	return output
}

func workspaceAttachmentSettingsOf(
	raw json.RawMessage,
) (WorkspaceAttachmentSettings, error) {
	var value engine.AttachmentData
	if err := json.Unmarshal(raw, &value); err != nil {
		return WorkspaceAttachmentSettings{}, fmt.Errorf(
			"%w: decode workspace attachment settings: %w",
			engine.ErrInvalidWorkspace,
			err,
		)
	}
	return WorkspaceAttachmentSettings{
		Recursive:     cloneBool(value.Recursive),
		Authoritative: cloneBool(value.Authoritative),
	}, nil
}

func attachmentDataOf(value WorkspaceAttachmentSettings) engine.AttachmentData {
	return engine.AttachmentData{
		Recursive:     cloneBool(value.Recursive),
		Authoritative: cloneBool(value.Authoritative),
	}
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func workspaceRecordViewOf(value record.Record) WorkspaceRecordView {
	var digest *artifactstore.Digest
	if value.ResolvedDefinition != nil {
		copyValue := *value.ResolvedDefinition
		digest = &copyValue
	}
	var pinned *artifactstore.Digest
	if value.PinnedDefinition != nil {
		copyValue := *value.PinnedDefinition
		pinned = &copyValue
	}
	runtimeAllowed, _ := engine.RecordRuntimeAllowed(value)
	return WorkspaceRecordView{
		ID:                 value.ID,
		Revision:           value.Revision,
		Name:               value.Name,
		Kind:               value.Kind,
		Enabled:            value.Enabled,
		State:              string(value.State),
		Mode:               string(value.Mode),
		PinnedDefinition:   pinned,
		ResolvedDefinition: digest,
		SourceID:           value.Occurrence.SourceID,
		Locator:            value.Occurrence.Locator,
		SubresourceLocator: value.Occurrence.SubresourceLocator,
		RuntimeAllowed:     runtimeAllowed,
		Diagnostics:        artifactstore.CloneDiagnostics(value.Diagnostics),
	}
}

func workspaceCatalogViewOf(
	value engine.CatalogView,
) (WorkspaceCatalogView, error) {
	workspaceValue, err := workspaceViewOf(value.Workspace)
	if err != nil {
		return WorkspaceCatalogView{}, err
	}
	output := WorkspaceCatalogView{
		Workspace:             workspaceValue,
		CatalogRevision:       value.Catalog.Revision,
		CatalogCurrent:        value.CatalogCurrent,
		Diagnostics:           artifactstore.CloneDiagnostics(value.Catalog.Diagnostics),
		Resources:             make([]WorkspaceResourceView, 0, len(value.Resources)),
		Groups:                make([]WorkspaceResourceGroupView, 0, len(value.Groups)),
		Occurrences:           make([]WorkspaceOccurrenceView, 0, len(value.Catalog.Occurrences)),
		UnrecordedCount:       len(value.Unrecorded),
		UnresolvedRecordCount: len(value.UnresolvedRecords),
	}
	recordsByOccurrence := make(map[string]record.Record, len(value.Resources))
	for _, resourceValue := range value.Resources {
		projected := WorkspaceResourceView{
			Record:           workspaceRecordViewOf(resourceValue.Record),
			DefinitionDigest: resourceValue.Definition.Digest,
			SourceID:         resourceValue.Source.ID,
			Locator:          resourceValue.Record.Occurrence.Locator,
			CatalogCurrent:   resourceValue.CatalogCurrent,
			ProjectionValid:  resourceValue.ProjectionValid,
			Diagnostics: artifactstore.AppendDiagnostics(
				resourceValue.Record.Diagnostics,
				resourceValue.Diagnostics...,
			),
		}
		output.Resources = append(output.Resources, projected)
		recordsByOccurrence[occurrenceViewKey(
			resourceValue.Record.Occurrence.SourceID,
			resourceValue.Record.Occurrence.Locator,
			resourceValue.Record.Occurrence.SubresourceLocator,
		)] = resourceValue.Record
	}
	for _, localRecord := range value.UnresolvedRecords {
		output.UnresolvedRecords = append(
			output.UnresolvedRecords,
			workspaceRecordViewOf(localRecord),
		)
		recordsByOccurrence[occurrenceViewKey(
			localRecord.Occurrence.SourceID,
			localRecord.Occurrence.Locator,
			localRecord.Occurrence.SubresourceLocator,
		)] = localRecord
	}
	for _, occurrence := range value.Catalog.Occurrences {
		projected := workspaceOccurrenceViewOf(
			occurrence,
			recordsByOccurrence,
		)
		output.Occurrences = append(output.Occurrences, projected)
		switch occurrence.State {
		case "valid":
			output.ValidOccurrences = append(output.ValidOccurrences, projected)
		case "invalid":
			output.InvalidOccurrences = append(output.InvalidOccurrences, projected)
		case "missing":
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
			projected.Resources = append(
				projected.Resources,
				WorkspaceResourceView{
					Record:           workspaceRecordViewOf(resourceValue.Record),
					DefinitionDigest: resourceValue.Definition.Digest,
					SourceID:         resourceValue.Source.ID,
					Locator:          resourceValue.Record.Occurrence.Locator,
					CatalogCurrent:   resourceValue.CatalogCurrent,
					ProjectionValid:  resourceValue.ProjectionValid,
					Diagnostics: artifactstore.AppendDiagnostics(
						resourceValue.Record.Diagnostics,
						resourceValue.Diagnostics...,
					),
				},
			)
		}
		for _, occurrence := range group.Unrecorded {
			projected.Unrecorded = append(
				projected.Unrecorded,
				workspaceOccurrenceViewOf(occurrence, recordsByOccurrence),
			)
		}
		output.Groups = append(output.Groups, projected)
	}
	return output, nil
}

func occurrenceViewKey(
	sourceID artifactstore.SourceID,
	locator artifactstore.Locator,
	subresource artifactstore.SubresourceLocator,
) string {
	return string(sourceID) + "\x00" +
		string(locator) + "\x00" +
		string(subresource)
}

func workspaceOccurrenceViewOf(
	value catalog.Occurrence,
	records map[string]record.Record,
) WorkspaceOccurrenceView {
	output := WorkspaceOccurrenceView{
		SourceID:            value.Key.SourceID,
		Locator:             value.Key.Locator,
		SubresourceLocator:  value.Key.SubresourceLocator,
		Kind:                value.Kind,
		LogicalName:         value.LogicalName,
		LogicalVersion:      value.LogicalVersion,
		DefinitionDigest:    cloneDigest(value.DefinitionDigest),
		SourceContentDigest: cloneDigest(value.SourceContentDigest),
		State:               string(value.State),
		Diagnostics:         artifactstore.CloneDiagnostics(value.Diagnostics),
	}
	if localRecord, found := records[occurrenceViewKey(
		value.Key.SourceID,
		value.Key.Locator,
		value.Key.SubresourceLocator,
	)]; found {
		recordID := localRecord.ID
		output.Recorded = true
		output.RecordID = &recordID
	}
	return output
}

func cloneDigest(value *artifactstore.Digest) *artifactstore.Digest {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func contextLoadPlanViewOf(
	value contextadapter.ContextLoadPlan,
) WorkspaceContextLoadPlan {
	output := WorkspaceContextLoadPlan{
		RootID:          value.RootID,
		CatalogRevision: value.CatalogRevision,
		Prompt:          value.Prompt,
		Diagnostics:     artifactstore.CloneDiagnostics(value.Diagnostics),
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
			RecordID:      decision.RecordID,
			Status:        string(decision.Status),
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
		RecordID:         value.RecordID,
		DefinitionDigest: value.DefinitionDigest,
		SourceID:         value.SourceID,
		Locator:          value.Locator,
		Priority:         value.Priority,
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
		RecordID:         value.RecordID,
		RecordRevision:   value.RecordRevision,
		DefinitionDigest: value.DefinitionDigest,
		SourceID:         value.SourceID,
		Locator:          value.Locator,
		Priority:         value.Priority,
		Name:             value.Name,
		Role:             value.Role,
		MediaType:        value.MediaType,
		Enabled:          value.Enabled,
		State:            string(value.State),
		CatalogCurrent:   value.CatalogCurrent,
		RuntimeAllowed:   value.RuntimeAllowed,
		Diagnostics:      artifactstore.CloneDiagnostics(value.Diagnostics),
	}
}

func workspaceSkillViewOf(value skilladapter.WorkspaceSkill) WorkspaceSkillView {
	summary := WorkspaceSkillSummary{
		SchemaVersion: value.Skill.SchemaVersion,
		ID:            value.Skill.ID,
		Slug:          value.Skill.Slug,
		Name:          value.Skill.Name,
		DisplayName:   value.Skill.DisplayName,
		Description:   value.Skill.Description,
		Tags:          append([]string(nil), value.Skill.Tags...),
		Insert:        value.Skill.Insert,
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
		RootID:           value.RootID,
		RecordID:         value.RecordID,
		DefinitionDigest: value.DefinitionDigest,
		SourceID:         value.SourceID,
		Locator:          value.Locator,
		Skill:            summary,
		MarkdownBody:     value.MarkdownBody,
		Priority:         value.Priority,
		RecordRevision:   value.RecordRevision,
		State:            string(value.State),
		CatalogCurrent:   value.CatalogCurrent,
		RuntimeAllowed:   value.RuntimeAllowed,
		Diagnostics:      artifactstore.CloneDiagnostics(value.Diagnostics),
	}
}

func workspaceSkillLoadViewOf(
	value skilladapter.SkillLoadPlan,
) WorkspaceSkillLoadView {
	output := WorkspaceSkillLoadView{
		RootID:          value.RootID,
		CatalogRevision: value.CatalogRevision,
		Diagnostics:     artifactstore.CloneDiagnostics(value.Diagnostics),
		Skills:          make([]WorkspaceSkillView, 0, len(value.Skills)),
	}
	for _, skill := range value.Skills {
		output.Skills = append(output.Skills, workspaceSkillViewOf(skill))
	}
	return output
}
