package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/skillv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

const (
	SkillArtifactKind artifact.ArtifactKind = artifact.ArtifactKind(
		skillv1.SkillType,
	)
	ManagedSkillPackageKind           source.PackageKind = "skill"
	BuiltinSkillCollectionPackageKind source.PackageKind = "skill-collection"
	SkillSchemaID                     schema.SchemaID    = skillv1.SkillSchemaID
	MarkdownDecoderID                 basespec.DecoderID = "agent.skill-markdown"

	SkillSchemaVersion     = skillv1.SkillSchemaVersion
	InsertLabelKey         = "skill.insert"
	BuiltInInstallerName   = "agent.skill"
	HydrationSchemaVersion = "agent.skill.builtin-hydration/v1"
)

func SkillDefinitionFileName() basespec.Locator {
	return documentTopology.DefaultSkillPackageDocumentFile()
}

func SkillDefinitionFiles() []basespec.Locator {
	return documentTopology.SkillPackageDocumentFiles()
}

func IsSkillDefinitionFile(locator basespec.Locator) bool {
	return documentTopology.IsSkillPackageDocument(locator)
}

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
