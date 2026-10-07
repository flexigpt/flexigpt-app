package source

import (
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	skillv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/contract/v1"
)

const (
	SkillArtifactKind artifactModel.ArtifactKind = artifactModel.ArtifactKind(
		skillv1.SkillType,
	)
	ManagedSkillPackageKind       managedpackageModel.PackageKind = "skill"
	BuiltinSkillPluginPackageKind managedpackageModel.PackageKind = "skill-plugin"
	SkillSchemaID                 schemaModel.SchemaID            = skillv1.SkillSchemaID
	MarkdownDecoderID             spec.DecoderID                  = "agent.skill-markdown"

	SkillSchemaVersion     = skillv1.SkillSchemaVersion
	InsertLabelKey         = "skill.insert"
	BuiltInInstallerName   = "agent.skill"
	HydrationSchemaVersion = "agent.skill.builtin-hydration/v1"
)

func SkillDefinitionFileName(
	documents support.Documents,
) spec.Locator {
	return documents.Default.Locator
}

func SkillDefinitionFiles(
	documents support.Documents,
) []spec.Locator {
	return append([]spec.Locator(nil), documents.Files...)
}

func IsSkillDefinitionFile(
	documents support.Documents,
	locator spec.Locator,
) bool {
	return documents.Matches(locator)
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
