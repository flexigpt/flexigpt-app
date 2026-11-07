package sessionv3

import (
	"encoding/json"
	"errors"
	"fmt"
)

// deferFrameForkLocked persists the exact child scheduling intent before any
// child ProgramFrame is created. The deterministic Fork WorkItem becomes the
// durable boundary between parent interpreter intent and child scheduling.
func (r *Runtime) deferFrameForkLocked(
	execution *Execution,
	frame *ProgramFrame,
	intents []ProgramFrameIntent,
	continueParent bool,
) error {
	if execution == nil || frame == nil {
		return errors.New("fork boundary owner is unavailable")
	}
	if frame.Fork != nil {
		return errors.New("ProgramFrame already has a pending fork boundary")
	}
	if len(intents) == 0 {
		return errors.New("fork boundary has no child intents")
	}

	prepared := make([]ProgramFrameIntent, 0, len(intents))
	childIDs := make([]ProgramFrameID, 0, len(intents))
	seen := make(map[ProgramFrameID]struct{}, len(intents))
	cleanup := false
	for _, input := range intents {
		intent, err := cloneForkIntent(input)
		if err != nil {
			return err
		}
		if intent.ID == "" {
			intent.ID = ProgramFrameID(r.newID())
		}
		if _, duplicate := seen[intent.ID]; duplicate {
			return errors.New("fork boundary child ProgramFrame identity is duplicated")
		}
		if mutableFrame(execution, intent.ID) != nil {
			return fmt.Errorf("fork boundary child ProgramFrame %q already exists", intent.ID)
		}
		if intent.Lifetime == ChildFrameLifetimeIndependent &&
			intent.ExecutionID == "" {
			intent.ExecutionID = ExecutionID(r.newID())
		}
		if err := validateDeferredFrameIntent(intent); err != nil {
			return err
		}
		seen[intent.ID] = struct{}{}
		prepared = append(prepared, intent)
		childIDs = append(childIDs, intent.ID)
		if workflowProgramRunsCleanup(intent.Program) {
			cleanup = true
		}
	}

	workID := WorkItemID(r.newID())
	now := r.now()
	work := WorkItem{
		ID:          workID,
		FrameID:     frame.ID,
		Type:        WorkItemTypeFork,
		State:       WorkItemStateReady,
		ProgramPath: frame.Path + "/fork",
		Effect:      EffectContract{Kind: EffectKindRead},
		Automatic:   true,
		Cleanup:     cleanup,
		Input: WorkItemPayload{
			Fork: &ForkWork{
				ChildFrames: append([]ProgramFrameID(nil), childIDs...),
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := work.Validate(); err != nil {
		return err
	}

	frame.Fork = &ForkBoundary{
		WorkItemID:     workID,
		Frames:         prepared,
		ContinueParent: continueParent,
	}
	frame.WorkItemIDs = append(frame.WorkItemIDs, workID)
	execution.WorkItems = append(execution.WorkItems, work)
	return nil
}

func cloneForkIntent(value ProgramFrameIntent) (ProgramFrameIntent, error) {
	output := value
	output.Deadline = cloneTime(value.Deadline)
	output.Program.Payload = value.Program.Payload.Clone()
	output.Budget = cloneBudget(value.Budget)
	if value.Program.Artifact != nil {
		pin := *value.Program.Artifact
		output.Program.Artifact = &pin
	}
	if value.Ownership != nil {
		ownership := *value.Ownership
		ownership.Budget = cloneExecutionBudget(value.Ownership.Budget)
		output.Ownership = &ownership
	}
	if value.Handoff != nil {
		handoff := *value.Handoff
		output.Handoff = &handoff
	}
	if err := validateDeferredFrameIntent(output); err != nil {
		return ProgramFrameIntent{}, err
	}
	return output, nil
}

func (r *Runtime) settleForkBoundaryLocked(
	session *Session,
	execution *Execution,
	frame *ProgramFrame,
	event ProgramEvent,
) (bool, error) {
	if session == nil || execution == nil || frame == nil ||
		frame.Fork == nil ||
		event.WorkItemID != frame.Fork.WorkItemID {
		return false, nil
	}
	item := mutableWorkItem(execution, event.WorkItemID)
	if item == nil || !item.State.Terminal() {
		return true, nil
	}

	boundary := *frame.Fork
	frame.Fork = nil
	if item.State != WorkItemStateCompleted {
		frame.Status = ProgramFrameStateFailed
		frame.Error = forkJoinFailure(
			"fork_work_failed",
			"durable child scheduling fork did not complete",
			item,
		)
		frame.UpdatedAt = r.now()
		if frame.ParentFrameID != "" {
			if err := r.queueChildJoinLocked(
				session,
				execution,
				frame.ParentFrameID,
				frame.ID,
			); err != nil {
				return true, err
			}
		}
		return true, nil
	}

	if err := r.materializeForkFramesLocked(
		session,
		execution,
		frame.ID,
		boundary,
	); err != nil {
		return true, err
	}
	return true, nil
}

func (r *Runtime) materializeForkFramesLocked(
	session *Session,
	execution *Execution,
	parentID ProgramFrameID,
	boundary ForkBoundary,
) error {
	if session == nil || execution == nil {
		return errors.New("fork materialization owner is unavailable")
	}
	parent := mutableFrame(execution, parentID)
	if parent == nil || parent.Status.Terminal() {
		return errors.New("fork parent ProgramFrame is unavailable")
	}

	attached := make([]ProgramFrameID, 0, len(boundary.Frames))
	independent := make([]ExecutionID, 0)
	for _, intent := range boundary.Frames {
		parent = mutableFrame(execution, parentID)
		if parent == nil || parent.Status.Terminal() {
			return errors.New("fork parent ProgramFrame disappeared")
		}

		childInterpreter := r.interpreterLocked(intent.Program)
		if childInterpreter == nil {
			return fmt.Errorf(
				"fork child interpreter %q is unavailable",
				intent.Program.InterpreterID,
			)
		}
		if intent.Program.InterpreterID == WorkflowInterpreterID {
			childProgram, err := workflowProgramFromSnapshot(intent.Program)
			if err != nil {
				return err
			}
			if err := r.mergeExecutionResourcesLocked(
				execution,
				childProgram.Resources,
			); err != nil {
				return err
			}
		}
		if err := childInterpreter.ValidateSnapshot(intent.Program); err != nil {
			return err
		}

		lifetime := intent.Lifetime
		switch lifetime {
		case "":
			lifetime = ChildFrameLifetimeAttached
		case ChildFrameLifetimeAttached:
		case ChildFrameLifetimeIndependent:
			child, err := r.createIndependentChildExecutionLocked(
				session,
				execution,
				parent,
				intent,
			)
			if err != nil {
				return err
			}
			independent = append(independent, child.ID)
			execution = mutableSessionExecution(session, execution.ID)
			if execution == nil {
				return errors.New("fork parent execution disappeared")
			}
			continue
		default:
			return errors.New("fork child ProgramFrame lifetime is invalid")
		}
		childBudget, budgetErr := narrowProgramFrameBudget(
			effectiveProgramFrameBudget(*execution, *parent),
			intent,
		)
		if budgetErr != nil {
			return budgetErr
		}
		depth, budgetErr := childProgramFrameDepth(*parent, childBudget)
		if budgetErr != nil {
			return budgetErr
		}
		now := r.now()
		deadline, budgetErr := programFrameDeadline(
			now,
			parent.Deadline,
			intent.Deadline,
			childBudget,
		)
		if budgetErr != nil {
			return budgetErr
		}
		state, err := childInterpreter.InitialState(intent.Program)
		if err != nil {
			return err
		}
		if mutableFrame(execution, intent.ID) != nil {
			return fmt.Errorf(
				"fork child ProgramFrame %q already exists",
				intent.ID,
			)
		}
		ports := bindFramePortPlans(intent.Ports, parent.ID, intent.ID)
		if len(ports) != 0 {
			intent.Program, err = workflowProgramWithPortInputs(
				intent.Program,
				nil,
			)
			if err != nil {
				return err
			}
		}

		handoffID := ""
		if intent.Handoff != nil {
			handoffID = intent.Handoff.ID
		}
		child := ProgramFrame{
			ID:            intent.ID,
			ParentFrameID: parent.ID,
			Lifetime:      lifetime,
			Depth:         depth,
			Program:       intent.Program,
			State:         state,
			Status:        ProgramFrameStateRunning,
			Path:          intent.Path,
			CreatedAt:     now,
			UpdatedAt:     now,
			HandoffID:     handoffID,
			Deadline:      deadline,
			Budget:        cloneBudget(&childBudget),
			Ports:         ports,
		}
		execution.ProgramFrames = append(execution.ProgramFrames, child)
		parent = mutableFrame(execution, parentID)
		if parent == nil {
			return errors.New("fork parent ProgramFrame disappeared after child creation")
		}
		parent.ChildFrameIDs = append(parent.ChildFrameIDs, child.ID)
		attached = append(attached, child.ID)
		childFrame := mutableFrame(execution, child.ID)
		if childFrame == nil {
			return errors.New("fork child ProgramFrame is unavailable")
		}
		if err := r.stageFramePortInputsLocked(
			session,
			execution,
			childFrame,
		); err != nil {
			return err
		}
		if len(childFrame.PortInputs) != 0 {
			next, inputErr := workflowProgramWithPortInputs(
				childFrame.Program,
				childFrame.PortInputs,
			)
			if inputErr != nil {
				return inputErr
			}
			childFrame.Program = next
		}

		if intent.Handoff != nil {
			childFrame := mutableFrame(execution, child.ID)
			if childFrame == nil {
				return errors.New("fork handoff child ProgramFrame is unavailable")
			}
			if err := r.applyConversationHandoffLocked(
				session,
				execution,
				parent,
				execution,
				childFrame.ID,
				*intent.Handoff,
			); err != nil {
				return err
			}
		}
		if parent.Program.InterpreterID == BuiltinConversationInterpreterID {
			var state conversationProgramState
			if json.Unmarshal(parent.State.Data, &state) == nil {
				state.ActiveFrameID = child.ID
				parent.State = encodeConversationState(state)
			}
		}
	}

	for _, childID := range attached {
		if err := r.advanceFrameLocked(
			session,
			execution,
			childID,
			ProgramEvent{Kind: ProgramEventStarted},
		); err != nil {
			return err
		}
	}
	for _, executionID := range independent {
		child := mutableSessionExecution(session, executionID)
		if child == nil || len(child.ProgramFrames) == 0 {
			return errors.New("fork independent child execution is unavailable")
		}
		if err := r.advanceFrameLocked(
			session,
			child,
			child.ProgramFrames[0].ID,
			ProgramEvent{Kind: ProgramEventStarted},
		); err != nil {
			return err
		}
		r.touchExecutionLocked(session, child)
	}

	parent = mutableFrame(execution, parentID)
	if parent == nil {
		return errors.New("fork parent ProgramFrame disappeared after child startup")
	}
	if boundary.ContinueParent && !parent.Status.Terminal() {
		return r.advanceFrameLocked(
			session,
			execution,
			parent.ID,
			ProgramEvent{Kind: ProgramEventContinue},
		)
	}
	return nil
}

func (r *Runtime) queueChildJoinLocked(
	session *Session,
	execution *Execution,
	parentID ProgramFrameID,
	childID ProgramFrameID,
) error {
	if session == nil || execution == nil {
		return errors.New("join boundary owner is unavailable")
	}
	parent := mutableFrame(execution, parentID)
	child := mutableFrame(execution, childID)
	if parent == nil || child == nil {
		return errors.New("join boundary ProgramFrame is unavailable")
	}
	if parent.Status.Terminal() {
		return nil
	}
	for _, pending := range parent.Joins {
		if pending.ChildFrameID == childID {
			return nil
		}
	}

	now := r.now()
	workID := WorkItemID(r.newID())
	work := WorkItem{
		ID:          workID,
		FrameID:     parent.ID,
		Type:        WorkItemTypeJoin,
		State:       WorkItemStateReady,
		ProgramPath: parent.Path + "/join/" + string(childID),
		Effect:      EffectContract{Kind: EffectKindRead},
		Automatic:   true,
		Cleanup:     workflowFrameRunsCleanup(*child),
		Input: WorkItemPayload{
			Join: &JoinWork{
				ChildFrames: []ProgramFrameID{childID},
				Policy:      "child-settled",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := work.Validate(); err != nil {
		return err
	}
	parent.WorkItemIDs = append(parent.WorkItemIDs, workID)
	parent.Joins = append(parent.Joins, JoinBoundary{
		WorkItemID:   workID,
		ChildFrameID: childID,
	})
	execution.WorkItems = append(execution.WorkItems, work)
	r.refreshExecutionStateLocked(execution)
	return nil
}

func (r *Runtime) settleJoinBoundaryLocked(
	session *Session,
	execution *Execution,
	frame *ProgramFrame,
	event ProgramEvent,
) (bool, error) {
	if session == nil || execution == nil || frame == nil ||
		event.WorkItemID == "" {
		return false, nil
	}
	index := -1
	for current := range frame.Joins {
		if frame.Joins[current].WorkItemID == event.WorkItemID {
			index = current
			break
		}
	}
	if index < 0 {
		return false, nil
	}

	boundary := frame.Joins[index]
	item := mutableWorkItem(execution, boundary.WorkItemID)
	if item == nil || !item.State.Terminal() {
		return true, nil
	}
	frame.Joins = append(
		frame.Joins[:index],
		frame.Joins[index+1:]...,
	)
	if item.State != WorkItemStateCompleted {
		frame.Status = ProgramFrameStateFailed
		frame.Error = forkJoinFailure(
			"join_work_failed",
			"durable child completion join did not complete",
			item,
		)
		frame.UpdatedAt = r.now()
		if frame.ParentFrameID != "" {
			if err := r.queueChildJoinLocked(
				session,
				execution,
				frame.ParentFrameID,
				frame.ID,
			); err != nil {
				return true, err
			}
		}
		return true, nil
	}

	child := mutableFrame(execution, boundary.ChildFrameID)
	if child == nil || !child.Status.Terminal() {
		return true, errors.New("join boundary child ProgramFrame is unavailable")
	}
	return true, r.advanceFrameLocked(
		session,
		execution,
		frame.ID,
		ProgramEvent{
			Kind:         ProgramEventChildSettled,
			ChildFrameID: child.ID,
		},
	)
}

func forkJoinFailure(
	code string,
	message string,
	item *WorkItem,
) *SanitizedError {
	if item != nil && item.Error != nil {
		value := *item.Error
		value.Details = item.Error.Details.Clone()
		value.Causes = append([]SanitizedError(nil), item.Error.Causes...)
		value.Code = code
		value.Message = message
		return &value
	}
	return &SanitizedError{
		Code:     code,
		Category: ErrorCategoryExecution,
		Message:  message,
	}
}
