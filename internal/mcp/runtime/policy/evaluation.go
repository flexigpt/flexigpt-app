package policy

import "fmt"

const (
	ToolDigestChangedReason = "tool digest changed"
	ToolPolicyDeniesReason  = "server/tool policy denies this tool"
	PolicyAllowedReason     = "policy allowed"
)

type ToolRiskHints struct {
	DestructiveHint *bool
	OpenWorldHint   *bool
	ReadOnlyHint    bool
}

func InferToolRisk(
	hints ToolRiskHints,
	trustLevel MCPTrustLevel,
) string {
	risk := "unknown"
	if hints.DestructiveHint != nil && *hints.DestructiveHint {
		risk = "destructive"
	}
	if hints.OpenWorldHint != nil && *hints.OpenWorldHint {
		risk = "openWorld"
	}
	if hints.ReadOnlyHint && trustLevel == MCPTrustLevelTrusted {
		risk = "read"
	}
	if hints.DestructiveHint != nil && !*hints.DestructiveHint {
		risk = "write"
	}
	return risk
}

type ToolDecision string

const (
	ToolDecisionAllowed          ToolDecision = "allowed"
	ToolDecisionDenied           ToolDecision = "denied"
	ToolDecisionApprovalRequired ToolDecision = "approvalRequired"
)

type ToolEvaluationInput struct {
	Enabled             bool
	TaskSupportRequired bool

	ToolDigest          string
	RequestedToolDigest string
	ExpectedDigest      string
	AllowStaleDigest    bool

	Risk          string
	ApprovalRule  MCPApprovalRule
	ExecutionMode MCPExecutionMode
	Source        string
}

type ToolEvaluation struct {
	Decision ToolDecision
	Reason   string
}

func EffectiveToolPolicy(
	value MCPPolicy,
	toolName string,
) (
	MCPApprovalRule,
	MCPExecutionMode,
	MCPToolPolicyOverride,
	bool,
) {
	value = NormalizeMCPPolicy(value)

	approvalRule := value.DefaultPolicy.DefaultApprovalRule
	executionMode := value.DefaultPolicy.DefaultExecutionMode

	override, found := value.ToolPolicies[toolName]
	if found {
		if override.ApprovalRule != nil {
			approvalRule = *override.ApprovalRule
		}
		if override.ExecutionMode != nil {
			executionMode = *override.ExecutionMode
		}
	}

	return approvalRule, executionMode, cloneToolPolicyOverride(override), found
}

func EffectiveToolConstraints(
	value MCPPolicy,
	toolName string,
	discoveredApproval MCPApprovalRule,
	discoveredExecution MCPExecutionMode,
) (MCPApprovalRule, MCPExecutionMode) {
	approvalRule, executionMode, _, _ := EffectiveToolPolicy(value, toolName)

	discoveredApproval = NormalizedApprovalRule(discoveredApproval)
	discoveredExecution = NormalizedExecutionMode(discoveredExecution)

	if ApprovalRuleRank(discoveredApproval) > ApprovalRuleRank(approvalRule) {
		approvalRule = discoveredApproval
	}
	if ExecutionModeRank(discoveredExecution) > ExecutionModeRank(executionMode) {
		executionMode = discoveredExecution
	}

	return approvalRule, executionMode
}

func EvaluateTool(
	value MCPPolicy,
	input ToolEvaluationInput,
) ToolEvaluation {
	value = NormalizeMCPPolicy(value)

	if !input.Enabled || input.TaskSupportRequired {
		return ToolEvaluation{
			Decision: ToolDecisionDenied,
			Reason:   "tool is disabled or unsupported",
		}
	}

	if input.ExpectedDigest != "" &&
		input.ExpectedDigest != input.ToolDigest &&
		!input.AllowStaleDigest {
		return ToolEvaluation{
			Decision: ToolDecisionDenied,
			Reason:   ToolDigestChangedReason,
		}
	}

	if input.RequestedToolDigest != "" &&
		input.RequestedToolDigest != input.ToolDigest &&
		!input.AllowStaleDigest {
		return ToolEvaluation{
			Decision: ToolDecisionDenied,
			Reason:   ToolDigestChangedReason,
		}
	}

	approvalRule := input.ApprovalRule
	if approvalRule == "" {
		approvalRule = value.DefaultPolicy.DefaultApprovalRule
	}
	approvalRule = NormalizedApprovalRule(approvalRule)

	executionMode := input.ExecutionMode
	if executionMode == "" {
		executionMode = value.DefaultPolicy.DefaultExecutionMode
	}
	executionMode = NormalizedExecutionMode(executionMode)

	if approvalRule == MCPApprovalRuleDeny {
		return ToolEvaluation{
			Decision: ToolDecisionDenied,
			Reason:   ToolPolicyDeniesReason,
		}
	}

	if executionMode == MCPExecutionModeManual && input.Source != "user" {
		return ToolEvaluation{
			Decision: ToolDecisionApprovalRequired,
			Reason:   "manual execution mode requires approval",
		}
	}

	if approvalRule == MCPApprovalRuleAsk {
		return ToolEvaluation{
			Decision: ToolDecisionApprovalRequired,
			Reason:   "approval rule is ask",
		}
	}

	switch input.Risk {
	case "unknown", "openWorld":
		if value.DefaultPolicy.RequireApprovalForUnknownRisk {
			return ToolEvaluation{
				Decision: ToolDecisionApprovalRequired,
				Reason:   "unknown-risk tool requires approval",
			}
		}
	case "write":
		if value.DefaultPolicy.RequireApprovalForWrite {
			return ToolEvaluation{
				Decision: ToolDecisionApprovalRequired,
				Reason:   "write-risk tool requires approval",
			}
		}
	case "destructive":
		if value.DefaultPolicy.RequireApprovalForDestructive {
			return ToolEvaluation{
				Decision: ToolDecisionApprovalRequired,
				Reason:   "destructive-risk tool requires approval",
			}
		}
	}

	return ToolEvaluation{
		Decision: ToolDecisionAllowed,
		Reason:   PolicyAllowedReason,
	}
}

func TightenToolPolicy(
	value MCPPolicy,
	toolName string,
	currentApproval MCPApprovalRule,
	currentExecution MCPExecutionMode,
	mappedApproval MCPApprovalRule,
	mappedExecution MCPExecutionMode,
) (MCPPolicy, error) {
	if err := ValidateMCPApprovalRule(mappedApproval); err != nil {
		return MCPPolicy{}, err
	}
	if err := ValidateMCPExecutionMode(mappedExecution); err != nil {
		return MCPPolicy{}, err
	}

	currentApproval = NormalizedApprovalRule(currentApproval)
	currentExecution = NormalizedExecutionMode(currentExecution)

	if ApprovalRuleRank(mappedApproval) < ApprovalRuleRank(currentApproval) {
		return MCPPolicy{}, fmt.Errorf(
			"%w: mapped approval rule weakens current MCP policy",
			ErrInvalid,
		)
	}
	if ExecutionModeRank(mappedExecution) < ExecutionModeRank(currentExecution) {
		return MCPPolicy{}, fmt.Errorf(
			"%w: mapped execution mode weakens current MCP policy",
			ErrInvalid,
		)
	}

	output := CloneMCPPolicy(value)
	override := output.ToolPolicies[toolName]
	override.ToolName = toolName

	approval := mappedApproval
	execution := mappedExecution
	override.ApprovalRule = &approval
	override.ExecutionMode = &execution

	output.ToolPolicies[toolName] = override
	return output, nil
}
