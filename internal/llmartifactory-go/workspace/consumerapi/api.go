package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/workspacecatalog/defaultpolicy"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/iofs"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/mcp"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/prompt"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/skill"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/domain"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

type StoreAPI struct {
	roots            root.API
	cat              catalog.API
	sources          source.API
	discovery        refreshFlow.API
	artifacts        artifact.API
	resources        resourceFlow.API
	workspaceSources workspaceSourceRegistry
	policy           defaultpolicy.Policy

	config        Config
	promptAdapter *prompt.Adapter
	skillAdapter  *skill.Adapter
	mcpAdapter    *mcp.Adapter
	resolver      *composition.Resolver
}

func NewStoreAPI(
	sources source.API,
	discovery refreshFlow.API,
	artifacts artifact.API,
	resources resourceFlow.API,
	nativeResources resourceFlow.NativePathAPI,
	roots root.API,
	cat catalog.API,
	config Config,
) (*StoreAPI, error) {
	if sources == nil || discovery == nil || artifacts == nil || resources == nil || roots == nil || cat == nil {
		return nil, fmt.Errorf(
			"%w: Workspace Store dependencies are incomplete",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	policy, err := defaultpolicy.Load()
	if err != nil {
		return nil, err
	}

	config = config.normalized()
	if err := config.ContextComposition.Validate(); err != nil {
		return nil, err
	}

	workspaceSources := newWorkspaceSourceRegistry(sources)
	refreshCoordinator := newWorkspaceRefreshCoordinator(
		sources, discovery, workspaceSources,
	)
	output := &StoreAPI{
		roots:            roots,
		sources:          sources,
		discovery:        discovery,
		artifacts:        artifacts,
		resources:        resources,
		workspaceSources: workspaceSources,
		policy:           policy,
		config:           config,
		cat:              cat,
	}

	promptAdapter, err := prompt.New(
		artifacts,
		resources,
		config.ContextComposition,
	)
	if err != nil {
		return nil, err
	}
	skillAdapter, err := skill.New(artifacts, resources, nativeResources)
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

	locators, err := composition.NewProviderLocatorResolver(
		config.LocatorResolvers,
		workspaceLocatorRuntime{
			cat: cat,
		},
	)
	if err != nil {
		return nil, err
	}
	resolver, err := composition.NewWithOptions(
		composition.ResolverOptions{
			Artifacts:            artifacts,
			Catalog:              cat,
			SourceEntries:        resources,
			Locators:             locators,
			FallbackProviders:    config.FallbackProviders,
			TargetMappers:        config.TargetMappers,
			ProtectedBuiltinRoot: topology.BuiltinRootID(),
			Refresh:              refreshCoordinator,
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

func (a *StoreAPI) ListWorkspaceDirectoryArtifacts(
	ctx context.Context,
	directory WorkspaceDirectoryRef,
) ([]WorkspaceArtifactView, error) {
	if err := directory.RootID.Validate(); err != nil {
		return nil, err
	}
	values, err := a.workspaceSources.required(ctx, directory.RootID)
	if err != nil {
		return nil, err
	}
	directoryEntries, err := a.cat.ListBySource(
		ctx,
		directory.RootID,
		values.Directory.ID,
		catalogModel.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	policyEntries, err := a.cat.ListBySource(
		ctx,
		directory.RootID,
		values.Policy.ID,
		catalogModel.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	output := make(
		[]WorkspaceArtifactView,
		0,
		len(directoryEntries)+1,
	)
	for _, entry := range directoryEntries {
		output = append(output, workspaceArtifactCatalogViewOf(entry))
	}
	for _, entry := range policyEntries {
		if entry.Binding.Locator !=
			spec.Locator(defaultpolicy.PolicyLocator) ||
			entry.Binding.SubresourceLocator != "" ||
			entry.Kind != workspaceDomain.WorkspaceArtifactKind ||
			string(entry.LogicalName) != defaultpolicy.PolicyID {
			continue
		}
		output = append(output, workspaceArtifactCatalogViewOf(entry))
		break
	}

	sort.Slice(output, func(left, right int) bool {
		leftKey := string(output[left].Kind) + "\x00" +
			string(output[left].LogicalName) + "\x00" +
			string(output[left].SourceID) + "\x00" +
			string(output[left].Locator) + "\x00" +
			string(output[left].Artifact.ArtifactID)
		rightKey := string(output[right].Kind) + "\x00" +
			string(output[right].LogicalName) + "\x00" +
			string(output[right].SourceID) + "\x00" +
			string(output[right].Locator) + "\x00" +
			string(output[right].Artifact.ArtifactID)
		return leftKey < rightKey
	})
	return output, nil
}

func (a *StoreAPI) SetWorkspaceDirectoryArtifactEnabled(
	ctx context.Context,
	directory WorkspaceDirectoryRef,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (WorkspaceArtifactView, error) {
	if err := directory.RootID.Validate(); err != nil {
		return WorkspaceArtifactView{}, err
	}
	if err := ref.Validate(); err != nil {
		return WorkspaceArtifactView{}, err
	}
	values, err := a.workspaceSources.required(ctx, directory.RootID)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	if ref.RootID != directory.RootID {
		return WorkspaceArtifactView{}, fmt.Errorf(
			"%w: Artifact belongs to another Root",
			workspaceDomain.ErrReferenceUnresolved,
		)
	}
	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	if !isWorkspaceDirectoryCatalogArtifact(values, record) {
		return WorkspaceArtifactView{}, fmt.Errorf(
			"%w: Artifact is not exposed by this Workspace directory",
			workspaceDomain.ErrReferenceUnresolved,
		)
	}
	if record.Ref() != ref {
		return WorkspaceArtifactView{}, fmt.Errorf(
			"%w: Artifact lookup returned another occurrence",
			spec.ErrInvalid,
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
	return workspaceArtifactViewOf(updated), nil
}

func (a *StoreAPI) RegisterWorkspaceDirectory(ctx context.Context, path string) (WorkspaceDirectoryView, error) {
	rootPath, err := normalizeWorkspaceDirectoryPath(path)
	if err != nil {
		return WorkspaceDirectoryView{}, err
	}
	existing, found, err := a.findWorkspaceRoot(ctx, rootPath)
	if err != nil {
		return WorkspaceDirectoryView{}, err
	}
	var rootValue rootModel.Root
	if found {
		rootValue = existing
	} else {
		created, _, err := a.ensureWorkspaceRoot(ctx, rootPath)
		if err != nil {
			return WorkspaceDirectoryView{}, err
		}
		rootValue = created
	}
	if _, _, err := a.ensureWorkspaceSources(ctx, rootValue.ID, rootPath); err != nil {
		return WorkspaceDirectoryView{}, err
	}
	if err := a.refreshEffectiveWorkspaces(ctx, rootValue.ID); err != nil {
		return WorkspaceDirectoryView{}, err
	}
	return a.buildDirectoryView(ctx, rootValue.ID)
}

func (a *StoreAPI) GetWorkspaceDirectory(
	ctx context.Context,
	ref WorkspaceDirectoryRef,
) (WorkspaceDirectoryView, error) {
	if err := ref.RootID.Validate(); err != nil {
		return WorkspaceDirectoryView{}, err
	}
	return a.buildDirectoryView(ctx, ref.RootID)
}

func (a *StoreAPI) RefreshWorkspaceDirectory(
	ctx context.Context,
	ref WorkspaceDirectoryRef,
) (WorkspaceDirectoryView, error) {
	if err := ref.RootID.Validate(); err != nil {
		return WorkspaceDirectoryView{}, err
	}
	values, err := a.workspaceSources.required(ctx, ref.RootID)
	if err != nil {
		return WorkspaceDirectoryView{}, err
	}
	if !values.Directory.Enabled || !values.Policy.Enabled {
		return a.buildDirectoryView(ctx, ref.RootID)
	}
	for _, summary := range []sourceModel.Summary{
		values.Directory,
		values.Policy,
	} {
		if _, err := a.discovery.RefreshSource(ctx, ref.RootID, summary.ID); err != nil {
			return WorkspaceDirectoryView{}, err
		}
	}
	if err := a.refreshEffectiveWorkspaces(ctx, ref.RootID); err != nil {
		return WorkspaceDirectoryView{}, err
	}
	return a.buildDirectoryView(ctx, ref.RootID)
}

func (a *StoreAPI) SetWorkspaceDirectoryEnabled(
	ctx context.Context,
	ref WorkspaceDirectoryRef,
	expectedRevision uint64,
	enabled bool,
) (WorkspaceDirectoryView, error) {
	if err := ref.RootID.Validate(); err != nil {
		return WorkspaceDirectoryView{}, err
	}
	values, err := a.workspaceSources.required(ctx, ref.RootID)
	if err != nil {
		return WorkspaceDirectoryView{}, err
	}
	if expectedRevision != 0 &&
		values.Directory.Revision != expectedRevision {
		return WorkspaceDirectoryView{}, spec.ErrConflict
	}

	update := func(current sourceModel.Summary) error {
		if current.Enabled == enabled {
			return nil
		}
		_, err := a.sources.Update(
			ctx,
			ref.RootID,
			current.ID,
			sourceModel.Update{
				ExpectedRevision: current.Revision,
				DisplayName:      current.DisplayName,
				Enabled:          enabled,
			},
		)
		return err
	}

	// Keep the aggregate disabled during a partial two-Source transition.
	order := []sourceModel.Summary{values.Directory, values.Policy}
	if enabled {
		order = []sourceModel.Summary{values.Policy, values.Directory}
	}
	for _, current := range order {
		if err := update(current); err != nil {
			return WorkspaceDirectoryView{}, err
		}
	}
	if enabled {
		return a.RefreshWorkspaceDirectory(ctx, ref)
	}
	return a.buildDirectoryView(ctx, ref.RootID)
}

func (a *StoreAPI) RemoveWorkspaceDirectory(
	ctx context.Context,
	ref WorkspaceDirectoryRef,
	expectedRevision uint64,
) error {
	if err := ref.RootID.Validate(); err != nil {
		return err
	}
	values, err := a.workspaceSources.load(ctx, ref.RootID)
	if err != nil {
		return err
	}
	if !values.HasDirectory && !values.HasPolicy {
		return fmt.Errorf(
			"%w: Workspace directory is not registered",
			spec.ErrReferenceUnresolved,
		)
	}
	if expectedRevision != 0 &&
		values.HasDirectory &&
		values.Directory.Revision != expectedRevision {
		return spec.ErrConflict
	}

	ownedSources := make(map[sourceModel.SourceID]struct{}, 2)
	retire := func(current sourceModel.Summary) error {
		ownedSources[current.ID] = struct{}{}
		_, err := a.sources.Retire(
			ctx,
			ref.RootID,
			current.ID,
			current.Revision,
		)
		return err
	}

	// Retire policy first so a partial operation cannot expose the default.
	if values.HasPolicy {
		if err := retire(values.Policy); err != nil {
			return err
		}
	}
	if values.HasDirectory {
		if err := retire(values.Directory); err != nil {
			return err
		}
	}

	records, err := a.cat.ListByRoot(
		ctx,
		ref.RootID,
		catalogModel.ListOptions{},
	)
	if err != nil {
		return err
	}
	for _, record := range records {
		if _, owned := ownedSources[record.Binding.SourceID]; !owned {
			continue
		}
		if record.State != artifactModel.StateMissing {
			return fmt.Errorf(
				"%w: retired Workspace Source still has a non-missing Artifact",
				spec.ErrConflict,
			)
		}
		if err := a.artifacts.Purge(ctx, record.Ref(), record.Revision); err != nil {
			return err
		}
	}
	return nil
}

func (a *StoreAPI) ListWorkspaceDirectories(ctx context.Context, request WorkspacePageRequest) (WorkspacePage, error) {
	limit := request.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	rootIDs, err := a.workspaceDirectoryRootIDs(ctx)
	if err != nil {
		return WorkspacePage{}, err
	}

	var cursor rootModel.RootID
	if request.Cursor != "" {
		cursor = rootModel.RootID(request.Cursor)
		if err := cursor.Validate(); err != nil {
			return WorkspacePage{}, err
		}
	}

	start := 0
	for start < len(rootIDs) && rootIDs[start] <= cursor {
		start++
	}
	end := min(start+limit, len(rootIDs))

	output := WorkspacePage{
		Items: make([]WorkspaceDirectoryListItem, 0, end-start),
	}
	for _, rootID := range rootIDs[start:end] {
		item, err := a.workspaceDirectoryListItem(ctx, rootID)
		if err != nil {
			return WorkspacePage{}, err
		}
		output.Items = append(output.Items, item)
	}
	if end < len(rootIDs) {
		output.NextCursor = string(rootIDs[end-1])
	}
	return output, nil
}

func (a *StoreAPI) WorkspaceDefaultPolicy() WorkspaceDefaultPolicyView {
	return WorkspaceDefaultPolicyView{
		ID:      a.policy.ID,
		Version: a.policy.Version,
		Digest:  a.policy.Digest,
		YAML:    string(a.policy.RawYAML),
	}
}

func (a *StoreAPI) workspaceDirectoryListItem(
	ctx context.Context,
	rootID rootModel.RootID,
) (WorkspaceDirectoryListItem, error) {
	rootValue, err := a.roots.Get(ctx, rootID)
	if err != nil {
		return WorkspaceDirectoryListItem{}, err
	}
	values, err := a.workspaceSources.required(ctx, rootID)
	if err != nil {
		return WorkspaceDirectoryListItem{}, err
	}
	return WorkspaceDirectoryListItem{
		Ref:                     WorkspaceDirectoryRef{RootID: rootID},
		RootID:                  rootID,
		RootDisplayName:         rootValue.DisplayName,
		Enabled:                 values.Directory.Enabled && values.Policy.Enabled,
		DirectorySourceID:       values.Directory.ID,
		DirectorySourceRevision: values.Directory.Revision,
		PolicyID:                a.policy.ID,
		PolicyVersion:           a.policy.Version,
		PolicyDigest:            a.policy.Digest,
	}, nil
}

func (a *StoreAPI) ensureWorkspaceSources(
	ctx context.Context,
	rootID rootModel.RootID,
	rootPath string,
) (dir, policy sourceModel.Summary, err error) {
	directoryDiscovery, err := a.defaultDiscovery()
	if err != nil {
		return sourceModel.Summary{}, sourceModel.Summary{}, err
	}
	directoryConfig, err := json.Marshal(struct {
		RootPath string `json:"rootPath"`
	}{RootPath: rootPath})
	if err != nil {
		return sourceModel.Summary{}, sourceModel.Summary{}, err
	}
	directory, err := refreshFlow.EnsureAndRefreshSource(
		ctx,
		a.sources,
		a.discovery,
		refreshFlow.EnsureAndRefreshSourceRequest{
			RootID: rootID,
			Draft: sourceModel.Draft{
				ID:          sourceModel.SourceID(uuidutil.NewUUIDv7()),
				StorageKey:  WorkspaceDirectorySourceStorageKey,
				Kind:        fsdir.Kind,
				DisplayName: "Workspace directory source",
				Enabled:     true,
				Config:      directoryConfig,
				Discovery:   directoryDiscovery,
			},
			ReconcileDiscovery: source.MergeDiscoveryScopes,
		},
	)
	if err != nil {
		return sourceModel.Summary{}, sourceModel.Summary{}, err
	}
	policyDiscovery, err := policySourceDiscovery()
	if err != nil {
		return sourceModel.Summary{}, sourceModel.Summary{}, err
	}
	policyConfig, err := json.Marshal(struct {
		ProviderKey string `json:"providerKey"`
		Root        string `json:"root"`
	}{ProviderKey: defaultpolicy.ProviderKey, Root: defaultpolicy.PolicyRoot})
	if err != nil {
		return sourceModel.Summary{}, sourceModel.Summary{}, err
	}
	policy, err = refreshFlow.EnsureAndRefreshSource(
		ctx,
		a.sources,
		a.discovery,
		refreshFlow.EnsureAndRefreshSourceRequest{
			RootID: rootID,
			Draft: sourceModel.Draft{
				ID:          sourceModel.SourceID(uuidutil.NewUUIDv7()),
				StorageKey:  WorkspaceBasePolicySourceStorageKey,
				Kind:        iofs.Kind,
				DisplayName: "Workspace base policy source",
				Enabled:     true,
				Config:      policyConfig,
				Discovery:   policyDiscovery,
			},
		},
	)
	if err != nil {
		return sourceModel.Summary{}, sourceModel.Summary{}, err
	}
	return directory, policy, nil
}

// defaultDiscovery bootstraps Workspace identification. An explicitly
// present Workspace declarations array is reconciled later by the loader.
func (a *StoreAPI) defaultDiscovery() (
	sourceModel.DiscoverySpec,
	error,
) {
	value, err := topology.DiscoverySpecForUse(
		topology.DiscoveryUseWorkspace,
	)
	if err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	for _, hint := range a.config.AdditionalDecoderHints {
		value.DecoderHints = source.AppendDecoderHint(
			value.DecoderHints,
			hint,
		)
	}
	value = value.Normalized()
	if err := value.Validate(); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	return value, nil
}

func (a *StoreAPI) listManifestIntent(
	ctx context.Context,
	rootID rootModel.RootID,
	directoryID sourceModel.SourceID,
) ([]spec.Locator, error) {
	entries, err := a.resources.ReadSourceTree(
		ctx,
		rootID,
		directoryID,
		".",
		topology.WorkspaceManifestPatterns(),
		nil,
		spec.DefaultMaxEntries,
		spec.MaxScanBytes,
	)
	if err != nil {
		return nil, err
	}
	output := make([]spec.Locator, 0, len(entries))
	for _, entry := range entries {
		output = append(output, entry.Locator)
	}
	slices.Sort(output)
	return output, nil
}

func (a *StoreAPI) listPhysicalWorkspaces(
	ctx context.Context,
	rootID rootModel.RootID,
	directoryID sourceModel.SourceID,
) ([]artifactModel.Artifact, error) {
	entries, err := a.cat.ListBySource(
		ctx,
		rootID,
		directoryID,
		catalogModel.ListOptions{},
	)
	if err != nil {
		return nil, err
	}
	refs := make([]artifactModel.ArtifactRef, 0)
	for _, entry := range entries {
		if entry.Kind != workspaceDomain.WorkspaceArtifactKind {
			continue
		}
		if entry.Binding.SubresourceLocator != "" ||
			!topology.IsWorkspaceManifestLocator(entry.Binding.Locator) {
			continue
		}
		if entry.State != artifactModel.StateAvailable {
			continue
		}
		refs = append(refs, entry.Ref())
	}
	output, err := a.artifacts.GetMany(ctx, refs)
	if err != nil {
		return nil, err
	}
	sort.Slice(output, func(i, j int) bool {
		if output[i].LogicalName != output[j].LogicalName {
			return output[i].LogicalName < output[j].LogicalName
		}
		return output[i].ID < output[j].ID
	})
	return output, nil
}

func (a *StoreAPI) effectiveWorkspaces(
	ctx context.Context,
	rootID rootModel.RootID,
	directory, policy sourceModel.Summary,
) ([]WorkspaceDirectoryWorkspace, []diagnostic.Diagnostic, error) {
	if !directory.Enabled {
		return []WorkspaceDirectoryWorkspace{}, nil, nil
	}
	if !policy.Enabled {
		return []WorkspaceDirectoryWorkspace{}, nil, nil
	}
	intent, err := a.listManifestIntent(ctx, rootID, directory.ID)
	if err != nil {
		return nil, nil, err
	}
	physical, err := a.listPhysicalWorkspaces(ctx, rootID, directory.ID)
	if err != nil {
		return nil, nil, err
	}
	diagnostics := invalidWorkspaceManifestDiagnostics(
		intent,
		physical,
	)
	if len(intent) == 0 && len(physical) == 0 {
		record, found, err := a.defaultWorkspaceRecord(
			ctx,
			workspaceSourceSet{
				Directory:    directory,
				Policy:       policy,
				HasDirectory: true,
				HasPolicy:    true,
			},
		)
		if err != nil {
			return nil, nil, err
		}
		if found &&
			record.State == artifactModel.StateAvailable &&
			record.Enabled {
			workspace, err := a.workspaceForRef(ctx, record.Ref())
			if err != nil {
				return nil, nil, err
			}
			if cryptoutil.DigestBytes(workspace.Definition.Body) != a.policy.Digest {
				return nil, nil, fmt.Errorf(
					"%w: default Workspace Definition differs from the loaded policy",
					spec.ErrDigestMismatch,
				)
			}
			return []WorkspaceDirectoryWorkspace{{
				Workspace: workspace.View(),
				Origin:    WorkspaceDirectoryOriginDefault,
			}}, diagnostics, nil
		}
		diagnostics = diagnostic.Append(
			diagnostics,
			diagnostic.Diagnostic{
				Severity: diagnostic.SeverityWarning,
				Code:     "workspace.default-unavailable",
				Message:  "default Workspace policy is unavailable",
			},
		)
		return []WorkspaceDirectoryWorkspace{}, diagnostics, nil
	}
	output := make([]WorkspaceDirectoryWorkspace, 0, len(physical))
	seen := make(map[artifactModel.ArtifactRef]struct{}, len(physical))
	for _, record := range physical {
		if !record.Enabled {
			continue
		}
		workspace, err := a.workspaceForRef(ctx, record.Ref())
		if err != nil {
			return nil, nil, err
		}
		if !workspace.Artifact.Enabled {
			continue
		}
		if _, duplicate := seen[workspace.Ref()]; duplicate {
			continue
		}
		seen[workspace.Ref()] = struct{}{}
		output = append(
			output,
			WorkspaceDirectoryWorkspace{
				Workspace:       workspace.View(),
				Origin:          WorkspaceDirectoryOriginManifest,
				ManifestLocator: record.Binding.Locator,
			},
		)
	}
	if len(output) == 0 && len(diagnostics) == 0 {
		diagnostics = diagnostic.Append(
			diagnostics,
			diagnostic.Diagnostic{
				Severity: diagnostic.SeverityWarning,
				Code:     "workspace.manifest-unavailable",
				Message:  "physical Workspace declarations are unavailable or disabled",
			},
		)
	}
	return output, diagnostics, nil
}

func invalidWorkspaceManifestDiagnostics(
	intent []spec.Locator,
	physical []artifactModel.Artifact,
) []diagnostic.Diagnostic {
	available := make(map[spec.Locator]struct{}, len(physical))
	for _, record := range physical {
		available[record.Binding.Locator] = struct{}{}
	}

	var output []diagnostic.Diagnostic
	for _, locator := range intent {
		if _, found := available[locator]; found {
			continue
		}
		output = diagnostic.Append(output, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityWarning,
			Code:     "workspace.manifest-invalid",
			Message:  "Workspace manifest did not produce an available top-level Workspace declaration",
			Location: &diagnostic.Location{
				Locator: locator,
			},
		})
	}
	return output
}

func (a *StoreAPI) buildDirectoryView(ctx context.Context, rootID rootModel.RootID) (WorkspaceDirectoryView, error) {
	rootValue, err := a.roots.Get(ctx, rootID)
	if err != nil {
		return WorkspaceDirectoryView{}, err
	}
	values, err := a.workspaceSources.required(ctx, rootID)
	if err != nil {
		return WorkspaceDirectoryView{}, err
	}
	enabled := values.Directory.Enabled && values.Policy.Enabled
	workspaces, diagnostics, err := a.effectiveWorkspaces(
		ctx,
		rootID,
		values.Directory,
		values.Policy,
	)
	if err != nil {
		return WorkspaceDirectoryView{}, err
	}
	return WorkspaceDirectoryView{
		Ref:             WorkspaceDirectoryRef{RootID: rootID},
		Root:            rootValue,
		DirectorySource: values.Directory,
		Enabled:         enabled,
		PolicyID:        a.policy.ID,
		PolicyVersion:   a.policy.Version,
		PolicyDigest:    a.policy.Digest,
		Workspaces:      workspaces,
		Diagnostics:     diagnostics,
	}, nil
}

func workspaceArtifactViewOf(
	value artifactModel.Artifact,
) WorkspaceArtifactView {
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
	}
}

func workspaceArtifactCatalogViewOf(
	value catalogModel.Entry,
) WorkspaceArtifactView {
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
	}
}

func isWorkspaceDirectoryCatalogArtifact(
	values workspaceSourceSet,
	value artifactModel.Artifact,
) bool {
	if value.RootID != values.Directory.RootID {
		return false
	}
	if value.Binding.SourceID == values.Directory.ID {
		return true
	}
	return value.Binding.SourceID == values.Policy.ID &&
		value.Binding.Locator == spec.Locator(defaultpolicy.PolicyLocator) &&
		value.Binding.SubresourceLocator == "" &&
		value.Kind == workspaceDomain.WorkspaceArtifactKind &&
		string(value.LogicalName) == defaultpolicy.PolicyID
}

func workspaceRootStorageKey(rootPath string) spec.StorageKey {
	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(rootPath))),
		cryptoutil.DigestSHA256Prefix,
	)
	return spec.StorageKey(WorkspaceRootStorageKeyPrefix + digest)
}

func policySourceDiscovery() (sourceModel.DiscoverySpec, error) {
	value := sourceModel.DiscoverySpec{
		ExplicitLocators: []spec.Locator{defaultpolicy.PolicyLocator},
		DecoderHints: []sourceModel.DecoderHint{{
			Locator:    defaultpolicy.PolicyLocator,
			Recursive:  false,
			DecoderIDs: []spec.DecoderID{"artifact-declaration-yaml"},
		}},
		AllowedDecoderIDs: []spec.DecoderID{"artifact-declaration-yaml"},
		Authoritative:     true,
	}
	value = value.Normalized()
	if err := value.Validate(); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	return value, nil
}
