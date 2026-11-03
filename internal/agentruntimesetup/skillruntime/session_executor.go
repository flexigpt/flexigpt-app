package skillruntime

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"

	"github.com/flexigpt/agentskills-go/provider"
	agentskillsRuntime "github.com/flexigpt/agentskills-go/runtime"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/toolruntime"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
)

const (
	skillToolLoadName   = "skills-load"
	skillToolUnloadName = "skills-unload"
)

type sessionSkillState struct {
	definitions  []provider.SkillDef
	byArtifact   map[string]provider.SkillDef
	byDefinition map[provider.SkillDef]sessionRuntime.ArtifactReference
}

// SessionToolExecutor owns Session-facing Skill Runtime execution. It resolves
// durable Artifact references itself and does not call back into the Session
// coordinator through an inversion interface.
type SessionToolExecutor struct {
	resolver *Resolver
	runtime  *skillRuntime.Service

	mu       sync.RWMutex
	sessions map[string]sessionSkillState
}

func NewSessionToolExecutor(
	resolver *Resolver,
	runtime *skillRuntime.Service,
) (*SessionToolExecutor, error) {
	if resolver == nil || runtime == nil {
		return nil, errors.New(
			"session Skill Tool executor dependencies are incomplete",
		)
	}

	return &SessionToolExecutor{
		resolver: resolver,
		runtime:  runtime,
		sessions: make(map[string]sessionSkillState),
	}, nil
}

func (e *SessionToolExecutor) EnsureSkillSession(
	ctx context.Context,
	request sessionRuntime.SkillSessionRequest,
) (sessionRuntime.SkillSessionResult, error) {
	if len(request.Enabled) == 0 {
		return sessionRuntime.SkillSessionResult{}, errors.New(
			"skill Session was requested without enabled Skills",
		)
	}

	state, err := e.resolveSkills(ctx, request.Enabled)
	if err != nil {
		return sessionRuntime.SkillSessionResult{}, err
	}

	active := make(
		[]provider.SkillDef,
		0,
		len(request.Active),
	)
	for _, reference := range request.Active {
		definition, found := state.byArtifact[artifactReferenceKey(reference)]
		if !found {
			return sessionRuntime.SkillSessionResult{}, errors.New(
				"active Skill is not part of the immutable operation preparation",
			)
		}
		active = append(active, definition)
	}

	response, err := e.runtime.CreateSkillSession(
		ctx,
		&skillRuntime.CreateSkillSessionRequest{
			Body: &skillRuntime.CreateSkillSessionRequestBody{
				CloseSessionID: agentskillsRuntimeSpec.SessionID(
					request.SessionID,
				),
				AllowedSkills: state.definitions,
				ActiveSkills:  active,
			},
		},
	)
	if err != nil {
		return sessionRuntime.SkillSessionResult{}, err
	}
	if response == nil ||
		response.Body == nil ||
		response.Body.SessionID == "" {
		return sessionRuntime.SkillSessionResult{}, errors.New(
			"skill runtime returned no Session identity",
		)
	}

	sessionID := string(response.Body.SessionID)
	e.mu.Lock()
	if request.SessionID != "" && request.SessionID != sessionID {
		delete(e.sessions, request.SessionID)
	}
	e.sessions[sessionID] = cloneSessionSkillState(state)
	e.mu.Unlock()

	result := sessionRuntime.SkillSessionResult{
		SessionID: sessionID,
		Active:    make([]sessionRuntime.ArtifactReference, 0, len(response.Body.ActiveSkills)),
	}
	for _, definition := range response.Body.ActiveSkills {
		if reference, found := state.byDefinition[definition]; found {
			result.Active = append(result.Active, reference)
		}
	}
	return result, nil
}

func (e *SessionToolExecutor) CloseSkillSession(
	ctx context.Context,
	sessionID string,
) error {
	if err := e.runtime.CloseSession(
		ctx,
		agentskillsRuntimeSpec.SessionID(sessionID),
	); err != nil {
		return err
	}

	e.mu.Lock()
	delete(e.sessions, sessionID)
	e.mu.Unlock()
	return nil
}

func (e *SessionToolExecutor) EvaluateTool(
	_ context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolEvaluation, error) {
	if request.SkillSessionID == "" {
		return sessionRuntime.ToolEvaluation{
			Decision: sessionRuntime.ToolEvaluationDenied,
			Reason:   "the Run has no active Skill Session",
			Result: &sessionRuntime.ToolExecutionResult{
				HasResult: true,
				Kind:      sessionRuntime.ToolResultKindError,
				Text:      "Skill Tool error: no active Skill Session exists for this Run.",
				Error:     "no active Skill Session",
			},
		}, nil
	}

	return sessionRuntime.ToolEvaluation{
		Decision: sessionRuntime.ToolEvaluationAllowed,
	}, nil
}

func (e *SessionToolExecutor) ExecuteTool(
	ctx context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolExecutionResult, error) {
	if strings.TrimSpace(request.SkillSessionID) == "" {
		return sessionRuntime.ToolExecutionResult{}, errors.New(
			"skill Tool execution requires a Skill Session",
		)
	}

	state, found := e.sessionState(request.SkillSessionID)
	if !found {
		return sessionRuntime.ToolExecutionResult{}, errors.New(
			"skill Tool execution has no registered Skill Session",
		)
	}

	response, err := e.runtime.InvokeSkillTool(
		ctx,
		&skillRuntime.InvokeSkillToolRequest{
			Body: &skillRuntime.InvokeSkillToolRequestBody{
				SessionID: agentskillsRuntimeSpec.SessionID(
					request.SkillSessionID,
				),
				ToolName: request.ToolCall.Name,
				Args:     string(request.ToolCall.Arguments),
			},
		},
	)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}
	if response == nil || response.Body == nil {
		return sessionRuntime.ToolExecutionResult{}, errors.New(
			"skill Tool invocation returned no response body",
		)
	}

	structured, err := toolruntime.CanonicalToolOutputs(
		response.Body.Outputs,
	)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}

	result := sessionRuntime.ToolExecutionResult{
		HasResult:  true,
		Kind:       sessionRuntime.ToolResultKindSuccess,
		Text:       string(structured),
		Contents:   toolruntime.OutputItemsFromLLMToolOutputs(response.Body.Outputs),
		Structured: structured,
	}
	if response.Body.IsError {
		result.Kind = sessionRuntime.ToolResultKindError
		result.Error = response.Body.ErrorMessage
		if strings.TrimSpace(result.Error) == "" {
			result.Error = "Skill Tool returned an error result"
		}
	}

	if isSkillLifecycleTool(request.ToolCall.Name) {
		active, activeErr := e.activeSkillReferences(
			ctx,
			request.SkillSessionID,
			state,
		)
		if activeErr == nil {
			result.SkillSessionActive = active
			result.SkillSessionActiveKnown = true
		}
	}
	return result, nil
}

func (e *SessionToolExecutor) ResolveToolWait(
	_ context.Context,
	_ sessionRuntime.ToolWaitResolutionRequest,
) (sessionRuntime.ToolWaitResolution, error) {
	return sessionRuntime.ToolWaitResolution{}, errors.New(
		"skill tool execution does not expose an approval wait",
	)
}

func (e *SessionToolExecutor) resolveSkills(
	ctx context.Context,
	references []sessionRuntime.ArtifactReference,
) (sessionSkillState, error) {
	refs := make([]artifactModel.ArtifactRef, 0, len(references))
	seen := make(map[string]struct{}, len(references))

	for _, reference := range references {
		key := artifactReferenceKey(reference)
		if _, duplicate := seen[key]; duplicate {
			return sessionSkillState{}, fmt.Errorf(
				"duplicate Skill Artifact reference %q",
				key,
			)
		}
		seen[key] = struct{}{}

		ref := artifactModel.ArtifactRef{
			RootID:     rootModel.RootID(reference.RootID),
			ArtifactID: artifactModel.ArtifactID(reference.ArtifactID),
		}
		if err := ref.Validate(); err != nil {
			return sessionSkillState{}, err
		}
		refs = append(refs, ref)
	}

	resolved, err := e.resolver.ResolveSkills(ctx, refs)
	if err != nil {
		return sessionSkillState{}, err
	}
	if len(resolved.Values) != len(refs) {
		return sessionSkillState{}, errors.New(
			"skill resolver returned an unexpected Skill count",
		)
	}

	state := sessionSkillState{
		definitions:  make([]provider.SkillDef, 0, len(resolved.Values)),
		byArtifact:   make(map[string]provider.SkillDef, len(resolved.Values)),
		byDefinition: make(map[provider.SkillDef]sessionRuntime.ArtifactReference, len(resolved.Values)),
	}
	for _, value := range resolved.Values {
		reference := sessionRuntime.ArtifactReference{
			RootID:     string(value.Artifact.RootID),
			ArtifactID: string(value.Artifact.ArtifactID),
		}
		key := artifactReferenceKey(reference)
		if _, requested := seen[key]; !requested {
			return sessionSkillState{}, errors.New(
				"skill resolver returned an Artifact that was not requested",
			)
		}
		if !value.Enabled {
			return sessionSkillState{}, fmt.Errorf(
				"selected Skill %q is disabled",
				value.Artifact.ArtifactID,
			)
		}

		state.definitions = append(state.definitions, value.Definition)
		state.byArtifact[key] = value.Definition
		state.byDefinition[value.Definition] = reference
	}
	return state, nil
}

func (e *SessionToolExecutor) sessionState(
	sessionID string,
) (sessionSkillState, bool) {
	e.mu.RLock()
	state, found := e.sessions[sessionID]
	e.mu.RUnlock()
	if !found {
		return sessionSkillState{}, false
	}
	return cloneSessionSkillState(state), true
}

func (e *SessionToolExecutor) activeSkillReferences(
	ctx context.Context,
	sessionID string,
	state sessionSkillState,
) ([]sessionRuntime.ArtifactReference, error) {
	records, err := e.runtime.ListAgentSkills(
		ctx,
		&agentskillsRuntime.SkillListFilter{
			SessionID: agentskillsRuntimeSpec.SessionID(sessionID),
			Activity:  agentskillsRuntimeSpec.SkillActivity("active"),
		},
	)
	if err != nil {
		return nil, err
	}

	output := make(
		[]sessionRuntime.ArtifactReference,
		0,
		len(records),
	)
	for _, record := range records {
		if reference, found := state.byDefinition[record.Def]; found {
			output = append(output, reference)
		}
	}
	return output, nil
}

func cloneSessionSkillState(input sessionSkillState) sessionSkillState {
	output := sessionSkillState{
		definitions: append([]provider.SkillDef(nil), input.definitions...),
		byArtifact: make(
			map[string]provider.SkillDef,
			len(input.byArtifact),
		),
		byDefinition: make(
			map[provider.SkillDef]sessionRuntime.ArtifactReference,
			len(input.byDefinition),
		),
	}
	maps.Copy(output.byArtifact, input.byArtifact)
	maps.Copy(output.byDefinition, input.byDefinition)
	return output
}

func artifactReferenceKey(
	value sessionRuntime.ArtifactReference,
) string {
	return "artifact:" + value.RootID + ":" + value.ArtifactID
}

func isSkillLifecycleTool(name string) bool {
	switch name {
	case skillToolLoadName, skillToolUnloadName:
		return true
	default:
		return false
	}
}
