package sessionv3

import (
	"errors"
	"fmt"
)

func (o WorkflowStepOutcome) Validate() error {
	if err := validateID("workflow step outcome ID", o.ID); err != nil {
		return err
	}
	if !o.Output.Valid() {
		return errors.New("workflow step outcome output is invalid")
	}
	if err := o.Effect.Validate(); err != nil {
		return err
	}
	if o.Receipt != nil {
		if err := o.Receipt.Validate(); err != nil {
			return err
		}
	}
	if err := validateReconciliationError(o.Error); err != nil {
		return err
	}

	switch o.Status {
	case WorkflowStepOutcomeSucceeded:
		if len(o.Output) == 0 || o.Error != nil {
			return errors.New("successful workflow step outcome is invalid")
		}
	case WorkflowStepOutcomeSkipped:
		if len(o.Output) != 0 || o.Error != nil {
			return errors.New("skipped workflow step outcome is invalid")
		}
	case WorkflowStepOutcomeCancelled:
		if len(o.Output) != 0 {
			return errors.New("cancelled workflow step outcome has output")
		}
	case WorkflowStepOutcomeFailed, WorkflowStepOutcomeUncertain:
		if o.Error == nil {
			return errors.New("failed or uncertain workflow step outcome has no error")
		}
	default:
		return errors.New("workflow step outcome status is invalid")
	}
	return nil
}

func cloneWorkflowStepOutcome(
	value WorkflowStepOutcome,
) WorkflowStepOutcome {
	output := value
	output.Output = value.Output.Clone()
	output.Receipt = cloneReconciliationReceipt(value.Receipt)
	output.Error = cloneWorkflowSanitizedError(value.Error)
	return output
}

func (s *WorkflowState) recordTerminalOutcome(
	outcome WorkflowStepOutcome,
) error {
	if s == nil {
		return errors.New("workflow state is unavailable")
	}
	if err := outcome.Validate(); err != nil {
		return err
	}

	for index := range s.Outcomes {
		current := s.Outcomes[index]
		if current.ID != outcome.ID {
			continue
		}
		if current.Status == WorkflowStepOutcomeUncertain &&
			outcome.Status == WorkflowStepOutcomeSucceeded {
			s.Outcomes[index] = cloneWorkflowStepOutcome(outcome)
		}
		return nil
	}
	s.Outcomes = append(s.Outcomes, cloneWorkflowStepOutcome(outcome))
	return nil
}

func (i *WorkflowInterpreter) recordTerminalWorkflowStepOutcome(
	program WorkflowProgram,
	before WorkflowState,
	transition ProgramTransition,
) (ProgramTransition, error) {
	if transition.Status == nil ||
		(*transition.Status != ProgramFrameStateFailed &&
			*transition.Status != ProgramFrameStateCancelled) {
		return transition, nil
	}
	if before.PendingPath == "" {
		return transition, nil
	}

	step, err := workflowStepForPath(program.Root, before.PendingPath)
	if err != nil {
		return transition, nil
	}
	state, err := DecodeCanonicalValue[WorkflowState](transition.State.Data)
	if err != nil {
		return ProgramTransition{}, err
	}

	status := WorkflowStepOutcomeFailed
	if *transition.Status == ProgramFrameStateCancelled {
		status = WorkflowStepOutcomeCancelled
	}
	target := before.PendingTarget
	if target == "" {
		target = state.PendingTarget
	}
	failure := cloneWorkflowSanitizedError(transition.Error)
	if failure == nil {
		failure = workflowTerminalFailure(*transition.Status, nil)
	}

	outcome := WorkflowStepOutcome{
		ID:     step.ID,
		Status: status,
		Target: target,
		Error:  failure,
	}
	if err := state.recordTerminalOutcome(outcome); err != nil {
		return ProgramTransition{}, fmt.Errorf(
			"record workflow terminal step outcome: %w",
			err,
		)
	}
	next, err := encodeWorkflowProgramState(state)
	if err != nil {
		return ProgramTransition{}, err
	}
	transition.State = next
	return transition, nil
}

func (r *Runtime) recordUncertainWorkflowStepOutcomeLocked(
	execution *Execution,
	item *WorkItem,
) error {
	if execution == nil || item == nil {
		return errors.New("workflow uncertain outcome owner is unavailable")
	}

	frame := mutableFrame(execution, item.FrameID)
	if frame == nil || frame.Program.InterpreterID != WorkflowInterpreterID {
		return nil
	}

	program, state, err := decodeWorkflowProgram(*frame)
	if err != nil {
		return err
	}
	if state.PendingPath == "" || state.PendingPath != item.ProgramPath {
		return nil
	}

	step, err := workflowStepForPath(program.Root, state.PendingPath)
	if err != nil {
		return err
	}

	failure := cloneWorkflowSanitizedError(item.Error)
	if failure == nil {
		failure = &SanitizedError{
			Code:     "workflow_work_uncertain",
			Category: ErrorCategoryUncertainty,
			Effect:   item.EffectState,
			Message:  "workflow external work outcome is uncertain",
		}
	}

	var receipt *ExternalReceipt
	if item.Output != nil {
		receipt = cloneReconciliationReceipt(item.Output.Receipt)
	}

	if err := state.recordTerminalOutcome(WorkflowStepOutcome{
		ID:      step.ID,
		Status:  WorkflowStepOutcomeUncertain,
		Target:  state.PendingTarget,
		Receipt: receipt,
		Error:   failure,
		Effect:  item.EffectState,
	}); err != nil {
		return err
	}

	next, err := encodeWorkflowProgramState(state)
	if err != nil {
		return err
	}
	frame.State = next
	frame.UpdatedAt = r.now()
	return nil
}
