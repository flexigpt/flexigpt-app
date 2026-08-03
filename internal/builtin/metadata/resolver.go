package metadata

import (
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/builtin"
)

// ResolveBuiltInSkill resolves only protected application metadata. It does
// not inspect user Roots, shareable packages, SQLite paths, or Source state.
func ResolveBuiltInSkill(
	collectionLogicalName string,
	skillLogicalName string,
) (artifact.ArtifactRef, error) {
	collection := basespec.LogicalName(collectionLogicalName)
	skill := basespec.LogicalName(skillLogicalName)
	if err := basespec.ValidateLogicalName(collection); err != nil {
		return artifact.ArtifactRef{}, err
	}
	if err := basespec.ValidateLogicalName(skill); err != nil {
		return artifact.ArtifactRef{}, err
	}

	registry, err := loadEmbeddedHydratedRegistry()
	if err != nil {
		return artifact.ArtifactRef{}, err
	}
	return registry.ResolveSkill(collection, skill)
}

func ResolveBuiltInSkillReference(
	reference SkillReference,
) (artifact.ArtifactRef, error) {
	if err := reference.Validate(); err != nil {
		return artifact.ArtifactRef{}, err
	}

	registry, err := loadEmbeddedHydratedRegistry()
	if err != nil {
		return artifact.ArtifactRef{}, err
	}
	return registry.ResolveSkill(reference.Collection, reference.Skill)
}

func loadEmbeddedHydratedRegistry() (HydratedRegistry, error) {
	registry, err := LoadRegistry()
	if err != nil {
		return HydratedRegistry{}, err
	}
	packages, err := fs.Sub(
		builtin.BuiltInSkillBundlesFS,
		builtin.BuiltInSkillBundlesRootDir,
	)
	if err != nil {
		return HydratedRegistry{}, err
	}
	return registry.Hydrate(packages)
}
