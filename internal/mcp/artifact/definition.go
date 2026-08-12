package artifact

import (
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
)

const (
	TransportLabelKey = "mcp.transport"
	AuthModeLabelKey  = "mcp.auth-mode"
)

// DefinitionForCanonicalServer converts an MCP server projected from an
// Artifact Store-canonicalized MCP Bundle into an immutable Definition.
//
// Portable document validation belongs to the Artifact Store shareable schema
// registry. This function intentionally performs only MCP Definition
// projection and generic Definition canonicalization.
func DefinitionForCanonicalServer(
	input schema.ServerDocument,
) (definition.Definition, error) {
	if input.Kind != schema.ServerKind ||
		input.SchemaID != schema.ServerSchemaID ||
		input.SchemaVersion != schema.SchemaVersion {
		return definition.Definition{}, fmt.Errorf(
			"%w: canonical MCP server input has another schema identity",
			basespec.ErrInvalid,
		)
	}

	body, err := definition.EncodeBody(
		schema.ServerDefinitionBody{
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
			SchemaVersion:  schema.SchemaVersion,
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

// DefinitionForCanonicalPolicy converts an MCP policy projected from an
// Artifact Store-canonicalized MCP Bundle into an immutable Definition.
func DefinitionForCanonicalPolicy(
	input schema.PolicyDocument,
) (definition.Definition, error) {
	if input.Kind != schema.PolicyKind ||
		input.SchemaID != schema.PolicySchemaID ||
		input.SchemaVersion != schema.SchemaVersion {
		return definition.Definition{}, fmt.Errorf(
			"%w: canonical MCP policy input has another schema identity",
			basespec.ErrInvalid,
		)
	}
	body, err := definition.EncodeBody(input.Body)
	if err != nil {
		return definition.Definition{}, err
	}
	return definition.Canonicalize(
		definition.Definition{
			Kind:           schema.PolicyKind,
			SchemaID:       schema.PolicySchemaID,
			SchemaVersion:  schema.SchemaVersion,
			LogicalName:    input.LogicalName,
			LogicalVersion: input.LogicalVersion,
			DisplayName:    input.DisplayName,
			Description:    input.Description,
			Labels:         maps.Clone(input.Labels),
			Body:           body,
		},
	)
}

func ServerBodyFromDefinition(
	input definition.Definition,
) (schema.ServerDefinitionBody, error) {
	value, err := definition.Canonicalize(input)
	if err != nil {
		return schema.ServerDefinitionBody{}, err
	}
	if value.Kind != schema.ServerKind ||
		value.SchemaID != schema.ServerSchemaID ||
		value.SchemaVersion != schema.SchemaVersion {
		return schema.ServerDefinitionBody{}, fmt.Errorf(
			"%w: Definition is not an MCP Server",
			basespec.ErrInvalid,
		)
	}

	body, err := definition.DecodeBody[schema.ServerDefinitionBody](value.Body)
	if err != nil {
		return schema.ServerDefinitionBody{}, err
	}

	document := schema.ServerDocument{
		Kind:           schema.ServerKind,
		SchemaID:       schema.ServerSchemaID,
		SchemaVersion:  schema.SchemaVersion,
		LogicalName:    value.LogicalName,
		LogicalVersion: value.LogicalVersion,
		DisplayName:    value.DisplayName,
		Description:    value.Description,
		Labels:         maps.Clone(value.Labels),
		MCPServer:      body.MCPServer,
		Extension:      body.Extension,
	}
	if err := schema.ValidateServer(document); err != nil {
		return schema.ServerDefinitionBody{}, err
	}
	return body, nil
}

func PolicyBodyFromDefinition(
	input definition.Definition,
) (policy.MCPPolicy, error) {
	value, err := definition.Canonicalize(input)
	if err != nil {
		return policy.MCPPolicy{}, err
	}
	if value.Kind != schema.PolicyKind ||
		value.SchemaID != schema.PolicySchemaID ||
		value.SchemaVersion != schema.SchemaVersion {
		return policy.MCPPolicy{}, fmt.Errorf(
			"%w: Definition is not an MCP Policy",
			basespec.ErrInvalid,
		)
	}
	body, err := definition.DecodeBody[policy.MCPPolicy](
		value.Body,
	)
	if err != nil {
		return policy.MCPPolicy{}, err
	}
	body = policy.NormalizePolicyBody(body)
	if err := policy.ValidatePolicyBody(body); err != nil {
		return policy.MCPPolicy{}, err
	}
	return body, nil
}
