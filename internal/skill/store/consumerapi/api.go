package consumerapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/collectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
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
}

func New(
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	protection compositionapi.ProtectionAPI,
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
	return &API{
		sources:          sources,
		discovery:        discovery,
		artifacts:        artifacts,
		resources:        resources,
		managedArtifacts: managedArtifacts,
		protection:       protection,
	}, nil
}

func SkillDiscoverySpec(
	r basespec.Locator,
) (source.DiscoverySpec, error) {
	if r == "" {
		r = "."
	}
	if err := r.Validate(true); err != nil {
		return source.DiscoverySpec{}, err
	}
	value := source.DiscoverySpec{
		DirectoryRoots: []source.DirectoryRoot{{
			Root:      r,
			Recursive: true,
			IncludePatterns: []string{
				"**/" + string(skillDomain.SkillDefinitionFileName),
			},
		}},
		DecoderHints: []source.DecoderHint{{
			Locator:   r,
			Recursive: true,
			DecoderIDs: []basespec.DecoderID{
				skillDomain.MarkdownDecoderID,
			},
		}},
		AllowedDecoderIDs: []basespec.DecoderID{
			skillDomain.MarkdownDecoderID,
		},
		Authoritative: true,
	}
	value = value.Normalized()
	if err := value.Validate(); err != nil {
		return source.DiscoverySpec{}, err
	}
	return value, nil
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

	rootPath, err := normalizeSkillDirectoryRoot(request.RootPath)
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

	value, _, err := a.sources.Ensure(
		ctx,
		request.RootID,
		source.Draft{
			ID:          source.SourceID(uuidutil.NewUUIDv7()),
			StorageKey:  skillPathStorageKey(rootPath),
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

func normalizeSkillDirectoryRoot(
	raw string,
) (string, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", fmt.Errorf(
			"%w: Skill Source root path is required",
			basespec.ErrInvalid,
		)
	}
	absolute, err := filepath.Abs(raw)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
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
	if err := request.RootID.Validate(); err != nil {
		return ManagedSkillCreateResult{}, err
	}
	if err := request.SourceID.Validate(); err != nil {
		return ManagedSkillCreateResult{}, err
	}
	if err := basespec.LogicalName(request.SkillName).Validate(); err != nil {
		return ManagedSkillCreateResult{}, err
	}
	if err := a.requireMutable(ctx, request.RootID, false); err != nil {
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
	if _, err := a.ensureManagedSkillDiscovery(
		ctx,
		request.RootID,
		request.SourceID,
		locator,
	); err != nil {
		return ManagedSkillCreateResult{}, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: request.RootID,
			Binding: artifact.SourceBinding{
				SourceID: request.SourceID,
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
		return ManagedSkillCreateResult{}, err
	}

	value := published.Artifact
	if value.Enabled != request.Enabled {
		value, err = a.artifacts.SetEnabled(
			ctx,
			value.Ref(),
			value.Revision,
			request.Enabled,
		)
		if err != nil {
			return ManagedSkillCreateResult{}, err
		}
	}
	return ManagedSkillCreateResult{
		Artifact: value,
		Address:  value.Address(),
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

	resolved, err := a.resources.ResolveArtifact(
		ctx,
		ref,
		resource.ResolveOptions{
			VerifySourceContent: true,
		},
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
		entry.SourceGeneration != resolved.RefreshState.SourceGeneration {
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
		return a.artifacts.Purge(ctx, ref, expectedRevision)
	}

	packageAddress, err := skillDomain.ManagedPackageAddressFromSkillLocator(
		value.Binding.Locator,
	)
	if err != nil {
		return err
	}
	return a.managedArtifacts.Remove(
		ctx,
		artifact.RemoveArtifactRequest{
			RootID:           value.RootID,
			SourceID:         value.Binding.SourceID,
			Package:          packageAddress,
			ExpectedArtifact: &ref,
		},
	)
}

func (a *API) InstallBuiltInSkill(
	ctx context.Context,
	request BuiltInSkillInstallRequest,
) (artifact.Artifact, error) {
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return artifact.Artifact{}, err
	}
	if err := request.RootID.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if err := request.SourceID.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if err := request.ExpectedLogicalName.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if err := cryptoutil.ValidateDigest(request.ExpectedDefinition); err != nil {
		return artifact.Artifact{}, err
	}
	if err := request.PackageAddress.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if request.PackageAddress.Kind != skillDomain.ManagedSkillPackageKind {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: built-in Skill package kind must be %q",
			basespec.ErrInvalid,
			skillDomain.ManagedSkillPackageKind,
		)
	}
	if !a.protection.IsProtectedRoot(request.RootID) {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: built-in Skill Root is not protected",
			basespec.ErrProtected,
		)
	}
	if err := a.requireMutable(ctx, request.RootID, true); err != nil {
		return artifact.Artifact{}, err
	}

	locator, err := skillDomain.ManagedPackageLocatorForSkill(
		request.PackageAddress,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if _, err := a.ensureManagedSkillDiscovery(
		ctx,
		request.RootID,
		request.SourceID,
		locator,
	); err != nil {
		return artifact.Artifact{}, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: request.RootID,
			Binding: artifact.SourceBinding{
				SourceID: request.SourceID,
				Locator:  locator,
			},
			ExpectedKind:        skillDomain.SkillArtifactKind,
			ExpectedLogicalName: request.ExpectedLogicalName,
			ExpectedDefinition:  request.ExpectedDefinition,
			Package: source.ManagedPackagePublication{
				Address: request.PackageAddress,
				Files:   request.PackageFiles,
			},
			AllowProtected: true,
		},
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	value := published.Artifact
	if value.Enabled != request.Enabled {
		value, err = a.artifacts.SetEnabled(
			ctx,
			value.Ref(),
			value.Revision,
			request.Enabled,
		)
		if err != nil {
			return artifact.Artifact{}, err
		}
	}
	return value, nil
}

// InstallBuiltInSkillCollection publishes the canonical Collection that
// exposes all installed built-in Skills as ordered symbolic members.
func (a *API) InstallBuiltInSkillCollection(
	ctx context.Context,
	request BuiltInSkillCollectionInstallRequest,
) (artifact.Artifact, error) {
	if a == nil {
		return artifact.Artifact{}, basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return artifact.Artifact{}, err
	}
	if err := request.RootID.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if err := request.SourceID.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if err := request.Name.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if len(request.Members) == 0 ||
		len(request.Members) > basespec.MaxDefinitionDependencies {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: built-in Skill Collection has invalid member count",
			basespec.ErrInvalid,
		)
	}
	if !a.protection.IsProtectedRoot(request.RootID) {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: built-in Skill Collection Root is not protected",
			basespec.ErrProtected,
		)
	}
	if err := a.requireMutable(ctx, request.RootID, true); err != nil {
		return artifact.Artifact{}, err
	}

	sourceValue, err := a.sources.Get(
		ctx,
		request.RootID,
		request.SourceID,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if sourceValue.Kind != source.SourceKindManagedDirectory {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: built-in Skill Collection Source must be managed",
			basespec.ErrInvalid,
		)
	}

	seen := make(map[basespec.LogicalName]struct{}, len(request.Members))
	members := make([]declaration.Entry, 0, len(request.Members))
	for index, name := range request.Members {
		if err := name.Validate(); err != nil {
			return artifact.Artifact{}, fmt.Errorf(
				"built-in Skill Collection members[%d]: %w",
				index,
				err,
			)
		}
		if _, duplicate := seen[name]; duplicate {
			return artifact.Artifact{}, fmt.Errorf(
				"%w: built-in Skill Collection repeats Skill %q",
				basespec.ErrInvalid,
				name,
			)
		}
		seen[name] = struct{}{}
		member, err := declaration.NewSymbolicEntry(
			declaration.TypeSkill,
			name,
		)
		if err != nil {
			return artifact.Artifact{}, err
		}
		members = append(members, member)
	}

	document := collectionv1.CollectionDocument{
		APIVersion:  collectionv1.CollectionSchemaVersion,
		Type:        collectionv1.CollectionType,
		Name:        string(request.Name),
		Description: request.Description,
		Version:     string(builtin.UnversionedPackageVersion),
		Members:     members,
	}
	raw, err := document.CanonicalJSON()
	if err != nil {
		return artifact.Artifact{}, err
	}
	entry, err := declaration.NewEntry(document)
	if err != nil {
		return artifact.Artifact{}, err
	}
	definitionValue, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return artifact.Artifact{}, err
	}

	address, err := source.NewManagedPackageAddress(
		skillDomain.BuiltinCollectionPackageKind,
		request.Name,
		builtin.UnversionedPackageVersion,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	locator, err := address.FileLocator(
		skillDomain.BuiltinCollectionDocumentFile,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if err := validateCanonicalCollectionSourceDiscovery(
		sourceValue.Discovery,
		locator,
	); err != nil {
		return artifact.Artifact{}, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: request.RootID,
			Binding: artifact.SourceBinding{
				SourceID: request.SourceID,
				Locator:  locator,
			},
			ExpectedKind: artifact.ArtifactKind(
				collectionv1.CollectionType,
			),
			ExpectedLogicalName: request.Name,
			ExpectedDefinition:  definitionValue.Digest,
			Package: source.ManagedPackagePublication{
				Address: address,
				Files: []source.ManagedPackageFile{{
					Locator: skillDomain.BuiltinCollectionDocumentFile,
					Content: raw,
				}},
			},
			AllowProtected: true,
		},
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	value := published.Artifact
	if value.Enabled != request.Enabled {
		return a.artifacts.SetEnabled(
			ctx,
			value.Ref(),
			value.Revision,
			request.Enabled,
		)
	}
	return value, nil
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

	inspection, err := a.discovery.InspectSource(
		ctx,
		rootID,
		sourceID,
	)
	if errors.Is(err, basespec.ErrRefreshStateNotFound) {
		_, err = a.discovery.RefreshSource(ctx, rootID, sourceID)
		return err
	}
	if err != nil {
		return err
	}
	if inspection.IsCurrent() {
		return nil
	}
	_, err = a.discovery.RefreshSource(ctx, rootID, sourceID)
	return err
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

func (a *API) ensureManagedSkillDiscovery(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	locator basespec.Locator,
) (source.Summary, error) {
	value, err := a.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return source.Summary{}, err
	}
	if value.Kind != source.SourceKindManagedDirectory {
		return source.Summary{}, fmt.Errorf(
			"%w: managed Skill Source must have kind %q",
			basespec.ErrInvalid,
			source.SourceKindManagedDirectory,
		)
	}
	if !value.Enabled {
		return source.Summary{}, fmt.Errorf(
			"%w: managed Skill Source is disabled",
			basespec.ErrConflict,
		)
	}

	next := value.Discovery.Clone()
	inScope, err := next.InScope(locator)
	if err != nil {
		return source.Summary{}, err
	}
	if !inScope {
		next.ExplicitLocators = append(
			next.ExplicitLocators,
			locator,
		)
	}
	if len(next.AllowedDecoderIDs) != 0 &&
		!slices.Contains(
			next.AllowedDecoderIDs,
			skillDomain.MarkdownDecoderID,
		) {
		next.AllowedDecoderIDs = append(
			next.AllowedDecoderIDs,
			skillDomain.MarkdownDecoderID,
		)
	}
	next = next.Normalized()
	if value.Discovery.Equal(next) {
		return value, nil
	}
	return a.sources.Update(
		ctx,
		rootID,
		sourceID,
		source.Update{
			ExpectedRevision: value.Revision,
			DisplayName:      value.DisplayName,
			Enabled:          value.Enabled,
			Discovery:        &next,
		},
	)
}

func validateCanonicalCollectionSourceDiscovery(
	discovery source.DiscoverySpec,
	locator basespec.Locator,
) error {
	discovery = discovery.Effective()
	if discovery.Empty() {
		return fmt.Errorf(
			"%w: Collection Source has no declaration discovery configuration",
			basespec.ErrRefreshRequired,
		)
	}
	inScope, err := discovery.InScope(locator)
	if err != nil {
		return err
	}
	if !inScope {
		return fmt.Errorf(
			"%w: Collection declaration locator %q is outside Source discovery",
			basespec.ErrInvalid,
			locator,
		)
	}
	if len(discovery.AllowedDecoderIDs) == 0 ||
		slices.Contains(
			discovery.AllowedDecoderIDs,
			decoder.JSONDecoderID,
		) {
		return nil
	}
	return fmt.Errorf(
		"%w: Collection Source does not allow decoder %q",
		basespec.ErrInvalid,
		decoder.JSONDecoderID,
	)
}
