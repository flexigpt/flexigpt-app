package skilladapter

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/flexigpt/agentskills-go/fsskillprovider"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
)

// RuntimeResolver adapts Workspace-owned Artifact projection to the generic
// Artifact-backed Skill Runtime router. The runtime package does not import
// Workspace and cannot infer Workspace ownership from a reference shape.
type RuntimeResolver struct {
	adapter *Adapter
}

func NewRuntimeResolver(adapter *Adapter) (*RuntimeResolver, error) {
	if adapter == nil {
		return nil, errors.New("workspace skill runtime resolver adapter is nil")
	}
	return &RuntimeResolver{adapter: adapter}, nil
}

func (r *RuntimeResolver) ResolveArtifactSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (skillRuntime.ResolvedArtifactSkill, error) {
	value, err := r.adapter.LoadArtifact(ctx, ref)
	if err != nil {
		return skillRuntime.ResolvedArtifactSkill{}, err
	}
	return workspaceResolvedArtifactSkill(value)
}

func (r *RuntimeResolver) ListCollectionSkills(
	ctx context.Context,
	workspace collection.CollectionRef,
) ([]skillRuntime.ResolvedArtifactSkill, error) {
	values, err := r.adapter.List(ctx, workspace)
	if err != nil {
		return nil, err
	}

	refs := make([]artifact.ArtifactRef, 0, len(values))
	for _, value := range values {
		if !value.ProjectionValid ||
			!value.RuntimePathBacked ||
			!value.WorkspaceEnabled ||
			!value.Skill.IsEnabled ||
			!value.CatalogCurrent ||
			value.RuntimeDisabled ||
			value.State != artifact.StateAvailable {
			continue
		}
		refs = append(refs, value.Artifact)
	}
	if len(refs) == 0 {
		return []skillRuntime.ResolvedArtifactSkill{}, nil
	}

	plan, err := r.adapter.Load(ctx, workspace, refs)
	if err != nil {
		return nil, err
	}
	if len(plan.Skills) != len(refs) {
		return nil, fmt.Errorf(
			"%w: one or more Workspace Skills could not be projected",
			basespec.ErrCatalogStale,
		)
	}

	output := make([]skillRuntime.ResolvedArtifactSkill, 0, len(plan.Skills))
	for _, value := range plan.Skills {
		projected, err := workspaceResolvedArtifactSkill(value)
		if err != nil {
			return nil, err
		}
		output = append(output, projected)
	}
	return output, nil
}

func workspaceResolvedArtifactSkill(
	value WorkspaceSkill,
) (skillRuntime.ResolvedArtifactSkill, error) {
	if !value.ProjectionValid ||
		!value.RuntimePathBacked ||
		!value.WorkspaceEnabled ||
		!value.CatalogCurrent ||
		!value.Skill.IsEnabled ||
		value.RuntimeDisabled ||
		value.State != artifact.StateAvailable {
		return skillRuntime.ResolvedArtifactSkill{}, fmt.Errorf(
			"%w: Workspace Skill is not eligible for runtime registration",
			basespec.ErrCatalogStale,
		)
	}

	versionInput := string(value.DefinitionDigest) + "\x00" +
		string(value.SourceContentDigest) + "\x00" +
		value.SourceGeneration + "\x00" +
		strconv.FormatUint(value.ArtifactRevision, 10)

	output := skillRuntime.ResolvedArtifactSkill{
		Artifact:   value.Artifact,
		Collection: value.Workspace,
		Definition: agentskillsSpec.SkillDef{
			Type:     fsskillprovider.Type,
			Name:     value.Skill.Name,
			Location: value.RuntimeLocation,
		},
		Version: "workspace:" + string(
			cryptoutil.DigestBytes([]byte(versionInput)),
		),
	}
	if err := output.Validate(); err != nil {
		return skillRuntime.ResolvedArtifactSkill{}, err
	}
	return output, nil
}
