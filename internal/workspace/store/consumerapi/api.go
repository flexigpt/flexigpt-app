package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/prompt"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/skill"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
	workspaceProviderAPI "github.com/flexigpt/flexigpt-app/internal/workspace/store/providerapi"
)

type StoreAPI struct {
	sources   compositionapi.SourceAPI
	discovery compositionapi.DiscoveryAPI
	artifacts compositionapi.ArtifactAPI
	resources compositionapi.ResourceAPI

	config         Config
	contextAdapter *prompt.Adapter
	skillAdapter   *skill.Adapter
	resolver       *resolve.Resolver
}

func NewStoreAPI(
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
	config Config,
) (*StoreAPI, error) {
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		resources == nil {
		return nil, fmt.Errorf(
			"%w: Workspace Store dependencies are incomplete",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	config = config.normalized()
	if err := config.ContextComposition.Validate(); err != nil {
		return nil, err
	}

	output := &StoreAPI{
		sources:   sources,
		discovery: discovery,
		artifacts: artifacts,
		resources: resources,
		config:    config,
	}

	contextAdapter, err := prompt.New(
		artifacts,
		resources,
		config.ContextComposition,
	)
	if err != nil {
		return nil, err
	}
	skillAdapter, err := skill.New(artifacts, resources)
	if err != nil {
		return nil, err
	}

	locators, err := resolve.NewProviderLocatorResolver(
		config.LocatorResolvers,
		output,
	)
	if err != nil {
		return nil, err
	}
	resolver, err := resolve.New(
		artifacts,
		locators,
		config.ResolverLimits,
		config.ResolverOptions,
	)
	if err != nil {
		return nil, err
	}

	output.contextAdapter = contextAdapter
	output.skillAdapter = skillAdapter
	output.resolver = resolver
	return output, nil
}

func (a *StoreAPI) RegisterFilesystemSource(
	ctx context.Context,
	request FilesystemSourceRegistration,
) (source.Summary, error) {
	if err := request.RootID.Validate(); err != nil {
		return source.Summary{}, err
	}
	if err := basespec.ValidateRequiredText(
		"Workspace Source display name",
		request.SourceDisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return source.Summary{}, err
	}

	discovery, err := a.defaultDiscovery()
	if err != nil {
		return source.Summary{}, err
	}
	config, err := json.Marshal(struct {
		RootPath string `json:"rootPath"`
	}{
		RootPath: request.RootPath,
	})
	if err != nil {
		return source.Summary{}, err
	}

	value, err := a.sources.Create(
		ctx,
		request.RootID,
		source.Draft{
			ID:          source.SourceID(uuidutil.NewUUIDv7()),
			StorageKey:  workspaceSourceStorageKey(),
			Kind:        source.SourceKindFilesystemDirectory,
			DisplayName: request.SourceDisplayName,
			Enabled:     true,
			Config:      config,
			Discovery:   discovery,
		},
	)
	if err != nil {
		return source.Summary{}, err
	}
	if _, err := a.discovery.RefreshSource(
		ctx,
		request.RootID,
		value.ID,
	); err != nil {
		return source.Summary{}, err
	}
	return value, nil
}

func (a *StoreAPI) GetWorkspace(
	ctx context.Context,
	ref WorkspaceRef,
) (workspaceDomain.Workspace, error) {
	if err := ref.Validate(); err != nil {
		return workspaceDomain.Workspace{}, err
	}
	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}
	if record.Kind != workspaceDomain.WorkspaceArtifactKind {
		return workspaceDomain.Workspace{}, fmt.Errorf(
			"%w: Artifact %q has kind %q",
			workspaceDomain.ErrNotWorkspace,
			record.ID,
			record.Kind,
		)
	}
	value, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}
	return workspaceDomain.NewWorkspace(record, value)
}

func (a *StoreAPI) ListWorkspaces(
	ctx context.Context,
	rootID root.RootID,
) ([]workspaceDomain.Workspace, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	values, err := a.artifacts.ListByRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}
	output := make([]workspaceDomain.Workspace, 0)
	for _, value := range values {
		if value.Kind != workspaceDomain.WorkspaceArtifactKind {
			continue
		}
		workspace, err := a.GetWorkspace(ctx, value.Ref())
		if err != nil {
			return nil, err
		}
		output = append(output, workspace)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].Definition.LogicalName !=
			output[right].Definition.LogicalName {
			return output[left].Definition.LogicalName <
				output[right].Definition.LogicalName
		}
		return output[left].Artifact.ID < output[right].Artifact.ID
	})
	return output, nil
}

func (a *StoreAPI) LoadWorkspace(
	ctx context.Context,
	ref WorkspaceRef,
) (WorkspaceLoad, error) {
	workspace, _, resolved, err := a.refreshAndResolveWorkspace(ctx, ref)
	if err != nil {
		return WorkspaceLoad{}, err
	}
	roots := make(
		[]declaration.Entry,
		len(workspace.Document.Roots),
	)
	for index, value := range workspace.Document.Roots {
		roots[index] = value.Clone()
	}
	return WorkspaceLoad{
		Workspace: workspace,
		Roots:     roots,
		resolved:  resolved.Root.Workspace,
	}, nil
}

func (a *StoreAPI) RefreshWorkspace(
	ctx context.Context,
	ref WorkspaceRef,
) (WorkspaceRefresh, error) {
	workspace, result, _, err := a.refreshAndResolveWorkspace(ctx, ref)
	if err != nil {
		return WorkspaceRefresh{}, err
	}
	return WorkspaceRefresh{
		Workspace: workspace.Ref(),
		Result:    result,
	}, nil
}

func (a *StoreAPI) ListWorkspaceArtifacts(
	ctx context.Context,
	workspace WorkspaceRef,
) ([]artifact.Artifact, error) {
	value, err := a.GetWorkspace(ctx, workspace)
	if err != nil {
		return nil, err
	}
	artifacts, err := a.artifacts.ListByRoot(
		ctx,
		value.Artifact.RootID,
	)
	if err != nil {
		return nil, err
	}
	output := make([]artifact.Artifact, len(artifacts))
	for index, value := range artifacts {
		output[index] = value.Clone()
	}
	return output, nil
}

func (a *StoreAPI) SetWorkspaceArtifactEnabled(
	ctx context.Context,
	workspace WorkspaceRef,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (WorkspaceArtifactView, error) {
	value, err := a.GetWorkspace(ctx, workspace)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	if ref.RootID != value.Artifact.RootID {
		return WorkspaceArtifactView{}, fmt.Errorf(
			"%w: Artifact belongs to another Root",
			workspaceDomain.ErrReferenceUnresolved,
		)
	}

	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	if record.Revision != expectedRevision {
		return WorkspaceArtifactView{}, basespec.ErrConflict
	}

	updated, err := a.artifacts.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	return workspaceArtifactViewOf(updated)
}

func (a *StoreAPI) SetArtifactRuntimeDisabled(
	ctx context.Context,
	workspace WorkspaceRef,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	runtimeDisabled bool,
) (WorkspaceArtifactView, error) {
	value, err := a.GetWorkspace(ctx, workspace)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	if ref.RootID != value.Artifact.RootID {
		return WorkspaceArtifactView{}, fmt.Errorf(
			"%w: Artifact belongs to another Root",
			workspaceDomain.ErrReferenceUnresolved,
		)
	}

	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	if record.Revision != expectedRevision {
		return WorkspaceArtifactView{}, basespec.ErrConflict
	}

	data, err := workspaceDomain.DecodeArtifactData(record.Data)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	data.RuntimeDisabled = runtimeDisabled
	raw, err := workspaceDomain.MergeArtifactData(
		record.Data,
		data,
	)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	updated, err := a.artifacts.UpdateData(
		ctx,
		ref,
		expectedRevision,
		raw,
	)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	return workspaceArtifactViewOf(updated)
}

func (a *StoreAPI) ContextAdapter() *prompt.Adapter {
	if a == nil {
		return nil
	}
	return a.contextAdapter
}

func (a *StoreAPI) SkillAdapter() *skill.Adapter {
	if a == nil {
		return nil
	}
	return a.skillAdapter
}

func (a *StoreAPI) defaultDiscovery() (
	source.DiscoverySpec,
	error,
) {
	value := source.DiscoverySpec{
		DirectoryRoots: []source.DirectoryRoot{
			{
				Root:      ".",
				Recursive: true,
				IncludePatterns: []string{
					"**/*.json",
					"**/*.yaml",
					"**/*.yml",
					"**/SKILL.md",
					"**/AGENTS.md",
					"**/AGENT.md",
					"**/*.agent.md",
					"**/CLAUDE.md",
					"**/README.md",
					"**/llms.txt",
					"docs/**/*.md",
				},
			},
		},
		DecoderHints: []source.DecoderHint{{
			Locator:   "docs",
			Recursive: true,
			DecoderIDs: []basespec.DecoderID{
				workspaceProviderAPI.ContextMarkdownDecoderID,
			},
		}},
		Authoritative: true,
	}
	for _, hint := range a.config.AdditionalDecoderHints {
		value.DecoderHints = appendDecoderHint(
			value.DecoderHints,
			hint,
		)
	}
	value = value.Normalized()
	if err := value.Validate(); err != nil {
		return source.DiscoverySpec{}, err
	}
	return value, nil
}

func workspaceArtifactViewOf(
	value artifact.Artifact,
) (WorkspaceArtifactView, error) {
	data, err := workspaceDomain.DecodeArtifactData(value.Data)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	return WorkspaceArtifactView{
		Artifact:           value.Ref(),
		Revision:           value.Revision,
		DisplayName:        value.DisplayName,
		Kind:               value.Kind,
		LogicalName:        value.LogicalName,
		LogicalVersion:     value.LogicalVersion,
		Enabled:            value.Enabled,
		State:              value.State,
		SourceID:           value.Binding.SourceID,
		Locator:            value.Binding.Locator,
		SubresourceLocator: value.Binding.SubresourceLocator,
		RuntimeDisabled:    data.RuntimeDisabled,
	}, nil
}

func workspaceSourceStorageKey() basespec.StorageKey {
	return basespec.StorageKey(
		"workspace-" + strings.ReplaceAll(
			uuidutil.NewUUIDv7(),
			"-",
			"",
		),
	)
}
