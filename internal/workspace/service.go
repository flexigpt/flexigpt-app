package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/baseutils"
	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

type Service struct {
	store       ArtifactStore
	planner     DiscoveryPlanner
	descriptors map[artifactstoreSpec.ArtifactKind]KindDescriptor
	projectors  map[artifactstoreSpec.ArtifactKind]ResourceProjector
	frontendIDs []artifactstoreSpec.FrontendID
}

type serviceConfig struct {
	planner        DiscoveryPlanner
	descriptors    map[artifactstoreSpec.ArtifactKind]KindDescriptor
	projectors     map[artifactstoreSpec.ArtifactKind]ResourceProjector
	extraFrontends []artifactstoreSpec.ArtifactFrontend
	yamlDecoder    YAMLDecoder
}

type Option func(*serviceConfig) error

func WithDiscoveryPlanner(planner DiscoveryPlanner) Option {
	return func(config *serviceConfig) error {
		if planner == nil {
			return fmt.Errorf("%w: discovery planner is nil", ErrInvalidWorkspace)
		}
		config.planner = planner
		return nil
	}
}

func WithKindDescriptor(descriptor KindDescriptor) Option {
	return func(config *serviceConfig) error {
		if descriptor.Kind == "" ||
			descriptor.DefinitionSchemaID == "" ||
			descriptor.CollectionSlug == "" ||
			strings.TrimSpace(descriptor.CollectionDisplayName) == "" {
			return fmt.Errorf("%w: incomplete kind descriptor", ErrInvalidWorkspace)
		}
		if _, exists := config.descriptors[descriptor.Kind]; exists {
			return fmt.Errorf("%w: duplicate workspace kind %q", ErrInvalidWorkspace, descriptor.Kind)
		}
		config.descriptors[descriptor.Kind] = descriptor
		return nil
	}
}

func WithArtifactFrontend(frontend artifactstoreSpec.ArtifactFrontend) Option {
	return func(config *serviceConfig) error {
		if frontend == nil {
			return fmt.Errorf("%w: artifact frontend is nil", ErrInvalidWorkspace)
		}
		config.extraFrontends = append(config.extraFrontends, frontend)
		return nil
	}
}

func WithResourceProjector(projector ResourceProjector) Option {
	return func(config *serviceConfig) error {
		if projector == nil || projector.Kind() == "" {
			return fmt.Errorf("%w: resource projector is invalid", ErrInvalidWorkspace)
		}
		if _, exists := config.projectors[projector.Kind()]; exists {
			return fmt.Errorf("%w: duplicate projector for %q", ErrInvalidWorkspace, projector.Kind())
		}
		config.projectors[projector.Kind()] = projector
		return nil
	}
}

func WithYAMLDecoder(decoder YAMLDecoder) Option {
	return func(config *serviceConfig) error {
		if decoder == nil {
			return fmt.Errorf("%w: YAML decoder is nil", ErrInvalidWorkspace)
		}
		config.yamlDecoder = decoder
		return nil
	}
}

func NewService(store ArtifactStore, options ...Option) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: artifact store is nil", ErrInvalidWorkspace)
	}
	config := serviceConfig{
		descriptors: defaultKindDescriptors(),
		projectors:  map[artifactstoreSpec.ArtifactKind]ResourceProjector{},
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}

	frontend := &nativeFrontend{
		descriptors: cloneDescriptors(config.descriptors),
		yamlDecoder: config.yamlDecoder,
	}
	if err := store.RegisterRootKindHook(rootKindHook{}); err != nil {
		return nil, fmt.Errorf("register Workspace root hook: %w", err)
	}
	if err := store.RegisterCollectionKindHook(collectionKindHook{}); err != nil {
		return nil, fmt.Errorf("register Workspace collection hook: %w", err)
	}
	if err := store.RegisterArtifactFrontend(frontend); err != nil {
		return nil, fmt.Errorf("register Workspace native frontend: %w", err)
	}

	frontendIDs := []artifactstoreSpec.FrontendID{
		NativeFrontendID,
		artifactstoreSpec.PortableDefinitionFrontendID,
	}
	for _, extra := range config.extraFrontends {
		if err := store.RegisterArtifactFrontend(extra); err != nil {
			return nil, fmt.Errorf("register Workspace frontend %q: %w", extra.ID(), err)
		}
		frontendIDs = append(frontendIDs, extra.ID())
	}
	slices.Sort(frontendIDs)

	planner := config.planner
	if planner == nil {
		planner = defaultDiscoveryPlanner{}
	}
	return &Service{
		store:       store,
		planner:     planner,
		descriptors: cloneDescriptors(config.descriptors),
		projectors:  config.projectors,
		frontendIDs: frontendIDs,
	}, nil
}

func (s *Service) SelectFilesystemRoot(
	ctx context.Context,
	request FilesystemSelectionRequest,
) (Workspace, error) {
	if s == nil || s.store == nil {
		return Workspace{}, fmt.Errorf("%w: service is not configured", ErrInvalidWorkspace)
	}
	if strings.TrimSpace(request.DisplayName) == "" ||
		strings.TrimSpace(request.RootPath) == "" {
		return Workspace{}, fmt.Errorf(
			"%w: displayName and rootPath are required",
			ErrInvalidWorkspace,
		)
	}
	sourceConfig, err := json.Marshal(artifactstoreSpec.FSDirectorySourceConfig{
		RootPath:       request.RootPath,
		FollowSymlinks: request.FollowSymlinks,
		ManagedByApp:   request.ManagedByApp,
	})
	if err != nil {
		return Workspace{}, err
	}
	source, err := s.store.CreateSource(ctx, artifactstoreSpec.SourceDraft{
		Kind:           artifactstoreSpec.SourceKindFSDirectory,
		DisplayName:    request.DisplayName,
		Enabled:        true,
		ConfigSchemaID: artifactstoreSpec.FSDirectoryConfigSchemaID,
		Config:         sourceConfig,
	})
	if err != nil {
		return Workspace{}, err
	}

	rootData := RootData{
		Mode:                     RootModeFilesystem,
		PrimarySourceID:          source.SourceID,
		RootTrustReference:       request.TrustReference,
		DiscoveryPreferences:     request.Discovery,
		CapabilityProfileVersion: CapabilityProfileVersion,
	}
	rootDataJSON, err := encodeRootData(rootData)
	if err != nil {
		deleteErr := s.store.DeleteSource(ctx, source.SourceID, source.ModifiedAt)
		return Workspace{}, errors.Join(err, deleteErr)
	}
	root, err := s.store.CreateRoot(ctx, artifactstoreSpec.RootDraft{
		Kind:         RootKind,
		DisplayName:  request.DisplayName,
		Description:  request.Description,
		Enabled:      true,
		DataSchemaID: RootDataSchemaID,
		Data:         rootDataJSON,
	})
	if err != nil {
		deleteErr := s.store.DeleteSource(ctx, source.SourceID, source.ModifiedAt)
		return Workspace{}, errors.Join(err, deleteErr)
	}
	_, err = s.store.AttachSource(ctx, artifactstore.RootSourceAttachmentDraft{
		RootID:   root.RootID,
		SourceID: source.SourceID,
		Role:     RolePrimary,
		Priority: 1_000_000,
		Enabled:  true,
		Data:     json.RawMessage("{}"),
	})
	if err != nil {
		_, rootDeleteErr := s.store.DeleteRoot(ctx, root.RootID, root.ModifiedAt)
		sourceDeleteErr := s.store.DeleteSource(ctx, source.SourceID, source.ModifiedAt)
		return Workspace{}, errors.Join(err, rootDeleteErr, sourceDeleteErr)
	}
	workspace, err := s.GetWorkspace(ctx, root.RootID)
	if err != nil {
		return Workspace{}, err
	}
	if request.DiscoverImmediately {
		_, refreshErr := s.Refresh(ctx, root.RootID)
		if refreshErr != nil {
			return workspace, refreshErr
		}
		return s.GetWorkspace(ctx, root.RootID)
	}
	return workspace, nil
}

func (s *Service) CreateEmptyWorkspace(
	ctx context.Context,
	request EmptyWorkspaceRequest,
) (Workspace, error) {
	if strings.TrimSpace(request.DisplayName) == "" {
		return Workspace{}, fmt.Errorf("%w: displayName is required", ErrInvalidWorkspace)
	}
	data := RootData{
		Mode:                     RootModeEmpty,
		RootTrustReference:       request.TrustReference,
		DiscoveryPreferences:     request.Discovery,
		CapabilityProfileVersion: CapabilityProfileVersion,
	}
	raw, err := encodeRootData(data)
	if err != nil {
		return Workspace{}, err
	}
	root, err := s.store.CreateRoot(ctx, artifactstoreSpec.RootDraft{
		Kind:         RootKind,
		DisplayName:  request.DisplayName,
		Description:  request.Description,
		Enabled:      true,
		DataSchemaID: RootDataSchemaID,
		Data:         raw,
	})
	if err != nil {
		return Workspace{}, err
	}
	workspace := Workspace{Root: root, Data: data}
	if request.DiscoverImmediately {
		_, refreshErr := s.Refresh(ctx, root.RootID)
		if refreshErr != nil {
			return workspace, refreshErr
		}
		return s.GetWorkspace(ctx, root.RootID)
	}
	return workspace, nil
}

func (s *Service) GetWorkspace(
	ctx context.Context,
	rootID artifactstoreSpec.RootID,
) (Workspace, error) {
	root, err := s.store.GetRoot(ctx, rootID)
	if err != nil {
		return Workspace{}, err
	}
	if root.Kind != RootKind {
		return Workspace{}, fmt.Errorf("%w: root %q has kind %q", ErrNotWorkspace, rootID, root.Kind)
	}
	data, err := decodeRootData(root.Data)
	if err != nil {
		return Workspace{}, fmt.Errorf("%w: %w", ErrInvalidWorkspace, err)
	}
	attachments, err := s.store.ListRootSources(ctx, rootID)
	if err != nil {
		return Workspace{}, err
	}
	return Workspace{Root: root, Data: data, Attachments: attachments}, nil
}

func (s *Service) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	roots, err := s.store.ListRoots(ctx, false)
	if err != nil {
		return nil, err
	}
	out := make([]Workspace, 0)
	for _, root := range roots {
		if root.Kind != RootKind {
			continue
		}
		workspace, err := s.GetWorkspace(ctx, root.RootID)
		if err != nil {
			return nil, err
		}
		out = append(out, workspace)
	}
	return out, nil
}

func (s *Service) Refresh(
	ctx context.Context,
	rootID artifactstoreSpec.RootID,
) (RefreshResult, error) {
	workspace, err := s.GetWorkspace(ctx, rootID)
	if err != nil {
		return RefreshResult{}, err
	}
	input := DiscoveryInput{
		Workspace:   workspace,
		FrontendIDs: append([]artifactstoreSpec.FrontendID(nil), s.frontendIDs...),
	}
	bootstrapPlan, err := s.planner.BuildBootstrapPlan(ctx, input)
	if err != nil {
		return RefreshResult{}, err
	}
	bootstrap, err := s.store.ScanRoot(ctx, rootID, bootstrapPlan)
	if err != nil {
		return RefreshResult{}, err
	}
	result := RefreshResult{
		Workspace: workspace,
		Bootstrap: &bootstrap,
	}

	definitionPreferences, err := s.workspaceDefinitionPreferences(ctx, workspace)
	if err != nil {
		return result, err
	}
	input.DefinitionPreferences = definitionPreferences
	expandedPlan, err := s.planner.BuildExpandedPlan(ctx, input)
	if err != nil {
		return result, err
	}
	published := bootstrap
	if len(workspace.Attachments) != 0 {
		published, err = s.store.ScanRoot(ctx, rootID, expandedPlan)
		if err != nil {
			return result, err
		}
	}
	result.Published = published
	result.Diagnostics = append(
		result.Diagnostics,
		bootstrap.Diagnostics...,
	)
	if published.Generation.Generation != bootstrap.Generation.Generation {
		result.Diagnostics = append(result.Diagnostics, published.Diagnostics...)
	}

	resources, err := s.store.ListCatalogResourcesForRoot(ctx, rootID)
	if err != nil {
		return result, err
	}
	collections, err := s.ensureCollections(ctx, rootID, resources)
	if err != nil {
		return result, err
	}
	syncResult, err := s.store.SyncRecords(
		ctx,
		rootID,
		&recordSyncPolicy{
			descriptors: s.descriptors,
			collections: collections,
		},
	)
	if err != nil {
		return result, err
	}
	result.Sync = syncResult
	result.Diagnostics = append(result.Diagnostics, syncResult.Diagnostics...)
	catalog, err := s.Catalog(ctx, rootID)
	if err != nil {
		return result, err
	}
	result.Catalog = catalog
	result.Workspace = catalog.Workspace
	return result, nil
}

func (s *Service) workspaceDefinitionPreferences(
	ctx context.Context,
	workspace Workspace,
) (DiscoveryPreferences, error) {
	if workspace.Data.PrimarySourceID == "" {
		return DiscoveryPreferences{}, nil
	}
	resources, err := s.store.ListCatalogResourcesForRoot(ctx, workspace.Root.RootID)
	if err != nil {
		return DiscoveryPreferences{}, err
	}
	matches := make([]artifactstoreSpec.CatalogResource, 0, 1)
	for _, resource := range resources {
		if resource.SourceID == workspace.Data.PrimarySourceID &&
			resource.Kind == KindWorkspaceDefinition &&
			resource.State == artifactstoreSpec.CatalogStateValid &&
			resource.CurrentDefinitionDigest != nil {
			matches = append(matches, resource)
		}
	}
	if len(matches) > 1 {
		return DiscoveryPreferences{}, fmt.Errorf(
			"%w: primary source contains %d definitions",
			ErrAmbiguousWorkspaceDefinition,
			len(matches),
		)
	}
	if len(matches) == 0 {
		return DiscoveryPreferences{}, nil
	}
	definition, err := s.store.GetDefinitionByDigest(ctx, *matches[0].CurrentDefinitionDigest)
	if err != nil {
		return DiscoveryPreferences{}, fmt.Errorf(
			"%w: %w",
			ErrWorkspaceDefinitionUnavailable,
			err,
		)
	}
	var document struct {
		Discovery DiscoveryPreferences `json:"discovery"`
	}
	if err := json.Unmarshal(definition.DefinitionJSON, &document); err != nil {
		return DiscoveryPreferences{}, fmt.Errorf(
			"%w: decode discovery preferences: %w",
			ErrInvalidWorkspace,
			err,
		)
	}
	if err := validateDiscoveryPreferences(document.Discovery); err != nil {
		return DiscoveryPreferences{}, fmt.Errorf("%w: %w", ErrInvalidWorkspace, err)
	}
	return document.Discovery, nil
}

func (s *Service) ensureCollections(
	ctx context.Context,
	rootID artifactstoreSpec.RootID,
	resources []artifactstoreSpec.CatalogResource,
) (map[artifactstoreSpec.ArtifactKind]artifactstoreSpec.CollectionID, error) {
	kinds := map[artifactstoreSpec.ArtifactKind]struct{}{}
	for _, resource := range resources {
		if resource.State != artifactstoreSpec.CatalogStateValid {
			continue
		}
		if _, known := s.descriptors[resource.Kind]; known {
			kinds[resource.Kind] = struct{}{}
		}
	}
	ordered := make([]artifactstoreSpec.ArtifactKind, 0, len(kinds))
	for kind := range kinds {
		ordered = append(ordered, kind)
	}
	slices.Sort(ordered)

	out := make(map[artifactstoreSpec.ArtifactKind]artifactstoreSpec.CollectionID, len(ordered))
	for _, kind := range ordered {
		descriptor := s.descriptors[kind]
		data, err := json.Marshal(CollectionData{ArtifactKind: kind})
		if err != nil {
			return nil, err
		}
		collection, err := s.store.EnsureBaseCollection(ctx, artifactstoreSpec.CollectionDraft{
			RootID:       rootID,
			Kind:         CollectionKind,
			Slug:         descriptor.CollectionSlug,
			DisplayName:  descriptor.CollectionDisplayName,
			Enabled:      true,
			DataSchemaID: CollectionDataSchemaID,
			Data:         data,
		})
		if err != nil {
			return nil, err
		}
		out[kind] = collection.CollectionID
	}
	return out, nil
}

func encodeRootData(data RootData) (json.RawMessage, error) {
	if err := validateRootData(data); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	canonical, err := baseutils.CanonicalizeJSON(raw)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

type recordSyncPolicy struct {
	descriptors map[artifactstoreSpec.ArtifactKind]KindDescriptor
	collections map[artifactstoreSpec.ArtifactKind]artifactstoreSpec.CollectionID
}

func (p *recordSyncPolicy) DeriveRecord(
	_ context.Context,
	_ artifactstoreSpec.ArtifactRoot,
	resource artifactstoreSpec.CatalogResource,
	definition artifactstoreSpec.CanonicalDefinition,
) (artifactstoreSpec.RecordDerivation, bool, []artifactstoreSpec.Diagnostic) {
	if p == nil {
		return artifactstoreSpec.RecordDerivation{}, false, nil
	}
	if _, known := p.descriptors[resource.Kind]; !known {
		return artifactstoreSpec.RecordDerivation{}, false, nil
	}
	collectionID, exists := p.collections[resource.Kind]
	if !exists {
		return artifactstoreSpec.RecordDerivation{}, false, workspaceDiagnostics(
			"workspace.synchronization.collection",
			fmt.Sprintf("collection for artifact kind %q is unavailable", resource.Kind),
		)
	}
	return artifactstoreSpec.RecordDerivation{
		CollectionID: &collectionID,
		Name:         workspaceRecordName(definition.LogicalName, definition.Digest),
		Version:      artifactstoreSpec.RecordVersion(definition.LogicalVersion),
		Enabled:      true,
		Data:         json.RawMessage("{}"),
	}, true, nil
}

func workspaceRecordName(
	logicalName artifactstoreSpec.LogicalName,
	digest artifactstoreSpec.Digest,
) artifactstoreSpec.RecordName {
	var builder strings.Builder
	hyphen := false
	runes := 0
	for _, character := range strings.ToLower(string(logicalName)) {
		switch {
		case unicode.IsLetter(character), unicode.IsDigit(character):
			builder.WriteRune(character)
			hyphen = false
			runes++
		default:
			if builder.Len() > 0 && !hyphen {
				builder.WriteByte('-')
				hyphen = true
				runes++
			}
		}
		if runes >= artifactstoreSpec.MaxSlugRunes {
			break
		}
	}
	value := strings.Trim(builder.String(), "-")
	if value == "" {
		hash := strings.TrimPrefix(string(digest), "sha256:")
		if len(hash) > 12 {
			hash = hash[:12]
		}
		value = "artifact-" + hash
	}
	return artifactstoreSpec.RecordName(value)
}

func defaultKindDescriptors() map[artifactstoreSpec.ArtifactKind]KindDescriptor {
	values := []KindDescriptor{
		{KindWorkspaceDefinition, "workspace.definition.v1", "workspace", "Workspace"},
		{KindAgentDefinition, "agent.definition.v1", "agents", "Agents"},
		{KindSkillDefinition, "skill.definition.v1", "skills", "Skills"},
		{KindModelDefinition, "model.definition.v1", "models", "Models"},
		{KindMCPServerDefinition, "mcp.server.definition.v1", "mcp-servers", "MCP Servers"},
		{KindToolDefinition, "tool.definition.v1", "tools", "Tools"},
		{KindInstructionDocument, "instruction.document.v1", "instructions", "Instructions"},
		{KindContextDocument, "context.document.v1", "context", "Context"},
	}
	out := make(map[artifactstoreSpec.ArtifactKind]KindDescriptor, len(values))
	for _, value := range values {
		out[value.Kind] = value
	}
	return out
}

func cloneDescriptors(
	input map[artifactstoreSpec.ArtifactKind]KindDescriptor,
) map[artifactstoreSpec.ArtifactKind]KindDescriptor {
	out := make(map[artifactstoreSpec.ArtifactKind]KindDescriptor, len(input))
	maps.Copy(out, input)
	return out
}

var _ artifactstoreSpec.RecordSyncPolicy = (*recordSyncPolicy)(nil)
