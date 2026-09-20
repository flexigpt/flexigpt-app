package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/consumerutil"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
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
	resolver, err := resolve.NewWithOptions(
		resolve.ResolverOptions{
			Artifacts:            artifacts,
			SourceArtifacts:      artifacts,
			SourceEntries:        resources,
			Locators:             locators,
			FallbackProviders:    config.FallbackProviders,
			ProtectedBuiltinRoot: builtin.BuiltinRootID,
			Refresh:              output,
			Limits:               config.ResolverLimits,
		},
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

	rootPath, err := consumerutil.NormalizeFilesystemSourceRoot(
		request.RootPath,
		"Workspace Source root path",
	)
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

	return consumerutil.EnsureAndRefreshSource(
		ctx,
		a.sources,
		a.discovery,
		consumerutil.EnsureAndRefreshSourceRequest{
			RootID: request.RootID,
			Draft: source.Draft{
				ID: source.SourceID(uuidutil.NewUUIDv7()),
				StorageKey: consumerutil.FilesystemSourceStorageKey(
					"workspace-path",
					rootPath,
				),
				Kind:        source.SourceKindFilesystemDirectory,
				DisplayName: request.SourceDisplayName,
				Enabled:     true,
				Config:      config,
				Discovery:   discovery,
			},
			ReconcileDiscovery: consumerutil.MergeDiscoveryScopes,
		},
	)
}

func (a *StoreAPI) GetWorkspace(
	ctx context.Context,
	ref artifact.ArtifactRef,
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

func (a *StoreAPI) ListWorkspaceArtifacts(
	ctx context.Context,
	workspace artifact.ArtifactRef,
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
	workspace artifact.ArtifactRef,
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
	if _, err := a.artifacts.Get(ctx, ref); err != nil {
		return WorkspaceArtifactView{}, err
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
	workspace artifact.ArtifactRef,
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
	if !workspaceRuntimeDisableSupported(record.Kind) {
		return WorkspaceArtifactView{}, fmt.Errorf(
			"%w: Workspace runtime disablement is not supported for MCP Artifacts",
			basespec.ErrUnsupported,
		)
	}

	baseline, err := a.isBaselineCollectionArtifact(ctx, record)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	if baseline {
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

func (a *StoreAPI) isBaselineCollectionArtifact(
	ctx context.Context,
	record artifact.Artifact,
) (bool, error) {
	if record.Kind != artifact.ArtifactKind(
		pluginv1.PluginType,
	) {
		return false, nil
	}
	sourceValue, err := a.sources.Get(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return false, err
	}
	return collection.IsBaselineCollectionArtifactForSource(
		record,
		sourceValue,
	), nil
}

// defaultDiscovery bootstraps Workspace identification. An explicitly
// present Workspace declarations array is reconciled later by the loader.
func (a *StoreAPI) defaultDiscovery() (
	source.DiscoverySpec,
	error,
) {
	value := documentTopology.WorkspaceDiscoverySpec()
	for _, hint := range a.config.AdditionalDecoderHints {
		value.DecoderHints = consumerutil.AppendDecoderHint(
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
	runtimeDisabled := data.RuntimeDisabled
	if !workspaceRuntimeDisableSupported(value.Kind) {
		runtimeDisabled = false
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
		RuntimeDisabled:    runtimeDisabled,
	}, nil
}

// Workspace runtime disablement remains a Workspace consumer feature for
// prompt and Skill materialization. MCP server and MCP policy behavior use no
// Workspace-specific enablement state.
func workspaceRuntimeDisableSupported(
	kind artifact.ArtifactKind,
) bool {
	return kind != mcpDomain.MCPArtifactKind &&
		kind != mcpDomain.MCPPolicyArtifactKind
}
