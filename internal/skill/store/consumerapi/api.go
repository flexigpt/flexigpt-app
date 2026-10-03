package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	managedpackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managedpackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

type API struct {
	cat              catalog.API
	sources          source.API
	discovery        refreshFlow.API
	artifacts        artifact.API
	resources        resourceFlow.API
	managedArtifacts managedpackageFlow.API
	protection       root.ProtectionAPI
	definitions      definition.API
	collections      *collection.API

	declarationResolver *resolve.Resolver
}

func New(
	sources source.API,
	discovery refreshFlow.API,
	artifacts artifact.API,
	resources resourceFlow.API,
	managedArtifacts managedpackageFlow.API,
	protection root.ProtectionAPI,
	cat catalog.API,
	definitions definition.API,
	options ...Option,
) (*API, error) {
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		resources == nil ||
		managedArtifacts == nil ||
		protection == nil || cat == nil || definitions == nil {
		return nil, fmt.Errorf(
			"%w: Skill Store dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	config := apiOptions{}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	output := &API{
		sources:          sources,
		discovery:        discovery,
		artifacts:        artifacts,
		resources:        resources,
		managedArtifacts: managedArtifacts,
		protection:       protection,
		cat:              cat,
		definitions:      definitions,
	}
	locators, err := resolve.NewProviderLocatorResolver(
		config.locatorResolvers,
		skillLocatorRuntime{cat: cat},
	)
	if err != nil {
		return nil, err
	}
	graphResolver, err := resolve.NewWithOptions(
		resolve.ResolverOptions{
			Artifacts:            artifacts,
			Catalog:              cat,
			SourceEntries:        resources,
			Locators:             locators,
			FallbackProviders:    config.fallbackProviders,
			TargetMappers:        config.targetMappers,
			ProtectedBuiltinRoot: documentTopology.BuiltinRootID(),
			Limits:               resolve.DefaultLimits(),
		},
	)
	if err != nil {
		return nil, err
	}
	collections, err := collection.NewWithResolver(
		artifacts,
		cat,
		sources,
		discovery,
		managedArtifacts,
		definitions,
		graphResolver,
		collection.SkillDomainPolicy(),
	)
	if err != nil {
		return nil, err
	}
	output.collections = collections
	output.declarationResolver = graphResolver
	return output, nil
}

func SkillDiscoverySpec(
	r spec.Locator,
) (sourceModel.DiscoverySpec, error) {
	if r == "" {
		r = "."
	}
	return documentTopology.DiscoverySpecAtForUse(
		documentTopology.DiscoveryUseSkill,
		r,
	)
}

func (a *API) RegisterSkillDirectory(
	ctx context.Context,
	request SkillDirectoryRegistration,
) (sourceModel.Summary, error) {
	if err := request.RootID.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := spec.ValidateRequiredText(
		"Skill Source display name",
		request.SourceDisplayName,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return sourceModel.Summary{}, err
	}

	rootPath, err := fsdir.NormalizeFilesystemSourceRoot(
		request.RootPath,
		"Skill Source root path",
	)
	if err != nil {
		return sourceModel.Summary{}, err
	}

	discovery, err := SkillDiscoverySpec(".")
	if err != nil {
		return sourceModel.Summary{}, err
	}
	config, err := json.Marshal(struct {
		RootPath string `json:"rootPath"`
	}{
		RootPath: rootPath,
	})
	if err != nil {
		return sourceModel.Summary{}, err
	}

	return refreshFlow.EnsureAndRefreshSource(
		ctx,
		a.sources,
		a.discovery,
		refreshFlow.EnsureAndRefreshSourceRequest{
			RootID: request.RootID,
			Draft: sourceModel.Draft{
				ID: sourceModel.SourceID(uuidutil.NewUUIDv7()),
				StorageKey: fsdir.FilesystemSourceStorageKey(
					"skill-path",
					rootPath,
				),
				Kind:        sourceModel.SourceKindFilesystemDirectory,
				DisplayName: request.SourceDisplayName,
				Enabled:     true,
				Config:      config,
				Discovery:   discovery,
			},
		},
	)
}

func (a *API) RefreshSkillSource(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) error {
	if err := rootID.Validate(); err != nil {
		return err
	}
	if err := sourceID.Validate(); err != nil {
		return err
	}
	_, err := a.discovery.RefreshSource(ctx, rootID, sourceID)
	return err
}

func (a *API) GetSkill(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (artifactModel.Artifact, error) {
	value, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if !skillDomain.IsSkillKind(value.Kind) {
		return artifactModel.Artifact{}, fmt.Errorf(
			"%w: Artifact %q is not a Skill",
			spec.ErrNotFound,
			ref.ArtifactID,
		)
	}
	return value, nil
}

func (a *API) SetSkillEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifactModel.Artifact, error) {
	if _, err := a.GetSkill(ctx, ref); err != nil {
		return artifactModel.Artifact{}, err
	}
	return a.artifacts.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
}

func (a *API) CreateManagedSkill(
	ctx context.Context,
	request ManagedSkillCreateRequest,
) (ManagedSkillCreateResult, error) {
	if a == nil || a.collections == nil {
		return ManagedSkillCreateResult{}, spec.ErrClosed
	}
	if err := request.Collection.Validate(); err != nil {
		return ManagedSkillCreateResult{}, err
	}
	if request.ExpectedCollectionRevision == 0 {
		return ManagedSkillCreateResult{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			spec.ErrInvalid,
		)
	}
	if err := a.requireMutable(
		ctx,
		request.Collection.RootID,
		false,
	); err != nil {
		return ManagedSkillCreateResult{}, err
	}
	if err := spec.LogicalName(request.SkillName).Validate(); err != nil {
		return ManagedSkillCreateResult{}, err
	}

	files, skillMD, err := skillDomain.NormalizeManagedSkillFiles(
		request.SKILLMD,
		request.Files,
	)
	if err != nil {
		return ManagedSkillCreateResult{}, err
	}
	definitionValue, _, err := skillDomain.DecodeSkillDocument(
		skillMD,
		request.SkillName,
	)
	if err != nil {
		return ManagedSkillCreateResult{}, err
	}
	if definitionValue.LogicalName != spec.LogicalName(request.SkillName) {
		return ManagedSkillCreateResult{}, fmt.Errorf(
			"%w: SKILL.md name does not match requested Skill name",
			spec.ErrInvalid,
		)
	}

	packageAddress, err := skillDomain.ManagedPackageAddressForSkill(
		definitionValue.LogicalName,
		definitionValue.LogicalVersion,
	)
	if err != nil {
		return ManagedSkillCreateResult{}, err
	}
	storageFiles, err := skillDomain.ManagedSkillStorageFiles(
		packageAddress,
		files,
	)
	if err != nil {
		return ManagedSkillCreateResult{}, err
	}
	locator, err := skillDomain.ManagedPackageLocatorForSkill(
		packageAddress,
	)
	if err != nil {
		return ManagedSkillCreateResult{}, err
	}

	membership, err := a.collections.EnsureMemberForCollectionSource(
		ctx,
		collection.EnsureMemberForCollectionSourceRequest{
			Collection:       request.Collection,
			ExpectedRevision: request.ExpectedCollectionRevision,
			Type:             declaration.TypeSkill,
			Name:             definitionValue.LogicalName,
			Locator:          locator,
		},
	)
	if err != nil {
		return ManagedSkillCreateResult{}, err
	}

	result := ManagedSkillCreateResult{
		Collection:        membership.Collection,
		MembershipCreated: membership.Created,
	}
	rootID := membership.Collection.Artifact.RootID
	sourceID := membership.Collection.Artifact.Binding.SourceID
	decoderID, err := documentTopology.DefaultDocumentDecoderID(
		documentTopology.DocumentUseSkillPackage,
	)
	if err != nil {
		return result, err
	}
	if _, err := a.collections.EnsureManagedDeclarationDiscovery(
		ctx,
		rootID,
		sourceID,
		locator,
		decoderID,
	); err != nil {
		return result, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		artifactModel.PublishArtifactRequest{
			RootID: rootID,
			Binding: artifactModel.SourceBinding{
				SourceID: sourceID,
				Locator:  locator,
			},
			ExpectedKind:        skillDomain.SkillArtifactKind,
			ExpectedLogicalName: definitionValue.LogicalName,
			ExpectedDefinition:  definitionValue.Digest,
			Package: sourceModel.ManagedPackagePublication{
				Address: packageAddress,
				Files:   storageFiles,
			},
		},
	)
	if err != nil {
		return result, err
	}

	value := published.Artifact
	result.Artifact = value
	result.Address = value.Address()
	if value.Enabled != request.Enabled {
		value, err = a.artifacts.SetEnabled(
			ctx,
			value.Ref(),
			value.Revision,
			request.Enabled,
		)
		if err != nil {
			return result, err
		}
		result.Artifact = value
		result.Address = value.Address()
	}
	return result, nil
}

// ReplaceManagedSkill replaces the complete managed Skill package for an
// existing managed Skill Artifact. The Skill Artifact identity, logical name,
// package address, Collection membership, and managed Source remain stable.
func (a *API) ReplaceManagedSkill(
	ctx context.Context,
	request ManagedSkillReplaceRequest,
) (ManagedSkillReplaceResult, error) {
	if a == nil || a.collections == nil {
		return ManagedSkillReplaceResult{}, spec.ErrClosed
	}
	if err := request.Collection.Validate(); err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	if err := request.Artifact.Validate(); err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	if request.Collection.RootID != request.Artifact.RootID {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill Artifact belongs to another Root",
			spec.ErrInvalid,
		)
	}
	if request.ExpectedCollectionRevision == 0 {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			spec.ErrInvalid,
		)
	}
	if request.ExpectedArtifactRevision == 0 {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: expected Skill Artifact revision is required",
			spec.ErrInvalid,
		)
	}
	if err := spec.LogicalName(request.SkillName).Validate(); err != nil {
		return ManagedSkillReplaceResult{}, err
	}

	collectionView, err := a.collections.Read(ctx, request.Collection)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	if collectionView.Artifact.Revision != request.ExpectedCollectionRevision {
		return ManagedSkillReplaceResult{}, spec.ErrConflict
	}
	if !collectionView.Editable &&
		!collectionView.Baseline {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill Collection is read-only",
			spec.ErrUnsupported,
		)
	}

	current, err := a.GetSkill(ctx, request.Artifact)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	if current.Revision != request.ExpectedArtifactRevision {
		return ManagedSkillReplaceResult{}, spec.ErrConflict
	}
	if current.Binding.SubresourceLocator != "" ||
		!skillDomain.IsSkillDefinitionFile(current.Binding.Locator) {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill is not a replaceable managed Skill package",
			spec.ErrUnsupported,
		)
	}
	if current.Binding.SourceID != collectionView.Artifact.Binding.SourceID {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill is not owned by this Collection Source",
			spec.ErrUnsupported,
		)
	}

	sourceValue, err := a.sources.Get(
		ctx,
		current.RootID,
		current.Binding.SourceID,
	)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	if sourceValue.Kind != sourceModel.SourceKindManagedDirectory {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill is not backed by a managed Source",
			spec.ErrUnsupported,
		)
	}

	memberships, err := a.collections.ListMembershipsForArtifact(
		ctx,
		request.Artifact,
	)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	memberFound := false
	for _, membership := range memberships {
		if membership.Collection != request.Collection ||
			!membership.ResolvedToArtifact {
			continue
		}
		memberFound = true
		break
	}
	if !memberFound {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill is not a direct member of the requested Collection",
			spec.ErrReferenceUnresolved,
		)
	}

	files, skillMD, err := skillDomain.NormalizeManagedSkillFiles(
		request.SKILLMD,
		request.Files,
	)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	definitionValue, _, err := skillDomain.DecodeSkillDocument(
		skillMD,
		request.SkillName,
	)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	if definitionValue.LogicalName != current.LogicalName {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: replacement Skill name must remain %q",
			spec.ErrInvalid,
			current.LogicalName,
		)
	}

	currentAddress, err := skillDomain.ManagedPackageAddressFromSkillLocator(
		current.Binding.Locator,
	)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	requestedAddress, err := skillDomain.ManagedPackageAddressForSkill(
		definitionValue.LogicalName,
		definitionValue.LogicalVersion,
	)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	if requestedAddress != currentAddress {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: replacement Skill cannot change managed package identity",
			spec.ErrInvalid,
		)
	}

	storageFiles, err := skillDomain.ManagedSkillStorageFiles(
		currentAddress,
		files,
	)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}

	inspection, err := a.discovery.InspectSource(
		ctx,
		current.RootID,
		current.Binding.SourceID,
	)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	if !inspection.IsCurrent() {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: managed Skill Source requires refresh",
			spec.ErrRefreshRequired,
		)
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		artifactModel.PublishArtifactRequest{
			RootID: current.RootID,
			Binding: artifactModel.SourceBinding{
				SourceID: current.Binding.SourceID,
				Locator:  current.Binding.Locator,
			},
			ExpectedKind:        skillDomain.SkillArtifactKind,
			ExpectedLogicalName: current.LogicalName,
			ExpectedDefinition:  definitionValue.Digest,
			Package: sourceModel.ManagedPackagePublication{
				Address:            currentAddress,
				ExpectedGeneration: inspection.State.SourceGeneration,
				Files:              storageFiles,
			},
			AllowPackageReplacement: true,
		},
	)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	if published.Artifact.Ref() != request.Artifact {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: replacement published another Skill Artifact",
			spec.ErrConflict,
		)
	}

	updated := published.Artifact
	if updated.Enabled != request.Enabled {
		updated, err = a.artifacts.SetEnabled(
			ctx,
			updated.Ref(),
			updated.Revision,
			request.Enabled,
		)
		if err != nil {
			return ManagedSkillReplaceResult{}, err
		}
	}

	collectionView, err = a.collections.Read(ctx, request.Collection)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}

	return ManagedSkillReplaceResult{
		Artifact:   updated,
		Address:    updated.Address(),
		Collection: collectionView,
	}, nil
}

func (a *API) GetManagedSkillDocument(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (skillDomain.ManagedSkillDocument, error) {
	if a == nil || a.resources == nil {
		return skillDomain.ManagedSkillDocument{}, spec.ErrClosed
	}
	return resourceFlow.WithVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (skillDomain.ManagedSkillDocument, error) {
			return a.getManagedSkillDocument(sessionCtx, ref)
		},
	)
}

func (a *API) PurgeSkill(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Skill Artifact revision is required",
			spec.ErrInvalid,
		)
	}
	value, err := a.GetSkill(ctx, ref)
	if err != nil {
		return err
	}
	if value.Revision != expectedRevision {
		return spec.ErrConflict
	}
	if err := a.requireMutable(ctx, value.RootID, false); err != nil {
		return err
	}

	sourceValue, err := a.sources.Get(
		ctx,
		value.RootID,
		value.Binding.SourceID,
	)
	if err != nil {
		return err
	}
	if sourceValue.Kind != sourceModel.SourceKindManagedDirectory {
		return fmt.Errorf(
			"%w: source-backed Skill removal must update or unregister its Source",
			spec.ErrUnsupported,
		)
	}
	if value.Binding.SubresourceLocator != "" ||
		!skillDomain.IsSkillDefinitionFile(
			value.Binding.Locator,
		) {
		return fmt.Errorf(
			"%w: managed Skill must originate at a configured package document",
			spec.ErrUnsupported,
		)
	}

	packageAddress, err := skillDomain.ManagedPackageAddressFromSkillLocator(
		value.Binding.Locator,
	)
	if err != nil {
		return err
	}
	removeRequest := artifactModel.RemoveArtifactRequest{
		RootID:           value.RootID,
		SourceID:         value.Binding.SourceID,
		Package:          packageAddress,
		ExpectedArtifact: &ref,
	}
	if sourceValue.StorageKey == collection.SkillManagedSourceStorageKey {
		locator := value.Binding.Locator
		removeRequest.PruneDiscoveryLocator = &locator
	}
	if err := a.managedArtifacts.Remove(ctx, removeRequest); err != nil {
		return err
	}

	missing, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return err
	}
	if missing.State != artifactModel.StateMissing {
		return fmt.Errorf(
			"%w: removed managed Skill Artifact is not missing",
			spec.ErrConflict,
		)
	}

	return a.artifacts.Purge(
		ctx,
		ref,
		missing.Revision,
	)
}

// ResolveSkillCapabilities resolves a Skill and preserves mapped Tool
// fallback occurrences from Skill.allowedTools.
func (a *API) ResolveSkillCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (resolve.CapabilityPlan, error) {
	if a == nil ||
		a.resources == nil ||
		a.declarationResolver == nil {
		return resolve.CapabilityPlan{}, spec.ErrClosed
	}
	return resourceFlow.WithVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (resolve.CapabilityPlan, error) {
			return a.declarationResolver.ResolveSkillCapabilities(
				sessionCtx,
				ref,
			)
		},
	)
}

func (a *API) getManagedSkillDocument(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (skillDomain.ManagedSkillDocument, error) {
	value, err := a.GetSkill(ctx, ref)
	if err != nil {
		return skillDomain.ManagedSkillDocument{}, err
	}
	sourceValue, err := a.sources.Get(
		ctx,
		value.RootID,
		value.Binding.SourceID,
	)
	if err != nil {
		return skillDomain.ManagedSkillDocument{}, err
	}
	if sourceValue.Kind != sourceModel.SourceKindManagedDirectory {
		return skillDomain.ManagedSkillDocument{}, fmt.Errorf(
			"%w: only managed Skills expose editable Skill documents",
			spec.ErrUnsupported,
		)
	}
	if value.Binding.SubresourceLocator != "" {
		return skillDomain.ManagedSkillDocument{}, fmt.Errorf(
			"%w: managed Skill must originate at a configured package document",
			spec.ErrUnsupported,
		)
	}
	if _, err := skillDomain.ManagedPackageAddressFromSkillLocator(
		value.Binding.Locator,
	); err != nil {
		return skillDomain.ManagedSkillDocument{}, fmt.Errorf(
			"%w: only application-managed Skill packages expose editable documents",
			spec.ErrUnsupported,
		)
	}

	resolved, err := a.resources.ResolveArtifact(
		ctx,
		ref,
		resourceModel.ResolveOptions{},
	)
	if err != nil {
		return skillDomain.ManagedSkillDocument{}, err
	}
	entry, err := a.resources.ReadSourceEntry(
		ctx,
		value.RootID,
		value.Binding.SourceID,
		value.Binding.Locator,
		spec.MaxCandidateBytes,
	)
	if err != nil {
		return skillDomain.ManagedSkillDocument{}, err
	}
	if value.SourceContentDigest == nil ||
		entry.Digest != *value.SourceContentDigest ||
		entry.SourceRevision !=
			resolved.RefreshState.SourceRevision ||
		entry.SourceGeneration !=
			resolved.RefreshState.SourceGeneration {
		return skillDomain.ManagedSkillDocument{}, fmt.Errorf(
			"%w: Skill Source changed while reading managed document",
			spec.ErrRefreshRequired,
		)
	}
	doc, _, err := skillDomain.ParseSkillDocument(
		entry.Content,
		string(value.LogicalName),
	)
	if err != nil {
		return skillDomain.ManagedSkillDocument{}, err
	}
	return skillDomain.ManagedSkillDocument{
		Artifact: value,
		Document: doc,
	}, nil
}

func (a *API) ensureSkillBaselineCollection(
	ctx context.Context,
	rootID rootModel.RootID,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.EnsureBaseline(ctx, rootID)
}

func (a *API) requireMutable(
	ctx context.Context,
	rootID rootModel.RootID,
	allowProtected bool,
) error {
	if err := rootID.Validate(); err != nil {
		return err
	}
	if !a.protection.IsProtectedRoot(rootID) {
		return nil
	}
	if !allowProtected {
		return fmt.Errorf(
			"%w: protected Root %q requires trusted installer access",
			spec.ErrProtected,
			rootID,
		)
	}
	return a.protection.RequirePrivilegedInstaller(ctx)
}
