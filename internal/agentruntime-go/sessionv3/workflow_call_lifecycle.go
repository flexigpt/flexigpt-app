package sessionv3

import (
	"slices"
)

type WorkflowChildLifetime string

const (
	WorkflowChildLifetimeAttached    WorkflowChildLifetime = "attached"
	WorkflowChildLifetimeIndependent WorkflowChildLifetime = "independent"
)

type WorkflowStartOutput struct {
	Handle   CompletionHandle      `json:"handle"`
	Lifetime WorkflowChildLifetime `json:"lifetime"`
}

type WorkflowCancelOutput struct {
	Handle    CompletionHandle `json:"handle"`
	Requested bool             `json:"requested"`
	Reason    CanonicalValue   `json:"reason,omitempty"`
}

func (i *WorkflowInterpreter) startCall(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	path string,
	step WorkflowStep,
	targetName string,
	target WorkflowTarget,
	input CanonicalValue,
	contextValue CanonicalValue,
	call WorkflowCall,
) (ProgramTransition, error) {
	lifetime := call.Lifetime
	if lifetime == "" {
		lifetime = WorkflowChildLifetimeAttached
	}
	if lifetime != WorkflowChildLifetimeAttached {
		return workflowFailure(
			state,
			"workflow_start_lifetime_unsupported",
			"independent workflow child lifetime requires a separate execution",
		)
	}

	for _, contract := range []*WorkflowCallableInterface{
		target.Contract,
		call.Contract,
		call.SetContract,
	} {
		if contract == nil || len(contract.Modes) == 0 {
			continue
		}
		if !slices.Contains(contract.Modes, WorkflowCallModeInvoke) {
			return workflowFailure(
				state,
				"workflow_start_invoke_unavailable",
				"started calls require an invoke-compatible target implementation",
			)
		}
	}

	frameID := ProgramFrameID(
		string(frame.ID) + ":" + path + ":started",
	)
	handle := CompletionHandle{FrameID: frameID}
	if err := handle.Validate(); err != nil {
		return ProgramTransition{}, err
	}

	childCall := call
	childCall.Mode = WorkflowCallModeInvoke
	childCall.Lifetime = ""
	childCall.Target = WorkflowValue{
		Kind:    WorkflowValueLiteral,
		Literal: MustCanonicalValue(targetName),
	}
	childCall.Input = WorkflowValue{
		Kind:    WorkflowValueLiteral,
		Literal: input.Clone(),
	}
	childCall.Context = WorkflowValue{
		Kind:    WorkflowValueLiteral,
		Literal: contextValue.Clone(),
	}
	childCall.AllowedTargets = append(
		[]string(nil),
		call.AllowedTargets...,
	)

	childStep := WorkflowStep{
		ID:           "invoke",
		Timeout:      step.Timeout,
		Retry:        cloneWorkflowRetryPolicy(step.Retry),
		OutputSchema: step.OutputSchema.Clone(),
		Call:         &childCall,
	}
	childBlock := WorkflowBlock{
		Steps: []WorkflowStep{childStep},
		Output: WorkflowValue{
			Kind:       WorkflowValueExpression,
			Expression: "steps.invoke.output",
		},
	}

	childProgram, err := i.childBlockProgram(
		frame,
		program,
		state,
		childBlock,
		false,
		nil,
		nil,
		nil,
	)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_start_prepare_failed",
			err.Error(),
		)
	}

	output := MustCanonicalValue(WorkflowStartOutput{
		Handle:   handle,
		Lifetime: lifetime,
	})
	if err := validateWorkflowSchema(step.OutputSchema, output); err != nil {
		return workflowFailure(
			state,
			"workflow_start_output_invalid",
			err.Error(),
		)
	}
	if err := state.recordOutcome(WorkflowStepOutcome{
		ID:     step.ID,
		Status: WorkflowStepOutcomeSucceeded,
		Output: output,
	}); err != nil {
		return workflowFailure(
			state,
			"workflow_state_invalid",
			err.Error(),
		)
	}

	state.Index++
	state.PendingPath = ""
	state.PendingTarget = ""
	state.Deadline = nil

	transition, err := workflowTransition(
		state,
		nil,
		nil,
		nil,
	)
	if err != nil {
		return ProgramTransition{}, err
	}
	transition.Frames = []ProgramFrameIntent{{
		ID:       frameID,
		Path:     path + "/started",
		Program:  childProgram,
		Deadline: cloneTime(frame.Deadline),
		Lifetime: ChildFrameLifetimeAttached,
	}}
	transition.ContinueParent = true
	return transition, nil
}

func (i *WorkflowInterpreter) cancelCall(
	program WorkflowProgram,
	state WorkflowState,
	path string,
	step WorkflowStep,
	spec WorkflowCancel,
) (ProgramTransition, error) {
	handleValue, err := i.evaluate(spec.Handle, program, state)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_cancel_invalid",
			err.Error(),
		)
	}

	handle, err := DecodeCanonicalValue[CompletionHandle](handleValue)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_cancel_invalid",
			"workflow cancel handle is invalid",
		)
	}
	if err := handle.Validate(); err != nil {
		return workflowFailure(
			state,
			"workflow_cancel_invalid",
			err.Error(),
		)
	}
	if handle.FrameID == "" {
		return workflowFailure(
			state,
			"workflow_cancel_invalid",
			"workflow cancel currently requires a child frame handle",
		)
	}

	var reason CanonicalValue
	if spec.Reason != nil {
		reason, err = i.evaluate(*spec.Reason, program, state)
		if err != nil {
			return workflowFailure(
				state,
				"workflow_cancel_invalid",
				err.Error(),
			)
		}
	}

	output := MustCanonicalValue(WorkflowCancelOutput{
		Handle:    handle,
		Requested: true,
		Reason:    reason,
	})
	if err := validateWorkflowSchema(step.OutputSchema, output); err != nil {
		return workflowFailure(
			state,
			"workflow_cancel_output_invalid",
			err.Error(),
		)
	}
	if err := state.recordOutcome(WorkflowStepOutcome{
		ID:     step.ID,
		Status: WorkflowStepOutcomeSucceeded,
		Output: output,
	}); err != nil {
		return workflowFailure(
			state,
			"workflow_state_invalid",
			err.Error(),
		)
	}

	state.Index++
	state.PendingPath = ""
	state.PendingTarget = ""
	state.Deadline = nil

	transition, err := workflowTransition(
		state,
		nil,
		nil,
		nil,
	)
	if err != nil {
		return ProgramTransition{}, err
	}
	transition.CancelChildFrameIDs = []ProgramFrameID{
		handle.FrameID,
	}
	transition.ContinueParent = true
	return transition, nil
}
