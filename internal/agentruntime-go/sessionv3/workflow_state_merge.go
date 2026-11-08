package sessionv3

import (
	"errors"
	"fmt"
	"strconv"
)

const workflowStateMergeWorkPrefix = "state-merge"

func cloneWorkflowParentStateMerge(
	value *WorkflowParentStateMerge,
) *WorkflowParentStateMerge {
	if value == nil {
		return nil
	}

	output := &WorkflowParentStateMerge{
		Values: make([]WorkflowValuePair, len(value.Values)),
		Remove: append([]string(nil), value.Remove...),
	}
	for index := range value.Values {
		output.Values[index] = WorkflowValuePair{
			Key:   value.Values[index].Key,
			Value: cloneWorkflowValue(value.Values[index].Value),
		}
	}
	return output
}

func cloneWorkflowStateUpdate(
	value StateUpdateWork,
) StateUpdateWork {
	output := value
	output.Base = value.Base.Clone()
	output.Schema = value.Schema.Clone()
	output.Assignments = make([]StateAssignment, len(value.Assignments))
	for index := range value.Assignments {
		output.Assignments[index] = StateAssignment{
			Path:  value.Assignments[index].Path,
			Value: value.Assignments[index].Value.Clone(),
		}
	}
	output.RemovePaths = append([]string(nil), value.RemovePaths...)
	return output
}

func (i *WorkflowInterpreter) ensureParentStateMerge(
	frame ProgramFrame,
	program WorkflowProgram,
	state *WorkflowState,
	output CanonicalValue,
) (bool, error) {
	if state == nil || state.Pending == nil ||
		state.Pending.ParentStateMerge == nil {
		return false, nil
	}

	pending := state.Pending
	if pending.StateMerge != nil {
		return !pending.StateMerge.Applied, nil
	}
	if pending.StateOwner == nil {
		return false, errors.New("workflow parent state merge has no state owner")
	}
	if err := pending.StateOwner.Validate(); err != nil {
		return false, err
	}

	result := output.Clone()
	evaluationProgram := program
	evaluationProgram.Scope.Result = &result

	assignments := make(
		[]StateAssignment,
		0,
		len(pending.ParentStateMerge.Values),
	)
	for _, assignment := range pending.ParentStateMerge.Values {
		value, err := i.evaluate(
			assignment.Value,
			evaluationProgram,
			*state,
		)
		if err != nil {
			return false, fmt.Errorf(
				"evaluate workflow parent state merge %q: %w",
				assignment.Key,
				err,
			)
		}
		assignments = append(assignments, StateAssignment{
			Path:  assignment.Key,
			Value: value,
		})
	}

	update := StateUpdateWork{
		StateOwnerID: pending.StateOwner.ID,
		Base:         state.State.Clone(),
		Schema:       pending.StateOwner.Schema.Clone(),
		Assignments:  assignments,
		RemovePaths:  append([]string(nil), pending.ParentStateMerge.Remove...),
	}
	if err := update.Validate(); err != nil {
		return false, err
	}

	workItemID := WorkItemID(
		workflowStateMergeWorkPrefix + ":" +
			string(frame.ID) + ":" +
			strconv.Itoa(state.Index),
	)
	if err := validateID(
		"workflow state merge work item ID",
		string(workItemID),
	); err != nil {
		return false, err
	}

	pending.StateMerge = &WorkflowParentStateMergeProgress{
		WorkItemID: workItemID,
		Owner:      *cloneWorkflowStateOwner(pending.StateOwner),
		Update:     cloneWorkflowStateUpdate(update),
		Output:     output.Clone(),
	}
	return true, nil
}

func (i *WorkflowInterpreter) driveParentStateMerge(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state *WorkflowState,
) (ProgramTransition, bool, error) {
	if state == nil || state.Pending == nil ||
		state.Pending.StateMerge == nil {
		return ProgramTransition{}, false, nil
	}

	progress := state.Pending.StateMerge
	if progress.Applied {
		return ProgramTransition{}, false, nil
	}

	item := executionWorkItem(execution, progress.WorkItemID)
	if item == nil {
		transition, err := workflowTransition(
			*state,
			nil,
			nil,
			[]WorkItemIntent{{
				ID:          progress.WorkItemID,
				ProgramPath: state.Pending.Path + "/state-merge",
				Type:        WorkItemTypeStateUpdate,
				Automatic:   true,
				Effect:      EffectContract{Kind: EffectKindRead},
				Input: WorkItemPayload{
					StateUpdate: ptrWorkflowStateUpdate(progress.Update),
				},
			}},
		)
		return transition, true, err
	}
	if !item.State.Terminal() {
		transition, err := workflowTransition(*state, nil, nil, nil)
		return transition, true, err
	}

	handled, err := i.settleParentStateMerge(
		frame,
		execution,
		program,
		state,
		progress.WorkItemID,
	)
	if !handled || err != nil {
		return ProgramTransition{}, handled, err
	}
	transition, err := i.drive(frame, execution, program, *state)
	return transition, true, err
}

func (i *WorkflowInterpreter) settleParentStateMerge(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state *WorkflowState,
	workItemID WorkItemID,
) (bool, error) {
	if state == nil || state.Pending == nil ||
		state.Pending.StateMerge == nil ||
		state.Pending.StateMerge.WorkItemID != workItemID {
		return false, nil
	}

	pending := state.Pending
	progress := pending.StateMerge
	item := executionWorkItem(execution, workItemID)
	if item == nil || item.FrameID != frame.ID {
		return true, errors.New("workflow state merge work item is unavailable")
	}
	if !item.State.Terminal() {
		return true, nil
	}
	if item.State != WorkItemStateCompleted ||
		item.Input.StateUpdate == nil {
		failure := workflowSettledFailure(program, *state, *item)
		if failure == nil {
			failure = &SanitizedError{
				Code:     "workflow_state_merge_failed",
				Category: ErrorCategoryExecution,
				Message:  "workflow parent state merge did not complete",
			}
		}
		return true, workflowChildError{errorValue: *failure}
	}

	update := item.Input.StateUpdate
	if update.StateOwnerID != progress.Owner.ID ||
		!canonicalValuesEqual(update.Schema, progress.Owner.Schema) ||
		!canonicalValuesEqual(update.Base, progress.Update.Base) {
		return true, errors.New(
			"workflow state merge work item differs from its persisted boundary",
		)
	}

	value, err := workflowWorkValue(*item)
	if err != nil {
		return true, err
	}
	if err := validateWorkflowSchema(progress.Owner.Schema, value); err != nil {
		return true, fmt.Errorf(
			"validate workflow parent state merge result: %w",
			err,
		)
	}

	state.State = value.Clone()
	progress.Applied = true

	step, err := workflowStepForPath(program.Root, pending.Path)
	if err != nil {
		return true, err
	}
	switch pending.Kind {
	case WorkflowPendingParallel:
		return true, i.finishParallel(frame, state, program, step)
	case WorkflowPendingForeach:
		return true, i.finishForeach(frame, state, program, step)
	default:
		return true, errors.New("workflow state merge has an invalid control owner")
	}
}

func ptrWorkflowStateUpdate(
	value StateUpdateWork,
) *StateUpdateWork {
	output := cloneWorkflowStateUpdate(value)
	return &output
}
