package server

import (
	"fmt"
	"maps"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/builtin/schema"
)

const (
	TransportLabelKey = "mcp.transport"
	AuthModeLabelKey  = "mcp.auth-mode"
)

func ServerSubresource(
	name basespec.LogicalName,
) basespec.SubresourceLocator {
	return basespec.SubresourceLocator(
		path.Join("mcpServers", string(name)),
	)
}

// DefinitionForCanonicalServer converts an MCP server projected from an
// Artifact Store-canonicalized MCP Bundle into an immutable Definition.
//
// Portable document validation belongs to the Artifact Store shareable schema
// registry. This function intentionally performs only MCP Definition
// projection and generic Definition canonicalization.
func DefinitionForCanonicalServer(
	input ServerDocument,
) (definition.Definition, error) {
	if input.Kind != schema.ServerKind ||
		input.SchemaID != schema.ServerSchemaID ||
		input.SchemaVersion != schema.MCPSchemaVersion {
		return definition.Definition{}, fmt.Errorf(
			"%w: canonical MCP server input has another schema identity",
			basespec.ErrInvalid,
		)
	}

	body, err := definition.EncodeBody(
		ServerDefinitionBody{
			MCPServer: input.MCPServer,
			Extension: input.Extension,
		},
	)
	if err != nil {
		return definition.Definition{}, err
	}

	labels := maps.Clone(input.Labels)
	if labels == nil {
		labels = map[string]string{}
	}
	labels[TransportLabelKey] = string(input.MCPServer.Type)
	labels[AuthModeLabelKey] = string(input.Extension.Auth.Mode)

	dependencies := []definition.Selector(nil)
	if input.Extension.Policy != nil {
		dependencies = append(
			dependencies,
			definition.Selector{
				Kind:        schema.PolicyKind,
				LogicalName: input.Extension.Policy.Ref,
			},
		)
	}

	return definition.Canonicalize(
		definition.Definition{
			Kind:           schema.ServerKind,
			SchemaID:       schema.ServerSchemaID,
			SchemaVersion:  schema.MCPSchemaVersion,
			LogicalName:    input.LogicalName,
			LogicalVersion: input.LogicalVersion,
			DisplayName:    input.DisplayName,
			Description:    input.Description,
			Labels:         labels,
			Body:           body,
			Dependencies:   dependencies,
		},
	)
}

func ServerBodyFromDefinition(
	input definition.Definition,
) (ServerDefinitionBody, error) {
	value, err := definition.Canonicalize(input)
	if err != nil {
		return ServerDefinitionBody{}, err
	}
	if value.Kind != schema.ServerKind ||
		value.SchemaID != schema.ServerSchemaID ||
		value.SchemaVersion != schema.MCPSchemaVersion {
		return ServerDefinitionBody{}, fmt.Errorf(
			"%w: Definition is not an MCP Server",
			basespec.ErrInvalid,
		)
	}

	body, err := definition.DecodeBody[ServerDefinitionBody](value.Body)
	if err != nil {
		return ServerDefinitionBody{}, err
	}

	document := ServerDocument{
		Kind:           schema.ServerKind,
		SchemaID:       schema.ServerSchemaID,
		SchemaVersion:  schema.MCPSchemaVersion,
		LogicalName:    value.LogicalName,
		LogicalVersion: value.LogicalVersion,
		DisplayName:    value.DisplayName,
		Description:    value.Description,
		Labels:         maps.Clone(value.Labels),
		MCPServer:      body.MCPServer,
		Extension:      body.Extension,
	}
	if err := ValidateServer(document); err != nil {
		return ServerDefinitionBody{}, err
	}
	return body, nil
}
