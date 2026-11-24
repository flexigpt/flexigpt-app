package sessionv3

import (
	"context"
	"errors"
)

func (r *Runtime) acceptWaitResolution(
	ctx context.Context,
	command ResolveWaitCommand,
) (*CommandReceipt, error) {
	if command.Meta.ID == "" {
		return nil, errors.New(errCommandIDRequired)
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}
	session, execution, err := r.executionLocked(command.SessionID, command.ExecutionID)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if receipt, found, receiptErr := r.commandReceiptLocked(
		session,
		command.Meta,
		command,
	); receiptErr != nil ||
		found {
		r.mu.Unlock()
		return receipt, receiptErr
	}
	if message := expectedExecutionRevision(session, execution, command.Meta); message != "" {
		receipt := r.rejectLocked(session, command.Meta, command, message)
		r.mu.Unlock()
		return receipt, nil
	}
	wait := mutableWait(execution, command.WaitID)
	if wait == nil || wait.Status != WaitStatusPending {
		receipt := r.rejectLocked(session, command.Meta, command, "wait is not pending")
		r.mu.Unlock()
		return receipt, nil
	}
	if err := humanWaitActorAuthorized(*wait, command.Actor); err != nil {
		receipt := r.rejectLocked(session, command.Meta, command, err.Error())
		r.mu.Unlock()
		return receipt, nil
	}
	if !containsResolution(wait.Request.AllowedResolutions, command.Resolution) {
		receipt := r.rejectLocked(session, command.Meta, command, "wait resolution is not allowed")
		r.mu.Unlock()
		return receipt, nil
	}
	if err := validateWaitResponse(*wait, command.Response); err != nil {
		receipt := r.rejectLocked(session, command.Meta, command, err.Error())
		r.mu.Unlock()
		return receipt, nil
	}

	previous := cloneSession(*session)
	now := r.now()
	wait.Status = WaitStatusResolving
	wait.ResolvedBy = cloneHumanActor(command.Actor)
	wait.UpdatedAt = now
	r.cancelWaitDeadlineLocked(wait.ID)
	r.touchExecutionLocked(session, execution)
	receipt := CommandReceipt{
		CommandID:         command.Meta.ID,
		Accepted:          true,
		SessionID:         session.ID,
		ExecutionID:       execution.ID,
		SessionRevision:   session.Revision,
		ExecutionRevision: execution.Revision,
		WaitIDs:           []WaitID{wait.ID},
	}
	wait.ResolutionIntent = &WaitResolutionIntent{
		Meta:        command.Meta,
		SessionID:   session.ID,
		ExecutionID: execution.ID,
		WaitID:      wait.ID,
		Resolution:  command.Resolution,
		Response:    command.Response.Clone(),
		Actor:       cloneHumanActor(command.Actor),
		Receipt:     receipt,
		RequestedAt: now,
	}
	r.rememberCommandLocked(session, command.Meta, command, receipt)
	if err := r.persistLocked(ctx, checkpointReasonWaitResolutionRequested, session); err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	r.publishLocked(ctx, session, execution.ID, RuntimeEventWaitChanged)
	r.mu.Unlock()

	r.launchAcceptedWaitResolution(ctx, command.SessionID, command.ExecutionID, command.WaitID)
	return &receipt, nil
}

func (r *Runtime) launchAcceptedWaitResolution(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	waitID WaitID,
) {
	r.launchAsync("session-v3 wait resolution", func() {
		r.resolveAcceptedWaitResolution(ctx, sessionID, executionID, waitID)
	}, nil)
}

func (r *Runtime) resolveAcceptedWaitResolution(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	waitID WaitID,
) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	session, execution, err := r.executionLocked(sessionID, executionID)
	if err != nil {
		r.mu.Unlock()
		return
	}
	wait := mutableWait(execution, waitID)
	if wait == nil || wait.Status != WaitStatusResolving || wait.ResolutionIntent == nil {
		r.mu.Unlock()
		return
	}
	intent := cloneWaitResolutionIntent(wait.ResolutionIntent)
	if wait.Request.Kind == WaitKindRecovery &&
		!workflowReconciliationReviewWait(execution, *wait) {
		r.resolveAcceptedRecoveryWaitLocked(ctx, session, execution, wait, *intent)
		r.mu.Unlock()
		return
	}
	item := mutableWorkItem(execution, wait.WorkItemID)
	if item == nil {
		r.mu.Unlock()
		return
	}
	resolutionCtx, cancel := context.WithCancel(r.lifecycleCtx)
	if r.waitResolutionCancels == nil {
		r.waitResolutionCancels = make(map[WaitID]context.CancelFunc)
	}
	r.waitResolutionCancels[wait.ID] = cancel
	request := WaitResolutionRequest{
		Session:    cloneSession(*session),
		Execution:  cloneExecution(*execution),
		WorkItem:   *item,
		Wait:       *wait,
		Resolution: intent.Resolution,
		Response:   intent.Response.Clone(),
		Private:    r.privateWaits[wait.ID],
	}
	r.mu.Unlock()

	result, resolveErr := r.deps.Binder.ResolveWait(resolutionCtx, request)
	cancel()

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.waitResolutionCancels, waitID)
	if r.closed {
		return
	}
	session, execution, err = r.executionLocked(sessionID, executionID)
	if err != nil {
		return
	}
	wait = mutableWait(execution, waitID)
	if wait == nil || wait.Status != WaitStatusResolving || wait.ResolutionIntent == nil {
		return
	}
	item = mutableWorkItem(execution, wait.WorkItemID)
	if item == nil {
		return
	}
	if resolutionCtx.Err() != nil {
		return
	}

	now := r.now()
	if resolveErr != nil {
		wait.Status = WaitStatusPending
		wait.Error = ClassifyDomainError(
			resolveErr,
			"wait_resolution_failed",
			ErrorCategoryExecution,
		)
		wait.UpdatedAt = now
		r.touchExecutionLocked(session, execution)
		if r.persistLocked(ctx, checkpointReasonWaitResolutionFailed, session) == nil &&
			wait.Request.Deadline != nil {
			r.scheduleWaitDeadlineLocked(ctx, session.ID, execution.ID, *wait)
		}
		r.publishLocked(ctx, session, execution.ID, RuntimeEventWaitChanged)
		return
	}

	wait.Status = WaitStatusResolved
	wait.Resolution = intent.Resolution
	wait.Response = intent.Response.Clone()
	wait.Error = nil
	wait.UpdatedAt = now
	wait.ResolvedBy = cloneHumanActor(intent.Actor)
	delete(r.privateWaits, wait.ID)
	item.WaitID = ""
	switch {
	case result.Output != nil:
		r.finishWorkLocked(session, execution, item, activeAttempt(execution, item.ID),
			WorkExecutionResult{
				Output: result.Output, Error: result.Error, Confirmed: result.Confirmed,
				EffectState: result.EffectState,
			})
	case result.Resume && result.Bound != nil:
		r.privateBindings[item.ID] = *result.Bound
		item.State = WorkItemStateReady
		item.Requested = true
		item.Binding = &result.Bound.SemanticBinding
		item.UpdatedAt = now
	default:
		failure := result.Error
		if failure == nil {
			failure = &SanitizedError{
				Code: "wait_resolution_invalid", Message: "wait resolver returned no resume or result",
			}
		}
		r.finishWorkLocked(session, execution, item, activeAttempt(execution, item.ID),
			WorkExecutionResult{Error: failure, Confirmed: true})
	}
	execution, err = r.finalizeExecutionSettlementLocked(ctx, session, executionID, nil)
	if err != nil || execution == nil {
		return
	}
	r.touchExecutionLocked(session, execution)
	if r.persistLocked(ctx, checkpointReasonWaitResolved, session) != nil {
		return
	}
	r.armSessionExecutionTimersLocked(ctx, session)
	r.publishLocked(ctx, session, execution.ID, RuntimeEventWaitChanged)
}

func (r *Runtime) resolveAcceptedRecoveryWaitLocked(
	ctx context.Context,
	session *Session,
	execution *Execution,
	wait *Wait,
	intent WaitResolutionIntent,
) {
	now := r.now()
	wait.Status = WaitStatusResolved
	wait.Resolution = intent.Resolution
	wait.Response = intent.Response.Clone()
	wait.ResolvedBy = cloneHumanActor(intent.Actor)
	wait.Error = nil
	wait.UpdatedAt = now
	delete(r.privateWaits, wait.ID)
	if intent.Resolution == WaitResolutionCancel {
		r.cancelIndependentExecutionLocked(
			ctx,
			session,
			execution,
			"execution was cancelled during recovery review",
		)
	} else if err := r.settleRecoveredReviewWorkLocked(session, execution, wait.WorkItemID); err != nil {
		wait.Status = WaitStatusPending
		wait.Error = ClassifyDomainError(err, "recovery_resolution_failed", ErrorCategoryExecution)
		wait.UpdatedAt = now
	}
	execution, err := r.finalizeExecutionSettlementLocked(ctx, session, execution.ID, nil)
	if err != nil || execution == nil {
		return
	}
	r.touchExecutionLocked(session, execution)
	if r.persistLocked(ctx, checkpointReasonRecoveryWaitResolved, session) != nil {
		return
	}
	r.armSessionExecutionTimersLocked(ctx, session)
	r.publishLocked(ctx, session, execution.ID, RuntimeEventWaitChanged)
}

func (r *Runtime) cancelWaitResolutionLocked(waitID WaitID) {
	if cancel := r.waitResolutionCancels[waitID]; cancel != nil {
		delete(r.waitResolutionCancels, waitID)
		cancel()
	}
}

func (r *Runtime) resumeAcceptedWaitResolutionsLocked(
	ctx context.Context,
	session *Session,
	execution *Execution,
) {
	if session == nil || execution == nil {
		return
	}
	for _, wait := range execution.Waits {
		if wait.Status != WaitStatusResolving ||
			wait.ResolutionIntent == nil ||
			wait.Request.RuntimeBound {
			continue
		}
		r.launchAcceptedWaitResolution(ctx, session.ID, execution.ID, wait.ID)
	}
}
