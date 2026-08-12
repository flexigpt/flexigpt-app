package policy

import (
	"encoding/json"
	"fmt"
	"maps"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Effective struct {
	Body      MCPPolicy         `json:"body"`
	Conflicts map[string]string `json:"conflicts,omitempty"`
	Digest    cryptoutil.Digest `json:"digest"`
}

func Baseline() MCPPolicy {
	return NormalizePolicyBody(MCPPolicy{
		TrustLevel:    MCPTrustLevelUntrusted,
		DefaultPolicy: DefaultMCPServerPolicy(),
		AppsPolicy: MCPAppsPolicy{
			Enabled:                          false,
			AllowAppInitiatedToolCalls:       false,
			RequireApprovalForOpenLink:       true,
			RequireApprovalForContextUpdates: true,
		},
	})
}

func Compose(
	baseline MCPPolicy,
	policies ...MCPPolicy,
) (Effective, error) {
	result := NormalizePolicyBody(baseline)
	if err := ValidatePolicyBody(result); err != nil {
		return Effective{}, err
	}

	conflicts := map[string]string{}
	for index, candidate := range policies {
		candidate = NormalizePolicyBody(candidate)
		if err := ValidatePolicyBody(candidate); err != nil {
			return Effective{}, fmt.Errorf(
				"policy %d: %w",
				index,
				err,
			)
		}

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
		result.DefaultPolicy.RequireApprovalForUnknownRisk = result.DefaultPolicy.RequireApprovalForUnknownRisk ||
			candidate.DefaultPolicy.RequireApprovalForUnknownRisk
		result.DefaultPolicy.RequireApprovalForWrite = result.DefaultPolicy.RequireApprovalForWrite ||
			candidate.DefaultPolicy.RequireApprovalForWrite
		result.DefaultPolicy.RequireApprovalForDestructive = result.DefaultPolicy.RequireApprovalForDestructive ||
			candidate.DefaultPolicy.RequireApprovalForDestructive

		result.AppsPolicy.Enabled = result.AppsPolicy.Enabled &&
			candidate.AppsPolicy.Enabled
		result.AppsPolicy.AllowAppInitiatedToolCalls = result.AppsPolicy.AllowAppInitiatedToolCalls &&
			candidate.AppsPolicy.AllowAppInitiatedToolCalls
		result.AppsPolicy.RequireApprovalForOpenLink = result.AppsPolicy.RequireApprovalForOpenLink ||
			candidate.AppsPolicy.RequireApprovalForOpenLink
		result.AppsPolicy.RequireApprovalForContextUpdates = result.AppsPolicy.RequireApprovalForContextUpdates ||
			candidate.AppsPolicy.RequireApprovalForContextUpdates

		if result.ToolPolicies == nil {
			result.ToolPolicies = map[string]MCPToolPolicyOverride{}
		}
		names := make([]string, 0, len(candidate.ToolPolicies))
		for name := range candidate.ToolPolicies {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			incoming := candidate.ToolPolicies[name]
			current, found := result.ToolPolicies[name]
			if !found {
				result.ToolPolicies[name] = cloneOverride(incoming)
				continue
			}
			merged, conflict := mergeOverride(current, incoming)
			result.ToolPolicies[name] = merged
			if conflict != "" {
				conflicts[name] = conflict
			}
		}
	}

	digest, err := effectiveDigest(result, conflicts)
	if err != nil {
		return Effective{}, err
	}
	return Effective{
		Body:      result,
		Conflicts: maps.Clone(conflicts),
		Digest:    digest,
	}, nil
}

func NormalizePolicyBody(input MCPPolicy) MCPPolicy {
	output := input
	output.ToolPolicies = maps.Clone(input.ToolPolicies)
	if output.TrustLevel == "" {
		output.TrustLevel = MCPTrustLevelUntrusted
	}
	if output.DefaultPolicy == (MCPServerPolicy{}) {
		output.DefaultPolicy = DefaultMCPServerPolicy()
	}
	if output.ToolPolicies == nil {
		output.ToolPolicies = map[string]MCPToolPolicyOverride{}
	}
	for name, override := range output.ToolPolicies {
		if override.ToolName == "" {
			override.ToolName = name
		}
		output.ToolPolicies[name] = override
	}
	if output.AppsPolicy == (MCPAppsPolicy{}) {
		output.AppsPolicy = MCPAppsPolicy{
			Enabled:                          false,
			AllowAppInitiatedToolCalls:       false,
			RequireApprovalForOpenLink:       true,
			RequireApprovalForContextUpdates: true,
		}
	}
	return output
}

func (e Effective) Validate() error {
	if err := ValidatePolicyBody(e.Body); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(e.Digest); err != nil {
		return err
	}
	calculated, err := effectiveDigest(e.Body, e.Conflicts)
	if err != nil {
		return err
	}
	if calculated != e.Digest {
		return fmt.Errorf(
			"%w: effective MCP policy digest mismatch",
			basespec.ErrDigestMismatch,
		)
	}
	return nil
}

func mergeOverride(
	left MCPToolPolicyOverride,
	right MCPToolPolicyOverride,
) (merged MCPToolPolicyOverride, conflict string) {
	output := cloneOverride(left)
	if output.ToolName == "" {
		output.ToolName = right.ToolName
	}

	if right.ApprovalRule != nil {
		value := *right.ApprovalRule
		if output.ApprovalRule != nil {
			value = restrictiveApproval(*output.ApprovalRule, value)
		}
		output.ApprovalRule = &value
	}
	if right.ExecutionMode != nil {
		value := *right.ExecutionMode
		if output.ExecutionMode != nil {
			value = restrictiveExecution(*output.ExecutionMode, value)
		}
		output.ExecutionMode = &value
	}

	output.AllowStaleDigest = output.AllowStaleDigest && right.AllowStaleDigest

	switch {
	case output.ExpectedDigest == "":
		output.ExpectedDigest = right.ExpectedDigest
	case right.ExpectedDigest == "":
	case output.ExpectedDigest != right.ExpectedDigest:
		conflict = "conflicting expected tool digests"
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
	rank := func(value MCPApprovalRule) int {
		switch value {
		case MCPApprovalRuleDeny:
			return 3
		case MCPApprovalRuleAsk:
			return 2
		default:
			return 1
		}
	}
	if rank(left) >= rank(right) {
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

func cloneOverride(
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

func effectiveDigest(
	body MCPPolicy,
	conflicts map[string]string,
) (cryptoutil.Digest, error) {
	raw, err := json.Marshal(struct {
		Body      MCPPolicy         `json:"body"`
		Conflicts map[string]string `json:"conflicts,omitempty"`
	}{
		Body:      body,
		Conflicts: conflicts,
	})
	if err != nil {
		return "", err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	return cryptoutil.DigestBytes(canonical), nil
}
