package sessionv3

import (
	"errors"
	"strconv"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func (i *WorkflowInterpreter) startWorkflowRetry(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	path string,
) (pt ProgramTransition, err error) {
	if step.Retry == nil || step.Retry.MaxAttempts <= 1 {
		return ProgramTransition{}, errors.New(
			"workflow retry boundary requires a retry policy",
		)
	}

	childStep := step
	childStep.When = nil
	childStep.Retry = nil
	childStep.Claims = nil
	childStep, err = i.captureWorkflowRetryCall(
		program,
		state,
		childStep,
	)
	if err != nil {
		return workflowFailure(state, "workflow_retry_capture_failed", err.Error())
	}

	childBlock := WorkflowBlock{
		DeclarationPath: step.DeclarationPath,
		Steps:           []WorkflowStep{childStep},
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
	transition.Frames[0].DeclarationPath = state.PendingDeclarationPath
	return transition, nil
}

// captureWorkflowRetryCall freezes call values at the first logical retry
// boundary. Later attempts receive literal target, input, context, typed
// messages, typed instructions, and keyed-effect values.
func (i *WorkflowInterpreter) captureWorkflowRetryCall(
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
) (WorkflowStep, error) {
	if step.Call == nil {
		return step, nil
	}

	output := jsonutil.CloneJSONInValue(step)
	target, err := i.stringValue(output.Call.Target, program, state)
	if err != nil {
		return WorkflowStep{}, err
	}
	input, err := i.evaluate(output.Call.Input, program, state)
	if err != nil {
		return WorkflowStep{}, err
	}
	contextValue, err := i.evaluate(output.Call.Context, program, state)
	if err != nil {
		return WorkflowStep{}, err
	}

	output.Call.Target = WorkflowValue{
		Kind:    WorkflowValueLiteral,
		Literal: jsonutil.MustCanonicalValue(target),
	}
	output.Call.Input = WorkflowValue{
		Kind:    WorkflowValueLiteral,
		Literal: input.Clone(),
	}
	output.Call.Context = WorkflowValue{
		Kind:    WorkflowValueLiteral,
		Literal: contextValue.Clone(),
	}
	if output.Call.LLMInvocation != nil {
		invocation, invocationErr := retryCaptureLLMInvocation(
			program,
			state,
			*output.Call.LLMInvocation,
		)
		if invocationErr != nil {
			return WorkflowStep{}, invocationErr
		}
		output.Call.LLMInvocation = invocation
	}
	if output.Effect != nil && output.Effect.Key != nil {
		key, keyErr := i.evaluate(*output.Effect.Key, program, state)
		if keyErr != nil {
			return WorkflowStep{}, keyErr
		}
		output.Effect.Key = &WorkflowValue{
			Kind:    WorkflowValueLiteral,
			Literal: key.Clone(),
		}
	}
	return output, nil
}
