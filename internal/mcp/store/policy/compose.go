package policy

import (
	"encoding/json"
	"fmt"
	"maps"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/spec"
)

type Effective struct {
	Body      MCPPolicy         `json:"body"`
	Conflicts map[string]string `json:"conflicts,omitempty"`
	Digest    cryptoutil.Digest `json:"digest"`
}

func Baseline() MCPPolicy {
	return NormalizePolicyBody(MCPPolicy{
		TrustLevel:    mcpSpec.MCPTrustLevelUntrusted,
		DefaultPolicy: DefaultMCPServerPolicy(),
		AppsPolicy: mcpSpec.MCPAppsPolicy{
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

	normalizedPolicies := make(
		[]MCPPolicy,
		0,
		1+len(policies),
	)
	normalizedPolicies = append(normalizedPolicies, result)

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
		normalizedPolicies = append(normalizedPolicies, candidate)

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

	}

	nameSet := make(map[string]struct{})
	for _, candidate := range normalizedPolicies {
		for name := range candidate.ToolPolicies {
			nameSet[name] = struct{}{}
		}
	}

	names := make([]string, 0, len(nameSet))
	for name := range nameSet {
		names = append(names, name)
	}
	sort.Strings(names)

	result.ToolPolicies = make(
		map[string]mcpSpec.MCPToolPolicyOverride,
		len(names),
	)
	for _, name := range names {
		merged, conflict := composeToolOverride(
			name,
			normalizedPolicies,
		)
		result.ToolPolicies[name] = merged
		if conflict != "" {
			conflicts[name] = conflict
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
		output.TrustLevel = mcpSpec.MCPTrustLevelUntrusted
	}
	if output.DefaultPolicy == (mcpSpec.MCPServerPolicy{}) {
		output.DefaultPolicy = DefaultMCPServerPolicy()
	}
	if output.ToolPolicies == nil {
		output.ToolPolicies = map[string]mcpSpec.MCPToolPolicyOverride{}
	}
	for name, override := range output.ToolPolicies {
		if override.ToolName == "" {
			override.ToolName = name
		}
		output.ToolPolicies[name] = override
	}
	if output.AppsPolicy == (mcpSpec.MCPAppsPolicy{}) {
		output.AppsPolicy = mcpSpec.MCPAppsPolicy{
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

func composeToolOverride(
	name string,
	policies []MCPPolicy,
) (merged mcpSpec.MCPToolPolicyOverride, conflict string) {
	output := mcpSpec.MCPToolPolicyOverride{ToolName: name}

	var (
		approvalRule  mcpSpec.MCPApprovalRule
		executionMode mcpSpec.MCPExecutionMode
		first         = true
		sawOverride   bool
		allowStale    = true
		expected      string
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

		approvalRule = restrictiveApproval(
			approvalRule,
			candidateApproval,
		)
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
		deny := mcpSpec.MCPApprovalRuleDeny
		output.ApprovalRule = &deny
		output.AllowStaleDigest = false
	}
	return output, conflict
}

func restrictiveTrust(
	left mcpSpec.MCPTrustLevel,
	right mcpSpec.MCPTrustLevel,
) mcpSpec.MCPTrustLevel {
	if left == mcpSpec.MCPTrustLevelUntrusted ||
		right == mcpSpec.MCPTrustLevelUntrusted {
		return mcpSpec.MCPTrustLevelUntrusted
	}
	return mcpSpec.MCPTrustLevelTrusted
}

func restrictiveApproval(
	left mcpSpec.MCPApprovalRule,
	right mcpSpec.MCPApprovalRule,
) mcpSpec.MCPApprovalRule {
	rank := func(value mcpSpec.MCPApprovalRule) int {
		switch value {
		case mcpSpec.MCPApprovalRuleDeny:
			return 3
		case mcpSpec.MCPApprovalRuleAsk:
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
	left mcpSpec.MCPExecutionMode,
	right mcpSpec.MCPExecutionMode,
) mcpSpec.MCPExecutionMode {
	if left == mcpSpec.MCPExecutionModeManual ||
		right == mcpSpec.MCPExecutionModeManual {
		return mcpSpec.MCPExecutionModeManual
	}
	return mcpSpec.MCPExecutionModeAuto
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
