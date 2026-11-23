package sessionv3

import (
	"errors"
	"slices"
	"strconv"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func (i *WorkflowInterpreter) startWorkflowRetry(
	frame ProgramFrame,
	execution Execution,
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
	if step.Call != nil {
		if transition, waiting, err := i.beginTargetSetResolution(
			frame,
			program,
			&state,
			path,
			*step.Call,
		); err != nil {
			return workflowFailure(state, "workflow_target_set_invalid", err.Error())
		} else if waiting {
			return transition, nil
		}
	}

	childStep := step
	childStep.When = nil
	childStep.Retry = nil
	childStep.Claims = nil
	childStep, err = i.captureWorkflowRetryCall(
		frame,
		program,
		state,
		execution,
		path,
		childStep,
	)
	if err != nil {
		return workflowFailure(state, "workflow_retry_capture_failed", err.Error())
	}
	if step.Call != nil && step.Call.AllowedTargetSet != "" {
		childStep.Call.AllowedTargetSet = step.Call.AllowedTargetSet
		childStep.Call.AllowedTargets = append(
			[]string(nil),
			step.Call.AllowedTargets...,
		)
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
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	execution Execution,
	path string,
	step WorkflowStep,
) (WorkflowStep, error) {
	if step.Call == nil {
		return step, nil
	}

	output := jsonutil.CloneJSONInValue(step)
	targetName, err := i.stringValue(output.Call.Target, program, state)
	if err != nil {
		return WorkflowStep{}, err
	}
	allowedTargets, err := i.resolveWorkflowCallTargets(
		program,
		&state,
		*output.Call,
	)
	if err != nil {
		return WorkflowStep{}, err
	}
	if len(allowedTargets) != 0 &&
		!slices.Contains(allowedTargets, targetName) {
		return WorkflowStep{}, errors.New(
			"workflow target is outside its allowed target set",
		)
	}
	target, found := program.Targets[targetName]
	if !found {
		return WorkflowStep{}, errors.New("workflow target is unavailable")
	}
	input, err := i.evaluate(output.Call.Input, program, state)
	if err != nil {
		return WorkflowStep{}, err
	}
	contextValue, err := i.evaluate(output.Call.Context, program, state)
	if err != nil {
		return WorkflowStep{}, err
	}
	ports, err := i.workflowCallPortPlans(
		frame,
		program,
		state,
		path,
		*output.Call,
		target,
	)
	if err != nil {
		return WorkflowStep{}, err
	}

	output.Call.Target = WorkflowValue{
		Kind:    WorkflowValueLiteral,
		Literal: jsonutil.MustCanonicalValue(targetName),
	}
	output.Call.Input = WorkflowValue{
		Kind:    WorkflowValueLiteral,
		Literal: input.Clone(),
	}
	output.Call.Context = WorkflowValue{
		Kind:    WorkflowValueLiteral,
		Literal: contextValue.Clone(),
	}
	output.Call.PortInputs = capturedWorkflowCallPortInputs(
		execution,
		*output.Call,
		ports,
	)
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
