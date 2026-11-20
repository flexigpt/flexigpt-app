package mcpruntime

import (
	"context"
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
	pins    ArtifactPinVerifier
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

// ArtifactPinVerifier is owned by the MCP target domain. It supplies generic
// current source facts only. Workflow wrappers do not perform this read.
type ArtifactPinVerifier interface {
	VerifyArtifactPin(
		ctx context.Context,
		pin sessionv3Runtime.ArtifactPin,
	) error
}

func NewV3Executor(
	runtime *mcpConnection.MCPRuntimeManager,
	bridge *invocation.ToolBridge,
	pins ArtifactPinVerifier,
) (*V3Executor, error) {
	if runtime == nil || bridge == nil || pins == nil {
		return nil, errors.New("session v3 MCP runtime dependencies are incomplete")
	}
	return &V3Executor{
		runtime: runtime,
		bridge:  bridge,
		pins:    pins,
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
				ID:       "prepared-choice:" + choice.ID,
				Name:     choice.Name,
				ChoiceID: choice.ID,
				Type:     inferenceSpec.ToolTypeFunction,
				Source:   sessionv3Runtime.ToolCallSourceModel,
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
	if err := e.pins.VerifyArtifactPin(ctx, payload.MCP.Server); err != nil {
		return sessionv3Runtime.ToolCall{}, fmt.Errorf(
			"workflow MCP server freshness: %w",
			err,
		)
	}
	current, err := prepareWorkflowMCPTool(
		ctx,
		e.runtime,
		payload.MCP.Server,
		payload.MCP.ToolName,
	)
	if err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}
	if current != *payload.MCP {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"workflow MCP tool changed since preparation",
		)
	}

	call := payload.Call
	if call.Name != "" && call.Name != payload.MCP.ToolName {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"workflow MCP tool name changed since Workflow admission",
		)
	}
	serverID, serverIDErr := workflowMCPServerID(
		payload.MCP.Server,
	)
	if serverIDErr != nil {
		return sessionv3Runtime.ToolCall{}, serverIDErr
	}

	serverPin := payload.MCP.Server
	call.Name = payload.MCP.ProviderToolName
	call.Type = inferenceSpec.ToolTypeFunction
	call.ChoiceID = payload.MCP.ChoiceID
	call.DisplayName = firstNonBlank(
		payload.MCP.ToolName,
		"MCP Tool",
	)
	call.Automatic = mcppolicy.MCPExecutionMode(
		payload.MCP.ExecutionMode,
	) ==
		mcppolicy.MCPExecutionModeAuto
	call.Binding = sessionv3Runtime.ToolBinding{
		Kind: sessionv3Runtime.ToolBindingKindMCP,
		MCP: &sessionv3Runtime.MCPToolBinding{
			ServerArtifact:   &serverPin,
			ServerID:         string(serverID),
			ProviderToolName: payload.MCP.ProviderToolName,
			ChoiceID:         payload.MCP.ChoiceID,
			ToolName:         payload.MCP.ToolName,
			ToolDigest:       payload.MCP.ToolDigest,
			ApprovalRule:     payload.MCP.ApprovalRule,
			ExecutionMode:    payload.MCP.ExecutionMode,
		},
	}
	return call, nil
}

func (e *V3Executor) PrepareAppTool(
	ctx context.Context,
	request sessionv3Runtime.AppToolPreparationRequest,
) (sessionv3Runtime.ToolCall, error) {
	if e == nil || e.runtime == nil {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"session v3 MCP runtime is unavailable",
		)
	}
	mapping, err := v3AppMapping(request.App)
	if err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}
	if request.App.ServerID != string(mapping.Server) {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"app route belongs to another MCP server",
		)
	}
	if _, err := jsonutil.DecodeCanonicalValue[map[string]any](
		nonEmptyV3Arguments(request.Arguments),
	); err != nil {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"app Tool arguments must be a JSON object",
		)
	}

	tools, err := e.runtime.ListTools(ctx, mapping.Server)
	if err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}
	var selected *mcpServer.MCPToolCapability
	for index := range tools {
		if tools[index].ToolName == strings.TrimSpace(request.ToolName) {
			selected = &tools[index]
			break
		}
	}
	if selected == nil ||
		!selected.Enabled ||
		!mcpApps.ToolVisibleToApp(selected.App) {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"app requested an unavailable MCP Tool",
		)
	}
	policy, err := e.runtime.AppPolicy(ctx, mapping.Server)
	if err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}
	if err := mcpApps.ValidateAppToolInvocation(
		policy,
		*selected,
		mapping.Server,
	); err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}

	serverPin := *request.App.Binding.MCP.ServerArtifact
	return sessionv3Runtime.ToolCall{
		Name:          selected.ProviderToolName,
		ChoiceID:      selected.ChoiceID,
		Type:          inferenceSpec.ToolTypeFunction,
		DisplayName:   firstNonBlank(selected.DisplayName, selected.ToolName, "MCP Tool"),
		Description:   strings.TrimSpace(selected.Description),
		Source:        sessionv3Runtime.ToolCallSourceApp,
		AppInstanceID: request.App.ID,
		Arguments:     nonEmptyV3Arguments(request.Arguments),
		Binding: sessionv3Runtime.ToolBinding{
			Kind: sessionv3Runtime.ToolBindingKindMCP,
			MCP: &sessionv3Runtime.MCPToolBinding{
				ServerArtifact:   &serverPin,
				ServerID:         string(mapping.Server),
				ProviderToolName: selected.ProviderToolName,
				ChoiceID:         selected.ChoiceID,
				ToolName:         selected.ToolName,
				ToolDigest:       selected.Digest,
				ApprovalRule:     string(selected.ApprovalRule),
				ExecutionMode:    string(selected.ExecutionMode),
			},
		},
		Automatic: false,
	}, nil
}

func (e *V3Executor) ReadAppResource(
	ctx context.Context,
	request sessionv3Runtime.AppResourceReadRequest,
) (sessionv3Runtime.AppResourceReadResult, error) {
	if e == nil || e.runtime == nil {
		return sessionv3Runtime.AppResourceReadResult{}, errors.New(
			"session v3 MCP runtime is unavailable",
		)
	}
	mapping, err := v3AppMapping(request.App)
	if err != nil {
		return sessionv3Runtime.AppResourceReadResult{}, err
	}
	if request.App.ServerID != string(mapping.Server) {
		return sessionv3Runtime.AppResourceReadResult{}, errors.New(
			"app route belongs to another MCP server",
		)
	}
	uri := strings.TrimSpace(request.ResourceURI)
	if uri == "" {
		uri = strings.TrimSpace(request.App.ResourceURI)
	}
	if uri == "" {
		return sessionv3Runtime.AppResourceReadResult{}, errors.New(
			"app resource URI is required",
		)
	}
	resources, err := e.runtime.ListResources(ctx, mapping.Server)
	if err != nil {
		return sessionv3Runtime.AppResourceReadResult{}, err
	}
	found := false
	for _, resource := range resources {
		if resource.URI == uri {
			found = true
			break
		}
	}
	if !found {
		return sessionv3Runtime.AppResourceReadResult{}, errors.New(
			"app resource is unavailable on the originating MCP server",
		)
	}
	response, err := e.runtime.ReadResource(ctx, mapping.Server, uri)
	if err != nil {
		return sessionv3Runtime.AppResourceReadResult{}, err
	}
	value, err := canonicalV3MCPValue(response)
	if err != nil {
		return sessionv3Runtime.AppResourceReadResult{}, err
	}
	return sessionv3Runtime.AppResourceReadResult{Content: value}, nil
}

func (e *V3Executor) ValidateAppContext(
	ctx context.Context,
	request sessionv3Runtime.AppContextStagingRequest,
) (sessionv3Runtime.AppContextUpdate, error) {
	mapping, err := v3AppMapping(request.App)
	if err != nil {
		return sessionv3Runtime.AppContextUpdate{}, err
	}
	policy, err := e.runtime.AppPolicy(ctx, mapping.Server)
	if err != nil {
		return sessionv3Runtime.AppContextUpdate{}, err
	}
	if !policy.Enabled {
		return sessionv3Runtime.AppContextUpdate{}, errors.New(
			"MCP Apps are not enabled for this server",
		)
	}
	update := request.Update
	update.AppInstanceID = request.App.ID
	update.ViewID = request.ViewID
	update.ServerID = request.App.ServerID
	if update.ResourceURI == "" {
		update.ResourceURI = request.App.ResourceURI
	}
	if update.UpdatedAt.IsZero() {
		update.UpdatedAt = time.Now().UTC()
	}
	if status, statusErr := e.runtime.Status(ctx, mapping.Server); statusErr == nil &&
		status != nil && status.ServerInfo != nil {
		update.ServerDisplayName = strings.TrimSpace(status.ServerInfo.Name)
	}
	for index := range update.Content {
		if update.Content[index].DisplayName == "" {
			update.Content[index].DisplayName = firstNonBlank(
				update.Content[index].Name,
				update.Content[index].Type,
				"app context",
			)
		}
	}
	return update, nil
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

	mapping, invokeRequest, err := v3MappedMCPRequest(payload)
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
		effect := sessionv3Runtime.EffectStateConfirmedNotApplied
		return sessionv3Runtime.BindResult{
			SemanticBinding:      semantic,
			ImmediateOutput:      v3MCPToolErrorOutput(message),
			ImmediateEffectState: &effect,
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
	mapping, invokeRequest, err := v3MappedMCPRequest(&effective)
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

	adapted, err := AdaptMCPToolOutput(
		response.Content,
		response.StructuredContent,
	)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{}, true, err
	}
	structured, err := jsonutil.CanonicalValueFrom(
		adapted.Structured,
	)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{}, true, err
	}
	kind := sessionv3Runtime.ToolOutputKindSuccess
	errorText := ""
	if response.IsError {
		kind = sessionv3Runtime.ToolOutputKindError
		errorText = firstNonBlank(
			adapted.Text,
			"MCP Tool returned an error result",
		)
	}
	var app *sessionv3Runtime.AppOutput
	if response.App != nil {
		resourceURI := strings.TrimSpace(response.App.ResourceURI)
		if resourceURI == "" {
			resourceURI = mapping.AppResourceURI
		}
		app = &sessionv3Runtime.AppOutput{
			ServerID:    string(mapping.Server),
			ResourceURI: resourceURI,
		}
	}

	result := sessionv3Runtime.WorkExecutionResult{
		Output: &sessionv3Runtime.WorkItemOutput{
			Value: structured,
			Tool: &sessionv3Runtime.ToolOutput{
				Kind:       kind,
				Text:       adapted.Text,
				Contents:   adapted.Contents,
				Structured: structured,
				Error:      errorText,
			},
			Receipt: &sessionv3Runtime.ExternalReceipt{
				System:      "mcp",
				Reference:   private.TargetFingerprint,
				ConfirmedAt: time.Now().UTC(),
			},
			App: app,
		},
		Confirmed: true,
	}

	if !response.IsError {
		// A successful MCP response is positive domain evidence that the
		// invocation settled. Runtime derives notApplicable for reads and
		// confirmedApplied for mutating WorkItems from this receipt.
		return result, true, nil
	}

	if request.WorkItem.Effect.Kind == sessionv3Runtime.EffectKindRead {
		result.Error = &sessionv3Runtime.SanitizedError{
			Code:     "mcp_tool_error",
			Category: sessionv3Runtime.ErrorCategoryExecution,
			Message:  errorText,
		}
		return result, true, nil
	}

	effect := sessionv3Runtime.EffectStateUncertain
	result.Confirmed = false
	result.EffectState = &effect
	result.Error = &sessionv3Runtime.SanitizedError{
		Code:     "mcp_tool_error",
		Category: sessionv3Runtime.ErrorCategoryUncertainty,
		Effect:   effect,
		Message:  errorText,
	}
	return result, true, nil
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
			effect := sessionv3Runtime.EffectStateConfirmedNotApplied
			return sessionv3Runtime.WaitResolutionResult{
				Output:      v3MCPToolErrorOutput(message),
				Confirmed:   true,
				EffectState: &effect,
			}, true, nil
		}
		return sessionv3Runtime.WaitResolutionResult{}, true, err
	}

	switch result.Decision {
	case mcpServer.MCPApprovalDecisionDenied:
		effect := sessionv3Runtime.EffectStateConfirmedNotApplied
		return sessionv3Runtime.WaitResolutionResult{
			Output:      v3MCPToolErrorOutput(toolCallDenied),
			Confirmed:   true,
			EffectState: &effect,
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

func v3MCPToolErrorOutput(
	message string,
) *sessionv3Runtime.WorkItemOutput {
	value := jsonutil.MustCanonicalValue(struct {
		Kind    sessionv3Runtime.ToolOutputKind `json:"kind"`
		Message string                          `json:"message"`
	}{
		Kind:    sessionv3Runtime.ToolOutputKindError,
		Message: message,
	})
	return &sessionv3Runtime.WorkItemOutput{
		Value: value,
		Tool: &sessionv3Runtime.ToolOutput{
			Kind:       sessionv3Runtime.ToolOutputKindError,
			Text:       message,
			Structured: value.Clone(),
			Error:      message,
		},
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

func v3AppMapping(
	app sessionv3Runtime.AppInstance,
) (invocation.MappedToolBinding, error) {
	if app.Binding.Kind != sessionv3Runtime.ToolBindingKindMCP ||
		app.Binding.MCP == nil ||
		app.Binding.MCP.ServerArtifact == nil {
		return invocation.MappedToolBinding{}, errors.New(
			"app has no MCP Tool binding",
		)
	}
	return v3MCPMapping(&sessionv3Runtime.ToolWork{
		Call: sessionv3Runtime.ToolCall{
			Binding: app.Binding,
		},
	})
}

func nonEmptyV3Arguments(
	value jsonutil.CanonicalValue,
) jsonutil.CanonicalValue {
	if len(value) != 0 {
		return value.Clone()
	}
	return jsonutil.MustCanonicalValue(map[string]any{})
}

func v3MappedMCPRequest(
	payload *sessionv3Runtime.ToolWork,
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
		arguments = jsonutil.MustCanonicalValue(map[string]any{})
	}
	values, err := jsonutil.DecodeCanonicalValue[map[string]any](
		arguments,
	)
	if err != nil || values == nil {
		return invocation.MappedToolBinding{},
			mcpServer.InvokeMCPToolRequestBody{},
			errors.New("mcp tool arguments must be a json object")
	}

	source := mcpServer.MCPInvocationSourceModel
	if payload.Call.Source == sessionv3Runtime.ToolCallSourceApp {
		source = mcpServer.MCPInvocationSourceApp
	}

	return mapping, mcpServer.InvokeMCPToolRequestBody{
		Source:           source,
		ToolName:         mapping.ToolName,
		ProviderToolName: mapping.ProviderToolName,
		ChoiceID:         mapping.ChoiceID,
		ToolDigest:       mapping.ToolDigest,
		Arguments:        values,
		ToolUseID: firstNonBlank(
			payload.Call.ProviderCallID,
			payload.Call.ID,
		),
		AppInstanceID: string(payload.Call.AppInstanceID),
	}, nil
}

func workflowMCPServerID(
	pin sessionv3Runtime.ArtifactPin,
) (mcpServer.ServerID, error) {
	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(pin.RootID),
		ArtifactID: artifactModel.ArtifactID(pin.ArtifactID),
	}
	if err := ref.Validate(); err != nil {
		return "", err
	}
	return ServerIDForArtifact(ref)
}

func canonicalV3MCPValue(
	value any,
) (jsonutil.CanonicalValue, error) {
	raw, err := jsonutil.EncodeToJSONRaw(value)
	if err != nil {
		return nil, err
	}
	return jsonutil.CanonicalValueFrom(raw)
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
	return MCPContentsText(contents)
}

func v3MCPContentText(content mcpServer.MCPContent) string {
	return MCPContentText(content)
}
