package sessionv3

import (
	"errors"
	"slices"
	"strings"
)

type WorkflowSagaCompensationPhase string

const (
	WorkflowSagaCompensationPhaseIntent WorkflowSagaCompensationPhase = "intent"
	WorkflowSagaCompensationPhaseChild  WorkflowSagaCompensationPhase = "child"
)

type WorkflowSagaActiveCompensation struct {
	EffectID   EffectRecordID                `json:"effectID"`
	WorkItemID WorkItemID                    `json:"workItemID"`
	ChildPath  string                        `json:"childPath,omitempty"`
	Phase      WorkflowSagaCompensationPhase `json:"phase"`
}

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
	ID                 string                           `json:"id"`
	CompensateOn       []WorkflowSagaOutcome            `json:"compensateOn"`
	CompensationPolicy CompensationErrorPolicy          `json:"compensationPolicy"`
	Concurrency        int                              `json:"concurrency"`
	Phase              WorkflowSagaPhase                `json:"phase"`
	BodyPath           string                           `json:"bodyPath"`
	PrimaryStatus      ProgramFrameState                `json:"primaryStatus,omitempty"`
	PrimaryError       *SanitizedError                  `json:"primaryError,omitempty"`
	PrimaryValue       CanonicalValue                   `json:"primaryValue,omitempty"`
	Remaining          []EffectRecordID                 `json:"remaining,omitempty"`
	Active             []WorkflowSagaActiveCompensation `json:"active,omitempty"`
	StopRequested      bool                             `json:"stopRequested,omitempty"`
	Causes             []SanitizedError                 `json:"causes,omitempty"`
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

	claims, err := i.workflowResourceClaims(program, state, step)
	if err != nil {
		return workflowFailure(state, "workflow_resource_claim_invalid", err.Error())
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
		Claims: claims,
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

	if progress.StopRequested {
		if len(progress.Active) == 0 {
			return i.finishSaga(frame, program, state), nil
		}
		return workflowTransition(state, nil, nil, nil)
	}

	intents := make([]WorkItemIntent, 0)
	for len(progress.Active) < progress.Concurrency {
		effectID, found, err := workflowSagaNextEligibleEffect(
			execution,
			*progress,
		)
		if err != nil {
			return workflowFailure(
				state,
				"workflow_compensation_invalid",
				err.Error(),
			)
		}
		if !found {
			break
		}

		record := effectRecordByID(&execution, effectID)
		if record == nil || record.Compensation == nil {
			return workflowFailure(
				state,
				"workflow_compensation_invalid",
				"workflow compensation effect record is unavailable",
			)
		}

		workID := WorkItemID(
			string(frame.ID) + ":saga:" + string(effectID) + ":compensation",
		)
		record.State = CompensationStateDispatching
		record.CompensationWorkItemID = workID
		progress.Remaining = removeWorkflowSagaEffect(
			progress.Remaining,
			effectID,
		)
		progress.Active = append(
			progress.Active,
			WorkflowSagaActiveCompensation{
				EffectID:   effectID,
				WorkItemID: workID,
				Phase:      WorkflowSagaCompensationPhaseIntent,
			},
		)
		progress.Phase = WorkflowSagaPhaseCompensationIntent
		intents = append(intents, WorkItemIntent{
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
		})
	}

	if len(intents) != 0 {
		return workflowTransition(state, nil, nil, intents)
	}
	if len(progress.Active) == 0 && len(progress.Remaining) == 0 {
		return i.finishSaga(frame, program, state), nil
	}

	if len(progress.Active) != 0 {
		return workflowTransition(state, nil, nil, nil)
	}

	return workflowFailure(
		state,
		"workflow_compensation_deadlock",
		"workflow compensation has no dependency-eligible effect",
	)
}

func workflowSagaNextEligibleEffect(
	execution Execution,
	progress WorkflowSagaProgress,
) (EffectRecordID, bool, error) {
	for _, effectID := range progress.Remaining {
		candidate := effectRecordByID(&execution, effectID)
		if candidate == nil {
			return "", false, errors.New("workflow compensation effect record is unavailable")
		}
		if candidate.State != CompensationStatePending {
			continue
		}

		blocked := false
		for _, other := range execution.Effects {
			if other.SagaID != progress.ID ||
				other.ID == candidate.ID ||
				!slices.Contains(other.Dependencies, candidate.OriginalWorkItemID) {
				continue
			}
			switch other.State {
			case CompensationStatePending, CompensationStateDispatching:
				blocked = true
			default:
			}
			if blocked {
				break
			}
		}
		if !blocked {
			return effectID, true, nil
		}
	}
	return "", false, nil
}

func removeWorkflowSagaEffect(
	values []EffectRecordID,
	want EffectRecordID,
) []EffectRecordID {
	output := make([]EffectRecordID, 0, len(values))
	for _, value := range values {
		if value != want {
			output = append(output, value)
		}
	}
	return output
}

func workflowSagaActiveIndexByWorkItem(
	values []WorkflowSagaActiveCompensation,
	workItemID WorkItemID,
) int {
	for index := range values {
		if values[index].WorkItemID == workItemID {
			return index
		}
	}
	return -1
}

func workflowSagaActiveIndexByChild(
	values []WorkflowSagaActiveCompensation,
	childPath string,
) int {
	for index := range values {
		if values[index].ChildPath == childPath {
			return index
		}
	}
	return -1
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
	activeIndex := workflowSagaActiveIndexByWorkItem(
		progress.Active,
		workID,
	)
	if activeIndex < 0 ||
		progress.Active[activeIndex].Phase != WorkflowSagaCompensationPhaseIntent {
		return ProgramTransition{}, false, nil
	}
	active := progress.Active[activeIndex]
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
			active,
			*cause,
		)
	}

	record := effectRecordByID(&execution, active.EffectID)
	if record == nil || record.Compensation == nil {
		return ProgramTransition{}, true, errors.New(
			"workflow compensation block is unavailable",
		)
	}

	completed := MustCanonicalValue(record.Completed)
	scope := WorkflowScopeContext{
		Completed: &completed,
		Cleanup:   true,
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
		string(active.EffectID) + "/child"
	progress.Active[activeIndex].ChildPath = childPath
	progress.Active[activeIndex].Phase = WorkflowSagaCompensationPhaseChild
	progress.Phase = WorkflowSagaPhaseCompensationChild

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

	if progress.Phase == WorkflowSagaPhaseBody {
		child := workflowTerminalChildFrame(
			execution,
			frame.ID,
			progress.BodyPath,
		)
		if child == nil || child.ID != childID || !child.Status.Terminal() {
			return ProgramTransition{}, false, nil
		}

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

	activeIndex := workflowSagaActiveIndexByChild(
		progress.Active,
		workflowTerminalChildPath(execution, childID),
	)
	if activeIndex < 0 {
		return ProgramTransition{}, false, nil
	}
	active := progress.Active[activeIndex]
	child := workflowTerminalChildFrame(
		execution,
		frame.ID,
		active.ChildPath,
	)
	if child == nil || child.ID != childID || !child.Status.Terminal() {
		return ProgramTransition{}, false, nil
	}

	result, failure := workflowFrameResult(*child)
	if failure != nil {
		return i.settleSagaCompensationFailure(
			frame,
			execution,
			program,
			state,
			active,
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
			active,
			SanitizedError{
				Code:     "compensation_control_invalid",
				Category: ErrorCategoryValidation,
				Message:  "compensation block returned unsupported control flow",
			},
		)
	}

	record := effectRecordByID(&execution, active.EffectID)
	if record == nil {
		return ProgramTransition{}, true, errors.New(
			"workflow compensation effect record is unavailable",
		)
	}
	record.State = CompensationStateCompleted
	record.CompensationFrameID = child.ID
	record.Receipt = compensationReceipt(record.ID, child.ID, state.LogicalTime)
	record.Error = nil
	settlement := CompensationSettlement{
		EffectID:   active.EffectID,
		WorkItemID: active.WorkItemID,
		FrameID:    child.ID,
		State:      CompensationStateCompleted,
		Receipt:    cloneReconciliationReceipt(record.Receipt),
	}

	progress.Active = append(
		progress.Active[:activeIndex],
		progress.Active[activeIndex+1:]...,
	)
	progress.Phase = workflowSagaCompensationPhase(progress.Active)

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
		workflowReleasePendingClaims(&state, pending)
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
	if progress.PrimaryStatus != "" {
		transition, err := i.driveSaga(frame, execution, program, state)
		if err != nil {
			return ProgramTransition{}, true, err
		}
		return transition, true, nil
	}
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
	active WorkflowSagaActiveCompensation,
	failure SanitizedError,
) (ProgramTransition, bool, error) {
	if state.Pending == nil || state.Pending.Saga == nil {
		return ProgramTransition{}, true, errors.New("workflow saga state is unavailable")
	}
	progress := state.Pending.Saga
	record := effectRecordByID(&execution, active.EffectID)
	if record == nil {
		return ProgramTransition{}, true, errors.New(
			"workflow compensation effect record is unavailable",
		)
	}
	stateValue := CompensationStateFailed
	if failure.Category == ErrorCategoryUncertainty {
		stateValue = CompensationStateUncertain
	}
	record.State = stateValue
	record.CompensationWorkItemID = active.WorkItemID
	record.CompensationFrameID = ""
	record.Error = cloneWorkflowSanitizedError(&failure)
	progress.Causes = append(
		progress.Causes,
		compensationFailureCause(*record, &failure),
	)
	settlement := CompensationSettlement{
		EffectID:   active.EffectID,
		WorkItemID: active.WorkItemID,
		State:      stateValue,
		Error:      cloneWorkflowSanitizedError(&failure),
	}
	progress.Active = append(
		progress.Active[:workflowSagaActiveIndexByWorkItem(
			progress.Active,
			active.WorkItemID,
		)],
		progress.Active[workflowSagaActiveIndexByWorkItem(
			progress.Active,
			active.WorkItemID,
		)+1:]...,
	)

	if progress.CompensationPolicy == CompensationErrorPolicyStop {
		progress.StopRequested = true
		progress.Remaining = nil
	}
	progress.Phase = workflowSagaCompensationPhase(progress.Active)
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

func workflowSagaCompensationPhase(
	active []WorkflowSagaActiveCompensation,
) WorkflowSagaPhase {
	for _, value := range active {
		if value.Phase == WorkflowSagaCompensationPhaseChild {
			return WorkflowSagaPhaseCompensationChild
		}
	}
	return WorkflowSagaPhaseCompensationIntent
}

func workflowTerminalChildPath(
	execution Execution,
	childID ProgramFrameID,
) string {
	for _, frame := range execution.ProgramFrames {
		if frame.ID == childID {
			return frame.Path
		}
	}
	return ""
}

func workflowProgramRunsCleanup(
	program ProgramSnapshot,
) bool {
	if program.InterpreterID != WorkflowInterpreterID {
		return false
	}
	value, err := workflowProgramFromSnapshot(program)
	return err == nil && value.Scope.Cleanup
}

func workflowFrameRunsCleanup(frame ProgramFrame) bool {
	return workflowProgramRunsCleanup(frame.Program)
}
