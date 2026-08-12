package bundle

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
)

const (
	CollectionDataSchemaVersion = "v1"
	AttachmentDataSchemaVersion = "v1"

	DiscoveryPolicyRevision = "mcp.bundle.discovery.v1"

	RoleManaged basespec.AttachmentRole = "managed"
	RoleBuiltIn basespec.AttachmentRole = "builtin"

	PackageDirectory       basespec.Locator = "package"
	DefaultDocumentLocator basespec.Locator = "package/mcps.json"
)

type CollectionData struct {
	SchemaVersion           string                  `json:"schemaVersion"`
	DiscoveryPolicyRevision string                  `json:"discoveryPolicyRevision"`
	LogicalName             basespec.LogicalName    `json:"logicalName"`
	LogicalVersion          basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	Labels                  map[string]string       `json:"labels,omitempty"`
	ManagedSourceID         basespec.SourceID       `json:"managedSourceID,omitempty"`
}

type AttachmentData struct {
	SchemaVersion   string           `json:"schemaVersion"`
	DocumentLocator basespec.Locator `json:"documentLocator"`
}

type BundleExtension struct {
	Servers  map[string]server.ServerExtension `json:"servers,omitempty"`
	Policies map[string]policy.PolicyDocument  `json:"policies,omitempty"`
}

type BundleDocument struct {
	Kind          basespec.CollectionKind `json:"kind"`
	SchemaID      basespec.SchemaID       `json:"schemaID"`
	SchemaVersion string                  `json:"schemaVersion"`
	Digest        cryptoutil.Digest       `json:"digest,omitempty"`

	LogicalName    basespec.LogicalName    `json:"logicalName"`
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string                  `json:"displayName,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Labels         map[string]string       `json:"labels,omitempty"`

	MCPServers      map[string]server.CoreServer `json:"mcpServers"`
	BundleExtension BundleExtension              `json:"bundleExtension"`
}
