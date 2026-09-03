package policy

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/policy"
)

// Published JSON Schema resources carry $schema and $id metadata. MCP document
// instances deliberately use kind, schemaID, and schemaVersion instead.
//
// This matches the existing Skill and Workspace document conventions and
// prevents schema-resource URLs from becoming semantic document content or
// affecting canonical Definition digests.

type PolicyDocument struct {
	Kind          basespec.ArtifactKind `json:"kind"`
	SchemaID      basespec.SchemaID     `json:"schemaID"`
	SchemaVersion string                `json:"schemaVersion"`
	Digest        cryptoutil.Digest     `json:"digest,omitempty"`

	LogicalName    basespec.LogicalName    `json:"logicalName"`
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string                  `json:"displayName,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Labels         map[string]string       `json:"labels,omitempty"`

	Body mcpPolicy.MCPPolicy `json:"body"`
}
