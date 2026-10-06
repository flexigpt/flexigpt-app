package aggregate

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/agentskills-go/provider"
	"github.com/flexigpt/agentskills-go/provider/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/materialize"
	skillSource "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/source"
)

// ArtifactRouter is the flat Root-scoped Skill Artifact resolver.
//
// It intentionally does not infer Skill ownership from Plugin membership.
type ArtifactRouter struct {
	artifacts       artifact.API
	cat             catalog.API
	resources       resourceFlow.API
	nativeResources resourceFlow.NativePathAPI
}

func NewArtifactRouter(
	artifacts artifact.API,
	cat catalog.API,
	resources resourceFlow.API,
	nativeResources resourceFlow.NativePathAPI,
) (*ArtifactRouter, error) {
	if artifacts == nil ||
		cat == nil ||
		resources == nil ||
		nativeResources == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Skill router dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	return &ArtifactRouter{
		artifacts:       artifacts,
		cat:             cat,
		resources:       resources,
		nativeResources: nativeResources,
	}, nil
}

// materializeResources is a trusted local composition value used only by
// Skill materialization. Ordinary ArtifactRouter operations retain the
// narrower portable resource.API capability.
type materializeResources struct {
	resourceFlow.API
	resourceFlow.NativePathAPI
}

func (r *ArtifactRouter) RootForArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (rootModel.RootID, error) {
	if err := ref.Validate(); err != nil {
		return "", err
	}
	value, err := r.artifacts.Get(ctx, ref)
	if err != nil {
		return "", err
	}
	if !skillSource.IsSkillKind(value.Kind) {
		return "", fmt.Errorf(
			"%w: Artifact %q is not a Skill",
			spec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	return value.RootID, nil
}

func (r *ArtifactRouter) ResolveArtifactSkill(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
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

// ResolveArtifactSkills materializes all requested Artifacts in one source
// verification batch. This is the bulk counterpart to ResolveArtifactSkill.
func (r *ArtifactRouter) ResolveArtifactSkills(
	ctx context.Context,
	refs []artifactModel.ArtifactRef,
) ([]ResolvedArtifactSkill, error) {
	for _, ref := range refs {
		if err := ref.Validate(); err != nil {
			return nil, err
		}
	}
	records, err := r.artifacts.GetMany(ctx, refs)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if !skillSource.IsSkillKind(record.Kind) {
			return nil, fmt.Errorf(
				"%w: Artifact %q is not a Skill",
				spec.ErrReferenceUnresolved,
				record.ID,
			)
		}
	}
	return r.resolveRecords(ctx, records)
}

func (r *ArtifactRouter) ListRootSkills(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]ResolvedArtifactSkill, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	entries, err := r.cat.ListByRoot(
		ctx,
		rootID,
		catalogModel.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	refs := make([]artifactModel.ArtifactRef, 0, len(entries))
	for _, entry := range entries {
		if !skillSource.IsSkillKind(entry.Kind) ||
			entry.State != artifactModel.StateAvailable ||
			!entry.Enabled {
			continue
		}
		refs = append(refs, entry.Ref())
	}
	candidates, err := r.artifacts.GetMany(ctx, refs)
	if err != nil {
		return nil, err
	}

	output, err := r.resolveRecords(ctx, candidates)
	if err != nil {
		return nil, err
	}

	seen := make(map[provider.SkillDef]artifactModel.ArtifactRef, len(output))
	for _, value := range output {
		if previous, duplicate := seen[value.Definition]; duplicate &&
			previous != value.Artifact {
			return nil, fmt.Errorf(
				"%w: Artifact Skills %q and %q resolve to one runtime Skill",
				spec.ErrConflict,
				previous.ArtifactID,
				value.Artifact.ArtifactID,
			)
		}
		seen[value.Definition] = value.Artifact
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Definition.Type != output[right].Definition.Type {
			return output[left].Definition.Type <
				output[right].Definition.Type
		}
		if output[left].Definition.Name != output[right].Definition.Name {
			return output[left].Definition.Name <
				output[right].Definition.Name
		}
		if output[left].Definition.Location != output[right].Definition.Location {
			return output[left].Definition.Location <
				output[right].Definition.Location
		}
		return output[left].Artifact.ArtifactID <
			output[right].Artifact.ArtifactID
	})
	return output, nil
}

func (r *ArtifactRouter) resolveRecords(
	ctx context.Context,
	records []artifactModel.Artifact,
) ([]ResolvedArtifactSkill, error) {
	if len(records) == 0 {
		return []ResolvedArtifactSkill{}, nil
	}

	materials, err := materialize.ResolveAll(
		ctx,
		materializeResources{
			API:           r.resources,
			NativePathAPI: r.nativeResources,
		},
		records,
	)
	if err != nil {
		return nil, err
	}
	if len(materials) != len(records) {
		return nil, fmt.Errorf(
			"%w: Skill materializer returned an unexpected result count",
			spec.ErrInvalid,
		)
	}

	output := make([]ResolvedArtifactSkill, 0, len(materials))
	for index, material := range materials {
		//nolint:gosec // Len equality is checked above.
		record := records[index]
		if material.Artifact != record.Ref() {
			return nil, fmt.Errorf(
				"%w: Skill materializer resolved another Artifact",
				spec.ErrRefreshRequired,
			)
		}

		value := ResolvedArtifactSkill{
			Artifact: record.Ref(),
			Definition: provider.SkillDef{
				Type:     fs.Type,
				Name:     material.Document.Name,
				Location: material.RuntimeLocation,
			},
			Version: "artifact-skill:" + string(material.VersionDigest),
			Enabled: record.Enabled,
		}
		if err := value.Validate(); err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	return output, nil
}

func (r *ArtifactRouter) resolveRecord(
	ctx context.Context,
	record artifactModel.Artifact,
) (ResolvedArtifactSkill, error) {
	values, err := r.resolveRecords(ctx, []artifactModel.Artifact{record})
	if err != nil {
		return ResolvedArtifactSkill{}, err
	}
	if len(values) != 1 {
		return ResolvedArtifactSkill{}, fmt.Errorf(
			"%w: expected one resolved Skill",
			spec.ErrInvalid,
		)
	}
	return values[0], nil
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
			spec.ErrInvalid,
		)
	}
	if s.Version == "" {
		return fmt.Errorf(
			"%w: runtime Skill version is required",
			spec.ErrInvalid,
		)
	}
	return nil
}
