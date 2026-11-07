package mcpruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	mcpApps "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/apps"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/connection"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/invocation"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	mcpWorkflowPorts "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/workflowports"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/toolruntime"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/mcppolicy"
)

// V3Executor owns MCP Tool discovery adaptation, policy evaluation, approval
// waits, and invocation for Session Runtime v3. Session retains only durable
// WorkItem state and generic wait orchestration.
type V3Executor struct {
	runtime *mcpConnection.MCPRuntimeManager
	bridge  *invocation.ToolBridge
}

type v3MCPExecution struct {
	Call              sessionv3Runtime.ToolCall
	ApprovalID        string
	ApprovalToken     string
	TargetFingerprint string
}

type v3MCPWait struct {
	Call              sessionv3Runtime.ToolCall
	Mapping           invocation.MappedToolBinding
	Request           mcpServer.InvokeMCPToolRequestBody
	ApprovalID        string
	TargetFingerprint string
}

func NewV3Executor(
	runtime *mcpConnection.MCPRuntimeManager,
	bridge *invocation.ToolBridge,
) (*V3Executor, error) {
	if runtime == nil || bridge == nil {
		return nil, errors.New("session v3 MCP runtime dependencies are incomplete")
	}
	return &V3Executor{
		runtime: runtime,
		bridge:  bridge,
	}, nil
}

// Context materializes selected MCP instructions, resources, resource
// templates, and prompts through the MCP domain. Session receives only text
// contributions prepared at this domain boundary.
func (e *V3Executor) Context(
	ctx context.Context,
	selections []sessionv3Runtime.MCPServerSelection,
) ([]string, error) {
	if len(selections) == 0 {
		return nil, nil
	}
	if e == nil || e.runtime == nil {
		return nil, errors.New("session v3 mcp runtime is unavailable")
	}

	output := make([]string, 0)
	for index := range selections {
		selection := selections[index]
		serverID, _, err := v3MCPServerSelection(selection)
		if err != nil {
			return nil, err
		}

		status, err := e.runtime.Status(ctx, serverID)
		if err != nil {
			return nil, err
		}
		if status == nil || status.Status != mcpServer.MCPServerStatusReady {
			return nil, fmt.Errorf("mcp server %q is not connected", serverID)
		}

		label := "MCP server"
		if status.ServerInfo != nil &&
			strings.TrimSpace(status.ServerInfo.Name) != "" {
			label = strings.TrimSpace(status.ServerInfo.Name)
		}
		if selection.IncludeServerInstructions &&
			strings.TrimSpace(status.Instructions) != "" {
			output = append(output, strings.Join([]string{
				"### MCP server instructions",
				"server: " + label,
				status.Instructions,
			}, "\n\n"))
		}

		if err := e.appendResourceContext(
			ctx,
			&output,
			serverID,
			label,
			selection.Resources,
		); err != nil {
			return nil, err
		}
		if err := e.appendResourceTemplateContext(
			ctx,
			&output,
			serverID,
			label,
			selection.ResourceTemplates,
		); err != nil {
			return nil, err
		}
		if err := e.appendPromptContext(
			ctx,
			&output,
			serverID,
			label,
			selection.Prompts,
		); err != nil {
			return nil, err
		}
	}
	return output, nil
}

// Choices adapts selected ready MCP servers into inference choices and durable
// bindings. A disconnected server is an error rather than a silently omitted
// capability.
func (e *V3Executor) Choices(
	ctx context.Context,
	selections []sessionv3Runtime.MCPServerSelection,
) ([]toolruntime.V3PreparedChoice, error) {
	if len(selections) == 0 {
		return nil, nil
	}
	if e == nil || e.runtime == nil {
		return nil, errors.New("session v3 MCP runtime is unavailable")
	}

	output := make([]toolruntime.V3PreparedChoice, 0)
	for index := range selections {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		selection := selections[index]
		serverID, pin, err := v3MCPServerSelection(selection)
		if err != nil {
			return nil, fmt.Errorf("selected MCP server %d: %w", index, err)
		}

		status, err := e.runtime.Status(ctx, serverID)
		if err != nil {
			return nil, err
		}
		if status == nil || status.Status != mcpServer.MCPServerStatusReady {
			return nil, fmt.Errorf("MCP server %q is not connected", serverID)
		}

		available, err := e.runtime.ListTools(ctx, serverID)
		if err != nil {
			return nil, err
		}
		selected, err := v3MCPSelectedTools(available, selection)
		if err != nil {
			return nil, err
		}

		sort.Slice(selected, func(left, right int) bool {
			return selected[left].ProviderToolName <
				selected[right].ProviderToolName
		})

		for _, tool := range selected {
			schema := jsonutil.CloneJSONInValue(tool.InputSchema)
			if len(schema) == 0 {
				schema = map[string]any{"type": "object"}
			}

			choice := inferenceSpec.ToolChoice{
				Type:        inferenceSpec.ToolTypeFunction,
				ID:          tool.ChoiceID,
				Name:        tool.ProviderToolName,
				Description: v3MCPToolDescription(tool),
				Arguments:   schema,
			}
			call := sessionv3Runtime.ToolCall{
				Name:     choice.Name,
				ChoiceID: choice.ID,
				DisplayName: firstNonBlank(
					tool.DisplayName,
					tool.ToolName,
					"MCP Tool",
				),
				Description: strings.TrimSpace(tool.Description),
				Automatic: tool.ExecutionMode ==
					mcppolicy.MCPExecutionModeAuto,
				Binding: sessionv3Runtime.ToolBinding{
					Kind: sessionv3Runtime.ToolBindingKindMCP,
					MCP: &sessionv3Runtime.MCPToolBinding{
						ServerArtifact:   pin,
						ServerID:         string(serverID),
						ProviderToolName: tool.ProviderToolName,
						ChoiceID:         tool.ChoiceID,
						ToolName:         tool.ToolName,
						ToolDigest:       tool.Digest,
						ApprovalRule:     string(tool.ApprovalRule),
						ExecutionMode:    string(tool.ExecutionMode),
					},
				},
			}
			output = append(output, toolruntime.V3PreparedChoice{
				Choice: choice,
				Call:   call,
			})
		}
	}
	return output, nil
}

// BindWorkflowTool resolves one Workflow MCP Tool target to the current
// discovered provider Tool identity. The Workflow binder verifies the
// persisted Server Artifact pin before this method runs; this method verifies
// the current MCP runtime identity, policy visibility, and discovery state.
func (e *V3Executor) BindWorkflowTool(
	ctx context.Context,
	payload *sessionv3Runtime.ToolWork,
) (sessionv3Runtime.ToolCall, error) {
	if e == nil || e.runtime == nil {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"session v3 MCP runtime is unavailable",
		)
	}
	if payload == nil || payload.MCP == nil {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"workflow MCP tool target is missing",
		)
	}
	if payload.MCP.Server.RootID == "" ||
		payload.MCP.Server.ArtifactID == "" ||
		payload.MCP.Server.Revision == 0 ||
		strings.TrimSpace(payload.MCP.Server.Digest) == "" {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"workflow MCP server artifact pin is incomplete",
		)
	}

	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(payload.MCP.Server.RootID),
		ArtifactID: artifactModel.ArtifactID(payload.MCP.Server.ArtifactID),
	}
	if err := ref.Validate(); err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}

	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}
	status, err := e.runtime.Status(ctx, serverID)
	if err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}
	if status == nil || status.Status != mcpServer.MCPServerStatusReady {
		return sessionv3Runtime.ToolCall{}, fmt.Errorf(
			"workflow MCP server %q is not connected",
			serverID,
		)
	}

	tools, err := e.runtime.ListTools(ctx, serverID)
	if err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}

	var selected *mcpServer.MCPToolCapability
	for index := range tools {
		if tools[index].ToolName == payload.MCP.ToolName {
			selected = &tools[index]
			break
		}
	}
	if selected == nil ||
		selected.Server != serverID ||
		!v3MCPToolAvailable(*selected) {
		return sessionv3Runtime.ToolCall{}, fmt.Errorf(
			"workflow MCP tool %q is unavailable",
			payload.MCP.ToolName,
		)
	}

	call := payload.Call
	if call.Name != "" && call.Name != payload.MCP.ToolName {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"workflow MCP tool name changed since Workflow admission",
		)
	}
	serverPin := payload.MCP.Server
	call.Name = selected.ProviderToolName
	call.ChoiceID = selected.ChoiceID
	call.DisplayName = firstNonBlank(
		selected.DisplayName,
		selected.ToolName,
		"MCP Tool",
	)
	call.Description = strings.TrimSpace(selected.Description)
	call.Automatic = selected.ExecutionMode ==
		mcppolicy.MCPExecutionModeAuto
	call.Binding = sessionv3Runtime.ToolBinding{
		Kind: sessionv3Runtime.ToolBindingKindMCP,
		MCP: &sessionv3Runtime.MCPToolBinding{
			ServerArtifact:   &serverPin,
			ServerID:         string(serverID),
			ProviderToolName: selected.ProviderToolName,
			ChoiceID:         selected.ChoiceID,
			ToolName:         selected.ToolName,
			ToolDigest:       selected.Digest,
			ApprovalRule:     string(selected.ApprovalRule),
			ExecutionMode:    string(selected.ExecutionMode),
		},
	}
	return call, nil
}

// Bind handles MCP Tool evaluation. The private values returned here remain in
// Session Runtime memory and are never written to checkpoints.
func (e *V3Executor) Bind(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
) (sessionv3Runtime.BindResult, bool, error) {
	payload := request.WorkItem.Input.Tool
	if payload == nil ||
		payload.Call.Binding.Kind != sessionv3Runtime.ToolBindingKindMCP {
		return sessionv3Runtime.BindResult{}, false, nil
	}

	call := payload.Call
	if err := call.Validate(); err != nil {
		return sessionv3Runtime.BindResult{}, true, err
	}

	mapping, err := v3MCPMapping(payload)
	if err != nil {
		return sessionv3Runtime.BindResult{}, true, err
	}
	portRequest := v3MCPWorkflowPortRequest(
		request,
		payload,
	)
	if portRequest != nil {
		if err := e.requireWorkflowMCPPortExtension(
			ctx,
			mapping.Server,
		); err != nil {
			return sessionv3Runtime.BindResult{}, true, err
		}
	}
	mapping, invokeRequest, err := v3MappedMCPRequest(payload, portRequest)
	if err != nil {
		return sessionv3Runtime.BindResult{}, true, err
	}
	evaluation, err := e.bridge.EvaluateMapped(
		ctx,
		mapping,
		invokeRequest,
	)
	if err != nil {
		return sessionv3Runtime.BindResult{}, true, err
	}
	if evaluation == nil {
		return sessionv3Runtime.BindResult{}, true, errors.New(
			"MCP approval evaluation returned no result",
		)
	}

	fingerprint := string(mcpServer.DigestBytes([]byte(
		string(mapping.Server) + "\x00" +
			mapping.ToolName + "\x00" +
			mapping.ToolDigest + "\x00" +
			call.ID,
	)))
	semantic := sessionv3Runtime.SemanticBinding{
		Version:           "session-v3-mcp/v1",
		TargetFingerprint: fingerprint,
		BoundAt:           time.Now().UTC(),
	}
	if call.Binding.MCP.ServerArtifact != nil {
		semantic.Artifacts = []sessionv3Runtime.ArtifactPin{
			*call.Binding.MCP.ServerArtifact,
		}
	}

	switch evaluation.Decision {
	case mcpServer.MCPApprovalDecisionAllowed:
		return sessionv3Runtime.BindResult{
			SemanticBinding: semantic,
			Private: v3MCPExecution{
				Call:              call,
				TargetFingerprint: fingerprint,
			},
		}, true, nil

	case mcpServer.MCPApprovalDecisionDenied:
		message := v3MCPFailureText(evaluation.Reason)
		return sessionv3Runtime.BindResult{
			SemanticBinding: semantic,
			ImmediateOutput: &sessionv3Runtime.WorkItemOutput{
				Tool: &sessionv3Runtime.ToolOutput{
					Kind:  sessionv3Runtime.ToolOutputKindError,
					Text:  message,
					Error: message,
				},
			},
		}, true, nil

	case mcpServer.MCPApprovalDecisionApprovalRequired:
		if evaluation.Summary == nil ||
			strings.TrimSpace(evaluation.ApprovalID) == "" {
			return sessionv3Runtime.BindResult{}, true, errors.New(
				"MCP approval evaluation has no approval identity or summary",
			)
		}

		summary, err := canonicalV3MCPValue(evaluation.Summary)
		if err != nil {
			return sessionv3Runtime.BindResult{}, true, err
		}
		deadline, err := parseV3MCPDeadline(evaluation.ExpiresAt)
		if err != nil {
			return sessionv3Runtime.BindResult{}, true, err
		}

		wait := &sessionv3Runtime.WaitRequest{
			Kind:    sessionv3Runtime.WaitKindApproval,
			Summary: summary,
			AllowedResolutions: []sessionv3Runtime.WaitResolution{
				sessionv3Runtime.WaitResolutionApproveOnce,
				sessionv3Runtime.WaitResolutionApproveAlways,
				sessionv3Runtime.WaitResolutionDenyOnce,
				sessionv3Runtime.WaitResolutionDenyAlways,
			},
			Deadline:     deadline,
			RuntimeBound: true,
		}
		return sessionv3Runtime.BindResult{
			SemanticBinding: semantic,
			Wait:            wait,
			Private: v3MCPWait{
				Call:              call,
				Mapping:           mapping,
				Request:           invokeRequest,
				ApprovalID:        evaluation.ApprovalID,
				TargetFingerprint: fingerprint,
			},
		}, true, nil

	default:
		return sessionv3Runtime.BindResult{}, true, fmt.Errorf(
			"unsupported MCP approval decision %q",
			evaluation.Decision,
		)
	}
}

// Execute handles a private binding returned by Bind.
func (e *V3Executor) Execute(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
	bound sessionv3Runtime.BoundWork,
) (sessionv3Runtime.WorkExecutionResult, bool, error) {
	private, found := bound.Private.(v3MCPExecution)
	if !found {
		return sessionv3Runtime.WorkExecutionResult{}, false, nil
	}

	payload := request.WorkItem.Input.Tool
	if payload == nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "invalid_mcp_binding",
				Message: "MCP Tool payload is missing",
			},
		}, true, nil
	}

	effective := *payload
	effective.Call = private.Call
	portRequest := v3MCPWorkflowPortRequest(
		request,
		&effective,
	)
	mapping, invokeRequest, err := v3MappedMCPRequest(&effective, portRequest)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "invalid_mcp_binding",
				Message: err.Error(),
			},
		}, true, nil
	}
	invokeRequest.ApprovalID = private.ApprovalID
	invokeRequest.ApprovalToken = private.ApprovalToken

	response, err := e.bridge.InvokeMapped(ctx, mapping, invokeRequest)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "mcp_failed",
				Message: err.Error(),
			},
		}, true, nil
	}
	if response == nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "mcp_empty",
				Message: "MCP Tool returned no response",
			},
		}, true, nil
	}

	content, err := canonicalV3MCPValue(response.Content)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{}, true, err
	}
	structured, err := canonicalV3MCPValue(struct {
		Content           []mcpServer.MCPContent          `json:"content"`
		StructuredContent any                             `json:"structuredContent,omitempty"`
		App               *mcpServer.MCPToolAppRenderInfo `json:"app,omitempty"`
	}{
		Content:           response.Content,
		StructuredContent: response.StructuredContent,
		App:               response.App,
	})
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{}, true, err
	}

	text := v3MCPContentsText(response.Content)
	if strings.TrimSpace(text) == "" {
		text = string(content)
	}
	kind := sessionv3Runtime.ToolOutputKindSuccess
	errorText := ""
	if response.IsError {
		kind = sessionv3Runtime.ToolOutputKindError
		errorText = firstNonBlank(
			text,
			"MCP Tool returned an error result",
		)
	}

	return sessionv3Runtime.WorkExecutionResult{
		Output: &sessionv3Runtime.WorkItemOutput{
			Value: structured,
			Tool: &sessionv3Runtime.ToolOutput{
				Kind:       kind,
				Text:       text,
				Structured: structured,
				Error:      errorText,
			},
			Receipt: &sessionv3Runtime.ExternalReceipt{
				System:      "mcp",
				Reference:   private.TargetFingerprint,
				ConfirmedAt: time.Now().UTC(),
			},
		},
		Confirmed: true,
	}, true, nil
}

// ResolveWait translates a Session v3 approval resolution into the MCP
// approval protocol and returns a resumed private execution binding.
func (e *V3Executor) ResolveWait(
	ctx context.Context,
	request sessionv3Runtime.WaitResolutionRequest,
) (sessionv3Runtime.WaitResolutionResult, bool, error) {
	private, found := request.Private.(v3MCPWait)
	if !found {
		return sessionv3Runtime.WaitResolutionResult{}, false, nil
	}

	resolution, err := v3MCPWaitResolution(request.Resolution)
	if err != nil {
		return sessionv3Runtime.WaitResolutionResult{}, true, err
	}
	result, err := e.bridge.ResolveApproval(
		ctx,
		private.ApprovalID,
		resolution,
	)
	if err != nil {
		if errors.Is(err, mcpServer.ErrMCPInvalidRuntimeRequest) ||
			errors.Is(err, mcpServer.ErrMCPApprovalNeeded) {
			message := "MCP approval expired or is no longer valid"
			return sessionv3Runtime.WaitResolutionResult{
				Output: &sessionv3Runtime.WorkItemOutput{
					Tool: &sessionv3Runtime.ToolOutput{
						Kind:  sessionv3Runtime.ToolOutputKindError,
						Text:  message,
						Error: message,
					},
				},
				Confirmed: true,
			}, true, nil
		}
		return sessionv3Runtime.WaitResolutionResult{}, true, err
	}

	switch result.Decision {
	case mcpServer.MCPApprovalDecisionDenied:
		return sessionv3Runtime.WaitResolutionResult{
			Output: &sessionv3Runtime.WorkItemOutput{
				Tool: &sessionv3Runtime.ToolOutput{
					Kind:  sessionv3Runtime.ToolOutputKindError,
					Text:  toolCallDenied,
					Error: toolCallDenied,
				},
			},
			Confirmed: true,
		}, true, nil

	case mcpServer.MCPApprovalDecisionAllowed:
		semantic := sessionv3Runtime.SemanticBinding{
			Version:           "session-v3-mcp-resume/v1",
			TargetFingerprint: private.TargetFingerprint,
			BoundAt:           time.Now().UTC(),
		}
		if private.Call.Binding.MCP != nil &&
			private.Call.Binding.MCP.ServerArtifact != nil {
			semantic.Artifacts = []sessionv3Runtime.ArtifactPin{
				*private.Call.Binding.MCP.ServerArtifact,
			}
		}
		if request.Resolution ==
			sessionv3Runtime.WaitResolutionApproveOnce &&
			strings.TrimSpace(result.Token) == "" {
			return sessionv3Runtime.WaitResolutionResult{}, true, errors.New(
				"MCP one-time approval returned no approval token",
			)
		}

		return sessionv3Runtime.WaitResolutionResult{
			Resume: true,
			Bound: &sessionv3Runtime.BoundWork{
				SemanticBinding: semantic,
				Private: v3MCPExecution{
					Call:              private.Call,
					ApprovalID:        result.ApprovalID,
					ApprovalToken:     result.Token,
					TargetFingerprint: private.TargetFingerprint,
				},
			},
		}, true, nil

	default:
		return sessionv3Runtime.WaitResolutionResult{}, true, fmt.Errorf(
			"unsupported MCP approval resolution decision %q",
			result.Decision,
		)
	}
}

func (e *V3Executor) appendResourceContext(
	ctx context.Context,
	output *[]string,
	serverID mcpServer.ServerID,
	label string,
	selections []sessionv3Runtime.MCPResourceSelection,
) error {
	if len(selections) == 0 {
		return nil
	}

	available, err := e.runtime.ListResources(ctx, serverID)
	if err != nil {
		return err
	}
	byURI := make(map[string]mcpServer.MCPResourceRef, len(available))
	for _, resource := range available {
		byURI[resource.URI] = resource
	}

	for _, selection := range selections {
		resource, found := byURI[selection.URI]
		if !found {
			return fmt.Errorf(
				"selected mcp resource %q is unavailable",
				selection.URI,
			)
		}
		if selection.ExpectedDigest != "" &&
			selection.ExpectedDigest != resource.Digest {
			return fmt.Errorf(
				"selected mcp resource %q changed since selection",
				selection.URI,
			)
		}

		response, err := e.runtime.ReadResource(
			ctx,
			serverID,
			selection.URI,
		)
		if err != nil {
			return err
		}
		if response == nil {
			return errors.New("mcp resource read returned no response")
		}
		text := v3MCPContentsText(response.Contents)
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf(
				"selected mcp resource %q returned no textual context",
				selection.URI,
			)
		}
		*output = append(*output, strings.Join([]string{
			"### MCP resource",
			"server: " + label,
			"uri: " + selection.URI,
			text,
		}, "\n\n"))
	}
	return nil
}

func (e *V3Executor) appendResourceTemplateContext(
	ctx context.Context,
	output *[]string,
	serverID mcpServer.ServerID,
	label string,
	selections []sessionv3Runtime.MCPResourceTemplateSelection,
) error {
	if len(selections) == 0 {
		return nil
	}

	available, err := e.runtime.ListResourceTemplates(ctx, serverID)
	if err != nil {
		return err
	}
	byTemplate := make(
		map[string]mcpServer.MCPResourceTemplateRef,
		len(available),
	)
	for _, resource := range available {
		byTemplate[resource.URITemplate] = resource
	}

	for _, selection := range selections {
		template, found := byTemplate[selection.URITemplate]
		if !found {
			return fmt.Errorf(
				"selected mcp resource template %q is unavailable",
				selection.URITemplate,
			)
		}
		if selection.ExpectedDigest != "" &&
			selection.ExpectedDigest != template.Digest {
			return fmt.Errorf(
				"selected mcp resource template %q changed since selection",
				selection.URITemplate,
			)
		}

		uri, err := v3MCPResourceTemplateURI(
			selection.URITemplate,
			selection.Arguments,
		)
		if err != nil {
			return err
		}
		response, err := e.runtime.ReadResource(ctx, serverID, uri)
		if err != nil {
			return err
		}
		if response == nil {
			return errors.New("mcp resource template read returned no response")
		}
		text := v3MCPContentsText(response.Contents)
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf(
				"selected mcp resource template %q returned no textual context",
				selection.URITemplate,
			)
		}
		*output = append(*output, strings.Join([]string{
			"### MCP resource template",
			"server: " + label,
			"uri: " + uri,
			text,
		}, "\n\n"))
	}
	return nil
}

func (e *V3Executor) appendPromptContext(
	ctx context.Context,
	output *[]string,
	serverID mcpServer.ServerID,
	label string,
	selections []sessionv3Runtime.MCPPromptSelection,
) error {
	if len(selections) == 0 {
		return nil
	}

	available, err := e.runtime.ListPrompts(ctx, serverID)
	if err != nil {
		return err
	}
	byName := make(map[string]mcpServer.MCPPromptRef, len(available))
	for _, prompt := range available {
		byName[prompt.PromptName] = prompt
	}

	for _, selection := range selections {
		prompt, found := byName[selection.PromptName]
		if !found {
			return fmt.Errorf(
				"selected mcp prompt %q is unavailable",
				selection.PromptName,
			)
		}
		if selection.ExpectedDigest != "" &&
			selection.ExpectedDigest != prompt.Digest {
			return fmt.Errorf(
				"selected mcp prompt %q changed since selection",
				selection.PromptName,
			)
		}

		response, err := e.runtime.GetPrompt(
			ctx,
			serverID,
			selection.PromptName,
			selection.Arguments,
		)
		if err != nil {
			return err
		}
		if response == nil {
			return errors.New("mcp prompt read returned no response")
		}

		parts := make([]string, 0, len(response.Messages))
		for _, message := range response.Messages {
			text := v3MCPContentText(message.Content)
			if strings.TrimSpace(text) != "" {
				parts = append(parts, message.Role+":\n"+text)
			}
		}
		if len(parts) == 0 {
			return fmt.Errorf(
				"selected mcp prompt %q returned no textual context",
				selection.PromptName,
			)
		}
		*output = append(*output, strings.Join([]string{
			"### MCP prompt",
			"server: " + label,
			"name: " + selection.PromptName,
			strings.Join(parts, "\n\n"),
		}, "\n\n"))
	}
	return nil
}

func v3MCPResourceTemplateURI(
	template string,
	arguments map[string]string,
) (string, error) {
	parsed, err := url.Parse(template)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New(
			"mcp resource template must be an absolute uri",
		)
	}

	for name, value := range arguments {
		template = strings.ReplaceAll(
			template,
			"{"+name+"}",
			url.PathEscape(value),
		)
	}
	if strings.Contains(template, "{") || strings.Contains(template, "}") {
		return "", errors.New("mcp resource template arguments are incomplete")
	}
	return template, nil
}

func v3MCPServerSelection(
	selection sessionv3Runtime.MCPServerSelection,
) (mcpServer.ServerID, *sessionv3Runtime.ArtifactPin, error) {
	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(selection.Server.RootID),
		ArtifactID: artifactModel.ArtifactID(selection.Server.ArtifactID),
	}
	if err := ref.Validate(); err != nil {
		return "", nil, err
	}

	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return "", nil, err
	}

	if selection.Server.ExpectedRevision == 0 ||
		strings.TrimSpace(selection.Server.ExpectedDigest) == "" {
		return serverID, nil, nil
	}

	return serverID, &sessionv3Runtime.ArtifactPin{
		RootID:     selection.Server.RootID,
		ArtifactID: selection.Server.ArtifactID,
		Revision:   selection.Server.ExpectedRevision,
		Digest:     selection.Server.ExpectedDigest,
	}, nil
}

func v3MCPSelectedTools(
	available []mcpServer.MCPToolCapability,
	selection sessionv3Runtime.MCPServerSelection,
) ([]mcpServer.MCPToolCapability, error) {
	exposure := selection.ToolExposure
	if exposure == "" {
		exposure = sessionv3Runtime.MCPToolExposureAll
	}

	switch exposure {
	case sessionv3Runtime.MCPToolExposureNone:
		return nil, nil

	case sessionv3Runtime.MCPToolExposureAll:
		output := make([]mcpServer.MCPToolCapability, 0, len(available))
		for _, tool := range available {
			if v3MCPToolAvailable(tool) {
				output = append(output, tool)
			}
		}
		return output, nil

	case sessionv3Runtime.MCPToolExposureSelected:
		byName := make(
			map[string]mcpServer.MCPToolCapability,
			len(available),
		)
		for _, tool := range available {
			byName[tool.ToolName] = tool
		}

		output := make(
			[]mcpServer.MCPToolCapability,
			0,
			len(selection.SelectedTools),
		)
		for _, requested := range selection.SelectedTools {
			tool, found := byName[requested.ToolName]
			if !found || !v3MCPToolAvailable(tool) {
				return nil, fmt.Errorf(
					"selected MCP Tool %q is unavailable",
					requested.ToolName,
				)
			}
			if requested.ExpectedDigest != "" &&
				requested.ExpectedDigest != tool.Digest {
				return nil, fmt.Errorf(
					"selected MCP Tool %q changed since selection",
					requested.ToolName,
				)
			}

			constrained, err := v3ConstrainMCPTool(tool, requested)
			if err != nil {
				return nil, err
			}
			output = append(output, constrained)
		}
		return output, nil

	default:
		return nil, fmt.Errorf(
			"unsupported MCP Tool exposure %q",
			selection.ToolExposure,
		)
	}
}

func v3MCPToolAvailable(
	tool mcpServer.MCPToolCapability,
) bool {
	return tool.Enabled &&
		tool.TaskSupport != mcpServer.MCPTaskSupportRequired &&
		mcpApps.ToolVisibleToModel(tool.App)
}

func v3ConstrainMCPTool(
	tool mcpServer.MCPToolCapability,
	selection sessionv3Runtime.MCPToolSelection,
) (mcpServer.MCPToolCapability, error) {
	output := tool

	if selection.ApprovalRule != nil {
		value, err := v3MCPApprovalRule(*selection.ApprovalRule)
		if err != nil {
			return mcpServer.MCPToolCapability{}, err
		}
		if mcppolicy.ApprovalRuleRank(value) <
			mcppolicy.ApprovalRuleRank(
				mcppolicy.NormalizedApprovalRule(tool.ApprovalRule),
			) {
			return mcpServer.MCPToolCapability{}, errors.New(
				"MCP approval override weakens the effective server policy",
			)
		}
		output.ApprovalRule = value
	}

	if selection.ExecutionMode != nil {
		value, err := v3MCPExecutionMode(*selection.ExecutionMode)
		if err != nil {
			return mcpServer.MCPToolCapability{}, err
		}
		if mcppolicy.ExecutionModeRank(value) <
			mcppolicy.ExecutionModeRank(
				mcppolicy.NormalizedExecutionMode(tool.ExecutionMode),
			) {
			return mcpServer.MCPToolCapability{}, errors.New(
				"MCP execution override weakens the effective server policy",
			)
		}
		output.ExecutionMode = value
	}
	return output, nil
}

func v3MCPApprovalRule(
	value string,
) (mcppolicy.MCPApprovalRule, error) {
	if strings.TrimSpace(value) != value {
		return "", errors.New("MCP approval override is not normalized")
	}

	rule := mcppolicy.MCPApprovalRule(value)
	switch rule {
	case mcppolicy.MCPApprovalRuleAllow,
		mcppolicy.MCPApprovalRuleAsk,
		mcppolicy.MCPApprovalRuleDeny:
		return rule, nil
	default:
		return "", errors.New("unsupported MCP approval override")
	}
}

func v3MCPExecutionMode(
	value string,
) (mcppolicy.MCPExecutionMode, error) {
	if strings.TrimSpace(value) != value {
		return "", errors.New("MCP execution override is not normalized")
	}

	mode := mcppolicy.MCPExecutionMode(value)
	switch mode {
	case mcppolicy.MCPExecutionModeAuto,
		mcppolicy.MCPExecutionModeManual:
		return mode, nil
	default:
		return "", errors.New("unsupported MCP execution override")
	}
}

func v3MCPMapping(
	payload *sessionv3Runtime.ToolWork,
) (invocation.MappedToolBinding, error) {
	if payload == nil || payload.Call.Binding.MCP == nil {
		return invocation.MappedToolBinding{}, errors.New(
			"mcp tool binding is missing",
		)
	}

	binding := payload.Call.Binding.MCP
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
		return invocation.MappedToolBinding{}, err
	}
	return mapping, nil
}

func v3MappedMCPRequest(
	payload *sessionv3Runtime.ToolWork,
	portRequest *mcpWorkflowPorts.Request,
) (
	invocation.MappedToolBinding,
	mcpServer.InvokeMCPToolRequestBody,
	error,
) {
	mapping, err := v3MCPMapping(payload)
	if err != nil {
		return invocation.MappedToolBinding{},
			mcpServer.InvokeMCPToolRequestBody{},
			err
	}

	arguments := payload.Call.Arguments.Clone()
	if len(arguments) == 0 {
		arguments = sessionv3Runtime.MustCanonicalValue(map[string]any{})
	}
	values, err := sessionv3Runtime.DecodeCanonicalValue[map[string]any](
		arguments,
	)
	if err != nil || values == nil {
		return invocation.MappedToolBinding{},
			mcpServer.InvokeMCPToolRequestBody{},
			errors.New("mcp tool arguments must be a json object")
	}

	return mapping, mcpServer.InvokeMCPToolRequestBody{
		Source:           mcpServer.MCPInvocationSourceModel,
		ToolName:         mapping.ToolName,
		ProviderToolName: mapping.ProviderToolName,
		ChoiceID:         mapping.ChoiceID,
		ToolDigest:       mapping.ToolDigest,
		Arguments:        values,
		ToolUseID: firstNonBlank(
			payload.Call.ProviderCallID,
			payload.Call.ID,
		),
		WorkflowPorts: mcpWorkflowPorts.CloneRequest(portRequest),
	}, nil
}

func (e *V3Executor) requireWorkflowMCPPortExtension(
	ctx context.Context,
	serverID mcpServer.ServerID,
) error {
	if e == nil || e.runtime == nil {
		return errors.New("session v3 mcp runtime is unavailable")
	}
	status, err := e.runtime.Status(ctx, serverID)
	if err != nil {
		return err
	}
	if status == nil || status.ServerCapabilities == nil {
		return errors.New("mcp server does not support workflow port transport")
	}
	if _, found := status.ServerCapabilities.Extensions[mcpWorkflowPorts.ExtensionID]; !found {
		return errors.New("mcp server does not support workflow port transport")
	}
	return nil
}

func v3MCPWorkflowPortRequest(
	request sessionv3Runtime.WorkDispatchRequest,
	payload *sessionv3Runtime.ToolWork,
) *mcpWorkflowPorts.Request {
	if payload == nil {
		return nil
	}
	output := &mcpWorkflowPorts.Request{
		Version: mcpWorkflowPorts.Version,
		Context: append(json.RawMessage(nil), payload.Context...),
	}
	for _, input := range request.WorkItem.PortInputs {
		value := mcpWorkflowPorts.PortInput{
			Name:     input.Name,
			Channel:  input.Channel,
			Cursor:   input.Cursor,
			Streamed: input.Streamed,
			Values:   make([]json.RawMessage, 0, len(input.Values)),
		}
		for _, current := range input.Values {
			value.Values = append(
				value.Values,
				json.RawMessage(current.Clone()),
			)
		}
		output.PortInputs = append(output.PortInputs, value)
	}
	for _, plan := range request.WorkItem.Ports {
		if plan.Direction != sessionv3Runtime.WorkflowPortDirectionOutput ||
			!plan.Stream {
			continue
		}
		output.OutputPorts = append(output.OutputPorts, plan.Name)
	}
	if len(output.Context) == 0 &&
		len(output.PortInputs) == 0 &&
		len(output.OutputPorts) == 0 {
		return nil
	}
	return output
}

func canonicalV3MCPValue(
	value any,
) (sessionv3Runtime.CanonicalValue, error) {
	raw, err := jsonutil.EncodeToJSONRaw(value)
	if err != nil {
		return nil, err
	}
	return sessionv3Runtime.CanonicalValueFrom(raw)
}

func parseV3MCPDeadline(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, fmt.Errorf("decode MCP approval expiration: %w", err)
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func v3MCPWaitResolution(
	value sessionv3Runtime.WaitResolution,
) (mcpServer.MCPApprovalResolution, error) {
	switch value {
	case sessionv3Runtime.WaitResolutionApproveOnce:
		return mcpServer.MCPApprovalResolutionAllowOnce, nil
	case sessionv3Runtime.WaitResolutionApproveAlways:
		return mcpServer.MCPApprovalResolutionAllowAlways, nil
	case sessionv3Runtime.WaitResolutionDenyOnce:
		return mcpServer.MCPApprovalResolutionDenyOnce, nil
	case sessionv3Runtime.WaitResolutionDenyAlways:
		return mcpServer.MCPApprovalResolutionDenyAlways, nil
	default:
		return "", fmt.Errorf("unsupported MCP wait resolution %q", value)
	}
}

func v3MCPToolDescription(
	tool mcpServer.MCPToolCapability,
) string {
	return firstNonBlank(
		strings.TrimSpace(tool.Description),
		tool.DisplayName,
		tool.ToolName,
	)
}

func v3MCPFailureText(reason string) string {
	return firstNonBlank(reason, "MCP policy denied the Tool call.")
}

func v3MCPContentsText(
	contents []mcpServer.MCPContent,
) string {
	parts := make([]string, 0, len(contents))
	for _, content := range contents {
		switch content.Type {
		case mcpServer.MCPContentTypeText:
			if strings.TrimSpace(content.Text) != "" {
				parts = append(parts, content.Text)
			}

		case mcpServer.MCPContentTypeResource:
			if content.Resource == nil {
				continue
			}
			if strings.TrimSpace(content.Resource.Text) != "" {
				parts = append(parts, content.Resource.Text)
			} else if strings.TrimSpace(content.Resource.URI) != "" {
				parts = append(
					parts,
					"[MCP resource omitted: "+content.Resource.URI+"]",
				)
			}

		case mcpServer.MCPContentTypeResourceLink:
			if value := strings.TrimSpace(
				content.Name + " " + content.URI,
			); value != "" {
				parts = append(parts, value)
			}

		case mcpServer.MCPContentTypeImage:
			parts = append(parts, "[MCP image content omitted]")

		case mcpServer.MCPContentTypeAudio:
			parts = append(parts, "[MCP audio content omitted]")
		}
	}
	return strings.Join(parts, "\n\n")
}

func v3MCPContentText(content mcpServer.MCPContent) string {
	switch content.Type {
	case mcpServer.MCPContentTypeText:
		return content.Text

	case mcpServer.MCPContentTypeResource:
		if content.Resource == nil {
			return ""
		}
		if strings.TrimSpace(content.Resource.Text) != "" {
			return content.Resource.Text
		}
		if strings.TrimSpace(content.Resource.URI) != "" {
			return "[mcp resource omitted: " + content.Resource.URI + "]"
		}

	case mcpServer.MCPContentTypeResourceLink:
		return strings.TrimSpace(content.Name + " " + content.URI)

	case mcpServer.MCPContentTypeImage:
		return "[mcp image content omitted]"

	case mcpServer.MCPContentTypeAudio:
		return "[mcp audio content omitted]"
	}
	return ""
}
