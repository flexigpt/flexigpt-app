package skillbundle

import (
	"context"
	"errors"

	"github.com/flexigpt/agentskills-go/fsskillprovider"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/skillruntime"
)

// RuntimeResolver is the Skill Bundle feature adapter registered with the
// Artifact-backed Skill Runtime router. It owns only skill.bundle projection;
// collection ownership itself is resolved by the generic router.
type RuntimeResolver struct {
	api *API
}

func NewRuntimeResolver(api *API) (*RuntimeResolver, error) {
	if api == nil {
		return nil, errors.New("skill bundle runtime resolver API is nil")
	}
	return &RuntimeResolver{api: api}, nil
}

func (r *RuntimeResolver) ResolveArtifactSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (skillruntime.ResolvedArtifactSkill, error) {
	value, err := r.api.LoadRuntimeSkill(ctx, ref)
	if err != nil {
		return skillruntime.ResolvedArtifactSkill{}, err
	}
	return resolvedArtifactSkillOf(value)
}

func (r *RuntimeResolver) ListCollectionSkills(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]skillruntime.ResolvedArtifactSkill, error) {
	values, err := r.api.ListRuntimeSkills(ctx, ref)
	if err != nil {
		return nil, err
	}

	output := make([]skillruntime.ResolvedArtifactSkill, 0, len(values))
	for _, value := range values {
		projected, err := resolvedArtifactSkillOf(value)
		if err != nil {
			return nil, err
		}
		output = append(output, projected)
	}
	return output, nil
}

func resolvedArtifactSkillOf(
	value RuntimeSkill,
) (skillruntime.ResolvedArtifactSkill, error) {
	output := skillruntime.ResolvedArtifactSkill{
		Artifact:   value.Artifact,
		Collection: value.Collection,
		Definition: agentskillsSpec.SkillDef{
			Type:     fsskillprovider.Type,
			Name:     value.Name,
			Location: value.Location,
		},
		Version: value.Version,
	}
	if err := output.Validate(); err != nil {
		return skillruntime.ResolvedArtifactSkill{}, err
	}
	return output, nil
}
