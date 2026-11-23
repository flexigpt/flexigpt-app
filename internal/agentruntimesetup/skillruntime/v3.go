package skillruntime

import (
	"context"
	"errors"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/agentskills-go/document"
	"github.com/flexigpt/agentskills-go/provider"
	agentskillsRuntime "github.com/flexigpt/agentskills-go/runtime"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/toolruntime"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	v3SkillToolLoadName          = "skills-load"
	v3SkillToolUnloadName        = "skills-unload"
	v3SkillToolReadResourceName  = "skills-readresource"
	v3SkillToolRunScriptName     = "skills-runscript"
	v3SkillBindingVersion        = "session-v3-skill/v1"
	v3SkillPreparedChoicePrefix  = "prepared-choice:"
	v3SkillFingerprintPrefix     = "skill:"
	v3SkillReceiptSystem         = "skill"
	v3SkillInvalidBindingCode    = "invalid_skill_binding"
	v3SkillSessionFailedCode     = "skill_session_failed"
	v3SkillInvalidArgsCode       = "invalid_skill_arguments"
	v3SkillFailedCode            = "skill_failed"
	v3SkillEmptyCode             = "skill_empty"
	v3SkillStateUnavailableCode  = "skill_state_unavailable"
	v3SkillToolErrorCode         = "skill_tool_error"
	defaultSkillToolErrorMessage = "skill tool returned an error result"
	errV3SkillRuntimeUnavailable = "session v3 skill runtime is unavailable"
)

// V3Executor owns source-backed Skill selection and per-Execution Skill
// Runtime sessions. Session v3 only stores Artifact selections and generic
// Tool WorkItems.
type V3Executor struct {
	resolver *Resolver
	runtime  *skillRuntime.Service

	mu       sync.Mutex
	sessions map[V3SkillSessionKey]v3SkillSession
}

type V3SkillSessionKey struct {
	ExecutionID sessionv3Runtime.ExecutionID
	FrameID     sessionv3Runtime.ProgramFrameID
}

type v3SkillSession struct {
	id           string
	definitions  []provider.SkillDef
	state        sessionv3Runtime.SkillSelectionState
	byDefinition map[provider.SkillDef]sessionv3Runtime.ArtifactPin
}

func NewV3Executor(
	resolver *Resolver,
	runtime *skillRuntime.Service,
) (*V3Executor, error) {
	if resolver == nil || runtime == nil {
		return nil, errors.New("session v3 skill runtime dependencies are incomplete")
	}

	return &V3Executor{
		resolver: resolver,
		runtime:  runtime,
		sessions: make(map[V3SkillSessionKey]v3SkillSession),
	}, nil
}

// Choices exposes Skill lifecycle Tools. When an existing runtime session has
// active Skills, the full lifecycle set is visible; otherwise only loading is
// available.
func (e *V3Executor) Choices(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
) ([]toolruntime.V3PreparedChoice, error) {
	owner := v3SkillOwnerKey(request, nil)
	defaults := request.Session.Defaults
	if request.WorkItem.Input.LLM != nil &&
		request.WorkItem.Input.LLM.InvocationDefaults != nil {
		defaults = *request.WorkItem.Input.LLM.InvocationDefaults
	}
	active, err := e.EffectiveActiveSelectionsForFrame(
		request.Execution,
		owner.FrameID,
		defaults.Skills,
		defaults.ActiveSkills,
	)
	if err != nil {
		return nil, err
	}
	hydrated, err := e.HydrateLLM(
		ctx,
		owner.ExecutionID,
		owner.FrameID,
		defaults.Skills,
		active,
	)
	if err != nil {
		return nil, err
	}
	return hydrated.Choices, nil
}

// Instruction resolves and renders an instruction-mode Skill through the
// Skill domain. It does not create an Execution-owned Skill session.
func (e *V3Executor) Instruction(
	ctx context.Context,
	selection sessionv3Runtime.ArtifactSelection,
) (string, error) {
	_, text, err := e.renderSelectedSkill(
		ctx,
		selection,
		nil,
		document.SkillInsertInstructions,
	)
	return text, err
}

func (e *V3Executor) PrepareUserMessageTemplate(
	ctx context.Context,
	selection sessionv3Runtime.ArtifactSelection,
	arguments map[string]string,
) (sessionv3Runtime.UserMessageSkillTemplateResult, error) {
	pin, text, err := e.renderSelectedSkill(
		ctx,
		selection,
		arguments,
		document.SkillInsertUserMessage,
	)
	if err != nil {
		return sessionv3Runtime.UserMessageSkillTemplateResult{}, err
	}
	return sessionv3Runtime.UserMessageSkillTemplateResult{
		Skill: pin,
		Text:  text,
	}, nil
}

func (e *V3Executor) PreviewInstruction(
	ctx context.Context,
	selection sessionv3Runtime.ArtifactSelection,
	arguments map[string]string,
) (sessionv3Runtime.InstructionSkillPreviewResult, error) {
	pin, text, err := e.renderSelectedSkill(
		ctx,
		selection,
		arguments,
		document.SkillInsertInstructions,
	)
	if err != nil {
		return sessionv3Runtime.InstructionSkillPreviewResult{}, err
	}
	return sessionv3Runtime.InstructionSkillPreviewResult{
		Skill: pin,
		Text:  text,
	}, nil
}

// Bind verifies the Skill allow-list and active-skill selections before a
// Skill Tool WorkItem is dispatched. Runtime Session allocation remains lazy
// and occurs only inside Execute.
func (e *V3Executor) Bind(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
	payload *sessionv3Runtime.ToolWork,
) (sessionv3Runtime.SemanticBinding, error) {
	if e == nil || e.resolver == nil || e.runtime == nil {
		return sessionv3Runtime.SemanticBinding{}, errors.New(errV3SkillRuntimeUnavailable)
	}
	if payload == nil ||
		payload.Call.Binding.Kind != sessionv3Runtime.ToolBindingKindSkill ||
		payload.Call.Binding.Skill == nil ||
		payload.Call.Binding.Skill.Name != payload.Call.Name {
		return sessionv3Runtime.SemanticBinding{}, errors.New(
			"skill tool binding does not match the requested tool",
		)
	}
	if err := payload.Call.Validate(); err != nil {
		return sessionv3Runtime.SemanticBinding{}, err
	}
	skills, active, originErr := v3SkillSelectionsForTool(
		request,
		payload,
	)
	if originErr != nil {
		return sessionv3Runtime.SemanticBinding{}, originErr
	}
	if len(skills) == 0 {
		return sessionv3Runtime.SemanticBinding{}, errors.New(
			"skill tool was requested without enabled skills",
		)
	}
	owner := v3SkillOwnerKey(request, payload)
	active, err := e.effectiveActiveSelections(
		request.Execution,
		owner.FrameID,
		skills,
		active,
	)
	if err != nil {
		return sessionv3Runtime.SemanticBinding{}, err
	}
	resolved, err := e.resolveV3Selections(
		ctx,
		skills,
		active,
	)
	if err != nil {
		return sessionv3Runtime.SemanticBinding{}, err
	}

	return sessionv3Runtime.SemanticBinding{
		Version:           v3SkillBindingVersion,
		Artifacts:         append([]sessionv3Runtime.ArtifactPin(nil), resolved.state.Enabled...),
		TargetFingerprint: v3SkillFingerprintPrefix + payload.Call.Name,
		BoundAt:           time.Now().UTC(),
	}, nil
}

// Execute handles only Skill Tool bindings.
func (e *V3Executor) Execute(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
	payload *sessionv3Runtime.ToolWork,
) (sessionv3Runtime.WorkExecutionResult, bool, error) {
	if payload == nil ||
		payload.Call.Binding.Kind != sessionv3Runtime.ToolBindingKindSkill {
		return sessionv3Runtime.WorkExecutionResult{}, false, nil
	}
	if payload.Call.Binding.Skill == nil ||
		payload.Call.Binding.Skill.Name != payload.Call.Name {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    v3SkillInvalidBindingCode,
				Message: "skill tool binding does not match the requested tool",
			},
		}, true, nil
	}

	skills, activeSelections, err := v3SkillSelectionsForTool(
		request,
		payload,
	)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{}, true, err
	}
	owner := v3SkillOwnerKey(request, payload)
	activeSelections, err = e.effectiveActiveSelections(
		request.Execution,
		owner.FrameID,
		skills,
		activeSelections,
	)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{}, true, err
	}
	resolved, err := e.resolveV3Selections(ctx, skills, activeSelections)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    v3SkillSessionFailedCode,
				Message: strings.ToLower(err.Error()),
			},
		}, true, nil
	}
	runtimeSession, err := e.ensureResolvedSkillSession(
		ctx,
		owner,
		resolved,
	)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    v3SkillSessionFailedCode,
				Message: strings.ToLower(err.Error()),
			},
		}, true, nil
	}

	arguments := payload.Call.Arguments.Clone()
	if len(arguments) == 0 {
		arguments = jsonutil.MustCanonicalValue(map[string]any{})
	}
	if !arguments.Valid() {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    v3SkillInvalidArgsCode,
				Message: "skill tool arguments are not canonical json",
			},
		}, true, nil
	}

	response, err := e.runtime.InvokeSkillTool(
		ctx,
		&skillRuntime.InvokeSkillToolRequest{
			Body: &skillRuntime.InvokeSkillToolRequestBody{
				SessionID: agentskillsRuntimeSpec.SessionID(runtimeSession.id),
				ToolName:  payload.Call.Name,
				Args:      string(arguments),
			},
		},
	)
	if err != nil {
		if v3SkillConversationSource(payload.Call.Source) && ctx.Err() == nil {
			activeState, stateErr := e.activeSkillSelectionState(ctx, owner, runtimeSession)
			if stateErr != nil {
				activeState = cloneV3SkillState(runtimeSession.state)
			}
			effect := sessionv3Runtime.EffectStateConfirmedNotApplied
			return sessionv3Runtime.WorkExecutionResult{
				Output:      v3SkillToolErrorOutput(err.Error(), &activeState),
				Confirmed:   true,
				EffectState: &effect,
			}, true, nil
		}
		effect := sessionv3Runtime.EffectStateUncertain
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:     v3SkillFailedCode,
				Category: sessionv3Runtime.ErrorCategoryUncertainty,
				Effect:   effect,
				Message:  strings.ToLower(err.Error()),
			},
			Confirmed:   false,
			EffectState: &effect,
		}, true, nil
	}
	if response == nil || response.Body == nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    v3SkillEmptyCode,
				Message: "skill tool returned no response",
			},
		}, true, nil
	}

	adapted, err := toolruntime.AdaptToolOutputs(response.Body.Outputs)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{}, true, err
	}
	structured, err := jsonutil.CanonicalValueFrom(
		adapted.Structured,
	)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{}, true, err
	}
	activeState, stateErr := e.activeSkillSelectionState(
		ctx,
		owner,
		runtimeSession,
	)
	if stateErr != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:     v3SkillStateUnavailableCode,
				Category: sessionv3Runtime.ErrorCategoryUncertainty,
				Message:  strings.ToLower(stateErr.Error()),
			},
			Confirmed: false,
		}, true, nil
	}

	kind := sessionv3Runtime.ToolOutputKindSuccess
	errorText := ""
	if response.Body.IsError {
		kind = sessionv3Runtime.ToolOutputKindError
		errorText = firstV3SkillText(
			response.Body.ErrorMessage,
			defaultSkillToolErrorMessage,
		)
		if strings.TrimSpace(adapted.Text) == "" {
			adapted.Text = errorText
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
				SkillState: &activeState,
			},
		},
		Confirmed: true,
	}
	if !response.Body.IsError {
		result.Output.Receipt = &sessionv3Runtime.ExternalReceipt{
			System:      v3SkillReceiptSystem,
			Reference:   v3SkillFingerprintPrefix + payload.Call.Name + ":" + payload.Call.ID,
			ConfirmedAt: time.Now().UTC(),
		}
		return result, true, nil
	}
	if v3SkillConversationSource(payload.Call.Source) && ctx.Err() == nil {
		effect := sessionv3Runtime.EffectStateConfirmedNotApplied
		result.Confirmed = true
		result.EffectState = &effect
		return result, true, nil
	}
	if ctx.Err() != nil {
		effect := sessionv3Runtime.EffectStateUncertain
		result.Confirmed = false
		result.EffectState = &effect
		result.Error = &sessionv3Runtime.SanitizedError{
			Code:     v3SkillToolErrorCode,
			Category: sessionv3Runtime.ErrorCategoryUncertainty,
			Effect:   effect,
			Message:  strings.ToLower(errorText),
		}
		return result, true, nil
	}
	effect := sessionv3Runtime.EffectStateConfirmedNotApplied
	result.Confirmed = true
	result.EffectState = &effect
	result.Error = &sessionv3Runtime.SanitizedError{
		Code:     v3SkillToolErrorCode,
		Category: sessionv3Runtime.ErrorCategoryExecution,
		Effect:   effect,
		Message:  strings.ToLower(errorText),
	}
	return result, true, nil
}

func v3SkillConversationSource(source sessionv3Runtime.ToolCallSource) bool {
	return source == sessionv3Runtime.ToolCallSourceModel ||
		source == sessionv3Runtime.ToolCallSourceApp
}

func v3SkillToolErrorOutput(
	message string,
	state *sessionv3Runtime.SkillSelectionState,
) *sessionv3Runtime.WorkItemOutput {
	text := firstV3SkillText(message, defaultSkillToolErrorMessage)
	value := jsonutil.MustCanonicalValue(struct {
		Kind    sessionv3Runtime.ToolOutputKind `json:"kind"`
		Message string                          `json:"message"`
	}{
		Kind:    sessionv3Runtime.ToolOutputKindError,
		Message: text,
	})
	var copiedState *sessionv3Runtime.SkillSelectionState
	if state != nil {
		cloned := cloneV3SkillState(*state)
		copiedState = &cloned
	}
	return &sessionv3Runtime.WorkItemOutput{
		Value: value,
		Tool: &sessionv3Runtime.ToolOutput{
			Kind:       sessionv3Runtime.ToolOutputKindError,
			Text:       text,
			Structured: value.Clone(),
			Error:      text,
			SkillState: copiedState,
		},
	}
}

// CloseExecution closes the process-local Skill Runtime session associated
// with one v3 Execution.
func (e *V3Executor) CloseExecution(
	ctx context.Context,
	executionID sessionv3Runtime.ExecutionID,
) error {
	if e == nil || executionID == "" {
		return nil
	}

	e.mu.Lock()
	sessions := make([]v3SkillSession, 0)
	for key, value := range e.sessions {
		if key.ExecutionID != executionID {
			continue
		}
		delete(e.sessions, key)
		sessions = append(sessions, value)
	}
	e.mu.Unlock()

	var output error
	for _, session := range sessions {
		if session.id == "" {
			continue
		}
		output = errors.Join(output, e.runtime.CloseSession(
			ctx,
			agentskillsRuntimeSpec.SessionID(session.id),
		))
	}
	return output
}

// CloseProgramFrame closes the private Skill Runtime session owned by one
// completed ProgramFrame. It is independent from whole-Execution cleanup so
// long-lived conversation Executions do not retain completed Agent resources.
func (e *V3Executor) CloseProgramFrame(
	ctx context.Context,
	executionID sessionv3Runtime.ExecutionID,
	frameID sessionv3Runtime.ProgramFrameID,
) error {
	if e == nil || executionID == "" || frameID == "" {
		return nil
	}

	key := V3SkillSessionKey{
		ExecutionID: executionID,
		FrameID:     frameID,
	}
	e.mu.Lock()
	value, found := e.sessions[key]
	if found {
		delete(e.sessions, key)
	}
	e.mu.Unlock()

	if !found || value.id == "" {
		return nil
	}
	return e.runtime.CloseSession(
		ctx,
		agentskillsRuntimeSpec.SessionID(value.id),
	)
}

func (e *V3Executor) renderSelectedSkill(
	ctx context.Context,
	selection sessionv3Runtime.ArtifactSelection,
	arguments map[string]string,
	expectedInsert document.SkillInsert,
) (sessionv3Runtime.ArtifactPin, string, error) {
	if e == nil || e.resolver == nil || e.runtime == nil {
		return sessionv3Runtime.ArtifactPin{}, "", errors.New(errV3SkillRuntimeUnavailable)
	}

	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(selection.RootID),
		ArtifactID: artifactModel.ArtifactID(selection.ArtifactID),
	}
	if err := ref.Validate(); err != nil {
		return sessionv3Runtime.ArtifactPin{}, "", err
	}

	resolved, err := e.resolver.ResolveSkills(
		ctx,
		[]artifactModel.ArtifactRef{ref},
	)
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, "", err
	}
	if len(resolved.Values) != 1 {
		return sessionv3Runtime.ArtifactPin{}, "", errors.New(
			"skill resolver returned an unexpected selected skill count",
		)
	}

	value := resolved.Values[0]
	if value.Artifact != ref {
		return sessionv3Runtime.ArtifactPin{}, "", errors.New(
			"skill resolver returned another selected skill",
		)
	}
	if selection.ExpectedRevision != 0 &&
		selection.ExpectedRevision != value.Revision {
		return sessionv3Runtime.ArtifactPin{}, "", errors.New(
			"selected skill revision changed",
		)
	}
	if selection.ExpectedDigest != "" &&
		selection.ExpectedDigest != value.DefinitionDigest {
		return sessionv3Runtime.ArtifactPin{}, "", errors.New(
			"selected skill definition changed",
		)
	}
	if !value.Enabled {
		return sessionv3Runtime.ArtifactPin{}, "", errors.New(
			"selected skill is disabled",
		)
	}

	records, err := e.runtime.ListAgentSkills(
		ctx,
		&agentskillsRuntime.SkillListFilter{
			AllowSkills: []provider.SkillDef{value.Definition},
		},
	)
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, "", err
	}
	for _, record := range records {
		if record.Def != value.Definition {
			continue
		}
		insert, supported := document.NormalizeSkillInsert(record.Insert)
		if !supported || insert != expectedInsert {
			return sessionv3Runtime.ArtifactPin{}, "", errors.New(
				"selected skill has a different insertion behavior",
			)
		}

		rendered, err := e.runtime.RenderSkill(
			ctx,
			&skillRuntime.RenderSkillRequest{
				Body: &skillRuntime.RenderSkillRequestBody{
					Definition: value.Definition,
					Arguments:  cloneV3SkillArguments(arguments),
				},
			},
		)
		if err != nil {
			return sessionv3Runtime.ArtifactPin{}, "", err
		}
		if rendered == nil || rendered.Body == nil {
			return sessionv3Runtime.ArtifactPin{}, "", errors.New(
				"skill runtime returned no rendered skill",
			)
		}
		return sessionv3Runtime.ArtifactPin{
			RootID:     string(value.Artifact.RootID),
			ArtifactID: string(value.Artifact.ArtifactID),
			Revision:   value.Revision,
			Digest:     value.DefinitionDigest,
		}, rendered.Body.Text, nil
	}

	return sessionv3Runtime.ArtifactPin{}, "", errors.New(
		"skill runtime did not index the selected skill",
	)
}

func cloneV3SkillArguments(
	value map[string]string,
) map[string]string {
	if len(value) == 0 {
		return map[string]string{}
	}
	output := make(map[string]string, len(value))
	maps.Copy(output, value)
	return output
}

func (e *V3Executor) executionSession(
	key V3SkillSessionKey,
) v3SkillSession {
	if e == nil || key.ExecutionID == "" || key.FrameID == "" {
		return v3SkillSession{}
	}

	e.mu.Lock()
	value := e.sessions[key]
	e.mu.Unlock()
	return cloneV3SkillSession(value)
}

func v3SkillChoices(
	includeAll bool,
	includeRunScript bool,
) ([]toolruntime.V3PreparedChoice, error) {
	choices, err := buildSkillToolChoices(includeAll, includeRunScript)
	if err != nil {
		return nil, err
	}
	output := make([]toolruntime.V3PreparedChoice, 0, len(choices))
	for _, choice := range choices {
		call := sessionv3Runtime.ToolCall{
			ID:          v3SkillPreparedChoicePrefix + choice.ID,
			Name:        choice.Name,
			ChoiceID:    choice.ID,
			Type:        choice.Type,
			DisplayName: v3SkillToolDisplayName(choice.Name),
			Description: choice.Description,
			Source:      sessionv3Runtime.ToolCallSourceModel,
			Automatic: choice.Name == v3SkillToolLoadName ||
				choice.Name == v3SkillToolUnloadName ||
				choice.Name == v3SkillToolReadResourceName,
			Binding: sessionv3Runtime.ToolBinding{
				Kind: sessionv3Runtime.ToolBindingKindSkill,
				Skill: &sessionv3Runtime.SkillToolBinding{
					Name: choice.Name,
				},
			},
		}
		output = append(output, toolruntime.V3PreparedChoice{
			Choice: choice,
			Call:   call,
		})
	}
	return output, nil
}

func v3SkillToolDisplayName(name string) string {
	switch name {
	case v3SkillToolLoadName:
		return "Load Skill"
	case v3SkillToolUnloadName:
		return "Unload Skill"
	case v3SkillToolReadResourceName:
		return "Read Skill resource"
	case v3SkillToolRunScriptName:
		return "Run Skill script"
	default:
		return "Skill Tool"
	}
}

func v3SkillOwnerKey(
	request sessionv3Runtime.WorkDispatchRequest,
	payload *sessionv3Runtime.ToolWork,
) V3SkillSessionKey {
	frameID := request.WorkItem.FrameID
	if payload != nil && payload.Origin != nil &&
		payload.Origin.OriginWorkItemID != "" {
		for _, item := range request.Execution.WorkItems {
			if item.ID != payload.Origin.OriginWorkItemID {
				continue
			}
			frameID = item.FrameID
			break
		}
	}
	return V3SkillSessionKey{
		ExecutionID: request.Execution.ID,
		FrameID:     frameID,
	}
}

func v3SkillArtifactKey(rootID, artifactID string) string {
	return rootID + "\x00" + artifactID
}

func firstV3SkillText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func v3SkillSelectionsForTool(
	request sessionv3Runtime.WorkDispatchRequest,
	payload *sessionv3Runtime.ToolWork,
) (
	skills []sessionv3Runtime.ArtifactSelection,
	active []sessionv3Runtime.ArtifactSelection,
	err error,
) {
	if payload == nil {
		return nil, nil, errors.New("skill tool payload is unavailable")
	}
	if payload.Origin == nil {
		if payload.Call.Source == sessionv3Runtime.ToolCallSourceModel {
			return nil, nil, errors.New(
				"generated skill tool has no immutable llm invocation origin",
			)
		}
		return request.Session.Defaults.Skills,
			request.Session.Defaults.ActiveSkills,
			nil
	}
	originPlan, originErr := sessionv3Runtime.LLMOriginPlan(
		request.Session,
		*payload.Origin,
	)
	if originErr != nil {
		return nil, nil, originErr
	}

	skills = make(
		[]sessionv3Runtime.ArtifactSelection,
		0,
		len(originPlan.Skills),
	)
	active = make(
		[]sessionv3Runtime.ArtifactSelection,
		0,
		len(originPlan.Skills),
	)
	for _, value := range originPlan.Skills {
		selection := value.Artifact.Selection()
		skills = append(skills, selection)
		if value.Active {
			active = append(active, selection)
		}
	}
	return skills, active, nil
}
