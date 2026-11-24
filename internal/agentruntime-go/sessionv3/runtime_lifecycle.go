package sessionv3

import (
	"context"
	"errors"
	"strings"
)

const (
	maxExecutionSettlementPasses = 64
	executionCancelledMessage    = "execution was cancelled"
	frameBindingCleanupOp        = "session-v3 committed program frame binding cleanup"
	executionBindingCleanupOp    = "session-v3 committed execution binding cleanup"
	lifecycleKeySeparator        = "\x00"

	checkpointReasonSessionCreated           CheckpointReason = "sessionCreated"
	checkpointReasonSessionChanged           CheckpointReason = "sessionChanged"
	checkpointReasonMessageSubmitted         CheckpointReason = "messageSubmitted"
	checkpointReasonMessageEditedAndResent   CheckpointReason = "messageEditedAndResent"
	checkpointReasonWorkflowStarted          CheckpointReason = "workflowStarted"
	checkpointReasonWorkBinding              CheckpointReason = "workBinding"
	checkpointReasonWorkBindingReverted      CheckpointReason = "workBindingReverted"
	checkpointReasonWorkDispatchReverted     CheckpointReason = "workDispatchReverted"
	checkpointReasonWorkDispatched           CheckpointReason = "workDispatched"
	checkpointReasonWorkFinished             CheckpointReason = "workFinished"
	checkpointReasonWorkBindingInvalid       CheckpointReason = "workBindingInvalid"
	checkpointReasonWorkNotInvoked           CheckpointReason = "workNotInvoked"
	checkpointReasonWorkPanicked             CheckpointReason = "workPanicked"
	checkpointReasonWorkBudgetBlocked        CheckpointReason = "workBudgetBlocked"
	checkpointReasonWorkStepBudgetBlocked    CheckpointReason = "workStepBudgetBlocked"
	checkpointReasonWorkConcurrencyWaiting   CheckpointReason = "workConcurrencyWaiting"
	checkpointReasonChildCallBudgetBlocked   CheckpointReason = "childCallBudgetBlocked"
	checkpointReasonPortInputBlocked         CheckpointReason = "portInputBlocked"
	checkpointReasonPortStream               CheckpointReason = "portStream"
	checkpointReasonResourceClaimInvalid     CheckpointReason = "resourceClaimInvalid"
	checkpointReasonResourceWait             CheckpointReason = "resourceWait"
	checkpointReasonResourceReady            CheckpointReason = "resourceReady"
	checkpointReasonPortBindingUnavailable   CheckpointReason = "portBindingUnavailable"
	checkpointReasonWorkBlocked              CheckpointReason = "workBlocked"
	checkpointReasonWorkDeadlineElapsed      CheckpointReason = "workDeadlineElapsed"
	checkpointReasonWaitRequested            CheckpointReason = "waitRequested"
	checkpointReasonWaitResolutionRequested  CheckpointReason = "waitResolutionRequested"
	checkpointReasonWaitResolutionFailed     CheckpointReason = "waitResolutionFailed"
	checkpointReasonWaitResolved             CheckpointReason = "waitResolved"
	checkpointReasonWaitExpired              CheckpointReason = "waitExpired"
	checkpointReasonRecoveryWaitResolved     CheckpointReason = "recoveryWaitResolved"
	checkpointReasonTimerFired               CheckpointReason = "timerFired"
	checkpointReasonEmitBlocked              CheckpointReason = "emitBlocked"
	checkpointReasonEmitFinished             CheckpointReason = "emitFinished"
	checkpointReasonRetryCancelled           CheckpointReason = "retryCancelled"
	checkpointReasonRetryDeadlineElapsed     CheckpointReason = "retryDeadlineElapsed"
	checkpointReasonRetryReady               CheckpointReason = "retryReady"
	checkpointReasonResourceWaitCancelled    CheckpointReason = "resourceWaitCancelled"
	checkpointReasonResourceDeadlineElapsed  CheckpointReason = "resourceDeadlineElapsed"
	checkpointReasonExecutionDeadlineElapsed CheckpointReason = "executionDeadlineElapsed"
	checkpointReasonCleanupDeadlineElapsed   CheckpointReason = "cleanupDeadlineElapsed"
	checkpointReasonExecutionChanged         CheckpointReason = "executionChanged"
	checkpointReasonLLMStream                CheckpointReason = "llmStream"
	checkpointReasonAppToolQueued            CheckpointReason = "appToolQueued"
	checkpointReasonAppContextStaged         CheckpointReason = "appContextStaged"
	checkpointReasonRecoveryApplied          CheckpointReason = "recoveryApplied"
	checkpointReasonTriggerRecovery          CheckpointReason = "triggerRecovery"
	checkpointReasonTriggerFired             CheckpointReason = "triggerFired"
	checkpointReasonReconciliationRecovery   CheckpointReason = "reconciliationRecovery"
	checkpointReasonDurableWaitRecovery      CheckpointReason = "durableWaitRecovery"
)

type executionSettlementSignature struct {
	state             ExecutionState
	cleanupActive     bool
	frames            int
	terminalFrames    int
	workItems         int
	terminalWorkItems int
	waits             int
	effects           int
}

func executionCancellationMessage(execution Execution) string {
	if execution.Cleanup.Primary != nil &&
		strings.TrimSpace(execution.Cleanup.Primary.Message) != "" {
		return execution.Cleanup.Primary.Message
	}
	return executionCancelledMessage
}

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

func executionLocalTerminalReady(execution Execution) bool {
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
	return true
}

func executionTerminalReadyLocked(
	session *Session,
	execution Execution,
) bool {
	if !executionLocalTerminalReady(execution) {
		return false
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
	frameCloser, frameCloseSupported := r.deps.Binder.(ProgramFrameBindingCloser)
	executionCloser, executionCloseSupported := r.deps.Binder.(ExecutionBindingCloser)
	if (!frameCloseSupported || frameCloser == nil) &&
		(!executionCloseSupported || executionCloser == nil) {
		return
	}
	if r.closedFrameBindings == nil {
		r.closedFrameBindings = make(map[string]struct{})
	}
	if r.closedExecutionBindings == nil {
		r.closedExecutionBindings = make(map[ExecutionID]struct{})
	}
	for index := range session.Executions {
		execution := &session.Executions[index]
		if frameCloseSupported && frameCloser != nil {
			for _, frame := range execution.ProgramFrames {
				if !frame.Status.Terminal() ||
					r.frameHasLiveOwnedWorkLocked(session, *execution, frame.ID) {
					continue
				}
				key := string(execution.ID) + lifecycleKeySeparator + string(frame.ID)
				if _, alreadyClosed := r.closedFrameBindings[key]; alreadyClosed {
					continue
				}
				r.closedFrameBindings[key] = struct{}{}
				executionID := execution.ID
				frameID := frame.ID
				r.launchAsync(
					frameBindingCleanupOp,
					func() {
						_ = frameCloser.CloseProgramFrameBinding(
							context.WithoutCancel(r.lifecycleCtx),
							executionID,
							frameID,
						)
					},
					nil,
				)
			}
		}
		if executionCloseSupported && executionCloser != nil && execution.State.Terminal() {
			if _, alreadyClosed := r.closedExecutionBindings[execution.ID]; alreadyClosed {
				continue
			}
			r.closedExecutionBindings[execution.ID] = struct{}{}
			executionID := execution.ID
			r.launchAsync(
				executionBindingCleanupOp,
				func() {
					_ = executionCloser.CloseExecutionBinding(
						context.WithoutCancel(r.lifecycleCtx),
						executionID,
					)
				},
				nil,
			)
		}
	}
}

func computeExecutionSettlementSignature(
	execution Execution,
) executionSettlementSignature {
	signature := executionSettlementSignature{
		state:         execution.State,
		cleanupActive: execution.Cleanup.Active,
		frames:        len(execution.ProgramFrames),
		workItems:     len(execution.WorkItems),
		waits:         len(execution.Waits),
		effects:       len(execution.Effects),
	}
	for _, frame := range execution.ProgramFrames {
		if frame.Status.Terminal() {
			signature.terminalFrames++
		}
	}
	for _, item := range execution.WorkItems {
		if item.State.Terminal() {
			signature.terminalWorkItems++
		}
	}
	return signature
}

// finalizeExecutionSettlementLocked converges all pre-checkpoint Saga,
// cleanup, frame-cancellation, child, wait, and Execution terminal mutations
// before the authoritative checkpoint commits.
func (r *Runtime) finalizeExecutionSettlementLocked(
	ctx context.Context,
	session *Session,
	executionID ExecutionID,
	primaryError *SanitizedError,
) (*Execution, error) {
	if session == nil || executionID == "" {
		return nil, ErrExecutionNotFound
	}

	for range maxExecutionSettlementPasses {
		execution := mutableSessionExecution(session, executionID)
		if execution == nil {
			return nil, ErrExecutionNotFound
		}
		before := computeExecutionSettlementSignature(*execution)

		r.resumeSagaCompensationFramesLocked(session, execution)
		execution = mutableSessionExecution(session, executionID)
		if execution == nil {
			return nil, ErrExecutionNotFound
		}

		r.beginFailureDrivenCleanupLocked(execution, primaryError)
		execution = mutableSessionExecution(session, executionID)
		if execution == nil {
			return nil, ErrExecutionNotFound
		}

		r.retireInactiveSagaCompensationsLocked(execution)
		execution = mutableSessionExecution(session, executionID)
		if execution == nil {
			return nil, ErrExecutionNotFound
		}
		if r.finishResumableCleanupLocked(execution) {
			execution.UpdatedAt = r.now()
		}

		if err := r.settlePendingFrameCancellationsLocked(session, execution); err != nil {
			return nil, err
		}
		execution = mutableSessionExecution(session, executionID)
		if execution == nil {
			return nil, ErrExecutionNotFound
		}

		if !execution.State.Terminal() &&
			execution.Controls.CancellationRequested &&
			executionTerminalReadyLocked(session, *execution) {
			r.completeExecutionLocked(
				context.WithoutCancel(ctx),
				session,
				execution,
				ExecutionOutcome{
					Kind:        ExecutionOutcomeCancelled,
					Code:        string(ExecutionOutcomeCancelled),
					Message:     executionCancellationMessage(*execution),
					CompletedAt: r.now(),
				},
			)
			execution = mutableSessionExecution(session, executionID)
			if execution == nil {
				return nil, ErrExecutionNotFound
			}
		}

		wasTerminal := execution.State.Terminal()
		r.refreshExecutionStateLocked(execution)
		if !wasTerminal && execution.State.Terminal() {
			r.settleTerminalExecutionLocked(session, execution)
			execution = mutableSessionExecution(session, executionID)
			if execution == nil {
				return nil, ErrExecutionNotFound
			}
		}

		if computeExecutionSettlementSignature(*execution) == before {
			return execution, nil
		}
	}

	execution := mutableSessionExecution(session, executionID)
	if execution == nil {
		return nil, ErrExecutionNotFound
	}
	return execution, nil
}

func (r *Runtime) commitTerminalMutationLocked(
	ctx context.Context,
	session *Session,
	executionID ExecutionID,
	reason CheckpointReason,
	event RuntimeEventKind,
) bool {
	execution, err := r.finalizeExecutionSettlementLocked(
		ctx,
		session,
		executionID,
		nil,
	)
	if err != nil || execution == nil {
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
