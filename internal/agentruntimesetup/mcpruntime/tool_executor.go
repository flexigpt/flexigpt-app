package mcpruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/invocation"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// SessionToolExecutor owns MCP Tool evaluation, approvals, authorization
// tokens, invocation, and response adaptation. Session owns only Tool-call
// state transitions and durable wait records.
type SessionToolExecutor struct {
	bridge *invocation.ToolBridge
}

func NewSessionToolExecutor(
	bridge *invocation.ToolBridge,
) (*SessionToolExecutor, error) {
	if bridge == nil {
		return nil, errors.New(
			"session MCP Tool executor requires an invocation bridge",
		)
	}

	return &SessionToolExecutor{bridge: bridge}, nil
}

// ToolBindingData creates canonical durable MCP semantic binding data.
func ToolBindingData(
	tool mcpServer.MCPToolCapability,
) (json.RawMessage, error) {
	return jsonutil.MarshalCanonicalObject(
		invocation.MappedToolBinding{
			Server:           tool.Server,
			ProviderToolName: tool.ProviderToolName,
			ChoiceID:         tool.ChoiceID,
			ToolName:         tool.ToolName,
			ToolDigest:       tool.Digest,
			ApprovalRule:     tool.ApprovalRule,
			ExecutionMode:    tool.ExecutionMode,
			AppResourceURI:   appResourceURI(tool.App),
			Visibility:       appVisibility(tool.App),
		},
		1<<20,
	)
}

func (e *SessionToolExecutor) EvaluateTool(
	ctx context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolEvaluation, error) {
	if len(request.Authorization) != 0 {
		return sessionRuntime.ToolEvaluation{
			Decision: sessionRuntime.ToolEvaluationAllowed,
			Authorization: append(
				json.RawMessage(nil),
				request.Authorization...,
			),
		}, nil
	}

	binding, invokeRequest, err := decodeMCPToolRequest(request)
	if err != nil {
		return sessionRuntime.ToolEvaluation{}, err
	}

	evaluation, err := e.bridge.EvaluateMapped(
		ctx,
		binding,
		invokeRequest,
	)
	if err != nil {
		return sessionRuntime.ToolEvaluation{}, err
	}
	if evaluation == nil {
		return sessionRuntime.ToolEvaluation{}, errors.New(
			"MCP invocation bridge returned no evaluation",
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
		if strings.TrimSpace(evaluation.ApprovalID) == "" {
			return sessionRuntime.ToolEvaluation{}, errors.New(
				"MCP invocation bridge required approval without an approval identity",
			)
		}

		summary, err := canonicalMCPJSON(evaluation.Summary)
		if err != nil {
			return sessionRuntime.ToolEvaluation{}, err
		}

		var expiresAt *time.Time
		if strings.TrimSpace(evaluation.ExpiresAt) != "" {
			value, err := time.Parse(
				time.RFC3339Nano,
				evaluation.ExpiresAt,
			)
			if err != nil {
				return sessionRuntime.ToolEvaluation{}, fmt.Errorf(
					"decode MCP approval expiration: %w",
					err,
				)
			}
			value = value.UTC()
			expiresAt = &value
		}

		return sessionRuntime.ToolEvaluation{
			Decision: sessionRuntime.ToolEvaluationWait,
			Reason:   evaluation.Reason,
			Wait: &sessionRuntime.ToolWaitSpec{
				Kind:    sessionRuntime.WaitKindApproval,
				Summary: summary,
				AllowedResolutions: []sessionRuntime.WaitResolution{
					sessionRuntime.WaitResolutionApproveOnce,
					sessionRuntime.WaitResolutionApproveAlways,
					sessionRuntime.WaitResolutionDenyOnce,
					sessionRuntime.WaitResolutionDenyAlways,
				},
				ExpiresAt:         expiresAt,
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

func (e *SessionToolExecutor) ExecuteTool(
	ctx context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolExecutionResult, error) {
	binding, invokeRequest, err := decodeMCPToolRequest(request)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}

	var authorization struct {
		ApprovalID string `json:"approvalID,omitempty"`
		Token      string `json:"token,omitempty"`
	}
	if len(request.Authorization) != 0 {
		authorization, err = jsonutil.DecodeJSONRaw[struct {
			ApprovalID string `json:"approvalID,omitempty"`
			Token      string `json:"token,omitempty"`
		}](request.Authorization)
		if err != nil {
			return sessionRuntime.ToolExecutionResult{}, fmt.Errorf(
				"decode Session MCP authorization: %w",
				err,
			)
		}
	}

	invokeRequest.ApprovalID = authorization.ApprovalID
	invokeRequest.ApprovalToken = authorization.Token

	response, err := e.bridge.InvokeMapped(
		ctx,
		binding,
		invokeRequest,
	)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}
	if response == nil {
		return sessionRuntime.ToolExecutionResult{}, errors.New(
			"MCP invocation bridge returned no response",
		)
	}

	structured, err := canonicalMCPJSON(struct {
		Content           []mcpServer.MCPContent `json:"content"`
		StructuredContent any                    `json:"structuredContent,omitempty"`
	}{
		Content:           response.Content,
		StructuredContent: response.StructuredContent,
	})
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}

	app, err := canonicalMCPJSON(response.App)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}

	result := sessionRuntime.ToolExecutionResult{
		HasResult:  true,
		Kind:       sessionRuntime.ToolResultKindSuccess,
		Text:       string(structured),
		Contents:   ToolOutputItemsFromMCPContents(response.Content),
		Structured: structured,
		App:        app,
	}
	if response.App != nil {
		resourceURI := strings.TrimSpace(response.App.ResourceURI)
		if resourceURI == "" {
			resourceURI = binding.AppResourceURI
		}
		result.AppRoute = &sessionRuntime.AppRoute{
			ServerID:    string(binding.Server),
			ResourceURI: resourceURI,
			Binding:     cloneSessionToolBinding(request.ToolCall.Binding),
		}
	}

	if response.IsError {
		result.Kind = sessionRuntime.ToolResultKindError
		result.Error = firstNonBlank(
			string(structured),
			"MCP Tool returned an error result",
		)
	}

	return result, nil
}

func (e *SessionToolExecutor) ResolveToolWait(
	ctx context.Context,
	request sessionRuntime.ToolWaitResolutionRequest,
) (sessionRuntime.ToolWaitResolution, error) {
	if strings.TrimSpace(request.ExecutorReference) == "" {
		return sessionRuntime.ToolWaitResolution{}, errors.New(
			"MCP approval wait has no runtime approval reference",
		)
	}

	resolution, err := approvalResolution(request.Resolution)
	if err != nil {
		return sessionRuntime.ToolWaitResolution{}, err
	}

	result, err := e.bridge.ResolveApproval(
		ctx,
		request.ExecutorReference,
		resolution,
	)
	if err != nil {
		if errors.Is(err, mcpServer.ErrMCPInvalidRuntimeRequest) ||
			errors.Is(err, mcpServer.ErrMCPApprovalNeeded) {
			message := "MCP approval expired or is no longer valid"
			return sessionRuntime.ToolWaitResolution{
				Status: sessionRuntime.ToolWaitResolutionDenied,
				Result: &sessionRuntime.ToolExecutionResult{
					HasResult: true,
					Kind:      sessionRuntime.ToolResultKindError,
					Text:      failureText(message),
					Error:     message,
				},
			}, nil
		}
		return sessionRuntime.ToolWaitResolution{}, err
	}

	switch result.Decision {
	case mcpServer.MCPApprovalDecisionAllowed:
		authorization, err := canonicalMCPJSON(struct {
			ApprovalID string `json:"approvalID"`
			Token      string `json:"token"`
		}{
			ApprovalID: result.ApprovalID,
			Token:      result.Token,
		})
		if err != nil {
			return sessionRuntime.ToolWaitResolution{}, err
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
				Text:      failureText("MCP Tool call was denied"),
				Error:     "MCP Tool call was denied",
			},
		}, nil

	default:
		return sessionRuntime.ToolWaitResolution{}, fmt.Errorf(
			"unsupported MCP approval resolution decision %q",
			result.Decision,
		)
	}
}

func decodeMCPToolRequest(
	request sessionRuntime.ToolExecutionRequest,
) (
	invocation.MappedToolBinding,
	mcpServer.InvokeMCPToolRequestBody,
	error,
) {
	binding, err := jsonutil.DecodeJSONRaw[invocation.MappedToolBinding](request.ToolCall.Binding.Data)
	if err != nil {
		return invocation.MappedToolBinding{},
			mcpServer.InvokeMCPToolRequestBody{},
			fmt.Errorf("decode MCP Session binding: %w", err)
	}
	if err := binding.Validate(); err != nil {
		return invocation.MappedToolBinding{},
			mcpServer.InvokeMCPToolRequestBody{},
			err
	}

	rawArguments := bytes.TrimSpace(request.ToolCall.Arguments)
	if len(rawArguments) == 0 {
		rawArguments = []byte(`{}`)
	}
	canonicalArguments, err := jsonutil.Canonicalize(rawArguments)
	if err != nil {
		return invocation.MappedToolBinding{},
			mcpServer.InvokeMCPToolRequestBody{},
			fmt.Errorf("decode MCP Tool arguments: %w", err)
	}

	arguments, err := jsonutil.DecodeJSONRaw[map[string]any](
		json.RawMessage(canonicalArguments),
	)
	if err != nil || arguments == nil {
		return invocation.MappedToolBinding{},
			mcpServer.InvokeMCPToolRequestBody{},
			errors.New("MCP Tool arguments must be a JSON object")
	}

	source := mcpServer.MCPInvocationSourceModel
	if request.ToolCall.Source == sessionRuntime.ToolCallSourceApp {
		source = mcpServer.MCPInvocationSourceApp
	}

	return binding, mcpServer.InvokeMCPToolRequestBody{
		Source:           source,
		ToolName:         binding.ToolName,
		ProviderToolName: binding.ProviderToolName,
		ChoiceID:         binding.ChoiceID,
		ToolDigest:       binding.ToolDigest,
		Arguments:        arguments,
		ToolUseID: firstNonBlank(
			request.ToolCall.ProviderCallID,
			string(request.ToolCall.ID),
		),
		AppInstanceID: string(request.ToolCall.AppInstanceID),
	}, nil
}

func approvalResolution(
	resolution sessionRuntime.WaitResolution,
) (mcpServer.MCPApprovalResolution, error) {
	switch resolution {
	case sessionRuntime.WaitResolutionApproveOnce:
		return mcpServer.MCPApprovalResolutionAllowOnce, nil
	case sessionRuntime.WaitResolutionApproveAlways:
		return mcpServer.MCPApprovalResolutionAllowAlways, nil
	case sessionRuntime.WaitResolutionDenyOnce:
		return mcpServer.MCPApprovalResolutionDenyOnce, nil
	case sessionRuntime.WaitResolutionDenyAlways:
		return mcpServer.MCPApprovalResolutionDenyAlways, nil
	default:
		return "", fmt.Errorf(
			"unsupported MCP approval wait resolution %q",
			resolution,
		)
	}
}

func canonicalMCPJSON(value any) (json.RawMessage, error) {
	raw, err := jsonutil.EncodeToJSONRaw(value)
	if err != nil {
		return nil, err
	}

	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(canonical), nil
}

func cloneSessionToolBinding(
	value sessionRuntime.ToolBinding,
) sessionRuntime.ToolBinding {
	output := value
	output.Data = append(json.RawMessage(nil), value.Data...)
	return output
}

func appResourceURI(info *mcpServer.MCPToolAppInfo) string {
	if info == nil {
		return ""
	}
	return info.ResourceURI
}

func appVisibility(info *mcpServer.MCPToolAppInfo) []string {
	if info == nil {
		return nil
	}
	return append([]string(nil), info.Visibility...)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func failureText(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "MCP policy denied this Tool call."
	}

	return strings.Join([]string{
		"MCP Tool error: " + message,
		"This result completes the requested Tool call.",
		"The model may correct the request or retry the Tool.",
	}, "\n\n")
}
