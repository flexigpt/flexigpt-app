package store

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	mcpStorePolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/policy"
	mcpStoreServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/server"
)

type CollectionData struct {
	SchemaVersion           string                  `json:"schemaVersion"`
	DiscoveryPolicyRevision string                  `json:"discoveryPolicyRevision"`
	LogicalName             basespec.LogicalName    `json:"logicalName"`
	LogicalVersion          basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	Labels                  map[string]string       `json:"labels,omitempty"`
	ManagedSourceID         basespec.SourceID       `json:"managedSourceID,omitempty"`
}

type BundleExtension struct {
	Servers  map[string]mcpStoreServer.ServerExtension `json:"servers,omitempty"`
	Policies map[string]mcpStorePolicy.PolicyDocument  `json:"policies,omitempty"`
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

	MCPServers      map[string]mcpStoreServer.CoreServer `json:"mcpServers"`
	BundleExtension BundleExtension                      `json:"bundleExtension"`
}
