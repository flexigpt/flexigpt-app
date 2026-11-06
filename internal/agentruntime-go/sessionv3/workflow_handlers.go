package sessionv3

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type WorkflowErrorMatch struct {
	Categories   []ErrorCategory `json:"categories,omitempty"`
	Codes        []string        `json:"codes,omitempty"`
	Retryable    *bool           `json:"retryable,omitempty"`
	EffectStates []EffectState   `json:"effectStates,omitempty"`
	Schema       CanonicalValue  `json:"schema,omitempty"`
}

type WorkflowCatchHandler struct {
	Match WorkflowErrorMatch `json:"match"`
	Do    WorkflowBlock      `json:"do"`
}

type WorkflowTerminalPhase string

const (
	WorkflowTerminalPhaseCatch   WorkflowTerminalPhase = "catch"
	WorkflowTerminalPhaseFinally WorkflowTerminalPhase = "finally"
)

// WorkflowTerminalState persists the normal, failed, or cancelled result that
// is being completed through catch and finally behavior. It contains no
// handler closure, timer, client, or compiled expression.
type WorkflowTerminalState struct {
	Status     ProgramFrameState     `json:"status"`
	Exit       WorkflowExitKind      `json:"exit,omitempty"`
	Value      CanonicalValue        `json:"value,omitempty"`
	Error      *SanitizedError       `json:"error,omitempty"`
	Phase      WorkflowTerminalPhase `json:"phase,omitempty"`
	Handler    int                   `json:"handler,omitempty"`
	ChildPath  string                `json:"childPath,omitempty"`
	MergeState bool                  `json:"mergeState,omitempty"`
}

type WorkflowOutcomeContext struct {
	Status ProgramFrameState `json:"status"`
	Exit   WorkflowExitKind  `json:"exit,omitempty"`
	Value  CanonicalValue    `json:"value,omitempty"`
	Error  *SanitizedError   `json:"error,omitempty"`
}

func (m WorkflowErrorMatch) Validate() error {
	for _, category := range m.Categories {
		switch category {
		case ErrorCategoryValidation,
			ErrorCategoryRouting,
			ErrorCategoryExecution,
			ErrorCategoryTransient,
			ErrorCategoryTimeout,
			ErrorCategoryAuthorization,
			ErrorCategoryLimit,
			ErrorCategoryCancellation,
			ErrorCategoryUncertainty:
		default:
			return fmt.Errorf(
				"unsupported workflow error category %q",
				category,
			)
		}
	}
	for _, code := range m.Codes {
		if err := validateID("workflow error code", code); err != nil {
			return err
		}
	}
	for _, state := range m.EffectStates {
		if err := state.Validate(); err != nil {
			return err
		}
		if state == "" {
			return errors.New("workflow error match has an empty effect state")
		}
	}
	if !m.Schema.Valid() {
		return errors.New("workflow error match schema is invalid")
	}
	return nil
}

func (m WorkflowErrorMatch) Matches(value SanitizedError) bool {
	if len(m.Categories) != 0 {
		found := slices.Contains(m.Categories, value.Category)
		if !found {
			return false
		}
	}
	if len(m.Codes) != 0 {
		found := slices.Contains(m.Codes, value.Code)
		if !found {
			return false
		}
	}
	if m.Retryable != nil && value.Retryable != *m.Retryable {
		return false
	}
	if len(m.EffectStates) != 0 {
		found := slices.Contains(m.EffectStates, value.Effect)
		if !found {
			return false
		}
	}
	if len(m.Schema) == 0 {
		return true
	}
	encoded, err := canonicalWorkflowValue(value)
	if err != nil {
		return false
	}
	return validateWorkflowSchema(m.Schema, encoded) == nil
}

func (h WorkflowCatchHandler) Validate() error {
	if err := h.Match.Validate(); err != nil {
		return err
	}
	return h.Do.Validate()
}

func (s WorkflowTerminalState) Validate() error {
	switch s.Status {
	case ProgramFrameStateCompleted,
		ProgramFrameStateFailed,
		ProgramFrameStateCancelled:
	default:
		return errors.New("workflow terminal state has an invalid status")
	}
	if !s.Value.Valid() {
		return errors.New("workflow terminal state value is invalid")
	}
	if s.Error != nil {
		if !s.Error.Details.Valid() {
			return errors.New(
				"workflow terminal state error details are invalid",
			)
		}
		if err := s.Error.Effect.Validate(); err != nil {
			return err
		}
	}
	switch s.Phase {
	case "", WorkflowTerminalPhaseCatch, WorkflowTerminalPhaseFinally:
	default:
		return errors.New("workflow terminal state phase is invalid")
	}
	if s.Phase != "" && strings.TrimSpace(s.ChildPath) == "" {
		return errors.New("workflow terminal state has no child path")
	}
	if s.Status == ProgramFrameStateFailed && s.Error == nil {
		return errors.New("failed workflow terminal state has no error")
	}
	return nil
}

func cloneWorkflowSanitizedError(
	value *SanitizedError,
) *SanitizedError {
	if value == nil {
		return nil
	}
	output := *value
	output.Details = value.Details.Clone()
	output.Causes = make([]SanitizedError, len(value.Causes))
	for index := range value.Causes {
		output.Causes[index] = cloneWorkflowErrorCause(
			value.Causes[index],
		)
	}
	return &output
}

func cloneWorkflowErrorCause(value SanitizedError) SanitizedError {
	output := value
	output.Details = value.Details.Clone()
	output.Causes = make([]SanitizedError, len(value.Causes))
	for index := range value.Causes {
		output.Causes[index] = cloneWorkflowErrorCause(
			value.Causes[index],
		)
	}
	return output
}

func appendWorkflowErrorCause(
	primary *SanitizedError,
	cause SanitizedError,
) *SanitizedError {
	if primary == nil {
		copyValue := cloneWorkflowErrorCause(cause)
		return &copyValue
	}
	output := cloneWorkflowSanitizedError(primary)
	output.Causes = append(
		output.Causes,
		cloneWorkflowErrorCause(cause),
	)
	return output
}

func workflowTerminalFailure(
	status ProgramFrameState,
	value *SanitizedError,
) *SanitizedError {
	if value != nil {
		return cloneWorkflowSanitizedError(value)
	}
	switch status {
	case ProgramFrameStateCancelled:
		return &SanitizedError{
			Code:     "workflow_cancelled",
			Category: ErrorCategoryCancellation,
			Message:  "workflow execution was cancelled",
		}
	default:
		return &SanitizedError{
			Code:     "workflow_failed",
			Category: ErrorCategoryExecution,
			Message:  "workflow execution failed without an error",
		}
	}
}

func workflowTerminalOutcomeValue(
	value WorkflowTerminalState,
) CanonicalValue {
	return MustCanonicalValue(WorkflowOutcomeContext{
		Status: value.Status,
		Exit:   value.Exit,
		Value:  value.Value.Clone(),
		Error:  cloneWorkflowSanitizedError(value.Error),
	})
}

func (i *WorkflowInterpreter) prepareWorkflowTerminal(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	transition ProgramTransition,
) (ProgramTransition, error) {
	if transition.Status == nil || !transition.Status.Terminal() {
		return transition, nil
	}

	state, err := DecodeCanonicalValue[WorkflowState](
		transition.State.Data,
	)
	if err != nil {
		return ProgramTransition{}, err
	}
	if state.Terminal != nil {
		return transition, nil
	}

	terminal := WorkflowTerminalState{
		Status: *transition.Status,
		Exit:   state.Exit,
		Value:  transition.Output.Clone(),
		Error:  cloneWorkflowSanitizedError(transition.Error),
	}
	if terminal.Status != ProgramFrameStateCompleted {
		terminal.Error = workflowTerminalFailure(
			terminal.Status,
			terminal.Error,
		)
	}
	if terminal.Error != nil &&
		terminal.Error.StepPath == "" {
		terminal.Error.StepPath = frame.Path
	}
	state.Terminal = &terminal

	if terminal.Status == ProgramFrameStateFailed {
		for index, handler := range program.Root.Catch {
			if terminal.Error == nil ||
				!handler.Match.Matches(*terminal.Error) {
				continue
			}
			state.Terminal.Phase = WorkflowTerminalPhaseCatch
			state.Terminal.Handler = index
			state.Terminal.MergeState = handler.Do.State == nil
			return i.startWorkflowTerminalChild(
				frame,
				program,
				state,
				handler.Do,
				"catch-"+strconv.Itoa(index),
			)
		}
	}

	return i.startWorkflowFinally(frame, execution, program, state)
}

func (i *WorkflowInterpreter) startWorkflowFinally(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
) (ProgramTransition, error) {
	_ = execution
	if state.Terminal == nil {
		return ProgramTransition{}, errors.New(
			"workflow terminal state is unavailable",
		)
	}
	if program.Root.Finally == nil {
		return i.finishWorkflowTerminal(program, state)
	}

	state.Terminal.Phase = WorkflowTerminalPhaseFinally
	state.Terminal.MergeState = program.Root.Finally.State == nil
	return i.startWorkflowTerminalChild(
		frame,
		program,
		state,
		*program.Root.Finally,
		"finally",
	)
}

func (i *WorkflowInterpreter) startWorkflowTerminalChild(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	block WorkflowBlock,
	name string,
) (ProgramTransition, error) {
	if state.Terminal == nil {
		return ProgramTransition{}, errors.New(
			"workflow terminal state is unavailable",
		)
	}

	path := "terminal/" + name
	snapshot, err := i.workflowTerminalChildProgram(
		frame,
		program,
		state,
		block,
	)
	if err != nil {
		return ProgramTransition{}, err
	}
	state.Terminal.ChildPath = path

	return workflowStructuredTransition(
		state,
		[]ProgramFrameIntent{{
			Path:     path,
			Program:  snapshot,
			Deadline: cloneTime(frame.Deadline),
		}},
	)
}

func (i *WorkflowInterpreter) workflowTerminalChildProgram(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	block WorkflowBlock,
) (ProgramSnapshot, error) {
	if state.Terminal == nil {
		return ProgramSnapshot{}, errors.New(
			"workflow terminal state is unavailable",
		)
	}

	snapshot, err := i.childBlockProgram(
		frame,
		program,
		state,
		block,
		state.Terminal.MergeState,
		nil,
		nil,
		nil,
	)
	if err != nil {
		return ProgramSnapshot{}, err
	}

	child, err := workflowProgramFromSnapshot(snapshot)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	if state.Terminal.Error != nil {
		errorValue, encodeErr := canonicalWorkflowValue(
			state.Terminal.Error,
		)
		if encodeErr != nil {
			return ProgramSnapshot{}, encodeErr
		}
		child.Scope.Error = &errorValue
	}
	outcomeValue := workflowTerminalOutcomeValue(*state.Terminal)
	child.Scope.Outcome = &outcomeValue
	child.Scope.Event = nil

	payload, err := canonicalWorkflowValue(child)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	snapshot.Payload = payload
	return snapshot, nil
}

func (i *WorkflowInterpreter) advanceWorkflowTerminal(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	event ProgramEvent,
) (ProgramTransition, error) {
	if state.Terminal == nil {
		return ProgramTransition{}, errors.New(
			"workflow terminal state is unavailable",
		)
	}

	if event.Kind == ProgramEventCancelled &&
		state.Terminal.Phase != WorkflowTerminalPhaseFinally {
		state.Terminal.Status = ProgramFrameStateCancelled
		state.Terminal.Error = workflowTerminalFailure(
			ProgramFrameStateCancelled,
			nil,
		)
		state.Terminal.Phase = ""
		state.Terminal.ChildPath = ""
		return i.startWorkflowFinally(frame, execution, program, state)
	}

	if event.Kind != ProgramEventChildSettled {
		return workflowTransition(state, nil, nil, nil)
	}

	child := workflowTerminalChildFrame(
		execution,
		frame.ID,
		state.Terminal.ChildPath,
	)
	if child == nil {
		return ProgramTransition{}, errors.New(
			"workflow terminal child frame is unavailable",
		)
	}
	if !child.Status.Terminal() {
		return workflowTransition(state, nil, nil, nil)
	}

	result, childErr := workflowFrameResult(*child)
	switch state.Terminal.Phase {
	case WorkflowTerminalPhaseCatch:
		return i.settleWorkflowCatch(
			frame,
			execution,
			program,
			state,
			result,
			childErr,
		)

	case WorkflowTerminalPhaseFinally:
		return i.settleWorkflowFinally(
			program,
			state,
			childErr,
		)

	default:
		return ProgramTransition{}, errors.New(
			"workflow terminal child has no terminal phase",
		)
	}
}

func workflowTerminalChildFrame(
	execution Execution,
	parent ProgramFrameID,
	path string,
) *ProgramFrame {
	for index := range execution.ProgramFrames {
		value := execution.ProgramFrames[index]
		if value.ParentFrameID == parent && value.Path == path {
			return &value
		}
	}
	return nil
}

func (i *WorkflowInterpreter) settleWorkflowCatch(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	result WorkflowBlockResult,
	childErr *SanitizedError,
) (ProgramTransition, error) {
	if state.Terminal == nil {
		return ProgramTransition{}, errors.New(
			"workflow terminal state is unavailable",
		)
	}

	if childErr != nil {
		if state.Terminal.Error != nil {
			state.Terminal.Error = appendWorkflowErrorCause(
				state.Terminal.Error,
				*childErr,
			)
		} else {
			state.Terminal.Error = cloneWorkflowSanitizedError(childErr)
		}
		state.Terminal.Status = ProgramFrameStateFailed
		state.Terminal.Phase = ""
		state.Terminal.ChildPath = ""
		return i.startWorkflowFinally(frame, execution, program, state)
	}

	switch result.Exit {
	case WorkflowExitNormal,
		WorkflowExitReturnBlock,
		WorkflowExitReturnWorkflow:
	default:
		return ProgramTransition{}, errors.New(
			"workflow catch handler returned unsupported control flow",
		)
	}

	if state.Terminal.MergeState {
		state.State = result.State.Clone()
	}
	state.Terminal.Status = ProgramFrameStateCompleted
	state.Terminal.Exit = result.Exit
	state.Terminal.Value = result.Value.Clone()
	state.Terminal.Error = nil
	state.Terminal.Phase = ""
	state.Terminal.ChildPath = ""
	return i.startWorkflowFinally(frame, execution, program, state)
}

func (i *WorkflowInterpreter) settleWorkflowFinally(
	program WorkflowProgram,
	state WorkflowState,
	childErr *SanitizedError,
) (ProgramTransition, error) {
	if state.Terminal == nil {
		return ProgramTransition{}, errors.New(
			"workflow terminal state is unavailable",
		)
	}

	if childErr == nil {
		return i.finishWorkflowTerminal(program, state)
	}

	if state.Terminal.Status == ProgramFrameStateCompleted {
		state.Terminal.Status = ProgramFrameStateFailed
		state.Terminal.Error = cloneWorkflowSanitizedError(childErr)
		return i.finishWorkflowTerminal(program, state)
	}

	state.Terminal.Error = appendWorkflowErrorCause(
		state.Terminal.Error,
		*childErr,
	)
	return i.finishWorkflowTerminal(program, state)
}

func (i *WorkflowInterpreter) finishWorkflowTerminal(
	program WorkflowProgram,
	state WorkflowState,
) (ProgramTransition, error) {
	if state.Terminal == nil {
		return ProgramTransition{}, errors.New(
			"workflow terminal state is unavailable",
		)
	}

	terminal := *state.Terminal
	state.Terminal = nil

	switch terminal.Status {
	case ProgramFrameStateCompleted:
		if program.Scope.Child {
			result := WorkflowBlockResult{
				Exit:  terminal.Exit,
				Value: terminal.Value.Clone(),
				State: state.State.Clone(),
			}
			if err := result.Validate(); err != nil {
				return ProgramTransition{}, err
			}
			status := ProgramFrameStateCompleted
			return workflowTransition(
				state,
				&status,
				MustCanonicalValue(result),
				nil,
			)
		}

		if err := validateWorkflowSchema(
			program.OutputSchema,
			terminal.Value,
		); err != nil {
			status := ProgramFrameStateFailed
			transition, transitionErr := workflowTransition(
				state,
				&status,
				nil,
				nil,
			)
			if transitionErr != nil {
				return ProgramTransition{}, transitionErr
			}
			transition.Error = &SanitizedError{
				Code:     "workflow_output_invalid",
				Category: ErrorCategoryValidation,
				Message:  err.Error(),
			}
			return transition, nil
		}
		status := ProgramFrameStateCompleted
		return workflowTransition(
			state,
			&status,
			terminal.Value,
			nil,
		)

	case ProgramFrameStateCancelled:
		status := ProgramFrameStateCancelled
		transition, err := workflowTransition(
			state,
			&status,
			nil,
			nil,
		)
		if err != nil {
			return ProgramTransition{}, err
		}
		transition.Error = workflowTerminalFailure(
			status,
			terminal.Error,
		)
		return transition, nil

	case ProgramFrameStateFailed:
		status := ProgramFrameStateFailed
		transition, err := workflowTransition(
			state,
			&status,
			nil,
			nil,
		)
		if err != nil {
			return ProgramTransition{}, err
		}
		transition.Error = workflowTerminalFailure(
			status,
			terminal.Error,
		)
		return transition, nil

	default:
		return ProgramTransition{}, errors.New(
			"workflow terminal state has an invalid status",
		)
	}
}
