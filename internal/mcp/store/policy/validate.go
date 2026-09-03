package policy

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/policy"
)

func CanonicalizePolicy(
	input PolicyDocument,
) (PolicyDocument, json.RawMessage, error) {
	value, err := jsonutil.CloneJSON(input)
	if err != nil {
		return PolicyDocument{}, nil, err
	}
	value.Labels = maps.Clone(value.Labels)

	if err := ValidatePolicy(value); err != nil {
		return PolicyDocument{}, nil, err
	}

	supplied := value.Digest
	value.Digest = ""
	calculated, err := cryptoutil.CanonicalDigest(value)
	if err != nil {
		return PolicyDocument{}, nil, err
	}
	if supplied != "" && supplied != calculated {
		return PolicyDocument{}, nil, fmt.Errorf(
			"%w: supplied MCP Policy digest %q, calculated %q",
			basespec.ErrDigestMismatch,
			supplied,
			calculated,
		)
	}
	value.Digest = calculated

	raw, err := jsonutil.MarshalCanonicalObject(value, basespec.MaxDefinitionBytes)
	if err != nil {
		return PolicyDocument{}, nil, err
	}
	return value, raw, nil
}

func ValidatePolicy(value PolicyDocument) error {
	if value.Kind != artifactbuiltin.PolicyKind ||
		value.SchemaID != artifactbuiltin.PolicySchemaID ||
		value.SchemaVersion != artifactbuiltin.MCPSchemaVersion {
		return fmt.Errorf(
			"%w: unsupported MCP Policy schema",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidatePortableMetadata(
		value.LogicalName,
		value.LogicalVersion,
		value.DisplayName,
		value.Description,
		value.Labels,
	); err != nil {
		return err
	}
	return ValidatePolicyBody(value.Body)
}

func ValidatePolicyBody(body mcpPolicy.MCPPolicy) error {
	if err := mcpPolicy.ValidateMCPPolicy(body); err != nil {
		return fmt.Errorf("%w: invalid MCP policy body: %w", basespec.ErrInvalid, err)
	}
	for name, override := range body.ToolPolicies {
		if override.ToolName != name {
			return fmt.Errorf(
				"%w: MCP tool policy key and toolName differ",
				basespec.ErrInvalid,
			)
		}
	}
	return nil
}
