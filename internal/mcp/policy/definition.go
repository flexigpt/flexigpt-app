package policy

import (
	"fmt"
	"maps"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/builtin/schema"
)

func PolicySubresource(
	name basespec.LogicalName,
) basespec.SubresourceLocator {
	return basespec.SubresourceLocator(
		path.Join("policies", string(name)),
	)
}

func PolicyBodyFromDefinition(
	input definition.Definition,
) (MCPPolicy, error) {
	value, err := definition.Canonicalize(input)
	if err != nil {
		return MCPPolicy{}, err
	}
	if value.Kind != schema.PolicyKind ||
		value.SchemaID != schema.PolicySchemaID ||
		value.SchemaVersion != schema.MCPSchemaVersion {
		return MCPPolicy{}, fmt.Errorf(
			"%w: Definition is not an MCP Policy",
			basespec.ErrInvalid,
		)
	}
	body, err := definition.DecodeBody[MCPPolicy](
		value.Body,
	)
	if err != nil {
		return MCPPolicy{}, err
	}
	body = NormalizePolicyBody(body)
	if err := ValidatePolicyBody(body); err != nil {
		return MCPPolicy{}, err
	}
	return body, nil
}

// DefinitionForCanonicalPolicy converts an MCP policy projected from an
// Artifact Store-canonicalized MCP Bundle into an immutable Definition.
func DefinitionForCanonicalPolicy(
	input PolicyDocument,
) (definition.Definition, error) {
	if input.Kind != schema.PolicyKind ||
		input.SchemaID != schema.PolicySchemaID ||
		input.SchemaVersion != schema.MCPSchemaVersion {
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
			SchemaVersion:  schema.MCPSchemaVersion,
			LogicalName:    input.LogicalName,
			LogicalVersion: input.LogicalVersion,
			DisplayName:    input.DisplayName,
			Description:    input.Description,
			Labels:         maps.Clone(input.Labels),
			Body:           body,
		},
	)
}
