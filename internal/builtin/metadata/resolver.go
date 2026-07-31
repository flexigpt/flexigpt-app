package metadata

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// ResolveBuiltInSkill resolves only protected application metadata. It does
// not inspect user Roots, shareable packages, SQLite paths, or Source state.
func ResolveBuiltInSkill(
	bundleLogicalName string,
	skillLogicalName string,
) (artifact.ArtifactRef, error) {
	bundle := basespec.LogicalName(bundleLogicalName)
	skill := basespec.LogicalName(skillLogicalName)
	if err := basespec.ValidateLogicalName(bundle); err != nil {
		return artifact.ArtifactRef{}, err
	}
	if err := basespec.ValidateLogicalName(skill); err != nil {
		return artifact.ArtifactRef{}, err
	}

	registry, err := LoadRegistry()
	if err != nil {
		return artifact.ArtifactRef{}, err
	}
	return registry.ResolveSkill(bundle, skill)
}

func ResolveBuiltInSkillReference(
	reference SkillReference,
) (artifact.ArtifactRef, error) {
	if err := reference.Validate(); err != nil {
		return artifact.ArtifactRef{}, err
	}

	registry, err := LoadRegistry()
	if err != nil {
		return artifact.ArtifactRef{}, err
	}
	return registry.ResolveSkill(reference.Bundle, reference.Skill)
}
