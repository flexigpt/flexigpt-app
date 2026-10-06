package inferencewrapper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"

	"github.com/flexigpt/inference-go"
	"github.com/flexigpt/inference-go/debugclient"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	conversationSpec "github.com/flexigpt/flexigpt-app/internal/conversation/spec"
	inferencewrapperSpec "github.com/flexigpt/flexigpt-app/internal/inferencewrapper/spec"

	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

const (
	// Backend batching is the only intentional stream throttle.
	defaultFlushIntervalMillis = 32
	defaultFlushChunkSize      = 256
)

// ProviderSetAPI is a thin aggregator on top of inference-go's ProviderSetAPI.
// It owns:
//   - provider lifecycle (add/delete/set API key),
//   - attachment/tool hydration,
//   - mapping Conversation+CurrentTurn -> inference-go FetchCompletionRequest.
type ProviderSetAPI struct {
	inner *inference.ProviderSetAPI

	models             inferencewrapperSpec.ModelRuntime
	toolsSvc           ToolSource
	artifactSkills     inferencewrapperSpec.SkillSource
	mcpInferenceBridge *MCPInferenceBridge
	workspaceSource    WorkspaceSource

	logger             *slog.Logger
	debugger           *debugclient.HTTPCompletionDebugger
	initialDebugConfig *debugclient.DebugConfig

	skillsRunScriptEnabled bool
}

type ProviderSetOption func(*ProviderSetAPI)

func WithLogger(logger *slog.Logger) ProviderSetOption {
	return func(ps *ProviderSetAPI) {
		ps.logger = logger
	}
}

func WithDebugConfig(debugConfig *debugclient.DebugConfig) ProviderSetOption {
	return func(ps *ProviderSetAPI) {
		if debugConfig == nil {
			ps.initialDebugConfig = nil
			return
		}
		cloned := *debugConfig
		ps.initialDebugConfig = &cloned
	}
}

// WithSkillsRunScriptEnabled narrows the Skill source's run-script policy
// for advertised completion tools. The source must still reject advertising
// script execution when runtime policy disables it.
func WithSkillsRunScriptEnabled(enabled bool) ProviderSetOption {
	return func(ps *ProviderSetAPI) { ps.skillsRunScriptEnabled = enabled }
}

// NewProviderSetAPI validates required capabilities once during assembly.
// MCP hydration is optional; the other capabilities are required.
func NewProviderSetAPI(
	models inferencewrapperSpec.ModelRuntime,
	tools ToolSource,
	artifactSkills inferencewrapperSpec.SkillSource,
	mcpBridge *MCPInferenceBridge,
	workspaceSource WorkspaceSource,
	opts ...ProviderSetOption,
) (*ProviderSetAPI, error) {
	if models == nil || tools == nil || artifactSkills == nil || workspaceSource == nil {
		return nil, errors.New("inferencewrapper: required capabilities are incomplete")
	}
	ps := &ProviderSetAPI{
		models:             models,
		toolsSvc:           tools,
		artifactSkills:     artifactSkills,
		mcpInferenceBridge: mcpBridge,
		workspaceSource:    workspaceSource,

		skillsRunScriptEnabled: artifactSkills.RunScriptsEnabled(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(ps)
		}
	}
	allOpts := make([]inference.ProviderSetOption, 0, 2)
	if ps.logger == nil {
		ps.logger = slog.Default()
	}
	allOpts = append(allOpts, inference.WithLogger(ps.logger))
	// Always install a debugger so runtime debug config changes work even if debugging starts out disabled.
	// If no config was provided, we begin in a disabled state and can enable later via SetDebugConfig without
	// rebuilding
	// providers or HTTP clients.
	debugCfg := disabledDebugConfig()
	if ps.initialDebugConfig != nil {
		debugCfg = *ps.initialDebugConfig
	}
	dbg := debugclient.NewHTTPCompletionDebugger(&debugCfg)
	ps.debugger = dbg
	allOpts = append(allOpts,
		inference.WithDebugClientBuilder(func(p inferenceSpec.ProviderParam) inferenceSpec.CompletionDebugger {
			return dbg
		}),
	)

	inner, err := inference.NewProviderSetAPI(allOpts...)
	if err != nil {
		return nil, err
	}
	ps.inner = inner

	return ps, nil
}

// SetDebugConfig updates the live debug client configuration used for future
// completions. Passing nil disables debugging while keeping the transport
// wrapper installed, so debugging can be enabled again later without
// reinitializing providers.
func (ps *ProviderSetAPI) SetDebugConfig(cfg *debugclient.DebugConfig) {
	if ps == nil || ps.debugger == nil {
		return
	}

	next := disabledDebugConfig()
	if cfg != nil {
		next = *cfg
	}

	ps.debugger.SetConfig(next)
}

// GetDebugConfig returns a defensive copy of the live debug configuration.
func (ps *ProviderSetAPI) GetDebugConfig() *debugclient.DebugConfig {
	if ps == nil || ps.debugger == nil {
		return nil
	}
	cfg := ps.debugger.GetConfig()
	return &cfg
}

// AddProvider forwards to inference-go ProviderSetAPI.AddProvider.
func (ps *ProviderSetAPI) AddProvider(
	ctx context.Context,
	req *inferencewrapperSpec.AddProviderRequest,
) (*inferencewrapperSpec.AddProviderResponse, error) {
	if req == nil || req.Body == nil || req.Provider == "" || strings.TrimSpace(req.Body.Origin) == "" {
		return nil, errors.New("invalid params")
	}

	cfg := &inference.AddProviderConfig{
		SDKType:                  req.Body.SDKType,
		Origin:                   req.Body.Origin,
		ChatCompletionPathPrefix: req.Body.ChatCompletionPathPrefix,
		APIKeyHeaderKey:          req.Body.APIKeyHeaderKey,
		DefaultHeaders:           req.Body.DefaultHeaders,
	}
	if _, err := ps.inner.AddProvider(ctx, req.Provider, cfg); err != nil {
		return nil, err
	}

	return &inferencewrapperSpec.AddProviderResponse{}, nil
}

// DeleteProvider forwards to inference-go ProviderSetAPI.DeleteProvider.
func (ps *ProviderSetAPI) DeleteProvider(
	ctx context.Context,
	req *inferencewrapperSpec.DeleteProviderRequest,
) (*inferencewrapperSpec.DeleteProviderResponse, error) {
	if req == nil || req.Provider == "" {
		return nil, errors.New("got empty provider input")
	}
	if err := ps.inner.DeleteProvider(ctx, req.Provider); err != nil {
		return nil, err
	}

	return &inferencewrapperSpec.DeleteProviderResponse{}, nil
}

// SetProviderAPIKey forwards to inference-go ProviderSetAPI.SetProviderAPIKey.
func (ps *ProviderSetAPI) SetProviderAPIKey(
	ctx context.Context,
	req *inferencewrapperSpec.SetProviderAPIKeyRequest,
) (*inferencewrapperSpec.SetProviderAPIKeyResponse, error) {
	if req == nil || req.Body == nil {
		return nil, errors.New("got empty provider input")
	}
	if err := ps.inner.SetProviderAPIKey(ctx, req.Provider, req.Body.APIKey); err != nil {
		return nil, err
	}
	return &inferencewrapperSpec.SetProviderAPIKeyResponse{}, nil
}

// FetchCompletion builds a normalized inference-go FetchCompletionRequest from
// app-level conversation types and calls inference-go's FetchCompletion.
func (ps *ProviderSetAPI) FetchCompletion(
	ctx context.Context,
	req *inferencewrapperSpec.CompletionRequest,
) (*inferencewrapperSpec.CompletionResponse, error) {
	if req == nil {
		return nil, errors.New("got empty completion input")
	}
	if req.Current.Role != inferenceSpec.RoleUser {
		return nil, errors.New("current turn must have role=user")
	}
	if len(req.Current.ToolChoices) > 0 {
		return nil, errors.New("prepopulated tool choices are not allowed in fetch completion, need tool store choices")
	}

	runtimeModel, err := ps.models.ResolveRuntimeConfiguration(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if err := runtimeModel.Validate(); err != nil {
		return nil, err
	}
	modelParam := runtimeModel.ModelParam

	ck := uuidutil.NewUUIDv7()
	currentMessage := req.Current
	runtimeProvider, release, err := ps.registerRuntimeProvider(
		ctx,
		runtimeModel,
		ck,
	)
	if err != nil {
		return nil, err
	}
	defer release()

	capabilityResolver, err := ps.newRuntimeCapabilityResolver(
		ctx,
		runtimeProvider,
		runtimeModel,
		ck,
	)
	if err != nil {
		return nil, err
	}

	// Flatten full conversation (history + current) into InputUnion list.
	inputs, currentInputs, err := ps.buildInputs(ctx, req.History, currentMessage)
	if err != nil {
		return nil, err
	}

	inputs, currentInputs = stripGeneratedCurrentContextInputs(
		inputs,
		currentInputs,
	)

	if err := validateArtifactSkillRefsForSelection(
		currentMessage.WorkspaceSelection,
		currentMessage.EnabledSkillRefs,
	); err != nil {
		return &inferencewrapperSpec.CompletionResponse{
			Body: &inferencewrapperSpec.CompletionResponseBody{
				InferenceResponse: &inferenceSpec.FetchCompletionResponse{
					Error: &inferenceSpec.Error{
						Code:    "workspace_selection_invalid",
						Message: err.Error(),
					},
				},
				HydratedCurrentInputs: currentInputs,
			},
		}, nil
	}

	var workspaceUsage *conversationSpec.WorkspaceConversationUsage
	if currentMessage.WorkspaceSelection != nil {
		hydrated, workspaceErr := HydrateCompletion(
			ctx,
			ps.workspaceSource,
			currentMessage.WorkspaceSelection,
		)
		if hydrated != nil {
			workspaceUsage = hydrated.Usage
			if len(hydrated.CurrentInputs) > 0 {
				inputs, currentInputs = prependCurrentInputs(
					inputs,
					currentInputs,
					hydrated.CurrentInputs...,
				)
			}
		}
		if workspaceErr != nil {
			return &inferencewrapperSpec.CompletionResponse{
				Body: &inferencewrapperSpec.CompletionResponseBody{
					InferenceResponse: &inferenceSpec.FetchCompletionResponse{
						Error: &inferenceSpec.Error{
							Code:    "workspace_unavailable",
							Message: workspaceErr.Error(),
						},
					},
					HydratedCurrentInputs: currentInputs,
					WorkspaceUsage:        workspaceUsage,
				},
			}, nil
		}
		if hydrated != nil && len(hydrated.SystemPromptParts) != 0 {
			// Workspace insert=instructions content is intentionally routed
			// through the system-prompt path. User-message Workspace content
			// remains in hydrated.CurrentInputs and is never merged here.
			modelParam.SystemPrompt = appendToSystemPrompt(
				modelParam.SystemPrompt,
				hydrated.SystemPromptParts...,
			)
		}
	}

	mcpContext := req.MCPContext
	if mcpContext == nil {
		mcpContext = currentMessage.MCPContext
	}
	if len(currentMessage.MCPAppContextUpdates) != 0 {
		if mcpContext == nil {
			return nil, errors.New(
				"MCP App context updates require an MCP conversation context",
			)
		}
		if err := mcpContext.ValidateAppContextUpdates(
			currentMessage.MCPAppContextUpdates,
		); err != nil {
			return nil, err
		}
	}
	if appCtxInput := buildMCPAppContextInput(currentMessage.MCPAppContextUpdates); appCtxInput != nil {
		inputs, currentInputs = prependCurrentInputs(inputs, currentInputs, *appCtxInput)
	}
	// Build tool choices for this call.
	toolChoices, err := buildToolChoices(ctx, ps.toolsSvc, req.ToolSelections)
	if err != nil {
		return nil, err
	}

	enabledSkillRefs := currentMessage.EnabledSkillRefs
	if workspaceUsage != nil {
		enabledSkillRefs = filterWorkspaceSkillRefsToResolvedSelection(
			enabledSkillRefs,
			workspaceUsage,
		)
	}
	skillSessionID := strings.TrimSpace(req.SkillSessionID)

	// A Workspace selection is authoritative for which Root-scoped Skills may
	// participate in this turn. If no runtime allow-list reaches Skill Runtime,
	// do not report selected Skills as silently available.
	//
	// This covers stale persisted conversations where Workspace selection
	// survives but a selected Artifact can no longer composition. A usable Context
	// may still make the turn partial rather than
	// completely unavailable.
	if workspaceUsage != nil &&
		len(workspaceUsage.Skills) > 0 &&
		len(enabledSkillRefs) == 0 {
		markWorkspaceSkillSessionUsage(
			workspaceUsage,
			enabledSkillRefs,
			nil,
			nil,
			false,
		)
		if workspaceUsage.Status == conversationSpec.WorkspaceConversationSelectionUnavailable {
			return workspaceUnavailableCompletionResponse(
				currentInputs,
				workspaceUsage,
				"selected Workspace Skills were not included in the Skill Runtime session allow-list",
			), nil
		}
	}

	if len(enabledSkillRefs) > 0 {
		if skillSessionID == "" {
			if workspaceUsage != nil && len(workspaceUsage.Skills) > 0 {
				markWorkspaceSkillSessionUsage(
					workspaceUsage,
					enabledSkillRefs,
					nil,
					nil,
					false,
				)
				return workspaceUnavailableCompletionResponse(
					currentInputs,
					workspaceUsage,
					"selected Workspace Skills require a current Skill Runtime session",
				), nil
			}
			return nil, errors.New("enabledSkillRefs provided but skillSessionID is missing")
		}

		skills, skillErr := ps.artifactSkills.ResolveSkillSession(
			ctx,
			inferencewrapperSpec.SkillSessionRequest{
				SessionID:        agentskillsRuntimeSpec.SessionID(skillSessionID),
				Artifacts:        enabledSkillRefs,
				IncludeRunScript: ps.skillsRunScriptEnabled,
			},
		)
		if skillErr != nil {
			if errors.Is(skillErr, agentskillsRuntimeSpec.ErrSessionNotFound) {
				return nil, fmt.Errorf("skill session %q not found: %w", skillSessionID, skillErr)
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			ps.logger.Warn("hydrate artifact skills failed; disabling skills for this turn", "err", skillErr)
			skills = inferencewrapperSpec.SkillSession{}
		}
		availableSkillRefs := skills.AvailableArtifacts
		activeSkillRefs := skills.ActiveArtifacts
		skillsPrompt := strings.TrimSpace(skills.Prompt)

		// Only expose skills tools if we also have the prompt.
		if skillsPrompt != "" {
			modelParam.SystemPrompt = appendToSystemPrompt(
				modelParam.SystemPrompt,
				skills.RulesPrompt,
				skillsPrompt,
			)
			toolChoices = append(toolChoices, skills.ToolChoices...)
		}

		markWorkspaceSkillSessionUsage(
			workspaceUsage,
			enabledSkillRefs,
			availableSkillRefs,
			activeSkillRefs,
			skillsPrompt != "",
		)

		if workspaceUsage != nil &&
			workspaceUsage.Status == conversationSpec.WorkspaceConversationSelectionUnavailable {
			return workspaceUnavailableCompletionResponse(
				currentInputs,
				workspaceUsage,
				"selected Workspace Skills could not enter the active Skill Runtime session",
			), nil
		}
	}

	var mcpDebugDetails map[string]any
	var mcpToolMappings []conversationSpec.MCPProviderToolMapping
	if ps.mcpInferenceBridge != nil && mcpContext != nil {
		hydrated, err := ps.mcpInferenceBridge.HydrateCompletion(ctx, MCPCompletionHydrationRequest{
			Context:             mcpContext,
			ExistingToolChoices: toolChoices,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to hydrate MCP context: %w", err)
		}
		if hydrated != nil {
			if len(hydrated.SystemPromptParts) > 0 {
				modelParam.SystemPrompt = appendToSystemPrompt(
					modelParam.SystemPrompt,
					hydrated.SystemPromptParts...,
				)
			}
			if len(hydrated.CurrentInputs) > 0 {
				inputs, currentInputs = prependCurrentInputs(inputs, currentInputs, hydrated.CurrentInputs...)
			}
			if len(hydrated.ToolChoices) > 0 {
				toolChoices = append(toolChoices, hydrated.ToolChoices...)
			}
			if len(hydrated.ToolMappings) > 0 {
				mcpToolMappings = append(
					mcpToolMappings,
					hydrated.ToolMappings...,
				)
			}

			mcpDebugDetails = hydrated.DebugDetails
		}
	}

	infReq := &inferenceSpec.FetchCompletionRequest{
		ModelParam:  modelParam,
		Inputs:      inputs,
		ToolChoices: toolChoices,
	}

	opts := &inferenceSpec.FetchCompletionOptions{
		CompletionKey:      ck,
		CapabilityResolver: capabilityResolver,
	}

	// Keep provider batching at the transport boundary. The frontend owns
	// paint-aligned rendering and additionally coalesces visible updates to
	// avoid UI work more often than its stream frame budget.
	if req.OnStreamText != nil || req.OnStreamThinking != nil {
		opts.StreamHandler = makeStreamHandler(req.OnStreamText, req.OnStreamThinking)
		opts.StreamConfig = &inferenceSpec.StreamConfig{
			FlushIntervalMillis: defaultFlushIntervalMillis,
			FlushChunkSize:      defaultFlushChunkSize,
		}
	}

	b, err := ps.inner.FetchCompletion(
		ctx,
		runtimeProvider,
		infReq,
		opts,
	)

	// A nil or empty successful final payload is not a usable completion.
	// Streaming may already have delivered visible text, so the frontend will
	// preserve it and append this error instead of replacing it with blank UI.
	if b == nil && err == nil {
		b = &inferenceSpec.FetchCompletionResponse{
			Error: &inferenceSpec.Error{
				Code:    "empty_completion_response",
				Message: "provider finished without a final response payload",
			},
		}
	}
	if b != nil && err == nil && b.Error == nil && len(b.Outputs) == 0 {
		b.Error = &inferenceSpec.Error{
			Code:    "empty_completion_response",
			Message: "provider finished without final output",
		}
	}

	if b != nil && workspaceUsage != nil {
		b.DebugDetails = mergeCompletionDebugDetails(
			b.DebugDetails,
			"workspace",
			workspaceUsage,
		)
	}
	if b != nil && mcpDebugDetails != nil {
		b.DebugDetails = mergeCompletionDebugDetails(b.DebugDetails, "mcp", mcpDebugDetails)
	}

	resp := &inferencewrapperSpec.CompletionResponse{Body: &inferencewrapperSpec.CompletionResponseBody{
		InferenceResponse:     b,
		HydratedCurrentInputs: currentInputs,
		MCPToolMappings:       mcpToolMappings,
		WorkspaceUsage:        workspaceUsage,
	}}

	return resp, err
}

func workspaceUnavailableCompletionResponse(
	currentInputs []inferenceSpec.InputUnion,
	workspaceUsage *conversationSpec.WorkspaceConversationUsage,
	message string,
) *inferencewrapperSpec.CompletionResponse {
	return &inferencewrapperSpec.CompletionResponse{
		Body: &inferencewrapperSpec.CompletionResponseBody{
			InferenceResponse: &inferenceSpec.FetchCompletionResponse{
				Error: &inferenceSpec.Error{
					Code:    "workspace_unavailable",
					Message: message,
				},
			},
			HydratedCurrentInputs: currentInputs,
			WorkspaceUsage:        workspaceUsage,
		},
	}
}

// buildInputs flattens History + Current into a single InputUnion slice.
// Attachments are always built from top level param and added to the union.
// If the caller hydrates it then there is a possibility of duplicates.
func (ps *ProviderSetAPI) buildInputs(
	ctx context.Context,
	history []conversationSpec.ConversationMessage,
	inCurrent conversationSpec.ConversationMessage,
) (all, current []inferenceSpec.InputUnion, err error) {
	out := make([]inferenceSpec.InputUnion, 0)

	// 1) History: replay stored unions exactly as they were.
	for _, turn := range history {
		// Inputs first, then Outputs, preserving stored order.

		out = append(out, cloneInputUnionsForLocalMutation(turn.Inputs)...)
		for _, outEv := range turn.Outputs {
			// Outputs are not directly part of InputUnion; but for replay
			// we want them to be visible as prior context. We embed them
			// as InputUnion using the matching InputKind* variants.
			o := outputToInput(outEv)
			if o != nil {
				out = append(out, *o)
			}
		}
	}

	cur := inCurrent

	// If the caller already provided normalized InputUnions, just reuse them.
	currentOut := make([]inferenceSpec.InputUnion, 0)
	if len(cur.Inputs) > 0 {
		currentOut = append(currentOut, cloneInputUnionsForLocalMutation(cur.Inputs)...)
	}

	// Always process attachments into content items.
	msgContentItems, err := buildContentItemsFromAttachments(ctx, cur.Attachments)
	if err != nil {
		return nil, nil, err
	}

	if len(msgContentItems) > 0 {
		// Try to merge into the last user InputMessage if present.
		merged := false
		for idx := range slices.Backward(currentOut) {
			iu := &currentOut[idx]
			if iu.Kind == inferenceSpec.InputKindInputMessage &&
				iu.InputMessage != nil &&
				iu.InputMessage.Role == inferenceSpec.RoleUser {
				iu.InputMessage.Contents = append(iu.InputMessage.Contents, msgContentItems...)
				merged = true
				break
			}
		}

		if !merged {
			inputMsg := inferenceSpec.InputOutputContent{
				ID:       "",
				Role:     inferenceSpec.RoleUser,
				Status:   inferenceSpec.StatusNone,
				Contents: msgContentItems,
			}
			currentOut = append(currentOut, inferenceSpec.InputUnion{
				Kind:         inferenceSpec.InputKindInputMessage,
				InputMessage: &inputMsg,
			})
		}
	}
	if len(currentOut) > 0 {
		out = append(out, currentOut...)
	}

	if len(out) == 0 {
		return nil, nil, errors.New("no usable inputs to send to inference-go")
	}

	return out, currentOut, nil
}

func appendToSystemPrompt(base string, parts ...string) string {
	base = strings.TrimSpace(base)
	var out []string
	if base != "" {
		out = append(out, base)
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, "\n\n")
}

// makeStreamHandler adapts inference-go streaming into the legacy
// text/thinking callback pair.
func makeStreamHandler(
	onText func(string) error,
	onThinking func(string) error,
) inferenceSpec.StreamHandler {
	if onText == nil && onThinking == nil {
		return nil
	}
	return func(ev inferenceSpec.StreamEvent) error {
		switch ev.Kind {
		case inferenceSpec.StreamContentKindText:
			if onText == nil || ev.Text == nil {
				return nil
			}

			// Text is a delta chunk. Empty chunks are valid no-ops and are not
			// completion markers. Do not collapse repeated equal chunks here.
			text := ev.Text.Text
			if text == "" {
				return nil
			}
			return onText(text)
		case inferenceSpec.StreamContentKindThinking:
			if onThinking == nil || ev.Thinking == nil {
				return nil
			}

			thinking := ev.Thinking.Text
			if thinking == "" {
				return nil
			}
			return onThinking(thinking)
		}
		return nil
	}
}
