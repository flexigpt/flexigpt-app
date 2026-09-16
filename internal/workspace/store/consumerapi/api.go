package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/format/markdown"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/mcp"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/prompt"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/skill"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

type StoreAPI struct {
	sources   compositionapi.SourceAPI
	discovery compositionapi.DiscoveryAPI
	artifacts compositionapi.ArtifactAPI
	resources compositionapi.ResourceAPI

	config        Config
	promptAdapter *prompt.Adapter
	skillAdapter  *skill.Adapter
	mcpAdapter    *mcp.Adapter
	resolver      *resolve.Resolver
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

	promptAdapter, err := prompt.New(
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

	var mcpAdapter *mcp.Adapter
	if config.MCPServers != nil {
		mcpAdapter, err = mcp.New(
			artifacts,
			config.MCPServers,
		)
		if err != nil {
			return nil, err
		}
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

	output.promptAdapter = promptAdapter
	output.skillAdapter = skillAdapter
	output.mcpAdapter = mcpAdapter
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

	rootPath, err := normalizeWorkspaceSourceRoot(request.RootPath)
	if err != nil {
		return source.Summary{}, err
	}

	discovery, err := a.defaultDiscovery()
	if err != nil {
		return source.Summary{}, err
	}
	config, err := json.Marshal(struct {
		RootPath string `json:"rootPath"`
	}{
		RootPath: rootPath,
	})
	if err != nil {
		return source.Summary{}, err
	}

	value, _, err := a.sources.Ensure(
		ctx,
		request.RootID,
		source.Draft{
			ID:          source.SourceID(uuidutil.NewUUIDv7()),
			StorageKey:  workspacePathStorageKey(rootPath),
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
	if !value.Enabled ||
		value.DisplayName != request.SourceDisplayName ||
		!value.Discovery.Equal(discovery) {
		value, err = a.sources.Update(
			ctx,
			request.RootID,
			value.ID,
			source.Update{
				ExpectedRevision: value.Revision,
				DisplayName:      request.SourceDisplayName,
				Enabled:          true,
				Discovery:        &discovery,
			},
		)
		if err != nil {
			return source.Summary{}, err
		}
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
	return a.workspaceForRef(ctx, ref)
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
	seen := make(map[artifact.ArtifactRef]struct{})
	for _, value := range values {
		if value.Kind != workspaceDomain.WorkspaceArtifactKind ||
			value.State != artifact.StateAvailable ||
			value.ResolvedDefinition == nil {
			continue
		}
		workspace, err := a.GetWorkspace(ctx, value.Ref())
		if err != nil {
			return nil, err
		}
		if _, duplicate := seen[workspace.Ref()]; duplicate {
			continue
		}
		seen[workspace.Ref()] = struct{}{}
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
	workspace, resolved, err := a.resolveCurrentWorkspace(ctx, ref)
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
	workspace, result, err := a.refreshWorkspace(ctx, ref)
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
	sort.Slice(output, func(left, right int) bool {
		leftKey := string(output[left].Kind) + "\x00" +
			string(output[left].LogicalName) + "\x00" +
			string(output[left].Binding.SourceID) + "\x00" +
			string(output[left].Binding.Locator) + "\x00" +
			string(output[left].ID)
		rightKey := string(output[right].Kind) + "\x00" +
			string(output[right].LogicalName) + "\x00" +
			string(output[right].Binding.SourceID) + "\x00" +
			string(output[right].Binding.Locator) + "\x00" +
			string(output[right].ID)
		return leftKey < rightKey
	})
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
	if collection.IsBaselineCollectionArtifact(record) {
		return WorkspaceArtifactView{}, fmt.Errorf(
			"%w: baseline Collections cannot be disabled",
			basespec.ErrProtected,
		)
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
	if collection.IsBaselineCollectionArtifact(record) {
		return WorkspaceArtifactView{}, fmt.Errorf(
			"%w: baseline Collections cannot be runtime-disabled",
			basespec.ErrProtected,
		)
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

func (a *StoreAPI) PromptAdapter() *prompt.Adapter {
	if a == nil {
		return nil
	}
	return a.promptAdapter
}

func (a *StoreAPI) SkillAdapter() *skill.Adapter {
	if a == nil {
		return nil
	}
	return a.skillAdapter
}

// defaultDiscovery bootstraps Workspace identification. An explicitly
// present Workspace declarations array is reconciled later by the loader.
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
					"**/docs/**/*.md",
				},
			},
		},
		Authoritative: true,
	}
	value.DecoderHints = append(value.DecoderHints, source.DecoderHint{
		Locator:   ".",
		Recursive: true,
		DecoderIDs: []basespec.DecoderID{
			markdown.ContextMarkdownDecoderID,
		},
	})
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

func normalizeWorkspaceSourceRoot(
	raw string,
) (string, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", fmt.Errorf(
			"%w: Workspace Source root path is required",
			basespec.ErrInvalid,
		)
	}
	absolute, err := filepath.Abs(raw)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
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
