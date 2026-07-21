package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
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

// WorkspaceView is the API-safe workspace representation. It deliberately
// excludes source configuration and raw catalog persistence data.
type WorkspaceView struct {
	RootID            artifactstore.RootID      `json:"rootID"`
	Revision          uint64                    `json:"revision"`
	DisplayName       string                    `json:"displayName"`
	Description       string                    `json:"description,omitempty"`
	Enabled           bool                      `json:"enabled"`
	Mode              string                    `json:"mode"`
	PrimarySourceID   artifactstore.SourceID    `json:"primarySourceID,omitempty"`
	HasTrustReference bool                      `json:"hasTrustReference"`
	Discovery         WorkspaceDiscoveryView    `json:"discovery"`
	Attachments       []WorkspaceAttachmentView `json:"attachments"`
}

type WorkspaceDiscoveryView struct {
	AdditionalLocators []artifactstore.Locator      `json:"additionalLocators,omitempty"`
	AdditionalRoots    []WorkspaceDiscoveryRootView `json:"additionalRoots,omitempty"`
	IncludeReadme      bool                         `json:"includeReadme,omitempty"`
}

type WorkspaceDiscoveryRootView struct {
	Root            artifactstore.Locator `json:"root"`
	Recursive       bool                  `json:"recursive"`
	IncludePatterns []string              `json:"includePatterns,omitempty"`
}

type WorkspaceAttachmentView struct {
	SourceID artifactstore.SourceID       `json:"sourceID"`
	Role     artifactstore.AttachmentRole `json:"role"`
	Priority int                          `json:"priority"`
	Enabled  bool                         `json:"enabled"`
}

type WorkspaceRecordView struct {
	ID                 artifactstore.RecordID     `json:"id"`
	Revision           uint64                     `json:"revision"`
	Name               string                     `json:"name"`
	Kind               artifactstore.ArtifactKind `json:"kind"`
	Enabled            bool                       `json:"enabled"`
	State              string                     `json:"state"`
	ResolvedDefinition *artifactstore.Digest      `json:"resolvedDefinition,omitempty"`
}

type WorkspaceResourceView struct {
	Record           WorkspaceRecordView    `json:"record"`
	DefinitionDigest artifactstore.Digest   `json:"definitionDigest"`
	SourceID         artifactstore.SourceID `json:"sourceID"`
	Locator          artifactstore.Locator  `json:"locator"`
	CatalogCurrent   bool                   `json:"catalogCurrent"`
}

type WorkspaceCatalogView struct {
	Workspace             WorkspaceView           `json:"workspace"`
	CatalogRevision       uint64                  `json:"catalogRevision"`
	Resources             []WorkspaceResourceView `json:"resources"`
	UnrecordedCount       int                     `json:"unrecordedCount"`
	UnresolvedRecordCount int                     `json:"unresolvedRecordCount"`
}

type WorkspaceDeleteResult struct {
	RootID   artifactstore.RootID `json:"rootID"`
	Revision uint64               `json:"revision"`
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

type WorkspaceWrapper struct {
	artifacts   *system.Components
	workspace   *workspace.Components
	provisioner *provision.Service
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
) (*WorkspaceView, error) {
	return middleware.WithRecoveryResp(func() (*WorkspaceView, error) {
		value, err := w.provisioner.CreateFilesystem(
			context.Background(),
			*request,
		)
		if err != nil {
			return nil, err
		}
		output := workspaceViewOf(value)
		return &output, nil
	})
}

func (w *WorkspaceWrapper) CreateEmpty(
	request *engine.EmptyWorkspaceRequest,
) (*WorkspaceView, error) {
	return middleware.WithRecoveryResp(func() (*WorkspaceView, error) {
		value, err := w.workspace.Service.CreateEmpty(
			context.Background(),
			*request,
		)
		if err != nil {
			return nil, err
		}
		output := workspaceViewOf(value)
		return &output, nil
	})
}

func (w *WorkspaceWrapper) Get(
	rootID artifactstore.RootID,
) (*WorkspaceView, error) {
	return middleware.WithRecoveryResp(func() (*WorkspaceView, error) {
		value, err := w.workspace.Service.Get(context.Background(), rootID)
		if err != nil {
			return nil, err
		}
		output := workspaceViewOf(value)
		return &output, nil
	})
}

func (w *WorkspaceWrapper) List() ([]WorkspaceView, error) {
	return middleware.WithRecoveryResp(func() ([]WorkspaceView, error) {
		values, err := w.workspace.Service.List(context.Background())
		if err != nil {
			return nil, err
		}
		output := make([]WorkspaceView, 0, len(values))
		for _, value := range values {
			output = append(output, workspaceViewOf(value))
		}
		return output, nil
	})
}

func (w *WorkspaceWrapper) Update(
	request *engine.UpdateRequest,
) (*WorkspaceView, error) {
	return middleware.WithRecoveryResp(func() (*WorkspaceView, error) {
		value, err := w.workspace.Service.Update(
			context.Background(),
			*request,
		)
		if err != nil {
			return nil, err
		}
		output := workspaceViewOf(value)
		return &output, nil
	})
}

func (w *WorkspaceWrapper) Delete(
	request *WorkspaceDeleteRequest,
) (*WorkspaceDeleteResult, error) {
	return middleware.WithRecoveryResp(func() (*WorkspaceDeleteResult, error) {
		value, err := w.workspace.Service.Delete(
			context.Background(),
			request.RootID,
			request.ExpectedRevision,
		)
		if err != nil {
			return nil, err
		}
		return &WorkspaceDeleteResult{
			RootID:   value.ID,
			Revision: value.Revision,
		}, nil
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
) (*WorkspaceCatalogView, error) {
	return middleware.WithRecoveryResp(func() (*WorkspaceCatalogView, error) {
		value, err := w.workspace.Query.Catalog(
			context.Background(),
			rootID,
		)
		if err != nil {
			return nil, err
		}
		output := workspaceCatalogViewOf(value)
		return &output, nil
	})
}

func (w *WorkspaceWrapper) ComposeContext(
	request *WorkspaceContextRequest,
) (*contextadapter.ContextLoadPlan, error) {
	return middleware.WithRecoveryResp(func() (*contextadapter.ContextLoadPlan, error) {
		value, err := w.workspace.ContextAdapter.Compose(
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
		return w.workspace.SkillAdapter.List(context.Background(), rootID)
	})
}

func (w *WorkspaceWrapper) LoadWorkspaceSkills(
	request *WorkspaceSkillLoadRequest,
) (*skilladapter.SkillLoadPlan, error) {
	return middleware.WithRecoveryResp(func() (*skilladapter.SkillLoadPlan, error) {
		value, err := w.workspace.SkillAdapter.Load(
			context.Background(),
			request.RootID,
			request.RecordIDs,
		)
		return &value, err
	})
}

func (w *WorkspaceWrapper) SetRecordEnabled(
	request *WorkspaceRecordEnabledRequest,
) (*WorkspaceRecordView, error) {
	return middleware.WithRecoveryResp(func() (*WorkspaceRecordView, error) {
		value, err := w.artifacts.Records.SetEnabled(
			context.Background(),
			request.RecordID,
			request.ExpectedRevision,
			request.Enabled,
		)
		if err != nil {
			return nil, err
		}
		output := workspaceRecordViewOf(value)
		return &output, nil
	})
}

func (w *WorkspaceWrapper) close() {
	if w == nil || w.artifacts == nil {
		return
	}
	_ = w.artifacts.Close()
}

func workspaceViewOf(value engine.Workspace) WorkspaceView {
	attachments := make(
		[]WorkspaceAttachmentView,
		0,
		len(value.Attachments),
	)
	for _, attachment := range value.Attachments {
		attachments = append(attachments, WorkspaceAttachmentView{
			SourceID: attachment.SourceID,
			Role:     attachment.Role,
			Priority: attachment.Priority,
			Enabled:  attachment.Enabled,
		})
	}
	return WorkspaceView{
		RootID:            value.Root.ID,
		Revision:          value.Root.Revision,
		DisplayName:       value.Root.DisplayName,
		Description:       value.Root.Description,
		Enabled:           value.Root.Enabled,
		Mode:              string(value.Data.Mode),
		PrimarySourceID:   value.Data.PrimarySourceID,
		HasTrustReference: value.Data.TrustReference != "",
		Discovery:         workspaceDiscoveryViewOf(value.Data.Discovery),
		Attachments:       attachments,
	}
}

func workspaceDiscoveryViewOf(
	value engine.DiscoveryPreferences,
) WorkspaceDiscoveryView {
	output := WorkspaceDiscoveryView{
		AdditionalLocators: append(
			[]artifactstore.Locator(nil),
			value.AdditionalLocators...,
		),
		IncludeReadme: value.IncludeReadme,
	}
	for _, root := range value.AdditionalRoots {
		output.AdditionalRoots = append(
			output.AdditionalRoots,
			WorkspaceDiscoveryRootView{
				Root:            root.Root,
				Recursive:       root.Recursive,
				IncludePatterns: append([]string(nil), root.IncludePatterns...),
			},
		)
	}
	return output
}

func workspaceRecordViewOf(value record.Record) WorkspaceRecordView {
	var digest *artifactstore.Digest
	if value.ResolvedDefinition != nil {
		copied := *value.ResolvedDefinition
		digest = &copied
	}
	return WorkspaceRecordView{
		ID:                 value.ID,
		Revision:           value.Revision,
		Name:               value.Name,
		Kind:               value.Kind,
		Enabled:            value.Enabled,
		State:              string(value.State),
		ResolvedDefinition: digest,
	}
}

func workspaceCatalogViewOf(value engine.CatalogView) WorkspaceCatalogView {
	output := WorkspaceCatalogView{
		Workspace:             workspaceViewOf(value.Workspace),
		CatalogRevision:       value.Catalog.Revision,
		Resources:             make([]WorkspaceResourceView, 0, len(value.Resources)),
		UnrecordedCount:       len(value.Unrecorded),
		UnresolvedRecordCount: len(value.UnresolvedRecords),
	}
	for _, resource := range value.Resources {
		output.Resources = append(output.Resources, WorkspaceResourceView{
			Record:           workspaceRecordViewOf(resource.Record),
			DefinitionDigest: resource.Definition.Digest,
			SourceID:         resource.Source.ID,
			Locator:          resource.Record.Occurrence.Locator,
			CatalogCurrent:   resource.CatalogCurrent,
		})
	}
	return output
}
