package session

import (
	"encoding/json"
	"fmt"
)

const (
	SequentialChatDriverKind    = "sequential-chat"
	SequentialChatDriverVersion = uint32(1)
)

type sequentialChatDriverState struct {
	BatchOperationID OperationID `json:"batchOperationID,omitempty"`
	BatchConsumed    bool        `json:"batchConsumed"`
}

type SequentialChatDriver struct{}

func NewSequentialChatDriver() *SequentialChatDriver {
	return &SequentialChatDriver{}
}

func (d *SequentialChatDriver) Kind() string {
	return SequentialChatDriverKind
}

func (d *SequentialChatDriver) Version() uint32 {
	return SequentialChatDriverVersion
}

func (d *SequentialChatDriver) InitialState() DriverState {
	return encodeDriverState(
		d.Kind(),
		d.Version(),
		sequentialChatDriverState{},
	)
}

func (d *SequentialChatDriver) Advance(
	state DriverState,
	event DriverEvent,
) (DriverTransition, error) {
	if state.Kind != "" && state.Kind != d.Kind() {
		return DriverTransition{}, fmt.Errorf(
			"sequential chat driver cannot advance state for driver %q",
			state.Kind,
		)
	}

	if state.Version != 0 && state.Version != d.Version() {
		return DriverTransition{}, fmt.Errorf(
			"sequential chat driver does not support state version %d",
			state.Version,
		)
	}

	current := sequentialChatDriverState{}
	if len(state.Data) != 0 {
		if err := json.Unmarshal(state.Data, &current); err != nil {
			return DriverTransition{}, fmt.Errorf(
				"decode sequential chat driver state: %w",
				err,
			)
		}
	}

	view := event.View
	if event.Kind == DriverEventModelFinished {
		current.BatchOperationID = event.OperationID
		current.BatchConsumed = false

		if !event.ModelUsable && !event.ModelHasToolCalls {
			outcome := RunOutcome{
				Kind:    RunOutcomeFailed,
				Code:    "model_response_unusable",
				Message: "the model operation ended without a usable response",
			}
			return DriverTransition{
				NextState: encodeDriverState(d.Kind(), d.Version(), current),
				Outcome:   &outcome,
			}, nil
		}
	}

	if view.RunState.Terminal() ||
		view.RunState == RunStatePaused ||
		view.RunState == RunStateCancelling ||
		view.Controls.Paused ||
		view.Controls.CancellationRequested {
		return DriverTransition{
			NextState: encodeDriverState(d.Kind(), d.Version(), current),
		}, nil
	}

	// The sequential driver never overlaps a new model operation with tools
	// from an earlier batch.
	if view.ActiveModelOperationID != "" {
		return DriverTransition{
			NextState: encodeDriverState(d.Kind(), d.Version(), current),
		}, nil
	}

	batch := localBatchCalls(view.ToolCalls, current.BatchOperationID)
	if len(batch) == 0 {
		if event.Kind == DriverEventModelFinished ||
			(event.Kind == DriverEventResumed &&
				view.ActiveModelOperationID == "") {
			outcome := RunOutcome{Kind: RunOutcomeCompleted}
			return DriverTransition{
				NextState: encodeDriverState(d.Kind(), d.Version(), current),
				Outcome:   &outcome,
			}, nil
		}

		return DriverTransition{
			NextState: encodeDriverState(d.Kind(), d.Version(), current),
		}, nil
	}

	if hasInFlight(batch) {
		running := RunStateRunning
		return DriverTransition{
			NextState:    encodeDriverState(d.Kind(), d.Version(), current),
			SetLifecycle: &running,
		}, nil
	}

	if requested := firstRequestedOutstanding(view.RequestedCallIDs, batch); requested != nil {
		running := RunStateRunning
		return DriverTransition{
			NextState: encodeDriverState(d.Kind(), d.Version(), current),
			Operations: []OperationIntent{{
				Kind:       OperationIntentExecuteTool,
				ToolCallID: requested.ID,
				Automatic:  false,
			}},
			SetLifecycle: &running,
		}, nil
	}

	if first := firstPending(batch); first != nil {
		if first.Automatic &&
			!automationHoldApplies(
				view.Controls.AutomaticTools,
				current.BatchOperationID,
			) {
			running := RunStateRunning
			return DriverTransition{
				NextState: encodeDriverState(d.Kind(), d.Version(), current),
				Operations: []OperationIntent{{
					Kind:       OperationIntentExecuteTool,
					ToolCallID: first.ID,
					Automatic:  true,
				}},
				SetLifecycle: &running,
			}, nil
		}

		waiting := RunStateWaiting
		return DriverTransition{
			NextState:    encodeDriverState(d.Kind(), d.Version(), current),
			SetLifecycle: &waiting,
		}, nil
	}

	if hasUnresolved(batch) {
		waiting := RunStateWaiting
		return DriverTransition{
			NextState:    encodeDriverState(d.Kind(), d.Version(), current),
			SetLifecycle: &waiting,
		}, nil
	}

	if current.BatchConsumed {
		return DriverTransition{
			NextState: encodeDriverState(d.Kind(), d.Version(), current),
		}, nil
	}

	automaticContinuation := !view.ContinuationRequested &&
		allAutomaticAndSuccessful(batch) &&
		!automationHoldApplies(
			view.Controls.AutomaticContinuation,
			current.BatchOperationID,
		)
	if view.ContinuationRequested || automaticContinuation {
		current.BatchConsumed = true
		running := RunStateRunning
		return DriverTransition{
			NextState: encodeDriverState(d.Kind(), d.Version(), current),
			Operations: []OperationIntent{{
				Kind:      OperationIntentStartModel,
				Automatic: automaticContinuation,
			}},
			SetLifecycle: &running,
		}, nil
	}

	waiting := RunStateWaiting
	return DriverTransition{
		NextState:    encodeDriverState(d.Kind(), d.Version(), current),
		SetLifecycle: &waiting,
	}, nil
}

func localBatchCalls(
	calls []DriverToolCallView,
	operationID OperationID,
) []DriverToolCallView {
	output := make([]DriverToolCallView, 0, len(calls))

	for _, call := range calls {
		if call.OriginOperationID != operationID ||
			!call.Required ||
			call.ProviderManaged {
			continue
		}
		output = append(output, call)
	}

	return output
}

func hasInFlight(calls []DriverToolCallView) bool {
	for _, call := range calls {
		switch call.State {
		case ToolCallStateEvaluationQueued,
			ToolCallStateExecutionQueued,
			ToolCallStateEvaluating,
			ToolCallStateRunning:
			return true
		default:
		}
	}

	return false
}

func firstRequestedOutstanding(
	requested []ToolCallID,
	calls []DriverToolCallView,
) *DriverToolCallView {
	byID := make(map[ToolCallID]DriverToolCallView, len(calls))
	for _, call := range calls {
		byID[call.ID] = call
	}

	for _, id := range requested {
		call, found := byID[id]
		if !found || call.State == ToolCallStateResolved {
			continue
		}
		if call.State == ToolCallStatePending ||
			call.State == ToolCallStateReady {
			copyValue := call
			return &copyValue
		}
	}

	return nil
}

func firstPending(calls []DriverToolCallView) *DriverToolCallView {
	for _, call := range calls {
		if call.State != ToolCallStatePending &&
			call.State != ToolCallStateReady {
			continue
		}
		copyValue := call
		return &copyValue
	}

	return nil
}

func hasUnresolved(calls []DriverToolCallView) bool {
	for _, call := range calls {
		if call.State == ToolCallStateResolved ||
			call.State == ToolCallStateObserved {
			continue
		}
		return true
	}

	return false
}

func allAutomaticAndSuccessful(calls []DriverToolCallView) bool {
	if len(calls) == 0 {
		return false
	}

	for _, call := range calls {
		if !call.Automatic ||
			call.State != ToolCallStateResolved ||
			call.SelectedResult == nil ||
			*call.SelectedResult != ToolResultKindSuccess {
			return false
		}
	}

	return true
}

func automationHoldApplies(
	hold AutomationHold,
	operationID OperationID,
) bool {
	if !hold.Enabled {
		return false
	}

	if hold.Scope == AutomationHoldIndefinite {
		return true
	}

	return hold.TargetOperationID == "" ||
		hold.TargetOperationID == operationID
}
