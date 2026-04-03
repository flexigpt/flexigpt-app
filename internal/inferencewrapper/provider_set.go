package inferencewrapper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/flexigpt/inference-go"
	"github.com/flexigpt/inference-go/debugclient"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"

	"github.com/google/uuid"

	"github.com/flexigpt/flexigpt-app/internal/attachment"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/inferencewrapper/spec"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	modelpresetStore "github.com/flexigpt/flexigpt-app/internal/modelpreset/store"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
)

var defaultDebugConfig = debugclient.DebugConfig{
	Disable:                 false,
	DisableRequestBody:      false,
	DisableResponseBody:     false,
	DisableContentStripping: false,
	LogToSlog:               false,
}

func DefaultDebugConfig() debugclient.DebugConfig {
	return defaultDebugConfig
}

func disabledDebugConfig() debugclient.DebugConfig {
	cfg := defaultDebugConfig
	cfg.Disable = true
	return cfg
}

type completionKeyCapabilityResolver struct {
	key  string
	caps *inferenceSpec.ModelCapabilities
}

func (r completionKeyCapabilityResolver) ResolveModelCapabilities(
	ctx context.Context,
	req inferenceSpec.ResolveModelCapabilitiesRequest,
) (*inferenceSpec.ModelCapabilities, error) {
	if r.caps == nil {
		return nil, errors.New("no model capabilities configured")
	}
	// Enforce mapping by completionKey to avoid any model-name uniqueness assumptions.
	if r.key != "" && req.CompletionKey != r.key {
		return nil, fmt.Errorf("capabilities not found for completionKey %q", req.CompletionKey)
	}
	return r.caps, nil
}

// ProviderSetAPI is a thin aggregator on top of inference-go's ProviderSetAPI.
// It owns:
//   - provider lifecycle (add/delete/set API key),
//   - attachment/tool hydration,
//   - mapping Conversation+CurrentTurn -> inference-go FetchCompletionRequest.
type ProviderSetAPI struct {
	inner      *inference.ProviderSetAPI
	toolStore  *toolStore.ToolStore
	mpStore    *modelpresetStore.ModelPresetStore
	skillStore *skillStore.SkillStore

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
	ss *skillStore.SkillStore,
	opts ...ProviderSetOption,
) (*ProviderSetAPI, error) {
	if ts == nil || mps == nil {
		return nil, errors.New("no tool store or model preset store provided to inference wrapper provider set")
	}
	ps := &ProviderSetAPI{
		toolStore:  ts,
		mpStore:    mps,
		skillStore: ss,
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
	ps.logger.Info("add provider", "name", req.Provider)
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
	ps.logger.Info("deleteProvider", "name", req.Provider)
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

	derivedModelCapabilities, err := ps.deriveCapabilitiesFromPreset(
		ctx,
		req.Provider,
		req.ModelPresetID,
		modelParam.Name,
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

	// Build tool choices for this call.
	toolChoices, err := ps.buildToolChoices(ctx, body.ToolStoreChoices)
	if err != nil {
		return nil, err
	}

	enabledSkillRefs := body.Current.EnabledSkillRefs

	skillSessionID := strings.TrimSpace(body.SkillSessionID)
	if ps.skillStore != nil && len(enabledSkillRefs) > 0 {
		if skillSessionID == "" {
			return nil, errors.New("enabledSkillRefs provided but skillSessionID is missing")
		}
		// Active skills count in this session (restricted to allowlist).
		activeResp, aerr := ps.skillStore.ListRuntimeSkills(ctx, &skillSpec.ListRuntimeSkillsRequest{
			Body: &skillSpec.ListRuntimeSkillsRequestBody{
				Filter: &skillSpec.RuntimeSkillFilter{
					SessionID:      agentskillsSpec.SessionID(skillSessionID),
					Activity:       agentskillsSpec.SkillActivityActive,
					AllowSkillRefs: enabledSkillRefs,
				},
			},
		})
		activeCount := 0
		if aerr != nil {
			if errors.Is(aerr, agentskillsSpec.ErrSessionNotFound) {
				return nil, fmt.Errorf("skill session %q not found", skillSessionID)
			}
			ps.logger.Warn("listRuntimeSkills failed; disabling skills for this turn", "err", aerr)
			activeResp = nil
		}
		if activeResp != nil && activeResp.Body != nil {
			activeCount = len(activeResp.Body.Skills)
		}

		// Pick prompt activity:
		// - if none active => show available-only (inactive)
		// - else => show active + available (any).
		promptActivity := agentskillsSpec.SkillActivityInactive
		if activeCount > 0 {
			promptActivity = agentskillsSpec.SkillActivityAny
		}

		promptResp, perr := ps.skillStore.GetSkillsPrompt(ctx, &skillSpec.GetSkillsPromptRequest{
			Body: &skillSpec.GetSkillsPromptRequestBody{
				Filter: &skillSpec.RuntimeSkillFilter{
					SessionID:      agentskillsSpec.SessionID(skillSessionID),
					Activity:       promptActivity,
					AllowSkillRefs: enabledSkillRefs,
				},
			},
		})
		if perr != nil {
			if errors.Is(perr, agentskillsSpec.ErrSessionNotFound) {
				return nil, fmt.Errorf("skill session %q not found", skillSessionID)
			}
			ps.logger.Warn("getSkillsPrompt failed; disabling skills for this turn", "err", perr)
		}

		xmlResp := ""
		if promptResp != nil && promptResp.Body != nil {
			xmlResp = strings.TrimSpace(promptResp.Body.XML)
		}

		// Only expose skills tools if we also have the XML prompt.
		if xmlResp != "" {
			includeAllTools := activeCount > 0
			modelParam.SystemPrompt = appendToSystemPrompt(
				modelParam.SystemPrompt,
				skillsRulesPrompt(includeAllTools, ps.skillsRunScriptEnabled),
				xmlResp,
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
	}

	infReq := &inferenceSpec.FetchCompletionRequest{
		ModelParam:  *modelParam,
		Inputs:      inputs,
		ToolChoices: toolChoices,
	}

	var ck string
	uid, err := uuid.NewV7()
	if err != nil {
		// Fallback only.
		ck = string(req.Provider) + "_" + string(req.ModelPresetID)
	} else {
		ck = uid.String()
	}

	opts := &inferenceSpec.FetchCompletionOptions{
		CompletionKey: ck,
		CapabilityResolver: completionKeyCapabilityResolver{
			key:  ck,
			caps: derivedModelCapabilities,
		},
	}

	if req.OnStreamText != nil || req.OnStreamThinking != nil {
		opts.StreamHandler = makeStreamHandler(req.OnStreamText, req.OnStreamThinking)
		opts.StreamConfig = &inferenceSpec.StreamConfig{
			FlushIntervalMillis: 16,
			FlushChunkSize:      256,
		}
	}

	b, err := ps.inner.FetchCompletion(ctx, req.Provider, infReq, opts)

	resp := &spec.CompletionResponse{Body: &spec.CompletionResponseBody{
		InferenceResponse:     b,
		HydratedCurrentInputs: currentInputs,
	}}

	return resp, err
}

func buildSkillToolChoices(includeAll, includeRunScript bool) ([]inferenceSpec.ToolChoice, error) {
	mk := func(choiceID, toolName string, t llmtoolsSpec.Tool) (inferenceSpec.ToolChoice, error) {
		schema, err := decodeToolArgSchema(toolSpec.JSONRawString(t.ArgSchema))
		if err != nil {
			return inferenceSpec.ToolChoice{}, err
		}
		return inferenceSpec.ToolChoice{
			Type:        inferenceSpec.ToolTypeFunction,
			ID:          choiceID, // choiceID (ToolCall.choiceID)
			Name:        toolName, // ToolCall.name
			Description: t.Description,
			Arguments:   schema,
		}, nil
	}

	var out []inferenceSpec.ToolChoice
	tc, err := mk("builtin.skills-load", "skills-load", agentskillsSpec.SkillsLoadTool())
	if err != nil {
		return nil, err
	}
	out = append(out, tc)

	if includeAll {
		if tc, err = mk("builtin.skills-unload", "skills-unload", agentskillsSpec.SkillsUnloadTool()); err != nil {
			return nil, err
		}
		out = append(out, tc)
		if tc, err = mk(
			"builtin.skills-readresource",
			"skills-readresource",
			agentskillsSpec.SkillsReadResourceTool(),
		); err != nil {
			return nil, err
		}
		out = append(out, tc)
		if includeRunScript {
			if tc, err = mk(
				"builtin.skills-runscript",
				"skills-runscript",
				agentskillsSpec.SkillsRunScriptTool(),
			); err != nil {
				return nil, err
			}
			out = append(out, tc)
		}
	}
	return out, nil
}

func (ps *ProviderSetAPI) deriveCapabilitiesFromPreset(
	ctx context.Context,
	provider inferenceSpec.ProviderName,
	modelPresetID modelpresetSpec.ModelPresetID,
	requestModelName inferenceSpec.ModelName,
) (*inferenceSpec.ModelCapabilities, error) {
	if ps == nil || ps.inner == nil {
		return nil, errors.New("providerset is not initialized")
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

	// Load preset spec (contains provider+model overrides).
	presp, err := ps.mpStore.GetModelPreset(ctx, &modelpresetSpec.GetModelPresetRequest{
		ProviderName:    provider,
		ModelPresetID:   modelPresetID,
		IncludeDisabled: false,
	})
	if err != nil {
		return nil, err
	}
	if presp == nil || presp.Body == nil {
		return nil, errors.New("GetModelPresetSpec: empty response")
	}

	presetModelName := inferenceSpec.ModelName(presp.Body.Model.Name)

	// Hardening: ensure model name consistency.
	if requestModelName != "" && presetModelName != "" && requestModelName != presetModelName {
		return nil, fmt.Errorf("model name mismatch: request=%q preset=%q", requestModelName, presetModelName)
	}

	modelName := requestModelName
	if modelName == "" {
		modelName = presetModelName
	}
	if modelName == "" {
		return nil, errors.New("cannot derive capabilities: model name is empty")
	}

	// Base capabilities from inference-go/provider SDK.
	base, err := ps.inner.GetProviderCapability(ctx, provider)
	if err != nil {
		return nil, err
	}

	derived := cloneModelCapabilities(base)
	applyModelCapabilitiesOverride(&derived, presp.Body.Provider.CapabilitiesOverride)
	applyModelCapabilitiesOverride(&derived, presp.Body.Model.CapabilitiesOverride)

	return &derived, nil
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
		for i := len(body.History) - 1; i >= 0; i-- {
			if body.History[i].ModelParam != nil {
				mp = body.History[i].ModelParam
				break
			}
		}
	}
	if mp == nil {
		return nil, errors.New("no valid modelparam found")
	}

	if mp.MaxPromptLength == 0 {
		mp.MaxPromptLength = defaultMaxPromptTokens
	}

	return mp, nil
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

		out = append(out, turn.Inputs...)
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
		currentOut = append(currentOut, cur.Inputs...)
	}

	// Always process attachments into content items.
	msgContentItems, err := buildContentItemsFromAttachments(ctx, cur.Attachments)
	if err != nil {
		return nil, nil, err
	}

	if len(msgContentItems) > 0 {
		// Try to merge into the last user InputMessage if present.
		merged := false
		for i := len(currentOut) - 1; i >= 0; i-- {
			iu := &currentOut[i]
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

// outputToInput converts an OutputUnion from a previous completion into an
// InputUnion so it can be replayed as prior context in the next call.
func outputToInput(o inferenceSpec.OutputUnion) *inferenceSpec.InputUnion {
	switch o.Kind {
	case inferenceSpec.OutputKindOutputMessage:
		return &inferenceSpec.InputUnion{
			Kind:          inferenceSpec.InputKindOutputMessage,
			OutputMessage: o.OutputMessage,
		}
	case inferenceSpec.OutputKindReasoningMessage:
		return &inferenceSpec.InputUnion{
			Kind:             inferenceSpec.InputKindReasoningMessage,
			ReasoningMessage: o.ReasoningMessage,
		}
	case inferenceSpec.OutputKindFunctionToolCall:
		return &inferenceSpec.InputUnion{
			Kind:             inferenceSpec.InputKindFunctionToolCall,
			FunctionToolCall: o.FunctionToolCall,
		}
	case inferenceSpec.OutputKindCustomToolCall:
		return &inferenceSpec.InputUnion{
			Kind:           inferenceSpec.InputKindCustomToolCall,
			CustomToolCall: o.CustomToolCall,
		}
	case inferenceSpec.OutputKindWebSearchToolCall:
		return &inferenceSpec.InputUnion{
			Kind:              inferenceSpec.InputKindWebSearchToolCall,
			WebSearchToolCall: o.WebSearchToolCall,
		}
	case inferenceSpec.OutputKindWebSearchToolOutput:
		return &inferenceSpec.InputUnion{
			Kind:                inferenceSpec.InputKindWebSearchToolOutput,
			WebSearchToolOutput: o.WebSearchToolOutput,
		}
	default:
		// Unknown kinds are dropped.
		return nil
	}
}

func buildContentItemsFromAttachments(
	ctx context.Context,
	atts []attachment.Attachment,
) ([]inferenceSpec.InputOutputContentItemUnion, error) {
	items := make([]inferenceSpec.InputOutputContentItemUnion, 0)
	if len(atts) == 0 {
		return items, nil
	}

	blocks, err := attachment.BuildContentBlocks(
		ctx,
		atts,
		attachment.WithOverrideOriginalContentBlock(true),
		attachment.WithOnlyTextKindContentBlock(false),
	)
	if err != nil {
		return nil, err
	}

	for _, b := range blocks {
		switch b.Kind {
		case attachment.ContentBlockText:
			if b.Text == nil {
				continue
			}
			txt := strings.TrimSpace(*b.Text)
			if txt == "" {
				continue
			}
			formattedTxt, err := attachment.FormatTextBlockForLLM(b)
			if err != nil {
				return nil, err
			}
			if formattedTxt == "" {
				continue
			}
			items = append(items, inferenceSpec.InputOutputContentItemUnion{
				Kind: inferenceSpec.ContentItemKindText,
				TextItem: &inferenceSpec.ContentItemText{
					Text: formattedTxt,
				},
			})

		case attachment.ContentBlockImage:
			var data, urlStr string
			if b.Base64Data != nil {
				data = strings.TrimSpace(*b.Base64Data)
			}
			if b.URL != nil {
				urlStr = strings.TrimSpace(*b.URL)
			}
			// Require at least one of base64 or URL to be present.
			if data == "" && urlStr == "" {
				continue
			}
			mime := inferenceSpec.DefaultImageDataMIME
			if b.MIMEType != nil && strings.TrimSpace(*b.MIMEType) != "" {
				mime = strings.TrimSpace(*b.MIMEType)
			}
			name := ""
			if b.FileName != nil {
				name = strings.TrimSpace(*b.FileName)
			}
			img := &inferenceSpec.ContentItemImage{
				ImageName: name,
				ImageMIME: mime,
			}
			if data != "" {
				img.ImageData = data
			}
			if urlStr != "" {
				img.ImageURL = urlStr
			}
			items = append(items, inferenceSpec.InputOutputContentItemUnion{
				Kind:      inferenceSpec.ContentItemKindImage,
				ImageItem: img,
			})

		case attachment.ContentBlockFile:
			var data, urlStr string
			if b.Base64Data != nil {
				data = strings.TrimSpace(*b.Base64Data)
			}
			if b.URL != nil {
				urlStr = strings.TrimSpace(*b.URL)
			}
			// Require at least one of base64 or URL to be present.
			if data == "" && urlStr == "" {
				continue
			}
			mime := inferenceSpec.DefaultFileDataMIME
			if b.MIMEType != nil && strings.TrimSpace(*b.MIMEType) != "" {
				mime = strings.TrimSpace(*b.MIMEType)
			}
			name := ""
			if b.FileName != nil {
				name = strings.TrimSpace(*b.FileName)
			}

			file := &inferenceSpec.ContentItemFile{
				FileName: name,
				FileMIME: mime,
			}
			if data != "" {
				file.FileData = data
			}
			if urlStr != "" {
				file.FileURL = urlStr
			}

			items = append(items, inferenceSpec.InputOutputContentItemUnion{
				Kind:     inferenceSpec.ContentItemKindFile,
				FileItem: file,
			})
		}
	}

	return items, nil
}

func (ps *ProviderSetAPI) buildToolChoices(
	ctx context.Context,
	toolStoreChoices []toolSpec.ToolStoreChoice,
) ([]inferenceSpec.ToolChoice, error) {
	out := make([]inferenceSpec.ToolChoice, 0)
	if len(toolStoreChoices) == 0 {
		return nil, nil
	}

	if ps.toolStore == nil {
		return nil, errors.New("tool store not configured for provider set")
	}

	for _, sc := range toolStoreChoices {
		if sc.ChoiceID == "" || sc.BundleID == "" || sc.ToolSlug == "" || strings.TrimSpace(sc.ToolVersion) == "" {
			return nil, fmt.Errorf(
				"invalid tool store choice: choiceID/bundleID/toolSlug/toolVersion required: %+v",
				sc,
			)
		}
		tc, err := ps.hydrateToolChoice(ctx, sc)
		if err != nil {
			return nil, err
		}
		out = append(out, *tc)
	}

	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// hydrateToolChoice loads the Tool definition from tool-store and converts it
// into an inference-go ToolChoice. This is only called when we don't already
// have a ToolChoice persisted in the conversation for the same tool.
func (ps *ProviderSetAPI) hydrateToolChoice(
	ctx context.Context,
	sc toolSpec.ToolStoreChoice,
) (toolChoice *inferenceSpec.ToolChoice, err error) {
	if sc.ChoiceID == "" {
		return nil, errors.New("invalid choiceID for tool store choice")
	}
	req := &toolSpec.GetToolRequest{
		BundleID: sc.BundleID,
		ToolSlug: sc.ToolSlug,
		Version:  bundleitemutils.ItemVersion(sc.ToolVersion),
	}
	resp, err := ps.toolStore.GetTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to load tool %s/%s@%s: %w",
			sc.BundleID,
			sc.ToolSlug,
			sc.ToolVersion,
			err,
		)
	}
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf(
			"tool %s/%s@%s not found",
			sc.BundleID,
			sc.ToolSlug,
			sc.ToolVersion,
		)
	}
	tool := resp.Body
	if !tool.IsEnabled {
		return nil, fmt.Errorf(
			"tool %s/%s@%s is disabled",
			sc.BundleID,
			sc.ToolSlug,
			sc.ToolVersion,
		)
	}
	if !tool.LLMCallable {
		return nil, fmt.Errorf(
			"tool %s/%s@%s is not LLM-callable",
			sc.BundleID, sc.ToolSlug, sc.ToolVersion,
		)
	}
	name := string(sc.ToolSlug)
	desc := tool.Description
	if desc == "" {
		desc = sc.Description
	}

	tc := &inferenceSpec.ToolChoice{
		Type:        inferenceSpec.ToolType(sc.ToolType),
		ID:          sc.ChoiceID,
		Name:        name,
		Description: desc,
	}

	switch tool.Type {
	case toolSpec.ToolTypeGo, toolSpec.ToolTypeHTTP:
		argSchema, err := decodeToolArgSchema(string(tool.ArgSchema))
		if err != nil {
			return nil, fmt.Errorf(
				"invalid argSchema for %s/%s@%s: %w",
				sc.BundleID,
				sc.ToolSlug,
				sc.ToolVersion,
				err,
			)
		}
		tc.Arguments = argSchema

	case toolSpec.ToolTypeSDK:
		// SDK-backed server tools. Semantics come from sc.ToolType
		// (e.g., "webSearch"), while implementation is described by
		// tool.SDK and user configuration by tool.UserArgSchema plus
		// sc.Config.
		switch sc.ToolType {
		case toolSpec.ToolStoreChoiceTypeWebSearch:
			// Decode per-choice config (if any) and map to the
			// inference-go WebSearchToolChoiceItem.
			var cfg inferenceSpec.WebSearchToolChoiceItem
			rawCfg := strings.TrimSpace(sc.UserArgSchemaInstance)
			if rawCfg != "" {
				if err := json.Unmarshal([]byte(rawCfg), &cfg); err != nil {
					return nil, fmt.Errorf(
						"invalid config for webSearch tool %s/%s@%s: %w",
						sc.BundleID, sc.ToolSlug, sc.ToolVersion, err,
					)
				}
			}
			tc.Type = inferenceSpec.ToolTypeWebSearch
			tc.WebSearchArguments = &cfg

		default:
			// Future SDK-backed tool kinds (function/custom) could be added here.
			// For now, we treat anything other than webSearch as unsupported.
			return nil, fmt.Errorf(
				"unsupported ToolType %q for sdk tool %s/%s@%s",
				sc.ToolType, sc.BundleID, sc.ToolSlug, sc.ToolVersion,
			)
		}

	default:
		return nil, fmt.Errorf("unsupported tool impl type %q", tool.Type)
	}

	return tc, nil
}

func decodeToolArgSchema(raw toolSpec.JSONRawString) (map[string]any, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return map[string]any{"type": "object"}, nil
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(s), &schema); err != nil {
		return nil, err
	}
	if len(schema) == 0 {
		schema = map[string]any{"type": "object"}
	}
	return schema, nil
}

func skillsRulesPrompt(includeAll, includeRunScript bool) string {
	if !includeAll {
		return agentskillsSpec.SkillsRulesPromptLoadOnly
	}

	if !includeRunScript {
		return agentskillsSpec.SkillsRulesPromptWithoutRunScript
	}

	return agentskillsSpec.SkillsRulesPromptAll
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
			if onText != nil && ev.Text != nil {
				return onText(ev.Text.Text)
			}
		case inferenceSpec.StreamContentKindThinking:
			if onThinking != nil && ev.Thinking != nil {
				return onThinking(ev.Thinking.Text)
			}
		}
		return nil
	}
}
