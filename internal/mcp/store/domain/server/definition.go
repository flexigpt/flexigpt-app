package server

import (
	"fmt"
	"maps"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpserverv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
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

func ServerDocumentFromDefinition(
	input definition.Definition,
) (ServerDocument, error) {
	document, _, err := serverDocumentAndBodyFromDefinition(input)
	if err != nil {
		return ServerDocument{}, err
	}
	return document, nil
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
	if input.Kind != mcpserverv1.MCPServerKind ||
		input.SchemaID != mcpserverv1.MCPServerSchemaID ||
		input.SchemaVersion != mcpserverv1.MCPServerSchemaVersion {
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
				Kind:        mcppolicyv1.MCPPolicyKind,
				LogicalName: input.Extension.Policy.Ref,
			},
		)
	}

	return definition.Canonicalize(
		definition.Definition{
			Kind:           mcpserverv1.MCPServerKind,
			SchemaID:       mcpserverv1.MCPServerSchemaID,
			SchemaVersion:  mcpserverv1.MCPServerSchemaVersion,
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
	_, body, err := serverDocumentAndBodyFromDefinition(input)
	if err != nil {
		return ServerDefinitionBody{}, err
	}
	return body, nil
}

func serverDocumentAndBodyFromDefinition(
	input definition.Definition,
) (ServerDocument, ServerDefinitionBody, error) {
	value := input
	if value.Kind != mcpserverv1.MCPServerKind ||
		value.SchemaID != mcpserverv1.MCPServerSchemaID ||
		value.SchemaVersion != mcpserverv1.MCPServerSchemaVersion {
		return ServerDocument{}, ServerDefinitionBody{}, fmt.Errorf(
			"%w: Definition is not an MCP Server",
			basespec.ErrInvalid,
		)
	}

	body, err := definition.DecodeBody[ServerDefinitionBody](value.Body)
	if err != nil {
		return ServerDocument{}, ServerDefinitionBody{}, err
	}

	document := ServerDocument{
		Kind:           mcpserverv1.MCPServerKind,
		SchemaID:       mcpserverv1.MCPServerSchemaID,
		SchemaVersion:  mcpserverv1.MCPServerSchemaVersion,
		LogicalName:    value.LogicalName,
		LogicalVersion: value.LogicalVersion,
		DisplayName:    value.DisplayName,
		Description:    value.Description,
		Labels:         maps.Clone(value.Labels),
		MCPServer:      body.MCPServer,
		Extension:      body.Extension,
	}
	if err := document.Validate(); err != nil {
		return ServerDocument{}, ServerDefinitionBody{}, err
	}
	return document, body, nil
}
