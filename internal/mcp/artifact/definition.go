package artifact

import (
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
)

const (
	TransportLabelKey = "mcp.transport"
	AuthModeLabelKey  = "mcp.auth-mode"
)

func DefinitionForServer(
	input schema.ServerDocument,
) (definition.Definition, error) {
	value, _, err := schema.CanonicalizeServer(input)
	if err != nil {
		return definition.Definition{}, err
	}

	body, err := definition.EncodeBody(
		schema.ServerDefinitionBody{
			MCPServer: value.MCPServer,
			Extension: value.Extension,
		},
	)
	if err != nil {
		return definition.Definition{}, err
	}

	labels := maps.Clone(value.Labels)
	if labels == nil {
		labels = map[string]string{}
	}
	labels[TransportLabelKey] = string(value.MCPServer.Type)
	labels[AuthModeLabelKey] = string(value.Extension.Auth.Mode)

	dependencies := []definition.Selector(nil)
	if value.Extension.Policy != nil {
		dependencies = append(
			dependencies,
			definition.Selector{
				Kind:        schema.PolicyKind,
				LogicalName: value.Extension.Policy.Ref,
			},
		)
	}

	return definition.Canonicalize(
		definition.Definition{
			Kind:           schema.ServerKind,
			SchemaID:       schema.ServerSchemaID,
			SchemaVersion:  schema.SchemaVersion,
			LogicalName:    value.LogicalName,
			LogicalVersion: value.LogicalVersion,
			DisplayName:    value.DisplayName,
			Description:    value.Description,
			Labels:         labels,
			Body:           body,
			Dependencies:   dependencies,
		},
	)
}

func DefinitionForPolicy(
	input schema.PolicyDocument,
) (definition.Definition, error) {
	value, _, err := schema.CanonicalizePolicy(input)
	if err != nil {
		return definition.Definition{}, err
	}
	body, err := definition.EncodeBody(value.Body)
	if err != nil {
		return definition.Definition{}, err
	}
	return definition.Canonicalize(
		definition.Definition{
			Kind:           schema.PolicyKind,
			SchemaID:       schema.PolicySchemaID,
			SchemaVersion:  schema.SchemaVersion,
			LogicalName:    value.LogicalName,
			LogicalVersion: value.LogicalVersion,
			DisplayName:    value.DisplayName,
			Description:    value.Description,
			Labels:         maps.Clone(value.Labels),
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
		SchemaURL:      schema.ServerSchemaURL,
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
) (schema.PolicyBody, error) {
	value, err := definition.Canonicalize(input)
	if err != nil {
		return schema.PolicyBody{}, err
	}
	if value.Kind != schema.PolicyKind ||
		value.SchemaID != schema.PolicySchemaID ||
		value.SchemaVersion != schema.SchemaVersion {
		return schema.PolicyBody{}, fmt.Errorf(
			"%w: Definition is not an MCP Policy",
			basespec.ErrInvalid,
		)
	}
	body, err := definition.DecodeBody[schema.PolicyBody](
		value.Body,
	)
	if err != nil {
		return schema.PolicyBody{}, err
	}
	body = schema.NormalizePolicyBody(body)
	if err := schema.ValidatePolicyBody(body); err != nil {
		return schema.PolicyBody{}, err
	}
	return body, nil
}
