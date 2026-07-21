package engine

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

	CapabilityProfileVersion = "1"
	PrimaryPriority          = 1_000_000

	workspaceSchemaVersionV1 = "1"

	WorkspaceMetadataDirectory = ".flexigpt"

	WorkspaceMetadataLocator artifactstore.Locator = WorkspaceMetadataDirectory
	RepositoryRootLocator    artifactstore.Locator = "."

	WorkspaceDefinitionFileName                          = "workspace.json"
	DefinitionLocator              artifactstore.Locator = WorkspaceMetadataDirectory + "/" + WorkspaceDefinitionFileName
	workspaceDefinitionLogicalName                       = "workspace"
	workspaceDefinitionDisplayName                       = "Workspace"

	jsonFilePattern      = "*.json"
	yamlFilePattern      = "*.yaml"
	yamlShortFilePattern = "*.yml"
	markdownFilePattern  = "*.md"

	defaultRecordName        = "artifact"
	recordNameSeparator      = "-"
	recordNameDigestLength   = 12
	exactVersionConstraintOp = "="
)
