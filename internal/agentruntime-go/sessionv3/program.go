package sessionv3

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	BuiltinConversationInterpreterID  = "builtin.conversation"
	BuiltinAgentLoopInterpreterID     = "builtin.agent-loop"
	BuiltinSingleModelInterpreterID   = "builtin.single-model"
	BuiltinRetryBoundaryInterpreterID = "builtin.retry-boundary"
	BuiltinInterpreterVersion         = uint32(1)

	conversationFramePathPrefix = "conversation/"
	agentLoopToolPathPrefix     = "tools/"
	agentLoopSkillWorkPrefix    = "skill-lifecycle:"
	agentLoopLLMPathFormat      = "llm/%d"
	singleModelWorkPath         = "model"
	providerManagedToolSummary  = "Provider managed tool observed."
	errAgentLoopInputRequired   = "agent-loop input activity is required"
	errSingleModelInputRequired = "single-model input activity is required"
	errUnsupportedConvProgram   = "unsupported conversation program version"
	errUnsupportedConvState     = "unsupported conversation state version"
	errUnsupportedAgentProgram  = "unsupported agent-loop program version"
	errUnsupportedAgentState    = "unsupported agent-loop state version"
	errUnsupportedModelProgram  = "unsupported single-model program version"
	errUnsupportedModelState    = "unsupported single-model state version"
	llmOutputMissingCode        = "llm_output_missing"
	llmOutputMissingMessage     = "llm work item completed without llm output"
	llmToolOriginMissingCode    = "llm_tool_origin_missing"
	llmWorkFailedCode           = "llm_work_failed"
	llmWorkCancelledMessage     = "llm work item was cancelled"
	llmWorkFailedMessage        = "llm work item did not complete successfully"
	modelOutputMissingCode      = "model_output_missing"
	modelOutputMissingMessage   = "single-model program frame has no output"
)

type (
	conversationProgramPayload struct {
		Defaults SessionDefaults `json:"defaults"`
	}
	conversationQueuedActivity struct {
		ID                 ActivityID       `json:"id"`
		InvocationDefaults *SessionDefaults `json:"invocationDefaults,omitempty"`
		InvocationBudget   *ExecutionBudget `json:"invocationBudget,omitempty"`
	}
	conversationProgramState struct {
		// QueuedActivityIDs preserves recovery compatibility for documents
		// created before immutable message invocation facts existed.
		QueuedActivityIDs []ActivityID                 `json:"queuedActivityIDs,omitempty"`
		QueuedActivities  []conversationQueuedActivity `json:"queuedActivities,omitempty"`
		ActiveFrameID     ProgramFrameID               `json:"activeFrameID,omitempty"`
	}
)

type agentLoopProgramPayload struct {
	Defaults           SessionDefaults             `json:"defaults"`
	InputActivityID    ActivityID                  `json:"inputActivityID,omitempty"`
	IsolatedInput      bool                        `json:"isolatedInput,omitempty"`
	HistoryActivityIDs []ActivityID                `json:"historyActivityIDs,omitempty"`
	ProviderTools      []ProviderManagedToolChoice `json:"providerTools,omitempty"`
	PreparedInvocation *WorkflowPreparedInvocation `json:"preparedInvocation,omitempty"`
}

type agentLoopProgramState struct {
	InputActivityID           ActivityID                   `json:"inputActivityID"`
	CompletedInputActivityIDs []ActivityID                 `json:"completedInputActivityIDs,omitempty"`
	LastLLMWorkItemID         WorkItemID                   `json:"lastLLMWorkItemID,omitempty"`
	PendingToolIDs            []WorkItemID                 `json:"pendingToolIDs,omitempty"`
	QueuedActivityIDs         []ActivityID                 `json:"queuedActivityIDs,omitempty"`
	QueuedActivities          []conversationQueuedActivity `json:"queuedActivities,omitempty"`
	InvocationDefaults        *SessionDefaults             `json:"invocationDefaults,omitempty"`
	InvocationBudget          *ExecutionBudget             `json:"invocationBudget,omitempty"`
	Round                     int                          `json:"round"`
	AwaitingContinue          bool                         `json:"awaitingContinue"`
	ContinueRequested         bool                         `json:"continueRequested,omitempty"`
}

type singleModelProgramPayload struct {
	Defaults           SessionDefaults             `json:"defaults"`
	InputActivityID    ActivityID                  `json:"inputActivityID,omitempty"`
	IsolatedInput      bool                        `json:"isolatedInput,omitempty"`
	HistoryActivityIDs []ActivityID                `json:"historyActivityIDs,omitempty"`
	ProviderTools      []ProviderManagedToolChoice `json:"providerTools,omitempty"`
	PreparedInvocation *WorkflowPreparedInvocation `json:"preparedInvocation,omitempty"`
}

type singleModelProgramState struct {
	InputActivityID ActivityID `json:"inputActivityID"`
	WorkItemID      WorkItemID `json:"workItemID,omitempty"`
}

func NewBuiltinInterpreters() []ProgramInterpreter {
	return []ProgramInterpreter{
		conversationInterpreter{},
		agentLoopInterpreter{},
		singleModelInterpreter{},
		retryBoundaryInterpreter{},
	}
}

func NewConversationProgram(defaults SessionDefaults) (ProgramSnapshot, error) {
	return programSnapshot(BuiltinConversationInterpreterID, conversationProgramPayload{Defaults: defaults})
}

func NewAgentLoopProgram(
	defaults SessionDefaults,
	input ActivityID,
) (ProgramSnapshot, error) {
	return newAgentLoopProgram(defaults, input, false, nil)
}

// NewAgentLoopInvocationProgram creates an Agent-loop ProgramFrame with an
// immutable explicit message boundary. It is reusable by any orchestrator
// that must not inherit ambient Session history.
func NewAgentLoopInvocationProgram(
	defaults SessionDefaults,
	input ActivityID,
	history []ActivityID,
) (ProgramSnapshot, error) {
	return newAgentLoopWorkflowInvocationProgram(
		defaults,
		input,
		history,
		nil,
	)
}

func NewAgentLoopWorkflowInvocationProgram(
	defaults SessionDefaults,
	input ActivityID,
	history []ActivityID,
	providerTools []ProviderManagedToolChoice,
	prepared ...*WorkflowPreparedInvocation,
) (ProgramSnapshot, error) {
	return newAgentLoopWorkflowInvocationProgram(
		defaults,
		input,
		history,
		providerTools,
		prepared...,
	)
}

func newAgentLoopProgram(
	defaults SessionDefaults,
	input ActivityID,
	isolated bool,
	history []ActivityID,
) (ProgramSnapshot, error) {
	if input == "" {
		return ProgramSnapshot{}, errors.New(errAgentLoopInputRequired)
	}
	return programSnapshot(
		BuiltinAgentLoopInterpreterID,
		agentLoopProgramPayload{
			Defaults:           defaults,
			InputActivityID:    input,
			IsolatedInput:      isolated,
			HistoryActivityIDs: append([]ActivityID(nil), history...),
		},
	)
}

func newAgentLoopWorkflowInvocationProgram(
	defaults SessionDefaults,
	input ActivityID,
	history []ActivityID,
	providerTools []ProviderManagedToolChoice,
	prepared ...*WorkflowPreparedInvocation,
) (ProgramSnapshot, error) {
	if input == "" {
		return ProgramSnapshot{}, errors.New(errAgentLoopInputRequired)
	}
	return programSnapshot(BuiltinAgentLoopInterpreterID, agentLoopProgramPayload{
		Defaults:           defaults,
		InputActivityID:    input,
		IsolatedInput:      true,
		HistoryActivityIDs: append([]ActivityID(nil), history...),
		ProviderTools:      jsonutil.CloneJSONInValue(providerTools),
		PreparedInvocation: optionalWorkflowPreparedInvocation(prepared),
	})
}

func NewSingleModelProgram(
	defaults SessionDefaults,
	input ActivityID,
) (ProgramSnapshot, error) {
	return newSingleModelProgram(defaults, input, false, nil)
}

// NewSingleModelInvocationProgram creates a single-model ProgramFrame with
// an immutable explicit message boundary. It does not synthesize a user
// message from arbitrary caller data.
func NewSingleModelInvocationProgram(
	defaults SessionDefaults,
	input ActivityID,
	history []ActivityID,
) (ProgramSnapshot, error) {
	return newSingleModelWorkflowInvocationProgram(
		defaults,
		input,
		history,
		nil,
	)
}

func NewSingleModelWorkflowInvocationProgram(
	defaults SessionDefaults,
	input ActivityID,
	history []ActivityID,
	providerTools []ProviderManagedToolChoice,
	prepared ...*WorkflowPreparedInvocation,
) (ProgramSnapshot, error) {
	return newSingleModelWorkflowInvocationProgram(
		defaults,
		input,
		history,
		providerTools,
		prepared...,
	)
}

func newSingleModelProgram(
	defaults SessionDefaults,
	input ActivityID,
	isolated bool,
	history []ActivityID,
) (ProgramSnapshot, error) {
	if input == "" {
		return ProgramSnapshot{}, errors.New(errSingleModelInputRequired)
	}
	return programSnapshot(
		BuiltinSingleModelInterpreterID,
		singleModelProgramPayload{
			Defaults:           defaults,
			InputActivityID:    input,
			IsolatedInput:      isolated,
			HistoryActivityIDs: append([]ActivityID(nil), history...),
		},
	)
}

func newSingleModelWorkflowInvocationProgram(
	defaults SessionDefaults,
	input ActivityID,
	history []ActivityID,
	providerTools []ProviderManagedToolChoice,
	prepared ...*WorkflowPreparedInvocation,
) (ProgramSnapshot, error) {
	if input == "" {
		return ProgramSnapshot{}, errors.New(errSingleModelInputRequired)
	}
	return programSnapshot(BuiltinSingleModelInterpreterID, singleModelProgramPayload{
		Defaults:           defaults,
		InputActivityID:    input,
		IsolatedInput:      true,
		HistoryActivityIDs: append([]ActivityID(nil), history...),
		ProviderTools:      jsonutil.CloneJSONInValue(providerTools),
		PreparedInvocation: optionalWorkflowPreparedInvocation(prepared),
	})
}

func programSnapshot(id string, payload any) (ProgramSnapshot, error) {
	return ProgramSnapshot{
		InterpreterID:      id,
		InterpreterVersion: BuiltinInterpreterVersion,
		Payload:            jsonutil.MustCanonicalValue(payload),
	}, nil
}

func optionalWorkflowPreparedInvocation(
	values []*WorkflowPreparedInvocation,
) *WorkflowPreparedInvocation {
	if len(values) == 0 || values[0] == nil {
		return nil
	}

	value := jsonutil.CloneJSONInValue(*values[0])
	return &value
}

func ProgramInterpreterKey(id string, version uint32) string {
	return strings.TrimSpace(id) + "@" + strconv.FormatUint(uint64(version), 10)
}

type conversationInterpreter struct{}

func (conversationInterpreter) ID() string      { return BuiltinConversationInterpreterID }
func (conversationInterpreter) Version() uint32 { return BuiltinInterpreterVersion }
func (conversationInterpreter) ValidateSnapshot(snapshot ProgramSnapshot) error {
	if snapshot.InterpreterID != BuiltinConversationInterpreterID ||
		snapshot.InterpreterVersion != BuiltinInterpreterVersion {
		return errors.New(errUnsupportedConvProgram)
	}
	var value conversationProgramPayload
	return json.Unmarshal(snapshot.Payload, &value)
}

func (conversationInterpreter) ValidateState(state ProgramState) error {
	if state.Type != BuiltinConversationInterpreterID || state.Version != BuiltinInterpreterVersion {
		return errors.New(errUnsupportedConvState)
	}
	var value conversationProgramState
	return json.Unmarshal(state.Data, &value)
}

func (conversationInterpreter) InitialState(ProgramSnapshot, time.Time) (ProgramState, error) {
	return encodeConversationState(conversationProgramState{}), nil
}

func (conversationInterpreter) Advance(frame ProgramFrame, _ Execution, event ProgramEvent) (ProgramTransition, error) {
	var payload conversationProgramPayload
	if err := json.Unmarshal(frame.Program.Payload, &payload); err != nil {
		return ProgramTransition{}, err
	}
	var state conversationProgramState
	if err := json.Unmarshal(frame.State.Data, &state); err != nil {
		return ProgramTransition{}, err
	}
	switch event.Kind {
	case ProgramEventActivity:
		defaults := cloneInvocationDefaults(event.InvocationDefaults)
		if event.ActivityID != "" {
			state.QueuedActivities = append(
				state.QueuedActivities,
				conversationQueuedActivity{
					ID:                 event.ActivityID,
					InvocationDefaults: defaults,
					InvocationBudget:   cloneBudget(event.InvocationBudget),
				},
			)
		}
	case ProgramEventChildSettled:
		if event.ChildFrameID == state.ActiveFrameID {
			state.ActiveFrameID = ""
		}
	case ProgramEventStarted, ProgramEventWorkSettled, ProgramEventContinue:
		// No state change; continue to queue handling below.
	case ProgramEventCancelled:
		status := ProgramFrameStateCancelled
		return ProgramTransition{State: encodeConversationState(state), Status: &status}, nil
	default:
	}

	// A durable Fork records child intent before child materialization. More
	// user activities can arrive during that boundary, but the conversation
	// must queue them until the current fork has produced its active child.
	if frame.Fork != nil {
		return ProgramTransition{State: encodeConversationState(state)}, nil
	}
	if len(state.QueuedActivities) == 0 &&
		len(state.QueuedActivityIDs) != 0 {
		for _, id := range state.QueuedActivityIDs {
			state.QueuedActivities = append(
				state.QueuedActivities,
				conversationQueuedActivity{ID: id},
			)
		}
		state.QueuedActivityIDs = nil
	}
	if state.ActiveFrameID != "" || len(state.QueuedActivities) == 0 {
		return ProgramTransition{State: encodeConversationState(state)}, nil
	}
	queued := state.QueuedActivities[0]
	state.QueuedActivities = state.QueuedActivities[1:]

	defaults := cloneSessionDefaults(payload.Defaults)
	if queued.InvocationDefaults != nil {
		defaults = cloneSessionDefaults(*queued.InvocationDefaults)
	}
	program, err := NewAgentLoopProgram(defaults, queued.ID)
	if err != nil {
		return ProgramTransition{}, err
	}
	return ProgramTransition{
		State: encodeConversationState(state),
		Frames: []ProgramFrameIntent{{
			Path:    conversationFramePathPrefix + string(queued.ID),
			Program: program,
			Budget:  cloneBudget(queued.InvocationBudget),
		}},
	}, nil
}

func encodeConversationState(value conversationProgramState) ProgramState {
	return ProgramState{
		Type:    BuiltinConversationInterpreterID,
		Version: BuiltinInterpreterVersion,
		Data:    jsonutil.MustCanonicalValue(value),
	}
}

type agentLoopInterpreter struct{}

func (agentLoopInterpreter) ID() string      { return BuiltinAgentLoopInterpreterID }
func (agentLoopInterpreter) Version() uint32 { return BuiltinInterpreterVersion }
func (agentLoopInterpreter) ValidateSnapshot(snapshot ProgramSnapshot) error {
	if snapshot.InterpreterID != BuiltinAgentLoopInterpreterID ||
		snapshot.InterpreterVersion != BuiltinInterpreterVersion {
		return errors.New(errUnsupportedAgentProgram)
	}
	var value agentLoopProgramPayload
	if err := json.Unmarshal(snapshot.Payload, &value); err != nil {
		return err
	}
	if value.InputActivityID == "" {
		return errors.New(errAgentLoopInputRequired)
	}
	return nil
}

func (agentLoopInterpreter) ValidateState(state ProgramState) error {
	if state.Type != BuiltinAgentLoopInterpreterID || state.Version != BuiltinInterpreterVersion {
		return errors.New(errUnsupportedAgentState)
	}
	var value agentLoopProgramState
	if err := json.Unmarshal(state.Data, &value); err != nil {
		return err
	}
	if value.InputActivityID == "" {
		return nil
	}
	return nil
}

func (agentLoopInterpreter) InitialState(snapshot ProgramSnapshot, _ time.Time) (ProgramState, error) {
	var payload agentLoopProgramPayload
	if err := json.Unmarshal(snapshot.Payload, &payload); err != nil {
		return ProgramState{}, err
	}
	return encodeAgentLoopState(agentLoopProgramState{InputActivityID: payload.InputActivityID}), nil
}

func (agentLoopInterpreter) Advance(
	frame ProgramFrame,
	execution Execution,
	event ProgramEvent,
) (ProgramTransition, error) {
	var payload agentLoopProgramPayload
	if err := json.Unmarshal(frame.Program.Payload, &payload); err != nil {
		return ProgramTransition{}, err
	}
	var state agentLoopProgramState
	if err := json.Unmarshal(frame.State.Data, &state); err != nil {
		return ProgramTransition{}, err
	}
	if len(state.QueuedActivities) == 0 &&
		len(state.QueuedActivityIDs) != 0 {
		for _, activityID := range state.QueuedActivityIDs {
			state.QueuedActivities = append(
				state.QueuedActivities,
				conversationQueuedActivity{ID: activityID},
			)
		}
		state.QueuedActivityIDs = nil
	}

	switch event.Kind {
	case ProgramEventActivity:
		if event.ActivityID == "" {
			return ProgramTransition{State: encodeAgentLoopState(state)}, nil
		}
		state.QueuedActivities = append(
			state.QueuedActivities,
			conversationQueuedActivity{
				ID:                 event.ActivityID,
				InvocationDefaults: cloneInvocationDefaults(event.InvocationDefaults),
				InvocationBudget:   cloneBudget(event.InvocationBudget),
			},
		)
		if !agentLoopCanStartQueuedActivity(
			frame,
			execution,
			state,
		) {
			status := ProgramFrameStateWaiting
			return ProgramTransition{
				State:  encodeAgentLoopState(state),
				Status: &status,
			}, nil
		}
		return startQueuedAgentLLM(state, payload)

	case ProgramEventStarted:
		if state.LastLLMWorkItemID == "" {
			return startAgentLLM(state, payload)
		}
	case ProgramEventWorkSettled:
		item := executionWorkItem(execution, event.WorkItemID)
		if item == nil {
			return ProgramTransition{State: encodeAgentLoopState(state)}, nil
		}
		if item.FrameID != frame.ID {
			return ProgramTransition{State: encodeAgentLoopState(state)}, nil
		}
		if item.Type == WorkItemTypeLLM {
			state.LastLLMWorkItemID = item.ID
			if item.State != WorkItemStateCompleted {
				transition, transitionErr := failedAgentWorkTransition(
					state,
					*item,
				)
				transition.Activities = interruptedAssistantActivitiesForAttempt(
					*item,
					latestAttemptForWorkItem(execution, item.ID),
				)
				return transition, transitionErr
			}
			if item.Output == nil || item.Output.LLM == nil {
				return failedAgentTransition(state, llmOutputMissingCode, llmOutputMissingMessage)
			}
			assistant := Activity{
				Kind:       ActivityKindMessage,
				Visibility: ActivityVisibilityBoth,
				WorkItemID: item.ID,
				Message:    completedLLMAssistantMessage(*item),
			}
			executable, observed := splitExecutableCalls(item.Output.LLM.ToolCalls)
			activities := make([]Activity, 0, 1+len(observed))
			activities = append(activities, assistant)
			activities = append(activities, observed...)
			if len(executable) == 0 {
				if frame.HandoffID != "" {
					if len(state.QueuedActivities) != 0 {
						transition, transitionErr := startQueuedAgentLLM(
							state,
							payload,
						)
						transition.Activities = activities
						return transition, transitionErr
					}
					status := ProgramFrameStateWaiting
					return ProgramTransition{
						State:      encodeAgentLoopState(state),
						Status:     &status,
						Output:     jsonutil.MustCanonicalValue(item.Output.LLM),
						Activities: activities,
					}, nil
				}
				status := ProgramFrameStateCompleted
				return ProgramTransition{
					State:      encodeAgentLoopState(state),
					Status:     &status,
					Output:     jsonutil.MustCanonicalValue(item.Output.LLM),
					Activities: activities,
				}, nil
			}
			state.PendingToolIDs = nil
			state.AwaitingContinue = true
			work := make([]WorkItemIntent, 0, len(executable))
			origin, originErr := LLMToolOriginForWorkItem(*item)
			if originErr != nil {
				return failedAgentTransition(
					state,
					llmToolOriginMissingCode,
					originErr.Error(),
				)
			}
			var previousSkillWork WorkItemID
			for callIndex, call := range executable {
				intent := WorkItemIntent{
					ProgramPath:      agentLoopToolPathPrefix + call.ID,
					Type:             WorkItemTypeTool,
					OriginWorkItemID: item.ID,
					Automatic:        call.Automatic,
					Effect:           EffectContract{Kind: EffectKindNonIdempotent},
					Input: WorkItemPayload{Tool: &ToolWork{
						Call:   call,
						Origin: cloneLLMToolOrigin(origin),
					}},
				}
				if call.Binding.Kind == ToolBindingKindSkill {
					callIdentity := call.ID
					if callIdentity == "" {
						callIdentity = strconv.Itoa(callIndex)
					}
					intent.ID = WorkItemID(
						agentLoopSkillWorkPrefix +
							string(item.ID) +
							":" +
							callIdentity,
					)
					if previousSkillWork != "" {
						intent.Dependencies = []WorkItemID{
							previousSkillWork,
						}
					}
					previousSkillWork = intent.ID
				}
				work = append(work, intent)
			}
			status := ProgramFrameStateWaiting
			return ProgramTransition{
				State:      encodeAgentLoopState(state),
				Status:     &status,
				Activities: activities,
				WorkItems:  work,
			}, nil
		}
		if containsWorkItemID(state.PendingToolIDs, event.WorkItemID) &&
			allToolItemsSettled(execution, state.PendingToolIDs) {
			automaticContinuation := allAutomaticToolItemsSucceeded(
				execution,
				state.PendingToolIDs,
			) && !automationHoldApplies(
				execution.Controls.AutomaticContinuation,
				state.LastLLMWorkItemID,
			)
			if state.ContinueRequested || automaticContinuation {
				return startAgentLLM(state, payload)
			}
			if frame.HandoffID != "" && len(state.QueuedActivities) != 0 {
				return startQueuedAgentLLM(state, payload)
			}
			status := ProgramFrameStateWaiting
			return ProgramTransition{State: encodeAgentLoopState(state), Status: &status}, nil
		}
	case ProgramEventChildSettled:
		// Agent loop observes only work settlement and continue; no action.
	case ProgramEventContinue:
		state.ContinueRequested = true
		if state.AwaitingContinue && allToolItemsSettled(execution, state.PendingToolIDs) {
			return startAgentLLM(state, payload)
		}
		if frame.HandoffID != "" &&
			agentLoopCanStartQueuedActivity(frame, execution, state) {
			return startQueuedAgentLLM(state, payload)
		}
	case ProgramEventHandoffComplete:
		if frame.HandoffID == "" ||
			!agentLoopCanCompleteHandoff(frame, execution, state) {
			return ProgramTransition{State: encodeAgentLoopState(state)}, nil
		}
		output := jsonutil.MustCanonicalValue(nil)
		if item := executionWorkItem(execution, state.LastLLMWorkItemID); item != nil &&
			item.Output != nil && item.Output.LLM != nil {
			output = jsonutil.MustCanonicalValue(item.Output.LLM)
		}
		status := ProgramFrameStateCompleted
		return ProgramTransition{
			State:  encodeAgentLoopState(state),
			Status: &status,
			Output: output,
		}, nil
	case ProgramEventCancelled:
		status := ProgramFrameStateCancelled
		return ProgramTransition{State: encodeAgentLoopState(state), Status: &status}, nil
	}
	return ProgramTransition{State: encodeAgentLoopState(state)}, nil
}

func agentLoopCanStartQueuedActivity(
	frame ProgramFrame,
	execution Execution,
	state agentLoopProgramState,
) bool {
	if len(state.QueuedActivities) == 0 {
		return false
	}

	for _, item := range execution.WorkItems {
		if item.FrameID == frame.ID && !item.State.Terminal() {
			return false
		}
	}
	if len(state.PendingToolIDs) != 0 &&
		!allToolItemsSettled(execution, state.PendingToolIDs) {
		return false
	}
	if state.LastLLMWorkItemID == "" {
		return true
	}
	last := executionWorkItem(execution, state.LastLLMWorkItemID)
	return last != nil && last.State.Terminal()
}

func agentLoopCanCompleteHandoff(
	frame ProgramFrame,
	execution Execution,
	state agentLoopProgramState,
) bool {
	if frame.HandoffID == "" ||
		len(state.QueuedActivities) != 0 ||
		state.AwaitingContinue {
		return false
	}
	for _, item := range execution.WorkItems {
		if item.FrameID == frame.ID && !item.State.Terminal() {
			return false
		}
	}
	return true
}

func startQueuedAgentLLM(
	state agentLoopProgramState,
	payload agentLoopProgramPayload,
) (ProgramTransition, error) {
	if len(state.QueuedActivities) == 0 {
		return ProgramTransition{
			State: encodeAgentLoopState(state),
		}, nil
	}
	if state.InputActivityID != "" &&
		!slices.Contains(state.CompletedInputActivityIDs, state.InputActivityID) {
		state.CompletedInputActivityIDs = append(
			state.CompletedInputActivityIDs,
			state.InputActivityID,
		)
	}
	queued := state.QueuedActivities[0]
	state.QueuedActivities = state.QueuedActivities[1:]
	state.InputActivityID = queued.ID
	state.InvocationDefaults = cloneInvocationDefaults(
		queued.InvocationDefaults,
	)
	state.InvocationBudget = cloneBudget(queued.InvocationBudget)
	state.LastLLMWorkItemID = ""
	state.Round = 0
	state.AwaitingContinue = false
	state.ContinueRequested = false
	return startAgentLLM(state, payload)
}

func startAgentLLM(
	state agentLoopProgramState,
	payload agentLoopProgramPayload,
) (ProgramTransition, error) {
	continuations := append([]WorkItemID(nil), state.PendingToolIDs...)
	state.Round++
	state.PendingToolIDs = nil
	state.AwaitingContinue = false
	state.ContinueRequested = false
	status := ProgramFrameStateRunning
	defaults := payload.Defaults
	if state.InvocationDefaults != nil {
		defaults = cloneSessionDefaults(*state.InvocationDefaults)
	}
	return ProgramTransition{
		State:  encodeAgentLoopState(state),
		Status: &status,
		WorkItems: []WorkItemIntent{
			{
				ProgramPath: fmt.Sprintf(agentLoopLLMPathFormat, state.Round),
				Type:        WorkItemTypeLLM,
				Automatic:   true,
				Budget:      cloneBudget(state.InvocationBudget),
				Effect:      EffectContract{Kind: EffectKindRead},
				Input: WorkItemPayload{
					LLM: new(agentLLMWork(
						state,
						agentLoopProgramPayload{
							Defaults:           defaults,
							InputActivityID:    payload.InputActivityID,
							IsolatedInput:      payload.IsolatedInput,
							HistoryActivityIDs: payload.HistoryActivityIDs,
							ProviderTools:      payload.ProviderTools,
							PreparedInvocation: jsonutil.CloneJSONInValue(payload.PreparedInvocation),
						},
						continuations,
					)),
				},
			},
		},
	}, nil
}

func agentLLMWork(
	state agentLoopProgramState,
	payload agentLoopProgramPayload,
	continuations []WorkItemID,
) LLMWork {
	defaults := payload.Defaults
	work := LLMWork{
		Model:           cloneSelection(defaults.Model),
		Agent:           cloneSelection(defaults.Agent),
		InputActivityID: state.InputActivityID,
		ContinuationWorkItemIDs: append(
			[]WorkItemID(nil),
			continuations...,
		),
		SystemInstructions: append([]Instruction(nil), defaults.SystemInstructions...),
		Temperature:        cloneFloat(defaults.Temperature),
	}
	applyLLMInvocationDefaults(
		&work,
		defaults,
	)
	if state.InvocationDefaults != nil {
		work.InvocationDefaults = cloneInvocationDefaults(state.InvocationDefaults)
	}
	history := append([]ActivityID(nil), payload.HistoryActivityIDs...)
	for _, activityID := range state.CompletedInputActivityIDs {
		if activityID != "" && !slices.Contains(history, activityID) {
			history = append(history, activityID)
		}
	}
	applyIsolatedLLMInput(
		&work,
		payload.IsolatedInput,
		history,
	)
	work.PreparedInvocation = jsonutil.CloneJSONInValue(payload.PreparedInvocation)
	work.ProviderTools = jsonutil.CloneJSONInValue(payload.ProviderTools)
	return work
}

func AgentLoopCompletedInputActivityIDs(frame ProgramFrame) []ActivityID {
	if frame.Program.InterpreterID != BuiltinAgentLoopInterpreterID ||
		len(frame.State.Data) == 0 {
		return nil
	}
	var state agentLoopProgramState
	if err := json.Unmarshal(frame.State.Data, &state); err != nil {
		return nil
	}
	return append([]ActivityID(nil), state.CompletedInputActivityIDs...)
}

func applyLLMInvocationDefaults(
	work *LLMWork,
	defaults SessionDefaults,
) {
	if work == nil {
		return
	}

	invocationDefaults := defaults
	work.InvocationDefaults = &invocationDefaults
}

func applyIsolatedLLMInput(
	work *LLMWork,
	isolate bool,
	history []ActivityID,
) {
	if work == nil || !isolate {
		return
	}

	work.IsolatedInput = true
	work.HistoryActivityIDs = append(
		[]ActivityID(nil),
		history...,
	)
}

func failedAgentTransition(state agentLoopProgramState, code, message string) (ProgramTransition, error) {
	status := ProgramFrameStateFailed
	return ProgramTransition{
		State:  encodeAgentLoopState(state),
		Status: &status,
		Error:  &SanitizedError{Code: code, Message: message},
	}, nil
}

func failedAgentWorkTransition(
	state agentLoopProgramState,
	item WorkItem,
) (ProgramTransition, error) {
	status := ProgramFrameStateFailed
	if item.State == WorkItemStateCancelled {
		status = ProgramFrameStateCancelled
	}
	failure := cloneWorkflowSanitizedError(item.Error)
	if failure == nil {
		failure = &SanitizedError{
			Code:     llmWorkFailedCode,
			Category: ErrorCategoryExecution,
			Message:  workItemFailureMessage(item),
		}
	}
	return ProgramTransition{
		State:  encodeAgentLoopState(state),
		Status: &status,
		Error:  failure,
	}, nil
}

func workItemFailureMessage(item WorkItem) string {
	if item.Error != nil && strings.TrimSpace(item.Error.Message) != "" {
		return item.Error.Message
	}
	if item.State == WorkItemStateCancelled {
		return llmWorkCancelledMessage
	}
	return llmWorkFailedMessage
}

func completedLLMAssistantMessage(item WorkItem) *MessageActivity {
	message := &MessageActivity{
		Role: MessageRoleAssistant,
	}
	if item.Output != nil && item.Output.LLM != nil {
		message.Text = item.Output.LLM.Text
	}
	if item.Error != nil {
		message.Status = MessageStatusFailed
		message.Error = cloneWorkflowSanitizedError(item.Error)
	}
	return message
}

// interruptedAssistantActivities retains observed assistant text after a
// failed or cancelled LLM WorkItem. The Activity reports its actual terminal
// meaning and does not fabricate a successful Model result.
func interruptedAssistantActivities(item WorkItem) []Activity {
	text := ""
	if item.Output != nil && item.Output.LLM != nil {
		text = item.Output.LLM.Text
	}
	if strings.TrimSpace(text) == "" && item.LLMStream != nil {
		text = item.LLMStream.Text
	}
	if strings.TrimSpace(text) == "" {
		return nil
	}

	status := MessageStatusFailed
	if item.State == WorkItemStateCancelled {
		status = MessageStatusCancelled
	}
	if item.State == WorkItemStateUncertain {
		status = MessageStatusUncertain
	}
	return []Activity{{
		Kind:       ActivityKindMessage,
		Visibility: ActivityVisibilityBoth,
		WorkItemID: item.ID,
		Message: &MessageActivity{
			Role:   MessageRoleAssistant,
			Status: status,
			Text:   text,
			Error:  cloneWorkflowSanitizedError(item.Error),
		},
	}}
}

func interruptedAssistantActivitiesForAttempt(
	item WorkItem,
	attempt *Attempt,
) []Activity {
	values := interruptedAssistantActivities(item)
	if len(values) == 0 || attempt == nil {
		return values
	}
	for index := range values {
		values[index].AttemptID = attempt.ID
	}
	return values
}

func (r *Runtime) appendInterruptedAssistantActivityLocked(
	session *Session,
	execution *Execution,
	item WorkItem,
	attempt *Attempt,
	now time.Time,
) {
	if session == nil || execution == nil {
		return
	}
	for _, activity := range session.Timeline {
		if activity.Kind != ActivityKindMessage ||
			activity.Message == nil ||
			activity.Message.Role != MessageRoleAssistant ||
			activity.WorkItemID != item.ID {
			continue
		}
		if attempt == nil || activity.AttemptID == attempt.ID {
			return
		}
	}
	for _, activity := range interruptedAssistantActivitiesForAttempt(item, attempt) {
		activity.ID = ActivityID(r.newID())
		activity.CreatedAt = now.UTC()
		activity.ExecutionID = execution.ID
		activity.FrameID = item.FrameID
		session.Timeline = append(session.Timeline, activity)
	}
}

func encodeAgentLoopState(value agentLoopProgramState) ProgramState {
	return ProgramState{
		Type:    BuiltinAgentLoopInterpreterID,
		Version: BuiltinInterpreterVersion,
		Data:    jsonutil.MustCanonicalValue(value),
	}
}

func splitExecutableCalls(calls []ToolCall) ([]ToolCall, []Activity) {
	executable := make([]ToolCall, 0, len(calls))
	observed := make([]Activity, 0)
	for _, call := range calls {
		if call.Binding.Kind == ToolBindingKindProvider {
			observed = append(observed, Activity{
				Kind:       ActivityKindAction,
				Visibility: ActivityVisibilityBoth,
				WorkItemID: WorkItemID(call.ID),
				Action: &ActionActivity{
					Title:      firstNonEmpty(call.DisplayName, call.Name),
					Summary:    providerManagedToolSummary,
					ActionKind: ActionKindTool,
					Outcome:    string(ToolOutputKindSuccess),
				},
			})
			continue
		}
		executable = append(executable, call)
	}
	return executable, observed
}

type singleModelInterpreter struct{}

func (singleModelInterpreter) ID() string      { return BuiltinSingleModelInterpreterID }
func (singleModelInterpreter) Version() uint32 { return BuiltinInterpreterVersion }
func (singleModelInterpreter) ValidateSnapshot(snapshot ProgramSnapshot) error {
	if snapshot.InterpreterID != BuiltinSingleModelInterpreterID ||
		snapshot.InterpreterVersion != BuiltinInterpreterVersion {
		return errors.New(errUnsupportedModelProgram)
	}
	var value singleModelProgramPayload
	if err := json.Unmarshal(snapshot.Payload, &value); err != nil {
		return err
	}
	if value.InputActivityID == "" {
		return errors.New(errSingleModelInputRequired)
	}
	return nil
}

func (singleModelInterpreter) ValidateState(state ProgramState) error {
	if state.Type != BuiltinSingleModelInterpreterID || state.Version != BuiltinInterpreterVersion {
		return errors.New(errUnsupportedModelState)
	}
	var value singleModelProgramState
	return json.Unmarshal(state.Data, &value)
}

func (singleModelInterpreter) InitialState(snapshot ProgramSnapshot, _ time.Time) (ProgramState, error) {
	var payload singleModelProgramPayload
	if err := json.Unmarshal(snapshot.Payload, &payload); err != nil {
		return ProgramState{}, err
	}
	return encodeSingleModelState(singleModelProgramState{InputActivityID: payload.InputActivityID}), nil
}

func (singleModelInterpreter) Advance(
	frame ProgramFrame,
	execution Execution,
	event ProgramEvent,
) (ProgramTransition, error) {
	var payload singleModelProgramPayload
	if err := json.Unmarshal(frame.Program.Payload, &payload); err != nil {
		return ProgramTransition{}, err
	}
	var state singleModelProgramState
	if err := json.Unmarshal(frame.State.Data, &state); err != nil {
		return ProgramTransition{}, err
	}
	switch event.Kind {
	case ProgramEventStarted:
		if state.WorkItemID == "" {
			work := LLMWork{
				Model:              cloneSelection(payload.Defaults.Model),
				InputActivityID:    payload.InputActivityID,
				SystemInstructions: append([]Instruction(nil), payload.Defaults.SystemInstructions...),
				Temperature:        cloneFloat(payload.Defaults.Temperature),
			}
			applyLLMInvocationDefaults(
				&work,
				payload.Defaults,
			)
			applyIsolatedLLMInput(
				&work,
				payload.IsolatedInput,
				payload.HistoryActivityIDs,
			)
			work.PreparedInvocation = jsonutil.CloneJSONInValue(payload.PreparedInvocation)
			work.ProviderTools = jsonutil.CloneJSONInValue(payload.ProviderTools)
			status := ProgramFrameStateRunning
			return ProgramTransition{
				State:  encodeSingleModelState(state),
				Status: &status,
				WorkItems: []WorkItemIntent{
					{
						ProgramPath: singleModelWorkPath,
						Type:        WorkItemTypeLLM,
						Automatic:   true,
						Effect:      EffectContract{Kind: EffectKindRead},
						Input: WorkItemPayload{
							LLM: &work,
						},
					},
				},
			}, nil
		}
	case ProgramEventWorkSettled:
		item := executionWorkItem(execution, event.WorkItemID)
		if item == nil ||
			item.FrameID != frame.ID ||
			item.ID != state.WorkItemID {
			return ProgramTransition{State: encodeSingleModelState(state)}, nil
		}
		if item.State != WorkItemStateCompleted {
			transition, transitionErr := failedAgentWorkTransition(
				agentLoopProgramState{},
				*item,
			)
			transition.State = encodeSingleModelState(state)
			transition.Activities = interruptedAssistantActivitiesForAttempt(
				*item,
				latestAttemptForWorkItem(execution, item.ID),
			)
			return transition, transitionErr
		}
		if item.Output == nil || item.Output.LLM == nil {
			status := ProgramFrameStateFailed
			return ProgramTransition{
				State:  encodeSingleModelState(state),
				Status: &status,
				Error: &SanitizedError{
					Code:    modelOutputMissingCode,
					Message: modelOutputMissingMessage,
				},
			}, nil
		}
		status := ProgramFrameStateCompleted
		return ProgramTransition{
			State:  encodeSingleModelState(state),
			Status: &status,
			Output: jsonutil.MustCanonicalValue(item.Output.LLM),
			Activities: []Activity{
				{
					Kind:       ActivityKindMessage,
					Visibility: ActivityVisibilityBoth,
					WorkItemID: item.ID,
					Message:    completedLLMAssistantMessage(*item),
				},
			},
		}, nil
	case ProgramEventActivity, ProgramEventChildSettled, ProgramEventContinue:
		// Single-model observes only start, work settlement, and cancellation.
	case ProgramEventCancelled:
		status := ProgramFrameStateCancelled
		return ProgramTransition{State: encodeSingleModelState(state), Status: &status}, nil
	default:
	}
	return ProgramTransition{State: encodeSingleModelState(state)}, nil
}

func encodeSingleModelState(value singleModelProgramState) ProgramState {
	return ProgramState{
		Type:    BuiltinSingleModelInterpreterID,
		Version: BuiltinInterpreterVersion,
		Data:    jsonutil.MustCanonicalValue(value),
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func executionWorkItem(execution Execution, id WorkItemID) *WorkItem {
	for index := range execution.WorkItems {
		if execution.WorkItems[index].ID == id {
			value := execution.WorkItems[index]
			return &value
		}
	}
	return nil
}

func containsWorkItemID(values []WorkItemID, want WorkItemID) bool {
	return slices.Contains(values, want)
}

func allToolItemsSettled(execution Execution, ids []WorkItemID) bool {
	if len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		item := executionWorkItem(execution, id)
		if item == nil || !item.State.Terminal() {
			return false
		}
	}
	return true
}

func allAutomaticToolItemsSucceeded(execution Execution, ids []WorkItemID) bool {
	if len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		item := executionWorkItem(execution, id)
		if item == nil || !item.Automatic || item.State != WorkItemStateCompleted || item.Output == nil ||
			item.Output.Tool == nil ||
			item.Output.Tool.Kind != ToolOutputKindSuccess {
			return false
		}
	}
	return true
}

func cloneSelection(value *ArtifactSelection) *ArtifactSelection {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cloneFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
