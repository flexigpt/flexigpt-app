package invocation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpConversation "github.com/flexigpt/flexigpt-app/internal/mcp/conversation"
	mcpApps "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/apps"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/connection"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
)

const (
	toolDigestChangedReason = mcpPolicy.ToolDigestChangedReason
	toolPolicyDeniesReason  = "server/tool policy denies this tool"
	policyAllowedReason     = "policy allowed"
)

type ToolBridge struct {
	runtime   *mcpConnection.MCPRuntimeManager
	approvals *ApprovalManager
}

func NewToolBridge(
	runtime *mcpConnection.MCPRuntimeManager,
	approvals *ApprovalManager,
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
	resolution mcpServer.MCPApprovalResolution,
) (mcpServer.MCPApprovalResolutionResult, error) {
	if b == nil || b.approvals == nil {
		return mcpServer.MCPApprovalResolutionResult{}, fmt.Errorf(
			"%w: MCP approval manager is unavailable",
			mcpServer.ErrMCPRuntimeNotReady,
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
	serverRef mcpServer.ServerID,
	request mcpServer.InvokeMCPToolRequestBody,
) (*mcpServer.MCPApprovalEvaluation, error) {
	if err := validateBridgeRequest(ctx, serverRef, request); err != nil {
		return nil, err
	}
	if b == nil || b.runtime == nil || b.approvals == nil {
		return nil, fmt.Errorf("%w: MCP tool bridge is unavailable", mcpServer.ErrMCPRuntimeNotReady)
	}

	config, tool, err := b.runtime.CallToolDryRun(
		ctx,
		serverRef,
		request,
	)
	if err != nil {
		return nil, err
	}

	if request.Source == mcpServer.MCPInvocationSourceApp {
		if err := mcpApps.ValidateAppToolInvocation(
			config.Policy.AppsPolicy,
			tool,
			serverRef,
		); err != nil {
			return nil, err
		}
	} else if config.Policy.AppsPolicy.Enabled &&
		!mcpApps.ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			mcpServer.ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := evaluateTool(config, tool, request)
	evaluation = b.applyCachedDecision(evaluation)

	if evaluation.Decision == mcpServer.MCPApprovalDecisionApprovalRequired &&
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
	mapping mcpConversation.MCPProviderToolMapping,
	request mcpServer.InvokeMCPToolRequestBody,
) (*mcpServer.MCPApprovalEvaluation, error) {
	config, tool, normalized, err := b.mappedDryRun(
		ctx,
		mapping,
		request,
	)
	if err != nil {
		return nil, err
	}

	if normalized.Source == mcpServer.MCPInvocationSourceApp {
		if err := mcpApps.ValidateAppToolInvocation(
			config.Policy.AppsPolicy,
			tool,
			mapping.Server,
		); err != nil {
			return nil, err
		}
	} else if config.Policy.AppsPolicy.Enabled &&
		!mcpApps.ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			mcpServer.ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := b.applyCachedDecision(
		evaluateTool(config, tool, normalized),
	)
	if evaluation.Decision != mcpServer.MCPApprovalDecisionApprovalRequired ||
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
	mapping mcpConversation.MCPProviderToolMapping,
	request mcpServer.InvokeMCPToolRequestBody,
) (*mcpServer.InvokeMCPToolResponseBody, error) {
	config, tool, normalized, err := b.mappedDryRun(
		ctx,
		mapping,
		request,
	)
	if err != nil {
		return nil, err
	}

	if normalized.Source == mcpServer.MCPInvocationSourceApp {
		if err := mcpApps.ValidateAppToolInvocation(
			config.Policy.AppsPolicy,
			tool,
			mapping.Server,
		); err != nil {
			return nil, err
		}
	} else if config.Policy.AppsPolicy.Enabled &&
		!mcpApps.ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			mcpServer.ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := b.applyCachedDecision(
		evaluateTool(config, tool, normalized),
	)
	switch evaluation.Decision {
	case mcpServer.MCPApprovalDecisionDenied:
		if evaluation.Reason == toolDigestChangedReason {
			return nil, fmt.Errorf(
				"%w: %s",
				mcpServer.ErrMCPStaleReference,
				evaluation.Reason,
			)
		}
		return nil, fmt.Errorf(
			"%w: %s",
			mcpServer.ErrMCPPolicyDenied,
			evaluation.Reason,
		)

	case mcpServer.MCPApprovalDecisionApprovalRequired:
		if normalized.ApprovalToken == "" || evaluation.Summary == nil {
			return nil, fmt.Errorf(
				"%w: approval token is required",
				mcpServer.ErrMCPApprovalNeeded,
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

	case mcpServer.MCPApprovalDecisionAllowed:
	default:
		return nil, fmt.Errorf(
			"%w: invalid MCP approval decision",
			mcpServer.ErrMCPInvalidRuntimeRequest,
		)
	}

	return b.runtime.CallTool(ctx, mapping.Server, normalized)
}

func (b *ToolBridge) Invoke(
	ctx context.Context,
	serverRef mcpServer.ServerID,
	request mcpServer.InvokeMCPToolRequestBody,
) (*mcpServer.InvokeMCPToolResponseBody, error) {
	if err := validateBridgeRequest(ctx, serverRef, request); err != nil {
		return nil, err
	}
	if b == nil || b.runtime == nil || b.approvals == nil {
		return nil, fmt.Errorf("%w: MCP tool bridge is unavailable", mcpServer.ErrMCPRuntimeNotReady)
	}

	config, tool, err := b.runtime.CallToolDryRun(
		ctx,
		serverRef,
		request,
	)
	if err != nil {
		return nil, err
	}

	if request.Source == mcpServer.MCPInvocationSourceApp {
		if err := mcpApps.ValidateAppToolInvocation(
			config.Policy.AppsPolicy,
			tool,
			serverRef,
		); err != nil {
			return nil, err
		}
	} else if config.Policy.AppsPolicy.Enabled &&
		!mcpApps.ToolVisibleToModel(tool.App) {
		return nil, fmt.Errorf(
			"%w: MCP tool %q is not visible to the model",
			mcpServer.ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}

	evaluation := evaluateTool(config, tool, request)
	evaluation = b.applyCachedDecision(evaluation)

	switch evaluation.Decision {
	case mcpServer.MCPApprovalDecisionDenied:
		if evaluation.Reason == toolDigestChangedReason {
			return nil, fmt.Errorf(
				"%w: %s",
				mcpServer.ErrMCPStaleReference,
				evaluation.Reason,
			)
		}
		return nil, fmt.Errorf(
			"%w: %s",
			mcpServer.ErrMCPPolicyDenied,
			evaluation.Reason,
		)

	case mcpServer.MCPApprovalDecisionApprovalRequired:
		if request.ApprovalToken == "" || evaluation.Summary == nil {
			return nil, fmt.Errorf(
				"%w: approval token is required",
				mcpServer.ErrMCPApprovalNeeded,
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

	case mcpServer.MCPApprovalDecisionAllowed:
	default:
		return nil, fmt.Errorf(
			"%w: invalid MCP approval decision",
			mcpServer.ErrMCPInvalidRuntimeRequest,
		)
	}

	return b.runtime.CallTool(ctx, serverRef, request)
}

func (b *ToolBridge) mappedDryRun(
	ctx context.Context,
	mapping mcpConversation.MCPProviderToolMapping,
	request mcpServer.InvokeMCPToolRequestBody,
) (
	config mcpServer.RuntimeConfig,
	tool mcpServer.MCPToolCapability,
	normalized mcpServer.InvokeMCPToolRequestBody,
	err error,
) {
	if b == nil || b.runtime == nil || b.approvals == nil {
		return mcpServer.RuntimeConfig{},
			mcpServer.MCPToolCapability{},
			mcpServer.InvokeMCPToolRequestBody{},
			fmt.Errorf(
				"%w: MCP tool bridge is unavailable",
				mcpServer.ErrMCPRuntimeNotReady,
			)
	}

	normalized, err = normalizeMappedToolRequest(mapping, request)
	if err != nil {
		return mcpServer.RuntimeConfig{},
			mcpServer.MCPToolCapability{},
			mcpServer.InvokeMCPToolRequestBody{},
			err
	}
	if err := validateBridgeRequest(ctx, mapping.Server, normalized); err != nil {
		return mcpServer.RuntimeConfig{},
			mcpServer.MCPToolCapability{},
			mcpServer.InvokeMCPToolRequestBody{},
			err
	}

	config, tool, err = b.runtime.CallToolDryRun(
		ctx,
		mapping.Server,
		normalized,
	)
	if err != nil {
		return mcpServer.RuntimeConfig{},
			mcpServer.MCPToolCapability{},
			mcpServer.InvokeMCPToolRequestBody{},
			err
	}
	if tool.ProviderToolName != mapping.ProviderToolName ||
		tool.ChoiceID != mapping.ChoiceID ||
		tool.ToolName != mapping.ToolName ||
		tool.Digest != mapping.ToolDigest {
		return mcpServer.RuntimeConfig{},
			mcpServer.MCPToolCapability{},
			mcpServer.InvokeMCPToolRequestBody{},
			fmt.Errorf(
				"%w: MCP provider tool mapping is stale",
				mcpServer.ErrMCPStaleReference,
			)
	}

	config, err = applyMappedPolicyConstraints(
		config,
		tool,
		mapping,
	)
	if err != nil {
		return mcpServer.RuntimeConfig{},
			mcpServer.MCPToolCapability{},
			mcpServer.InvokeMCPToolRequestBody{},
			err
	}
	return config, tool, normalized, nil
}

func normalizeMappedToolRequest(
	mapping mcpConversation.MCPProviderToolMapping,
	request mcpServer.InvokeMCPToolRequestBody,
) (mcpServer.InvokeMCPToolRequestBody, error) {
	if err := mcpConversation.ValidateMCPProviderToolMapping(mapping); err != nil {
		return mcpServer.InvokeMCPToolRequestBody{}, err
	}

	request.ToolName = strings.TrimSpace(request.ToolName)
	request.ProviderToolName = strings.TrimSpace(
		request.ProviderToolName,
	)

	switch request.ToolName {
	case "":
		if request.ProviderToolName != mapping.ProviderToolName {
			return mcpServer.InvokeMCPToolRequestBody{}, fmt.Errorf(
				"%w: provider tool name does not match the persisted mapping",
				mcpServer.ErrMCPInvalidRuntimeRequest,
			)
		}
		request.ToolName = mapping.ToolName

	case mapping.ProviderToolName:
		request.ToolName = mapping.ToolName

	case mapping.ToolName:
	default:
		return mcpServer.InvokeMCPToolRequestBody{}, fmt.Errorf(
			"%w: tool name does not match the persisted mapping",
			mcpServer.ErrMCPInvalidRuntimeRequest,
		)
	}

	if request.ProviderToolName != "" &&
		request.ProviderToolName != mapping.ProviderToolName {
		return mcpServer.InvokeMCPToolRequestBody{}, fmt.Errorf(
			"%w: provider tool name does not match the persisted mapping",
			mcpServer.ErrMCPInvalidRuntimeRequest,
		)
	}

	if request.ToolDigest != "" &&
		request.ToolDigest != mapping.ToolDigest {
		return mcpServer.InvokeMCPToolRequestBody{}, fmt.Errorf(
			"%w: provider tool digest does not match the persisted mapping",
			mcpServer.ErrMCPStaleReference,
		)
	}

	request.ProviderToolName = mapping.ProviderToolName
	request.ToolDigest = mapping.ToolDigest
	return request, nil
}

func applyMappedPolicyConstraints(
	config mcpServer.RuntimeConfig,
	tool mcpServer.MCPToolCapability,
	mapping mcpConversation.MCPProviderToolMapping,
) (mcpServer.RuntimeConfig, error) {
	currentApproval, currentExecution := currentToolConstraints(config, tool)

	effective, err := mcpPolicy.TightenToolPolicy(
		config.Policy,
		tool.ToolName,
		currentApproval,
		currentExecution,
		mapping.ApprovalRule,
		mapping.ExecutionMode,
	)
	if err != nil {
		return mcpServer.RuntimeConfig{}, fmt.Errorf(
			"%w: %w",
			mcpServer.ErrMCPPolicyDenied,
			err,
		)
	}

	output := config
	output.Policy = effective
	return output, nil
}

func currentToolConstraints(
	config mcpServer.RuntimeConfig,
	tool mcpServer.MCPToolCapability,
) (mcpPolicy.MCPApprovalRule, mcpPolicy.MCPExecutionMode) {
	return mcpPolicy.EffectiveToolConstraints(
		config.Policy,
		tool.ToolName,
		tool.ApprovalRule,
		tool.ExecutionMode,
	)
}

func evaluateTool(
	config mcpServer.RuntimeConfig,
	tool mcpServer.MCPToolCapability,
	request mcpServer.InvokeMCPToolRequestBody,
) mcpServer.MCPApprovalEvaluation {
	approvalRule, executionMode := currentToolConstraints(config, tool)
	override := config.Policy.ToolPolicies[tool.ToolName]

	outcome := mcpPolicy.EvaluateTool(
		config.Policy,
		mcpPolicy.ToolEvaluationInput{
			Enabled:             tool.Enabled,
			TaskSupportRequired: tool.TaskSupport == mcpServer.MCPTaskSupportRequired,
			ToolDigest:          tool.Digest,
			RequestedToolDigest: request.ToolDigest,
			ExpectedDigest:      override.ExpectedDigest,
			AllowStaleDigest:    override.AllowStaleDigest,
			Risk:                string(tool.InferredRisk),
			ApprovalRule:        approvalRule,
			ExecutionMode:       executionMode,
			Source:              string(request.Source),
		},
	)

	decision := mcpServer.MCPApprovalDecisionDenied
	switch outcome.Decision {
	case mcpPolicy.ToolDecisionAllowed:
		decision = mcpServer.MCPApprovalDecisionAllowed
	case mcpPolicy.ToolDecisionApprovalRequired:
		decision = mcpServer.MCPApprovalDecisionApprovalRequired
	default:
	}

	return mcpServer.MCPApprovalEvaluation{
		Decision: decision,
		Reason:   outcome.Reason,
		Summary:  approvalSummary(config, tool, request),
	}
}

func approvalSummary(
	config mcpServer.RuntimeConfig,
	tool mcpServer.MCPToolCapability,
	request mcpServer.InvokeMCPToolRequestBody,
) *mcpServer.MCPApprovalSummary {
	arguments := []byte(`{}`)
	if request.Arguments != nil {
		encoded, err := json.Marshal(request.Arguments)
		if err == nil {
			arguments = encoded
		}
	}

	return &mcpServer.MCPApprovalSummary{
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
	evaluation mcpServer.MCPApprovalEvaluation,
) mcpServer.MCPApprovalEvaluation {
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
	case mcpServer.MCPApprovalResolutionAllowAlways:
		// A remembered user approval may satisfy an approval prompt, but it
		// must never override a hard policy denial.
		if evaluation.Decision ==
			mcpServer.MCPApprovalDecisionApprovalRequired {
			evaluation.Decision = mcpServer.MCPApprovalDecisionAllowed
			evaluation.Reason = "remembered session approval"
		}

	case mcpServer.MCPApprovalResolutionDenyAlways:
		// A remembered denial also applies when the base policy would
		// otherwise permit the call.
		if evaluation.Decision != mcpServer.MCPApprovalDecisionDenied {
			evaluation.Decision = mcpServer.MCPApprovalDecisionDenied
			evaluation.Reason = "remembered session denial"
		}

	default:
	}
	return evaluation
}

func validateBridgeRequest(
	ctx context.Context,
	serverRef mcpServer.ServerID,
	request mcpServer.InvokeMCPToolRequestBody,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: MCP tool invocation context is nil",
			mcpServer.ErrMCPInvalidRuntimeRequest,
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
	if request.Source == mcpServer.MCPInvocationSourceApp &&
		strings.TrimSpace(request.AppInstanceID) == "" {
		return fmt.Errorf(
			"%w: appInstanceID is required for app-initiated MCP calls",
			mcpServer.ErrMCPInvalidRuntimeRequest,
		)
	}
	if strings.TrimSpace(request.ToolName) == "" {
		return fmt.Errorf(
			"%w: MCP tool name is required",
			mcpServer.ErrMCPInvalidRuntimeRequest,
		)
	}
	return nil
}

// validateMCPInvocationSource validates the source of a tool invocation.
// The caller is still responsible for policy and approval enforcement.
func validateMCPInvocationSource(value mcpServer.MCPInvocationSource) error {
	switch value {
	case mcpServer.MCPInvocationSourceModel,
		mcpServer.MCPInvocationSourceUser,
		mcpServer.MCPInvocationSourceApp:
		return nil
	default:
		return fmt.Errorf(
			"%w: invalid MCP invocation source %q",
			mcpServer.ErrInvalid,
			value,
		)
	}
}
