package policy

import (
	"fmt"
	"maps"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
)

func PolicySubresource(
	name basespec.LogicalName,
) basespec.SubresourceLocator {
	return basespec.SubresourceLocator(
		path.Join(string(artifactbuiltin.MCPPolicySubresourceDirectory), string(name)),
	)
}

func PolicyBodyFromDefinition(
	input definition.Definition,
) (mcpPolicy.MCPPolicy, error) {
	value, err := definition.Canonicalize(input)
	if err != nil {
		return mcpPolicy.MCPPolicy{}, err
	}
	if value.Kind != artifactbuiltin.PolicyKind ||
		value.SchemaID != artifactbuiltin.PolicySchemaID ||
		value.SchemaVersion != artifactbuiltin.MCPSchemaVersion {
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

	if err := ValidatePolicyBody(body); err != nil {
		return mcpPolicy.MCPPolicy{}, err
	}
	return body, nil
}

// DefinitionForCanonicalPolicy converts an MCP policy projected from an
// Artifact Store-canonicalized MCP Bundle into an immutable Definition.
func DefinitionForCanonicalPolicy(
	input PolicyDocument,
) (definition.Definition, error) {
	if input.Kind != artifactbuiltin.PolicyKind ||
		input.SchemaID != artifactbuiltin.PolicySchemaID ||
		input.SchemaVersion != artifactbuiltin.MCPSchemaVersion {
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
			Kind:           artifactbuiltin.PolicyKind,
			SchemaID:       artifactbuiltin.PolicySchemaID,
			SchemaVersion:  artifactbuiltin.MCPSchemaVersion,
			LogicalName:    input.LogicalName,
			LogicalVersion: input.LogicalVersion,
			DisplayName:    input.DisplayName,
			Description:    input.Description,
			Labels:         maps.Clone(input.Labels),
			Body:           body,
		},
	)
}
