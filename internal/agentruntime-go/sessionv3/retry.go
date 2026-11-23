package sessionv3

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"
)

const defaultRetryInitialDelay = time.Second

func (r *Runtime) prepareRetryLocked(
	execution *Execution,
	item *WorkItem,
	attempt *Attempt,
	now time.Time,
) bool {
	if execution == nil ||
		item == nil ||
		attempt == nil ||
		item.State != WorkItemStateFailed ||
		item.Retry == nil ||
		item.Error == nil {
		return false
	}
	if (!retryEffectSafe(item.Effect) &&
		item.EffectState != EffectStateConfirmedNotApplied) ||
		!retryMatches(item.Retry.Match, *item.Error) ||
		attempt.Number >= item.Retry.MaxAttempts {
		return false
	}

	nextAttemptAt, err := retryAt(*item.Retry, item.ID, attempt.Number, now)
	if err != nil {
		item.Error = &SanitizedError{
			Code:     "retry_schedule_invalid",
			Category: ErrorCategoryValidation,
			Message:  err.Error(),
		}
		return false
	}
	if deadline := r.effectiveWorkDeadlineLocked(*execution, *item); deadline != nil &&
		!nextAttemptAt.Before(deadline.UTC()) {
		item.Error = deadlineError()
		return false
	}

	lastError := *item.Error
	lastError.Details = item.Error.Details.Clone()
	lastError.Causes = append([]SanitizedError(nil), item.Error.Causes...)

	item.State = WorkItemStateRetryWaiting
	item.Output = nil
	item.LLMStream = nil
	item.PortStreams = nil
	item.EffectState = EffectStateNotStarted
	item.CompletedAt = nil
	item.RetryState = &RetryState{
		ScheduledAt:   now,
		NextAttemptAt: nextAttemptAt,
		LastError:     &lastError,
	}
	item.UpdatedAt = now
	return true
}

func retryMatches(match RetryMatcher, failure SanitizedError) bool {
	if len(match.Categories) != 0 {
		found := slices.Contains(match.Categories, failure.Category)
		if !found {
			return false
		}
	}
	if len(match.Codes) != 0 {
		found := slices.Contains(match.Codes, failure.Code)
		if !found {
			return false
		}
	}
	if match.Retryable != nil && failure.Retryable != *match.Retryable {
		return false
	}
	if len(match.EffectStates) != 0 &&
		!slices.Contains(match.EffectStates, failure.Effect) {
		return false
	}
	if len(match.Schema) != 0 {
		value, err := canonicalWorkflowValue(failure)
		if err != nil {
			return false
		}
		if err := validateWorkflowSchema(match.Schema, value); err != nil {
			return false
		}
	}
	return true
}

func retryAt(
	policy RetryPolicy,
	workItemID WorkItemID,
	failedAttempt int,
	now time.Time,
) (time.Time, error) {
	initial := defaultRetryInitialDelay
	if policy.Backoff.Initial != "" {
		next, err := policy.Backoff.Initial.Deadline(now)
		if err != nil {
			return time.Time{}, err
		}
		initial = next.Sub(now)
	}
	if initial < 0 {
		return time.Time{}, errors.New("retry initial delay is negative")
	}

	multiplier := policy.Backoff.Multiplier
	if multiplier == 0 {
		multiplier = 2
	}
	delay := float64(initial) * math.Pow(multiplier, float64(max(failedAttempt-1, 0)))
	if math.IsInf(delay, 0) || math.IsNaN(delay) || delay > float64(math.MaxInt64) {
		return time.Time{}, errors.New("retry delay exceeds the supported clock range")
	}

	if policy.Backoff.Maximum != "" {
		maximumAt, err := policy.Backoff.Maximum.Deadline(now)
		if err != nil {
			return time.Time{}, err
		}
		maximumDelay := float64(maximumAt.Sub(now))
		if delay > maximumDelay {
			delay = maximumDelay
		}
	}

	if policy.Backoff.Jitter != 0 {
		sum := sha256.Sum256([]byte(
			string(workItemID) + "\x00" + fmt.Sprintf("%c", failedAttempt),
		))
		fraction := float64(binary.BigEndian.Uint64(sum[:8])) /
			float64(math.MaxUint64)
		factor := 1 - policy.Backoff.Jitter +
			2*policy.Backoff.Jitter*fraction
		delay *= factor
	}
	if delay < 0 || delay > float64(math.MaxInt64) {
		return time.Time{}, errors.New("retry delay exceeds the supported clock range")
	}

	return now.Add(time.Duration(delay)).UTC(), nil
}

func attemptDeadline(
	item WorkItem,
	now time.Time,
) (*time.Time, error) {
	deadline := cloneTime(item.Deadline)
	if item.Retry == nil || item.Retry.PerAttemptTimeout == "" {
		return deadline, nil
	}
	next, err := item.Retry.PerAttemptTimeout.Deadline(now)
	if err != nil {
		return nil, err
	}
	return earlierDeadline(deadline, &next), nil
}

func (r *Runtime) armRetryLocked(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	item WorkItem,
) {
	if r == nil ||
		item.ID == "" ||
		item.State != WorkItemStateRetryWaiting ||
		item.RetryState == nil {
		return
	}
	retryCtx, armed := r.armDueTimeLocked(retryDueTimeKey(item.ID))
	if !armed {
		return
	}

	retryAt := item.RetryState.NextAttemptAt.UTC()

	r.launchAsync(
		"session-v3 retry timer",
		func() {
			r.waitForRetry(
				retryCtx,
				context.WithoutCancel(ctx),
				sessionID,
				executionID,
				item.ID,
				retryAt,
			)
		},
		nil,
	)
}

func (r *Runtime) armExecutionRetriesLocked(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	execution Execution,
) {
	for _, item := range execution.WorkItems {
		r.armRetryLocked(ctx, sessionID, executionID, item)
	}
}

func (r *Runtime) cancelRetryLocked(id WorkItemID) {
	r.cancelDueTimeLocked(retryDueTimeKey(id))
}

func (r *Runtime) waitForRetry(
	ctx context.Context,
	persistContext context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	workItemID WorkItemID,
	at time.Time,
) {
	delay := at.Sub(r.now())
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return
	}
	session, execution, err := r.executionLocked(sessionID, executionID)
	if err != nil {
		return
	}
	item := mutableWorkItem(execution, workItemID)
	if item == nil ||
		item.State != WorkItemStateRetryWaiting ||
		item.RetryState == nil ||
		!item.RetryState.NextAttemptAt.Equal(at) {
		return
	}
	r.consumeDueTimeLocked(retryDueTimeKey(workItemID))

	now := r.now()
	if execution.State.Terminal() ||
		(execution.Controls.CancellationRequested &&
			!r.cleanupDispatchAllowedLocked(*execution, *item)) {
		failure := &SanitizedError{
			Code:     workflowCancelledBeforeInvocation,
			Category: ErrorCategoryCancellation,
			Message:  "retry was cancelled before external invocation",
		}
		execution = r.settleUndispatchedWorkLocked(
			session,
			execution,
			item,
			WorkItemStateCancelled,
			failure,
			true,
		)
		if execution == nil {
			return
		}
		_ = r.commitTerminalMutationLocked(
			persistContext,
			session,
			execution.ID,
			checkpointReasonRetryCancelled,
			RuntimeEventWorkChanged,
		)
		return
	}
	if deadline := r.effectiveWorkDeadlineLocked(*execution, *item); deadline != nil &&
		!now.Before(deadline.UTC()) {
		execution = r.settleUndispatchedWorkLocked(
			session,
			execution,
			item,
			WorkItemStateFailed,
			deadlineError(),
			true,
		)
		if execution == nil {
			return
		}
		_ = r.commitTerminalMutationLocked(
			persistContext,
			session,
			execution.ID,
			checkpointReasonRetryDeadlineElapsed,
			RuntimeEventWorkChanged,
		)
		return
	}

	item.State = WorkItemStateReady
	item.RetryState = nil
	item.UpdatedAt = now
	r.touchExecutionLocked(session, execution)
	if err := r.persistLocked(persistContext, "retryReady", session); err != nil {
		return
	}
	r.publishLocked(
		persistContext,
		session,
		execution.ID,
		RuntimeEventWorkChanged,
	)
}
