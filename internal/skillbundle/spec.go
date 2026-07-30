package skillbundle

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

const (
	CollectionKind basespec.CollectionKind = "skill.bundle"

	CollectionSchemaVersion = "v1"
	DiscoveryPolicyRevision = "skill.bundle.discovery.v1"

	RoleManaged  basespec.AttachmentRole = "managed"
	RoleBuiltIn  basespec.AttachmentRole = "builtin"
	RoleExternal basespec.AttachmentRole = "external"
	RoleImported basespec.AttachmentRole = "imported"
	RoleLibrary  basespec.AttachmentRole = "library"
)
