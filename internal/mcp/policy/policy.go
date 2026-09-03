package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"
)

var ErrInvalid = errors.New("invalid MCP policy")

type MCPApprovalRule string

const (
	MCPApprovalRuleAsk   MCPApprovalRule = "ask"
	MCPApprovalRuleAllow MCPApprovalRule = "allow"
	MCPApprovalRuleDeny  MCPApprovalRule = "deny"
)

type MCPExecutionMode string

const (
	MCPExecutionModeManual MCPExecutionMode = "manual"
	MCPExecutionModeAuto   MCPExecutionMode = "auto"
)

type MCPTrustLevel string

const (
	MCPTrustLevelUntrusted MCPTrustLevel = "untrusted"
	MCPTrustLevelTrusted   MCPTrustLevel = "trusted"
)

type MCPServerPolicy struct {
	DefaultApprovalRule  MCPApprovalRule  `json:"defaultApprovalRule"`
	DefaultExecutionMode MCPExecutionMode `json:"defaultExecutionMode"`

	RequireApprovalForUnknownRisk bool `json:"requireApprovalForUnknownRisk"`
	RequireApprovalForWrite       bool `json:"requireApprovalForWrite"`
	RequireApprovalForDestructive bool `json:"requireApprovalForDestructive"`
}

type MCPToolPolicyOverride struct {
	ToolName string `json:"toolName"`

	ApprovalRule  *MCPApprovalRule  `json:"approvalRule,omitempty"`
	ExecutionMode *MCPExecutionMode `json:"executionMode,omitempty"`

	AllowStaleDigest bool   `json:"allowStaleDigest,omitempty"`
	ExpectedDigest   string `json:"expectedDigest,omitempty"`
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
	return NormalizeMCPPolicy(MCPPolicy{
		TrustLevel:    MCPTrustLevelUntrusted,
		DefaultPolicy: DefaultMCPServerPolicy(),
		AppsPolicy:    DefaultMCPAppsPolicy(),
	})
}

func CloneMCPPolicy(input MCPPolicy) MCPPolicy {
	output := input
	output.ToolPolicies = CloneToolPolicies(input.ToolPolicies)
	if output.ToolPolicies == nil {
		output.ToolPolicies = map[string]MCPToolPolicyOverride{}
	}
	return output
}

// CloneToolPolicies performs a deep clone of override pointer fields. A
// shallow map clone is unsafe because ApprovalRule and ExecutionMode are
// mutable pointers.
func CloneToolPolicies(
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

func NormalizeMCPPolicy(input MCPPolicy) MCPPolicy {
	output := CloneMCPPolicy(input)

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

func ValidateMCPPolicy(value MCPPolicy) error {
	switch value.TrustLevel {
	case MCPTrustLevelTrusted, MCPTrustLevelUntrusted:
	default:
		return fmt.Errorf(
			"%w: invalid MCP trust level %q",
			ErrInvalid,
			value.TrustLevel,
		)
	}

	if err := ValidateMCPServerPolicy(value.DefaultPolicy); err != nil {
		return err
	}

	for name, override := range value.ToolPolicies {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("%w: empty MCP tool policy name", ErrInvalid)
		}
		if err := ValidateMCPToolPolicyOverride(name, override); err != nil {
			return err
		}
	}

	return nil
}

func ValidateMCPServerPolicy(value MCPServerPolicy) error {
	if err := ValidateMCPApprovalRule(value.DefaultApprovalRule); err != nil {
		return err
	}
	return ValidateMCPExecutionMode(value.DefaultExecutionMode)
}

func ValidateMCPToolPolicyOverride(
	name string,
	value MCPToolPolicyOverride,
) error {
	if value.ToolName != "" && value.ToolName != name {
		return fmt.Errorf(
			"%w: MCP tool policy key differs from toolName",
			ErrInvalid,
		)
	}
	if value.ApprovalRule != nil {
		if err := ValidateMCPApprovalRule(*value.ApprovalRule); err != nil {
			return err
		}
	}
	if value.ExecutionMode != nil {
		if err := ValidateMCPExecutionMode(*value.ExecutionMode); err != nil {
			return err
		}
	}
	if value.ExpectedDigest != "" {
		if err := ValidateDigest(value.ExpectedDigest); err != nil {
			return err
		}
	}
	return nil
}

func ValidateMCPApprovalRule(value MCPApprovalRule) error {
	switch value {
	case MCPApprovalRuleAllow, MCPApprovalRuleAsk, MCPApprovalRuleDeny:
		return nil
	default:
		return fmt.Errorf(
			"%w: invalid MCP approval rule %q",
			ErrInvalid,
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
			ErrInvalid,
			value,
		)
	}
}

func ValidateDigest(value string) error {
	raw, found := strings.CutPrefix(value, "sha256:")
	if !found || len(raw) != sha256.Size*2 {
		return fmt.Errorf("%w: invalid MCP digest", ErrInvalid)
	}
	if _, err := hex.DecodeString(raw); err != nil {
		return fmt.Errorf("%w: invalid MCP digest: %w", ErrInvalid, err)
	}
	return nil
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

func ComposeMCPPolicy(
	baseline MCPPolicy,
	policies ...MCPPolicy,
) (Composition, error) {
	result := NormalizeMCPPolicy(baseline)
	if err := ValidateMCPPolicy(result); err != nil {
		return Composition{}, err
	}

	normalized := make([]MCPPolicy, 0, len(policies)+1)
	normalized = append(normalized, result)

	conflicts := map[string]string{}
	for index, candidate := range policies {
		candidate = NormalizeMCPPolicy(candidate)
		if err := ValidateMCPPolicy(candidate); err != nil {
			return Composition{}, fmt.Errorf("policy %d: %w", index, err)
		}

		normalized = append(normalized, candidate)
		result.TrustLevel = restrictiveTrust(
			result.TrustLevel,
			candidate.TrustLevel,
		)
		result.DefaultPolicy.DefaultApprovalRule = restrictiveApproval(
			result.DefaultPolicy.DefaultApprovalRule,
			candidate.DefaultPolicy.DefaultApprovalRule,
		)
		result.DefaultPolicy.DefaultExecutionMode = restrictiveExecution(
			result.DefaultPolicy.DefaultExecutionMode,
			candidate.DefaultPolicy.DefaultExecutionMode,
		)
		result.DefaultPolicy.RequireApprovalForUnknownRisk =
			result.DefaultPolicy.RequireApprovalForUnknownRisk ||
				candidate.DefaultPolicy.RequireApprovalForUnknownRisk
		result.DefaultPolicy.RequireApprovalForWrite =
			result.DefaultPolicy.RequireApprovalForWrite ||
				candidate.DefaultPolicy.RequireApprovalForWrite
		result.DefaultPolicy.RequireApprovalForDestructive =
			result.DefaultPolicy.RequireApprovalForDestructive ||
				candidate.DefaultPolicy.RequireApprovalForDestructive

		result.AppsPolicy.Enabled =
			result.AppsPolicy.Enabled && candidate.AppsPolicy.Enabled
		result.AppsPolicy.AllowAppInitiatedToolCalls =
			result.AppsPolicy.AllowAppInitiatedToolCalls &&
				candidate.AppsPolicy.AllowAppInitiatedToolCalls
		result.AppsPolicy.RequireApprovalForOpenLink =
			result.AppsPolicy.RequireApprovalForOpenLink ||
				candidate.AppsPolicy.RequireApprovalForOpenLink
		result.AppsPolicy.RequireApprovalForContextUpdates =
			result.AppsPolicy.RequireApprovalForContextUpdates ||
				candidate.AppsPolicy.RequireApprovalForContextUpdates
	}

	names := map[string]struct{}{}
	for _, candidate := range normalized {
		for name := range candidate.ToolPolicies {
			names[name] = struct{}{}
		}
	}

	orderedNames := make([]string, 0, len(names))
	for name := range names {
		orderedNames = append(orderedNames, name)
	}
	sort.Strings(orderedNames)

	result.ToolPolicies = make(
		map[string]MCPToolPolicyOverride,
		len(orderedNames),
	)
	for _, name := range orderedNames {
		override, conflict := composeToolPolicyOverride(name, normalized)
		result.ToolPolicies[name] = override
		if conflict != "" {
			conflicts[name] = conflict
		}
	}

	return Composition{
		Body:      result,
		Conflicts: maps.Clone(conflicts),
	}, nil
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
