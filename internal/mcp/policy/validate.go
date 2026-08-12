package policy

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func CanonicalizePolicy(
	input PolicyDocument,
) (PolicyDocument, json.RawMessage, error) {
	value, err := jsonutil.CloneJSON(input)
	if err != nil {
		return PolicyDocument{}, nil, err
	}
	value.Labels = maps.Clone(value.Labels)
	value.Body = NormalizePolicyBody(value.Body)

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
	if value.Kind != schema.PolicyKind ||
		value.SchemaID != schema.PolicySchemaID ||
		value.SchemaVersion != schema.MCPSchemaVersion {
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

func ValidatePolicyBody(body MCPPolicy) error {
	switch body.TrustLevel {
	case MCPTrustLevelTrusted, MCPTrustLevelUntrusted:
	default:
		return fmt.Errorf(
			"%w: invalid MCP trust level %q",
			basespec.ErrInvalid,
			body.TrustLevel,
		)
	}

	switch body.DefaultPolicy.DefaultApprovalRule {
	case MCPApprovalRuleAllow,
		MCPApprovalRuleAsk,
		MCPApprovalRuleDeny:
	default:
		return fmt.Errorf(
			"%w: invalid MCP approval rule",
			basespec.ErrInvalid,
		)
	}
	switch body.DefaultPolicy.DefaultExecutionMode {
	case MCPExecutionModeAuto,
		MCPExecutionModeManual:
	default:
		return fmt.Errorf(
			"%w: invalid MCP execution mode",
			basespec.ErrInvalid,
		)
	}

	for name, override := range body.ToolPolicies {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf(
				"%w: empty MCP tool policy name",
				basespec.ErrInvalid,
			)
		}
		if override.ToolName != name {
			return fmt.Errorf(
				"%w: MCP tool policy key and toolName differ",
				basespec.ErrInvalid,
			)
		}
		if override.ApprovalRule != nil {
			switch *override.ApprovalRule {
			case MCPApprovalRuleAllow,
				MCPApprovalRuleAsk,
				MCPApprovalRuleDeny:
			default:
				return fmt.Errorf(
					"%w: invalid tool approval rule",
					basespec.ErrInvalid,
				)
			}
		}
		if override.ExecutionMode != nil {
			switch *override.ExecutionMode {
			case MCPExecutionModeAuto,
				MCPExecutionModeManual:
			default:
				return fmt.Errorf(
					"%w: invalid tool execution mode",
					basespec.ErrInvalid,
				)
			}
		}
		if override.ExpectedDigest != "" {
			if err := cryptoutil.ValidateDigest(
				cryptoutil.Digest(override.ExpectedDigest),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func ValidateMCPApprovalRule(value MCPApprovalRule) error {
	switch value {
	case MCPApprovalRuleAllow,
		MCPApprovalRuleAsk,
		MCPApprovalRuleDeny:
		return nil
	default:
		return fmt.Errorf(
			"%w: invalid MCP approval rule %q",
			basespec.ErrInvalid,
			value,
		)
	}
}

func ValidateMCPExecutionMode(value MCPExecutionMode) error {
	switch value {
	case MCPExecutionModeAuto, MCPExecutionModeManual:
		return nil
	default:
		return fmt.Errorf(
			"%w: invalid MCP execution mode %q",
			basespec.ErrInvalid,
			value,
		)
	}
}

func ApprovalRuleRank(value MCPApprovalRule) int {
	switch value {
	case MCPApprovalRuleDeny:
		return 3
	case MCPApprovalRuleAsk:
		return 2
	default:
		return 1
	}
}

func ExecutionModeRank(value MCPExecutionMode) int {
	if value == MCPExecutionModeManual {
		return 2
	}
	return 1
}

func NormalizedApprovalRule(
	value MCPApprovalRule,
) MCPApprovalRule {
	if value == "" {
		return MCPApprovalRuleAsk
	}
	return value
}

func NormalizedExecutionMode(
	value MCPExecutionMode,
) MCPExecutionMode {
	if value == "" {
		return MCPExecutionModeManual
	}
	return value
}
