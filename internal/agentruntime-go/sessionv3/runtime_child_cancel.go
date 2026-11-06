package sessionv3

import (
	"context"
	"errors"
)

func (r *Runtime) cancelChildFramesLocked(
	session *Session,
	execution *Execution,
	parentID ProgramFrameID,
	paths []string,
) error {
	if len(paths) == 0 {
		return nil
	}
	if session == nil || execution == nil {
		return errors.New("workflow child cancellation owner is unavailable")
	}

	for _, path := range paths {
		var child *ProgramFrame
		for index := range execution.ProgramFrames {
			candidate := &execution.ProgramFrames[index]
			if candidate.ParentFrameID == parentID &&
				candidate.Path == path {
				child = candidate
				break
			}
		}
		if child == nil || child.Status.Terminal() {
			continue
		}

		r.cancelFrameTreeLocked(execution, child.ID)
		if err := r.queueChildJoinLocked(
			session,
			execution,
			parentID,
			child.ID,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) cancelChildFrameIDsLocked(
	session *Session,
	execution *Execution,
	parentID ProgramFrameID,
	ids []ProgramFrameID,
) error {
	if len(ids) == 0 {
		return nil
	}
	if session == nil || execution == nil {
		return errors.New("workflow child cancellation owner is unavailable")
	}

	for _, id := range ids {
		frame := mutableFrame(execution, id)
		if frame == nil {
			return errors.New("workflow cancellation target frame is unavailable")
		}
		if !isDescendantFrame(*execution, parentID, frame.ID) {
			return errors.New(
				"workflow cancellation target is not owned by this frame",
			)
		}
		if frame.Status.Terminal() {
			continue
		}

		r.cancelFrameTreeLocked(execution, frame.ID)
		r.wakeCompletionFrameWaitsLocked(session, execution, *frame)
	}
	return nil
}

func (r *Runtime) cancelWorkItemsLocked(
	session *Session,
	execution *Execution,
	parentFrameID ProgramFrameID,
	ids []WorkItemID,
) error {
	if len(ids) == 0 {
		return nil
	}
	if session == nil || execution == nil {
		return errors.New("workflow work cancellation owner is unavailable")
	}

	now := r.now()
	for _, id := range ids {
		item := mutableWorkItem(execution, id)
		if item == nil {
			return errors.New(
				"workflow cancellation target work item is unavailable",
			)
		}
		if item.FrameID != parentFrameID {
			return errors.New(
				"workflow cancellation target work item is not owned by this frame",
			)
		}
		if item.State.Terminal() {
			continue
		}
		if item.Cleanup {
			continue
		}

		if cancel := r.cancels[item.ID]; cancel != nil {
			cancel()
			delete(r.cancels, item.ID)
		}
		r.cancelResourceWaitLocked(item.ID)
		released := r.releaseWorkResourcesLocked(execution, item.ID)
		r.wakeResourceWaitersLocked(execution, released)
		r.cancelRetryLocked(item.ID)

		if item.WaitID != "" {
			wait := mutableWait(execution, item.WaitID)
			if wait != nil &&
				(wait.Status == WaitStatusPending ||
					wait.Status == WaitStatusResolving) {
				wait.Status = WaitStatusCancelled
				wait.UpdatedAt = now
				wait.Error = &SanitizedError{
					Code:     workflowChildCancelled,
					Category: ErrorCategoryCancellation,
					Message:  "workflow wait was cancelled by its owner",
				}
				r.cancelWaitDeadlineLocked(wait.ID)
				delete(r.privateWaits, wait.ID)
			}
			item.WaitID = ""
		}

		item.State = WorkItemStateCancelled
		item.EffectState = EffectStateConfirmedNotApplied
		item.CompletedAt = &now
		item.UpdatedAt = now
		item.Error = &SanitizedError{
			Code:     workflowChildCancelled,
			Category: ErrorCategoryCancellation,
			Effect:   EffectStateConfirmedNotApplied,
			Message:  "workflow work item was cancelled by its owner",
		}
		if attempt := activeAttempt(execution, item.ID); attempt != nil {
			attempt.State = AttemptStateCancelled
			attempt.EffectState = EffectStateConfirmedNotApplied
			attempt.FinishedAt = &now
			attempt.Error = cloneWorkflowSanitizedError(item.Error)
		}
	}
	return nil
}

func isDescendantFrame(
	execution Execution,
	parentID ProgramFrameID,
	childID ProgramFrameID,
) bool {
	if parentID == "" || childID == "" || parentID == childID {
		return false
	}

	current := mutableFrame(&execution, childID)
	for current != nil && current.ParentFrameID != "" {
		if current.ParentFrameID == parentID {
			return true
		}
		current = mutableFrame(&execution, current.ParentFrameID)
	}
	return false
}

func (r *Runtime) cancelFrameTreeLocked(
	execution *Execution,
	frameID ProgramFrameID,
) {
	frame := mutableFrame(execution, frameID)
	if frame == nil || frame.Status.Terminal() {
		return
	}

	now := r.now()
	frame.Status = ProgramFrameStateCancelled
	frame.Error = &SanitizedError{
		Code:     workflowChildCancelled,
		Category: ErrorCategoryCancellation,
		Message:  "workflow child was cancelled by its parent completion policy",
	}
	frame.UpdatedAt = now

	for index := range execution.WorkItems {
		item := &execution.WorkItems[index]
		if item.FrameID != frameID || item.State.Terminal() {
			continue
		}
		if item.Cleanup {
			continue
		}
		if cancel := r.cancels[item.ID]; cancel != nil {
			cancel()
			delete(r.cancels, item.ID)
		}
		r.cancelResourceWaitLocked(item.ID)
		released := r.releaseWorkResourcesLocked(execution, item.ID)
		r.wakeResourceWaitersLocked(execution, released)
		r.cancelRetryLocked(item.ID)
		item.State = WorkItemStateCancelled
		item.CompletedAt = &now
		item.UpdatedAt = now
	}

	for index := range execution.Waits {
		wait := &execution.Waits[index]
		item := mutableWorkItem(execution, wait.WorkItemID)
		if item == nil || item.FrameID != frameID {
			continue
		}
		if wait.Status != WaitStatusPending &&
			wait.Status != WaitStatusResolving {
			continue
		}
		wait.Status = WaitStatusCancelled
		wait.Error = &SanitizedError{
			Code:     workflowChildCancelled,
			Category: ErrorCategoryCancellation,
			Message:  "workflow child was cancelled by its parent completion policy",
		}
		wait.UpdatedAt = now
		r.cancelWaitDeadlineLocked(wait.ID)
		delete(r.privateWaits, wait.ID)
	}

	for _, childID := range frame.ChildFrameIDs {
		r.cancelFrameTreeLocked(execution, childID)
	}
}

func (r *Runtime) cancelChildExecutionIDsLocked(
	ctx context.Context,
	session *Session,
	parent *Execution,
	parentFrameID ProgramFrameID,
	ids []ExecutionID,
	reason string,
) error {
	if len(ids) == 0 {
		return nil
	}
	if session == nil || parent == nil {
		return errors.New("workflow child execution cancellation owner is unavailable")
	}

	for _, id := range ids {
		child := mutableSessionExecution(session, id)
		if child == nil {
			return errors.New("workflow cancellation target execution is unavailable")
		}
		if child.ParentExecutionID != parent.ID || child.ParentFrameID != parentFrameID {
			return errors.New("workflow cancellation target execution is not owned by this frame")
		}
		r.cancelIndependentExecutionLocked(ctx, session, child, reason)
	}
	return nil
}

func (r *Runtime) cancelOwnedChildExecutionsLocked(
	ctx context.Context,
	session *Session,
	parentID ExecutionID,
	reason string,
) {
	if session == nil || parentID == "" {
		return
	}

	children := make([]ExecutionID, 0)
	for _, execution := range session.Executions {
		if execution.ParentExecutionID == parentID &&
			execution.CancelWithParent &&
			!execution.State.Terminal() {
			children = append(children, execution.ID)
		}
	}
	for _, childID := range children {
		child := mutableSessionExecution(session, childID)
		if child != nil {
			r.cancelIndependentExecutionLocked(ctx, session, child, reason)
		}
	}
}

func (r *Runtime) cancelIndependentExecutionLocked(
	ctx context.Context,
	session *Session,
	execution *Execution,
	reason string,
) {
	if session == nil || execution == nil || execution.State.Terminal() {
		return
	}

	r.cancelOwnedChildExecutionsLocked(ctx, session, execution.ID, reason)

	now := r.now()
	execution.Controls.CancellationRequested = true
	execution.State = ExecutionStateCancelling
	for index := range execution.WorkItems {
		item := &execution.WorkItems[index]
		if item.Cleanup {
			continue
		}
		switch item.State {
		case WorkItemStateReady,
			WorkItemStateBinding,
			WorkItemStateWaiting:
			if cancel := r.cancels[item.ID]; cancel != nil {
				cancel()
				delete(r.cancels, item.ID)
			}
			item.State = WorkItemStateCancelled
			item.CompletedAt = &now
			r.cancelResourceWaitLocked(item.ID)
			released := r.releaseWorkResourcesLocked(execution, item.ID)
			r.wakeResourceWaitersLocked(execution, released)
			item.UpdatedAt = now
		case WorkItemStateResourceWaiting:
			r.cancelResourceWaitLocked(item.ID)
			item.State = WorkItemStateCancelled
			item.CompletedAt = &now
			item.UpdatedAt = now
		case WorkItemStateRetryWaiting:
			r.cancelRetryLocked(item.ID)
			item.State = WorkItemStateCancelled
			item.CompletedAt = &now
			item.UpdatedAt = now
		case WorkItemStateDispatched:
			if cancel := r.cancels[item.ID]; cancel != nil {
				cancel()
				delete(r.cancels, item.ID)
			}
		default:
		}
	}
	for index := range execution.Waits {
		wait := &execution.Waits[index]
		if wait.Status != WaitStatusPending && wait.Status != WaitStatusResolving {
			continue
		}
		wait.Status = WaitStatusCancelled
		wait.UpdatedAt = now
		r.cancelWaitDeadlineLocked(wait.ID)
		delete(r.privateWaits, wait.ID)
	}
	for _, frame := range execution.ProgramFrames {
		if frame.Status.Terminal() {
			continue
		}
		_ = r.advanceFrameLocked(
			session,
			execution,
			frame.ID,
			ProgramEvent{Kind: ProgramEventCancelled},
		)
	}
	if hasDispatched(*execution) {
		r.touchExecutionLocked(session, execution)
		return
	}
	r.completeExecutionLocked(ctx, session, execution, ExecutionOutcome{
		Kind:        ExecutionOutcomeCancelled,
		Code:        string(ExecutionOutcomeCancelled),
		Message:     firstText(reason, "execution was cancelled"),
		CompletedAt: now,
	})
}
