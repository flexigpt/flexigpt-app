package policy

import (
	"fmt"
	"maps"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
)

func PolicySubresource(
	name basespec.LogicalName,
) basespec.SubresourceLocator {
	return basespec.SubresourceLocator(
		path.Join("policies", string(name)),
	)
}

func BodyFromDefinition(
	input definition.Definition,
) (mcpPolicy.MCPPolicy, error) {
	value := input
	if value.Kind != mcppolicyv1.MCPPolicyKind ||
		value.SchemaID != mcppolicyv1.MCPPolicySchemaID ||
		value.SchemaVersion != mcppolicyv1.MCPPolicySchemaVersion {
		return mcpPolicy.MCPPolicy{}, fmt.Errorf(
			"%w: Definition is not an MCP Policy",
			basespec.ErrInvalid,
		)
	}
	body, err := definition.DecodeBody[mcpPolicy.MCPPolicy](
		value.Body,
	)
	if err != nil {
		return mcpPolicy.MCPPolicy{}, err
	}

	if err := body.Validate(); err != nil {
		return mcpPolicy.MCPPolicy{}, err
	}
	return body, nil
}

// DefinitionForCanonicalPolicy converts an MCP policy projected from an
// Artifact Store-canonicalized MCP Bundle into an immutable Definition.
func DefinitionForCanonicalPolicy(
	input PolicyDocument,
) (definition.Definition, error) {
	if input.Kind != mcppolicyv1.MCPPolicyKind ||
		input.SchemaID != mcppolicyv1.MCPPolicySchemaID ||
		input.SchemaVersion != mcppolicyv1.MCPPolicySchemaVersion {
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
			Kind:           mcppolicyv1.MCPPolicyKind,
			SchemaID:       mcppolicyv1.MCPPolicySchemaID,
			SchemaVersion:  mcppolicyv1.MCPPolicySchemaVersion,
			LogicalName:    input.LogicalName,
			LogicalVersion: input.LogicalVersion,
			DisplayName:    input.DisplayName,
			Description:    input.Description,
			Labels:         maps.Clone(input.Labels),
			Body:           body,
		},
	)
}
