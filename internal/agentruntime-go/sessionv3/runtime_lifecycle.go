package sessionv3

import (
	"context"
	"errors"
)

// settlePendingFrameCancellationsLocked keeps a cancelled frame alive until
// every owned WorkItem, child frame, retained claim, and cancellation-delivery
// obligation has settled. A dispatched target therefore remains attached to
// its original semantic owner until its late evidence is durable.
func (r *Runtime) settlePendingFrameCancellationsLocked(
	session *Session,
	execution *Execution,
) error {
	if session == nil || execution == nil {
		return errors.New("frame cancellation owner is unavailable")
	}

	for {
		executionID := execution.ID
		ready := make([]ProgramFrameID, 0)
		for _, frame := range execution.ProgramFrames {
			if !frame.CancellationRequested ||
				frame.Status.Terminal() ||
				r.frameHasLiveOwnedWorkLocked(session, *execution, frame.ID) {
				continue
			}
			ready = append(ready, frame.ID)
		}
		if len(ready) == 0 {
			return nil
		}

		progressed := false
		for _, frameID := range ready {
			current := mutableSessionExecution(session, executionID)
			if current == nil {
				return ErrExecutionNotFound
			}
			frame := mutableFrame(current, frameID)
			if frame == nil ||
				frame.Status.Terminal() ||
				!frame.CancellationRequested ||
				r.frameHasLiveOwnedWorkLocked(session, *current, frameID) {
				continue
			}
			if err := r.advanceFrameLocked(
				session,
				current,
				frameID,
				ProgramEvent{Kind: ProgramEventCancelled},
			); err != nil {
				return err
			}
			progressed = true
		}
		if !progressed {
			return nil
		}
		execution = mutableSessionExecution(session, executionID)
		if execution == nil {
			return ErrExecutionNotFound
		}
	}
}

func (r *Runtime) frameHasLiveOwnedWorkLocked(
	session *Session,
	execution Execution,
	frameID ProgramFrameID,
) bool {
	if mutableFrame(&execution, frameID) == nil {
		return false
	}
	for _, item := range execution.WorkItems {
		if item.FrameID != frameID &&
			!isDescendantFrame(execution, frameID, item.FrameID) {
			continue
		}
		if !item.State.Terminal() {
			return true
		}
	}
	for _, frame := range execution.ProgramFrames {
		if frame.ID == frameID ||
			!isDescendantFrame(execution, frameID, frame.ID) {
			continue
		}
		if !frame.Status.Terminal() {
			return true
		}
	}
	frame := mutableFrame(&execution, frameID)
	if frame != nil {
		for _, childID := range frame.ChildExecutionIDs {
			child := mutableSessionExecution(session, childID)
			if child != nil && child.CancelWithParent && !child.State.Terminal() {
				return true
			}
		}
	}
	for _, lease := range execution.ResourceLeases {
		if lease.FrameID == frameID {
			return true
		}
	}
	return false
}

func executionTerminalReadyLocked(
	session *Session,
	execution Execution,
) bool {
	for _, item := range execution.WorkItems {
		if !item.State.Terminal() {
			return false
		}
	}
	for _, wait := range execution.Waits {
		if wait.Status == WaitStatusPending ||
			wait.Status == WaitStatusResolving {
			return false
		}
	}
	if len(execution.ResourceLeases) != 0 ||
		hasOutstandingCleanup(execution) {
		return false
	}
	for _, effect := range execution.Effects {
		switch effect.State {
		case CompensationStatePending, CompensationStateDispatching:
			return false
		default:
		}
	}
	for _, childID := range execution.ChildExecutionIDs {
		child := mutableSessionExecution(session, childID)
		if child != nil && child.CancelWithParent && !child.State.Terminal() {
			return false
		}
	}
	return true
}

func (r *Runtime) beginFailureDrivenCleanupLocked(
	execution *Execution,
	primary *SanitizedError,
) {
	if execution == nil || execution.Cleanup.Active {
		return
	}
	for _, frame := range execution.ProgramFrames {
		if frame.Status.Terminal() {
			continue
		}
		if workflowFrameRunsCleanup(frame) || frame.Cleanup {
			r.beginCleanupLocked(execution, primary)
			return
		}
	}
	for _, item := range execution.WorkItems {
		if !item.State.Terminal() && item.Cleanup {
			r.beginCleanupLocked(execution, primary)
			return
		}
	}
}

func (r *Runtime) scheduleCommittedFrameClosuresLocked(session *Session) {
	if r == nil || session == nil {
		return
	}
	closer, supported := r.deps.Binder.(ProgramFrameBindingCloser)
	if !supported || closer == nil {
		return
	}
	if r.closedFrameBindings == nil {
		r.closedFrameBindings = make(map[string]struct{})
	}
	for index := range session.Executions {
		execution := &session.Executions[index]
		for _, frame := range execution.ProgramFrames {
			if !frame.Status.Terminal() ||
				r.frameHasLiveOwnedWorkLocked(session, *execution, frame.ID) {
				continue
			}
			key := string(execution.ID) + "\x00" + string(frame.ID)
			if _, alreadyClosed := r.closedFrameBindings[key]; alreadyClosed {
				continue
			}
			r.closedFrameBindings[key] = struct{}{}
			executionID := execution.ID
			frameID := frame.ID
			r.launchAsync(
				"session-v3 committed program frame binding cleanup",
				func() {
					_ = closer.CloseProgramFrameBinding(
						context.WithoutCancel(r.lifecycleCtx),
						executionID,
						frameID,
					)
				},
				nil,
			)
		}
	}
}

func (r *Runtime) commitTerminalMutationLocked(
	ctx context.Context,
	session *Session,
	executionID ExecutionID,
	reason string,
	event RuntimeEventKind,
) bool {
	execution := mutableSessionExecution(session, executionID)
	if execution == nil {
		return false
	}
	if err := r.settlePendingFrameCancellationsLocked(session, execution); err != nil {
		return false
	}
	execution = mutableSessionExecution(session, executionID)
	if execution == nil {
		return false
	}
	r.refreshExecutionStateLocked(execution)
	r.settleTerminalExecutionLocked(session, execution)
	execution = mutableSessionExecution(session, executionID)
	if execution == nil {
		return false
	}
	r.touchExecutionLocked(session, execution)
	if err := r.persistLocked(ctx, reason, session); err != nil {
		return false
	}
	r.armSessionExecutionTimersLocked(ctx, session)
	r.publishLocked(ctx, session, execution.ID, event)
	return true
}
