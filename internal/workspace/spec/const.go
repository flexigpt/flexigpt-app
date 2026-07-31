package spec

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"

const (
	CollectionKind                   basespec.CollectionKind = "workspace.collection"
	WorkspaceDescriptorSchemaID      basespec.SchemaID       = "workspace.collection.v1"
	WorkspaceDescriptorSchemaVersion                         = "v1"

	RolePrimary         basespec.AttachmentRole = "primary"
	RoleBuiltIn         basespec.AttachmentRole = "builtin"
	RoleLibrary         basespec.AttachmentRole = "library"
	RoleAttachedPackage basespec.AttachmentRole = "attached-package"
	RoleOverlay         basespec.AttachmentRole = "overlay"

	WorkspaceMetadataDirectory                  = ".flexigpt"
	WorkspaceMetadataLocator   basespec.Locator = WorkspaceMetadataDirectory
	RepositoryRootLocator      basespec.Locator = "."

	WorkspaceDescriptorFileName                  = "workspace.json"
	DescriptorLocator           basespec.Locator = WorkspaceMetadataDirectory + "/" + WorkspaceDescriptorFileName
)
