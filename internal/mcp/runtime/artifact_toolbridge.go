package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/mcp/apps"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/validate"
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
	return &ToolBridge{
		runtime:   runtime,
		approvals: approvals,
	}
}

func (b *ToolBridge) ResolveApproval(
	ctx context.Context,
	approvalID string,
	resolution spec.MCPApprovalResolution,
) (*spec.MCPApprovalToken, error) {
	if b == nil || b.approvals == nil {
		return nil, fmt.Errorf(
			"%w: MCP approval manager is unavailable",
			spec.ErrMCPRuntimeNotReady,
		)
	}
	return b.approvals.Resolve(ctx, approvalID, resolution)
}

func (b *ToolBridge) Evaluate(
	ctx context.Context,
	serverRef artifact.ArtifactRef,
	request spec.InvokeMCPToolRequestBody,
) (*spec.MCPApprovalEvaluation, error) {
	if err := validateBridgeRequest(ctx, serverRef, request); err != nil {
		return nil, err
	}
	if b == nil || b.runtime == nil || b.approvals == nil {
		return nil, fmt.Errorf("%w: MCP tool bridge is unavailable", spec.ErrMCPRuntimeNotReady)
	}

	config, tool, err := b.runtime.CallToolDryRun(
		ctx,
		serverRef,
		request,
	)
	if err != nil {
		return nil, err
	}

	if request.Source == spec.MCPInvocationSourceApp {
		if err := apps.ValidateArtifactAppToolInvocation(
			config.AppsPolicy,
			tool,
			serverRef,
		); err != nil {
			return nil, err
		}
	} else if config.AppsPolicy.Enabled &&
		!apps.ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			spec.ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := evaluateTool(config, tool, request)
	evaluation = b.applyCachedDecision(evaluation)

	if evaluation.Decision == spec.MCPApprovalDecisionApprovalRequired &&
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
	mapping spec.MCPProviderToolMapping,
	request spec.InvokeMCPToolRequestBody,
) (*spec.MCPApprovalEvaluation, error) {
	config, tool, normalized, err := b.mappedDryRun(
		ctx,
		mapping,
		request,
	)
	if err != nil {
		return nil, err
	}

	if normalized.Source == spec.MCPInvocationSourceApp {
		if err := apps.ValidateArtifactAppToolInvocation(
			config.AppsPolicy,
			tool,
			mapping.Server,
		); err != nil {
			return nil, err
		}
	} else if config.AppsPolicy.Enabled &&
		!apps.ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			spec.ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := b.applyCachedDecision(
		evaluateTool(config, tool, normalized),
	)
	if evaluation.Decision != spec.MCPApprovalDecisionApprovalRequired ||
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
	mapping spec.MCPProviderToolMapping,
	request spec.InvokeMCPToolRequestBody,
) (*spec.InvokeMCPToolResponseBody, error) {
	config, tool, normalized, err := b.mappedDryRun(
		ctx,
		mapping,
		request,
	)
	if err != nil {
		return nil, err
	}

	if normalized.Source == spec.MCPInvocationSourceApp {
		if err := apps.ValidateArtifactAppToolInvocation(
			config.AppsPolicy,
			tool,
			mapping.Server,
		); err != nil {
			return nil, err
		}
	} else if config.AppsPolicy.Enabled &&
		!apps.ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			spec.ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := b.applyCachedDecision(
		evaluateTool(config, tool, normalized),
	)
	switch evaluation.Decision {
	case spec.MCPApprovalDecisionDenied:
		if evaluation.Reason == toolDigestChangedReason {
			return nil, fmt.Errorf(
				"%w: %s",
				spec.ErrMCPStaleReference,
				evaluation.Reason,
			)
		}
		return nil, fmt.Errorf(
			"%w: %s",
			spec.ErrMCPPolicyDenied,
			evaluation.Reason,
		)

	case spec.MCPApprovalDecisionApprovalRequired:
		if normalized.ApprovalToken == "" || evaluation.Summary == nil {
			return nil, fmt.Errorf(
				"%w: approval token is required",
				spec.ErrMCPApprovalNeeded,
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

	case spec.MCPApprovalDecisionAllowed:
	default:
		return nil, fmt.Errorf(
			"%w: invalid MCP approval decision",
			spec.ErrMCPInvalidRequest,
		)
	}

	return b.runtime.CallTool(ctx, mapping.Server, normalized)
}

func (b *ToolBridge) Invoke(
	ctx context.Context,
	serverRef artifact.ArtifactRef,
	request spec.InvokeMCPToolRequestBody,
) (*spec.InvokeMCPToolResponseBody, error) {
	if err := validateBridgeRequest(ctx, serverRef, request); err != nil {
		return nil, err
	}
	if b == nil || b.runtime == nil || b.approvals == nil {
		return nil, fmt.Errorf("%w: MCP tool bridge is unavailable", spec.ErrMCPRuntimeNotReady)
	}

	config, tool, err := b.runtime.CallToolDryRun(
		ctx,
		serverRef,
		request,
	)
	if err != nil {
		return nil, err
	}

	if request.Source == spec.MCPInvocationSourceApp {
		if err := apps.ValidateArtifactAppToolInvocation(
			config.AppsPolicy,
			tool,
			serverRef,
		); err != nil {
			return nil, err
		}
	} else if config.AppsPolicy.Enabled &&
		!apps.ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			spec.ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := evaluateTool(config, tool, request)
	evaluation = b.applyCachedDecision(evaluation)

	switch evaluation.Decision {
	case spec.MCPApprovalDecisionDenied:
		if evaluation.Reason == toolDigestChangedReason {
			return nil, fmt.Errorf(
				"%w: %s",
				spec.ErrMCPStaleReference,
				evaluation.Reason,
			)
		}
		return nil, fmt.Errorf(
			"%w: %s",
			spec.ErrMCPPolicyDenied,
			evaluation.Reason,
		)

	case spec.MCPApprovalDecisionApprovalRequired:
		if request.ApprovalToken == "" || evaluation.Summary == nil {
			return nil, fmt.Errorf(
				"%w: approval token is required",
				spec.ErrMCPApprovalNeeded,
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

	case spec.MCPApprovalDecisionAllowed:
	default:
		return nil, fmt.Errorf(
			"%w: invalid MCP approval decision",
			spec.ErrMCPInvalidRequest,
		)
	}

	return b.runtime.CallTool(ctx, serverRef, request)
}

func (b *ToolBridge) mappedDryRun(
	ctx context.Context,
	mapping spec.MCPProviderToolMapping,
	request spec.InvokeMCPToolRequestBody,
) (
	config server.RuntimeConfig,
	tool spec.MCPToolCapability,
	normalized spec.InvokeMCPToolRequestBody,
	err error,
) {
	if b == nil || b.runtime == nil || b.approvals == nil {
		return server.RuntimeConfig{},
			spec.MCPToolCapability{},
			spec.InvokeMCPToolRequestBody{},
			fmt.Errorf(
				"%w: MCP tool bridge is unavailable",
				spec.ErrMCPRuntimeNotReady,
			)
	}

	normalized, err = normalizeMappedToolRequest(mapping, request)
	if err != nil {
		return server.RuntimeConfig{},
			spec.MCPToolCapability{},
			spec.InvokeMCPToolRequestBody{},
			err
	}
	if err := validateBridgeRequest(ctx, mapping.Server, normalized); err != nil {
		return server.RuntimeConfig{},
			spec.MCPToolCapability{},
			spec.InvokeMCPToolRequestBody{},
			err
	}

	config, tool, err = b.runtime.CallToolDryRun(
		ctx,
		mapping.Server,
		normalized,
	)
	if err != nil {
		return server.RuntimeConfig{},
			spec.MCPToolCapability{},
			spec.InvokeMCPToolRequestBody{},
			err
	}
	if tool.ProviderToolName != mapping.ProviderToolName ||
		tool.ChoiceID != mapping.ChoiceID ||
		tool.ToolName != mapping.ToolName ||
		tool.Digest != mapping.ToolDigest {
		return server.RuntimeConfig{},
			spec.MCPToolCapability{},
			spec.InvokeMCPToolRequestBody{},
			fmt.Errorf(
				"%w: MCP provider tool mapping is stale",
				spec.ErrMCPStaleReference,
			)
	}

	config, err = applyMappedPolicyConstraints(
		config,
		tool,
		mapping,
	)
	if err != nil {
		return server.RuntimeConfig{},
			spec.MCPToolCapability{},
			spec.InvokeMCPToolRequestBody{},
			err
	}
	return config, tool, normalized, nil
}

func normalizeMappedToolRequest(
	mapping spec.MCPProviderToolMapping,
	request spec.InvokeMCPToolRequestBody,
) (spec.InvokeMCPToolRequestBody, error) {
	if err := validate.ValidateMCPProviderToolMapping(mapping); err != nil {
		return spec.InvokeMCPToolRequestBody{}, err
	}

	request.ToolName = strings.TrimSpace(request.ToolName)
	request.ProviderToolName = strings.TrimSpace(
		request.ProviderToolName,
	)

	switch request.ToolName {
	case "":
		if request.ProviderToolName != mapping.ProviderToolName {
			return spec.InvokeMCPToolRequestBody{}, fmt.Errorf(
				"%w: provider tool name does not match the persisted mapping",
				spec.ErrMCPInvalidRequest,
			)
		}
		request.ToolName = mapping.ToolName

	case mapping.ProviderToolName:
		request.ToolName = mapping.ToolName

	case mapping.ToolName:
	default:
		return spec.InvokeMCPToolRequestBody{}, fmt.Errorf(
			"%w: tool name does not match the persisted mapping",
			spec.ErrMCPInvalidRequest,
		)
	}

	if request.ProviderToolName != "" &&
		request.ProviderToolName != mapping.ProviderToolName {
		return spec.InvokeMCPToolRequestBody{}, fmt.Errorf(
			"%w: provider tool name does not match the persisted mapping",
			spec.ErrMCPInvalidRequest,
		)
	}

	if request.ToolDigest != "" &&
		request.ToolDigest != mapping.ToolDigest {
		return spec.InvokeMCPToolRequestBody{}, fmt.Errorf(
			"%w: provider tool digest does not match the persisted mapping",
			spec.ErrMCPStaleReference,
		)
	}

	request.ProviderToolName = mapping.ProviderToolName
	request.ToolDigest = mapping.ToolDigest
	return request, nil
}

func applyMappedPolicyConstraints(
	config server.RuntimeConfig,
	tool spec.MCPToolCapability,
	mapping spec.MCPProviderToolMapping,
) (server.RuntimeConfig, error) {
	currentApproval, currentExecution := currentToolConstraints(
		config,
		tool,
	)
	if approvalRuleRank(mapping.ApprovalRule) <
		approvalRuleRank(currentApproval) {
		return server.RuntimeConfig{}, fmt.Errorf(
			"%w: mapped approval rule weakens current MCP policy",
			spec.ErrMCPPolicyDenied,
		)
	}
	if executionModeRank(mapping.ExecutionMode) <
		executionModeRank(currentExecution) {
		return server.RuntimeConfig{}, fmt.Errorf(
			"%w: mapped execution mode weakens current MCP policy",
			spec.ErrMCPPolicyDenied,
		)
	}

	output := config
	output.ToolPolicies = maps.Clone(config.ToolPolicies)
	if output.ToolPolicies == nil {
		output.ToolPolicies = map[string]spec.MCPToolPolicyOverride{}
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
	tool spec.MCPToolCapability,
) (spec.MCPApprovalRule, spec.MCPExecutionMode) {
	defaults := spec.DefaultMCPServerPolicy()
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

func approvalRuleRank(value spec.MCPApprovalRule) int {
	switch value {
	case spec.MCPApprovalRuleDeny:
		return 3
	case spec.MCPApprovalRuleAsk:
		return 2
	default:
		return 1
	}
}

func executionModeRank(value spec.MCPExecutionMode) int {
	if value == spec.MCPExecutionModeManual {
		return 2
	}
	return 1
}

func normalizedApprovalRule(
	value spec.MCPApprovalRule,
) spec.MCPApprovalRule {
	if value == "" {
		return spec.MCPApprovalRuleAsk
	}
	return value
}

func normalizedExecutionMode(
	value spec.MCPExecutionMode,
) spec.MCPExecutionMode {
	if value == "" {
		return spec.MCPExecutionModeManual
	}
	return value
}

func evaluateTool(
	config server.RuntimeConfig,
	tool spec.MCPToolCapability,
	request spec.InvokeMCPToolRequestBody,
) spec.MCPApprovalEvaluation {
	policy := config.DefaultPolicy
	if policy == (spec.MCPServerPolicy{}) {
		policy = spec.DefaultMCPServerPolicy()
	}

	if !tool.Enabled ||
		tool.TaskSupport == spec.MCPTaskSupportRequired {
		return deniedEvaluation(
			config,
			tool,
			request,
			"tool is disabled or unsupported",
		)
	}

	rule := policy.DefaultApprovalRule
	executionMode := policy.DefaultExecutionMode
	if executionMode == "" {
		executionMode = spec.MCPExecutionModeManual
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

	if rule == spec.MCPApprovalRuleDeny {
		return deniedEvaluation(
			config,
			tool,
			request,
			toolPolicyDeniesReason,
		)
	}
	if executionMode == spec.MCPExecutionModeManual &&
		request.Source != spec.MCPInvocationSourceUser {
		return approvalRequiredEvaluation(
			config,
			tool,
			request,
			"manual execution mode requires approval",
		)
	}
	if rule == spec.MCPApprovalRuleAsk {
		return approvalRequiredEvaluation(
			config,
			tool,
			request,
			"approval rule is ask",
		)
	}

	switch tool.InferredRisk {
	case spec.MCPToolRiskUnknown:
		if policy.RequireApprovalForUnknownRisk {
			return approvalRequiredEvaluation(
				config,
				tool,
				request,
				"unknown-risk tool requires approval",
			)
		}
	case spec.MCPToolRiskOpenWorld:
		if policy.RequireApprovalForUnknownRisk {
			return approvalRequiredEvaluation(
				config,
				tool,
				request,
				"open-world tool requires approval",
			)
		}
	case spec.MCPToolRiskWrite:
		if policy.RequireApprovalForWrite {
			return approvalRequiredEvaluation(
				config,
				tool,
				request,
				"write-risk tool requires approval",
			)
		}
	case spec.MCPToolRiskDestructive:
		if policy.RequireApprovalForDestructive {
			return approvalRequiredEvaluation(
				config,
				tool,
				request,
				"destructive-risk tool requires approval",
			)
		}
	default:
	}

	return spec.MCPApprovalEvaluation{
		Decision: spec.MCPApprovalDecisionAllowed,
		Reason:   policyAllowedReason,
		Summary:  approvalSummary(config, tool, request),
	}
}

func deniedEvaluation(
	config server.RuntimeConfig,
	tool spec.MCPToolCapability,
	request spec.InvokeMCPToolRequestBody,
	reason string,
) spec.MCPApprovalEvaluation {
	return spec.MCPApprovalEvaluation{
		Decision: spec.MCPApprovalDecisionDenied,
		Reason:   reason,
		Summary:  approvalSummary(config, tool, request),
	}
}

func approvalRequiredEvaluation(
	config server.RuntimeConfig,
	tool spec.MCPToolCapability,
	request spec.InvokeMCPToolRequestBody,
	reason string,
) spec.MCPApprovalEvaluation {
	return spec.MCPApprovalEvaluation{
		Decision: spec.MCPApprovalDecisionApprovalRequired,
		Reason:   reason,
		Summary:  approvalSummary(config, tool, request),
	}
}

func approvalSummary(
	config server.RuntimeConfig,
	tool spec.MCPToolCapability,
	request spec.InvokeMCPToolRequestBody,
) *spec.MCPApprovalSummary {
	arguments := []byte(`{}`)
	if request.Arguments != nil {
		encoded, err := json.Marshal(request.Arguments)
		if err == nil {
			arguments = encoded
		}
	}

	return &spec.MCPApprovalSummary{
		Server:            config.Server,
		ServerDisplayName: config.DisplayName,
		ToolName:          tool.ToolName,
		ToolDigest:        tool.Digest,
		Risk:              tool.InferredRisk,
		Arguments:         spec.JSONRawString(arguments),
	}
}

func (b *ToolBridge) applyCachedDecision(
	evaluation spec.MCPApprovalEvaluation,
) spec.MCPApprovalEvaluation {
	if b == nil ||
		b.approvals == nil ||
		evaluation.Decision != spec.MCPApprovalDecisionApprovalRequired ||
		evaluation.Summary == nil {
		return evaluation
	}

	cached, found := b.approvals.LookupDecision(*evaluation.Summary)
	if !found {
		return evaluation
	}

	switch cached {
	case spec.MCPApprovalResolutionAllowAlways:
		evaluation.Decision = spec.MCPApprovalDecisionAllowed
		evaluation.Reason = "cached allow-always decision"
	case spec.MCPApprovalResolutionDenyAlways:
		evaluation.Decision = spec.MCPApprovalDecisionDenied
		evaluation.Reason = "cached deny-always decision"
	default:
	}
	return evaluation
}

func validateBridgeRequest(
	ctx context.Context,
	serverRef artifact.ArtifactRef,
	request spec.InvokeMCPToolRequestBody,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: MCP tool invocation context is nil",
			spec.ErrMCPInvalidRequest,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := serverRef.Validate(); err != nil {
		return err
	}
	if err := validate.ValidateMCPInvocationSource(request.Source); err != nil {
		return err
	}
	if request.Source == spec.MCPInvocationSourceApp &&
		strings.TrimSpace(request.AppInstanceID) == "" {
		return fmt.Errorf(
			"%w: appInstanceID is required for app-initiated MCP calls",
			spec.ErrMCPInvalidRequest,
		)
	}
	if strings.TrimSpace(request.ToolName) == "" {
		return fmt.Errorf(
			"%w: MCP tool name is required",
			spec.ErrMCPInvalidRequest,
		)
	}
	return nil
}
