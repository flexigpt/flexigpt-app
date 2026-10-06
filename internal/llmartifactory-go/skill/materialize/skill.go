// Package materialize resolves verified Skill Artifact material required by
// Skill runtime consumers.
package materialize

import (
	"context"
	"fmt"
	"strconv"

	"github.com/flexigpt/agentskills-go/document"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillSource "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/source"
)

// ResourceReader is the narrow generic Artifact Store resource capability
// needed to materialize one Skill package.
type ResourceReader interface {
	// ResolveAll requires an explicit shared verification session. Native-path
	// access remains separately required below because Skill materialization is
	// trusted local runtime composition.
	resourceFlow.SessionAPI

	ResolveArtifact(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		options resourceModel.ResolveOptions,
	) (resourceModel.ResolvedArtifact, error)

	ReadSourceEntry(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		locator spec.Locator,
		maximumBytes int64,
	) (resourceModel.VerifiedEntry, error)

	ResolveVerifiedLocalPath(
		ctx context.Context,
		resolved resourceModel.ResolvedArtifact,
		localLocator spec.Locator,
	) (string, error)
}

// ResolvedSkill is verified Skill package material. It contains no session,
// Workspace, catalog, or execution state.
type ResolvedSkill struct {
	Artifact         artifactModel.ArtifactRef
	ArtifactRevision uint64
	DefinitionDigest cryptoutil.Digest
	SourceID         sourceModel.SourceID
	Locator          spec.Locator

	Document        document.SkillDocument
	RuntimeLocation string
	VersionDigest   cryptoutil.Digest
}

// ResolveAll materializes all records under one shared Artifact Store source
// verification session when the supplied ResourceReader supports one.
func ResolveAll(
	ctx context.Context,
	resources ResourceReader,
	records []artifactModel.Artifact,
) ([]ResolvedSkill, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Skill batch materialization context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if resources == nil {
		return nil, fmt.Errorf(
			"%w: Skill materializer ResourceReader is nil",
			spec.ErrInvalid,
		)
	}
	if len(records) == 0 {
		return []ResolvedSkill{}, nil
	}

	return resourceFlow.WithVerificationSession(
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
	record artifactModel.Artifact,
) (ResolvedSkill, error) {
	if ctx == nil {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill materialization context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return ResolvedSkill{}, err
	}
	if resources == nil {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill materializer ResourceReader is nil",
			spec.ErrInvalid,
		)
	}
	if !skillSource.IsSkillKind(record.Kind) ||
		record.State != artifactModel.StateAvailable ||
		record.ResolvedDefinition == nil ||
		record.SourceContentDigest == nil {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	resolved, err := resources.ResolveArtifact(
		ctx,
		record.Ref(),
		resourceModel.ResolveOptions{},
	)
	if err != nil {
		return ResolvedSkill{}, err
	}
	if resolved.Artifact.Ref() != record.Ref() ||
		resolved.Artifact.Revision != record.Revision ||
		resolved.Artifact.Binding != record.Binding ||
		resolved.Definition.Digest != *record.ResolvedDefinition {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill Artifact changed during resource resolution",
			spec.ErrRefreshRequired,
		)
	}

	declarationValue, err := skillSource.SkillDeclarationFromDefinition(
		resolved.Definition,
	)
	if err != nil {
		return ResolvedSkill{}, err
	}
	documentLocator, err := skillSource.SourceDocumentLocator(
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
		spec.MaxCandidateBytes,
	)
	if err != nil {
		return ResolvedSkill{}, err
	}
	if sourceEntry.SourceRevision != resolved.RefreshState.SourceRevision ||
		sourceEntry.SourceGeneration !=
			resolved.RefreshState.SourceGeneration {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill Source changed during materialization",
			spec.ErrRefreshRequired,
		)
	}
	if documentLocator == resolved.Artifact.Binding.Locator &&
		sourceEntry.Digest != *resolved.Artifact.SourceContentDigest {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill declaration source changed during materialization",
			spec.ErrRefreshRequired,
		)
	}

	documentValue, _, err := skillSource.ParseSkillDocument(
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
	packageLocator, err := skillSource.RuntimePackageLocator(
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
