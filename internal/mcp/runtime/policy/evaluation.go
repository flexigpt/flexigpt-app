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
	switch {
	case hints.DestructiveHint != nil && *hints.DestructiveHint:
		return "destructive"
	case hints.OpenWorldHint != nil && *hints.OpenWorldHint:
		return "openWorld"
	case hints.ReadOnlyHint && trustLevel == MCPTrustLevelTrusted:
		return "read"
	case hints.DestructiveHint != nil && !*hints.DestructiveHint:
		return "write"
	default:
		return "unknown"
	}
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
	value = Normalize(value)

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
	value = Normalize(value)

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
	if err := mappedApproval.Validate(); err != nil {
		return MCPPolicy{}, err
	}
	if err := mappedExecution.Validate(); err != nil {
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

	output := Clone(value)
	override := output.ToolPolicies[toolName]
	override.ToolName = toolName

	approval := mappedApproval
	execution := mappedExecution
	override.ApprovalRule = &approval
	override.ExecutionMode = &execution

	output.ToolPolicies[toolName] = override
	if err := output.Validate(); err != nil {
		return MCPPolicy{}, err
	}
	return output, nil
}
