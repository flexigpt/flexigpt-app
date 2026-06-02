package runtime

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	toolDigestChangedReason = "tool digest changed"
	toolPolicyDeniesReason  = "server/tool policy denies this tool"
	policyAllowedReason     = "policy allowed"
)

type EvaluationInput struct {
	Server spec.MCPServerConfig
	Tool   spec.MCPToolCapability
	Req    spec.InvokeMCPToolRequestBody
}

func Evaluate(in EvaluationInput) spec.MCPApprovalEvaluation {
	p := in.Server.DefaultPolicy
	if p == (spec.MCPServerPolicy{}) {
		p = spec.DefaultMCPServerPolicy()
	}

	if !in.Tool.Enabled {
		return spec.MCPApprovalEvaluation{
			Decision: spec.MCPApprovalDecisionDenied,
			Reason:   "tool is disabled or unsupported",
			Summary:  summary(in),
		}
	}

	if in.Tool.TaskSupport == spec.MCPTaskSupportRequired {
		return spec.MCPApprovalEvaluation{
			Decision: spec.MCPApprovalDecisionDenied,
			Reason:   "task-required MCP tools are unsupported",
			Summary:  summary(in),
		}
	}

	rule := p.DefaultApprovalRule
	digestChanged := false
	allowStaleDigest := false

	if ov, ok := in.Server.ToolPolicies[in.Tool.ToolName]; ok {
		allowStaleDigest = ov.AllowStaleDigest
		if ov.ApprovalRule != nil {
			rule = *ov.ApprovalRule
		}
		if ov.ExpectedDigest != "" && ov.ExpectedDigest != in.Tool.Digest && !ov.AllowStaleDigest {
			digestChanged = true
		}
	}

	if in.Req.ToolDigest != "" &&
		in.Tool.Digest != "" &&
		in.Req.ToolDigest != in.Tool.Digest &&
		!allowStaleDigest {
		digestChanged = true
	}

	if rule == spec.MCPApprovalRuleDeny {
		return spec.MCPApprovalEvaluation{
			Decision: spec.MCPApprovalDecisionDenied,
			Reason:   toolPolicyDeniesReason,
			Summary:  summary(in),
		}
	}

	if digestChanged {
		return spec.MCPApprovalEvaluation{
			Decision: spec.MCPApprovalDecisionDenied,
			Reason:   toolDigestChangedReason,
			Summary:  summary(in),
		}
	}

	if rule == spec.MCPApprovalRuleAsk {
		return spec.MCPApprovalEvaluation{
			Decision: spec.MCPApprovalDecisionApprovalRequired,
			Reason:   "approval rule is ask",
			Summary:  summary(in),
		}
	}

	switch in.Tool.InferredRisk {
	case spec.MCPToolRiskUnknown:
		if p.RequireApprovalForUnknownRisk {
			return spec.MCPApprovalEvaluation{
				Decision: spec.MCPApprovalDecisionApprovalRequired,
				Reason:   "unknown-risk tool requires approval",
				Summary:  summary(in),
			}
		}
	case spec.MCPToolRiskWrite:
		if p.RequireApprovalForWrite {
			return spec.MCPApprovalEvaluation{
				Decision: spec.MCPApprovalDecisionApprovalRequired,
				Reason:   "write-risk tool requires approval",
				Summary:  summary(in),
			}
		}
	case spec.MCPToolRiskDestructive:
		if p.RequireApprovalForDestructive {
			return spec.MCPApprovalEvaluation{
				Decision: spec.MCPApprovalDecisionApprovalRequired,
				Reason:   "destructive-risk tool requires approval",
				Summary:  summary(in),
			}
		}
	default:
	}
	return spec.MCPApprovalEvaluation{
		Decision: spec.MCPApprovalDecisionAllowed,
		Reason:   policyAllowedReason,
		Summary:  summary(in),
	}
}

func summary(in EvaluationInput) *spec.MCPApprovalSummary {
	raw := []byte(`{}`)
	if in.Req.Arguments != nil {
		raw, _ = json.Marshal(in.Req.Arguments)
	}
	return &spec.MCPApprovalSummary{
		BundleID:          in.Server.BundleID,
		ServerID:          in.Server.ID,
		ServerDisplayName: in.Server.DisplayName,
		ToolName:          in.Tool.ToolName,
		ToolDigest:        in.Tool.Digest,
		Risk:              in.Tool.InferredRisk,
		Arguments:         spec.JSONRawString(raw),
	}
}

func ExecutionMode(server spec.MCPServerConfig, tool spec.MCPToolCapability) spec.MCPExecutionMode {
	mode := server.DefaultPolicy.DefaultExecutionMode
	if mode == "" {
		mode = spec.MCPExecutionModeManual
	}
	if ov, ok := server.ToolPolicies[tool.ToolName]; ok && ov.ExecutionMode != nil {
		mode = *ov.ExecutionMode
	}
	return mode
}

func EnsureDigest(expected, got string, allowStale bool) error {
	if expected == "" || got == "" || expected == got || allowStale {
		return nil
	}
	return fmt.Errorf("%w: expected %s got %s", spec.ErrMCPStaleReference, expected, got)
}
