package inferencewrapper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"

	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"

	"github.com/flexigpt/inference-go"
	"github.com/flexigpt/inference-go/capabilityoverride"
	"github.com/flexigpt/inference-go/debugclient"
	"github.com/flexigpt/inference-go/modelpreset"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/inferencewrapper/spec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	modelpresetStore "github.com/flexigpt/flexigpt-app/internal/modelpreset/store"
	skillAggregate "github.com/flexigpt/flexigpt-app/internal/skill/aggregate"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/selection"
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

	toolStore          *toolStore.ToolStore
	mpStore            *modelpresetStore.ModelPresetStore
	skillRuntime       *skillAggregate.Service
	mcpInferenceBridge *MCPInferenceBridge
	workspaceBridge    *WorkspaceInferenceBridge

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

// WithSkillsRunScriptEnabled controls whether skills-runscript is advertised to the model.
// Default: false (safer; matches the default fsskillprovider which disables scripts).
func WithSkillsRunScriptEnabled(enabled bool) ProviderSetOption {
	return func(ps *ProviderSetAPI) { ps.skillsRunScriptEnabled = enabled }
}

// NewProviderSetAPI creates a new ProviderSetAPI wrapper.
//
//   - ts:   tool store used to hydrate ToolChoices when needed.
//   - opts: functional options for configuring the wrapper (e.g. WithLogger, WithDebugConfig).
func NewProviderSetAPI(
	ts *toolStore.ToolStore,
	mps *modelpresetStore.ModelPresetStore,
	sr *skillAggregate.Service,
	mcpBridge *MCPInferenceBridge,
	workspaceBridge *WorkspaceInferenceBridge,
	opts ...ProviderSetOption,
) (*ProviderSetAPI, error) {
	if ts == nil || mps == nil || sr == nil || mcpBridge == nil || workspaceBridge == nil {
		return nil, errors.New("inferencewrapper: missing input")
	}
	ps := &ProviderSetAPI{
		toolStore:          ts,
		mpStore:            mps,
		skillRuntime:       sr,
		mcpInferenceBridge: mcpBridge,
		workspaceBridge:    workspaceBridge,
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
	req *spec.AddProviderRequest,
) (*spec.AddProviderResponse, error) {
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

	return &spec.AddProviderResponse{}, nil
}

// DeleteProvider forwards to inference-go ProviderSetAPI.DeleteProvider.
func (ps *ProviderSetAPI) DeleteProvider(
	ctx context.Context,
	req *spec.DeleteProviderRequest,
) (*spec.DeleteProviderResponse, error) {
	if req == nil || req.Provider == "" {
		return nil, errors.New("got empty provider input")
	}
	if err := ps.inner.DeleteProvider(ctx, req.Provider); err != nil {
		return nil, err
	}

	return &spec.DeleteProviderResponse{}, nil
}

// SetProviderAPIKey forwards to inference-go ProviderSetAPI.SetProviderAPIKey.
func (ps *ProviderSetAPI) SetProviderAPIKey(
	ctx context.Context,
	req *spec.SetProviderAPIKeyRequest,
) (*spec.SetProviderAPIKeyResponse, error) {
	if req == nil || req.Body == nil {
		return nil, errors.New("got empty provider input")
	}
	if err := ps.inner.SetProviderAPIKey(ctx, req.Provider, req.Body.APIKey); err != nil {
		return nil, err
	}
	return &spec.SetProviderAPIKeyResponse{}, nil
}

// FetchCompletion builds a normalized inference-go FetchCompletionRequest from
// app-level conversation types and calls inference-go's FetchCompletion.
func (ps *ProviderSetAPI) FetchCompletion(
	ctx context.Context,
	req *spec.CompletionRequest,
) (*spec.CompletionResponse, error) {
	if req == nil || req.Body == nil {
		return nil, errors.New("got empty completion input")
	}
	if req.Provider == "" {
		return nil, errors.New("missing provider")
	}

	if req.ModelPresetID == "" {
		return nil, errors.New("missing modelPresetID")
	}

	body := req.Body

	// Resolve model param for this call (prefer explicit body.ModelParam,
	// otherwise last non-nil ModelParam from history).
	modelParam, err := ps.resolveModelParam(body)
	if err != nil {
		return nil, err
	}
	if modelParam.Name == "" {
		return nil, errors.New("model name is required")
	}

	if len(body.Current.ToolChoices) > 0 {
		return nil, errors.New("prepopulated tool choices are not allowed in fetch completion, need tool store choices")
	}

	ck := uuidutil.NewUUIDv7()

	capabilityResolver, err := ps.newPresetCapabilityResolver(
		ctx,
		req.Provider,
		req.ModelPresetID,
		modelParam.Name,
		ck,
	)
	if err != nil {
		return nil, err
	}

	// Flatten full conversation (history + current) into InputUnion list.
	inputs, currentInputs, err := ps.buildInputs(ctx, body)
	if err != nil {
		return nil, err
	}

	if len(inputs) == 0 {
		return nil, errors.New("no usable inputs to send to inference-go")
	}

	inputs, currentInputs = stripGeneratedCurrentContextInputs(
		inputs,
		currentInputs,
	)

	if err := validateArtifactSkillRefsForSelection(
		body.Current.WorkspaceSelection,
		body.Current.EnabledSkillRefs,
	); err != nil {
		//nolint:nilerr // Explicit.
		return &spec.CompletionResponse{
			Body: &spec.CompletionResponseBody{
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

	var workspaceUsage *selection.ConversationUsage
	if body.Current.WorkspaceSelection != nil {
		hydrated, workspaceErr := ps.workspaceBridge.HydrateCompletion(
			ctx,
			body.Current.WorkspaceSelection,
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
			//nolint:nilerr // Deliberate.
			return &spec.CompletionResponse{
				Body: &spec.CompletionResponseBody{
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
	}

	mcpContext := body.MCPContext
	if mcpContext == nil {
		mcpContext = body.Current.MCPContext
	}
	if len(body.Current.MCPAppContextUpdates) != 0 {
		if mcpContext == nil {
			return nil, errors.New(
				"MCP App context updates require an MCP conversation context",
			)
		}
		if err := runtime.ValidateMCPAppContextUpdatesForContext(
			*mcpContext,
			body.Current.MCPAppContextUpdates,
		); err != nil {
			return nil, err
		}
	}
	if appCtxInput := buildMCPAppContextInput(body.Current.MCPAppContextUpdates); appCtxInput != nil {
		inputs, currentInputs = prependCurrentInputs(inputs, currentInputs, *appCtxInput)
	}
	// Build tool choices for this call.
	toolChoices, err := buildToolChoices(ctx, ps.toolStore, body.ToolStoreChoices)
	if err != nil {
		return nil, err
	}

	enabledSkillRefs := body.Current.EnabledSkillRefs
	if workspaceUsage != nil {
		enabledSkillRefs = filterWorkspaceSkillRefsToResolvedSelection(
			enabledSkillRefs,
			workspaceUsage,
		)
	}
	skillSessionID := strings.TrimSpace(body.SkillSessionID)

	// A Workspace selection is authoritative for which Workspace Skills may
	// participate in this turn. If no runtime allow-list reaches the Skill
	// Runtime, do not report selected Workspace Skills as silently available.
	//
	// This covers stale persisted conversations and frontend catalog races
	// where Workspace selection survives but its corresponding runtime ref was
	// omitted. A usable Context may still make the turn partial rather than
	// completely unavailable.
	if workspaceUsage != nil &&
		len(workspaceUsage.Skills) > 0 &&
		(ps.skillRuntime == nil || len(enabledSkillRefs) == 0) {
		markWorkspaceSkillSessionUsage(
			workspaceUsage,
			enabledSkillRefs,
			nil,
			nil,
			false,
		)
		if workspaceUsage.Status == selection.ConversationSelectionUnavailable {
			return workspaceUnavailableCompletionResponse(
				currentInputs,
				workspaceUsage,
				"selected Workspace Skills were not included in the Skill Runtime session allow-list",
			), nil
		}
	}

	if ps.skillRuntime != nil && len(enabledSkillRefs) > 0 {
		var availableSkillItems []skillAggregate.RuntimeSkillListItem
		var activeSkillItems []skillAggregate.RuntimeSkillListItem
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
		// Active skills count in this session (restricted to allowlist).
		activeResp, aerr := ps.skillRuntime.ListRuntimeSkills(ctx, &skillAggregate.ListRuntimeSkillsRequest{
			Body: &skillAggregate.ListRuntimeSkillsRequestBody{
				Filter: &skillAggregate.RuntimeSkillFilter{
					SessionID:      agentskillsRuntimeSpec.SessionID(skillSessionID),
					Activity:       agentskillsRuntimeSpec.SkillActivityAny,
					AllowArtifacts: enabledSkillRefs,
				},
			},
		})
		if aerr != nil {
			if errors.Is(aerr, agentskillsRuntimeSpec.ErrSessionNotFound) {
				return nil, fmt.Errorf("skill session %q not found", skillSessionID)
			}
			ps.logger.Warn("listRuntimeSkills failed; Workspace Skill session availability is unknown", "err", aerr)
			activeResp = nil
		}
		if activeResp != nil && activeResp.Body != nil {
			availableSkillItems = activeResp.Body.Skills
		}

		activeResp, aerr = ps.skillRuntime.ListRuntimeSkills(ctx, &skillAggregate.ListRuntimeSkillsRequest{
			Body: &skillAggregate.ListRuntimeSkillsRequestBody{
				Filter: &skillAggregate.RuntimeSkillFilter{
					SessionID:      agentskillsRuntimeSpec.SessionID(skillSessionID),
					Activity:       agentskillsRuntimeSpec.SkillActivityActive,
					AllowArtifacts: enabledSkillRefs,
				},
			},
		})
		activeCount := 0
		if aerr != nil {
			if errors.Is(aerr, agentskillsRuntimeSpec.ErrSessionNotFound) {
				return nil, fmt.Errorf("skill session %q not found", skillSessionID)
			}
			ps.logger.Warn("listRuntimeSkills failed; disabling skills for this turn", "err", aerr)
			activeResp = nil
		}
		if activeResp != nil && activeResp.Body != nil {
			activeCount = len(activeResp.Body.Skills)
			activeSkillItems = activeResp.Body.Skills
		}

		// Pick prompt activity:
		// - if none active => show available-only (inactive)
		// - else => show active + available (any).
		promptActivity := agentskillsRuntimeSpec.SkillActivityInactive
		if activeCount > 0 {
			promptActivity = agentskillsRuntimeSpec.SkillActivityAny
		}

		promptResp, perr := ps.skillRuntime.GetSkillsPrompt(ctx, &skillAggregate.GetSkillsPromptRequest{
			Body: &skillAggregate.GetSkillsPromptRequestBody{
				Filter: &skillAggregate.RuntimeSkillFilter{
					SessionID:      agentskillsRuntimeSpec.SessionID(skillSessionID),
					Activity:       promptActivity,
					AllowArtifacts: enabledSkillRefs,
				},
			},
		})
		if perr != nil {
			if errors.Is(perr, agentskillsRuntimeSpec.ErrSessionNotFound) {
				return nil, fmt.Errorf("skill session %q not found", skillSessionID)
			}
			ps.logger.Warn("getSkillsPrompt failed; disabling skills for this turn", "err", perr)
		}

		skillsPrompt := ""
		if promptResp != nil && promptResp.Body != nil {
			skillsPrompt = strings.TrimSpace(promptResp.Body.Prompt)
		}

		// Only expose skills tools if we also have the prompt.
		if skillsPrompt != "" {
			includeAllTools := activeCount > 0
			modelParam.SystemPrompt = appendToSystemPrompt(
				modelParam.SystemPrompt,
				skillsRulesPrompt(includeAllTools, ps.skillsRunScriptEnabled),
				skillsPrompt,
			)
			// Tool choices:
			// - if none active => only skills-load
			// - else => load/unload/readresource/runscript.
			skillToolChoices, err := buildSkillToolChoices(activeCount > 0, ps.skillsRunScriptEnabled)
			if err != nil {
				return nil, fmt.Errorf("failed to build skill tool choices: %w", err)
			}
			toolChoices = append(toolChoices, skillToolChoices...)
		}

		markWorkspaceSkillSessionUsage(
			workspaceUsage,
			enabledSkillRefs,
			availableSkillItems,
			activeSkillItems,
			skillsPrompt != "",
		)

		if workspaceUsage != nil &&
			workspaceUsage.Status == selection.ConversationSelectionUnavailable {
			return workspaceUnavailableCompletionResponse(
				currentInputs,
				workspaceUsage,
				"selected Workspace Skills could not enter the active Skill Runtime session",
			), nil
		}
	}

	var mcpDebugDetails map[string]any
	var mcpToolMappings []runtime.MCPProviderToolMapping
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
		ModelParam:  *modelParam,
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

	b, err := ps.inner.FetchCompletion(ctx, req.Provider, infReq, opts)

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

	resp := &spec.CompletionResponse{Body: &spec.CompletionResponseBody{
		InferenceResponse:     b,
		HydratedCurrentInputs: currentInputs,
		MCPToolMappings:       mcpToolMappings,
		WorkspaceUsage:        workspaceUsage,
	}}

	return resp, err
}

func workspaceUnavailableCompletionResponse(
	currentInputs []inferenceSpec.InputUnion,
	workspaceUsage *selection.ConversationUsage,
	message string,
) *spec.CompletionResponse {
	return &spec.CompletionResponse{
		Body: &spec.CompletionResponseBody{
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

func (ps *ProviderSetAPI) newPresetCapabilityResolver(
	ctx context.Context,
	provider inferenceSpec.ProviderName,
	modelPresetID modelpreset.ModelPresetID,
	requestModelName inferenceSpec.ModelName,
	completionKey string,
) (inferenceSpec.ModelCapabilityResolver, error) {
	if ps == nil || ps.inner == nil {
		return nil, errors.New("provider set is not initialized")
	}
	if provider == "" {
		return nil, errors.New("provider is required for capability derivation")
	}
	if strings.TrimSpace(string(modelPresetID)) == "" {
		return nil, errors.New("modelPresetID is required for capability derivation")
	}
	if ps.mpStore == nil {
		return nil, errors.New("model preset store not configured on inference wrapper")
	}

	presp, err := ps.mpStore.GetModelPreset(ctx, &modelpresetSpec.GetModelPresetRequest{
		ProviderName:    provider,
		ModelPresetID:   modelPresetID,
		IncludeDisabled: false,
	})
	if err != nil {
		return nil, err
	}
	if presp == nil || presp.Body == nil {
		return nil, errors.New("GetModelPreset: empty response")
	}

	modelName := requestModelName
	if modelName == "" {
		modelName = presp.Body.Model.Name
	}
	if modelName == "" {
		return nil, errors.New("cannot derive capabilities: model name is empty")
	}

	return ps.inner.NewPresetCapabilityResolver(
		ctx,
		provider,
		inferenceProviderPresetFromApp(presp.Body.Provider),
		inferenceModelPresetFromApp(presp.Body.Model, modelName),
		completionKey,
	)
}

func inferenceProviderPresetFromApp(pp modelpresetSpec.ProviderPreset) modelpreset.ProviderPreset {
	return modelpreset.ProviderPreset{
		Name:                     pp.Name,
		DisplayName:              pp.DisplayName,
		SDKType:                  pp.SDKType,
		Origin:                   pp.Origin,
		ChatCompletionPathPrefix: pp.ChatCompletionPathPrefix,
		APIKeyHeaderKey:          pp.APIKeyHeaderKey,
		DefaultHeaders:           maps.Clone(pp.DefaultHeaders),
		CapabilitiesOverride:     capabilityoverride.CloneModelCapabilitiesOverride(pp.CapabilitiesOverride),
	}
}

func inferenceModelPresetFromApp(
	mp modelpresetSpec.ModelPreset,
	modelName inferenceSpec.ModelName,
) modelpreset.ModelPreset {
	if modelName == "" {
		modelName = mp.Name
	}

	return modelpreset.ModelPreset{
		ID:          mp.ID,
		Name:        modelName,
		DisplayName: mp.DisplayName,
		ModelParam: inferenceSpec.ModelParam{
			Name: modelName,
		},
		CapabilitiesOverride: capabilityoverride.CloneModelCapabilitiesOverride(mp.CapabilitiesOverride),
	}
}

// resolveModelParam chooses the effective ModelParam for this call.
//
// Priority:
//  1. body.ModelParam if non-nil.
//  2. Last non-nil History[i].ModelParam.
//
// If still empty, returns an error.
func (ps *ProviderSetAPI) resolveModelParam(
	body *spec.CompletionRequestBody,
) (*inferenceSpec.ModelParam, error) {
	var mp *inferenceSpec.ModelParam
	defaultMaxPromptTokens := 8000
	if body.ModelParam != nil {
		mp = body.ModelParam
	} else {
		for _, v := range slices.Backward(body.History) {
			if v.ModelParam != nil {
				mp = v.ModelParam
				break
			}
		}
	}
	if mp == nil {
		return nil, errors.New("no valid modelparam found")
	}

	mpCopy := *mp

	if mpCopy.MaxPromptLength == 0 {
		mpCopy.MaxPromptLength = defaultMaxPromptTokens
	}

	return &mpCopy, nil
}

// buildInputs flattens History + Current into a single InputUnion slice.
// Attachments are always built from top level param and added to the union.
// If the caller hydrates it then there is a possibility of duplicates.
func (ps *ProviderSetAPI) buildInputs(
	ctx context.Context,
	body *spec.CompletionRequestBody,
) (all, current []inferenceSpec.InputUnion, err error) {
	out := make([]inferenceSpec.InputUnion, 0)

	// 1) History: replay stored unions exactly as they were.
	for _, turn := range body.History {
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

	cur := body.Current
	if cur.Role != inferenceSpec.RoleUser {
		return nil, nil, errors.New("current turn must have role=user")
	}

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
