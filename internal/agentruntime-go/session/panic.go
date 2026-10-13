package session

import (
	"context"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/backgroundutil"
)

// launchWork associates the common background panic boundary with the exact
// Run-owned work item. Failure handling completes before the tracked effect
// finishes, so Runtime.Close cannot race past unsettled panic recovery.
func (r *Runtime) launchWork(
	ctx context.Context,
	work runtimeWork,
) {
	r.launchEffectWithFailure(
		"session "+string(work.Kind),
		func(failure backgroundutil.PanicFailureError) {
			r.handleWorkPanic(ctx, work, failure)
		},
		func() {
			r.dispatchWork(ctx, work)
		},
	)
}

func (r *Runtime) handleWorkPanic(
	ctx context.Context,
	work runtimeWork,
	failure backgroundutil.PanicFailureError,
) {
	r.reportDiagnostic(ctx, RuntimeDiagnostic{
		SessionID: work.SessionID,
		RunID:     work.RunID,
		Operation: failure.Operation,
		Message:   "background execution panicked",
		Cause:     failure.Error(),
		At:        time.Now().UTC(),
	})

	switch work.Kind {
	case workModel:
		r.finishModelExecution(
			ctx,
			work,
			ModelExecutionResult{
				Error: &OperationError{
					Code:    backgroundutil.PanicFailureCode,
					Message: failure.Error(),
				},
			},
			failure,
		)

	case workToolEvaluate, workToolExecute:
		r.finishPanickedToolWork(ctx, work, failure)

	case workResolveWait, workExpireWait:
		r.finishPanickedWaitWork(ctx, work, failure)

	case workCloseSkill:
		// Cleanup failure is diagnostic-only. It must not rewrite an already
		// settled Run outcome or resurrect its private Skill session handle.
	default:
	}
}

func (r *Runtime) finishPanickedToolWork(
	ctx context.Context,
	work runtimeWork,
	failure backgroundutil.PanicFailureError,
) {
	r.mu.Lock()

	sessionValue, run, err := r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		r.mu.Unlock()
		return
	}

	call := findToolCall(run, work.ToolCallID)
	if call == nil {
		r.mu.Unlock()
		return
	}

	switch call.State {
	case ToolCallStateEvaluationQueued,
		ToolCallStateEvaluating,
		ToolCallStateExecutionQueued,
		ToolCallStateRunning:
	default:
		r.mu.Unlock()
		return
	}

	effects := runtimeEffects{}
	releaseDispatchCancel(
		r.cancels,
		toolEvaluationCancelKey(run.ID, call.ID),
		&effects,
	)

	// A panic after an external invocation started is not evidence that the
	// side effect failed. Preserve the attempt as uncertain and do not create
	// a selected result that continuation could consume.
	for index := range call.Attempts {
		attempt := &call.Attempts[index]
		if attempt.State != ToolAttemptStateRunning {
			continue
		}

		attempt.State = ToolAttemptStateUncertain
		attempt.Uncertain = true
		attempt.Error = failure.Error()
		attempt.FinishedAt = r.now()

		releaseDispatchCancel(
			r.cancels,
			toolExecutionCancelKey(run.ID, attempt.ID),
			&effects,
		)
	}

	if call.WaitID != "" {
		r.invalidatePanickedWaitLocked(
			sessionValue,
			run,
			call.WaitID,
			failure.Error(),
			&effects,
		)
	}

	call.State = ToolCallStateFailed
	call.LastFailure = failure.Error()
	call.Authorization = nil
	call.WaitID = ""
	removeRequestedCall(run, call.ID)
	r.touchRunLocked(sessionValue, run)

	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		"",
		RuntimeEventToolCallFailed,
		call.OriginOperationID,
		call.ID,
		"",
		"",
		ToolCallFailedEventData{Error: failure.Error()},
	)

	r.markRunTerminalLocked(
		sessionValue,
		run,
		"",
		panicRunOutcome(run, failure),
		&effects,
	)
	r.checkpointLocked(
		&effects,
		RuntimeEventToolCallFailed,
		sessionValue,
		run,
	)

	r.mu.Unlock()
	r.flushEffects(ctx, effects)
}

func (r *Runtime) finishPanickedWaitWork(
	ctx context.Context,
	work runtimeWork,
	failure backgroundutil.PanicFailureError,
) {
	r.mu.Lock()

	sessionValue, run, err := r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		r.mu.Unlock()
		return
	}

	wait := findWaitingRequest(run, work.WaitID)
	if wait == nil ||
		(wait.Status != WaitStatusPending &&
			wait.Status != WaitStatusResolving) {
		r.mu.Unlock()
		return
	}

	effects := runtimeEffects{}
	releaseDispatchCancel(
		r.cancels,
		waitResolutionCancelKey(run.ID, wait.ID),
		&effects,
	)

	r.invalidatePanickedWaitLocked(
		sessionValue,
		run,
		wait.ID,
		failure.Error(),
		&effects,
	)

	if call := findToolCall(run, wait.ToolCallID); call != nil {
		call.State = ToolCallStateFailed
		call.LastFailure = failure.Error()
		call.Authorization = nil
		call.WaitID = ""
		removeRequestedCall(run, call.ID)

		r.recordRunEventLocked(
			&effects,
			sessionValue,
			run,
			"",
			RuntimeEventToolCallFailed,
			call.OriginOperationID,
			call.ID,
			"",
			wait.ID,
			ToolCallFailedEventData{Error: failure.Error()},
		)
	}

	r.touchRunLocked(sessionValue, run)
	r.markRunTerminalLocked(
		sessionValue,
		run,
		"",
		panicRunOutcome(run, failure),
		&effects,
	)
	r.checkpointLocked(
		&effects,
		RuntimeEventWaitResolved,
		sessionValue,
		run,
	)

	r.mu.Unlock()
	r.flushEffects(ctx, effects)
}

func (r *Runtime) invalidatePanickedWaitLocked(
	sessionValue *Session,
	run *Run,
	waitID WaitID,
	message string,
	effects *runtimeEffects,
) {
	wait := findWaitingRequest(run, waitID)
	if wait == nil {
		return
	}

	switch wait.Status {
	case WaitStatusPending, WaitStatusResolving:
	default:
		return
	}

	wait.Status = WaitStatusInvalid
	wait.Error = message
	wait.UpdatedAt = r.now()
	r.clearWaitReferenceLocked(wait.ID)

	r.recordRunEventLocked(
		effects,
		sessionValue,
		run,
		"",
		RuntimeEventWaitResolved,
		wait.OperationID,
		wait.ToolCallID,
		"",
		wait.ID,
		wait,
	)
}

func panicRunOutcome(
	run *Run,
	failure backgroundutil.PanicFailureError,
) RunOutcome {
	kind := RunOutcomeFailed
	if run.Controls.CancellationRequested {
		kind = RunOutcomeCancelled
	}

	return RunOutcome{
		Kind:    kind,
		Code:    backgroundutil.PanicFailureCode,
		Message: failure.Error(),
	}
}

func releaseDispatchCancel(
	cancels map[string]context.CancelFunc,
	key string,
	effects *runtimeEffects,
) {
	cancel, found := cancels[key]
	if !found {
		return
	}

	delete(cancels, key)
	effects.cancels = append(effects.cancels, cancel)
}
