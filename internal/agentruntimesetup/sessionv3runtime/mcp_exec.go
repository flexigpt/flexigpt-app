package sessionv3runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/invocation"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/mcppolicy"
)

type privateTool struct {
	Kind             sessionv3Runtime.ToolBindingKind
	Call             sessionv3Runtime.ToolCall
	MCPApprovalID    string
	MCPApprovalToken string
}

type privateToolWait struct {
	ApprovalID string
	Summary    mcpServer.MCPApprovalSummary
	Mapping    invocation.MappedToolBinding
	Request    mcpServer.InvokeMCPToolRequestBody
	Call       sessionv3Runtime.ToolCall
}

func (b *Binder) bindTool(
	ctx context.Context,
	req sessionv3Runtime.WorkDispatchRequest,
) (sessionv3Runtime.BindResult, error) {
	payload := req.WorkItem.Input.Tool
	if payload == nil {
		return sessionv3Runtime.BindResult{}, errors.New("tool payload is missing")
	}
	call := payload.Call
	if err := call.Validate(); err != nil {
		return sessionv3Runtime.BindResult{}, err
	}

	semantic := sessionv3Runtime.SemanticBinding{
		Version:           "v3-tool/v1",
		TargetFingerprint: call.Name + ":" + call.ID,
		BoundAt:           boundNow(),
	}
	if call.Binding.Kind == sessionv3Runtime.ToolBindingKindMCP {
		effective := *payload
		effective.Call = call
		mapping, invokeReq, err := b.mappedMCPRequest(&effective)
		if err != nil {
			return sessionv3Runtime.BindResult{}, err
		}
		evaluation, err := b.deps.MCPBridge.EvaluateMapped(ctx, mapping, invokeReq)
		if err != nil {
			return sessionv3Runtime.BindResult{}, err
		}
		if evaluation == nil {
			return sessionv3Runtime.BindResult{}, errors.New(
				"MCP approval evaluation returned no result",
			)
		}
		switch evaluation.Decision {
		case mcpServer.MCPApprovalDecisionApprovalRequired:
			if evaluation.Summary == nil || evaluation.ApprovalID == "" {
				return sessionv3Runtime.BindResult{}, errors.New(
					"MCP approval evaluation has no approval identity or summary",
				)
			}
			wait := sessionv3Runtime.WaitRequest{
				Kind:    sessionv3Runtime.WaitKindApproval,
				Summary: sessionv3Runtime.MustCanonicalValue(evaluation.Summary),
				AllowedResolutions: []sessionv3Runtime.WaitResolution{
					sessionv3Runtime.WaitResolutionApproveOnce,
					sessionv3Runtime.WaitResolutionDenyOnce,
					sessionv3Runtime.WaitResolutionApproveAlways,
					sessionv3Runtime.WaitResolutionDenyAlways,
				},
				RuntimeBound: true,
			}
			return sessionv3Runtime.BindResult{
				SemanticBinding: semantic,
				Wait:            &wait,
				Private: privateToolWait{
					ApprovalID: evaluation.ApprovalID,
					Summary:    *evaluation.Summary,
					Mapping:    mapping,
					Request:    invokeReq,
					Call:       call,
				},
			}, nil

		case mcpServer.MCPApprovalDecisionDenied:
			return sessionv3Runtime.BindResult{
				SemanticBinding: semantic,
				ImmediateOutput: &sessionv3Runtime.WorkItemOutput{
					Tool: &sessionv3Runtime.ToolOutput{
						Kind: sessionv3Runtime.ToolOutputKindError,
						Text: firstText(
							evaluation.Reason,
							"MCP policy denied the Tool call.",
						),
					},
				},
			}, nil

		case mcpServer.MCPApprovalDecisionAllowed:
			// Continue into the normal bound WorkItem path.

		default:
			return sessionv3Runtime.BindResult{}, fmt.Errorf(
				"unsupported MCP approval decision %q", evaluation.Decision,
			)
		}
	}
	return sessionv3Runtime.BindResult{
		SemanticBinding: semantic,
		Private: privateTool{
			Kind: call.Binding.Kind,
			Call: call,
		},
	}, nil
}

func (b *Binder) executeTool(
	ctx context.Context,
	req sessionv3Runtime.WorkDispatchRequest,
	_ sessionv3Runtime.BoundWork,
	private privateTool,
) (sessionv3Runtime.WorkExecutionResult, error) {
	payload := req.WorkItem.Input.Tool
	if payload == nil {
		return sessionv3Runtime.WorkExecutionResult{}, errors.New("tool payload is missing")
	}
	effective := *payload
	effective.Call = private.Call

	switch private.Kind {
	case sessionv3Runtime.ToolBindingKindGo:
		return b.executeGoTool(ctx, &effective)
	case sessionv3Runtime.ToolBindingKindMCP:
		return b.executeMCPTool(ctx, &effective, private)
	case sessionv3Runtime.ToolBindingKindSkill:
		return b.executeSkillTool(ctx, req, &effective)
	case sessionv3Runtime.ToolBindingKindProvider:
		return sessionv3Runtime.WorkExecutionResult{
			Output: &sessionv3Runtime.WorkItemOutput{
				Tool: &sessionv3Runtime.ToolOutput{
					Kind: sessionv3Runtime.ToolOutputKindSuccess,
					Text: "provider managed tool observed",
				},
			},
			Confirmed: true,
		}, nil
	default:
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{Code: "unresolved_tool", Message: "tool binding is unresolved"},
		}, nil
	}
}

func (b *Binder) mappedMCPRequest(
	payload *sessionv3Runtime.ToolWork,
) (invocation.MappedToolBinding, mcpServer.InvokeMCPToolRequestBody, error) {
	binding := payload.Call.Binding.MCP
	if binding == nil {
		return invocation.MappedToolBinding{}, mcpServer.InvokeMCPToolRequestBody{}, errors.New(
			"mcp binding is missing",
		)
	}
	mapping := invocation.MappedToolBinding{
		Server:           mcpServer.ServerID(binding.ServerID),
		ProviderToolName: binding.ProviderToolName,
		ChoiceID:         binding.ChoiceID,
		ToolName:         binding.ToolName,
		ToolDigest:       binding.ToolDigest,
		ApprovalRule:     mcppolicy.MCPApprovalRule(binding.ApprovalRule),
		ExecutionMode:    mcppolicy.MCPExecutionMode(binding.ExecutionMode),
	}
	if err := mapping.Validate(); err != nil {
		return invocation.MappedToolBinding{}, mcpServer.InvokeMCPToolRequestBody{}, err
	}

	arguments := payload.Call.Arguments
	if len(arguments) == 0 {
		arguments = sessionv3Runtime.MustCanonicalValue(map[string]any{})
	}
	if !arguments.Valid() {
		return invocation.MappedToolBinding{},
			mcpServer.InvokeMCPToolRequestBody{},
			errors.New("MCP Tool arguments are not canonical JSON")
	}

	var args map[string]any
	if err := json.Unmarshal(arguments, &args); err != nil || args == nil {
		return invocation.MappedToolBinding{},
			mcpServer.InvokeMCPToolRequestBody{},
			errors.New("MCP Tool arguments must be a JSON object")
	}

	invokeReq := mcpServer.InvokeMCPToolRequestBody{
		Source:           mcpServer.MCPInvocationSourceModel,
		ToolName:         binding.ToolName,
		ProviderToolName: binding.ProviderToolName,
		ChoiceID:         binding.ChoiceID,
		ToolDigest:       binding.ToolDigest,
		Arguments:        args,
		ToolUseID:        payload.Call.ID,
	}
	return mapping, invokeReq, nil
}

func (b *Binder) executeMCPTool(
	ctx context.Context,
	payload *sessionv3Runtime.ToolWork,
	private privateTool,
) (sessionv3Runtime.WorkExecutionResult, error) {
	mapping, invokeReq, err := b.mappedMCPRequest(payload)
	if err != nil {
		//nolint:nilerr // Explicit return.
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{Code: "invalid_mcp_binding", Message: err.Error()},
		}, nil
	}
	invokeReq.ApprovalID = private.MCPApprovalID
	invokeReq.ApprovalToken = private.MCPApprovalToken
	resp, err := b.deps.MCPBridge.InvokeMapped(ctx, mapping, invokeReq)
	if err != nil {
		//nolint:nilerr // Explicit return.
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{Code: "mcp_failed", Message: err.Error()},
		}, nil
	}
	if resp == nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{Code: "mcp_empty", Message: "MCP returned no response"},
		}, nil
	}
	text := string(mustMarshal(resp.Content))
	kind := sessionv3Runtime.ToolOutputKindSuccess
	if resp.IsError {
		kind = sessionv3Runtime.ToolOutputKindError
	}
	return sessionv3Runtime.WorkExecutionResult{
		Output: &sessionv3Runtime.WorkItemOutput{
			Tool: &sessionv3Runtime.ToolOutput{
				Kind:       kind,
				Text:       text,
				Structured: sessionv3Runtime.MustCanonicalValue(resp),
			},
		},
		Confirmed: true,
	}, nil
}

func (b *Binder) resolveToolWait(
	ctx context.Context,
	req sessionv3Runtime.WaitResolutionRequest,
	private privateToolWait,
) (sessionv3Runtime.WaitResolutionResult, error) {
	resolution, err := mcpWaitResolution(req.Resolution)
	if err != nil {
		return sessionv3Runtime.WaitResolutionResult{}, err
	}
	result, err := b.deps.MCPBridge.ResolveApproval(ctx, private.ApprovalID, resolution)
	if err != nil {
		return sessionv3Runtime.WaitResolutionResult{}, err
	}
	if result.Decision == mcpServer.MCPApprovalDecisionDenied {
		return sessionv3Runtime.WaitResolutionResult{
			Resume: false,
			Output: &sessionv3Runtime.WorkItemOutput{
				Tool: &sessionv3Runtime.ToolOutput{
					Kind: sessionv3Runtime.ToolOutputKindError,
					Text: "MCP tool call was denied",
				},
			},
			Confirmed: true,
		}, nil
	}

	if result.Decision != mcpServer.MCPApprovalDecisionAllowed {
		return sessionv3Runtime.WaitResolutionResult{}, fmt.Errorf(
			"unsupported MCP approval resolution decision %q",
			result.Decision,
		)
	}
	if req.Resolution == sessionv3Runtime.WaitResolutionApproveOnce &&
		result.Token == "" {
		return sessionv3Runtime.WaitResolutionResult{}, errors.New(
			"MCP one-time approval returned no approval token",
		)
	}
	return sessionv3Runtime.WaitResolutionResult{
		Resume: true,
		Bound: &sessionv3Runtime.BoundWork{
			SemanticBinding: sessionv3Runtime.SemanticBinding{
				Version:           "v3-mcp-resume/v1",
				TargetFingerprint: private.Mapping.ToolName,
				BoundAt:           boundNow(),
			},
			Private: privateTool{
				Kind:             sessionv3Runtime.ToolBindingKindMCP,
				Call:             private.Call,
				MCPApprovalID:    result.ApprovalID,
				MCPApprovalToken: result.Token,
			},
		},
	}, nil
}

func mcpWaitResolution(v sessionv3Runtime.WaitResolution) (mcpServer.MCPApprovalResolution, error) {
	switch v {
	case sessionv3Runtime.WaitResolutionApproveOnce:
		return mcpServer.MCPApprovalResolutionAllowOnce, nil
	case sessionv3Runtime.WaitResolutionApproveAlways:
		return mcpServer.MCPApprovalResolutionAllowAlways, nil
	case sessionv3Runtime.WaitResolutionDenyOnce:
		return mcpServer.MCPApprovalResolutionDenyOnce, nil
	case sessionv3Runtime.WaitResolutionDenyAlways:
		return mcpServer.MCPApprovalResolutionDenyAlways, nil
	default:
		return "", fmt.Errorf("unsupported MCP wait resolution %q", v)
	}
}
