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

		if err := r.cancelFrameTreeLocked(
			session,
			execution,
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

		if err := r.cancelFrameTreeLocked(
			session,
			execution,
			frame.ID,
		); err != nil {
			return err
		}
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

	executionID := execution.ID
	for _, id := range ids {
		current := mutableSessionExecution(session, executionID)
		if current == nil {
			return errors.New(
				"workflow cancellation execution is unavailable",
			)
		}
		item := mutableWorkItem(current, id)
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
		r.cancelFrameWorkItemLocked(session, current, item)
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
	session *Session,
	execution *Execution,
	frameID ProgramFrameID,
) error {
	frame := mutableFrame(execution, frameID)
	if frame == nil || frame.Status.Terminal() {
		return nil
	}
	if workflowFrameRunsCleanup(*frame) {
		return nil
	}

	children := append([]ProgramFrameID(nil), frame.ChildFrameIDs...)
	for _, childID := range children {
		if err := r.cancelFrameTreeLocked(
			session,
			execution,
			childID,
		); err != nil {
			return err
		}
	}

	executionID := execution.ID
	workItemIDs := make([]WorkItemID, 0)
	for _, item := range execution.WorkItems {
		if item.FrameID != frameID || item.State.Terminal() {
			continue
		}
		workItemIDs = append(workItemIDs, item.ID)
	}
	for _, itemID := range workItemIDs {
		current := mutableSessionExecution(session, executionID)
		if current == nil {
			return errors.New(
				"workflow cancellation execution is unavailable",
			)
		}
		item := mutableWorkItem(current, itemID)
		if item == nil || item.State.Terminal() {
			continue
		}
		r.cancelFrameWorkItemLocked(session, current, item)
	}

	execution = mutableSessionExecution(session, executionID)
	if execution == nil {
		return errors.New("workflow cancellation execution is unavailable")
	}
	frame = mutableFrame(execution, frameID)
	if frame == nil || frame.Status.Terminal() {
		return nil
	}
	return r.advanceFrameLocked(
		session,
		execution,
		frame.ID,
		ProgramEvent{Kind: ProgramEventCancelled},
	)
}

func (r *Runtime) cancelFrameWorkItemLocked(
	session *Session,
	execution *Execution,
	item *WorkItem,
) {
	if session == nil ||
		execution == nil ||
		item == nil ||
		item.State.Terminal() ||
		item.Cleanup {
		return
	}

	switch item.State {
	case WorkItemStateDispatched:
		if cancel := r.cancels[item.ID]; cancel != nil {
			cancel()
			delete(r.cancels, item.ID)
		}
		return

	case WorkItemStateBinding:
	case WorkItemStateReady,
		WorkItemStateResourceWaiting,
		WorkItemStateRetryWaiting,
		WorkItemStateWaiting:
	default:
		return
	}

	r.settleUndispatchedWorkLocked(
		session,
		execution,
		item,
		WorkItemStateCancelled,
		&SanitizedError{
			Code:     workflowChildCancelled,
			Category: ErrorCategoryCancellation,
			Message:  "workflow work item was cancelled by its owner",
		},
		false,
	)
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
	r.beginCleanupLocked(
		execution,
		&SanitizedError{
			Code:     "execution_cancelled",
			Category: ErrorCategoryCancellation,
			Message:  firstText(reason, "execution was cancelled"),
		},
	)

	executionID := execution.ID
	workItemIDs := make([]WorkItemID, 0, len(execution.WorkItems))
	for _, item := range execution.WorkItems {
		if item.State.Terminal() {
			continue
		}
		workItemIDs = append(workItemIDs, item.ID)
	}
	for _, itemID := range workItemIDs {
		current := mutableSessionExecution(session, executionID)
		if current == nil {
			return
		}
		item := mutableWorkItem(current, itemID)
		if item == nil || item.State.Terminal() {
			continue
		}
		r.cancelFrameWorkItemLocked(session, current, item)
	}

	execution = mutableSessionExecution(session, executionID)
	if execution == nil {
		return
	}

	rootFrames := make([]ProgramFrameID, 0, len(execution.ProgramFrames))
	for _, frame := range execution.ProgramFrames {
		if frame.ParentFrameID != "" ||
			frame.Status.Terminal() ||
			workflowFrameRunsCleanup(frame) {
			continue
		}
		rootFrames = append(rootFrames, frame.ID)
	}
	for _, frameID := range rootFrames {
		if err := r.cancelFrameTreeLocked(
			session,
			execution,
			frameID,
		); err != nil {
			return
		}
	}
	if hasDispatched(*execution) ||
		hasOutstandingCleanup(*execution) {
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
