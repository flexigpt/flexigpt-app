package sessionv3

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

const (
	retryBoundaryInterpreterVersion = uint32(1)
	retryBoundaryWorkPrefix         = "retry-backoff"
	retryBoundaryFramePrefix        = "retry-frame"
)

type retryProgramPayload struct {
	Child ProgramSnapshot `json:"child"`
	Retry RetryPolicy     `json:"retry"`
}

type retryProgramAttempt struct {
	ID         AttemptID       `json:"id"`
	Number     int             `json:"number"`
	FrameID    ProgramFrameID  `json:"frameID"`
	StartedAt  time.Time       `json:"startedAt"`
	FinishedAt *time.Time      `json:"finishedAt,omitempty"`
	Error      *SanitizedError `json:"error,omitempty"`
}

type retryProgramState struct {
	Attempt         int                      `json:"attempt"`
	ChildFrameID    ProgramFrameID           `json:"childFrameID,omitempty"`
	RetryWorkItemID WorkItemID               `json:"retryWorkItemID,omitempty"`
	RetryAt         *time.Time               `json:"retryAt,omitempty"`
	LastError       *SanitizedError          `json:"lastError,omitempty"`
	Attempts        []retryProgramAttempt    `json:"attempts,omitempty"`
	TargetSets      []WorkflowTargetSetState `json:"targetSets,omitempty"`
}

func NewRetryBoundaryProgram(
	child ProgramSnapshot,
	retry RetryPolicy,
) (ProgramSnapshot, error) {
	if child.InterpreterID == "" ||
		child.InterpreterVersion == 0 ||
		!child.Payload.Valid() {
		return ProgramSnapshot{}, errors.New("retry boundary child program is invalid")
	}
	if err := retry.Validate(); err != nil {
		return ProgramSnapshot{}, err
	}
	return programSnapshot(
		BuiltinRetryBoundaryInterpreterID,
		retryProgramPayload{
			Child: child,
			Retry: retry,
		},
	)
}

type retryBoundaryInterpreter struct{}

func (retryBoundaryInterpreter) ID() string {
	return BuiltinRetryBoundaryInterpreterID
}

func (retryBoundaryInterpreter) Version() uint32 {
	return retryBoundaryInterpreterVersion
}

func (retryBoundaryInterpreter) ValidateSnapshot(
	snapshot ProgramSnapshot,
) error {
	if snapshot.InterpreterID != BuiltinRetryBoundaryInterpreterID ||
		snapshot.InterpreterVersion != retryBoundaryInterpreterVersion {
		return errors.New("unsupported retry boundary program version")
	}

	var payload retryProgramPayload
	if err := json.Unmarshal(snapshot.Payload, &payload); err != nil {
		return err
	}
	if payload.Child.InterpreterID == "" ||
		payload.Child.InterpreterVersion == 0 ||
		!payload.Child.Payload.Valid() {
		return errors.New("retry boundary child program is invalid")
	}
	return payload.Retry.Validate()
}

func (retryBoundaryInterpreter) ValidateState(state ProgramState) error {
	if state.Type != BuiltinRetryBoundaryInterpreterID ||
		state.Version != retryBoundaryInterpreterVersion {
		return errors.New("unsupported retry boundary state version")
	}

	var value retryProgramState
	if err := json.Unmarshal(state.Data, &value); err != nil {
		return err
	}
	return value.Validate()
}

func (retryBoundaryInterpreter) InitialState(
	_ ProgramSnapshot,
) (ProgramState, error) {
	return encodeRetryProgramState(retryProgramState{}), nil
}

func (i retryBoundaryInterpreter) Advance(
	frame ProgramFrame,
	execution Execution,
	event ProgramEvent,
) (ProgramTransition, error) {
	var payload retryProgramPayload
	if err := json.Unmarshal(frame.Program.Payload, &payload); err != nil {
		return ProgramTransition{}, err
	}
	var state retryProgramState
	if err := json.Unmarshal(frame.State.Data, &state); err != nil {
		return ProgramTransition{}, err
	}

	now := retryProgramEventTime(frame, event)

	switch event.Kind {
	case ProgramEventStarted:
		if state.ChildFrameID == "" && state.RetryWorkItemID == "" {
			return i.startChild(frame, payload, state, now)
		}

	case ProgramEventChildSettled:
		if event.ChildFrameID != state.ChildFrameID {
			return ProgramTransition{
				State: encodeRetryProgramState(state),
			}, nil
		}
		return i.settleChild(frame, execution, payload, state, now)

	case ProgramEventWorkSettled:
		if event.WorkItemID != state.RetryWorkItemID {
			return ProgramTransition{
				State: encodeRetryProgramState(state),
			}, nil
		}
		return i.settleBackoff(frame, execution, payload, state, now)

	case ProgramEventContinue:
		if state.ChildFrameID != "" {
			child := mutableFrame(&execution, state.ChildFrameID)
			if child != nil && child.Status.Terminal() {
				return i.settleChild(frame, execution, payload, state, now)
			}
		}
		if state.RetryWorkItemID != "" {
			item := executionWorkItem(execution, state.RetryWorkItemID)
			if item != nil && item.State.Terminal() {
				return i.settleBackoff(
					frame,
					execution,
					payload,
					state,
					now,
				)
			}
		}

	case ProgramEventCancelled:
		status := ProgramFrameStateCancelled
		return ProgramTransition{
			State:  encodeRetryProgramState(state),
			Status: &status,
			Error: &SanitizedError{
				Code:     "workflow_retry_cancelled",
				Category: ErrorCategoryCancellation,
				Message:  "workflow retry boundary was cancelled",
			},
		}, nil
	default:
	}

	return ProgramTransition{
		State: encodeRetryProgramState(state),
	}, nil
}

func (i retryBoundaryInterpreter) startChild(
	frame ProgramFrame,
	payload retryProgramPayload,
	state retryProgramState,
	now time.Time,
) (ProgramTransition, error) {
	if state.Attempt == 0 {
		state.Attempt = 1
	}

	childID := ProgramFrameID(
		retryBoundaryFramePrefix + ":" +
			string(frame.ID) + ":" +
			strconv.Itoa(state.Attempt),
	)
	child, err := retryProgramChildSnapshot(
		payload.Child,
		state.TargetSets,
	)
	if err != nil {
		return ProgramTransition{}, err
	}

	deadline := cloneTime(frame.Deadline)
	if payload.Retry.PerAttemptTimeout != "" {
		timeout, err := payload.Retry.PerAttemptTimeout.Deadline(now)
		if err != nil {
			return ProgramTransition{}, err
		}
		deadline = earlierDeadline(deadline, &timeout)
	}

	state.ChildFrameID = childID
	state.RetryWorkItemID = ""
	state.RetryAt = nil
	state.Attempts = append(state.Attempts, retryProgramAttempt{
		ID: AttemptID(
			retryBoundaryFramePrefix + ":" +
				string(frame.ID) + ":" +
				strconv.Itoa(state.Attempt),
		),
		Number:    state.Attempt,
		FrameID:   childID,
		StartedAt: now,
	})

	status := ProgramFrameStateRunning
	return ProgramTransition{
		State:  encodeRetryProgramState(state),
		Status: &status,
		Frames: []ProgramFrameIntent{{
			ID:       childID,
			Path:     "attempt/" + strconv.Itoa(state.Attempt),
			Program:  child,
			Deadline: deadline,
		}},
	}, nil
}

func (i retryBoundaryInterpreter) settleChild(
	frame ProgramFrame,
	execution Execution,
	payload retryProgramPayload,
	state retryProgramState,
	now time.Time,
) (ProgramTransition, error) {
	child := mutableFrame(&execution, state.ChildFrameID)
	if child == nil || child.ParentFrameID != frame.ID {
		return ProgramTransition{}, errors.New(
			"retry boundary child program frame is unavailable",
		)
	}
	if !child.Status.Terminal() {
		return ProgramTransition{
			State: encodeRetryProgramState(state),
		}, nil
	}

	failure := retryBoundaryFrameFailure(*child)
	retryCaptureTargetSets(*child, &state)
	retryFinishAttempt(&state, child.ID, now, failure)
	state.ChildFrameID = ""

	if child.Status == ProgramFrameStateCompleted {
		status := ProgramFrameStateCompleted
		return ProgramTransition{
			State:  encodeRetryProgramState(state),
			Status: &status,
			Output: child.Output.Clone(),
		}, nil
	}

	if retryBoundaryCanRetry(execution, payload.Retry, state, *child, failure) {
		return i.scheduleBackoff(frame, payload, state, now)
	}

	status := ProgramFrameStateFailed
	if child.Status == ProgramFrameStateCancelled {
		status = ProgramFrameStateCancelled
	}
	return ProgramTransition{
		State:  encodeRetryProgramState(state),
		Status: &status,
		Error:  failure,
	}, nil
}

func (i retryBoundaryInterpreter) scheduleBackoff(
	frame ProgramFrame,
	payload retryProgramPayload,
	state retryProgramState,
	now time.Time,
) (ProgramTransition, error) {
	next, err := retryAt(
		payload.Retry,
		WorkItemID(frame.ID),
		state.Attempt,
		now,
	)
	if err != nil {
		return ProgramTransition{}, err
	}

	workID := WorkItemID(
		retryBoundaryWorkPrefix + ":" +
			string(frame.ID) + ":" +
			strconv.Itoa(state.Attempt+1),
	)
	state.RetryWorkItemID = workID
	state.RetryAt = &next

	status := ProgramFrameStateWaiting
	return ProgramTransition{
		State:  encodeRetryProgramState(state),
		Status: &status,
		WorkItems: []WorkItemIntent{{
			ID:          workID,
			ProgramPath: "retry/backoff/" + strconv.Itoa(state.Attempt+1),
			Type:        WorkItemTypeWait,
			Automatic:   true,
			Effect:      EffectContract{Kind: EffectKindRead},
			Input: WorkItemPayload{
				Wait: &WaitWork{
					Request: WaitRequest{
						Kind:     WaitKindTimer,
						Deadline: cloneTime(frame.Deadline),
						Timer: &TimerWait{
							ScheduledAt: now,
							FireAt:      next,
						},
					},
				},
			},
		}},
	}, nil
}

func (i retryBoundaryInterpreter) settleBackoff(
	frame ProgramFrame,
	execution Execution,
	payload retryProgramPayload,
	state retryProgramState,
	now time.Time,
) (ProgramTransition, error) {
	item := executionWorkItem(execution, state.RetryWorkItemID)
	if item == nil || item.FrameID != frame.ID {
		return ProgramTransition{}, errors.New(
			"retry boundary backoff work item is unavailable",
		)
	}
	if !item.State.Terminal() {
		return ProgramTransition{
			State: encodeRetryProgramState(state),
		}, nil
	}
	if item.State != WorkItemStateCompleted {
		status := ProgramFrameStateFailed
		return ProgramTransition{
			State:  encodeRetryProgramState(state),
			Status: &status,
			Error:  retryBoundaryWorkFailure(*item),
		}, nil
	}

	state.RetryWorkItemID = ""
	state.RetryAt = nil
	state.Attempt++
	return i.startChild(frame, payload, state, now)
}

func retryBoundaryCanRetry(
	execution Execution,
	policy RetryPolicy,
	state retryProgramState,
	child ProgramFrame,
	failure *SanitizedError,
) bool {
	if failure == nil ||
		state.Attempt >= policy.MaxAttempts ||
		execution.Controls.CancellationRequested ||
		!retryMatches(policy.Match, *failure) {
		return false
	}
	return retryBoundaryReplaySafe(execution, child.ID)
}

func retryBoundaryReplaySafe(
	execution Execution,
	root ProgramFrameID,
) bool {
	for _, item := range execution.WorkItems {
		if item.FrameID != root &&
			!isDescendantFrame(execution, root, item.FrameID) {
			continue
		}
		if item.State == WorkItemStateUncertain ||
			item.EffectState == EffectStateUncertain {
			return false
		}
		if item.Effect.Kind == EffectKindRead {
			continue
		}
		if effectReplaySafe(item.Effect) ||
			item.EffectState == EffectStateConfirmedNotApplied {
			continue
		}
		return false
	}
	return true
}

func retryBoundaryFrameFailure(
	frame ProgramFrame,
) *SanitizedError {
	if frame.Error != nil {
		return cloneWorkflowSanitizedError(frame.Error)
	}
	if frame.Status == ProgramFrameStateCancelled {
		return &SanitizedError{
			Code:     workflowChildCancelled,
			Category: ErrorCategoryCancellation,
			Message:  "retry boundary child program frame was cancelled",
		}
	}
	return &SanitizedError{
		Code:     "workflow_retry_child_failed",
		Category: ErrorCategoryExecution,
		Message:  "retry boundary child program frame failed",
	}
}

func retryBoundaryWorkFailure(
	item WorkItem,
) *SanitizedError {
	if item.Error != nil {
		return cloneWorkflowSanitizedError(item.Error)
	}
	return &SanitizedError{
		Code:     "workflow_retry_backoff_failed",
		Category: ErrorCategoryExecution,
		Message:  "workflow retry backoff did not complete",
	}
}

func retryFinishAttempt(
	state *retryProgramState,
	frameID ProgramFrameID,
	now time.Time,
	failure *SanitizedError,
) {
	if state == nil {
		return
	}
	for index := len(state.Attempts) - 1; index >= 0; index-- {
		attempt := &state.Attempts[index]
		if attempt.FrameID != frameID || attempt.FinishedAt != nil {
			continue
		}
		attempt.FinishedAt = &now
		attempt.Error = cloneWorkflowSanitizedError(failure)
		break
	}
	state.LastError = cloneWorkflowSanitizedError(failure)
}

func retryCaptureTargetSets(
	frame ProgramFrame,
	state *retryProgramState,
) {
	if state == nil || frame.Program.InterpreterID != WorkflowInterpreterID {
		return
	}
	_, workflowState, err := decodeWorkflowProgram(frame)
	if err != nil {
		return
	}
	state.TargetSets = workflowState.TargetSets
}

func retryProgramChildSnapshot(
	snapshot ProgramSnapshot,
	targetSets []WorkflowTargetSetState,
) (ProgramSnapshot, error) {
	if len(targetSets) == 0 ||
		snapshot.InterpreterID != WorkflowInterpreterID {
		return snapshot, nil
	}

	program, err := workflowProgramFromSnapshot(snapshot)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	program.Scope.TargetSets = targetSets

	payload, err := canonicalWorkflowValue(program)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	snapshot.Payload = payload
	return snapshot, nil
}

func retryProgramEventTime(
	frame ProgramFrame,
	event ProgramEvent,
) time.Time {
	if !event.At.IsZero() {
		return event.At.UTC()
	}
	if !frame.UpdatedAt.IsZero() {
		return frame.UpdatedAt.UTC()
	}
	return frame.CreatedAt.UTC()
}

func encodeRetryProgramState(
	value retryProgramState,
) ProgramState {
	return ProgramState{
		Type:    BuiltinRetryBoundaryInterpreterID,
		Version: retryBoundaryInterpreterVersion,
		Data:    MustCanonicalValue(value),
	}
}

func (s retryProgramState) Validate() error {
	if s.Attempt < 0 {
		return errors.New("retry boundary attempt is invalid")
	}
	if s.ChildFrameID != "" {
		if err := validateID(
			"retry boundary child frame ID",
			string(s.ChildFrameID),
		); err != nil {
			return err
		}
	}
	if s.RetryWorkItemID != "" {
		if err := validateID(
			"retry boundary work item ID",
			string(s.RetryWorkItemID),
		); err != nil {
			return err
		}
		if s.RetryAt == nil || s.RetryAt.IsZero() {
			return errors.New("retry boundary backoff has no fire time")
		}
	}
	if s.ChildFrameID != "" && s.RetryWorkItemID != "" {
		return errors.New("retry boundary has child and backoff work")
	}
	if s.LastError != nil {
		if err := validateReconciliationError(s.LastError); err != nil {
			return err
		}
	}

	seen := make(map[AttemptID]struct{}, len(s.Attempts))
	for _, attempt := range s.Attempts {
		if attempt.ID == "" || attempt.Number < 1 ||
			attempt.FrameID == "" || attempt.StartedAt.IsZero() {
			return errors.New("retry boundary attempt is invalid")
		}
		if attempt.FinishedAt != nil &&
			attempt.FinishedAt.Before(attempt.StartedAt) {
			return errors.New("retry boundary attempt finished before it started")
		}
		if err := validateReconciliationError(attempt.Error); err != nil {
			return err
		}
		if _, duplicate := seen[attempt.ID]; duplicate {
			return errors.New("retry boundary attempt is duplicated")
		}
		seen[attempt.ID] = struct{}{}
	}
	return validateWorkflowTargetSetStates(s.TargetSets)
}
