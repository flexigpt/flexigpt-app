package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var ErrInvalid = errors.New("invalid MCP policy")

type MCPApprovalRule string

const (
	MCPApprovalRuleAsk   MCPApprovalRule = "ask"
	MCPApprovalRuleAllow MCPApprovalRule = "allow"
	MCPApprovalRuleDeny  MCPApprovalRule = "deny"
)

func (value MCPApprovalRule) Validate() error {
	switch value {
	case MCPApprovalRuleAllow, MCPApprovalRuleAsk, MCPApprovalRuleDeny:
		return nil
	default:
		return fmt.Errorf("%w: invalid MCP approval rule %q", ErrInvalid, value)
	}
}

type MCPExecutionMode string

const (
	MCPExecutionModeManual MCPExecutionMode = "manual"
	MCPExecutionModeAuto   MCPExecutionMode = "auto"
)

func (value MCPExecutionMode) Validate() error {
	switch value {
	case MCPExecutionModeAuto, MCPExecutionModeManual:
		return nil
	default:
		return fmt.Errorf("%w: invalid MCP execution mode %q", ErrInvalid, value)
	}
}

type MCPTrustLevel string

const (
	MCPTrustLevelUntrusted MCPTrustLevel = "untrusted"
	MCPTrustLevelTrusted   MCPTrustLevel = "trusted"
)

func (value MCPTrustLevel) Validate() error {
	switch value {
	case MCPTrustLevelTrusted, MCPTrustLevelUntrusted:
		return nil
	default:
		return fmt.Errorf("%w: invalid MCP trust level %q", ErrInvalid, value)
	}
}

type MCPServerPolicy struct {
	DefaultApprovalRule  MCPApprovalRule  `json:"defaultApprovalRule"`
	DefaultExecutionMode MCPExecutionMode `json:"defaultExecutionMode"`

	RequireApprovalForUnknownRisk bool `json:"requireApprovalForUnknownRisk"`
	RequireApprovalForWrite       bool `json:"requireApprovalForWrite"`
	RequireApprovalForDestructive bool `json:"requireApprovalForDestructive"`
}

func (value MCPServerPolicy) Validate() error {
	if err := value.DefaultApprovalRule.Validate(); err != nil {
		return err
	}
	return value.DefaultExecutionMode.Validate()
}

type MCPToolPolicyOverride struct {
	ToolName string `json:"toolName"`

	ApprovalRule  *MCPApprovalRule  `json:"approvalRule,omitempty"`
	ExecutionMode *MCPExecutionMode `json:"executionMode,omitempty"`

	AllowStaleDigest bool   `json:"allowStaleDigest,omitempty"`
	ExpectedDigest   string `json:"expectedDigest,omitempty"`
}

func (value MCPToolPolicyOverride) Validate() error {
	if value.ToolName != "" && strings.TrimSpace(value.ToolName) == "" {
		return fmt.Errorf("%w: invalid MCP tool policy name", ErrInvalid)
	}
	if value.ApprovalRule != nil {
		if err := value.ApprovalRule.Validate(); err != nil {
			return err
		}
	}
	if value.ExecutionMode != nil {
		if err := value.ExecutionMode.Validate(); err != nil {
			return err
		}
	}
	if value.ExpectedDigest != "" {
		if err := validateDigest(value.ExpectedDigest); err != nil {
			return err
		}
	}
	return nil
}

type MCPAppsPolicy struct {
	Enabled                          bool `json:"enabled"`
	AllowAppInitiatedToolCalls       bool `json:"allowAppInitiatedToolCalls"`
	RequireApprovalForOpenLink       bool `json:"requireApprovalForOpenLink"`
	RequireApprovalForContextUpdates bool `json:"requireApprovalForContextUpdates"`
}

type MCPPolicy struct {
	TrustLevel    MCPTrustLevel                    `json:"trustLevel"`
	DefaultPolicy MCPServerPolicy                  `json:"defaultPolicy"`
	ToolPolicies  map[string]MCPToolPolicyOverride `json:"toolPolicies,omitempty"`
	AppsPolicy    MCPAppsPolicy                    `json:"appsPolicy"`
}

func (value MCPPolicy) Validate() error {
	if err := value.TrustLevel.Validate(); err != nil {
		return err
	}
	if err := value.DefaultPolicy.Validate(); err != nil {
		return err
	}

	for name, override := range value.ToolPolicies {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("%w: empty MCP tool policy name", ErrInvalid)
		}
		if override.ToolName != "" && override.ToolName != name {
			return fmt.Errorf(
				"%w: MCP tool policy key %q differs from toolName %q",
				ErrInvalid,
				name,
				override.ToolName,
			)
		}
		if err := override.Validate(); err != nil {
			return fmt.Errorf("MCP tool policy %q: %w", name, err)
		}
	}
	return nil
}

type Composition struct {
	Body      MCPPolicy
	Conflicts map[string]string
}

func DefaultMCPServerPolicy() MCPServerPolicy {
	return MCPServerPolicy{
		DefaultApprovalRule:           MCPApprovalRuleAsk,
		DefaultExecutionMode:          MCPExecutionModeManual,
		RequireApprovalForUnknownRisk: true,
		RequireApprovalForWrite:       true,
		RequireApprovalForDestructive: true,
	}
}

func DefaultMCPAppsPolicy() MCPAppsPolicy {
	return MCPAppsPolicy{
		Enabled:                          false,
		AllowAppInitiatedToolCalls:       false,
		RequireApprovalForOpenLink:       true,
		RequireApprovalForContextUpdates: true,
	}
}

func Baseline() MCPPolicy {
	return Normalize(MCPPolicy{
		TrustLevel:    MCPTrustLevelUntrusted,
		DefaultPolicy: DefaultMCPServerPolicy(),
		AppsPolicy:    DefaultMCPAppsPolicy(),
	})
}

func Clone(input MCPPolicy) MCPPolicy {
	output := input
	output.ToolPolicies = cloneToolPolicies(input.ToolPolicies)
	if output.ToolPolicies == nil {
		output.ToolPolicies = map[string]MCPToolPolicyOverride{}
	}
	return output
}

func Normalize(input MCPPolicy) MCPPolicy {
	output := Clone(input)

	if output.TrustLevel == "" {
		output.TrustLevel = MCPTrustLevelUntrusted
	}
	if output.DefaultPolicy == (MCPServerPolicy{}) {
		output.DefaultPolicy = DefaultMCPServerPolicy()
	}
	if output.ToolPolicies == nil {
		output.ToolPolicies = map[string]MCPToolPolicyOverride{}
	}
	if output.AppsPolicy == (MCPAppsPolicy{}) {
		output.AppsPolicy = DefaultMCPAppsPolicy()
	}

	for name, override := range output.ToolPolicies {
		if override.ToolName == "" {
			override.ToolName = name
		}
		output.ToolPolicies[name] = cloneToolPolicyOverride(override)
	}

	return output
}

func ComposeMCPPolicy(
	baseline MCPPolicy,
	policies ...MCPPolicy,
) (Composition, error) {
	return composePolicies(baseline, policies...)
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

func NormalizedApprovalRule(value MCPApprovalRule) MCPApprovalRule {
	if value == "" {
		return MCPApprovalRuleAsk
	}
	return value
}

func NormalizedExecutionMode(value MCPExecutionMode) MCPExecutionMode {
	if value == "" {
		return MCPExecutionModeManual
	}
	return value
}

func cloneToolPolicies(
	input map[string]MCPToolPolicyOverride,
) map[string]MCPToolPolicyOverride {
	if input == nil {
		return nil
	}

	output := make(map[string]MCPToolPolicyOverride, len(input))
	for name, override := range input {
		output[name] = cloneToolPolicyOverride(override)
	}
	return output
}

func cloneToolPolicyOverride(
	input MCPToolPolicyOverride,
) MCPToolPolicyOverride {
	output := input
	if input.ApprovalRule != nil {
		value := *input.ApprovalRule
		output.ApprovalRule = &value
	}
	if input.ExecutionMode != nil {
		value := *input.ExecutionMode
		output.ExecutionMode = &value
	}
	return output
}

func composeToolPolicyOverride(
	name string,
	policies []MCPPolicy,
) (override MCPToolPolicyOverride, conflict string) {
	output := MCPToolPolicyOverride{ToolName: name}

	var (
		approvalRule  MCPApprovalRule
		executionMode MCPExecutionMode
		expected      string

		first       = true
		sawOverride bool
		allowStale  = true
	)

	for _, candidate := range policies {
		candidateApproval := candidate.DefaultPolicy.DefaultApprovalRule
		candidateExecution := candidate.DefaultPolicy.DefaultExecutionMode

		override, found := candidate.ToolPolicies[name]
		if found {
			sawOverride = true
			if override.ApprovalRule != nil {
				candidateApproval = *override.ApprovalRule
			}
			if override.ExecutionMode != nil {
				candidateExecution = *override.ExecutionMode
			}

			allowStale = allowStale && override.AllowStaleDigest
			switch {
			case override.ExpectedDigest == "":
			case expected == "":
				expected = override.ExpectedDigest
			case expected != override.ExpectedDigest:
				conflict = "conflicting expected tool digests"
			}
		}

		if first {
			approvalRule = candidateApproval
			executionMode = candidateExecution
			first = false
			continue
		}

		approvalRule = restrictiveApproval(approvalRule, candidateApproval)
		executionMode = restrictiveExecution(
			executionMode,
			candidateExecution,
		)
	}

	output.ApprovalRule = &approvalRule
	output.ExecutionMode = &executionMode
	output.ExpectedDigest = expected
	if sawOverride {
		output.AllowStaleDigest = allowStale
	}

	if conflict != "" {
		deny := MCPApprovalRuleDeny
		output.ApprovalRule = &deny
		output.AllowStaleDigest = false
	}

	return output, conflict
}

func restrictiveTrust(
	left MCPTrustLevel,
	right MCPTrustLevel,
) MCPTrustLevel {
	if left == MCPTrustLevelUntrusted ||
		right == MCPTrustLevelUntrusted {
		return MCPTrustLevelUntrusted
	}
	return MCPTrustLevelTrusted
}

func restrictiveApproval(
	left MCPApprovalRule,
	right MCPApprovalRule,
) MCPApprovalRule {
	if ApprovalRuleRank(left) >= ApprovalRuleRank(right) {
		return left
	}
	return right
}

func restrictiveExecution(
	left MCPExecutionMode,
	right MCPExecutionMode,
) MCPExecutionMode {
	if left == MCPExecutionModeManual ||
		right == MCPExecutionModeManual {
		return MCPExecutionModeManual
	}
	return MCPExecutionModeAuto
}

func validateDigest(value string) error {
	raw, found := strings.CutPrefix(value, "sha256:")
	if !found || len(raw) != sha256.Size*2 {
		return fmt.Errorf("%w: invalid MCP digest", ErrInvalid)
	}
	if _, err := hex.DecodeString(raw); err != nil {
		return fmt.Errorf("%w: invalid MCP digest: %w", ErrInvalid, err)
	}
	return nil
}
