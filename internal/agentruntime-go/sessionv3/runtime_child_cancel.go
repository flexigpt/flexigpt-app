package sessionv3

import (
	"errors"
)

const workflowChildCancelled = "workflow_child_cancelled"

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
		if err := r.advanceFrameLocked(
			session,
			execution,
			parentID,
			ProgramEvent{
				Kind:         ProgramEventChildSettled,
				ChildFrameID: child.ID,
			},
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
		if cancel := r.cancels[item.ID]; cancel != nil {
			cancel()
			delete(r.cancels, item.ID)
		}
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
