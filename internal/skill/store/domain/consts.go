package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/skillv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

const (
	SkillArtifactKind artifactModel.ArtifactKind = artifactModel.ArtifactKind(
		skillv1.SkillType,
	)
	ManagedSkillPackageKind           sourceModel.PackageKind = "skill"
	BuiltinSkillCollectionPackageKind sourceModel.PackageKind = "skill-collection"
	SkillSchemaID                     schemaModel.SchemaID    = skillv1.SkillSchemaID
	MarkdownDecoderID                 spec.DecoderID          = "agent.skill-markdown"

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

func IsSkillKind(value artifactModel.ArtifactKind) bool {
	return value == SkillArtifactKind
}

func IsSkillSchema(
	value schemaModel.Key,
) bool {
	return value.Entity == schemaModel.EntityArtifact &&
		value.Kind == schemaModel.Kind(SkillArtifactKind) &&
		value.SchemaID == SkillSchemaID &&
		value.SchemaVersion == SkillSchemaVersion
}
