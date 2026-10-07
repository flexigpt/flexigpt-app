package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"

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
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	corerefresh "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/refresh"
	workspacemcp "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/mcp"
	workspaceprompt "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/prompt"
	workspaceskill "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/skill"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/domain"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

type Service struct {
	roots            root.API
	cat              catalog.API
	sources          source.API
	discovery        refreshFlow.API
	artifacts        artifact.API
	resources        resourceFlow.API
	workspaceSources workspaceSourceRegistry
	policy           workspaceDomain.DefaultPolicy
	policySource     DefaultPolicySource

	config        Config
	promptAdapter *workspaceprompt.Adapter
	skillAdapter  *workspaceskill.Adapter
	mcpAdapter    *workspacemcp.Adapter
	resolver      *composition.Resolver
	refresher     *corerefresh.Service
}

func New(
	sources source.API,
	discovery refreshFlow.API,
	artifacts artifact.API,
	resources resourceFlow.API,
	nativeResources resourceFlow.NativePathAPI,
	roots root.API,
	cat catalog.API,
	config Config,
) (*Service, error) {
	if sources == nil || discovery == nil || artifacts == nil ||
		resources == nil || nativeResources == nil || roots == nil || cat == nil {
		return nil, fmt.Errorf(
			"%w: Workspace Store dependencies are incomplete",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	config = config.normalized()
	if err := config.DefaultPolicySource.Validate(); err != nil {
		return nil, err
	}
	if err := config.ContextComposition.Validate(); err != nil {
		return nil, err
	}
	if err := config.Support.Validate(); err != nil {
		return nil, err
	}
	if config.Composition == nil {
		return nil, fmt.Errorf(
			"%w: Workspace composition resolver is required",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	policy, err := config.DefaultPolicySource.Policy.Clone()
	if err != nil {
		return nil, err
	}

	workspaceSources := newWorkspaceSourceRegistry(
		sources,
		config.Support.DirectorySource,
		config.Support.PolicySource,
	)
	refreshCoordinator := newWorkspaceRefreshCoordinator(
		sources,
		workspaceSources,
		config.Support,
	)

	output := &Service{
		roots:            roots,
		sources:          sources,
		discovery:        discovery,
		artifacts:        artifacts,
		resources:        resources,
		workspaceSources: workspaceSources,
		policy:           policy,
		policySource:     config.DefaultPolicySource,
		config:           config,
		cat:              cat,
	}

	promptAdapter, err := workspaceprompt.New(
		artifacts,
		resources,
		config.ContextComposition,
	)
	if err != nil {
		return nil, err
	}
	skillAdapter, err := workspaceskill.New(
		artifacts,
		resources,
		nativeResources,
		config.Support.SkillDocuments,
	)
	if err != nil {
		return nil, err
	}

	var mcpAdapter *workspacemcp.Adapter
	if config.MCPServers != nil {
		mcpAdapter, err = workspacemcp.New(
			artifacts,
			config.MCPServers,
		)
		if err != nil {
			return nil, err
		}
	}

	output.resolver = config.Composition
	planner, err := composition.NewReachableDiscoveryPlanner(
		func(
			ctx context.Context,
			ref artifactModel.ArtifactRef,
		) (*composition.ResolvedEntry, error) {
			workspace, err := output.workspaceForRef(ctx, ref)
			if err != nil {
				return nil, err
			}
			return config.Composition.ResolveWorkspaceWithCompositionSource(
				ctx,
				workspace.Ref(),
				workspace.CompositionSourceID,
			)
		},
		refreshCoordinator,
	)
	if err != nil {
		return nil, err
	}
	refresher, err := corerefresh.New(
		sources,
		discovery,
		planner,
		composition.DefaultLimits().MaxDepth,
	)
	if err != nil {
		return nil, err
	}
	output.promptAdapter = promptAdapter
	output.skillAdapter = skillAdapter
	output.mcpAdapter = mcpAdapter
	output.refresher = refresher
	return output, nil
}

func (a *Service) ListWorkspaceDirectoryArtifacts(
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
			a.policySource.Locator ||
			entry.Binding.SubresourceLocator != "" ||
			entry.Kind != workspaceDomain.WorkspaceArtifactKind ||
			string(entry.LogicalName) != a.policy.ID {
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

func (a *Service) SetWorkspaceDirectoryArtifactEnabled(
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
	if !a.isWorkspaceDirectoryCatalogArtifact(values, record) {
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

func (a *Service) RegisterWorkspaceDirectory(ctx context.Context, p string) (WorkspaceDirectoryView, error) {
	rootPath, err := normalizeWorkspaceDirectoryPath(p)
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

func (a *Service) GetWorkspaceDirectory(
	ctx context.Context,
	ref WorkspaceDirectoryRef,
) (WorkspaceDirectoryView, error) {
	if err := ref.RootID.Validate(); err != nil {
		return WorkspaceDirectoryView{}, err
	}
	return a.buildDirectoryView(ctx, ref.RootID)
}

func (a *Service) RefreshWorkspaceDirectory(
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

func (a *Service) SetWorkspaceDirectoryEnabled(
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

func (a *Service) RemoveWorkspaceDirectory(
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

func (a *Service) ListWorkspaceDirectories(ctx context.Context, request WorkspacePageRequest) (WorkspacePage, error) {
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

func (a *Service) WorkspaceDefaultPolicy() WorkspaceDefaultPolicyView {
	return WorkspaceDefaultPolicyView{
		ID:      a.policy.ID,
		Version: a.policy.Version,
		Digest:  a.policy.Digest,
		YAML:    string(a.policy.RawYAML),
	}
}

func (a *Service) workspaceDirectoryListItem(
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

func (a *Service) ensureWorkspaceSources(
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
				StorageKey:  a.config.Support.DirectorySource.StorageKey,
				Kind:        a.config.Support.DirectorySource.Kind,
				DisplayName: a.config.Support.DirectorySource.DisplayName,
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
	policyDiscovery, err := a.policySourceDiscovery()
	if err != nil {
		return sourceModel.Summary{}, sourceModel.Summary{}, err
	}
	policyConfig, err := json.Marshal(struct {
		ProviderKey string `json:"providerKey"`
		Root        string `json:"root"`
	}{
		ProviderKey: a.policySource.ProviderKey,
		Root:        string(a.policySource.Root),
	})
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
				StorageKey:  a.config.Support.PolicySource.StorageKey,
				Kind:        a.config.Support.PolicySource.Kind,
				DisplayName: a.config.Support.PolicySource.DisplayName,
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
func (a *Service) defaultDiscovery() (
	sourceModel.DiscoverySpec,
	error,
) {
	value := a.config.Support.DirectoryDiscovery.Clone()
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

func (a *Service) listManifestIntent(
	ctx context.Context,
	rootID rootModel.RootID,
	directoryID sourceModel.SourceID,
) ([]spec.Locator, error) {
	entries, err := a.resources.ReadSourceTree(
		ctx,
		rootID,
		directoryID,
		".",
		a.config.Support.ManifestPatterns,
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

func (a *Service) listPhysicalWorkspaces(
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
			!a.isWorkspaceManifestLocator(entry.Binding.Locator) {
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

func (a *Service) effectiveWorkspaces(
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

func (a *Service) buildDirectoryView(ctx context.Context, rootID rootModel.RootID) (WorkspaceDirectoryView, error) {
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

func (a *Service) isWorkspaceDirectoryCatalogArtifact(
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
		value.Binding.Locator == a.policySource.Locator &&
		value.Binding.SubresourceLocator == "" &&
		value.Kind == workspaceDomain.WorkspaceArtifactKind &&
		string(value.LogicalName) == a.policy.ID
}

func (a *Service) isWorkspaceManifestLocator(
	locator spec.Locator,
) bool {
	value := string(locator)
	name := path.Base(value)
	for _, pattern := range a.config.Support.ManifestPatterns {
		matched, err := spec.MatchPathPattern(pattern, value)
		if err == nil && matched {
			return true
		}
		matched, err = spec.MatchPathPattern(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func (a *Service) workspaceRootStorageKey(rootPath string) spec.StorageKey {
	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(rootPath))),
		cryptoutil.DigestSHA256Prefix,
	)
	return spec.StorageKey(
		string(a.config.Support.RootStorageKeyPrefix) + digest,
	)
}

func (a *Service) policySourceDiscovery() (sourceModel.DiscoverySpec, error) {
	value := sourceModel.DiscoverySpec{
		ExplicitLocators: []spec.Locator{a.policySource.Locator},
		DecoderHints: []sourceModel.DecoderHint{{
			Locator:    a.policySource.Locator,
			Recursive:  false,
			DecoderIDs: []spec.DecoderID{a.policySource.DecoderID},
		}},
		AllowedDecoderIDs: []spec.DecoderID{a.policySource.DecoderID},
		Authoritative:     true,
	}
	value = value.Normalized()
	if err := value.Validate(); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	return value, nil
}
