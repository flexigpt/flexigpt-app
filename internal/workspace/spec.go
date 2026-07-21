package workspace

import "github.com/flexigpt/flexigpt-app/internal/artifactstore"

const (
	RootKind artifactstore.RootKind = "workspace.root"

	RolePrimary         artifactstore.AttachmentRole = "primary"
	RoleBuiltIn         artifactstore.AttachmentRole = "built-in"
	RoleLibrary         artifactstore.AttachmentRole = "library"
	RoleAttachedPackage artifactstore.AttachmentRole = "attached-package"
	RoleOverlay         artifactstore.AttachmentRole = "overlay"

	DefinitionKind      artifactstore.ArtifactKind = "workspace.definition"
	DefinitionSchemaID  artifactstore.SchemaID     = "workspace.definition.v1"
	DefinitionDecoderID artifactstore.DecoderID    = "workspace.definition-json"

	ContextKind      artifactstore.ArtifactKind = "workspace.context"
	ContextSchemaID  artifactstore.SchemaID     = "workspace.context.v1"
	ContextDecoderID artifactstore.DecoderID    = "workspace.context-markdown"

	SkillKind      artifactstore.ArtifactKind = "workspace.skill"
	SkillSchemaID  artifactstore.SchemaID     = "workspace.skill.v1"
	SkillDecoderID artifactstore.DecoderID    = "workspace.skill-markdown"

	CapabilityProfileVersion = "1"
	PrimaryPriority          = 1_000_000

	workspaceSchemaVersionV1 = "1"

	WorkspaceMetadataDirectory = ".flexigpt"
	WorkspaceSkillsDirectory   = ".skills"

	WorkspaceMetadataLocator artifactstore.Locator = WorkspaceMetadataDirectory
	WorkspaceSkillsLocator   artifactstore.Locator = WorkspaceSkillsDirectory
	RepositoryRootLocator    artifactstore.Locator = "."

	WorkspaceDefinitionFileName                          = "workspace.json"
	DefinitionLocator              artifactstore.Locator = WorkspaceMetadataDirectory + "/" + WorkspaceDefinitionFileName
	workspaceDefinitionLogicalName                       = "workspace"
	workspaceDefinitionDisplayName                       = "Workspace"

	contextAgentsFileName = "AGENTS.md"
	contextClaudeFileName = "CLAUDE.md"
	contextReadmeFileName = "README.md"

	ContextAgentsLocator artifactstore.Locator = contextAgentsFileName
	ContextClaudeLocator artifactstore.Locator = contextClaudeFileName
	ContextReadmeLocator artifactstore.Locator = contextReadmeFileName

	contextRoleAgentInstructions     = "agent-instructions"
	contextRoleAssistantInstructions = "assistant-instructions"
	contextRoleProjectReadme         = "project-readme"
	contextRoleProjectContext        = "project-context"
	contextRoleLabelKey              = "context.role"
	contextMarkdownMediaType         = "text/markdown"

	skillDefinitionFileName = "SKILL.md"
	skillInsertLabelKey     = "skill.insert"

	jsonFilePattern      = "*.json"
	yamlFilePattern      = "*.yaml"
	yamlShortFilePattern = "*.yml"
	markdownFilePattern  = "*.md"

	yamlDocumentStart = "---"
	yamlDocumentEnd   = "..."
	yamlMergeKey      = "<<:"

	contextPromptSeparator   = "\n\n"
	contextPromptStartFormat = "<<<WORKSPACE_CONTEXT name=%q role=%q source=%q>>>\n"
	contextPromptEndMarker   = "\n<<<END_WORKSPACE_CONTEXT>>>"

	defaultRecordName        = "artifact"
	recordNameSeparator      = "-"
	recordNameDigestLength   = 12
	exactVersionConstraintOp = "="

	skillFrontmatterNameKey        = "name"
	skillFrontmatterDescriptionKey = "description"
	skillFrontmatterInsertKey      = "insert"
	skillFrontmatterArgumentsKey   = "arguments"
	skillFrontmatterTagsKey        = "tags"
	skillArgumentNameKey           = "name"
	skillArgumentDescriptionKey    = "description"
	skillArgumentDefaultKey        = "default"

	skillNameRepeatedHyphen       = "--"
	skillMarkdownHeadingPrefix    = "# "
	workspaceSkillNamePatternText = `^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`
	maxWorkspaceSkillBytes        = 2 * 1024 * 1024
	maxSkillDescriptionBytes      = 1024
	parentDirectoryPath           = ".."

	restrictedYAMLTopLevelKeyPattern = `^([A-Za-z][A-Za-z0-9_.-]*)[ \t]*:`
)

const (
	diagnosticCodeArtifactInvalid         = "workspace.artifact.invalid"
	diagnosticCodeContextInvalidContent   = "workspace.context.invalid-content"
	diagnosticCodeContextInvalidUTF8      = "workspace.context.invalid-utf8"
	diagnosticCodeDefinitionInvalid       = "workspace.definition.invalid"
	diagnosticCodeRecordSchemaUnsupported = "workspace.record.schema-unsupported"
	diagnosticCodeSkillInvalid            = "workspace.skill.invalid"
)

type contextFileSupport struct {
	fileName string
	role     string
}

var contextFileSupportMatrix = [...]contextFileSupport{
	{
		fileName: contextAgentsFileName,
		role:     contextRoleAgentInstructions,
	},
	{
		fileName: contextClaudeFileName,
		role:     contextRoleAssistantInstructions,
	},
	{
		fileName: contextReadmeFileName,
		role:     contextRoleProjectReadme,
	},
}
