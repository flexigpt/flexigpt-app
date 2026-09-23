// Package materialize resolves verified Skill Artifact material required by
// Skill runtime consumers.
package materialize

import (
	"context"
	"fmt"
	"strconv"

	"github.com/flexigpt/agentskills-go/document"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/consumerutil"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

// ResourceReader is the narrow generic Artifact Store resource capability
// needed to materialize one Skill package.
type ResourceReader interface {
	ResolveArtifact(
		ctx context.Context,
		ref artifact.ArtifactRef,
		options resource.ResolveOptions,
	) (resource.ResolvedArtifact, error)

	ReadSourceEntry(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		locator basespec.Locator,
		maximumBytes int64,
	) (resource.VerifiedEntry, error)

	ResolveVerifiedLocalPath(
		ctx context.Context,
		resolved resource.ResolvedArtifact,
		localLocator basespec.Locator,
	) (string, error)
}

// ResolvedSkill is verified Skill package material. It contains no session,
// Workspace, catalog, or execution state.
type ResolvedSkill struct {
	Artifact         artifact.ArtifactRef
	ArtifactRevision uint64
	DefinitionDigest cryptoutil.Digest
	SourceID         source.SourceID
	Locator          basespec.Locator

	Document        document.SkillDocument
	RuntimeLocation string
	VersionDigest   cryptoutil.Digest
}

// ResolveAll materializes all records under one shared Artifact Store source
// verification session when the supplied ResourceReader supports one.
func ResolveAll(
	ctx context.Context,
	resources ResourceReader,
	records []artifact.Artifact,
) ([]ResolvedSkill, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Skill batch materialization context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if resources == nil {
		return nil, fmt.Errorf(
			"%w: Skill materializer ResourceReader is nil",
			basespec.ErrInvalid,
		)
	}
	if len(records) == 0 {
		return []ResolvedSkill{}, nil
	}

	return consumerutil.WithResourceVerificationSession(
		ctx,
		resources,
		func(sessionCtx context.Context) ([]ResolvedSkill, error) {
			output := make([]ResolvedSkill, 0, len(records))
			for _, record := range records {
				value, err := Resolve(
					sessionCtx,
					resources,
					record,
				)
				if err != nil {
					return nil, err
				}
				output = append(output, value)
			}
			return output, nil
		},
	)
}

func Resolve(
	ctx context.Context,
	resources ResourceReader,
	record artifact.Artifact,
) (ResolvedSkill, error) {
	if ctx == nil {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill materialization context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return ResolvedSkill{}, err
	}
	if resources == nil {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill materializer ResourceReader is nil",
			basespec.ErrInvalid,
		)
	}
	if !skillDomain.IsSkillKind(record.Kind) ||
		record.State != artifact.StateAvailable ||
		record.ResolvedDefinition == nil ||
		record.SourceContentDigest == nil {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	resolved, err := resources.ResolveArtifact(
		ctx,
		record.Ref(),
		resource.ResolveOptions{},
	)
	if err != nil {
		return ResolvedSkill{}, err
	}
	if err := resolved.Validate(); err != nil {
		return ResolvedSkill{}, err
	}
	if resolved.Artifact.Ref() != record.Ref() ||
		resolved.Artifact.Revision != record.Revision ||
		resolved.Artifact.Binding != record.Binding ||
		resolved.Definition.Digest != *record.ResolvedDefinition {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill Artifact changed during resource resolution",
			basespec.ErrRefreshRequired,
		)
	}

	declarationValue, err := skillDomain.SkillDeclarationFromDefinition(
		resolved.Definition,
	)
	if err != nil {
		return ResolvedSkill{}, err
	}
	documentLocator, err := skillDomain.SourceDocumentLocator(
		declarationValue.Locator,
		resolved.Artifact.Binding.Locator,
	)
	if err != nil {
		return ResolvedSkill{}, err
	}

	sourceEntry, err := resources.ReadSourceEntry(
		ctx,
		resolved.Artifact.RootID,
		resolved.Artifact.Binding.SourceID,
		documentLocator,
		basespec.MaxCandidateBytes,
	)
	if err != nil {
		return ResolvedSkill{}, err
	}
	if sourceEntry.SourceRevision != resolved.RefreshState.SourceRevision ||
		sourceEntry.SourceGeneration !=
			resolved.RefreshState.SourceGeneration {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill Source changed during materialization",
			basespec.ErrRefreshRequired,
		)
	}
	if documentLocator == resolved.Artifact.Binding.Locator &&
		sourceEntry.Digest != *resolved.Artifact.SourceContentDigest {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill declaration source changed during materialization",
			basespec.ErrRefreshRequired,
		)
	}

	documentValue, _, err := skillDomain.ParseSkillDocument(
		sourceEntry.Content,
		string(resolved.Definition.LogicalName),
	)
	if err != nil {
		return ResolvedSkill{}, err
	}

	subresource := resolved.Artifact.Binding.SubresourceLocator
	if documentLocator != resolved.Artifact.Binding.Locator {
		subresource = ""
	}
	packageLocator, err := skillDomain.RuntimePackageLocator(
		documentLocator,
		subresource,
	)
	if err != nil {
		return ResolvedSkill{}, err
	}
	location, err := resources.ResolveVerifiedLocalPath(
		ctx,
		resolved,
		packageLocator,
	)
	if err != nil {
		return ResolvedSkill{}, err
	}

	versionInput := string(resolved.Definition.Digest) + "\x00" +
		string(sourceEntry.Digest) + "\x00" +
		resolved.RefreshState.SourceGeneration + "\x00" +
		strconv.FormatUint(resolved.Artifact.Revision, 10)

	return ResolvedSkill{
		Artifact:         resolved.Artifact.Ref(),
		ArtifactRevision: resolved.Artifact.Revision,
		DefinitionDigest: resolved.Definition.Digest,
		SourceID:         resolved.Artifact.Binding.SourceID,
		Locator:          documentLocator,
		Document:         documentValue,
		RuntimeLocation:  location,
		VersionDigest:    cryptoutil.DigestBytes([]byte(versionInput)),
	}, nil
}
