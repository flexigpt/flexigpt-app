package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
)

const (
	toolDigestChangedReason = "tool digest changed"
	toolPolicyDeniesReason  = "server/tool policy denies this tool"
	policyAllowedReason     = "policy allowed"
)

type ToolBridge struct {
	runtime   *MCPRuntimeManager
	approvals *ApprovalManager
}

func NewToolBridge(
	runtime *MCPRuntimeManager,
	approvals *ApprovalManager,
) *ToolBridge {
	if runtime != nil {
		runtime.bindApprovalManager(approvals)
	}
	return &ToolBridge{
		runtime:   runtime,
		approvals: approvals,
	}
}

func (b *ToolBridge) ResolveApproval(
	ctx context.Context,
	approvalID string,
	resolution MCPApprovalResolution,
) (MCPApprovalResolutionResult, error) {
	if b == nil || b.approvals == nil {
		return MCPApprovalResolutionResult{}, fmt.Errorf(
			"%w: MCP approval manager is unavailable",
			ErrMCPRuntimeNotReady,
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
	serverRef artifact.ArtifactRef,
	request InvokeMCPToolRequestBody,
) (*MCPApprovalEvaluation, error) {
	if err := validateBridgeRequest(ctx, serverRef, request); err != nil {
		return nil, err
	}
	if b == nil || b.runtime == nil || b.approvals == nil {
		return nil, fmt.Errorf("%w: MCP tool bridge is unavailable", ErrMCPRuntimeNotReady)
	}

	config, tool, err := b.runtime.CallToolDryRun(
		ctx,
		serverRef,
		request,
	)
	if err != nil {
		return nil, err
	}

	if request.Source == MCPInvocationSourceApp {
		if err := ValidateArtifactAppToolInvocation(
			config.AppsPolicy,
			tool,
			serverRef,
		); err != nil {
			return nil, err
		}
	} else if config.AppsPolicy.Enabled &&
		!ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := evaluateTool(config, tool, request)
	evaluation = b.applyCachedDecision(evaluation)

	if evaluation.Decision == MCPApprovalDecisionApprovalRequired &&
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
	mapping MCPProviderToolMapping,
	request InvokeMCPToolRequestBody,
) (*MCPApprovalEvaluation, error) {
	config, tool, normalized, err := b.mappedDryRun(
		ctx,
		mapping,
		request,
	)
	if err != nil {
		return nil, err
	}

	if normalized.Source == MCPInvocationSourceApp {
		if err := ValidateArtifactAppToolInvocation(
			config.AppsPolicy,
			tool,
			mapping.Server,
		); err != nil {
			return nil, err
		}
	} else if config.AppsPolicy.Enabled &&
		!ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := b.applyCachedDecision(
		evaluateTool(config, tool, normalized),
	)
	if evaluation.Decision != MCPApprovalDecisionApprovalRequired ||
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
	mapping MCPProviderToolMapping,
	request InvokeMCPToolRequestBody,
) (*InvokeMCPToolResponseBody, error) {
	config, tool, normalized, err := b.mappedDryRun(
		ctx,
		mapping,
		request,
	)
	if err != nil {
		return nil, err
	}

	if normalized.Source == MCPInvocationSourceApp {
		if err := ValidateArtifactAppToolInvocation(
			config.AppsPolicy,
			tool,
			mapping.Server,
		); err != nil {
			return nil, err
		}
	} else if config.AppsPolicy.Enabled &&
		!ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := b.applyCachedDecision(
		evaluateTool(config, tool, normalized),
	)
	switch evaluation.Decision {
	case MCPApprovalDecisionDenied:
		if evaluation.Reason == toolDigestChangedReason {
			return nil, fmt.Errorf(
				"%w: %s",
				ErrMCPStaleReference,
				evaluation.Reason,
			)
		}
		return nil, fmt.Errorf(
			"%w: %s",
			ErrMCPPolicyDenied,
			evaluation.Reason,
		)

	case MCPApprovalDecisionApprovalRequired:
		if normalized.ApprovalToken == "" || evaluation.Summary == nil {
			return nil, fmt.Errorf(
				"%w: approval token is required",
				ErrMCPApprovalNeeded,
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

	case MCPApprovalDecisionAllowed:
	default:
		return nil, fmt.Errorf(
			"%w: invalid MCP approval decision",
			ErrMCPInvalidRuntimeRequest,
		)
	}

	return b.runtime.CallTool(ctx, mapping.Server, normalized)
}

func (b *ToolBridge) Invoke(
	ctx context.Context,
	serverRef artifact.ArtifactRef,
	request InvokeMCPToolRequestBody,
) (*InvokeMCPToolResponseBody, error) {
	if err := validateBridgeRequest(ctx, serverRef, request); err != nil {
		return nil, err
	}
	if b == nil || b.runtime == nil || b.approvals == nil {
		return nil, fmt.Errorf("%w: MCP tool bridge is unavailable", ErrMCPRuntimeNotReady)
	}

	config, tool, err := b.runtime.CallToolDryRun(
		ctx,
		serverRef,
		request,
	)
	if err != nil {
		return nil, err
	}

	if request.Source == MCPInvocationSourceApp {
		if err := ValidateArtifactAppToolInvocation(
			config.AppsPolicy,
			tool,
			serverRef,
		); err != nil {
			return nil, err
		}
	} else if config.AppsPolicy.Enabled &&
		!ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := evaluateTool(config, tool, request)
	evaluation = b.applyCachedDecision(evaluation)

	switch evaluation.Decision {
	case MCPApprovalDecisionDenied:
		if evaluation.Reason == toolDigestChangedReason {
			return nil, fmt.Errorf(
				"%w: %s",
				ErrMCPStaleReference,
				evaluation.Reason,
			)
		}
		return nil, fmt.Errorf(
			"%w: %s",
			ErrMCPPolicyDenied,
			evaluation.Reason,
		)

	case MCPApprovalDecisionApprovalRequired:
		if request.ApprovalToken == "" || evaluation.Summary == nil {
			return nil, fmt.Errorf(
				"%w: approval token is required",
				ErrMCPApprovalNeeded,
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

	case MCPApprovalDecisionAllowed:
	default:
		return nil, fmt.Errorf(
			"%w: invalid MCP approval decision",
			ErrMCPInvalidRuntimeRequest,
		)
	}

	return b.runtime.CallTool(ctx, serverRef, request)
}

func (b *ToolBridge) mappedDryRun(
	ctx context.Context,
	mapping MCPProviderToolMapping,
	request InvokeMCPToolRequestBody,
) (
	config server.RuntimeConfig,
	tool MCPToolCapability,
	normalized InvokeMCPToolRequestBody,
	err error,
) {
	if b == nil || b.runtime == nil || b.approvals == nil {
		return server.RuntimeConfig{},
			MCPToolCapability{},
			InvokeMCPToolRequestBody{},
			fmt.Errorf(
				"%w: MCP tool bridge is unavailable",
				ErrMCPRuntimeNotReady,
			)
	}

	normalized, err = normalizeMappedToolRequest(mapping, request)
	if err != nil {
		return server.RuntimeConfig{},
			MCPToolCapability{},
			InvokeMCPToolRequestBody{},
			err
	}
	if err := validateBridgeRequest(ctx, mapping.Server, normalized); err != nil {
		return server.RuntimeConfig{},
			MCPToolCapability{},
			InvokeMCPToolRequestBody{},
			err
	}

	config, tool, err = b.runtime.CallToolDryRun(
		ctx,
		mapping.Server,
		normalized,
	)
	if err != nil {
		return server.RuntimeConfig{},
			MCPToolCapability{},
			InvokeMCPToolRequestBody{},
			err
	}
	if tool.ProviderToolName != mapping.ProviderToolName ||
		tool.ChoiceID != mapping.ChoiceID ||
		tool.ToolName != mapping.ToolName ||
		tool.Digest != mapping.ToolDigest {
		return server.RuntimeConfig{},
			MCPToolCapability{},
			InvokeMCPToolRequestBody{},
			fmt.Errorf(
				"%w: MCP provider tool mapping is stale",
				ErrMCPStaleReference,
			)
	}

	config, err = applyMappedPolicyConstraints(
		config,
		tool,
		mapping,
	)
	if err != nil {
		return server.RuntimeConfig{},
			MCPToolCapability{},
			InvokeMCPToolRequestBody{},
			err
	}
	return config, tool, normalized, nil
}

func normalizeMappedToolRequest(
	mapping MCPProviderToolMapping,
	request InvokeMCPToolRequestBody,
) (InvokeMCPToolRequestBody, error) {
	if err := ValidateMCPProviderToolMapping(mapping); err != nil {
		return InvokeMCPToolRequestBody{}, err
	}

	request.ToolName = strings.TrimSpace(request.ToolName)
	request.ProviderToolName = strings.TrimSpace(
		request.ProviderToolName,
	)

	switch request.ToolName {
	case "":
		if request.ProviderToolName != mapping.ProviderToolName {
			return InvokeMCPToolRequestBody{}, fmt.Errorf(
				"%w: provider tool name does not match the persisted mapping",
				ErrMCPInvalidRuntimeRequest,
			)
		}
		request.ToolName = mapping.ToolName

	case mapping.ProviderToolName:
		request.ToolName = mapping.ToolName

	case mapping.ToolName:
	default:
		return InvokeMCPToolRequestBody{}, fmt.Errorf(
			"%w: tool name does not match the persisted mapping",
			ErrMCPInvalidRuntimeRequest,
		)
	}

	if request.ProviderToolName != "" &&
		request.ProviderToolName != mapping.ProviderToolName {
		return InvokeMCPToolRequestBody{}, fmt.Errorf(
			"%w: provider tool name does not match the persisted mapping",
			ErrMCPInvalidRuntimeRequest,
		)
	}

	if request.ToolDigest != "" &&
		request.ToolDigest != mapping.ToolDigest {
		return InvokeMCPToolRequestBody{}, fmt.Errorf(
			"%w: provider tool digest does not match the persisted mapping",
			ErrMCPStaleReference,
		)
	}

	request.ProviderToolName = mapping.ProviderToolName
	request.ToolDigest = mapping.ToolDigest
	return request, nil
}

func applyMappedPolicyConstraints(
	config server.RuntimeConfig,
	tool MCPToolCapability,
	mapping MCPProviderToolMapping,
) (server.RuntimeConfig, error) {
	currentApproval, currentExecution := currentToolConstraints(
		config,
		tool,
	)
	if approvalRuleRank(mapping.ApprovalRule) <
		approvalRuleRank(currentApproval) {
		return server.RuntimeConfig{}, fmt.Errorf(
			"%w: mapped approval rule weakens current MCP policy",
			ErrMCPPolicyDenied,
		)
	}
	if executionModeRank(mapping.ExecutionMode) <
		executionModeRank(currentExecution) {
		return server.RuntimeConfig{}, fmt.Errorf(
			"%w: mapped execution mode weakens current MCP policy",
			ErrMCPPolicyDenied,
		)
	}

	output := config
	output.ToolPolicies = maps.Clone(config.ToolPolicies)
	if output.ToolPolicies == nil {
		output.ToolPolicies = map[string]policy.MCPToolPolicyOverride{}
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
	config server.RuntimeConfig,
	tool MCPToolCapability,
) (policy.MCPApprovalRule, policy.MCPExecutionMode) {
	defaults := policy.DefaultMCPServerPolicy()
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

	toolApproval := normalizedApprovalRule(tool.ApprovalRule)
	if approvalRuleRank(toolApproval) > approvalRuleRank(approval) {
		approval = toolApproval
	}
	toolExecution := normalizedExecutionMode(tool.ExecutionMode)
	if executionModeRank(toolExecution) > executionModeRank(execution) {
		execution = toolExecution
	}
	return approval, execution
}

func approvalRuleRank(value policy.MCPApprovalRule) int {
	switch value {
	case policy.MCPApprovalRuleDeny:
		return 3
	case policy.MCPApprovalRuleAsk:
		return 2
	default:
		return 1
	}
}

func executionModeRank(value policy.MCPExecutionMode) int {
	if value == policy.MCPExecutionModeManual {
		return 2
	}
	return 1
}

func normalizedApprovalRule(
	value policy.MCPApprovalRule,
) policy.MCPApprovalRule {
	if value == "" {
		return policy.MCPApprovalRuleAsk
	}
	return value
}

func normalizedExecutionMode(
	value policy.MCPExecutionMode,
) policy.MCPExecutionMode {
	if value == "" {
		return policy.MCPExecutionModeManual
	}
	return value
}

func evaluateTool(
	config server.RuntimeConfig,
	tool MCPToolCapability,
	request InvokeMCPToolRequestBody,
) MCPApprovalEvaluation {
	p := config.DefaultPolicy
	if p == (policy.MCPServerPolicy{}) {
		p = policy.DefaultMCPServerPolicy()
	}

	if !tool.Enabled ||
		tool.TaskSupport == MCPTaskSupportRequired {
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
		executionMode = policy.MCPExecutionModeManual
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

	if rule == policy.MCPApprovalRuleDeny {
		return deniedEvaluation(
			config,
			tool,
			request,
			toolPolicyDeniesReason,
		)
	}
	if executionMode == policy.MCPExecutionModeManual &&
		request.Source != MCPInvocationSourceUser {
		return approvalRequiredEvaluation(
			config,
			tool,
			request,
			"manual execution mode requires approval",
		)
	}
	if rule == policy.MCPApprovalRuleAsk {
		return approvalRequiredEvaluation(
			config,
			tool,
			request,
			"approval rule is ask",
		)
	}

	switch tool.InferredRisk {
	case MCPToolRiskUnknown:
		if p.RequireApprovalForUnknownRisk {
			return approvalRequiredEvaluation(
				config,
				tool,
				request,
				"unknown-risk tool requires approval",
			)
		}
	case MCPToolRiskOpenWorld:
		if p.RequireApprovalForUnknownRisk {
			return approvalRequiredEvaluation(
				config,
				tool,
				request,
				"open-world tool requires approval",
			)
		}
	case MCPToolRiskWrite:
		if p.RequireApprovalForWrite {
			return approvalRequiredEvaluation(
				config,
				tool,
				request,
				"write-risk tool requires approval",
			)
		}
	case MCPToolRiskDestructive:
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

	return MCPApprovalEvaluation{
		Decision: MCPApprovalDecisionAllowed,
		Reason:   policyAllowedReason,
		Summary:  approvalSummary(config, tool, request),
	}
}

func deniedEvaluation(
	config server.RuntimeConfig,
	tool MCPToolCapability,
	request InvokeMCPToolRequestBody,
	reason string,
) MCPApprovalEvaluation {
	return MCPApprovalEvaluation{
		Decision: MCPApprovalDecisionDenied,
		Reason:   reason,
		Summary:  approvalSummary(config, tool, request),
	}
}

func approvalRequiredEvaluation(
	config server.RuntimeConfig,
	tool MCPToolCapability,
	request InvokeMCPToolRequestBody,
	reason string,
) MCPApprovalEvaluation {
	return MCPApprovalEvaluation{
		Decision: MCPApprovalDecisionApprovalRequired,
		Reason:   reason,
		Summary:  approvalSummary(config, tool, request),
	}
}

func approvalSummary(
	config server.RuntimeConfig,
	tool MCPToolCapability,
	request InvokeMCPToolRequestBody,
) *MCPApprovalSummary {
	arguments := []byte(`{}`)
	if request.Arguments != nil {
		encoded, err := json.Marshal(request.Arguments)
		if err == nil {
			arguments = encoded
		}
	}

	return &MCPApprovalSummary{
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
	evaluation MCPApprovalEvaluation,
) MCPApprovalEvaluation {
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
	case MCPApprovalResolutionAllowAlways:
		// A remembered user approval may satisfy an approval prompt, but it
		// must never override a hard policy denial.
		if evaluation.Decision ==
			MCPApprovalDecisionApprovalRequired {
			evaluation.Decision = MCPApprovalDecisionAllowed
			evaluation.Reason = "remembered session approval"
		}

	case MCPApprovalResolutionDenyAlways:
		// A remembered denial also applies when the base policy would
		// otherwise permit the call.
		if evaluation.Decision != MCPApprovalDecisionDenied {
			evaluation.Decision = MCPApprovalDecisionDenied
			evaluation.Reason = "remembered session denial"
		}

	default:
	}
	return evaluation
}

func validateBridgeRequest(
	ctx context.Context,
	serverRef artifact.ArtifactRef,
	request InvokeMCPToolRequestBody,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: MCP tool invocation context is nil",
			ErrMCPInvalidRuntimeRequest,
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
	if request.Source == MCPInvocationSourceApp &&
		strings.TrimSpace(request.AppInstanceID) == "" {
		return fmt.Errorf(
			"%w: appInstanceID is required for app-initiated MCP calls",
			ErrMCPInvalidRuntimeRequest,
		)
	}
	if strings.TrimSpace(request.ToolName) == "" {
		return fmt.Errorf(
			"%w: MCP tool name is required",
			ErrMCPInvalidRuntimeRequest,
		)
	}
	return nil
}

// validateMCPInvocationSource validates the source of a tool invocation.
// The caller is still responsible for policy and approval enforcement.
func validateMCPInvocationSource(value MCPInvocationSource) error {
	switch value {
	case MCPInvocationSourceModel,
		MCPInvocationSourceUser,
		MCPInvocationSourceApp:
		return nil
	default:
		return fmt.Errorf(
			"%w: invalid MCP invocation source %q",
			basespec.ErrInvalid,
			value,
		)
	}
}
