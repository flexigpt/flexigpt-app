package sessionv3

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// executionDispatchAvailableLocked identifies execution-wide conditions that
// stop all new work, including cleanup. Cancellation is intentionally handled
// per WorkItem by dispatchAllowedForWorkLocked so eligible cleanup can proceed.
func (r *Runtime) executionDispatchAvailableLocked(execution Execution) bool {
	if r.checkpointPending[execution.SessionID] ||
		execution.Controls.Paused ||
		execution.State.Terminal() {
		return false
	}
	for _, wait := range execution.Waits {
		if wait.Request.Kind == WaitKindRecovery &&
			(wait.Status == WaitStatusPending || wait.Status == WaitStatusResolving) {
			return false
		}
	}
	return true
}

func (r *Runtime) workContext(deadline *time.Time) (context.Context, context.CancelFunc) {
	if deadline == nil {
		return context.WithCancel(r.lifecycleCtx)
	}
	return context.WithDeadline(r.lifecycleCtx, deadline.UTC())
}

// effectiveWorkDeadlineLocked returns the only deadline that can stop a
// WorkItem at the current lifecycle boundary. Cleanup work must not inherit an
// already elapsed ordinary deadline after Runtime starts bounded cleanup.
func (r *Runtime) effectiveWorkDeadlineLocked(
	execution Execution,
	item WorkItem,
) *time.Time {
	if item.Cleanup &&
		execution.Cleanup.Active &&
		execution.Cleanup.Deadline != nil {
		return cloneTime(execution.Cleanup.Deadline)
	}
	return cloneTime(item.Deadline)
}

func (r *Runtime) effectiveAttemptDeadlineLocked(
	execution Execution,
	item WorkItem,
	now time.Time,
) (*time.Time, error) {
	effective := item
	effective.Deadline = r.effectiveWorkDeadlineLocked(execution, item)
	return attemptDeadline(effective, now)
}

// resumeRestoredWork starts only after the complete recovery source has
// returned successfully. Thus no restored execution can race an incomplete
// restore of the same Session document set.
func (r *Runtime) resumeRestoredWork(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	// Restore runs under Runtime.restoring. Every mutation persists, but this
	// function intentionally performs no per-Execution launch. Restore emits
	// one committed Session-wide readiness wave after all Sessions are stable.
	for _, session := range r.sessions {
		if r.checkpointPending[session.ID] {
			continue
		}
		triggerChanged, triggerErr := r.recoverSessionTriggersLocked(
			ctx,
			session,
		)
		queueChanged := false
		if triggerErr == nil {
			queueChanged = r.drainTriggerQueuesLocked(ctx, session, "")
		}
		if triggerErr == nil && (triggerChanged || queueChanged) {
			if err := r.persistLocked(ctx, "triggerRecovery", session); err != nil {
				continue
			}
		}
		r.armSessionTriggersLocked(ctx, session)
		for index := range session.Executions {
			execution := &session.Executions[index]
			if execution.State.Terminal() {
				continue
			}

			reconciling := make([]struct {
				frameID ProgramFrameID
				workID  WorkItemID
			}, 0)
			for _, item := range execution.WorkItems {
				if item.State != WorkItemStateReconciling {
					continue
				}
				reconciling = append(reconciling, struct {
					frameID ProgramFrameID
					workID  WorkItemID
				}{
					frameID: item.FrameID,
					workID:  item.ID,
				})
			}
			for _, value := range reconciling {
				current := mutableSessionExecution(session, execution.ID)
				if current == nil {
					break
				}
				item := mutableWorkItem(current, value.workID)
				if item == nil ||
					item.State != WorkItemStateReconciling ||
					item.FrameID != value.frameID {
					continue
				}
				if err := r.advanceFrameLocked(
					session,
					current,
					value.frameID,
					ProgramEvent{
						Kind:       ProgramEventWorkSettled,
						WorkItemID: value.workID,
					},
				); err != nil {
					continue
				}
				r.touchExecutionLocked(session, current)
				if err := r.persistLocked(
					ctx,
					"reconciliationRecovery",
					session,
				); err != nil {
					continue
				}
				execution = current
			}

			for _, frame := range execution.ProgramFrames {
				if !reconciliationResumeRequired(
					frame,
					*execution,
				) {
					continue
				}
				if err := r.advanceFrameLocked(
					session,
					execution,
					frame.ID,
					ProgramEvent{Kind: ProgramEventContinue},
				); err != nil {
					continue
				}
				r.touchExecutionLocked(session, execution)
				if err := r.persistLocked(
					ctx,
					"reconciliationRecovery",
					session,
				); err != nil {
					continue
				}
				execution = mutableSessionExecution(
					session,
					execution.ID,
				)
				if execution == nil {
					break
				}
			}

			waitChanged, waitErr := r.resolveRestoredDurableWaitsLocked(
				session,
				execution,
			)
			if waitErr == nil && waitChanged {
				current := mutableSessionExecution(session, execution.ID)
				if current == nil {
					continue
				}
				r.touchExecutionLocked(session, current)
				if err := r.persistLocked(
					ctx,
					"durableWaitRecovery",
					session,
				); err != nil {
					continue
				}
				execution = current
			}

			r.armExecutionDueTimesLocked(
				ctx,
				session.ID,
				execution.ID,
				*execution,
			)
		}
	}
}

func (r *Runtime) resolveRestoredDurableWaitsLocked(
	session *Session,
	execution *Execution,
) (bool, error) {
	if session == nil || execution == nil {
		return false, errors.New("restored durable wait owner is unavailable")
	}

	ids := make([]WaitID, 0)
	for _, wait := range execution.Waits {
		if wait.Status != WaitStatusPending || wait.Request.RuntimeBound {
			continue
		}
		switch wait.Request.Kind {
		case WaitKindEvent, WaitKindChannel, WaitKindChild:
			ids = append(ids, wait.ID)
		default:
		}
	}

	changed := false
	for _, id := range ids {
		before := mutableWait(execution, id)
		if before == nil || before.Status != WaitStatusPending {
			continue
		}
		if err := r.resolveReadyWaitLocked(session, execution, id); err != nil {
			return changed, err
		}
		after := mutableWait(execution, id)
		if after != nil && after.Status != WaitStatusPending {
			changed = true
		}
	}
	return changed, nil
}

func validateWaitResponse(wait Wait, response jsonutil.CanonicalValue) error {
	if !response.Valid() {
		return errors.New("wait response is not canonical json")
	}
	value := response
	if len(value) == 0 {
		value = jsonutil.MustCanonicalValue(nil)
	}
	if err := validateWorkflowSchema(wait.Request.InputSchema, value); err != nil {
		return fmt.Errorf("validate wait response: %w", err)
	}
	return nil
}

func waitWakeAt(wait Wait) *time.Time {
	deadline := cloneTime(wait.Request.Deadline)
	if wait.Request.Timer != nil {
		deadline = earlierDeadline(deadline, &wait.Request.Timer.FireAt)
	}
	return deadline
}

// settleTimerLocked handles the timer's successful firing separately from
// its enclosing operation deadline. Equal times give the deadline priority.
func (r *Runtime) settleTimerLocked(
	ctx context.Context,
	session *Session,
	execution *Execution,
	wait *Wait,
	now time.Time,
) bool {
	if wait.Request.Kind != WaitKindTimer || wait.Request.Timer == nil {
		return false
	}

	timer := wait.Request.Timer
	if wait.Request.Deadline != nil &&
		!now.Before(*wait.Request.Deadline) &&
		!timer.FireAt.Before(*wait.Request.Deadline) {
		return false
	}
	if now.Before(timer.FireAt) {
		r.scheduleWaitDeadlineLocked(ctx, session.ID, execution.ID, *wait)
		return true
	}

	item := mutableWorkItem(execution, wait.WorkItemID)
	if item == nil || item.State != WorkItemStateWaiting {
		return true
	}

	wait.Status = WaitStatusResolved
	wait.UpdatedAt = now
	wait.Response = jsonutil.MustCanonicalValue(TimerWaitOutput{
		ScheduledAt: timer.ScheduledAt,
		FireAt:      timer.FireAt,
		FiredAt:     now,
	})
	item.WaitID = ""
	delete(r.privateWaits, wait.ID)

	r.finishWorkLocked(session, execution, item, nil, WorkExecutionResult{
		Output: &WorkItemOutput{
			Value: wait.Response.Clone(),
			Receipt: &ExternalReceipt{
				System:      "session-v3-timer",
				Reference:   string(wait.ID),
				ConfirmedAt: now,
			},
		},
		Confirmed: true,
	})
	_ = r.commitTerminalMutationLocked(
		ctx,
		session,
		execution.ID,
		checkpointReasonTimerFired,
		RuntimeEventWaitChanged,
	)
	return true
}

// normalizeWorkResult prevents invalid domain output from entering durable
// state. Invalid output after dispatch is not evidence of non-application.
func normalizeWorkResult(item WorkItem, result WorkExecutionResult) WorkExecutionResult {
	var invalid error
	switch {
	case result.Output != nil:
		invalid = result.Output.Validate(item.Type)
	case result.Confirmed && result.Error == nil:
		invalid = errors.New("confirmed work has no output")
	}
	if invalid == nil {
		return result
	}

	return WorkExecutionResult{
		Error: &SanitizedError{
			Code:     "work_output_invalid",
			Category: ErrorCategoryValidation,
			Message:  invalid.Error(),
		},
		Confirmed: false,
	}
}

// settleUnstartedAttemptLocked is used only before ExecuteWork is entered.
// This is positive evidence that this attempt did not invoke the external
// target, unlike cancellation after dispatch entered the domain executor.
func (r *Runtime) settleUnstartedAttemptLocked(
	ctx context.Context,
	session *Session,
	execution *Execution,
	item *WorkItem,
) {
	failure := deadlineError()
	if execution.Controls.CancellationRequested || execution.State.Terminal() {
		failure = &SanitizedError{
			Code:     workflowCancelledBeforeInvocation,
			Category: ErrorCategoryCancellation,
			Message:  "work was cancelled before external invocation",
		}
	}
	delete(r.privateBindings, item.ID)
	r.finishWorkLocked(
		session,
		execution,
		item,
		activeAttempt(execution, item.ID),
		WorkExecutionResult{Error: failure, Confirmed: true},
	)
	_ = r.commitTerminalMutationLocked(
		ctx,
		session,
		execution.ID,
		checkpointReasonWorkNotInvoked,
		RuntimeEventWorkChanged,
	)
}

// settleUndispatchedWorkLocked settles work that has not crossed the native
// invocation boundary. It is used by Runtime-owned retry, resource, and
// deadline paths so they retain the ordinary owner, effect, resource, wait,
// completion, Saga, and Execution consequences.
func (r *Runtime) settleUndispatchedWorkLocked(
	session *Session,
	execution *Execution,
	item *WorkItem,
	terminalState WorkItemState,
	failure *SanitizedError,
	advanceOwner bool,
) *Execution {
	if session == nil || execution == nil || item == nil || item.State.Terminal() {
		return execution
	}
	switch terminalState {
	case WorkItemStateCancelled, WorkItemStateFailed, WorkItemStateBlocked:
	default:
		return execution
	}

	executionID := execution.ID
	workItemID := item.ID
	now := r.now()

	if failure == nil {
		failure = &SanitizedError{
			Code:     "work_not_dispatched",
			Category: ErrorCategoryCancellation,
			Message:  "work ended before external invocation",
		}
	}

	if cancel := r.cancels[item.ID]; cancel != nil {
		cancel()
		delete(r.cancels, item.ID)
	}
	delete(r.privateBindings, item.ID)
	r.cancelRetryLocked(item.ID)
	r.cancelResourceWaitLocked(item.ID)

	if item.WaitID != "" {
		if wait := mutableWait(execution, item.WaitID); wait != nil &&
			(wait.Status == WaitStatusPending ||
				wait.Status == WaitStatusResolving) {
			wait.Status = WaitStatusCancelled
			wait.Error = cloneWorkflowSanitizedError(failure)
			wait.UpdatedAt = now
		}
		r.cancelWaitDeadlineLocked(item.WaitID)
		delete(r.privateWaits, item.WaitID)
		item.WaitID = ""
	}

	item.State = terminalState
	item.Output = nil
	item.Error = cloneWorkflowSanitizedError(failure)
	item.CompletedAt = &now
	item.UpdatedAt = now
	item.RetryState = nil
	item.ResourceWait = nil
	item.BudgetConcurrencyWait = false

	attempt := activeAttempt(execution, item.ID)
	if attempt != nil {
		switch terminalState {
		case WorkItemStateCancelled:
			attempt.State = AttemptStateCancelled
		default:
			attempt.State = AttemptStateFailed
		}
		attempt.FinishedAt = &now
		attempt.Error = cloneWorkflowSanitizedError(failure)
		attempt.Receipt = nil
	}
	recordEffectSettlement(
		item,
		attempt,
		WorkExecutionResult{
			Error:     item.Error,
			Confirmed: true,
		},
	)

	released := r.releaseWorkResourcesLocked(execution, item.ID)
	r.wakeResourceWaitersLocked(execution, released)
	if advanceOwner {
		r.advanceSettledWorkLocked(
			session,
			execution,
			item.FrameID,
			item.ID,
		)

		execution = mutableSessionExecution(session, executionID)
		if execution == nil {
			return nil
		}
		item = mutableWorkItem(execution, workItemID)
		if item == nil {
			return execution
		}
	}
	r.wakeCompletionWorkWaitsLocked(session, execution, *item)

	execution, err := r.finalizeExecutionSettlementLocked(
		context.Background(),
		session,
		executionID,
		failure,
	)
	if err != nil || execution == nil {
		return nil
	}
	r.touchExecutionLocked(session, execution)
	return execution
}

// advanceSettledWorkLocked converts interpreter integrity failures into an
// inspectable terminal frame instead of discarding the error and stalling.
func (r *Runtime) advanceSettledWorkLocked(
	session *Session,
	execution *Execution,
	frameID ProgramFrameID,
	itemID WorkItemID,
) {
	if err := r.advanceFrameLocked(session, execution, frameID, ProgramEvent{
		Kind:       ProgramEventWorkSettled,
		WorkItemID: itemID,
	}); err != nil {
		frame := mutableFrame(execution, frameID)
		if frame == nil {
			return
		}
		frame.Status = ProgramFrameStateFailed
		frame.Error = &SanitizedError{
			Code:     "program_transition_failed",
			Category: ErrorCategoryExecution,
			Message:  err.Error(),
			StepPath: frame.Path,
		}
		frame.UpdatedAt = r.now()
		for frame.ParentFrameID != "" {
			parent := mutableFrame(execution, frame.ParentFrameID)
			if parent == nil {
				break
			}
			parent.Status = ProgramFrameStateFailed
			parent.Error = frame.Error
			parent.UpdatedAt = r.now()
			frame = parent
		}
	}
}
