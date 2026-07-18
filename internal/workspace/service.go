package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/baseutils"
	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

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
	descriptor, known := p.descriptors[resource.Kind]
	if !known {
		return artifactstoreSpec.RecordDerivation{}, false, nil
	}
	if definition.SchemaID != descriptor.DefinitionSchemaID {
		return artifactstoreSpec.RecordDerivation{}, false, workspaceDiagnostics(
			"workspace.synchronization.schema",
			fmt.Sprintf(
				"definition schema %q does not match Workspace schema %q for kind %q",
				definition.SchemaID,
				descriptor.DefinitionSchemaID,
				resource.Kind,
			),
		)
	}
	if err := validateWorkspaceCanonicalDefinition(definition); err != nil {
		return artifactstoreSpec.RecordDerivation{}, false, workspaceDiagnostics(
			"workspace.synchronization.definition",
			err.Error(),
		)
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
		Name: workspaceRecordName(
			definition.LogicalName,
			resource.SourceID,
			resource.Locator,
			resource.SubresourceLocator,
		),
		Version: artifactstoreSpec.RecordVersion(definition.LogicalVersion),
		Enabled: true,
		Data:    json.RawMessage("{}"),
	}, true, nil
}

type Service struct {
	store       ArtifactStore
	planner     DiscoveryPlanner
	descriptors map[artifactstoreSpec.ArtifactKind]KindDescriptor
	projectors  map[artifactstoreSpec.ArtifactKind]ResourceProjector
	frontendIDs []artifactstoreSpec.FrontendID
}

type serviceConfig struct {
	planner         DiscoveryPlanner
	descriptors     map[artifactstoreSpec.ArtifactKind]KindDescriptor
	projectors      map[artifactstoreSpec.ArtifactKind]ResourceProjector
	extraFrontends  []artifactstoreSpec.ArtifactFrontend
	yamlDecoder     YAMLDecoder
	versionMatchers map[artifactstoreSpec.ArtifactKind]artifactstoreSpec.ArtifactVersionMatcher
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
		// Explicit options replace defaults so application composition can swap
		// schemas without retaining a hidden compatibility descriptor.
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
		// A caller-supplied projector intentionally replaces the default.
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

func WithArtifactVersionMatcher(matcher artifactstoreSpec.ArtifactVersionMatcher) Option {
	return func(config *serviceConfig) error {
		if matcher == nil || matcher.Kind() == "" {
			return fmt.Errorf("%w: artifact version matcher is invalid", ErrInvalidWorkspace)
		}
		config.versionMatchers[matcher.Kind()] = matcher
		return nil
	}
}

func NewService(store ArtifactStore, options ...Option) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: artifact store is nil", ErrInvalidWorkspace)
	}
	config := serviceConfig{
		descriptors:     defaultKindDescriptors(),
		projectors:      defaultProjectors(),
		yamlDecoder:     DecodeYAML,
		versionMatchers: map[artifactstoreSpec.ArtifactKind]artifactstoreSpec.ArtifactVersionMatcher{},
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}
	for kind := range config.projectors {
		if _, exists := config.descriptors[kind]; !exists {
			return nil, fmt.Errorf(
				"%w: projector kind %q has no descriptor",
				ErrInvalidWorkspace,
				kind,
			)
		}
	}
	seenFrontendIDs := map[artifactstoreSpec.FrontendID]struct{}{
		NativeFrontendID: {},
		artifactstoreSpec.PortableDefinitionFrontendID: {},
	}
	for _, extra := range config.extraFrontends {
		if extra == nil || strings.TrimSpace(string(extra.ID())) == "" {
			return nil, fmt.Errorf(
				"%w: Workspace frontend has an empty ID",
				ErrInvalidWorkspace,
			)
		}
		if _, duplicate := seenFrontendIDs[extra.ID()]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate Workspace frontend %q",
				ErrInvalidWorkspace,
				extra.ID(),
			)
		}
		seenFrontendIDs[extra.ID()] = struct{}{}
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
	if err := store.RegisterDependencyResolver(workspaceDependencyResolver{}); err != nil {
		return nil, fmt.Errorf("register Workspace dependency resolver: %w", err)
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
	kinds := make([]artifactstoreSpec.ArtifactKind, 0, len(config.descriptors))
	for kind := range config.descriptors {
		kinds = append(kinds, kind)
	}
	slices.Sort(kinds)
	for _, kind := range kinds {
		matcher := config.versionMatchers[kind]
		if matcher == nil {
			matcher = exactVersionMatcher{kind: kind}
		}
		if err := store.RegisterArtifactVersionMatcher(matcher); err != nil {
			return nil, fmt.Errorf("register Workspace version matcher for %q: %w", kind, err)
		}
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
		projectors:  maps.Clone(config.projectors),
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
	attachment, err := s.store.AttachSource(ctx, artifactstore.RootSourceAttachmentDraft{
		RootID:   root.RootID,
		SourceID: source.SourceID,
		Role:     RolePrimary,
		Priority: workspacePrimaryPriority,
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
		detachErr := s.store.DetachSource(
			context.WithoutCancel(ctx),
			root.RootID,
			source.SourceID,
			attachment.ModifiedAt,
		)
		_, rootDeleteErr := s.store.DeleteRoot(context.WithoutCancel(ctx), root.RootID, root.ModifiedAt)
		sourceDeleteErr := s.store.DeleteSource(context.WithoutCancel(ctx), source.SourceID, source.ModifiedAt)
		return Workspace{}, errors.Join(err, detachErr, rootDeleteErr, sourceDeleteErr)
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

// MountEmbeddedSource creates an embedded-fs-directory source and attaches it
// to a Workspace. The provider must already have been registered while the
// Artifact Store composition was being built.
func (s *Service) MountEmbeddedSource(
	ctx context.Context,
	request EmbeddedSourceAttachmentRequest,
) (Workspace, error) {
	if strings.TrimSpace(request.DisplayName) == "" ||
		strings.TrimSpace(request.ProviderKey) == "" ||
		request.RootID == "" {
		return Workspace{}, fmt.Errorf(
			"%w: rootID, displayName, and providerKey are required",
			ErrInvalidWorkspace,
		)
	}
	role := request.Role
	if role == "" {
		role = RoleBuiltIn
	}
	if role == RolePrimary {
		return Workspace{}, fmt.Errorf(
			"%w: an embedded source cannot be a primary filesystem workspace source",
			ErrInvalidWorkspace,
		)
	}
	rootLocator := request.RootLocator
	if rootLocator == "" {
		rootLocator = "."
	}
	config, err := json.Marshal(artifactstoreSpec.EmbeddedFSDirectorySourceConfig{
		ProviderKey: request.ProviderKey,
		RootLocator: rootLocator,
	})
	if err != nil {
		return Workspace{}, err
	}
	source, err := s.store.CreateSource(ctx, artifactstoreSpec.SourceDraft{
		Kind:           artifactstoreSpec.SourceKindEmbeddedFSDirectory,
		DisplayName:    request.DisplayName,
		Enabled:        true,
		ConfigSchemaID: artifactstoreSpec.EmbeddedFSDirectoryConfigSchemaID,
		Config:         config,
	})
	if err != nil {
		return Workspace{}, err
	}
	workspace, attachErr := s.AttachSource(ctx, AttachSourceRequest{
		RootID:              request.RootID,
		SourceID:            source.SourceID,
		Role:                role,
		Priority:            request.Priority,
		AttachmentData:      request.AttachmentData,
		DiscoverImmediately: false,
	})
	if attachErr != nil {
		deleteErr := s.store.DeleteSource(ctx, source.SourceID, source.ModifiedAt)
		return Workspace{}, errors.Join(attachErr, deleteErr)
	}
	if !request.DiscoverImmediately {
		return workspace, nil
	}
	if _, err := s.Refresh(ctx, request.RootID); err != nil {
		return workspace, err
	}
	return s.GetWorkspace(ctx, request.RootID)
}

// AttachSource attaches an existing Artifact Store source to a Workspace.
// Source configuration and source-kind validation remain Artifact Store
// responsibilities. Workspace owns only the typed attachment role and data.
func (s *Service) AttachSource(
	ctx context.Context,
	request AttachSourceRequest,
) (Workspace, error) {
	if s == nil || s.store == nil {
		return Workspace{}, fmt.Errorf("%w: service is not configured", ErrInvalidWorkspace)
	}
	if request.RootID == "" || request.SourceID == "" || request.Role == "" {
		return Workspace{}, fmt.Errorf(
			"%w: rootID, sourceID, and role are required",
			ErrInvalidWorkspace,
		)
	}
	if _, err := s.GetWorkspace(ctx, request.RootID); err != nil {
		return Workspace{}, err
	}
	dataSchemaID, data, err := encodeAttachmentData(request.AttachmentData)
	if err != nil {
		return Workspace{}, err
	}
	attachment, err := s.store.AttachSource(ctx, artifactstore.RootSourceAttachmentDraft{
		RootID:       request.RootID,
		SourceID:     request.SourceID,
		Role:         request.Role,
		Priority:     request.Priority,
		Enabled:      true,
		DataSchemaID: dataSchemaID,
		Data:         data,
	})
	if err != nil {
		detachErr := s.store.DetachSource(
			context.WithoutCancel(ctx),
			request.RootID,
			request.SourceID,
			attachment.ModifiedAt,
		)
		return Workspace{}, errors.Join(err, detachErr)
	}
	workspace, err := s.GetWorkspace(ctx, request.RootID)
	if err != nil {
		return Workspace{}, err
	}
	if !request.DiscoverImmediately {
		return workspace, nil
	}
	if _, err := s.Refresh(ctx, request.RootID); err != nil {
		return workspace, err
	}
	return s.GetWorkspace(ctx, request.RootID)
}

// DetachSource removes only the Workspace root/source relationship. It does
// not delete the source registration, source catalog, records, or definitions.
func (s *Service) DetachSource(
	ctx context.Context,
	rootID artifactstoreSpec.RootID,
	sourceID artifactstoreSpec.SourceID,
	discoverImmediately bool,
) (Workspace, error) {
	if _, err := s.GetWorkspace(ctx, rootID); err != nil {
		return Workspace{}, err
	}
	attachment, err := s.store.GetRootSourceAttachment(ctx, rootID, sourceID)
	if err != nil {
		return Workspace{}, err
	}
	if attachment.Role == RolePrimary {
		return Workspace{}, fmt.Errorf(
			"%w: the primary source cannot be detached from a filesystem workspace",
			ErrInvalidWorkspace,
		)
	}
	if err := s.store.DetachSource(ctx, rootID, sourceID, attachment.ModifiedAt); err != nil {
		return Workspace{}, err
	}
	workspace, err := s.GetWorkspace(ctx, rootID)
	if err != nil {
		return Workspace{}, err
	}
	if !discoverImmediately {
		return workspace, nil
	}
	if _, err := s.Refresh(ctx, rootID); err != nil {
		return workspace, err
	}
	return s.GetWorkspace(ctx, rootID)
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

func (s *Service) GetWorkspace(
	ctx context.Context,
	rootID artifactstoreSpec.RootID,
) (Workspace, error) {
	if s == nil || s.store == nil {
		return Workspace{}, fmt.Errorf("%w: service is not configured", ErrInvalidWorkspace)
	}
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
	if err := validateRootData(data); err != nil {
		return Workspace{}, fmt.Errorf("%w: %w", ErrInvalidWorkspace, err)
	}

	attachments, err := s.store.ListRootSources(ctx, rootID)
	if err != nil {
		return Workspace{}, err
	}
	if err := validateWorkspaceAttachmentSet(data, attachments); err != nil {
		return Workspace{}, fmt.Errorf("%w: %w", ErrInvalidWorkspace, err)
	}

	sources := make([]AttachedSource, 0, len(attachments))

	hook := rootKindHook{}
	for _, attachment := range attachments {
		if err := workspaceDiagnosticsError(
			"workspace attachment",
			hook.ValidateSourceAttachment(ctx, root, attachment),
		); err != nil {
			return Workspace{}, err
		}
		source, err := s.store.GetSource(ctx, attachment.SourceID)
		if err != nil {
			return Workspace{}, err
		}
		if err := workspaceDiagnosticsError(
			"workspace attachment source",
			hook.ValidateSourceAttachmentSource(
				ctx,
				root,
				attachment,
				source,
			),
		); err != nil {
			return Workspace{}, err
		}
		sources = append(sources, AttachedSource{
			Attachment: attachment,
			Source:     source,
		})
	}

	return Workspace{Root: root, Data: data, Attachments: attachments, Sources: sources}, nil
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
		collectionID, err := s.ensureCollectionForKind(ctx, rootID, kind)
		if err != nil {
			return nil, err
		}
		out[kind] = collectionID
	}
	return out, nil
}

func (s *Service) ensureCollectionForKind(
	ctx context.Context,
	rootID artifactstoreSpec.RootID,
	kind artifactstoreSpec.ArtifactKind,
) (artifactstoreSpec.CollectionID, error) {
	descriptor, known := s.descriptors[kind]
	if !known {
		return "", fmt.Errorf("%w: unsupported artifact kind %q", ErrProjectionUnavailable, kind)
	}
	data, err := json.Marshal(CollectionData{ArtifactKind: kind})
	if err != nil {
		return "", err
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
		return "", err
	}
	var collectionData CollectionData
	if collection.DataSchemaID != CollectionDataSchemaID ||
		decodeStrictJSONObject(collection.Data, &collectionData, true) != nil ||
		collectionData.ArtifactKind != kind {
		return "", fmt.Errorf(
			"%w: collection %q is incompatible with artifact kind %q",
			artifactstoreSpec.ErrConflict,
			collection.CollectionID,
			kind,
		)
	}
	return collection.CollectionID, nil
}

func workspaceDiagnosticsError(
	scope string,
	diagnostics []artifactstoreSpec.Diagnostic,
) error {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == artifactstoreSpec.DiagnosticSeverityError {
			return fmt.Errorf(
				"%w: %s: %s",
				ErrInvalidWorkspace,
				scope,
				diagnostic.Message,
			)
		}
	}
	return nil
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

func encodeAttachmentData(data AttachmentData) (artifactstoreSpec.SchemaID, json.RawMessage, error) {
	if data.Recursive == nil && data.Authoritative == nil {
		return "", json.RawMessage("{}"), nil
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return "", nil, err
	}
	canonical, err := baseutils.CanonicalizeJSON(raw)
	if err != nil {
		return "", nil, err
	}
	return AttachmentDataSchemaID, json.RawMessage(canonical), nil
}

func workspaceRecordName(
	logicalName artifactstoreSpec.LogicalName,
	sourceID artifactstoreSpec.SourceID,
	locator artifactstoreSpec.SourceLocator,
	subresource artifactstoreSpec.SubresourceLocator,
) artifactstoreSpec.RecordName {
	occurrenceDigest := baseutils.DigestBytes([]byte(
		string(sourceID) + "\x00" + string(locator) + "\x00" + string(subresource),
	))
	hash := strings.TrimPrefix(string(occurrenceDigest), "sha256:")[:workspaceRecordNameHashLength]
	suffix := "-" + hash
	maximumBaseRunes := artifactstoreSpec.MaxSlugRunes - len(suffix)
	var builder strings.Builder
	hyphen := false
	runes := 0
	for _, character := range strings.ToLower(string(logicalName)) {
		switch {
		case character >= 'a' && character <= 'z', character >= '0' && character <= '9':
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
		if runes >= maximumBaseRunes {
			break
		}
	}
	value := strings.Trim(builder.String(), "-")
	if value == "" {
		value = workspaceRecordNameFallback
	}
	return artifactstoreSpec.RecordName(value + suffix)
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
