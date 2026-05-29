package runtime

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

type ToolBridge struct {
	runtime   *RuntimeManager
	approvals *ApprovalManager
}

func NewToolBridge(rt *RuntimeManager, approvals *ApprovalManager) *ToolBridge {
	return &ToolBridge{runtime: rt, approvals: approvals}
}

func (b *ToolBridge) Evaluate(
	ctx context.Context,
	req *spec.EvaluateMCPToolCallRequest,
) (*spec.EvaluateMCPToolCallResponse, error) {
	if b == nil || b.runtime == nil || b.approvals == nil {
		return nil, fmt.Errorf("%w: nil tool bridge", spec.ErrMCPRuntimeNotReady)
	}
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", spec.ErrMCPInvalidRequest)
	}

	_, cfg, tool, err := b.runtime.CallToolDryRun(ctx, *req.Body)
	if err != nil {
		return nil, err
	}

	eval := Evaluate(EvaluationInput{
		Server: cfg,
		Tool:   tool,
		Req:    *req.Body,
	})

	if eval.Decision == spec.MCPApprovalDecisionApprovalRequired && eval.Summary != nil {
		id, err := b.approvals.Create(ctx, *eval.Summary)
		if err != nil {
			return nil, err
		}
		eval.ApprovalID = id
	}

	return &spec.EvaluateMCPToolCallResponse{Body: &eval}, nil
}

func (b *ToolBridge) Invoke(
	ctx context.Context,
	req *spec.InvokeMCPToolRequest,
) (*spec.InvokeMCPToolResponse, error) {
	if b == nil || b.runtime == nil || b.approvals == nil {
		return nil, fmt.Errorf("%w: nil tool bridge", spec.ErrMCPRuntimeNotReady)
	}
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", spec.ErrMCPInvalidRequest)
	}

	_, cfg, tool, err := b.runtime.CallToolDryRun(ctx, *req.Body)
	if err != nil {
		return nil, err
	}

	eval := Evaluate(EvaluationInput{
		Server: cfg,
		Tool:   tool,
		Req:    *req.Body,
	})

	if eval.Decision == spec.MCPApprovalDecisionApprovalRequired && eval.Summary != nil {
		if cached, ok := b.approvals.LookupDecision(*eval.Summary); ok {
			switch cached {
			case spec.MCPApprovalResolutionAllowAlways:
				eval.Decision = spec.MCPApprovalDecisionAllowed
				eval.Reason = "cached allow-always decision"
			case spec.MCPApprovalResolutionDenyAlways:
				eval.Decision = spec.MCPApprovalDecisionDenied
				eval.Reason = "cached deny-always decision"
			default:
			}
		}
	}

	switch eval.Decision {
	case spec.MCPApprovalDecisionDenied:
		return nil, fmt.Errorf("%w: %s", spec.ErrMCPPolicyDenied, eval.Reason)

	case spec.MCPApprovalDecisionApprovalRequired:
		if req.Body.ApprovalToken == "" {
			return nil, fmt.Errorf("%w: %s", spec.ErrMCPApprovalNeeded, eval.Reason)
		}
		if eval.Summary == nil {
			return nil, fmt.Errorf("%w: missing approval summary", spec.ErrMCPApprovalNeeded)
		}
		if err := b.approvals.VerifyAndConsumeToken(ctx, req.Body.ApprovalToken, *eval.Summary); err != nil {
			return nil, err
		}

	case spec.MCPApprovalDecisionAllowed:
		// Continue.

	default:
		return nil, fmt.Errorf("%w: invalid approval decision %q", spec.ErrMCPInvalidRequest, eval.Decision)
	}

	out, _, _, err := b.runtime.CallTool(ctx, *req.Body)
	if err != nil {
		return nil, err
	}
	return &spec.InvokeMCPToolResponse{Body: out}, nil
}
