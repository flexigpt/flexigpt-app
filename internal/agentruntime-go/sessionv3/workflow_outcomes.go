package sessionv3

import (
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
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
		if current.Status == WorkflowStepOutcomeUncertain {
			switch outcome.Status {
			case WorkflowStepOutcomeSucceeded:
				s.Outcomes[index] = cloneWorkflowStepOutcome(outcome)
			case WorkflowStepOutcomeFailed,
				WorkflowStepOutcomeCancelled:
				// Reconciliation can prove confirmed non-application after
				// an earlier uncertain external result. Preserve uncertainty
				// for all other cancellation and failure paths.
				if outcome.Error != nil &&
					outcome.Error.Effect ==
						EffectStateConfirmedNotApplied {
					s.Outcomes[index] = cloneWorkflowStepOutcome(outcome)
				}
			default:
			}
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
	execution Execution,
	frame ProgramFrame,
	event ProgramEvent,
) (ProgramTransition, error) {
	if transition.Status == nil || !transition.Status.Terminal() {
		return transition, nil
	}
	state, err := jsonutil.DecodeCanonicalValue[WorkflowState](transition.State.Data)
	if err != nil {
		return ProgramTransition{}, err
	}
	path := firstText(state.PendingPath, before.PendingPath)
	if path == "" {
		return transition, nil
	}
	step, err := workflowStepForPath(program.Root, path)
	if err != nil {
		return transition, nil
	}

	for _, current := range state.Outcomes {
		if current.ID == step.ID &&
			current.Status != WorkflowStepOutcomeUncertain {
			return transition, nil
		}
	}

	outcome := WorkflowStepOutcome{
		ID:              step.ID,
		Target:          firstText(state.PendingTarget, before.PendingTarget),
		DeclarationPath: firstText(state.PendingDeclarationPath, step.DeclarationPath),
	}
	var settled *WorkItem
	if event.WorkItemID != "" {
		item := executionWorkItem(execution, event.WorkItemID)
		if item != nil &&
			item.FrameID == frame.ID &&
			item.ProgramPath == path {
			settled = item
			outcome.Receipt = cloneReconciliationReceipt(
				workItemOutputReceipt(item),
			)
			outcome.Effect = item.EffectState
			if value, valueErr := workflowWorkValue(*item); valueErr == nil {
				outcome.Output = value
			}
		}
	}
	switch *transition.Status {
	case ProgramFrameStateCompleted:
		value := transition.Output.Clone()
		if program.Scope.Child {
			if result, decodeErr := jsonutil.DecodeCanonicalValue[WorkflowBlockResult](value); decodeErr == nil {
				value = result.Value.Clone()
			}
		}
		if len(value) != 0 && value.Valid() {
			outcome.Output = value
		}
		if len(outcome.Output) == 0 || !outcome.Output.Valid() {
			return transition, nil
		}
		outcome.Status = WorkflowStepOutcomeSucceeded

	case ProgramFrameStateCancelled:
		outcome.Status = WorkflowStepOutcomeCancelled
		outcome.Output = nil
		outcome.Error = cloneWorkflowSanitizedError(transition.Error)
		if outcome.Error == nil && settled != nil {
			outcome.Error = cloneWorkflowSanitizedError(settled.Error)
		}
		if outcome.Error == nil {
			outcome.Error = workflowTerminalFailure(
				ProgramFrameStateCancelled,
				nil,
			)
		}
		if outcome.Effect == "" {
			outcome.Effect = outcome.Error.Effect
		}

	case ProgramFrameStateFailed:
		outcome.Status = WorkflowStepOutcomeFailed
		outcome.Error = cloneWorkflowSanitizedError(transition.Error)
		if outcome.Error == nil && settled != nil {
			outcome.Error = cloneWorkflowSanitizedError(settled.Error)
		}
		if outcome.Error == nil {
			outcome.Error = workflowTerminalFailure(
				ProgramFrameStateFailed,
				nil,
			)
		}
		if outcome.Effect == "" {
			outcome.Effect = outcome.Error.Effect
		}
	default:
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
		ID:              step.ID,
		Status:          WorkflowStepOutcomeUncertain,
		Target:          state.PendingTarget,
		Receipt:         receipt,
		Error:           failure,
		Effect:          item.EffectState,
		DeclarationPath: firstText(item.DeclarationPath, step.DeclarationPath),
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

func recordInlineWorkflowStepOutcome(
	state *WorkflowState,
	step WorkflowStep,
	status WorkflowStepOutcomeStatus,
	value jsonutil.CanonicalValue,
	failure *SanitizedError,
) error {
	if state == nil {
		return errors.New("workflow state is unavailable")
	}
	outcome := WorkflowStepOutcome{
		ID:              step.ID,
		Status:          status,
		Output:          value.Clone(),
		Error:           cloneWorkflowSanitizedError(failure),
		DeclarationPath: state.PendingDeclarationPath,
	}
	if failure != nil {
		outcome.Effect = failure.Effect
	}
	return state.recordTerminalOutcome(outcome)
}

func workItemOutputReceipt(item *WorkItem) *ExternalReceipt {
	if item == nil || item.Output == nil {
		return nil
	}
	return item.Output.Receipt
}
