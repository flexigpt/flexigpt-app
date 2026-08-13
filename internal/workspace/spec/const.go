package spec

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

const (
	CollectionKind                   basespec.CollectionKind = artifactbuiltin.WorkspaceCollectionV1Kind
	WorkspaceDescriptorSchemaID      basespec.SchemaID       = artifactbuiltin.WorkspaceCollectionV1SchemaID
	WorkspaceDescriptorSchemaVersion                         = artifactbuiltin.WorkspaceCollectionV1SchemaVersion

	RolePrimary         basespec.AttachmentRole = "primary"
	RoleLibrary         basespec.AttachmentRole = "library"
	RoleAttachedPackage basespec.AttachmentRole = "attached-package"
	RoleOverlay         basespec.AttachmentRole = "overlay"
)
