package policy

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
)

func BodyFromDefinition(
	input definition.Definition,
) (mcpPolicy.MCPPolicy, error) {
	if err := input.Validate(); err != nil {
		return mcpPolicy.MCPPolicy{}, err
	}
	if input.Kind != mcpDomain.MCPPolicyArtifactKind ||
		input.SchemaID != mcppolicyv1.MCPPolicySchemaKey.SchemaID ||
		input.SchemaVersion != mcppolicyv1.MCPPolicySchemaKey.SchemaVersion {
		return mcpPolicy.MCPPolicy{}, fmt.Errorf(
			"%w: Definition is not an MCP Policy",
			basespec.ErrInvalid,
		)
	}

	document, err := mcppolicyv1.DecodeMCPPolicyJSON(input.Body)
	if err != nil {
		return mcpPolicy.MCPPolicy{}, err
	}
	if document.Body == nil {
		return mcpPolicy.MCPPolicy{}, fmt.Errorf(
			"%w: MCP Policy declaration has no resolved body",
			basespec.ErrReferenceUnresolved,
		)
	}

	raw, err := jsonutil.MarshalCanonicalObject(
		*document.Body,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return mcpPolicy.MCPPolicy{}, err
	}
	var body mcpPolicy.MCPPolicy
	if err := json.Unmarshal(raw, &body); err != nil {
		return mcpPolicy.MCPPolicy{}, fmt.Errorf(
			"%w: decode MCP Policy body: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	body = mcpPolicy.Normalize(body)
	if err := body.Validate(); err != nil {
		return mcpPolicy.MCPPolicy{}, err
	}
	return body, nil
}

func DefinitionForDocument(
	input mcppolicyv1.MCPPolicyDocument,
) (definition.Definition, error) {
	if err := input.Validate(); err != nil {
		return definition.Definition{}, err
	}
	body, err := input.CanonicalJSON()
	if err != nil {
		return definition.Definition{}, err
	}
	value := definition.Definition{
		Kind:          mcpDomain.MCPPolicyArtifactKind,
		SchemaID:      mcppolicyv1.MCPPolicySchemaKey.SchemaID,
		SchemaVersion: mcppolicyv1.MCPPolicySchemaKey.SchemaVersion,
		LogicalName:   basespec.LogicalName(input.Name),
		DisplayName:   input.Name,
		Description:   input.Description,
		Body:          body,
		Dependencies:  nil,
	}
	return definition.Canonicalize(value)
}

func DocumentFromSourceBody(
	name basespec.LogicalName,
	description string,
	body mcppolicyv1.MCPPolicyBody,
) (mcppolicyv1.MCPPolicyDocument, error) {
	value := mcppolicyv1.MCPPolicyDocument{
		APIVersion:  mcppolicyv1.MCPPolicySchemaVersion,
		Type:        mcppolicyv1.MCPPolicyType,
		Name:        string(name),
		Description: description,
		Body:        &body,
	}
	if err := value.Validate(); err != nil {
		return mcppolicyv1.MCPPolicyDocument{}, err
	}
	return value, nil
}
