package sessionv3

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

const (
	BuiltinConversationInterpreterID = "builtin.conversation"
	BuiltinAgentLoopInterpreterID    = "builtin.agent-loop"
	BuiltinSingleModelInterpreterID  = "builtin.single-model"
	BuiltinInterpreterVersion        = uint32(1)
)

type (
	conversationProgramPayload struct {
		Defaults SessionDefaults `json:"defaults"`
	}
	conversationProgramState struct {
		QueuedActivityIDs []ActivityID   `json:"queuedActivityIDs,omitempty"`
		ActiveFrameID     ProgramFrameID `json:"activeFrameID,omitempty"`
	}
)

type agentLoopProgramPayload struct {
	Defaults        SessionDefaults `json:"defaults"`
	InputActivityID ActivityID      `json:"inputActivityID"`
}
type agentLoopProgramState struct {
	InputActivityID   ActivityID   `json:"inputActivityID"`
	LastLLMWorkItemID WorkItemID   `json:"lastLLMWorkItemID,omitempty"`
	PendingToolIDs    []WorkItemID `json:"pendingToolIDs,omitempty"`
	Round             int          `json:"round"`
	AwaitingContinue  bool         `json:"awaitingContinue"`
}

type singleModelProgramPayload struct {
	Defaults        SessionDefaults `json:"defaults"`
	InputActivityID ActivityID      `json:"inputActivityID"`
}
type singleModelProgramState struct {
	InputActivityID ActivityID `json:"inputActivityID"`
	WorkItemID      WorkItemID `json:"workItemID,omitempty"`
}

func NewBuiltinInterpreters() []ProgramInterpreter {
	return []ProgramInterpreter{conversationInterpreter{}, agentLoopInterpreter{}, singleModelInterpreter{}}
}

func NewConversationProgram(defaults SessionDefaults) (ProgramSnapshot, error) {
	return programSnapshot(BuiltinConversationInterpreterID, conversationProgramPayload{Defaults: defaults})
}

func NewAgentLoopProgram(defaults SessionDefaults, input ActivityID) (ProgramSnapshot, error) {
	if input == "" {
		return ProgramSnapshot{}, errors.New("agent-loop input Activity is required")
	}
	return programSnapshot(
		BuiltinAgentLoopInterpreterID,
		agentLoopProgramPayload{Defaults: defaults, InputActivityID: input},
	)
}

func NewSingleModelProgram(defaults SessionDefaults, input ActivityID) (ProgramSnapshot, error) {
	if input == "" {
		return ProgramSnapshot{}, errors.New("single-model input Activity is required")
	}
	return programSnapshot(
		BuiltinSingleModelInterpreterID,
		singleModelProgramPayload{Defaults: defaults, InputActivityID: input},
	)
}

func programSnapshot(id string, payload any) (ProgramSnapshot, error) {
	return ProgramSnapshot{
		InterpreterID:      id,
		InterpreterVersion: BuiltinInterpreterVersion,
		Payload:            MustCanonicalValue(payload),
	}, nil
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
		return errors.New("unsupported conversation program version")
	}
	var value conversationProgramPayload
	return json.Unmarshal(snapshot.Payload, &value)
}

func (conversationInterpreter) ValidateState(state ProgramState) error {
	if state.Type != BuiltinConversationInterpreterID || state.Version != BuiltinInterpreterVersion {
		return errors.New("unsupported conversation state version")
	}
	var value conversationProgramState
	return json.Unmarshal(state.Data, &value)
}

func (conversationInterpreter) InitialState(ProgramSnapshot) (ProgramState, error) {
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
		if event.ActivityID != "" {
			state.QueuedActivityIDs = append(state.QueuedActivityIDs, event.ActivityID)
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
	}
	if state.ActiveFrameID != "" || len(state.QueuedActivityIDs) == 0 {
		return ProgramTransition{State: encodeConversationState(state)}, nil
	}
	input := state.QueuedActivityIDs[0]
	state.QueuedActivityIDs = state.QueuedActivityIDs[1:]
	program, err := NewAgentLoopProgram(payload.Defaults, input)
	if err != nil {
		return ProgramTransition{}, err
	}
	return ProgramTransition{
		State:  encodeConversationState(state),
		Frames: []ProgramFrameIntent{{Path: "conversation/" + string(input), Program: program}},
	}, nil
}

func encodeConversationState(value conversationProgramState) ProgramState {
	return ProgramState{
		Type:    BuiltinConversationInterpreterID,
		Version: BuiltinInterpreterVersion,
		Data:    MustCanonicalValue(value),
	}
}

type agentLoopInterpreter struct{}

func (agentLoopInterpreter) ID() string      { return BuiltinAgentLoopInterpreterID }
func (agentLoopInterpreter) Version() uint32 { return BuiltinInterpreterVersion }
func (agentLoopInterpreter) ValidateSnapshot(snapshot ProgramSnapshot) error {
	if snapshot.InterpreterID != BuiltinAgentLoopInterpreterID ||
		snapshot.InterpreterVersion != BuiltinInterpreterVersion {
		return errors.New("unsupported Agent-loop program version")
	}
	var value agentLoopProgramPayload
	if err := json.Unmarshal(snapshot.Payload, &value); err != nil {
		return err
	}
	if value.InputActivityID == "" {
		return errors.New("agent-loop input Activity is required")
	}
	return nil
}

func (agentLoopInterpreter) ValidateState(state ProgramState) error {
	if state.Type != BuiltinAgentLoopInterpreterID || state.Version != BuiltinInterpreterVersion {
		return errors.New("unsupported Agent-loop state version")
	}
	var value agentLoopProgramState
	if err := json.Unmarshal(state.Data, &value); err != nil {
		return err
	}
	if value.InputActivityID == "" {
		return errors.New("agent-loop state input Activity is required")
	}
	return nil
}

func (agentLoopInterpreter) InitialState(snapshot ProgramSnapshot) (ProgramState, error) {
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
	switch event.Kind {
	case ProgramEventStarted:
		if state.LastLLMWorkItemID == "" {
			return startAgentLLM(state, payload.Defaults)
		}
	case ProgramEventWorkSettled:
		item := executionWorkItem(execution, event.WorkItemID)
		if item == nil {
			return ProgramTransition{State: encodeAgentLoopState(state)}, nil
		}
		if item.Type == WorkItemTypeLLM && item.FrameID == frame.ID {
			state.LastLLMWorkItemID = item.ID
			if item.Output == nil || item.Output.LLM == nil {
				return failedAgentTransition(state, "llm_output_missing", "LLM WorkItem completed without LLM output")
			}
			assistant := Activity{
				Kind:       ActivityKindMessage,
				Visibility: ActivityVisibilityBoth,
				WorkItemID: item.ID,
				Message:    &MessageActivity{Role: MessageRoleAssistant, Text: item.Output.LLM.Text},
			}
			executable, observed := splitExecutableCalls(item.Output.LLM.ToolCalls)
			activities := make([]Activity, 0, 1+len(observed))
			activities = append(activities, assistant)
			activities = append(activities, observed...)
			if len(executable) == 0 {
				status := ProgramFrameStateCompleted
				return ProgramTransition{
					State:      encodeAgentLoopState(state),
					Status:     &status,
					Output:     MustCanonicalValue(item.Output.LLM),
					Activities: activities,
				}, nil
			}
			state.PendingToolIDs = nil
			state.AwaitingContinue = true
			work := make([]WorkItemIntent, 0, len(executable))
			for _, call := range executable {
				work = append(
					work,
					WorkItemIntent{
						ProgramPath: "tools/" + call.ID,
						Type:        WorkItemTypeTool,
						Automatic:   call.Automatic,
						Effect:      EffectContract{Kind: EffectKindNonIdempotent},
						Input:       WorkItemPayload{Tool: &ToolWork{Call: call}},
					},
				)
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
			if allAutomaticToolItemsSucceeded(execution, state.PendingToolIDs) {
				return startAgentLLM(state, payload.Defaults)
			}
			status := ProgramFrameStateWaiting
			return ProgramTransition{State: encodeAgentLoopState(state), Status: &status}, nil
		}
	case ProgramEventActivity, ProgramEventChildSettled:
		// Agent loop observes only work settlement and continue; no action.
	case ProgramEventContinue:
		if state.AwaitingContinue && allToolItemsSettled(execution, state.PendingToolIDs) {
			return startAgentLLM(state, payload.Defaults)
		}
	case ProgramEventCancelled:
		status := ProgramFrameStateCancelled
		return ProgramTransition{State: encodeAgentLoopState(state), Status: &status}, nil
	}
	return ProgramTransition{State: encodeAgentLoopState(state)}, nil
}

func startAgentLLM(state agentLoopProgramState, defaults SessionDefaults) (ProgramTransition, error) {
	continuations := append([]WorkItemID(nil), state.PendingToolIDs...)
	state.Round++
	state.PendingToolIDs = nil
	state.AwaitingContinue = false
	status := ProgramFrameStateRunning
	return ProgramTransition{
		State:  encodeAgentLoopState(state),
		Status: &status,
		WorkItems: []WorkItemIntent{
			{
				ProgramPath: fmt.Sprintf("llm/%d", state.Round),
				Type:        WorkItemTypeLLM,
				Automatic:   true,
				Effect:      EffectContract{Kind: EffectKindRead},
				Input: WorkItemPayload{
					LLM: &LLMWork{
						Model:                   cloneSelection(defaults.Model),
						Agent:                   cloneSelection(defaults.Agent),
						InputActivityID:         state.InputActivityID,
						ContinuationWorkItemIDs: continuations,
						SystemInstructions:      append([]Instruction(nil), defaults.SystemInstructions...),
						Temperature:             cloneFloat(defaults.Temperature),
					},
				},
			},
		},
	}, nil
}

func failedAgentTransition(state agentLoopProgramState, code, message string) (ProgramTransition, error) {
	status := ProgramFrameStateFailed
	return ProgramTransition{
		State:  encodeAgentLoopState(state),
		Status: &status,
		Error:  &SanitizedError{Code: code, Message: message},
	}, nil
}

func encodeAgentLoopState(value agentLoopProgramState) ProgramState {
	return ProgramState{
		Type:    BuiltinAgentLoopInterpreterID,
		Version: BuiltinInterpreterVersion,
		Data:    MustCanonicalValue(value),
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
					Summary:    "Provider managed tool observed.",
					ActionKind: "tool",
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
		return errors.New("unsupported single-model program version")
	}
	var value singleModelProgramPayload
	if err := json.Unmarshal(snapshot.Payload, &value); err != nil {
		return err
	}
	if value.InputActivityID == "" {
		return errors.New("single-model input Activity is required")
	}
	return nil
}

func (singleModelInterpreter) ValidateState(state ProgramState) error {
	if state.Type != BuiltinSingleModelInterpreterID || state.Version != BuiltinInterpreterVersion {
		return errors.New("unsupported single-model state version")
	}
	var value singleModelProgramState
	return json.Unmarshal(state.Data, &value)
}

func (singleModelInterpreter) InitialState(snapshot ProgramSnapshot) (ProgramState, error) {
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
			status := ProgramFrameStateRunning
			return ProgramTransition{
				State:  encodeSingleModelState(state),
				Status: &status,
				WorkItems: []WorkItemIntent{
					{
						ProgramPath: "model",
						Type:        WorkItemTypeLLM,
						Automatic:   true,
						Effect:      EffectContract{Kind: EffectKindRead},
						Input: WorkItemPayload{
							LLM: &LLMWork{
								Model:              cloneSelection(payload.Defaults.Model),
								InputActivityID:    payload.InputActivityID,
								SystemInstructions: append([]Instruction(nil), payload.Defaults.SystemInstructions...),
								Temperature:        cloneFloat(payload.Defaults.Temperature),
							},
						},
					},
				},
			}, nil
		}
	case ProgramEventWorkSettled:
		item := executionWorkItem(execution, event.WorkItemID)
		if item == nil || item.Output == nil || item.Output.LLM == nil {
			status := ProgramFrameStateFailed
			return ProgramTransition{
				State:  encodeSingleModelState(state),
				Status: &status,
				Error: &SanitizedError{
					Code:    "model_output_missing",
					Message: "single-model ProgramFrame has no output",
				},
			}, nil
		}
		status := ProgramFrameStateCompleted
		return ProgramTransition{
			State:  encodeSingleModelState(state),
			Status: &status,
			Output: MustCanonicalValue(item.Output.LLM),
			Activities: []Activity{
				{
					Kind:       ActivityKindMessage,
					Visibility: ActivityVisibilityBoth,
					WorkItemID: item.ID,
					Message:    &MessageActivity{Role: MessageRoleAssistant, Text: item.Output.LLM.Text},
				},
			},
		}, nil
	case ProgramEventActivity, ProgramEventChildSettled, ProgramEventContinue:
		// Single-model observes only start, work settlement, and cancellation.
	case ProgramEventCancelled:
		status := ProgramFrameStateCancelled
		return ProgramTransition{State: encodeSingleModelState(state), Status: &status}, nil
	}
	return ProgramTransition{State: encodeSingleModelState(state)}, nil
}

func encodeSingleModelState(value singleModelProgramState) ProgramState {
	return ProgramState{
		Type:    BuiltinSingleModelInterpreterID,
		Version: BuiltinInterpreterVersion,
		Data:    MustCanonicalValue(value),
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
