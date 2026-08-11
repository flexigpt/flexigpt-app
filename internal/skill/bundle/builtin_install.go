package bundle

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/managedartifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/managed"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillArtifact "github.com/flexigpt/flexigpt-app/internal/skill/artifact"
)

const locatorPackage basespec.Locator = "package"

// builtInSkillArtifactPolicy deliberately creates no observed Artifacts.
// Protected topology has one declared static Artifact ID for every member.
// Unknown source content may remain visible as a catalog occurrence, but it
// must never silently become a canonical built-in Artifact.
type builtInSkillArtifactPolicy struct{}

func (builtInSkillArtifactPolicy) Derive(
	_ context.Context,
	_ collection.Collection,
	_ catalog.Occurrence,
	_ definition.Definition,
) (artifact.Draft, bool, []diagnostic.Diagnostic, error) {
	return artifact.Draft{}, false, nil, nil
}

type BuiltInCollectionSkill struct {
	ArtifactID basespec.ArtifactID
	Member     basespec.Locator
	Enabled    bool
}
type preparedBuiltInSkill struct {
	request          BuiltInCollectionSkill
	name             string
	locator          basespec.Locator
	definitionDigest cryptoutil.Digest
	packageSHA256    cryptoutil.Digest
}

// BuiltInCollectionInstallRequest is trusted installer input. It materializes
// exactly one complete portable Collection package in one managed Source
// package directory, pins all static Artifacts, and then refreshes once.
type BuiltInCollectionInstallRequest struct {
	Bundle                     collection.CollectionRef
	ExpectedCollectionRevision uint64
	PackageDirectory           basespec.Locator
	PackageFiles               []source.ManagedPackageFile
	Skills                     []BuiltInCollectionSkill
}

func (a *API) InstallBuiltInCollection(
	ctx context.Context,
	request BuiltInCollectionInstallRequest,
) ([]CreateManagedSkillResponse, error) {
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return nil, err
	}
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if err := request.Bundle.Validate(); err != nil {
		return nil, err
	}
	if request.ExpectedCollectionRevision == 0 {
		return nil, fmt.Errorf(
			"%w: expected built-in Collection revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := source.ValidateManagedPackageDirectory(
		request.PackageDirectory,
	); err != nil {
		return nil, err
	}
	if err := a.requireBundleMutation(ctx, request.Bundle.RootID, true); err != nil {
		return nil, err
	}

	bundle, err := a.GetBundle(ctx, request.Bundle)
	if err != nil {
		return nil, err
	}
	if bundle.Collection.Revision != request.ExpectedCollectionRevision {
		return nil, basespec.ErrConflict
	}
	attachment, sourceValue, err := managedAttachmentForRole(
		bundle,
		RoleBuiltIn,
	)
	if err != nil {
		return nil, err
	}
	if !attachment.Enabled || !sourceValue.Enabled {
		return nil, fmt.Errorf(
			"%w: built-in managed Source is disabled",
			basespec.ErrConflict,
		)
	}
	attachmentData, err := DecodeAttachmentData(attachment.Data)
	if err != nil {
		return nil, err
	}
	if attachmentData.DiscoveryRoot != request.PackageDirectory {
		return nil, fmt.Errorf(
			"%w: built-in package directory does not match attachment discovery root",
			basespec.ErrInvalid,
		)
	}
	if len(request.Skills) == 0 ||
		len(request.Skills) != len(attachmentData.ExpectedMemberDigests) {
		return nil, fmt.Errorf(
			"%w: built-in static Artifact registrations do not match expected members",
			basespec.ErrInvalid,
		)
	}

	publication, err := source.NormalizeManagedPackagePublication(
		source.ManagedPackagePublication{
			Directory: request.PackageDirectory,
			Files:     request.PackageFiles,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := managed.ValidatePackagePublication(publication); err != nil {
		return nil, err
	}

	filesByLocator := make(
		map[basespec.Locator][]byte,
		len(publication.Files),
	)
	hasCollectionFile := false
	for _, file := range publication.Files {
		filesByLocator[file.Locator] = append([]byte(nil), file.Content...)
		if file.Locator == builtinSchema.SkillCollectionV1FileName {
			hasCollectionFile = true
		}
	}
	if !hasCollectionFile {
		return nil, fmt.Errorf(
			"%w: built-in Collection package lacks %q",
			basespec.ErrInvalid,
			builtinSchema.SkillCollectionV1FileName,
		)
	}

	seenArtifactIDs := make(map[basespec.ArtifactID]struct{}, len(request.Skills))
	seenMembers := make(map[basespec.Locator]struct{}, len(request.Skills))
	prepared := make([]preparedBuiltInSkill, 0, len(request.Skills))

	for index, skill := range request.Skills {
		if err := basespec.ValidateArtifactID(skill.ArtifactID); err != nil {
			return nil, fmt.Errorf("skills[%d]: %w", index, err)
		}
		if err := basespec.ValidatePortableLocator(skill.Member, false); err != nil {
			return nil, fmt.Errorf("skills[%d]: %w", index, err)
		}
		if path.Base(string(skill.Member)) != skillArtifact.DefinitionFileName ||
			path.Dir(string(skill.Member)) == "." {
			return nil, fmt.Errorf(
				"%w: built-in member %q is not a packaged %q",
				basespec.ErrInvalid,
				skill.Member,
				skillArtifact.DefinitionFileName,
			)
		}
		if _, duplicate := seenArtifactIDs[skill.ArtifactID]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate built-in Artifact ID %q",
				basespec.ErrInvalid,
				skill.ArtifactID,
			)
		}
		if _, duplicate := seenMembers[skill.Member]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate built-in member %q",
				basespec.ErrInvalid,
				skill.Member,
			)
		}
		seenArtifactIDs[skill.ArtifactID] = struct{}{}
		seenMembers[skill.Member] = struct{}{}

		expectedDigest, expected := attachmentData.ExpectedMemberDigests[skill.Member]
		if !expected {
			return nil, fmt.Errorf(
				"%w: built-in member %q is not declared by attachment integrity data",
				basespec.ErrReferenceUnresolved,
				skill.Member,
			)
		}
		skillMD, exists := filesByLocator[skill.Member]
		if !exists {
			return nil, fmt.Errorf(
				"%w: built-in package lacks member %q",
				basespec.ErrReferenceUnresolved,
				skill.Member,
			)
		}
		if actual := cryptoutil.DigestBytes(skillMD); actual != expectedDigest {
			return nil, fmt.Errorf(
				"%w: built-in member %q does not match attachment integrity data",
				basespec.ErrDigestMismatch,
				skill.Member,
			)
		}

		memberFiles, err := filesForBuiltInMember(
			publication.Files,
			skill.Member,
		)
		if err != nil {
			return nil, err
		}
		normalizedFiles, normalizedSkillMD, err := normalizeManagedSkillFiles(
			skillMD,
			memberFiles,
		)
		if err != nil {
			return nil, err
		}
		expectedName := path.Base(path.Dir(string(skill.Member)))
		definitionValue, _, err := skillArtifact.DecodeSkillDocument(
			normalizedSkillMD,
			expectedName,
		)
		if err != nil {
			return nil, err
		}
		packageSHA256, err := managedSkillPackageDigest(normalizedFiles)
		if err != nil {
			return nil, err
		}
		name := definitionValue.DisplayName
		if name == "" {
			name = string(definitionValue.LogicalName)
		}
		locator := basespec.Locator(path.Join(
			string(request.PackageDirectory),
			string(skill.Member),
		))
		if err := basespec.ValidateLocator(locator, false); err != nil {
			return nil, err
		}

		if _, err := a.ensurePinnedManagedSkill(
			ctx,
			request.Bundle,
			request.ExpectedCollectionRevision,
			skill.ArtifactID,
			sourceValue.ID,
			locator,
			name,
			skill.Enabled,
			packageSHA256,
		); err != nil {
			return nil, err
		}
		prepared = append(prepared, preparedBuiltInSkill{
			request:          skill,
			name:             name,
			locator:          locator,
			definitionDigest: definitionValue.Digest,
			packageSHA256:    packageSHA256,
		})
	}
	for member := range attachmentData.ExpectedMemberDigests {
		if _, found := seenMembers[member]; !found {
			return nil, fmt.Errorf(
				"%w: attachment integrity member %q has no static Artifact",
				basespec.ErrInvalid,
				member,
			)
		}
	}

	plan, err := a.discoveryPlan(bundle)
	if err != nil {
		return nil, pendingBuiltInCollectionInstallError(request.Bundle, err)
	}
	if _, err := a.dependencies.ManagedArtifacts.PublishCollection(
		ctx,
		managedartifact.PublishCollectionRequest{
			Collection:     request.Bundle,
			SourceID:       sourceValue.ID,
			Package:        publication,
			Plan:           plan,
			RefreshPolicy:  builtInSkillArtifactPolicy{},
			AllowProtected: true,
			ForceRefresh:   true,
		},
	); err != nil {
		return nil, pendingBuiltInCollectionInstallError(request.Bundle, err)
	}

	output := make([]CreateManagedSkillResponse, 0, len(prepared))
	for _, value := range prepared {
		resolved, err := a.dependencies.Artifacts.Get(
			ctx,
			artifact.ArtifactRef{
				RootID:     request.Bundle.RootID,
				ArtifactID: value.request.ArtifactID,
			},
		)
		if err != nil {
			return nil, pendingBuiltInCollectionInstallError(request.Bundle, err)
		}
		result, complete := managedSkillCreateResult(
			resolved,
			sourceValue.ID,
			value.locator,
			value.definitionDigest,
			value.packageSHA256,
		)
		if !complete {
			return nil, pendingBuiltInCollectionInstallError(
				request.Bundle,
				fmt.Errorf(
					"%w: built-in Artifact %q did not resolve from its portable member",
					basespec.ErrReferenceUnresolved,
					value.request.ArtifactID,
				),
			)
		}
		output = append(output, result)
	}
	return output, nil
}

func filesForBuiltInMember(
	files []source.ManagedPackageFile,
	member basespec.Locator,
) ([]source.ManagedPackageFile, error) {
	memberDirectory := path.Dir(string(member))
	prefix := memberDirectory + "/"
	output := make([]source.ManagedPackageFile, 0)
	for _, file := range files {
		relative, found := strings.CutPrefix(string(file.Locator), prefix)
		if !found {
			continue
		}
		output = append(output, source.ManagedPackageFile{
			Locator: basespec.Locator(relative),
			Content: append([]byte(nil), file.Content...),
		})
	}
	if len(output) == 0 {
		return nil, fmt.Errorf(
			"%w: member package %q has no files",
			basespec.ErrReferenceUnresolved,
			member,
		)
	}
	normalized, err := source.NormalizeManagedPackagePublication(
		source.ManagedPackagePublication{
			Directory: locatorPackage,
			Files:     output,
		},
	)
	if err != nil {
		return nil, err
	}
	return normalized.Files, nil
}

func (a *API) ensurePinnedManagedSkill(
	ctx context.Context,
	bundle collection.CollectionRef,
	expectedCollectionRevision uint64,
	artifactID basespec.ArtifactID,
	sourceID basespec.SourceID,
	skillLocator basespec.Locator,
	name string,
	enabled bool,
	packageSHA256 cryptoutil.Digest,
) (artifact.Artifact, error) {
	pinned, err := a.managedSkillByID(ctx, bundle.RootID, artifactID)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if pinned == nil {
		localData, err := encodeManagedSkillArtifactData(
			newManagedSkillArtifactData(packageSHA256, enabled),
		)
		if err != nil {
			return artifact.Artifact{}, err
		}
		value, pinErr := a.dependencies.Artifacts.Pin(ctx, artifact.PinRequest{
			ArtifactID:                 artifactID,
			Collection:                 bundle,
			ExpectedCollectionRevision: expectedCollectionRevision,
			Binding: artifact.SourceBinding{
				SourceID:     sourceID,
				Locator:      skillLocator,
				ExpectedKind: skillArtifact.Kind,
			},
			Name:    name,
			Enabled: enabled,
			Data:    localData,
		})
		switch {
		case pinErr == nil:
			pinned = &value
		case errors.Is(pinErr, basespec.ErrConflict):
			pinned, err = a.managedSkillByID(ctx, bundle.RootID, artifactID)
			if err != nil {
				return artifact.Artifact{}, err
			}
			if pinned == nil {
				return artifact.Artifact{}, pinErr
			}
		default:
			return artifact.Artifact{}, pinErr
		}
	}
	if err := validateManagedSkillOperationIntent(
		*pinned,
		bundle,
		artifactID,
		sourceID,
		skillLocator,
	); err != nil {
		return artifact.Artifact{}, err
	}
	if pinned.Name != name {
		updated, err := a.dependencies.Artifacts.SetName(
			ctx,
			pinned.Ref(),
			pinned.Revision,
			name,
		)
		if err != nil {
			return artifact.Artifact{}, err
		}
		pinned = &updated
	}

	intent, err := decodeManagedSkillArtifactData(pinned.Data)
	if err != nil ||
		intent.PackageSHA256 != packageSHA256 ||
		intent.Enabled == nil || *intent.Enabled != enabled {
		localData, err := encodeManagedSkillArtifactData(
			newManagedSkillArtifactData(packageSHA256, enabled),
		)
		if err != nil {
			return artifact.Artifact{}, err
		}
		updated, err := a.dependencies.Artifacts.UpdateData(
			ctx,
			pinned.Ref(),
			pinned.Revision,
			localData,
		)
		if err != nil {
			return artifact.Artifact{}, err
		}
		pinned = &updated
	}

	if pinned.Enabled != enabled {
		updated, err := a.dependencies.Artifacts.SetEnabled(
			ctx,
			pinned.Ref(),
			pinned.Revision,
			enabled,
		)
		if err != nil {
			return artifact.Artifact{}, err
		}
		pinned = &updated
	}
	return pinned.Clone(), nil
}

func pendingBuiltInCollectionInstallError(
	ref collection.CollectionRef,
	cause error,
) error {
	return fmt.Errorf(
		"built-in Collection install for %q remains pending; retry installer convergence: %w",
		ref.CollectionID,
		cause,
	)
}
