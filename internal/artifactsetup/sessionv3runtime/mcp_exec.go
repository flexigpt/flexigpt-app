package sessionv3runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/invocation"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/mcppolicy"
)

type privateTool struct {
	Kind             sessionv3.ToolBindingKind
	Call             sessionv3.ToolCall
	MCPApprovalID    string
	MCPApprovalToken string
}

type privateToolWait struct {
	ApprovalID string
	Summary    mcpServer.MCPApprovalSummary
	Mapping    invocation.MappedToolBinding
	Request    mcpServer.InvokeMCPToolRequestBody
	Call       sessionv3.ToolCall
}

func (b *Binder) bindTool(ctx context.Context, req sessionv3.WorkDispatchRequest) (sessionv3.BindResult, error) {
	payload := req.WorkItem.Input.Tool
	if payload == nil {
		return sessionv3.BindResult{}, errors.New("tool payload is missing")
	}
	call := payload.Call
	if err := call.Validate(); err != nil {
		return sessionv3.BindResult{}, err
	}

	semantic := sessionv3.SemanticBinding{
		Version:           "v3-tool/v1",
		TargetFingerprint: call.Name + ":" + call.ID,
		BoundAt:           boundNow(),
	}
	if call.Binding.Kind == sessionv3.ToolBindingKindMCP {
		effective := *payload
		effective.Call = call
		mapping, invokeReq, err := b.mappedMCPRequest(&effective)
		if err != nil {
			return sessionv3.BindResult{}, err
		}
		evaluation, err := b.deps.MCPBridge.EvaluateMapped(ctx, mapping, invokeReq)
		if err != nil {
			return sessionv3.BindResult{}, err
		}
		if evaluation == nil {
			return sessionv3.BindResult{}, errors.New(
				"MCP approval evaluation returned no result",
			)
		}
		if evaluation.Decision == mcpServer.MCPApprovalDecisionApprovalRequired &&
			evaluation.Summary != nil {
			wait := sessionv3.WaitRequest{
				Kind:    sessionv3.WaitKindApproval,
				Summary: sessionv3.MustCanonicalValue(evaluation.Summary),
				AllowedResolutions: []sessionv3.WaitResolution{
					sessionv3.WaitResolutionApproveOnce,
					sessionv3.WaitResolutionDenyOnce,
					sessionv3.WaitResolutionApproveAlways,
					sessionv3.WaitResolutionDenyAlways,
				},
				RuntimeBound: true,
			}
			return sessionv3.BindResult{
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
		}
		if evaluation.Decision == mcpServer.MCPApprovalDecisionDenied {
			return sessionv3.BindResult{
				SemanticBinding: semantic,
				ImmediateOutput: &sessionv3.WorkItemOutput{
					Tool: &sessionv3.ToolOutput{
						Kind: sessionv3.ToolOutputKindError,
						Text: firstText(
							evaluation.Reason,
							"MCP policy denied the Tool call.",
						),
					},
				},
			}, nil
		}
	}
	return sessionv3.BindResult{
		SemanticBinding: semantic,
		Private: privateTool{
			Kind: call.Binding.Kind,
			Call: call,
		},
	}, nil
}

func (b *Binder) executeTool(
	ctx context.Context,
	req sessionv3.WorkDispatchRequest,
	_ sessionv3.BoundWork,
	private privateTool,
) (sessionv3.WorkExecutionResult, error) {
	payload := req.WorkItem.Input.Tool
	if payload == nil {
		return sessionv3.WorkExecutionResult{}, errors.New("tool payload is missing")
	}
	effective := *payload
	effective.Call = private.Call

	switch private.Kind {
	case sessionv3.ToolBindingKindGo:
		return b.executeGoTool(ctx, &effective)
	case sessionv3.ToolBindingKindMCP:
		return b.executeMCPTool(ctx, &effective, private)
	case sessionv3.ToolBindingKindSkill:
		return b.executeSkillTool(ctx, req, &effective)
	case sessionv3.ToolBindingKindProvider:
		return sessionv3.WorkExecutionResult{
			Output: &sessionv3.WorkItemOutput{
				Tool: &sessionv3.ToolOutput{
					Kind: sessionv3.ToolOutputKindSuccess,
					Text: "provider managed tool observed",
				},
			},
			Confirmed: true,
		}, nil
	default:
		return sessionv3.WorkExecutionResult{
			Error: &sessionv3.SanitizedError{Code: "unresolved_tool", Message: "tool binding is unresolved"},
		}, nil
	}
}

func (b *Binder) mappedMCPRequest(
	payload *sessionv3.ToolWork,
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
		arguments = sessionv3.MustCanonicalValue(map[string]any{})
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
	payload *sessionv3.ToolWork,
	private privateTool,
) (sessionv3.WorkExecutionResult, error) {
	mapping, invokeReq, err := b.mappedMCPRequest(payload)
	if err != nil {
		//nolint:nilerr // Explicit return.
		return sessionv3.WorkExecutionResult{
			Error: &sessionv3.SanitizedError{Code: "invalid_mcp_binding", Message: err.Error()},
		}, nil
	}
	invokeReq.ApprovalID = private.MCPApprovalID
	invokeReq.ApprovalToken = private.MCPApprovalToken
	resp, err := b.deps.MCPBridge.InvokeMapped(ctx, mapping, invokeReq)
	if err != nil {
		//nolint:nilerr // Explicit return.
		return sessionv3.WorkExecutionResult{
			Error: &sessionv3.SanitizedError{Code: "mcp_failed", Message: err.Error()},
		}, nil
	}
	text := string(mustMarshal(resp.Content))
	kind := sessionv3.ToolOutputKindSuccess
	if resp.IsError {
		kind = sessionv3.ToolOutputKindError
	}
	return sessionv3.WorkExecutionResult{
		Output: &sessionv3.WorkItemOutput{
			Tool: &sessionv3.ToolOutput{
				Kind:       kind,
				Text:       text,
				Structured: sessionv3.MustCanonicalValue(resp),
			},
		},
		Confirmed: true,
	}, nil
}

func (b *Binder) resolveToolWait(
	ctx context.Context,
	req sessionv3.WaitResolutionRequest,
	private privateToolWait,
) (sessionv3.WaitResolutionResult, error) {
	resolution, err := mcpWaitResolution(req.Resolution)
	if err != nil {
		return sessionv3.WaitResolutionResult{}, err
	}
	result, err := b.deps.MCPBridge.ResolveApproval(ctx, private.ApprovalID, resolution)
	if err != nil {
		return sessionv3.WaitResolutionResult{}, err
	}
	if result.Decision == mcpServer.MCPApprovalDecisionDenied {
		return sessionv3.WaitResolutionResult{
			Resume: false,
			Output: &sessionv3.WorkItemOutput{
				Tool: &sessionv3.ToolOutput{Kind: sessionv3.ToolOutputKindError, Text: "MCP tool call was denied"},
			},
			Confirmed: true,
		}, nil
	}
	return sessionv3.WaitResolutionResult{
		Resume: true,
		Bound: &sessionv3.BoundWork{
			SemanticBinding: sessionv3.SemanticBinding{
				Version:           "v3-mcp-resume/v1",
				TargetFingerprint: private.Mapping.ToolName,
				BoundAt:           boundNow(),
			},
			Private: privateTool{
				Kind:             sessionv3.ToolBindingKindMCP,
				Call:             private.Call,
				MCPApprovalID:    result.ApprovalID,
				MCPApprovalToken: result.Token,
			},
		},
	}, nil
}

func mcpWaitResolution(v sessionv3.WaitResolution) (mcpServer.MCPApprovalResolution, error) {
	switch v {
	case sessionv3.WaitResolutionApproveOnce:
		return mcpServer.MCPApprovalResolutionAllowOnce, nil
	case sessionv3.WaitResolutionApproveAlways:
		return mcpServer.MCPApprovalResolutionAllowAlways, nil
	case sessionv3.WaitResolutionDenyOnce:
		return mcpServer.MCPApprovalResolutionDenyOnce, nil
	case sessionv3.WaitResolutionDenyAlways:
		return mcpServer.MCPApprovalResolutionDenyAlways, nil
	default:
		return "", fmt.Errorf("unsupported MCP wait resolution %q", v)
	}
}
