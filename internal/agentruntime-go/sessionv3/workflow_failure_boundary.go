package sessionv3

import (
	"errors"
	"fmt"
)

func workflowSettledFailure(
	program WorkflowProgram,
	state WorkflowState,
	item WorkItem,
) *SanitizedError {
	failure := item.Error
	if failure == nil && item.Output != nil && item.Output.Tool != nil &&
		item.Output.Tool.Kind != ToolOutputKindSuccess {
		failure = &SanitizedError{
			Code:     "workflow_tool_failed",
			Category: ErrorCategoryExecution,
			Message: firstNonEmpty(
				item.Output.Tool.Error,
				"workflow tool returned an error result",
			),
		}
	}
	if failure == nil && item.State != WorkItemStateCompleted {
		failure = &SanitizedError{
			Code:     "workflow_work_failed",
			Category: ErrorCategoryExecution,
			Message:  "workflow work did not complete successfully",
		}
	}
	if failure == nil {
		return nil
	}

	copyValue := *failure
	copyValue.StepPath = item.ProgramPath
	copyValue.Details = failure.Details.Clone()
	failure = &copyValue
	failure.Effect = item.EffectState

	step, err := workflowStepForPath(program.Root, item.ProgramPath)
	if err != nil || step.Call == nil {
		return failure
	}
	target, found := program.Targets[state.PendingTarget]
	if !found {
		if state.PendingTarget == "" {
			return failure
		}
		return &SanitizedError{
			Code:     "workflow_selected_target_missing",
			Category: ErrorCategoryValidation,
			Message:  "workflow call has no persisted selected target",
			Causes:   []SanitizedError{*failure},
		}
	}

	for _, contract := range []*WorkflowCallableInterface{
		target.Contract,
		step.Call.Contract,
		step.Call.SetContract,
	} {
		if contract == nil || len(contract.ErrorSchema) == 0 {
			continue
		}
		value, encodeErr := canonicalWorkflowValue(failure)
		if encodeErr != nil {
			err = encodeErr
		} else {
			err = validateWorkflowSchema(contract.ErrorSchema, value)
		}
		if err != nil {
			return &SanitizedError{
				Code:     "workflow_error_schema_invalid",
				Category: ErrorCategoryValidation,
				Message:  fmt.Sprintf("validate workflow error: %v", err),
				StepPath: item.ProgramPath,
				Causes:   []SanitizedError{*failure},
			}
		}
	}
	return failure
}

func workflowFailureTransition(
	state WorkflowState,
	failure *SanitizedError,
) (ProgramTransition, error) {
	if failure == nil {
		return ProgramTransition{}, errors.New("workflow failure is missing")
	}
	status := ProgramFrameStateFailed
	transition, err := workflowTransition(state, &status, nil, nil)
	if err != nil {
		return ProgramTransition{}, err
	}
	transition.Error = failure
	return transition, nil
}
