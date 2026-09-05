package server

import (
	"fmt"
	"maps"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

const (
	TransportLabelKey = "mcp.transport"
	AuthModeLabelKey  = "mcp.auth-mode"
)

func ServerSubresource(
	name basespec.LogicalName,
) basespec.SubresourceLocator {
	return basespec.SubresourceLocator(
		path.Join(string(artifactbuiltin.MCPServerSubresourceDirectory), string(name)),
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
) (providerapi.Definition, error) {
	if input.Kind != artifactbuiltin.ServerKind ||
		input.SchemaID != artifactbuiltin.ServerSchemaID ||
		input.SchemaVersion != artifactbuiltin.MCPSchemaVersion {
		return providerapi.Definition{}, fmt.Errorf(
			"%w: canonical MCP server input has another schema identity",
			basespec.ErrInvalid,
		)
	}

	body, err := providerapi.EncodeBody(
		ServerDefinitionBody{
			MCPServer: input.MCPServer,
			Extension: input.Extension,
		},
	)
	if err != nil {
		return providerapi.Definition{}, err
	}

	labels := maps.Clone(input.Labels)
	if labels == nil {
		labels = map[string]string{}
	}
	labels[TransportLabelKey] = string(input.MCPServer.Type)
	labels[AuthModeLabelKey] = string(input.Extension.Auth.Mode)

	dependencies := []providerapi.Selector(nil)
	if input.Extension.Policy != nil {
		dependencies = append(
			dependencies,
			providerapi.Selector{
				Kind:        artifactbuiltin.PolicyKind,
				LogicalName: input.Extension.Policy.Ref,
			},
		)
	}

	return providerapi.Canonicalize(
		providerapi.Definition{
			Kind:           artifactbuiltin.ServerKind,
			SchemaID:       artifactbuiltin.ServerSchemaID,
			SchemaVersion:  artifactbuiltin.MCPSchemaVersion,
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
	input providerapi.Definition,
) (ServerDefinitionBody, error) {
	value, err := providerapi.Canonicalize(input)
	if err != nil {
		return ServerDefinitionBody{}, err
	}
	if value.Kind != artifactbuiltin.ServerKind ||
		value.SchemaID != artifactbuiltin.ServerSchemaID ||
		value.SchemaVersion != artifactbuiltin.MCPSchemaVersion {
		return ServerDefinitionBody{}, fmt.Errorf(
			"%w: Definition is not an MCP Server",
			basespec.ErrInvalid,
		)
	}

	body, err := providerapi.DecodeBody[ServerDefinitionBody](value.Body)
	if err != nil {
		return ServerDefinitionBody{}, err
	}

	document := ServerDocument{
		Kind:           artifactbuiltin.ServerKind,
		SchemaID:       artifactbuiltin.ServerSchemaID,
		SchemaVersion:  artifactbuiltin.MCPSchemaVersion,
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
