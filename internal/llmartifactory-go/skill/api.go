package skill

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	managepackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/model"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	skillPackage "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/package"
	skillSource "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/source"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

type Service struct {
	cat              catalog.API
	sources          source.API
	discovery        refreshFlow.API
	artifacts        artifact.API
	resources        resourceFlow.API
	managedArtifacts managepackageFlow.API
	protection       root.ProtectionAPI
	definitions      definition.API
	plugins          *pluginAPI.API

	declarationResolver *composition.Resolver
	support             Support
}

func New(
	sources source.API,
	discovery refreshFlow.API,
	artifacts artifact.API,
	resources resourceFlow.API,
	managedArtifacts managepackageFlow.API,
	protection root.ProtectionAPI,
	cat catalog.API,
	definitions definition.API,
	options ...Option,
) (*Service, error) {
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
	if config.resolver == nil {
		return nil, fmt.Errorf(
			"%w: Skill composition resolver is required",
			spec.ErrInvalid,
		)
	}
	if err := config.support.Validate(); err != nil {
		return nil, err
	}
	graphResolver := config.resolver

	output := &Service{
		sources:          sources,
		discovery:        discovery,
		artifacts:        artifacts,
		resources:        resources,
		managedArtifacts: managedArtifacts,
		protection:       protection,
		cat:              cat,
		definitions:      definitions,
		support:          config.support.Clone(),
	}
	plugins, err := pluginAPI.New(
		artifacts,
		cat,
		sources,
		discovery,
		managedArtifacts,
		definitions,
		graphResolver,
		config.support.PluginProfile,
	)
	if err != nil {
		return nil, err
	}
	output.plugins = plugins
	output.declarationResolver = graphResolver
	return output, nil
}

func (a *Service) RegisterSkillDirectory(
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

	discovery := a.support.Discovery.Clone()
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
				Kind:        fsdir.Kind,
				DisplayName: request.SourceDisplayName,
				Enabled:     true,
				Config:      config,
				Discovery:   discovery,
			},
		},
	)
}

func (a *Service) RefreshSkillSource(
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

func (a *Service) GetSkill(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (artifactModel.Artifact, error) {
	value, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if !skillSource.IsSkillKind(value.Kind) {
		return artifactModel.Artifact{}, fmt.Errorf(
			"%w: Artifact %q is not a Skill",
			spec.ErrNotFound,
			ref.ArtifactID,
		)
	}
	return value, nil
}

func (a *Service) SetSkillEnabled(
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

func (a *Service) CreateManagedSkill(
	ctx context.Context,
	request ManagedSkillCreateRequest,
) (ManagedSkillCreateResult, error) {
	if err := request.Plugin.Validate(); err != nil {
		return ManagedSkillCreateResult{}, err
	}
	if request.ExpectedPluginRevision == 0 {
		return ManagedSkillCreateResult{}, fmt.Errorf(
			"%w: expected Plugin revision is required",
			spec.ErrInvalid,
		)
	}
	if err := a.requireMutable(
		ctx,
		request.Plugin.RootID,
	); err != nil {
		return ManagedSkillCreateResult{}, err
	}
	if err := spec.LogicalName(request.SkillName).Validate(); err != nil {
		return ManagedSkillCreateResult{}, err
	}

	files, skillMD, err := skillPackage.NormalizeManagedSkillFiles(
		request.SKILLMD,
		request.Files,
		a.support.Documents,
	)
	if err != nil {
		return ManagedSkillCreateResult{}, err
	}
	definitionValue, _, err := skillSource.DecodeSkillDocument(
		a.support.Documents,
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

	packageAddress, err := skillPackage.ManagedPackageAddressForSkill(
		a.support.Package,
		definitionValue.LogicalName,
		definitionValue.LogicalVersion,
	)
	if err != nil {
		return ManagedSkillCreateResult{}, err
	}
	storageFiles, err := skillPackage.ManagedSkillStorageFiles(
		a.support.Documents,
		packageAddress,
		files,
	)
	if err != nil {
		return ManagedSkillCreateResult{}, err
	}
	locator, err := skillPackage.ManagedPackageLocatorForSkill(
		a.support.Package,
		packageAddress,
	)
	if err != nil {
		return ManagedSkillCreateResult{}, err
	}

	membership, err := a.plugins.EnsureMemberForPluginSource(
		ctx,
		pluginAPI.EnsureMemberForPluginSourceRequest{
			Plugin:           request.Plugin,
			ExpectedRevision: request.ExpectedPluginRevision,
			Type:             declaration.TypeSkill,
			Name:             definitionValue.LogicalName,
			Locator:          locator,
		},
	)
	if err != nil {
		return ManagedSkillCreateResult{}, err
	}

	result := ManagedSkillCreateResult{
		Plugin:            membership.Plugin,
		MembershipCreated: membership.Created,
	}
	rootID := membership.Plugin.Artifact.RootID
	sourceID := membership.Plugin.Artifact.Binding.SourceID
	if _, err := a.plugins.EnsureManagedDeclarationDiscovery(
		ctx,
		rootID,
		sourceID,
		locator,
		a.support.Package.Document.DecoderID,
	); err != nil {
		return result, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		managepackageModel.PublishRequest{
			RootID: rootID,
			Binding: artifactModel.SourceBinding{
				SourceID: sourceID,
				Locator:  locator,
			},
			ExpectedKind:        skillSource.SkillArtifactKind,
			ExpectedLogicalName: definitionValue.LogicalName,
			ExpectedDefinition:  definitionValue.Digest,
			Package: managedpackageModel.ManagedPackagePublication{
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
// package address, Plugin membership, and managed Source remain stable.
func (a *Service) ReplaceManagedSkill(
	ctx context.Context,
	request ManagedSkillReplaceRequest,
) (ManagedSkillReplaceResult, error) {
	if err := request.Plugin.Validate(); err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	if err := request.Artifact.Validate(); err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	if request.Plugin.RootID != request.Artifact.RootID {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill Artifact belongs to another Root",
			spec.ErrInvalid,
		)
	}
	if request.ExpectedPluginRevision == 0 {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: expected Plugin revision is required",
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

	pluginView, err := a.plugins.Read(ctx, request.Plugin)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	if pluginView.Artifact.Revision != request.ExpectedPluginRevision {
		return ManagedSkillReplaceResult{}, spec.ErrConflict
	}
	if !pluginView.Editable &&
		!pluginView.Baseline {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill Plugin is read-only",
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
		!skillSource.IsSkillDefinitionFile(
			a.support.Documents,
			current.Binding.Locator,
		) {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill is not a replaceable managed Skill package",
			spec.ErrUnsupported,
		)
	}
	if current.Binding.SourceID != pluginView.Artifact.Binding.SourceID {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill is not owned by this Plugin Source",
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
	if a.support.PluginProfile.Source == nil ||
		!a.support.PluginProfile.Source.Matches(sourceValue) {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill is not backed by the configured managed Skill Source",
			spec.ErrUnsupported,
		)
	}

	memberships, err := a.plugins.ListMembershipsForArtifact(
		ctx,
		request.Artifact,
	)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	memberFound := false
	for _, membership := range memberships {
		if membership.Plugin != request.Plugin ||
			!membership.ResolvedToArtifact {
			continue
		}
		memberFound = true
		break
	}
	if !memberFound {
		return ManagedSkillReplaceResult{}, fmt.Errorf(
			"%w: Skill is not a direct member of the requested Plugin",
			spec.ErrReferenceUnresolved,
		)
	}

	files, skillMD, err := skillPackage.NormalizeManagedSkillFiles(
		request.SKILLMD,
		request.Files,
		a.support.Documents,
	)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	definitionValue, _, err := skillSource.DecodeSkillDocument(
		a.support.Documents,
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

	currentAddress, err := skillPackage.ManagedPackageAddressFromSkillLocator(
		a.support.Package,
		current.Binding.Locator,
	)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}
	requestedAddress, err := skillPackage.ManagedPackageAddressForSkill(
		a.support.Package,
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

	storageFiles, err := skillPackage.ManagedSkillStorageFiles(
		a.support.Documents,
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
		managepackageModel.PublishRequest{
			RootID: current.RootID,
			Binding: artifactModel.SourceBinding{
				SourceID: current.Binding.SourceID,
				Locator:  current.Binding.Locator,
			},
			ExpectedKind:        skillSource.SkillArtifactKind,
			ExpectedLogicalName: current.LogicalName,
			ExpectedDefinition:  definitionValue.Digest,
			Package: managedpackageModel.ManagedPackagePublication{
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

	pluginView, err = a.plugins.Read(ctx, request.Plugin)
	if err != nil {
		return ManagedSkillReplaceResult{}, err
	}

	return ManagedSkillReplaceResult{
		Artifact: updated,
		Address:  updated.Address(),
		Plugin:   pluginView,
	}, nil
}

func (a *Service) GetManagedSkillDocument(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (skillSource.ManagedSkillDocument, error) {
	return resourceFlow.WithVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (skillSource.ManagedSkillDocument, error) {
			return a.getManagedSkillDocument(sessionCtx, ref)
		},
	)
}

func (a *Service) PurgeSkill(
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
	if err := a.requireMutable(ctx, value.RootID); err != nil {
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
	if a.support.PluginProfile.Source == nil ||
		!a.support.PluginProfile.Source.Matches(sourceValue) {
		return fmt.Errorf(
			"%w: Skill is not backed by the configured managed Skill Source",
			spec.ErrUnsupported,
		)
	}
	if value.Binding.SubresourceLocator != "" ||
		!skillSource.IsSkillDefinitionFile(
			a.support.Documents,
			value.Binding.Locator,
		) {
		return fmt.Errorf(
			"%w: managed Skill must originate at a configured package document",
			spec.ErrUnsupported,
		)
	}

	packageAddress, err := skillPackage.ManagedPackageAddressFromSkillLocator(
		a.support.Package,
		value.Binding.Locator,
	)
	if err != nil {
		return err
	}
	removeRequest := managepackageModel.RemoveRequest{
		RootID:           value.RootID,
		SourceID:         value.Binding.SourceID,
		Package:          packageAddress,
		ExpectedArtifact: &ref,
	}
	locator := value.Binding.Locator
	removeRequest.PruneDiscoveryLocator = &locator
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
func (a *Service) ResolveSkillCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (composition.CapabilityPlan, error) {
	return resourceFlow.WithVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (composition.CapabilityPlan, error) {
			return a.declarationResolver.ResolveSkillCapabilities(
				sessionCtx,
				ref,
			)
		},
	)
}

func (a *Service) EnsureSkillBaselinePlugin(
	ctx context.Context,
	rootID rootModel.RootID,
) (pluginAPI.PluginView, error) {
	return a.plugins.EnsureBaseline(ctx, rootID)
}

func (a *Service) getManagedSkillDocument(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (skillSource.ManagedSkillDocument, error) {
	value, err := a.GetSkill(ctx, ref)
	if err != nil {
		return skillSource.ManagedSkillDocument{}, err
	}
	sourceValue, err := a.sources.Get(
		ctx,
		value.RootID,
		value.Binding.SourceID,
	)
	if err != nil {
		return skillSource.ManagedSkillDocument{}, err
	}
	if a.support.PluginProfile.Source == nil ||
		!a.support.PluginProfile.Source.Matches(sourceValue) {
		return skillSource.ManagedSkillDocument{}, fmt.Errorf(
			"%w: only configured managed Skills expose editable documents",
			spec.ErrUnsupported,
		)
	}
	if value.Binding.SubresourceLocator != "" {
		return skillSource.ManagedSkillDocument{}, fmt.Errorf(
			"%w: managed Skill must originate at a configured package document",
			spec.ErrUnsupported,
		)
	}
	if _, err := skillPackage.ManagedPackageAddressFromSkillLocator(
		a.support.Package,
		value.Binding.Locator,
	); err != nil {
		return skillSource.ManagedSkillDocument{}, fmt.Errorf(
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
		return skillSource.ManagedSkillDocument{}, err
	}
	entry, err := a.resources.ReadSourceEntry(
		ctx,
		value.RootID,
		value.Binding.SourceID,
		value.Binding.Locator,
		spec.MaxCandidateBytes,
	)
	if err != nil {
		return skillSource.ManagedSkillDocument{}, err
	}
	if value.SourceContentDigest == nil ||
		entry.Digest != *value.SourceContentDigest ||
		entry.SourceRevision !=
			resolved.RefreshState.SourceRevision ||
		entry.SourceGeneration !=
			resolved.RefreshState.SourceGeneration {
		return skillSource.ManagedSkillDocument{}, fmt.Errorf(
			"%w: Skill Source changed while reading managed document",
			spec.ErrRefreshRequired,
		)
	}
	doc, _, err := skillSource.ParseSkillDocument(
		entry.Content,
		string(value.LogicalName),
	)
	if err != nil {
		return skillSource.ManagedSkillDocument{}, err
	}
	return skillSource.ManagedSkillDocument{
		Artifact: value,
		Document: doc,
	}, nil
}

func (a *Service) requireMutable(
	ctx context.Context,
	rootID rootModel.RootID,
) error {
	if !a.protection.IsProtectedRoot(rootID) {
		return nil
	}
	return a.protection.RequireInstallerPrivilege(ctx)
}
