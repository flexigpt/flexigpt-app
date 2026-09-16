package consumerapi

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

// InstallBuiltInSkillPackage publishes one canonical collection package and
// verifies the root Collection and independently declared Skill Artifacts.
//
// Collection membership remains external. Each Skill originates at its own
// SKILL.md source entry and is not a collection subresource Artifact.
func (a *API) InstallBuiltInSkillPackage(
	ctx context.Context,
	request BuiltInSkillPackageInstallRequest,
) ([]artifact.Artifact, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: built-in Skill package context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return nil, err
	}
	if err := request.RootID.Validate(); err != nil {
		return nil, err
	}
	if err := request.SourceID.Validate(); err != nil {
		return nil, err
	}
	if err := request.PackageAddress.Validate(); err != nil {
		return nil, err
	}
	if request.PackageAddress.Kind !=
		skillDomain.BuiltinSkillCollectionPackageKind {
		return nil, fmt.Errorf(
			"%w: built-in Skill package kind must be %q",
			basespec.ErrInvalid,
			skillDomain.BuiltinSkillCollectionPackageKind,
		)
	}
	if request.DocumentFile !=
		skillDomain.BuiltinSkillCollectionDocumentFile {
		return nil, fmt.Errorf(
			"%w: built-in Skill document file must be %q",
			basespec.ErrInvalid,
			skillDomain.BuiltinSkillCollectionDocumentFile,
		)
	}
	if err := request.DocumentFile.ValidatePortable(false); err != nil {
		return nil, err
	}
	if !a.protection.IsProtectedRoot(request.RootID) {
		return nil, fmt.Errorf(
			"%w: built-in Skill Root is not protected",
			basespec.ErrProtected,
		)
	}
	if err := a.requireMutable(ctx, request.RootID, true); err != nil {
		return nil, err
	}

	expectations, rootExpectation, err := normalizeBuiltInSkillPackageExpectations(
		request.Expectations,
	)
	if err != nil {
		return nil, err
	}

	sourceValue, err := a.sources.Get(
		ctx,
		request.RootID,
		request.SourceID,
	)
	if err != nil {
		return nil, err
	}
	if sourceValue.Kind != source.SourceKindManagedDirectory {
		return nil, fmt.Errorf(
			"%w: built-in Skill Source must be managed",
			basespec.ErrInvalid,
		)
	}

	documentLocator, err := request.PackageAddress.FileLocator(
		request.DocumentFile,
	)
	if err != nil {
		return nil, err
	}
	if _, err := a.ensureManagedBuiltinSkillCollectionDiscovery(
		ctx,
		request.RootID,
		request.SourceID,
		documentLocator,
	); err != nil {
		return nil, err
	}

	if _, err := a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: request.RootID,
			Binding: artifact.SourceBinding{
				SourceID:           request.SourceID,
				Locator:            documentLocator,
				SubresourceLocator: rootExpectation.Subresource,
			},
			ExpectedKind:        rootExpectation.Kind,
			ExpectedLogicalName: rootExpectation.LogicalName,
			ExpectedDefinition:  rootExpectation.DefinitionDigest,
			Package: source.ManagedPackagePublication{
				Address: request.PackageAddress,
				Files:   request.PackageFiles,
			},
			AllowPackageReplacement: true,
			AllowProtected:          true,
		},
	); err != nil {
		return nil, err
	}

	output := make([]artifact.Artifact, 0, len(expectations))
	for _, expected := range expectations {
		originLocator, err := request.PackageAddress.FileLocator(
			expected.Locator,
		)
		if err != nil {
			return nil, err
		}

		record, err := a.artifacts.FindByOrigin(
			ctx,
			request.RootID,
			artifact.SourceBinding{
				SourceID:           request.SourceID,
				Locator:            originLocator,
				SubresourceLocator: expected.Subresource,
			},
			expected.Kind,
		)
		if err != nil {
			return nil, err
		}
		if record.State != artifact.StateAvailable ||
			record.LogicalName != expected.LogicalName ||
			record.ResolvedDefinition == nil ||
			*record.ResolvedDefinition != expected.DefinitionDigest {
			return nil, fmt.Errorf(
				"%w: built-in Skill Artifact %q does not match package expectation",
				basespec.ErrReferenceUnresolved,
				record.ID,
			)
		}
		if record.Enabled != expected.Enabled {
			record, err = a.artifacts.SetEnabled(
				ctx,
				record.Ref(),
				record.Revision,
				expected.Enabled,
			)
			if err != nil {
				return nil, err
			}
		}
		output = append(output, record)
	}

	return output, nil
}

func (a *API) RemoveBuiltInSkillPackage(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	address source.ManagedPackageAddress,
) error {
	if a == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	if err := rootID.Validate(); err != nil {
		return err
	}
	if err := sourceID.Validate(); err != nil {
		return err
	}
	if err := address.Validate(); err != nil {
		return err
	}
	if address.Kind != skillDomain.BuiltinSkillCollectionPackageKind {
		return fmt.Errorf(
			"%w: built-in Skill package kind must be %q",
			basespec.ErrInvalid,
			skillDomain.BuiltinSkillCollectionPackageKind,
		)
	}
	if !a.protection.IsProtectedRoot(rootID) {
		return fmt.Errorf(
			"%w: built-in Skill Root is not protected",
			basespec.ErrProtected,
		)
	}
	if err := a.requireMutable(ctx, rootID, true); err != nil {
		return err
	}
	return a.managedArtifacts.Remove(
		ctx,
		artifact.RemoveArtifactRequest{
			RootID:         rootID,
			SourceID:       sourceID,
			Package:        address,
			AllowProtected: true,
		},
	)
}

func normalizeBuiltInSkillPackageExpectations(
	values []BuiltInSkillArtifactExpectation,
) (
	expectations []BuiltInSkillArtifactExpectation,
	rootExpectation BuiltInSkillArtifactExpectation,
	err error,
) {
	if len(values) == 0 ||
		len(values) > basespec.MaxDiscoveryEntries {
		return nil, BuiltInSkillArtifactExpectation{}, fmt.Errorf(
			"%w: built-in Skill package has invalid Artifact expectation count",
			basespec.ErrInvalid,
		)
	}

	type origin struct {
		locator     basespec.Locator
		subresource basespec.SubresourceLocator
		kind        artifact.ArtifactKind
	}

	output := append(
		[]BuiltInSkillArtifactExpectation(nil),
		values...,
	)
	seen := make(map[origin]struct{}, len(output))

	var (
		r         BuiltInSkillArtifactExpectation
		rootFound bool
	)
	for index, expected := range output {
		if err := expected.Locator.ValidatePortable(false); err != nil {
			return nil, BuiltInSkillArtifactExpectation{}, err
		}
		if err := expected.Subresource.Validate(); err != nil {
			return nil, BuiltInSkillArtifactExpectation{}, fmt.Errorf(
				"built-in Skill package expectations[%d]: %w",
				index,
				err,
			)
		}
		if err := expected.Kind.Validate(); err != nil {
			return nil, BuiltInSkillArtifactExpectation{}, fmt.Errorf(
				"built-in Skill package expectations[%d]: %w",
				index,
				err,
			)
		}
		if err := expected.LogicalName.Validate(); err != nil {
			return nil, BuiltInSkillArtifactExpectation{}, fmt.Errorf(
				"built-in Skill package expectations[%d]: %w",
				index,
				err,
			)
		}
		if err := cryptoutil.ValidateDigest(
			expected.DefinitionDigest,
		); err != nil {
			return nil, BuiltInSkillArtifactExpectation{}, fmt.Errorf(
				"built-in Skill package expectations[%d]: %w",
				index,
				err,
			)
		}

		key := origin{
			locator:     expected.Locator,
			subresource: expected.Subresource,
			kind:        expected.Kind,
		}
		if _, duplicate := seen[key]; duplicate {
			return nil, BuiltInSkillArtifactExpectation{}, fmt.Errorf(
				"%w: built-in Skill package repeats Artifact origin %q/%q",
				basespec.ErrInvalid,
				expected.Subresource,
				expected.Kind,
			)
		}
		seen[key] = struct{}{}

		if expected.Locator != skillDomain.BuiltinSkillCollectionDocumentFile ||
			expected.Subresource != "" {
			continue
		}
		if rootFound {
			return nil, BuiltInSkillArtifactExpectation{}, fmt.Errorf(
				"%w: built-in Skill package has multiple root Artifacts",
				basespec.ErrInvalid,
			)
		}
		if expected.Kind != artifact.ArtifactKind(
			declaration.TypeCollection,
		) {
			return nil, BuiltInSkillArtifactExpectation{}, fmt.Errorf(
				"%w: built-in Skill package root must be a Collection",
				basespec.ErrInvalid,
			)
		}
		r = expected
		rootFound = true
	}
	if !rootFound {
		return nil, BuiltInSkillArtifactExpectation{}, fmt.Errorf(
			"%w: built-in Skill package has no root Collection Artifact",
			basespec.ErrInvalid,
		)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Locator != output[right].Locator {
			return output[left].Locator < output[right].Locator
		}
		if output[left].Subresource != output[right].Subresource {
			return output[left].Subresource <
				output[right].Subresource
		}
		return output[left].Kind < output[right].Kind
	})
	return output, r, nil
}

func (a *API) ensureManagedBuiltinSkillCollectionDiscovery(
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
			"%w: built-in Skill Source must have kind %q",
			basespec.ErrInvalid,
			source.SourceKindManagedDirectory,
		)
	}
	if !value.Enabled {
		return source.Summary{}, fmt.Errorf(
			"%w: built-in Skill Source is disabled",
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
			decoder.YAMLDecoderID,
		) {
		next.AllowedDecoderIDs = append(
			next.AllowedDecoderIDs,
			decoder.YAMLDecoderID,
		)
	}
	next = next.Normalized()
	if err := next.Validate(); err != nil {
		return source.Summary{}, err
	}
	if value.Discovery.Equal(next) {
		return value, nil
	}

	// Do not add the SKILL.md decoder here. "collection.yaml" is the sole
	// declaration origin for this package. SKILL.md files are source-backed
	// resources reached through the nested Skill path locators.
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
