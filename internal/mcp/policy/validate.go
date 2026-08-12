package policy

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

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
