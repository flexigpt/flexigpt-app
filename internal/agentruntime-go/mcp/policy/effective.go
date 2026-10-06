package policy

import (
	"encoding/json"
	"fmt"
	"maps"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Effective struct {
	Body      MCPPolicy         `json:"body"`
	Conflicts map[string]string `json:"conflicts,omitempty"`
	Digest    cryptoutil.Digest `json:"digest"`
}

func (value Effective) Validate() error {
	if err := value.Body.Validate(); err != nil {
		return fmt.Errorf("%w: invalid effective MCP policy: %w", ErrInvalid, err)
	}
	if err := cryptoutil.ValidateDigest(value.Digest); err != nil {
		return err
	}

	calculated, err := effectiveDigest(value.Body, value.Conflicts)
	if err != nil {
		return err
	}
	if calculated != value.Digest {
		return fmt.Errorf("%w: effective MCP policy digest mismatch", ErrInvalid)
	}
	return nil
}

func Compose(
	baseline MCPPolicy,
	policies ...MCPPolicy,
) (Effective, error) {
	composed, err := compose(baseline, policies...)
	if err != nil {
		return Effective{}, err
	}

	digest, err := effectiveDigest(composed.Body, composed.Conflicts)
	if err != nil {
		return Effective{}, err
	}

	output := Effective{
		Body:      Clone(composed.Body),
		Conflicts: maps.Clone(composed.Conflicts),
		Digest:    digest,
	}
	return output, nil
}

func compose(
	baseline MCPPolicy,
	policies ...MCPPolicy,
) (Composition, error) {
	return ComposePolicies(baseline, policies...)
}

// ComposePolicies exists only as the composition boundary for callers that
// require the conflict set without calculating an Effective digest.
func ComposePolicies(
	baseline MCPPolicy,
	policies ...MCPPolicy,
) (Composition, error) {
	return composePolicies(baseline, policies...)
}

func composePolicies(
	baseline MCPPolicy,
	policies ...MCPPolicy,
) (Composition, error) {
	result := Normalize(baseline)
	if err := result.Validate(); err != nil {
		return Composition{}, err
	}

	normalized := make([]MCPPolicy, 0, len(policies)+1)
	normalized = append(normalized, result)

	conflicts := map[string]string{}
	for index, candidate := range policies {
		candidate = Normalize(candidate)
		if err := candidate.Validate(); err != nil {
			return Composition{}, fmt.Errorf("policy %d: %w", index, err)
		}

		normalized = append(normalized, candidate)
		result.TrustLevel = restrictiveTrust(result.TrustLevel, candidate.TrustLevel)
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

	if err := result.Validate(); err != nil {
		return Composition{}, err
	}

	return Composition{
		Body:      result,
		Conflicts: maps.Clone(conflicts),
	}, nil
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
