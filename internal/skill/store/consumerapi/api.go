package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/consumerutil"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

type API struct {
	sources          compositionapi.SourceAPI
	discovery        compositionapi.DiscoveryAPI
	artifacts        compositionapi.ArtifactAPI
	resources        compositionapi.ResourceAPI
	managedArtifacts compositionapi.ManagedArtifactAPI
	protection       compositionapi.ProtectionAPI
	collections      *collection.API

	declarationResolver *resolve.Resolver
}

func New(
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	protection compositionapi.ProtectionAPI,
	options ...Option,
) (*API, error) {
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		resources == nil ||
		managedArtifacts == nil ||
		protection == nil {
		return nil, fmt.Errorf(
			"%w: Skill Store dependencies are incomplete",
			basespec.ErrInvalid,
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
	}
	locators, err := resolve.NewProviderLocatorResolver(
		config.locatorResolvers,
		skillLocatorRuntime{api: output},
	)
	if err != nil {
		return nil, err
	}
	graphResolver, err := resolve.NewWithOptions(
		resolve.ResolverOptions{
			Artifacts:            artifacts,
			SourceArtifacts:      artifacts,
			SourceEntries:        resources,
			Locators:             locators,
			FallbackProviders:    config.fallbackProviders,
			ProtectedBuiltinRoot: documentTopology.BuiltinRootID(),
			Limits:               resolve.DefaultLimits(),
		},
	)
	if err != nil {
		return nil, err
	}
	collections, err := collection.NewWithResolver(
		sources,
		discovery,
		artifacts,
		managedArtifacts,
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
	r basespec.Locator,
) (source.DiscoverySpec, error) {
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
) (source.Summary, error) {
	if err := request.RootID.Validate(); err != nil {
		return source.Summary{}, err
	}
	if err := basespec.ValidateRequiredText(
		"Skill Source display name",
		request.SourceDisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return source.Summary{}, err
	}

	rootPath, err := consumerutil.NormalizeFilesystemSourceRoot(
		request.RootPath,
		"Skill Source root path",
	)
	if err != nil {
		return source.Summary{}, err
	}

	discovery, err := SkillDiscoverySpec(".")
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
					"skill-path",
					rootPath,
				),
				Kind:        source.SourceKindFilesystemDirectory,
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
	rootID root.RootID,
	sourceID source.SourceID,
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

func (a *API) EnsureSkillBaselineCollection(
	ctx context.Context,
	rootID root.RootID,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return a.collections.EnsureBaseline(ctx, rootID)
}

func (a *API) ListSkills(
	ctx context.Context,
	rootID root.RootID,
) ([]artifact.Artifact, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	values, err := a.artifacts.ListByRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}
	output := make([]artifact.Artifact, 0, len(values))
	for _, value := range values {
		if !skillDomain.IsSkillKind(value.Kind) {
			continue
		}
		output = append(output, value.Clone())
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].LogicalName != output[right].LogicalName {
			return output[left].LogicalName <
				output[right].LogicalName
		}
		return output[left].ID < output[right].ID
	})
	return output, nil
}

func (a *API) GetSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	value, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if !skillDomain.IsSkillKind(value.Kind) {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: Artifact %q is not a Skill",
			basespec.ErrNotFound,
			ref.ArtifactID,
		)
	}
	return value, nil
}

func (a *API) SetSkillEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if _, err := a.GetSkill(ctx, ref); err != nil {
		return artifact.Artifact{}, err
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
		return ManagedSkillCreateResult{}, basespec.ErrClosed
	}
	if err := request.Collection.Validate(); err != nil {
		return ManagedSkillCreateResult{}, err
	}
	if request.ExpectedCollectionRevision == 0 {
		return ManagedSkillCreateResult{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := a.requireMutable(
		ctx,
		request.Collection.RootID,
		false,
	); err != nil {
		return ManagedSkillCreateResult{}, err
	}
	if err := basespec.LogicalName(request.SkillName).Validate(); err != nil {
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
	if definitionValue.LogicalName != basespec.LogicalName(request.SkillName) {
		return ManagedSkillCreateResult{}, fmt.Errorf(
			"%w: SKILL.md name does not match requested Skill name",
			basespec.ErrInvalid,
		)
	}

	packageAddress, err := skillDomain.ManagedPackageAddressForSkill(
		definitionValue.LogicalName,
		definitionValue.LogicalVersion,
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
		artifact.PublishArtifactRequest{
			RootID: rootID,
			Binding: artifact.SourceBinding{
				SourceID: sourceID,
				Locator:  locator,
			},
			ExpectedKind:        skillDomain.SkillArtifactKind,
			ExpectedLogicalName: definitionValue.LogicalName,
			ExpectedDefinition:  definitionValue.Digest,
			Package: source.ManagedPackagePublication{
				Address: packageAddress,
				Files:   files,
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
		return ManagedSkillReplaceResult{}, basespec.ErrClosed
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
			basespec.ErrInvalid,
		)
	}
	if request.ExpectedCollectionRevision == 0 {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			basespec.ErrInvalid,
		)
	}
	if request.ExpectedArtifactRevision == 0 {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: expected Skill Artifact revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.LogicalName(request.SkillName).Validate(); err != nil {
		return ManagedSkillReplaceResult{}, err
	}

	collectionView, err := a.collections.Read(ctx, request.Collection)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	if collectionView.Artifact.Revision != request.ExpectedCollectionRevision {
		return ManagedSkillReplaceResult{}, basespec.ErrConflict
	}
	if !collectionView.Editable {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill Collection is read-only",
			basespec.ErrUnsupported,
		)
	}

	current, err := a.GetSkill(ctx, request.Artifact)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	if current.Revision != request.ExpectedArtifactRevision {
		return ManagedSkillReplaceResult{}, basespec.ErrConflict
	}
	if current.Binding.SubresourceLocator != "" ||
		!skillDomain.IsSkillDefinitionFile(current.Binding.Locator) {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill is not a replaceable managed Skill package",
			basespec.ErrUnsupported,
		)
	}
	if current.Binding.SourceID != collectionView.Artifact.Binding.SourceID {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill is not owned by this Collection Source",
			basespec.ErrUnsupported,
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
	if sourceValue.Kind != source.SourceKindManagedDirectory {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill is not backed by a managed Source",
			basespec.ErrUnsupported,
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
			basespec.ErrReferenceUnresolved,
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
			basespec.ErrInvalid,
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
			basespec.ErrInvalid,
		)
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
			basespec.ErrRefreshRequired,
		)
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: current.RootID,
			Binding: artifact.SourceBinding{
				SourceID: current.Binding.SourceID,
				Locator:  current.Binding.Locator,
			},
			ExpectedKind:        skillDomain.SkillArtifactKind,
			ExpectedLogicalName: current.LogicalName,
			ExpectedDefinition:  definitionValue.Digest,
			Package: source.ManagedPackagePublication{
				Address:            currentAddress,
				ExpectedGeneration: inspection.State.SourceGeneration,
				Files:              files,
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
			basespec.ErrConflict,
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
	ref artifact.ArtifactRef,
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
	if sourceValue.Kind != source.SourceKindManagedDirectory {
		return skillDomain.ManagedSkillDocument{}, fmt.Errorf(
			"%w: only managed Skills expose editable Skill documents",
			basespec.ErrUnsupported,
		)
	}
	if value.Binding.SubresourceLocator != "" ||
		!skillDomain.IsSkillDefinitionFile(
			value.Binding.Locator,
		) {
		return skillDomain.ManagedSkillDocument{}, fmt.Errorf(
			"%w: managed Skill must originate at a configured package document",
			basespec.ErrUnsupported,
		)
	}

	resolved, err := a.resources.ResolveArtifact(
		ctx,
		ref,
		resource.ResolveOptions{},
	)
	if err != nil {
		return skillDomain.ManagedSkillDocument{}, err
	}
	entry, err := a.resources.ReadSourceEntry(
		ctx,
		value.RootID,
		value.Binding.SourceID,
		value.Binding.Locator,
		basespec.MaxCandidateBytes,
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
			basespec.ErrRefreshRequired,
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

func (a *API) PurgeSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Skill Artifact revision is required",
			basespec.ErrInvalid,
		)
	}
	value, err := a.GetSkill(ctx, ref)
	if err != nil {
		return err
	}
	if value.Revision != expectedRevision {
		return basespec.ErrConflict
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
	if sourceValue.Kind != source.SourceKindManagedDirectory {
		return fmt.Errorf(
			"%w: source-backed Skill removal must update or unregister its Source",
			basespec.ErrUnsupported,
		)
	}
	if value.Binding.SubresourceLocator != "" ||
		!skillDomain.IsSkillDefinitionFile(
			value.Binding.Locator,
		) {
		return fmt.Errorf(
			"%w: managed Skill must originate at a configured package document",
			basespec.ErrUnsupported,
		)
	}

	packageAddress, err := skillDomain.ManagedPackageAddressFromSkillLocator(
		value.Binding.Locator,
	)
	if err != nil {
		return err
	}
	removeRequest := artifact.RemoveArtifactRequest{
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
	if missing.State != artifact.StateMissing {
		return fmt.Errorf(
			"%w: removed managed Skill Artifact is not missing",
			basespec.ErrConflict,
		)
	}

	return a.artifacts.Purge(
		ctx,
		ref,
		missing.Revision,
	)
}

func (a *API) EnsureBuiltInSkillSourceCurrent(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) error {
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	if err := a.requireMutable(ctx, rootID, true); err != nil {
		return err
	}

	return compositionapi.EnsureSourceCurrent(
		ctx,
		a.discovery,
		rootID,
		sourceID,
	)
}

// ResolveArtifactCapabilities returns the complete capability tree for any
// declaration Artifact visible through the Skill consumer.
func (a *API) ResolveArtifactCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (resolve.CapabilityPlan, error) {
	if a == nil || a.declarationResolver == nil {
		return resolve.CapabilityPlan{}, basespec.ErrClosed
	}
	return a.declarationResolver.ResolveCapabilities(ctx, ref)
}

// ResolveSkillCapabilities resolves a Skill and preserves mapped Tool
// fallback occurrences from Skill.allowedTools.
func (a *API) ResolveSkillCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (resolve.CapabilityPlan, error) {
	if a == nil || a.declarationResolver == nil {
		return resolve.CapabilityPlan{}, basespec.ErrClosed
	}
	return a.declarationResolver.ResolveSkillCapabilities(ctx, ref)
}

func (a *API) requireMutable(
	ctx context.Context,
	rootID root.RootID,
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
			basespec.ErrProtected,
			rootID,
		)
	}
	return a.protection.RequirePrivilegedInstaller(ctx)
}
