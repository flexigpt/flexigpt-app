package invocation

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpRuntime "github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/spec"
)

const (
	toolDigestChangedReason = "tool digest changed"
	toolPolicyDeniesReason  = "server/tool policy denies this tool"
	policyAllowedReason     = "policy allowed"
)

type ToolBridge struct {
	runtime   Runtime
	approvals *mcpRuntime.ApprovalManager
}

func NewToolBridge(
	runtime Runtime,
	approvals *mcpRuntime.ApprovalManager,
) *ToolBridge {
	if runtime != nil {
		runtime.SetSessionLifecycleCleaner(approvals)
	}
	return &ToolBridge{
		runtime:   runtime,
		approvals: approvals,
	}
}

func (b *ToolBridge) ResolveApproval(
	ctx context.Context,
	approvalID string,
	resolution mcpRuntime.MCPApprovalResolution,
) (mcpRuntime.MCPApprovalResolutionResult, error) {
	if b == nil || b.approvals == nil {
		return mcpRuntime.MCPApprovalResolutionResult{}, fmt.Errorf(
			"%w: MCP approval manager is unavailable",
			mcpRuntime.ErrMCPRuntimeNotReady,
		)
	}
	return b.approvals.Resolve(
		ctx,
		approvalID,
		resolution,
	)
}

func (b *ToolBridge) Evaluate(
	ctx context.Context,
	serverRef mcpSpec.ServerID,
	request mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.MCPApprovalEvaluation, error) {
	if err := validateBridgeRequest(ctx, serverRef, request); err != nil {
		return nil, err
	}
	if b == nil || b.runtime == nil || b.approvals == nil {
		return nil, fmt.Errorf("%w: MCP tool bridge is unavailable", mcpRuntime.ErrMCPRuntimeNotReady)
	}

	config, tool, err := b.runtime.CallToolDryRun(
		ctx,
		serverRef,
		request,
	)
	if err != nil {
		return nil, err
	}

	if request.Source == mcpRuntime.MCPInvocationSourceApp {
		if err := mcpRuntime.ValidateAppToolInvocation(
			config.AppsPolicy,
			tool,
			serverRef,
		); err != nil {
			return nil, err
		}
	} else if config.AppsPolicy.Enabled &&
		!mcpRuntime.ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			mcpRuntime.ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := evaluateTool(config, tool, request)
	evaluation = b.applyCachedDecision(evaluation)

	if evaluation.Decision == mcpRuntime.MCPApprovalDecisionApprovalRequired &&
		evaluation.Summary != nil {
		approvalID, err := b.approvals.Create(
			ctx,
			*evaluation.Summary,
		)
		if err != nil {
			return nil, err
		}
		evaluation.ApprovalID = approvalID
	}

	return &evaluation, nil
}

// EvaluateMapped evaluates a model-originated provider-tool call against the
// persisted mapping produced by MCP inference hydration.
//
// The mapping is authoritative for server identity, provider name, choice ID,
// discovered digest, and any conversation policy tightening.
func (b *ToolBridge) EvaluateMapped(
	ctx context.Context,
	mapping mcpRuntime.MCPProviderToolMapping,
	request mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.MCPApprovalEvaluation, error) {
	config, tool, normalized, err := b.mappedDryRun(
		ctx,
		mapping,
		request,
	)
	if err != nil {
		return nil, err
	}

	if normalized.Source == mcpRuntime.MCPInvocationSourceApp {
		if err := mcpRuntime.ValidateAppToolInvocation(
			config.AppsPolicy,
			tool,
			mapping.Server,
		); err != nil {
			return nil, err
		}
	} else if config.AppsPolicy.Enabled &&
		!mcpRuntime.ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			mcpRuntime.ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := b.applyCachedDecision(
		evaluateTool(config, tool, normalized),
	)
	if evaluation.Decision != mcpRuntime.MCPApprovalDecisionApprovalRequired ||
		evaluation.Summary == nil {
		return &evaluation, nil
	}

	approvalID, err := b.approvals.Create(ctx, *evaluation.Summary)
	if err != nil {
		return nil, err
	}
	evaluation.ApprovalID = approvalID
	return &evaluation, nil
}

// InvokeMapped invokes a provider-tool mapping after revalidating the current
// connected runtime snapshot and applying only policy-tightening constraints.
func (b *ToolBridge) InvokeMapped(
	ctx context.Context,
	mapping mcpRuntime.MCPProviderToolMapping,
	request mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.InvokeMCPToolResponseBody, error) {
	config, tool, normalized, err := b.mappedDryRun(
		ctx,
		mapping,
		request,
	)
	if err != nil {
		return nil, err
	}

	if normalized.Source == mcpRuntime.MCPInvocationSourceApp {
		if err := mcpRuntime.ValidateAppToolInvocation(
			config.AppsPolicy,
			tool,
			mapping.Server,
		); err != nil {
			return nil, err
		}
	} else if config.AppsPolicy.Enabled &&
		!mcpRuntime.ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			mcpRuntime.ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := b.applyCachedDecision(
		evaluateTool(config, tool, normalized),
	)
	switch evaluation.Decision {
	case mcpRuntime.MCPApprovalDecisionDenied:
		if evaluation.Reason == toolDigestChangedReason {
			return nil, fmt.Errorf(
				"%w: %s",
				mcpRuntime.ErrMCPStaleReference,
				evaluation.Reason,
			)
		}
		return nil, fmt.Errorf(
			"%w: %s",
			mcpRuntime.ErrMCPPolicyDenied,
			evaluation.Reason,
		)

	case mcpRuntime.MCPApprovalDecisionApprovalRequired:
		if normalized.ApprovalToken == "" || evaluation.Summary == nil {
			return nil, fmt.Errorf(
				"%w: approval token is required",
				mcpRuntime.ErrMCPApprovalNeeded,
			)
		}
		approvalID, err := b.approvals.VerifyAndConsumeToken(
			ctx,
			normalized.ApprovalToken,
			*evaluation.Summary,
		)
		if err != nil {
			return nil, err
		}
		normalized.ApprovalID = approvalID

	case mcpRuntime.MCPApprovalDecisionAllowed:
	default:
		return nil, fmt.Errorf(
			"%w: invalid MCP approval decision",
			mcpRuntime.ErrMCPInvalidRuntimeRequest,
		)
	}

	return b.runtime.CallTool(ctx, mapping.Server, normalized)
}

func (b *ToolBridge) Invoke(
	ctx context.Context,
	serverRef mcpSpec.ServerID,
	request mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.InvokeMCPToolResponseBody, error) {
	if err := validateBridgeRequest(ctx, serverRef, request); err != nil {
		return nil, err
	}
	if b == nil || b.runtime == nil || b.approvals == nil {
		return nil, fmt.Errorf("%w: MCP tool bridge is unavailable", mcpRuntime.ErrMCPRuntimeNotReady)
	}

	config, tool, err := b.runtime.CallToolDryRun(
		ctx,
		serverRef,
		request,
	)
	if err != nil {
		return nil, err
	}

	if request.Source == mcpRuntime.MCPInvocationSourceApp {
		if err := mcpRuntime.ValidateAppToolInvocation(
			config.AppsPolicy,
			tool,
			serverRef,
		); err != nil {
			return nil, err
		}
	} else if config.AppsPolicy.Enabled &&
		!mcpRuntime.ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			mcpRuntime.ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := evaluateTool(config, tool, request)
	evaluation = b.applyCachedDecision(evaluation)

	switch evaluation.Decision {
	case mcpRuntime.MCPApprovalDecisionDenied:
		if evaluation.Reason == toolDigestChangedReason {
			return nil, fmt.Errorf(
				"%w: %s",
				mcpRuntime.ErrMCPStaleReference,
				evaluation.Reason,
			)
		}
		return nil, fmt.Errorf(
			"%w: %s",
			mcpRuntime.ErrMCPPolicyDenied,
			evaluation.Reason,
		)

	case mcpRuntime.MCPApprovalDecisionApprovalRequired:
		if request.ApprovalToken == "" || evaluation.Summary == nil {
			return nil, fmt.Errorf(
				"%w: approval token is required",
				mcpRuntime.ErrMCPApprovalNeeded,
			)
		}
		approvalID, err := b.approvals.VerifyAndConsumeToken(
			ctx,
			request.ApprovalToken,
			*evaluation.Summary,
		)
		if err != nil {
			return nil, err
		}
		request.ApprovalID = approvalID

	case mcpRuntime.MCPApprovalDecisionAllowed:
	default:
		return nil, fmt.Errorf(
			"%w: invalid MCP approval decision",
			mcpRuntime.ErrMCPInvalidRuntimeRequest,
		)
	}

	return b.runtime.CallTool(ctx, serverRef, request)
}

func (b *ToolBridge) mappedDryRun(
	ctx context.Context,
	mapping mcpRuntime.MCPProviderToolMapping,
	request mcpRuntime.InvokeMCPToolRequestBody,
) (
	config mcpSpec.RuntimeConfig,
	tool mcpRuntime.MCPToolCapability,
	normalized mcpRuntime.InvokeMCPToolRequestBody,
	err error,
) {
	if b == nil || b.runtime == nil || b.approvals == nil {
		return mcpSpec.RuntimeConfig{},
			mcpRuntime.MCPToolCapability{},
			mcpRuntime.InvokeMCPToolRequestBody{},
			fmt.Errorf(
				"%w: MCP tool bridge is unavailable",
				mcpRuntime.ErrMCPRuntimeNotReady,
			)
	}

	normalized, err = normalizeMappedToolRequest(mapping, request)
	if err != nil {
		return mcpSpec.RuntimeConfig{},
			mcpRuntime.MCPToolCapability{},
			mcpRuntime.InvokeMCPToolRequestBody{},
			err
	}
	if err := validateBridgeRequest(ctx, mapping.Server, normalized); err != nil {
		return mcpSpec.RuntimeConfig{},
			mcpRuntime.MCPToolCapability{},
			mcpRuntime.InvokeMCPToolRequestBody{},
			err
	}

	config, tool, err = b.runtime.CallToolDryRun(
		ctx,
		mapping.Server,
		normalized,
	)
	if err != nil {
		return mcpSpec.RuntimeConfig{},
			mcpRuntime.MCPToolCapability{},
			mcpRuntime.InvokeMCPToolRequestBody{},
			err
	}
	if tool.ProviderToolName != mapping.ProviderToolName ||
		tool.ChoiceID != mapping.ChoiceID ||
		tool.ToolName != mapping.ToolName ||
		tool.Digest != mapping.ToolDigest {
		return mcpSpec.RuntimeConfig{},
			mcpRuntime.MCPToolCapability{},
			mcpRuntime.InvokeMCPToolRequestBody{},
			fmt.Errorf(
				"%w: MCP provider tool mapping is stale",
				mcpRuntime.ErrMCPStaleReference,
			)
	}

	config, err = applyMappedPolicyConstraints(
		config,
		tool,
		mapping,
	)
	if err != nil {
		return mcpSpec.RuntimeConfig{},
			mcpRuntime.MCPToolCapability{},
			mcpRuntime.InvokeMCPToolRequestBody{},
			err
	}
	return config, tool, normalized, nil
}

func normalizeMappedToolRequest(
	mapping mcpRuntime.MCPProviderToolMapping,
	request mcpRuntime.InvokeMCPToolRequestBody,
) (mcpRuntime.InvokeMCPToolRequestBody, error) {
	if err := mcpRuntime.ValidateMCPProviderToolMapping(mapping); err != nil {
		return mcpRuntime.InvokeMCPToolRequestBody{}, err
	}

	request.ToolName = strings.TrimSpace(request.ToolName)
	request.ProviderToolName = strings.TrimSpace(
		request.ProviderToolName,
	)

	switch request.ToolName {
	case "":
		if request.ProviderToolName != mapping.ProviderToolName {
			return mcpRuntime.InvokeMCPToolRequestBody{}, fmt.Errorf(
				"%w: provider tool name does not match the persisted mapping",
				mcpRuntime.ErrMCPInvalidRuntimeRequest,
			)
		}
		request.ToolName = mapping.ToolName

	case mapping.ProviderToolName:
		request.ToolName = mapping.ToolName

	case mapping.ToolName:
	default:
		return mcpRuntime.InvokeMCPToolRequestBody{}, fmt.Errorf(
			"%w: tool name does not match the persisted mapping",
			mcpRuntime.ErrMCPInvalidRuntimeRequest,
		)
	}

	if request.ProviderToolName != "" &&
		request.ProviderToolName != mapping.ProviderToolName {
		return mcpRuntime.InvokeMCPToolRequestBody{}, fmt.Errorf(
			"%w: provider tool name does not match the persisted mapping",
			mcpRuntime.ErrMCPInvalidRuntimeRequest,
		)
	}

	if request.ToolDigest != "" &&
		request.ToolDigest != mapping.ToolDigest {
		return mcpRuntime.InvokeMCPToolRequestBody{}, fmt.Errorf(
			"%w: provider tool digest does not match the persisted mapping",
			mcpRuntime.ErrMCPStaleReference,
		)
	}

	request.ProviderToolName = mapping.ProviderToolName
	request.ToolDigest = mapping.ToolDigest
	return request, nil
}

func applyMappedPolicyConstraints(
	config mcpSpec.RuntimeConfig,
	tool mcpRuntime.MCPToolCapability,
	mapping mcpRuntime.MCPProviderToolMapping,
) (mcpSpec.RuntimeConfig, error) {
	currentApproval, currentExecution := currentToolConstraints(
		config,
		tool,
	)
	if mcpPolicy.ApprovalRuleRank(mapping.ApprovalRule) <
		mcpPolicy.ApprovalRuleRank(currentApproval) {
		return mcpSpec.RuntimeConfig{}, fmt.Errorf(
			"%w: mapped approval rule weakens current MCP policy",
			mcpRuntime.ErrMCPPolicyDenied,
		)
	}
	if mcpPolicy.ExecutionModeRank(mapping.ExecutionMode) <
		mcpPolicy.ExecutionModeRank(currentExecution) {
		return mcpSpec.RuntimeConfig{}, fmt.Errorf(
			"%w: mapped execution mode weakens current MCP policy",
			mcpRuntime.ErrMCPPolicyDenied,
		)
	}

	output := config
	output.ToolPolicies = maps.Clone(config.ToolPolicies)
	if output.ToolPolicies == nil {
		output.ToolPolicies = map[string]mcpPolicy.MCPToolPolicyOverride{}
	}

	override := output.ToolPolicies[tool.ToolName]
	override.ToolName = tool.ToolName
	approval := mapping.ApprovalRule
	execution := mapping.ExecutionMode
	override.ApprovalRule = &approval
	override.ExecutionMode = &execution
	output.ToolPolicies[tool.ToolName] = override
	return output, nil
}

func currentToolConstraints(
	config mcpSpec.RuntimeConfig,
	tool mcpRuntime.MCPToolCapability,
) (mcpPolicy.MCPApprovalRule, mcpPolicy.MCPExecutionMode) {
	defaults := mcpPolicy.DefaultMCPServerPolicy()
	approval := config.DefaultPolicy.DefaultApprovalRule
	execution := config.DefaultPolicy.DefaultExecutionMode
	if approval == "" {
		approval = defaults.DefaultApprovalRule
	}
	if execution == "" {
		execution = defaults.DefaultExecutionMode
	}
	if override, found := config.ToolPolicies[tool.ToolName]; found {
		if override.ApprovalRule != nil {
			approval = *override.ApprovalRule
		}
		if override.ExecutionMode != nil {
			execution = *override.ExecutionMode
		}
	}

	toolApproval := mcpPolicy.NormalizedApprovalRule(tool.ApprovalRule)
	if mcpPolicy.ApprovalRuleRank(toolApproval) > mcpPolicy.ApprovalRuleRank(approval) {
		approval = toolApproval
	}
	toolExecution := mcpPolicy.NormalizedExecutionMode(tool.ExecutionMode)
	if mcpPolicy.ExecutionModeRank(toolExecution) > mcpPolicy.ExecutionModeRank(execution) {
		execution = toolExecution
	}
	return approval, execution
}

func evaluateTool(
	config mcpSpec.RuntimeConfig,
	tool mcpRuntime.MCPToolCapability,
	request mcpRuntime.InvokeMCPToolRequestBody,
) mcpRuntime.MCPApprovalEvaluation {
	p := config.DefaultPolicy
	if p == (mcpPolicy.MCPServerPolicy{}) {
		p = mcpPolicy.DefaultMCPServerPolicy()
	}

	if !tool.Enabled ||
		tool.TaskSupport == mcpRuntime.MCPTaskSupportRequired {
		return deniedEvaluation(
			config,
			tool,
			request,
			"tool is disabled or unsupported",
		)
	}

	rule := p.DefaultApprovalRule
	executionMode := p.DefaultExecutionMode
	if executionMode == "" {
		executionMode = mcpPolicy.MCPExecutionModeManual
	}
	allowStaleDigest := false

	if override, found := config.ToolPolicies[tool.ToolName]; found {
		if override.ApprovalRule != nil {
			rule = *override.ApprovalRule
		}
		if override.ExecutionMode != nil {
			executionMode = *override.ExecutionMode
		}
		allowStaleDigest = override.AllowStaleDigest
		if override.ExpectedDigest != "" &&
			override.ExpectedDigest != tool.Digest &&
			!allowStaleDigest {
			return deniedEvaluation(
				config,
				tool,
				request,
				toolDigestChangedReason,
			)
		}
	}

	if request.ToolDigest != "" &&
		request.ToolDigest != tool.Digest &&
		!allowStaleDigest {
		return deniedEvaluation(
			config,
			tool,
			request,
			toolDigestChangedReason,
		)
	}

	if rule == mcpPolicy.MCPApprovalRuleDeny {
		return deniedEvaluation(
			config,
			tool,
			request,
			toolPolicyDeniesReason,
		)
	}
	if executionMode == mcpPolicy.MCPExecutionModeManual &&
		request.Source != mcpRuntime.MCPInvocationSourceUser {
		return approvalRequiredEvaluation(
			config,
			tool,
			request,
			"manual execution mode requires approval",
		)
	}
	if rule == mcpPolicy.MCPApprovalRuleAsk {
		return approvalRequiredEvaluation(
			config,
			tool,
			request,
			"approval rule is ask",
		)
	}

	switch tool.InferredRisk {
	case mcpRuntime.MCPToolRiskUnknown:
		if p.RequireApprovalForUnknownRisk {
			return approvalRequiredEvaluation(
				config,
				tool,
				request,
				"unknown-risk tool requires approval",
			)
		}
	case mcpRuntime.MCPToolRiskOpenWorld:
		if p.RequireApprovalForUnknownRisk {
			return approvalRequiredEvaluation(
				config,
				tool,
				request,
				"open-world tool requires approval",
			)
		}
	case mcpRuntime.MCPToolRiskWrite:
		if p.RequireApprovalForWrite {
			return approvalRequiredEvaluation(
				config,
				tool,
				request,
				"write-risk tool requires approval",
			)
		}
	case mcpRuntime.MCPToolRiskDestructive:
		if p.RequireApprovalForDestructive {
			return approvalRequiredEvaluation(
				config,
				tool,
				request,
				"destructive-risk tool requires approval",
			)
		}
	default:
	}

	return mcpRuntime.MCPApprovalEvaluation{
		Decision: mcpRuntime.MCPApprovalDecisionAllowed,
		Reason:   policyAllowedReason,
		Summary:  approvalSummary(config, tool, request),
	}
}

func deniedEvaluation(
	config mcpSpec.RuntimeConfig,
	tool mcpRuntime.MCPToolCapability,
	request mcpRuntime.InvokeMCPToolRequestBody,
	reason string,
) mcpRuntime.MCPApprovalEvaluation {
	return mcpRuntime.MCPApprovalEvaluation{
		Decision: mcpRuntime.MCPApprovalDecisionDenied,
		Reason:   reason,
		Summary:  approvalSummary(config, tool, request),
	}
}

func approvalRequiredEvaluation(
	config mcpSpec.RuntimeConfig,
	tool mcpRuntime.MCPToolCapability,
	request mcpRuntime.InvokeMCPToolRequestBody,
	reason string,
) mcpRuntime.MCPApprovalEvaluation {
	return mcpRuntime.MCPApprovalEvaluation{
		Decision: mcpRuntime.MCPApprovalDecisionApprovalRequired,
		Reason:   reason,
		Summary:  approvalSummary(config, tool, request),
	}
}

func approvalSummary(
	config mcpSpec.RuntimeConfig,
	tool mcpRuntime.MCPToolCapability,
	request mcpRuntime.InvokeMCPToolRequestBody,
) *mcpRuntime.MCPApprovalSummary {
	arguments := []byte(`{}`)
	if request.Arguments != nil {
		encoded, err := json.Marshal(request.Arguments)
		if err == nil {
			arguments = encoded
		}
	}

	return &mcpRuntime.MCPApprovalSummary{
		Server:            config.Server,
		ServerDisplayName: config.DisplayName,
		Source:            request.Source,
		AppInstanceID:     request.AppInstanceID,
		ToolName:          tool.ToolName,
		ToolDigest:        tool.Digest,
		Risk:              tool.InferredRisk,
		Arguments:         jsonutil.JSONRawString(arguments),
	}
}

func (b *ToolBridge) applyCachedDecision(
	evaluation mcpRuntime.MCPApprovalEvaluation,
) mcpRuntime.MCPApprovalEvaluation {
	if b == nil ||
		b.approvals == nil ||
		evaluation.Summary == nil {
		return evaluation
	}

	cached, found := b.approvals.LookupDecision(*evaluation.Summary)
	if !found {
		return evaluation
	}

	switch cached {
	case mcpRuntime.MCPApprovalResolutionAllowAlways:
		// A remembered user approval may satisfy an approval prompt, but it
		// must never override a hard policy denial.
		if evaluation.Decision ==
			mcpRuntime.MCPApprovalDecisionApprovalRequired {
			evaluation.Decision = mcpRuntime.MCPApprovalDecisionAllowed
			evaluation.Reason = "remembered session approval"
		}

	case mcpRuntime.MCPApprovalResolutionDenyAlways:
		// A remembered denial also applies when the base policy would
		// otherwise permit the call.
		if evaluation.Decision != mcpRuntime.MCPApprovalDecisionDenied {
			evaluation.Decision = mcpRuntime.MCPApprovalDecisionDenied
			evaluation.Reason = "remembered session denial"
		}

	default:
	}
	return evaluation
}

func validateBridgeRequest(
	ctx context.Context,
	serverRef mcpSpec.ServerID,
	request mcpRuntime.InvokeMCPToolRequestBody,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: MCP tool invocation context is nil",
			mcpRuntime.ErrMCPInvalidRuntimeRequest,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := serverRef.Validate(); err != nil {
		return err
	}
	if err := validateMCPInvocationSource(request.Source); err != nil {
		return err
	}
	if request.Source == mcpRuntime.MCPInvocationSourceApp &&
		strings.TrimSpace(request.AppInstanceID) == "" {
		return fmt.Errorf(
			"%w: appInstanceID is required for app-initiated MCP calls",
			mcpRuntime.ErrMCPInvalidRuntimeRequest,
		)
	}
	if strings.TrimSpace(request.ToolName) == "" {
		return fmt.Errorf(
			"%w: MCP tool name is required",
			mcpRuntime.ErrMCPInvalidRuntimeRequest,
		)
	}
	return nil
}

// validateMCPInvocationSource validates the source of a tool invocation.
// The caller is still responsible for policy and approval enforcement.
func validateMCPInvocationSource(value mcpRuntime.MCPInvocationSource) error {
	switch value {
	case mcpRuntime.MCPInvocationSourceModel,
		mcpRuntime.MCPInvocationSourceUser,
		mcpRuntime.MCPInvocationSourceApp:
		return nil
	default:
		return fmt.Errorf(
			"%w: invalid MCP invocation source %q",
			mcpSpec.ErrInvalid,
			value,
		)
	}
}
