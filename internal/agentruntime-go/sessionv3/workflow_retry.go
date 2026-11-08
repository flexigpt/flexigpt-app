package sessionv3

import (
	"errors"
	"strconv"
)

func (i *WorkflowInterpreter) startWorkflowRetry(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	path string,
) (ProgramTransition, error) {
	if step.Retry == nil || step.Retry.MaxAttempts <= 1 {
		return ProgramTransition{}, errors.New(
			"workflow retry boundary requires a retry policy",
		)
	}

	childStep := step
	childStep.When = nil
	childStep.Retry = nil
	childStep.Claims = nil

	childBlock := WorkflowBlock{
		Steps: []WorkflowStep{childStep},
		Output: WorkflowValue{
			Kind: WorkflowValueExpression,
			Expression: "steps[" +
				strconv.Quote(step.ID) +
				"].output",
		},
	}

	transition, err := i.startSingleChild(
		frame,
		program,
		state,
		step,
		path,
		WorkflowPendingScope,
		"retry",
		childBlock,
		true,
		nil,
		nil,
		nil,
	)
	if err != nil {
		return transition, err
	}
	if len(transition.Frames) != 1 {
		return ProgramTransition{}, errors.New(
			"workflow retry boundary did not create one child frame",
		)
	}

	retryProgram, err := NewRetryBoundaryProgram(
		transition.Frames[0].Program,
		*cloneWorkflowRetryPolicy(step.Retry),
	)
	if err != nil {
		return ProgramTransition{}, err
	}
	transition.Frames[0].Program = retryProgram
	return transition, nil
}
