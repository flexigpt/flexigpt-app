package sessionv3

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func effectReplaySafe(effect EffectContract) bool {
	switch effect.Kind {
	case EffectKindRead, EffectKindIdempotent, EffectKindKeyed:
		return true
	default:
		return false
	}
}

// dispatchAllowedLocked is used both when work is queued and immediately
// before binding. A failed checkpoint or recovery review prevents new work.
func (r *Runtime) dispatchAllowedLocked(execution Execution) bool {
	if r.checkpointPending[execution.SessionID] ||
		execution.Controls.Paused ||
		execution.Controls.CancellationRequested ||
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

// resumeRestoredWork starts only after the complete recovery source has
// returned successfully. Thus no restored execution can race an incomplete
// restore of the same Session document set.
func (r *Runtime) resumeRestoredWork(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	for _, session := range r.sessions {
		if r.checkpointPending[session.ID] {
			continue
		}
		for index := range session.Executions {
			execution := &session.Executions[index]
			if execution.State.Terminal() {
				continue
			}
			for _, wait := range execution.Waits {
				if wait.Status == WaitStatusPending {
					r.scheduleWaitDeadlineLocked(ctx, session.ID, execution.ID, wait)
				}
			}
			r.launchWork(ctx, session.ID, execution.ID, r.readyWorkLocked(*execution))
		}
	}
}

func validateWaitResponse(wait Wait, response CanonicalValue) error {
	if !response.Valid() {
		return errors.New("wait response is not canonical json")
	}
	value := response
	if len(value) == 0 {
		value = MustCanonicalValue(nil)
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
	wait.Response = MustCanonicalValue(TimerWaitOutput{
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
	if err := r.persistLocked(ctx, "timerFired", session); err != nil {
		return true
	}
	r.publishLocked(ctx, session, execution.ID, RuntimeEventWaitChanged)
	r.launchWork(ctx, session.ID, execution.ID, r.readyWorkLocked(*execution))
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
			Code:     "cancelled_before_invocation",
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
	if execution.Controls.CancellationRequested && !hasDispatched(*execution) {
		r.completeExecutionLocked(ctx, session, execution, ExecutionOutcome{
			Kind:        ExecutionOutcomeCancelled,
			Code:        string(ExecutionOutcomeCancelled),
			Message:     "execution was cancelled",
			CompletedAt: r.now(),
		})
	}
	if err := r.persistLocked(ctx, "workNotInvoked", session); err != nil {
		return
	}
	r.publishLocked(ctx, session, execution.ID, RuntimeEventWorkChanged)
	r.launchWork(ctx, session.ID, execution.ID, r.readyWorkLocked(*execution))
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
