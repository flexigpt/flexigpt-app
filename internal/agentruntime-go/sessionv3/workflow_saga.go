package sessionv3

import (
	"errors"
	"slices"
	"strings"
)

type WorkflowSaga struct {
	CompensateOn       []WorkflowSagaOutcome   `json:"compensateOn,omitempty"`
	CompensationErrors CompensationErrorPolicy `json:"compensationErrors,omitempty"`
	Concurrency        int                     `json:"concurrency"`
	Do                 WorkflowBlock           `json:"do"`
}

type WorkflowSagaOutcome string

const (
	WorkflowSagaOutcomeFailed    WorkflowSagaOutcome = "failed"
	WorkflowSagaOutcomeCancelled WorkflowSagaOutcome = "cancelled"
)

type WorkflowSagaPhase string

const (
	WorkflowSagaPhaseBody               WorkflowSagaPhase = "body"
	WorkflowSagaPhaseCompensationIntent WorkflowSagaPhase = "compensationIntent"
	WorkflowSagaPhaseCompensationChild  WorkflowSagaPhase = "compensationChild"
)

type WorkflowSagaScope struct {
	ID                 string                  `json:"id"`
	CompensationPolicy CompensationErrorPolicy `json:"compensationPolicy"`
}

type WorkflowSagaProgress struct {
	ID                 string                  `json:"id"`
	CompensateOn       []WorkflowSagaOutcome   `json:"compensateOn"`
	CompensationPolicy CompensationErrorPolicy `json:"compensationPolicy"`
	Concurrency        int                     `json:"concurrency"`
	Phase              WorkflowSagaPhase       `json:"phase"`
	BodyPath           string                  `json:"bodyPath"`
	PrimaryStatus      ProgramFrameState       `json:"primaryStatus,omitempty"`
	PrimaryError       *SanitizedError         `json:"primaryError,omitempty"`
	PrimaryValue       CanonicalValue          `json:"primaryValue,omitempty"`
	Remaining          []EffectRecordID        `json:"remaining,omitempty"`
	ActiveEffectID     EffectRecordID          `json:"activeEffectID,omitempty"`
	ActiveWorkItemID   WorkItemID              `json:"activeWorkItemID,omitempty"`
	ActiveChildPath    string                  `json:"activeChildPath,omitempty"`
	Causes             []SanitizedError        `json:"causes,omitempty"`
}

func (s WorkflowSaga) Validate() error {
	if len(s.CompensateOn) == 0 {
		return errors.New("workflow saga has no compensation outcomes")
	}
	seen := make(map[WorkflowSagaOutcome]struct{}, len(s.CompensateOn))
	for _, value := range s.CompensateOn {
		switch value {
		case WorkflowSagaOutcomeFailed, WorkflowSagaOutcomeCancelled:
		default:
			return errors.New("workflow saga compensation outcome is invalid")
		}
		if _, found := seen[value]; found {
			return errors.New("workflow saga compensation outcome is duplicated")
		}
		seen[value] = struct{}{}
	}
	if err := s.CompensationErrors.Validate(); err != nil {
		return err
	}
	if s.Concurrency < 1 {
		return errors.New("workflow saga compensation concurrency is invalid")
	}
	return s.Do.Validate()
}

func (s WorkflowSagaScope) Validate() error {
	if err := validateID("workflow saga scope ID", s.ID); err != nil {
		return err
	}
	return s.CompensationPolicy.Validate()
}

func (p WorkflowSagaProgress) Validate() error {
	if err := validateID("workflow saga ID", p.ID); err != nil {
		return err
	}
	if err := validateID("workflow saga body path", p.BodyPath); err != nil {
		return err
	}
	if err := p.CompensationPolicy.Validate(); err != nil {
		return err
	}
	if p.Concurrency < 1 {
		return errors.New("workflow saga concurrency is invalid")
	}
	switch p.Phase {
	case WorkflowSagaPhaseBody,
		WorkflowSagaPhaseCompensationIntent,
		WorkflowSagaPhaseCompensationChild:
	default:
		return errors.New("workflow saga phase is invalid")
	}
	if p.PrimaryStatus != "" &&
		p.PrimaryStatus != ProgramFrameStateFailed &&
		p.PrimaryStatus != ProgramFrameStateCancelled {
		return errors.New("workflow saga primary status is invalid")
	}
	if p.PrimaryStatus != "" && p.PrimaryError == nil {
		return errors.New("workflow saga primary error is unavailable")
	}
	if p.ActiveEffectID != "" {
		if err := validateID(
			"workflow saga active effect ID",
			string(p.ActiveEffectID),
		); err != nil {
			return err
		}
	}
	if p.ActiveWorkItemID != "" {
		if err := validateID(
			"workflow saga active work item ID",
			string(p.ActiveWorkItemID),
		); err != nil {
			return err
		}
	}
	for _, cause := range p.Causes {
		if strings.TrimSpace(cause.Code) == "" ||
			strings.TrimSpace(cause.Message) == "" ||
			!cause.Details.Valid() {
			return errors.New("workflow saga cause is invalid")
		}
	}
	return nil
}

func (i *WorkflowInterpreter) startSaga(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	path string,
) (ProgramTransition, error) {
	if step.Saga == nil {
		return workflowFailure(state, "workflow_saga_invalid", "workflow saga is missing")
	}
	if err := step.Saga.Validate(); err != nil {
		return workflowFailure(state, "workflow_saga_invalid", err.Error())
	}

	id := string(frame.ID) + ":" + path + ":saga"
	scope := WorkflowScopeContext{
		Saga: &WorkflowSagaScope{
			ID:                 id,
			CompensationPolicy: step.Saga.CompensationErrors,
		},
	}
	snapshot, err := i.childBlockProgramWithScope(
		frame,
		program,
		state,
		step.Saga.Do,
		step.Saga.Do.State == nil,
		scope,
	)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_saga_prepare_failed",
			err.Error(),
		)
	}

	bodyPath := path + "/saga/body"
	state.Pending = &WorkflowPending{
		Kind:   WorkflowPendingSaga,
		StepID: step.ID,
		Path:   path,
		Children: []WorkflowChild{{
			Name:  "body",
			Path:  bodyPath,
			State: WorkflowChildStatePending,
		}},
		Saga: &WorkflowSagaProgress{
			ID:                 id,
			CompensateOn:       append([]WorkflowSagaOutcome(nil), step.Saga.CompensateOn...),
			CompensationPolicy: step.Saga.CompensationErrors,
			Concurrency:        step.Saga.Concurrency,
			Phase:              WorkflowSagaPhaseBody,
			BodyPath:           bodyPath,
		},
	}
	return workflowStructuredTransition(
		state,
		[]ProgramFrameIntent{{
			Path:     bodyPath,
			Program:  snapshot,
			Deadline: cloneTime(state.Deadline),
		}},
	)
}

func (i *WorkflowInterpreter) driveSaga(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
) (ProgramTransition, error) {
	if state.Pending == nil || state.Pending.Saga == nil {
		return ProgramTransition{}, errors.New("workflow saga state is unavailable")
	}
	progress := state.Pending.Saga

	if progress.PrimaryStatus == "" {
		return workflowTransition(state, nil, nil, nil)
	}
	if len(progress.Remaining) == 0 &&
		progress.ActiveEffectID == "" {
		return i.finishSaga(frame, program, state), nil
	}
	if progress.ActiveEffectID != "" {
		return workflowTransition(state, nil, nil, nil)
	}

	effectID := progress.Remaining[0]
	progress.Remaining = progress.Remaining[1:]
	record := effectRecordByID(&execution, effectID)
	if record == nil {
		return workflowFailure(
			state,
			"workflow_compensation_invalid",
			"workflow compensation effect record is unavailable",
		)
	}
	workID := WorkItemID(
		string(frame.ID) + ":saga:" + string(effectID) + ":compensation",
	)
	progress.ActiveEffectID = effectID
	progress.ActiveWorkItemID = workID
	progress.Phase = WorkflowSagaPhaseCompensationIntent

	return workflowTransition(
		state,
		nil,
		nil,
		[]WorkItemIntent{{
			ID:          workID,
			ProgramPath: state.Pending.Path + "/compensation/" + string(effectID),
			Type:        WorkItemTypeCompensation,
			Automatic:   true,
			Cleanup:     true,
			Effect:      EffectContract{Kind: EffectKindRead},
			Input: WorkItemPayload{
				Compensation: &CompensationWork{
					OriginalWorkItemID: record.OriginalWorkItemID,
					EffectID:           effectID,
					Input:              MustCanonicalValue(record.Completed),
				},
			},
		}},
	)
}

func (i *WorkflowInterpreter) settleSagaWork(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	workID WorkItemID,
) (ProgramTransition, bool, error) {
	if state.Pending == nil ||
		state.Pending.Kind != WorkflowPendingSaga ||
		state.Pending.Saga == nil {
		return ProgramTransition{}, false, nil
	}
	progress := state.Pending.Saga
	if progress.Phase != WorkflowSagaPhaseCompensationIntent ||
		progress.ActiveWorkItemID != workID {
		return ProgramTransition{}, false, nil
	}
	item := executionWorkItem(execution, workID)
	if item == nil || item.FrameID != frame.ID {
		return ProgramTransition{}, true, errors.New(
			"workflow compensation work item is unavailable",
		)
	}
	if item.State != WorkItemStateCompleted {
		cause := workflowSettledFailure(program, state, *item)
		if cause == nil {
			cause = &SanitizedError{
				Code:     "compensation_work_failed",
				Category: ErrorCategoryExecution,
				Message:  "compensation work item did not complete",
			}
		}
		return i.settleSagaCompensationFailure(
			frame,
			execution,
			program,
			state,
			*cause,
		)
	}

	record := effectRecordByID(&execution, progress.ActiveEffectID)
	if record == nil || record.Compensation == nil {
		return ProgramTransition{}, true, errors.New(
			"workflow compensation block is unavailable",
		)
	}

	completed := MustCanonicalValue(record.Completed)
	scope := WorkflowScopeContext{
		Completed: &completed,
	}
	snapshot, err := i.childBlockProgramWithScope(
		frame,
		program,
		state,
		*record.Compensation,
		false,
		scope,
	)
	if err != nil {
		return ProgramTransition{}, true, err
	}

	childPath := state.Pending.Path + "/compensation/" +
		string(record.ID) + "/child"
	progress.ActiveChildPath = childPath
	progress.Phase = WorkflowSagaPhaseCompensationChild
	record.CompensationWorkItemID = workID

	t, err := workflowStructuredTransition(
		state,
		[]ProgramFrameIntent{{
			Path:     childPath,
			Program:  snapshot,
			Deadline: cloneTime(state.Deadline),
		}},
	)
	if err != nil {
		return ProgramTransition{}, false, err
	}
	return t, true, nil
}

func (i *WorkflowInterpreter) settleSagaChild(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	childID ProgramFrameID,
) (ProgramTransition, bool, error) {
	if state.Pending == nil ||
		state.Pending.Kind != WorkflowPendingSaga ||
		state.Pending.Saga == nil {
		return ProgramTransition{}, false, nil
	}
	progress := state.Pending.Saga

	child := workflowTerminalChildFrame(
		execution,
		frame.ID,
		progress.BodyPath,
	)
	if progress.Phase == WorkflowSagaPhaseCompensationChild {
		child = workflowTerminalChildFrame(
			execution,
			frame.ID,
			progress.ActiveChildPath,
		)
	}
	if child == nil || child.ID != childID || !child.Status.Terminal() {
		return ProgramTransition{}, false, nil
	}

	if progress.Phase == WorkflowSagaPhaseBody {
		result, failure := workflowFrameResult(*child)
		if failure == nil {
			return i.settleSagaBodySuccess(
				frame,
				program,
				state,
				result,
			), true, nil
		}
		return i.beginSagaCompensation(
			frame,
			execution,
			program,
			state,
			child.Status,
			failure,
		)
	}

	if progress.Phase != WorkflowSagaPhaseCompensationChild {
		return ProgramTransition{}, true, errors.New(
			"workflow saga child settled in an invalid phase",
		)
	}

	result, failure := workflowFrameResult(*child)
	if failure != nil {
		return i.settleSagaCompensationFailure(
			frame,
			execution,
			program,
			state,
			*failure,
		)
	}
	if result.Exit != WorkflowExitNormal &&
		result.Exit != WorkflowExitReturnBlock {
		return i.settleSagaCompensationFailure(
			frame,
			execution,
			program,
			state,
			SanitizedError{
				Code:     "compensation_control_invalid",
				Category: ErrorCategoryValidation,
				Message:  "compensation block returned unsupported control flow",
			},
		)
	}

	record := effectRecordByID(&execution, progress.ActiveEffectID)
	if record == nil {
		return ProgramTransition{}, true, errors.New(
			"workflow compensation effect record is unavailable",
		)
	}
	record.CompensationFrameID = child.ID
	settlement := CompensationSettlement{
		EffectID: progress.ActiveEffectID,
		State:    CompensationStateCompleted,
		Receipt:  compensationReceipt(record.ID, child.ID, state.LogicalTime),
	}

	progress.ActiveEffectID = ""
	progress.ActiveWorkItemID = ""
	progress.ActiveChildPath = ""
	progress.Phase = WorkflowSagaPhaseCompensationIntent

	next, err := i.driveSaga(frame, execution, program, state)
	if err != nil {
		return ProgramTransition{}, true, err
	}
	next.Compensations = append(next.Compensations, settlement)
	return next, true, nil
}

func (i *WorkflowInterpreter) settleSagaBodySuccess(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	result WorkflowBlockResult,
) ProgramTransition {
	if state.Pending == nil || state.Pending.Saga == nil {
		return ProgramTransition{}
	}
	pending := state.Pending
	switch result.Exit {
	case WorkflowExitNormal, WorkflowExitReturnBlock:
		step, err := workflowStepForPath(program.Root, pending.Path)
		if err != nil {
			transition, _ := workflowFailure(
				state,
				"workflow_saga_invalid",
				err.Error(),
			)
			return transition
		}
		if pending.MergeState {
			state.State = result.State.Clone()
		}
		if err := i.validateStepOutput(program, state, step, result.Value); err != nil {
			transition, _ := workflowFailure(
				state,
				"workflow_saga_output_invalid",
				err.Error(),
			)
			return transition
		}
		if err := state.recordOutcome(WorkflowStepOutcome{
			ID:     step.ID,
			Status: WorkflowStepOutcomeSucceeded,
			Output: result.Value.Clone(),
		}); err != nil {
			transition, _ := workflowFailure(
				state,
				"workflow_state_invalid",
				err.Error(),
			)
			return transition
		}
		state.Pending = nil
		state.Index++
		state.PendingPath = ""
		state.Deadline = nil
		transition, _ := i.drive(frame, Execution{}, program, state)
		return transition

	case WorkflowExitReturnWorkflow:
		state.Exit = result.Exit
		state.ExitValue = result.Value.Clone()
		state.Pending = nil
		transition, _ := i.drive(frame, Execution{}, program, state)
		return transition

	default:
		transition, _ := workflowFailure(
			state,
			"workflow_saga_control_invalid",
			"workflow saga body returned unsupported control flow",
		)
		return transition
	}
}

func (i *WorkflowInterpreter) beginSagaCompensation(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	status ProgramFrameState,
	failure *SanitizedError,
) (ProgramTransition, bool, error) {
	if state.Pending == nil || state.Pending.Saga == nil {
		return ProgramTransition{}, true, errors.New("workflow saga state is unavailable")
	}
	progress := state.Pending.Saga
	progress.PrimaryStatus = status
	progress.PrimaryError = cloneWorkflowSanitizedError(failure)

	if !sagaShouldCompensate(*progress) {
		return i.finishSaga(frame, program, state), true, nil
	}
	order, err := compensationRecordsInReverseDependencyOrder(
		execution,
		progress.ID,
	)
	if err != nil {
		return ProgramTransition{}, true, err
	}
	progress.Remaining = order
	progress.Phase = WorkflowSagaPhaseCompensationIntent
	if len(order) == 0 {
		return i.finishSaga(frame, program, state), true, nil
	}
	t, err := i.driveSaga(frame, execution, program, state)
	if err != nil {
		return ProgramTransition{}, true, err
	}
	return t, true, nil
}

func (i *WorkflowInterpreter) settleSagaCompensationFailure(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	failure SanitizedError,
) (ProgramTransition, bool, error) {
	if state.Pending == nil || state.Pending.Saga == nil {
		return ProgramTransition{}, true, errors.New("workflow saga state is unavailable")
	}
	progress := state.Pending.Saga
	record := effectRecordByID(&execution, progress.ActiveEffectID)
	if record != nil {
		record.CompensationFrameID = ""
	}
	progress.Causes = append(
		progress.Causes,
		compensationFailureCause(
			EffectRecord{OriginalWorkItemID: WorkItemID(progress.ActiveEffectID)},
			&failure,
		),
	)
	settlement := CompensationSettlement{
		EffectID: progress.ActiveEffectID,
		State:    CompensationStateFailed,
		Error:    cloneWorkflowSanitizedError(&failure),
	}
	progress.ActiveEffectID = ""
	progress.ActiveWorkItemID = ""
	progress.ActiveChildPath = ""
	progress.Phase = WorkflowSagaPhaseCompensationIntent

	if progress.CompensationPolicy == CompensationErrorPolicyStop {
		progress.Remaining = nil
	}
	next, err := i.driveSaga(frame, execution, program, state)
	if err != nil {
		return ProgramTransition{}, true, err
	}
	next.Compensations = append(next.Compensations, settlement)
	return next, true, nil
}

func (i *WorkflowInterpreter) finishSaga(
	_ ProgramFrame,
	_ WorkflowProgram,
	state WorkflowState,
) ProgramTransition {
	if state.Pending == nil || state.Pending.Saga == nil {
		transition, _ := workflowFailure(
			state,
			"workflow_saga_invalid",
			"workflow saga state is unavailable",
		)
		return transition
	}
	progress := state.Pending.Saga
	primary := compensationPrimaryError(progress.PrimaryError, progress.Causes)
	status := progress.PrimaryStatus
	if status == "" {
		status = ProgramFrameStateFailed
	}
	state.Pending = nil
	state.PendingPath = ""
	state.Deadline = nil
	next, err := workflowTransition(state, &status, nil, nil)
	if err != nil {
		transition, _ := workflowFailure(
			state,
			"workflow_saga_failed",
			err.Error(),
		)
		return transition
	}
	next.Error = primary
	return next
}

func sagaShouldCompensate(progress WorkflowSagaProgress) bool {
	switch progress.PrimaryStatus {
	case ProgramFrameStateFailed:
		if slices.Contains(progress.CompensateOn, WorkflowSagaOutcomeFailed) {
			return true
		}
	case ProgramFrameStateCancelled:
		if slices.Contains(progress.CompensateOn, WorkflowSagaOutcomeCancelled) {
			return true
		}
	default:
	}
	return false
}
