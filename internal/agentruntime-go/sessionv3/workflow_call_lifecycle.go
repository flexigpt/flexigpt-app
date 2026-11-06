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
	if lifetime != WorkflowChildLifetimeAttached &&
		lifetime != WorkflowChildLifetimeIndependent {
		return workflowFailure(
			state,
			"workflow_start_lifetime_invalid",
			"workflow start has an invalid child lifetime",
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
		if !slices.Contains(contract.Modes, WorkflowCallModeStart) {
			return workflowFailure(
				state,
				"workflow_start_unavailable",
				"started calls require a start-compatible target implementation",
			)
		}
	}

	ownership, err := i.childOwnership(
		program,
		state,
		call,
		lifetime,
	)
	if err != nil {
		return workflowFailure(state, "workflow_start_ownership_invalid", err.Error())
	}

	frameID := ProgramFrameID(
		string(frame.ID) + ":" + path + ":started",
	)
	executionID := ExecutionID("")
	handle := CompletionHandle{FrameID: frameID}
	intentLifetime := ChildFrameLifetimeAttached
	if lifetime == WorkflowChildLifetimeIndependent {
		executionID = ExecutionID(string(frame.ID) + ":" + path + ":execution")
		handle = CompletionHandle{ExecutionID: executionID}
		intentLifetime = ChildFrameLifetimeIndependent
	}
	if err := handle.Validate(); err != nil {
		return ProgramTransition{}, err
	}

	childCall := call
	childCall.Mode = WorkflowCallModeInvoke
	childCall.Lifetime = ""
	childCall.Conversation = nil
	childCall.Ownership = nil
	childCall.OnCompletion = ""
	childCall.InternalLifecycleChild = true
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
		Target: targetName,
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
		ID:          frameID,
		ExecutionID: executionID,
		Path:        path + "/started",
		Program:     childProgram,
		Deadline:    cloneTime(frame.Deadline),
		Lifetime:    intentLifetime,
		Ownership:   &ownership,
	}}
	transition.ContinueParent = true
	return transition, nil
}

func (i *WorkflowInterpreter) childOwnership(
	program WorkflowProgram,
	state WorkflowState,
	call WorkflowCall,
	lifetime WorkflowChildLifetime,
) (ChildExecutionOwnership, error) {
	value := ChildExecutionOwnership{
		CancelWithParent: lifetime == WorkflowChildLifetimeAttached,
	}
	if call.Ownership == nil {
		return value, nil
	}

	owner, err := i.stringValue(call.Ownership.Owner, program, state)
	if err != nil {
		return ChildExecutionOwnership{}, err
	}
	value.Owner = owner
	value.CancelWithParent = call.Ownership.CancelWithParent
	value.Budget = call.Ownership.Budget
	if err := value.Budget.Validate(); err != nil {
		return ChildExecutionOwnership{}, err
	}
	return value, nil
}

func (i *WorkflowInterpreter) handoffCall(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	path string,
	targetName string,
	target WorkflowTarget,
	input CanonicalValue,
	contextValue CanonicalValue,
	call WorkflowCall,
) (ProgramTransition, error) {
	if target.Kind != WorkflowTargetAgent || target.Artifact == nil {
		return workflowFailure(
			state,
			"workflow_handoff_target_invalid",
			"workflow handoff requires an Agent target with an Artifact pin",
		)
	}
	if call.Conversation == nil {
		return workflowFailure(
			state,
			"workflow_handoff_invalid",
			"workflow handoff requires a conversation",
		)
	}

	conversationID, err := i.stringValue(*call.Conversation, program, state)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_handoff_invalid",
			"workflow handoff conversation must resolve to a string",
		)
	}
	ownership, err := i.childOwnership(
		program,
		state,
		call,
		WorkflowChildLifetimeAttached,
	)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_handoff_ownership_invalid",
			err.Error(),
		)
	}

	childProgram, err := NewWorkflowAgentLoopProgram(
		SessionDefaults{
			Agent: workflowSelectionForPin(*target.Artifact),
		},
		LLMInvocationPlan{
			Input:   input,
			Context: contextValue,
		},
	)
	if err != nil {
		return workflowFailure(state, "workflow_handoff_prepare_failed", err.Error())
	}

	next := state
	next.PendingPath = path
	next.PendingTarget = targetName
	nextState, err := encodeWorkflowProgramState(next)
	if err != nil {
		return ProgramTransition{}, err
	}

	handoffID := string(frame.ID) + ":" + path + ":handoff"
	return ProgramTransition{
		State: nextState,
		Frames: []ProgramFrameIntent{{
			ID:        ProgramFrameID(string(frame.ID) + ":" + path + ":handoff"),
			Path:      path + "/handoff",
			Program:   childProgram,
			Deadline:  cloneTime(state.Deadline),
			Lifetime:  ChildFrameLifetimeAttached,
			Ownership: &ownership,
			Handoff: &ConversationHandoffIntent{
				ID:                  handoffID,
				ConversationID:      conversationID,
				Owner:               ownership.Owner,
				TargetAgent:         *target.Artifact,
				RestoreOnCompletion: call.OnCompletion != "retain",
			},
		}},
	}, nil
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
	if handle.FrameID == "" && handle.ExecutionID == "" {
		return workflowFailure(
			state,
			"workflow_cancel_invalid",
			"workflow cancel requires a child frame or Execution handle",
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
	if handle.FrameID != "" {
		transition.CancelChildFrameIDs = []ProgramFrameID{handle.FrameID}
	}
	if handle.ExecutionID != "" {
		transition.CancelChildExecutionIDs = []ExecutionID{
			handle.ExecutionID,
		}
	}
	transition.ContinueParent = true
	return transition, nil
}
