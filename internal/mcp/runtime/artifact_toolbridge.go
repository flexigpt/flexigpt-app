package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/mcp/apps"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
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
	allowStaleDigest := false

	if override, found := config.ToolPolicies[tool.ToolName]; found {
		if override.ApprovalRule != nil {
			rule = *override.ApprovalRule
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
	if strings.TrimSpace(request.ToolName) == "" {
		return fmt.Errorf(
			"%w: MCP tool name is required",
			spec.ErrMCPInvalidRequest,
		)
	}
	if request.Source == spec.MCPInvocationSourceApp &&
		strings.TrimSpace(request.AppInstanceID) == "" {
		return fmt.Errorf(
			"%w: appInstanceID is required for app-initiated MCP calls",
			spec.ErrMCPInvalidRequest,
		)
	}
	return nil
}
