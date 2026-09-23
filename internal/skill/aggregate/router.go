package aggregate

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/agentskills-go/provider"
	"github.com/flexigpt/agentskills-go/provider/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/skill/store/materialize"
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

// ResolveArtifactSkills materializes all requested Artifacts in one source
// verification batch. This is the bulk counterpart to ResolveArtifactSkill.
func (r *ArtifactRouter) ResolveArtifactSkills(
	ctx context.Context,
	refs []artifact.ArtifactRef,
) ([]ResolvedArtifactSkill, error) {
	records := make([]artifact.Artifact, 0, len(refs))
	for _, ref := range refs {
		if err := ref.Validate(); err != nil {
			return nil, err
		}
		record, err := r.artifacts.Get(ctx, ref)
		if err != nil {
			return nil, err
		}
		if !skillDomain.IsSkillKind(record.Kind) {
			return nil, fmt.Errorf(
				"%w: Artifact %q is not a Skill",
				basespec.ErrReferenceUnresolved,
				ref.ArtifactID,
			)
		}
		records = append(records, record)
	}
	return r.resolveRecords(ctx, records)
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

	candidates := make([]artifact.Artifact, 0, len(records))
	for _, record := range records {
		if !skillDomain.IsSkillKind(record.Kind) ||
			record.State != artifact.StateAvailable ||
			!record.Enabled {
			continue
		}
		candidates = append(candidates, record)
	}

	output, err := r.resolveRecords(ctx, candidates)
	if err != nil {
		return nil, err
	}

	seen := make(map[provider.SkillDef]artifact.ArtifactRef, len(output))
	for _, value := range output {
		if previous, duplicate := seen[value.Definition]; duplicate &&
			previous != value.Artifact {
			return nil, fmt.Errorf(
				"%w: Artifact Skills %q and %q resolve to one runtime Skill",
				basespec.ErrConflict,
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
	records []artifact.Artifact,
) ([]ResolvedArtifactSkill, error) {
	if len(records) == 0 {
		return []ResolvedArtifactSkill{}, nil
	}

	materials, err := materialize.ResolveAll(ctx, r.resources, records)
	if err != nil {
		return nil, err
	}
	if len(materials) != len(records) {
		return nil, fmt.Errorf(
			"%w: Skill materializer returned an unexpected result count",
			basespec.ErrInvalid,
		)
	}

	output := make([]ResolvedArtifactSkill, 0, len(materials))
	for index, material := range materials {
		//nolint:gosec // Len equality is checked above.
		record := records[index]
		if material.Artifact != record.Ref() {
			return nil, fmt.Errorf(
				"%w: Skill materializer resolved another Artifact",
				basespec.ErrRefreshRequired,
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
	record artifact.Artifact,
) (ResolvedArtifactSkill, error) {
	values, err := r.resolveRecords(ctx, []artifact.Artifact{record})
	if err != nil {
		return ResolvedArtifactSkill{}, err
	}
	if len(values) != 1 {
		return ResolvedArtifactSkill{}, fmt.Errorf(
			"%w: expected one resolved Skill",
			basespec.ErrInvalid,
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
