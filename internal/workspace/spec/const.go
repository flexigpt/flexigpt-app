package spec

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
)

const (
	CollectionKind                   basespec.CollectionKind = builtinSchema.WorkspaceCollectionV1Kind
	WorkspaceDescriptorSchemaID      basespec.SchemaID       = builtinSchema.WorkspaceCollectionV1SchemaID
	WorkspaceDescriptorSchemaVersion                         = builtinSchema.WorkspaceCollectionV1SchemaVersion

	RolePrimary         basespec.AttachmentRole = "primary"
	RoleLibrary         basespec.AttachmentRole = "library"
	RoleAttachedPackage basespec.AttachmentRole = "attached-package"
	RoleOverlay         basespec.AttachmentRole = "overlay"
)
