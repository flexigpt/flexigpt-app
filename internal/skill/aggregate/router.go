package aggregate

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/flexigpt/agentskills-go/provider"
	"github.com/flexigpt/agentskills-go/provider/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

// ArtifactRouter is the flat Root-scoped Skill Artifact resolver.
//
// It intentionally does not infer Skill ownership from Collection membership.
type ArtifactRouter struct {
	artifacts compositionapi.ArtifactAPI
	resources compositionapi.ResourceAPI
}

func NewArtifactRouter(
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
) (*ArtifactRouter, error) {
	if artifacts == nil || resources == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Skill router dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &ArtifactRouter{
		artifacts: artifacts,
		resources: resources,
	}, nil
}

func (r *ArtifactRouter) RootForArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (root.RootID, error) {
	if err := ref.Validate(); err != nil {
		return "", err
	}
	value, err := r.artifacts.Get(ctx, ref)
	if err != nil {
		return "", err
	}
	if !skillDomain.IsSkillKind(value.Kind) {
		return "", fmt.Errorf(
			"%w: Artifact %q is not a Skill",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	return value.RootID, nil
}

func (r *ArtifactRouter) ResolveArtifactSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ResolvedArtifactSkill, error) {
	if err := ref.Validate(); err != nil {
		return ResolvedArtifactSkill{}, err
	}
	record, err := r.artifacts.Get(ctx, ref)
	if err != nil {
		return ResolvedArtifactSkill{}, err
	}
	return r.resolveRecord(ctx, record)
}

func (r *ArtifactRouter) ListRootSkills(
	ctx context.Context,
	rootID root.RootID,
) ([]ResolvedArtifactSkill, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	records, err := r.artifacts.ListByRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}

	output := make([]ResolvedArtifactSkill, 0, len(records))
	for _, record := range records {
		if !skillDomain.IsSkillKind(record.Kind) ||
			!record.Enabled ||
			record.State != artifact.StateAvailable {
			continue
		}
		value, err := r.resolveRecord(ctx, record)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].Definition.Name != output[right].Definition.Name {
			return output[left].Definition.Name <
				output[right].Definition.Name
		}
		return output[left].Artifact.ArtifactID <
			output[right].Artifact.ArtifactID
	})
	return output, nil
}

func (r *ArtifactRouter) resolveRecord(
	ctx context.Context,
	record artifact.Artifact,
) (ResolvedArtifactSkill, error) {
	if !skillDomain.IsSkillKind(record.Kind) ||
		!record.Enabled ||
		record.State != artifact.StateAvailable ||
		record.ResolvedDefinition == nil ||
		record.SourceContentDigest == nil {
		return ResolvedArtifactSkill{}, fmt.Errorf(
			"%w: Skill Artifact %q is not enabled and available",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	resolved, err := r.resources.ResolveArtifact(
		ctx,
		record.Ref(),
		resource.ResolveOptions{},
	)
	if err != nil {
		return ResolvedArtifactSkill{}, err
	}
	if resolved.Artifact.Revision != record.Revision ||
		resolved.Artifact.Binding != record.Binding ||
		resolved.Definition.Digest != *record.ResolvedDefinition {
		return ResolvedArtifactSkill{}, fmt.Errorf(
			"%w: Skill Artifact changed during resource resolution",
			basespec.ErrRefreshRequired,
		)
	}
	if err := skillDomain.ValidateDefinition(resolved.Definition); err != nil {
		return ResolvedArtifactSkill{}, err
	}

	packageLocator, err := skillDomain.RuntimePackageLocator(
		record.Binding.Locator,
		record.Binding.SubresourceLocator,
	)
	if err != nil {
		return ResolvedArtifactSkill{}, err
	}
	location, err := r.resources.ResolveVerifiedLocalPath(
		ctx,
		resolved,
		packageLocator,
	)
	if err != nil {
		return ResolvedArtifactSkill{}, err
	}

	versionInput := string(resolved.Definition.Digest) + "\x00" +
		string(*record.SourceContentDigest) + "\x00" +
		resolved.RefreshState.SourceGeneration + "\x00" +
		strconv.FormatUint(record.Revision, 10)
	value := ResolvedArtifactSkill{
		Artifact: record.Ref(),
		Definition: provider.SkillDef{
			Type:     fs.Type,
			Name:     string(resolved.Definition.LogicalName),
			Location: location,
		},
		Version: "artifact-skill:" + string(
			cryptoutil.DigestBytes([]byte(versionInput)),
		),
	}
	if err := value.Validate(); err != nil {
		return ResolvedArtifactSkill{}, err
	}
	return value, nil
}

func (s ResolvedArtifactSkill) Validate() error {
	if err := s.Artifact.Validate(); err != nil {
		return err
	}
	if s.Definition.Type == "" ||
		s.Definition.Name == "" ||
		s.Definition.Location == "" {
		return fmt.Errorf(
			"%w: runtime Skill definition is incomplete",
			basespec.ErrInvalid,
		)
	}
	if s.Version == "" {
		return fmt.Errorf(
			"%w: runtime Skill version is required",
			basespec.ErrInvalid,
		)
	}
	return nil
}
