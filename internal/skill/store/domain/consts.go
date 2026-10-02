package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/skillv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	schema "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

const (
	SkillArtifactKind artifact.ArtifactKind = artifact.ArtifactKind(
		skillv1.SkillType,
	)
	ManagedSkillPackageKind           source.PackageKind = "skill"
	BuiltinSkillCollectionPackageKind source.PackageKind = "skill-collection"
	SkillSchemaID                     schema.SchemaID    = skillv1.SkillSchemaID
	MarkdownDecoderID                 spec.DecoderID     = "agent.skill-markdown"

	SkillSchemaVersion     = skillv1.SkillSchemaVersion
	InsertLabelKey         = "skill.insert"
	BuiltInInstallerName   = "agent.skill"
	HydrationSchemaVersion = "agent.skill.builtin-hydration/v1"
)

func SkillDefinitionFileName() spec.Locator {
	return documentTopology.DefaultSkillPackageDocumentFile()
}

func SkillDefinitionFiles() []spec.Locator {
	return documentTopology.SkillPackageDocumentFiles()
}

func IsSkillDefinitionFile(locator spec.Locator) bool {
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
