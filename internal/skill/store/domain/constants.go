package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/skillv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

const (
	SkillArtifactKind artifact.ArtifactKind = artifact.ArtifactKind(
		skillv1.SkillType,
	)
	SkillSchemaID      schema.SchemaID = skillv1.SkillSchemaID
	SkillSchemaVersion                 = skillv1.SkillSchemaVersion

	MarkdownDecoderID  basespec.DecoderID = "agent.skill-markdown"
	CanonicalDecoderID basespec.DecoderID = "artifact.skill-json"

	ManagedSkillPackageKind source.PackageKind = "skill"
	SkillDefinitionFileName basespec.Locator   = "SKILL.md"

	InsertLabelKey = "skill.insert"

	BuiltInInstallerName   = "agent.skill"
	HydrationSchemaVersion = "agent.skill.builtin-hydration/v2"
	RegistrySchemaVersion  = "v2"
)

func IsSkillKind(value artifact.ArtifactKind) bool {
	return value == SkillArtifactKind
}

func IsSkillSchema(
	value schema.Key,
) bool {
	return value.Entity == schema.EntityArtifact &&
		value.Kind == schema.Kind(SkillArtifactKind) &&
		value.SchemaID == SkillSchemaID &&
		value.SchemaVersion == SkillSchemaVersion
}
