package bundle

import (
	"context"
	"errors"

	"github.com/flexigpt/agentskills-go/provider"
	"github.com/flexigpt/agentskills-go/provider/fs"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
)

// StoreLoader is the Skill Bundle feature adapter registered with the
// Artifact-backed Skill Runtime router. It owns only skill.bundle projection;
// collection ownership itself is resolved by the generic router.
type StoreLoader struct {
	api *API
}

func NewStoreLoader(api *API) (*StoreLoader, error) {
	if api == nil {
		return nil, errors.New("skill bundle runtime resolver API is nil")
	}
	return &StoreLoader{api: api}, nil
}

func (r *StoreLoader) ResolveArtifactSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (skillStore.ResolvedArtifactSkill, error) {
	value, err := r.api.ResolveSkill(ctx, ref)
	if err != nil {
		return skillStore.ResolvedArtifactSkill{}, err
	}
	return resolvedArtifactSkillOf(value)
}

func (r *StoreLoader) ListCollectionSkills(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]skillStore.ResolvedArtifactSkill, error) {
	values, err := r.api.ListResolvedSkills(ctx, ref)
	if err != nil {
		return nil, err
	}

	output := make([]skillStore.ResolvedArtifactSkill, 0, len(values))
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
	value ResolvedSkill,
) (skillStore.ResolvedArtifactSkill, error) {
	output := skillStore.ResolvedArtifactSkill{
		Artifact:   value.Artifact,
		Collection: value.Collection,
		Definition: provider.SkillDef{
			Type:     fs.Type,
			Name:     value.Name,
			Location: value.Location,
		},
		Version: value.Version,
	}
	if err := output.Validate(); err != nil {
		return skillStore.ResolvedArtifactSkill{}, err
	}
	return output, nil
}
