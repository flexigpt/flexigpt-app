package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/invocation"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	conversationSpec "github.com/flexigpt/flexigpt-app/internal/conversation/spec"
)

type Adapter struct {
	bridge *invocation.ToolBridge
}

func NewAdapter(
	bridge *invocation.ToolBridge,
) (*Adapter, error) {
	if bridge == nil {
		return nil, errors.New("session MCP adapter requires a ToolBridge")
	}

	return &Adapter{bridge: bridge}, nil
}

func (a *Adapter) EvaluateTool(
	ctx context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolEvaluation, error) {
	mapping, invokeRequest, err := decodeRequest(request)
	if err != nil {
		return sessionRuntime.ToolEvaluation{}, err
	}

	evaluation, err := a.bridge.EvaluateMapped(
		ctx,
		mapping,
		invokeRequest,
	)
	if err != nil {
		return sessionRuntime.ToolEvaluation{}, err
	}
	if evaluation == nil {
		return sessionRuntime.ToolEvaluation{}, errors.New(
			"MCP ToolBridge returned no evaluation",
		)
	}

	switch evaluation.Decision {
	case mcpServer.MCPApprovalDecisionAllowed:
		return sessionRuntime.ToolEvaluation{
			Decision: sessionRuntime.ToolEvaluationAllowed,
		}, nil

	case mcpServer.MCPApprovalDecisionDenied:
		return sessionRuntime.ToolEvaluation{
			Decision: sessionRuntime.ToolEvaluationDenied,
			Reason:   evaluation.Reason,
			Result: &sessionRuntime.ToolExecutionResult{
				HasResult: true,
				Kind:      sessionRuntime.ToolResultKindError,
				Text:      failureText(evaluation.Reason),
				Error:     evaluation.Reason,
			},
		}, nil

	case mcpServer.MCPApprovalDecisionApprovalRequired:
		if evaluation.ApprovalID == "" {
			return sessionRuntime.ToolEvaluation{}, errors.New(
				"MCP ToolBridge required approval without an approval ID",
			)
		}

		summary, marshalErr := json.Marshal(evaluation.Summary)
		if marshalErr != nil {
			return sessionRuntime.ToolEvaluation{}, marshalErr
		}

		return sessionRuntime.ToolEvaluation{
			Decision: sessionRuntime.ToolEvaluationWait,
			Reason:   evaluation.Reason,
			Wait: &sessionRuntime.ToolWaitSpec{
				Kind:    sessionRuntime.WaitKindApproval,
				Summary: summary,
				AllowedResolutions: []string{
					string(mcpServer.MCPApprovalResolutionAllowOnce),
					string(mcpServer.MCPApprovalResolutionAllowAlways),
					string(mcpServer.MCPApprovalResolutionDenyOnce),
					string(mcpServer.MCPApprovalResolutionDenyAlways),
				},
				ExecutorReference: evaluation.ApprovalID,
			},
		}, nil

	default:
		return sessionRuntime.ToolEvaluation{}, fmt.Errorf(
			"unsupported MCP approval decision %q",
			evaluation.Decision,
		)
	}
}

func (a *Adapter) ExecuteTool(
	ctx context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolExecutionResult, error) {
	mapping, invokeRequest, err := decodeRequest(request)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}

	var authorization struct {
		ApprovalID string `json:"approvalID,omitempty"`
		Token      string `json:"token,omitempty"`
	}
	if len(request.Authorization) != 0 {
		if err := json.Unmarshal(request.Authorization, &authorization); err != nil {
			return sessionRuntime.ToolExecutionResult{}, fmt.Errorf(
				"decode MCP Session authorization: %w",
				err,
			)
		}
	}
	invokeRequest.ApprovalID = authorization.ApprovalID
	invokeRequest.ApprovalToken = authorization.Token

	response, err := a.bridge.InvokeMapped(
		ctx,
		mapping,
		invokeRequest,
	)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}
	if response == nil {
		return sessionRuntime.ToolExecutionResult{}, errors.New(
			"MCP ToolBridge returned no invocation response",
		)
	}

	structured, marshalErr := json.Marshal(map[string]any{
		"content":           response.Content,
		"structuredContent": response.StructuredContent,
	})
	if marshalErr != nil {
		return sessionRuntime.ToolExecutionResult{}, marshalErr
	}

	app, marshalErr := json.Marshal(response.App)
	if marshalErr != nil {
		return sessionRuntime.ToolExecutionResult{}, marshalErr
	}

	result := sessionRuntime.ToolExecutionResult{
		HasResult:  true,
		Kind:       sessionRuntime.ToolResultKindSuccess,
		Text:       string(structured),
		Structured: structured,
		App:        app,
	}
	if response.IsError {
		result.Kind = sessionRuntime.ToolResultKindError
		result.Error = firstNonBlank(
			string(structured),
			"MCP tool returned an error result",
		)
	}

	return result, nil
}

func (a *Adapter) ResolveToolWait(
	ctx context.Context,
	request sessionRuntime.ToolWaitResolutionRequest,
) (sessionRuntime.ToolWaitResolution, error) {
	if strings.TrimSpace(request.ExecutorReference) == "" {
		return sessionRuntime.ToolWaitResolution{}, errors.New(
			"MCP approval wait has no runtime approval reference",
		)
	}

	resolution := mcpServer.MCPApprovalResolution(
		strings.TrimSpace(request.Resolution),
	)
	result, err := a.bridge.ResolveApproval(
		ctx,
		request.ExecutorReference,
		resolution,
	)
	if err != nil {
		return sessionRuntime.ToolWaitResolution{}, err
	}

	switch result.Decision {
	case mcpServer.MCPApprovalDecisionAllowed:
		authorization, marshalErr := json.Marshal(map[string]string{
			"approvalID": result.ApprovalID,
			"token":      result.Token,
		})
		if marshalErr != nil {
			return sessionRuntime.ToolWaitResolution{}, marshalErr
		}

		return sessionRuntime.ToolWaitResolution{
			Status:        sessionRuntime.ToolWaitResolutionApproved,
			Authorization: authorization,
		}, nil

	case mcpServer.MCPApprovalDecisionDenied:
		return sessionRuntime.ToolWaitResolution{
			Status: sessionRuntime.ToolWaitResolutionDenied,
			Result: &sessionRuntime.ToolExecutionResult{
				HasResult: true,
				Kind:      sessionRuntime.ToolResultKindError,
				Text:      failureText("MCP tool call was denied"),
				Error:     "MCP tool call was denied",
			},
		}, nil

	default:
		return sessionRuntime.ToolWaitResolution{}, fmt.Errorf(
			"unsupported MCP approval resolution decision %q",
			result.Decision,
		)
	}
}

func decodeRequest(
	request sessionRuntime.ToolExecutionRequest,
) (
	conversationSpec.MCPProviderToolMapping,
	mcpServer.InvokeMCPToolRequestBody,
	error,
) {
	var mapping conversationSpec.MCPProviderToolMapping
	if err := json.Unmarshal(request.ToolCall.Binding.Data, &mapping); err != nil {
		return conversationSpec.MCPProviderToolMapping{},
			mcpServer.InvokeMCPToolRequestBody{},
			fmt.Errorf("decode MCP Session binding: %w", err)
	}

	arguments := map[string]any{}
	if len(request.ToolCall.Arguments) != 0 {
		if err := json.Unmarshal(request.ToolCall.Arguments, &arguments); err != nil {
			return conversationSpec.MCPProviderToolMapping{},
				mcpServer.InvokeMCPToolRequestBody{},
				fmt.Errorf("decode MCP tool arguments: %w", err)
		}
	}

	return mapping, mcpServer.InvokeMCPToolRequestBody{
		Source:           mcpServer.MCPInvocationSourceModel,
		ToolName:         mapping.ToolName,
		ProviderToolName: mapping.ProviderToolName,
		ChoiceID:         mapping.ChoiceID,
		ToolDigest:       mapping.ToolDigest,
		Arguments:        arguments,
		ToolUseID:        request.ToolCall.ProviderCallID,
	}, nil
}

func failureText(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "MCP policy denied this tool call."
	}

	return strings.Join([]string{
		"MCP tool error: " + message,
		"This result completes the requested tool call.",
		"The model may correct the request or retry the tool.",
	}, "\n\n")
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
