package engine

import "github.com/flexigpt/flexigpt-app/internal/artifactstore"

const (
	CollectionKind                   artifactstore.CollectionKind = "workspace.collection"
	WorkspaceDescriptorSchemaID      artifactstore.SchemaID       = "workspace.collection.v1"
	WorkspaceDescriptorSchemaVersion                              = "v1"

	RolePrimary         artifactstore.AttachmentRole = "primary"
	RoleBuiltIn         artifactstore.AttachmentRole = "built-in"
	RoleLibrary         artifactstore.AttachmentRole = "library"
	RoleAttachedPackage artifactstore.AttachmentRole = "attached-package"
	RoleOverlay         artifactstore.AttachmentRole = "overlay"

	WorkspaceMetadataDirectory                       = ".flexigpt"
	WorkspaceMetadataLocator   artifactstore.Locator = WorkspaceMetadataDirectory
	RepositoryRootLocator      artifactstore.Locator = "."

	WorkspaceDescriptorFileName                       = "workspace.json"
	DescriptorLocator           artifactstore.Locator = WorkspaceMetadataDirectory + "/" + WorkspaceDescriptorFileName
	markdownFilePattern                               = "*.md"

	defaultArtifactName      = "artifact"
	artifactNameSeparator    = "-"
	artifactNameDigestLength = 12
	exactVersionConstraintOp = "="
)
