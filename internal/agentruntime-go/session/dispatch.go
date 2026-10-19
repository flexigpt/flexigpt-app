package session

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/backgroundutil"
)

var errStaleDispatch = errors.New("runtime dispatch became stale")

const waitOwnerMissingStr = "wait_owner_missing"

func (r *Runtime) dispatchWork(
	ctx context.Context,
	work runtimeWork,
) {
	switch work.Kind {
	case workModel:
		r.dispatchModel(ctx, work)

	case workToolEvaluate:
		r.dispatchToolEvaluation(ctx, work)

	case workToolExecute:
		r.dispatchToolExecution(ctx, work)

	case workResolveWait:
		r.dispatchWaitResolution(ctx, work)

	case workExpireWait:
		r.dispatchWaitExpiry(ctx, work)

	case workCloseSkill:
		r.dispatchSkillClose(ctx, work)

	default:
		r.reportDiagnostic(ctx, RuntimeDiagnostic{
			SessionID: work.SessionID,
			RunID:     work.RunID,
			Operation: "dispatch runtime work",
			Message:   "runtime work has an unsupported kind",
			Cause:     string(work.Kind),
			At:        r.now(),
		})
	}
}

func (r *Runtime) dispatchModel(
	ctx context.Context,
	work runtimeWork,
) {
	operationCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}

	sessionValue, run, err := r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		r.mu.Unlock()
		return
	}

	operation := findModelOperation(run, work.OperationID)
	if operation == nil ||
		operation.State != OperationStateQueued ||
		run.Controls.Paused ||
		run.Controls.CancellationRequested ||
		run.State.Terminal() {
		r.mu.Unlock()
		return
	}
	if operation.Automatic &&
		automationHoldApplies(
			run.Controls.AutomaticContinuation,
			operation.ID,
		) {
		effects := runtimeEffects{}
		if run.State != RunStateWaiting {
			run.State = RunStateWaiting
			r.touchRunLocked(sessionValue, run)
			r.recordRunEventLocked(
				&effects,
				sessionValue,
				run,
				"",
				RuntimeEventRunStateChanged,
				operation.ID,
				"",
				"",
				"",
				RunStateChangedEventData{State: RunStateWaiting},
			)
			r.checkpointLocked(
				&effects,
				RuntimeEventRunStateChanged,
				sessionValue,
				run,
			)
		}
		r.mu.Unlock()
		r.flushEffects(operationCtx, effects)
		return
	}

	operation.State = OperationStatePreparing
	r.cancels[modelCancelKey(run.ID, operation.ID)] = cancel
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		"",
		RuntimeEventModelOperationStarted,
		operation.ID,
		"",
		"",
		"",
		ModelOperationStateEventData{State: OperationStatePreparing},
	)
	r.checkpointLocked(
		&effects,
		RuntimeEventModelOperationStarted,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(operationCtx, effects)

	prepared, preparationErr := r.prepareModelOperationForDispatch(
		operationCtx,
		work,
	)
	if preparationErr != nil {
		if errors.Is(preparationErr, errStaleDispatch) {
			return
		}
		r.finishModelExecution(
			operationCtx,
			work,
			ModelExecutionResult{
				Error: &OperationError{
					Code:    "operation_preparation_failed",
					Message: preparationErr.Error(),
				},
			},
			preparationErr,
		)
		return
	}

	if err := r.ensureSkillSessionForModel(
		operationCtx,
		work.SessionID,
		work.RunID,
		work.OperationID,
	); err != nil {
		if errors.Is(err, errStaleDispatch) {
			return
		}

		r.finishModelExecution(
			operationCtx,
			work,
			ModelExecutionResult{
				Error: &OperationError{
					Code:    "skill_session_unavailable",
					Message: err.Error(),
				},
			},
			err,
		)
		return
	}

	r.mu.Lock()
	sessionValue, run, err = r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		r.mu.Unlock()
		return
	}

	operation = findModelOperation(run, work.OperationID)
	if operation != nil &&
		operation.State == OperationStatePreparing &&
		run.Controls.Paused &&
		!run.Controls.CancellationRequested &&
		!run.State.Terminal() {
		operation.State = OperationStateQueued
		delete(r.cancels, modelCancelKey(run.ID, operation.ID))
		r.touchRunLocked(sessionValue, run)
		effects = runtimeEffects{}
		r.checkpointLocked(
			&effects,
			RuntimeEventRunStateChanged,
			sessionValue,
			run,
		)
		r.mu.Unlock()
		r.flushEffects(operationCtx, effects)
		return
	}
	if operation == nil ||
		operation.State != OperationStatePreparing ||
		run.Controls.Paused ||
		run.Controls.CancellationRequested ||
		run.State.Terminal() {
		r.mu.Unlock()
		return
	}

	operation.State = OperationStateRunning
	operation.StartedAt = r.now()

	request := ModelExecutionRequest{
		SessionID:      sessionValue.ID,
		RunID:          run.ID,
		OperationID:    operation.ID,
		Configuration:  cloneJSON(operation.Request.Configuration),
		History:        cloneJSON(operation.Request.History),
		Current:        cloneJSON(operation.Request.Current),
		SkillSessionID: run.SkillSession.ID,
		Preparation:    cloneOperationPreparation(prepared),
	}
	r.touchRunLocked(sessionValue, run)

	effects = runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		"",
		RuntimeEventModelOperationStarted,
		operation.ID,
		"",
		"",
		"",
		ModelOperationStateEventData{State: OperationStateRunning},
	)
	r.checkpointLocked(
		&effects,
		RuntimeEventModelOperationStarted,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(operationCtx, effects)

	result, executeErr := r.deps.Model.ExecuteModel(
		operationCtx,
		request,
		ModelStream{
			OnText: func(text string) error {
				r.appendModelStream(
					operationCtx,
					work.SessionID,
					work.RunID,
					work.OperationID,
					text,
					false,
				)
				return operationCtx.Err()
			},
			OnThinking: func(thinking string) error {
				r.appendModelStream(
					operationCtx,
					work.SessionID,
					work.RunID,
					work.OperationID,
					thinking,
					true,
				)
				return operationCtx.Err()
			},
		},
	)

	r.finishModelExecution(
		operationCtx,
		work,
		result,
		executeErr,
	)
}

func (r *Runtime) ensureSkillSessionForModel(
	ctx context.Context,
	sessionID SessionID,
	runID RunID,
	operationID OperationID,
) error {
	r.mu.RLock()
	_, run, err := r.sessionAndRunLocked(
		sessionID,
		runID,
	)
	if err != nil {
		r.mu.RUnlock()
		return err
	}

	operation := findModelOperation(run, operationID)
	if operation == nil ||
		operation.State != OperationStatePreparing {
		r.mu.RUnlock()
		return errStaleDispatch
	}

	preparation := cloneOperationPreparation(operation.Preparation)
	generation := run.SkillSession.Generation
	r.mu.RUnlock()

	if err := RequirePreparedOperation(preparation); err != nil {
		return err
	}
	enabled := uniqueArtifactReferences(
		cloneJSON(preparation.EffectiveSkills),
	)

	r.mu.Lock()
	var sessionValue *Session
	sessionValue, run, err = r.sessionAndRunLocked(
		sessionID,
		runID,
	)
	if err != nil {
		r.mu.Unlock()
		return err
	}

	operation = findModelOperation(run, operationID)
	if operation == nil ||
		operation.State != OperationStatePreparing ||
		run.SkillSession.Generation != generation ||
		run.Controls.CancellationRequested {
		r.mu.Unlock()
		return errStaleDispatch
	}

	effects := runtimeEffects{}
	previousID := run.SkillSession.ID
	currentEnabled := uniqueArtifactReferences(run.SkillSession.Enabled)
	changed := false

	if !artifactReferencesEqual(currentEnabled, enabled) {
		run.SkillSession.Enabled = cloneJSON(enabled)
		run.SkillSession.Active = filterReferences(
			cloneJSON(preparation.ActiveSkills),
			enabled,
		)
		run.SkillSession.Generation++
		run.SkillSession.EstablishedGeneration = 0
		generation = run.SkillSession.Generation
		changed = true
	} else {
		active := filterReferences(
			cloneJSON(run.SkillSession.Active),
			enabled,
		)
		if !artifactReferencesEqual(run.SkillSession.Active, active) {
			run.SkillSession.Active = cloneJSON(active)
			changed = true
		}
	}

	if len(enabled) == 0 {
		if run.SkillSession.ID != "" ||
			run.SkillSession.EstablishedGeneration != 0 {
			run.SkillSession.ID = ""
			run.SkillSession.EstablishedGeneration = 0
			changed = true
		}

		if changed {
			r.touchRunLocked(sessionValue, run)
			r.recordRunEventLocked(
				&effects,
				sessionValue,
				run,
				"",
				RuntimeEventSkillSessionChanged,
				operationID,
				"",
				"",
				"",
				SkillSessionChangedEventData{
					SkillSessionID: "",
					ActiveSkills:   nil,
				},
			)
			r.checkpointLocked(
				&effects,
				RuntimeEventSkillSessionChanged,
				sessionValue,
				run,
			)
		}
		if previousID != "" {
			effects.work = append(effects.work, runtimeWork{
				Kind:      workCloseSkill,
				SessionID: sessionID,
				RunID:     runID,
				SkillID:   previousID,
			})
		}
		r.mu.Unlock()
		r.flushEffects(ctx, effects)
		return nil
	}

	if previousID != "" &&
		run.SkillSession.EstablishedGeneration == generation {
		r.mu.Unlock()
		r.flushEffects(ctx, effects)
		return nil
	}

	active := cloneJSON(run.SkillSession.Active)
	if changed {
		r.touchRunLocked(sessionValue, run)
		r.recordRunEventLocked(
			&effects,
			sessionValue,
			run,
			"",
			RuntimeEventSkillSessionChanged,
			operationID,
			"",
			"",
			"",
			SkillSessionChangedEventData{
				SkillSessionID: previousID,
				ActiveSkills:   cloneJSON(active),
			},
		)
		r.checkpointLocked(
			&effects,
			RuntimeEventSkillSessionChanged,
			sessionValue,
			run,
		)
	}
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	result, ensureErr := r.deps.Skills.EnsureSkillSession(
		ctx,
		SkillSessionRequest{
			SessionID:   previousID,
			Enabled:     cloneJSON(enabled),
			Active:      active,
			Preparation: cloneOperationPreparation(preparation),
		},
	)
	if ensureErr != nil {
		return ensureErr
	}
	if result.SessionID == "" {
		return errors.New(
			"skill-session controller returned an empty session ID",
		)
	}

	r.mu.Lock()
	sessionValue, run, err = r.sessionAndRunLocked(
		sessionID,
		runID,
	)
	if err != nil {
		r.mu.Unlock()
		if result.SessionID != previousID {
			r.closeSkillSession(
				context.WithoutCancel(ctx),
				result.SessionID,
				sessionID,
				runID,
			)
		}
		return err
	}

	operation = findModelOperation(run, operationID)
	if operation == nil ||
		operation.State != OperationStatePreparing ||
		run.SkillSession.Generation != generation ||
		run.Controls.CancellationRequested {
		r.mu.Unlock()
		if result.SessionID != previousID {
			r.closeSkillSession(
				context.WithoutCancel(ctx),
				result.SessionID,
				sessionID,
				runID,
			)
		}
		return errStaleDispatch
	}

	run.SkillSession.ID = result.SessionID
	run.SkillSession.Active = filterReferences(
		result.Active,
		run.SkillSession.Enabled,
	)
	run.SkillSession.EstablishedGeneration = generation
	r.touchRunLocked(sessionValue, run)

	effects = runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		"",
		RuntimeEventSkillSessionChanged,
		operationID,
		"",
		"",
		"",
		SkillSessionChangedEventData{
			SkillSessionID: result.SessionID,
			ActiveSkills:   cloneJSON(run.SkillSession.Active),
		},
	)
	r.checkpointLocked(
		&effects,
		RuntimeEventSkillSessionChanged,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return nil
}

func (r *Runtime) finishModelExecution(
	ctx context.Context,
	work runtimeWork,
	result ModelExecutionResult,
	executeErr error,
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

	operation := findModelOperation(run, work.OperationID)
	if operation == nil || operation.State.Terminal() {
		r.mu.Unlock()
		return
	}

	delete(r.streamCheckpointAt, operation.ID)
	delete(r.cancels, modelCancelKey(run.ID, operation.ID))
	operation.FinishedAt = r.now()
	run.ActiveModelOperationID = ""

	if executeErr != nil && result.Error == nil {
		result.Error = &OperationError{
			Code:    "model_execution_failed",
			Message: executeErr.Error(),
		}
	}

	if run.Controls.CancellationRequested &&
		isContextCancellationError(result.Error) {
		result.Error = &OperationError{
			Code:    string(RunStateCancelled),
			Message: "Generation was stopped by the user.",
		}
	}

	preferStreamedOutput := result.Error != nil ||
		run.Controls.CancellationRequested ||
		errors.Is(executeErr, context.Canceled)
	result.Text = canonicalModelOutput(
		result.Text,
		operation.StreamedText,
		preferStreamedOutput,
	)
	result.Thinking = canonicalModelOutput(
		result.Thinking,
		operation.StreamedThinking,
		preferStreamedOutput,
	)

	//nolint:gocritic // Not suitable for switch.
	if run.Controls.CancellationRequested ||
		errors.Is(executeErr, context.Canceled) {
		operation.State = OperationStateCancelled
		if result.Error == nil {
			operation.Error = &OperationError{
				Code:    string(RunStateCancelled),
				Message: "model operation was cancelled",
			}
		} else {
			operation.Error = cloneOperationError(result.Error)
		}
	} else if result.Error != nil &&
		!result.Usable &&
		len(result.ToolCalls) == 0 {
		operation.State = OperationStateFailed
		operation.Error = cloneOperationError(result.Error)
	} else {
		operation.State = OperationStateCompleted
		operation.Error = cloneOperationError(result.Error)
	}

	operation.Usable = result.Usable
	operation.ProtocolOutputs = cloneJSON(result.ProtocolOutputs)
	operation.Usage = cloneInferenceUsage(result.Usage)
	operation.DebugDetails = cloneRaw(result.DebugDetails)
	operation.DebugDetailsError = result.DebugDetailsError
	operation.Metadata = cloneRaw(result.Metadata)
	operation.RuntimeMetadata = cloneRaw(result.RuntimeMetadata)
	operation.Preparation.Private = nil
	operation.StreamedText = result.Text
	operation.StreamedThinking = result.Thinking
	operation.Preparation.Private = nil

	currentIndex := transcriptMessageIndex(
		sessionValue.Transcript,
		operation.Request.Current.ID,
	)
	if currentIndex >= 0 {
		if len(result.HydratedCurrentContext) != 0 {
			sessionValue.Transcript[currentIndex].InjectedContext = append(
				[]string(nil),
				result.HydratedCurrentContext...,
			)
			operation.Request.Current.InjectedContext = append(
				[]string(nil),
				result.HydratedCurrentContext...,
			)
		}
		if len(result.HydratedCurrentInputs) != 0 {
			sessionValue.Transcript[currentIndex].ProtocolInputs = cloneJSON(
				result.HydratedCurrentInputs,
			)
			sessionValue.Transcript[currentIndex].AttachmentsHydrated =
				sessionValue.Transcript[currentIndex].AttachmentsHydrated ||
					result.AttachmentsHydrated
		}
		if len(result.RuntimeMetadata) != 0 {
			sessionValue.Transcript[currentIndex].RuntimeMetadata = cloneRaw(
				result.RuntimeMetadata,
			)
		}
		if len(result.Metadata) != 0 {
			sessionValue.Transcript[currentIndex].Metadata = cloneRaw(
				result.Metadata,
			)
		}
	}

	if len(result.ProtocolOutputs) != 0 ||
		strings.TrimSpace(result.Text) != "" ||
		result.Error != nil ||
		len(result.ToolCalls) != 0 {
		status := "completed"
		switch operation.State {
		case OperationStateCancelled:
			status = "cancelled"
		case OperationStateInterrupted:
			status = "interrupted"
		case OperationStateFailed:
			status = "failed"
		default:
			if result.Error != nil {
				status = "failed"
			}
		}

		sessionValue.Transcript = append(
			sessionValue.Transcript,
			TranscriptMessage{
				ID:              r.newID(),
				CreatedAt:       r.now(),
				Role:            MessageRoleAssistant,
				Status:          status,
				Text:            result.Text,
				ProtocolOutputs: cloneJSON(result.ProtocolOutputs),
				OperationID:     operation.ID,
				Error:           cloneOperationError(result.Error),
				Metadata:        cloneRaw(result.Metadata),
				RuntimeMetadata: cloneRaw(result.RuntimeMetadata),
			},
		)
	}

	effects := runtimeEffects{}
	for _, proposal := range result.ToolCalls {
		call := r.registerToolCallLocked(
			sessionValue,
			run,
			operation,
			proposal,
			&effects,
		)
		operation.ToolCallIDs = append(
			operation.ToolCallIDs,
			call.ID,
		)
	}

	r.touchRunLocked(sessionValue, run)
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		"",
		RuntimeEventModelOperationFinished,
		operation.ID,
		"",
		"",
		"",
		ModelOperationFinishedEventData{
			State:  operation.State,
			Usable: result.Usable,
			Error:  cloneOperationError(result.Error),
		},
	)

	if run.Controls.CancellationRequested {
		r.markRunTerminalLocked(
			sessionValue,
			run,
			"",
			RunOutcome{
				Kind:    RunOutcomeCancelled,
				Code:    string(RunStateCancelled),
				Message: "run cancellation was requested",
			},
			&effects,
		)
	} else {
		r.advanceDriverLocked(
			sessionValue,
			run,
			"",
			DriverEvent{
				Kind:              DriverEventModelFinished,
				OperationID:       operation.ID,
				ModelUsable:       result.Usable,
				ModelHasToolCalls: len(result.ToolCalls) != 0,
			},
			&effects,
		)
	}

	r.checkpointLocked(
		&effects,
		RuntimeEventModelOperationFinished,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)
}

func (r *Runtime) appendModelStream(
	ctx context.Context,
	sessionID SessionID,
	runID RunID,
	operationID OperationID,
	value string,
	thinking bool,
) {
	if value == "" {
		return
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}

	sessionValue, run, err := r.sessionAndRunLocked(
		sessionID,
		runID,
	)
	if err != nil {
		r.mu.Unlock()
		return
	}

	operation := findModelOperation(run, operationID)
	if operation == nil ||
		(operation.State != OperationStatePreparing &&
			operation.State != OperationStateRunning) ||
		run.Controls.CancellationRequested {
		r.mu.Unlock()
		return
	}

	if thinking {
		operation.StreamedThinking += value
	} else {
		operation.StreamedText += value
	}
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		"",
		RuntimeEventModelOutputAppended,
		operationID,
		"",
		"",
		"",
		ModelOutputAppendedEventData{
			Thinking: thinking,
			Text:     value,
		},
	)
	if r.shouldCheckpointStreamLocked(operationID) {
		r.checkpointLocked(
			&effects,
			RuntimeEventModelOutputAppended,
			sessionValue,
			run,
		)
	}
	r.mu.Unlock()
	r.flushEffects(ctx, effects)
}

func (r *Runtime) registerToolCallLocked(
	sessionValue *Session,
	run *Run,
	operation *ModelOperation,
	proposal ToolCallProposal,
	effects *runtimeEffects,
) ToolCall {
	callID := proposal.ID
	if callID == "" || findToolCall(run, callID) != nil {
		callID = ToolCallID(r.newID())
	}

	call := ToolCall{
		ID:                callID,
		OriginOperationID: operation.ID,
		Source:            ToolCallSourceModel,
		ProviderCallID:    proposal.ProviderCallID,
		Name:              proposal.Name,
		ChoiceID:          proposal.ChoiceID,
		Type:              proposal.Type,
		Arguments:         cloneRaw(proposal.Arguments),
		Binding:           cloneJSON(proposal.Binding),
		Required:          proposal.Required,
		ProviderManaged:   proposal.ProviderManaged,
		Automatic:         proposal.Automatic,
		State:             ToolCallStatePending,
	}

	if call.ProviderManaged || !call.Required {
		call.State = ToolCallStateObserved
	}
	if call.Binding.Kind == ToolBindingKindUnresolved && call.Required {
		call.State = ToolCallStateFailed
		call.LastFailure = proposal.Error
		if call.LastFailure == "" {
			call.LastFailure = "model tool call could not be resolved to an execution binding"
		}
	}

	run.ToolCalls = append(run.ToolCalls, call)
	r.recordRunEventLocked(
		effects,
		sessionValue,
		run,
		"",
		RuntimeEventToolCallProposed,
		operation.ID,
		call.ID,
		"",
		"",
		call,
	)

	return call
}

func (r *Runtime) dispatchToolEvaluation(
	ctx context.Context,
	work runtimeWork,
) {
	evaluationCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}

	sessionValue, run, err := r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		r.mu.Unlock()
		return
	}

	call := findToolCall(run, work.ToolCallID)
	if call == nil ||
		call.State != ToolCallStateEvaluationQueued ||
		run.Controls.CancellationRequested ||
		run.State.Terminal() {
		r.mu.Unlock()
		return
	}

	if run.Controls.Paused ||
		hasPendingRecoveryReview(run) ||
		(work.Automatic &&
			automationHoldApplies(
				run.Controls.AutomaticTools,
				call.OriginOperationID,
			)) {
		changed := r.releaseQueuedToolWorkLocked(
			sessionValue,
			run,
			call,
		)
		effects := runtimeEffects{}
		if changed {
			if !run.Controls.Paused && !hasPendingRecoveryReview(run) {
				r.advanceDriverLocked(
					sessionValue,
					run,
					"",
					DriverEvent{Kind: DriverEventControlsChanged},
					&effects,
				)
			}
			r.checkpointLocked(
				&effects,
				RuntimeEventRunStateChanged,
				sessionValue,
				run,
			)
		}
		r.mu.Unlock()
		r.flushEffects(evaluationCtx, effects)
		return
	}

	call.State = ToolCallStateEvaluating
	r.cancels[toolEvaluationCancelKey(run.ID, call.ID)] = cancel
	r.touchRunLocked(sessionValue, run)

	request := ToolExecutionRequest{
		SessionID:      sessionValue.ID,
		RunID:          run.ID,
		ToolCall:       cloneJSON(*call),
		SkillSessionID: run.SkillSession.ID,
	}
	request.Authorization = cloneRaw(call.Authorization)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		"",
		RuntimeEventToolEvaluationStarted,
		call.OriginOperationID,
		call.ID,
		"",
		"",
		ToolEvaluationStartedEventData{Automatic: work.Automatic},
	)
	r.checkpointLocked(
		&effects,
		RuntimeEventToolEvaluationStarted,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(evaluationCtx, effects)

	evaluation, evaluationErr := r.deps.Tools.EvaluateTool(
		evaluationCtx,
		request,
	)

	r.mu.Lock()
	sessionValue, run, err = r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		r.mu.Unlock()
		return
	}

	call = findToolCall(run, work.ToolCallID)
	if call == nil {
		r.mu.Unlock()
		return
	}

	delete(r.cancels, toolEvaluationCancelKey(run.ID, call.ID))
	if call.State != ToolCallStateEvaluating {
		r.mu.Unlock()
		return
	}

	effects = runtimeEffects{}
	if evaluationErr != nil {
		call.State = ToolCallStateFailed
		call.LastFailure = evaluationErr.Error()
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
			ToolCallFailedEventData{Error: evaluationErr.Error()},
		)
		if call.Required {
			r.advanceDriverLocked(
				sessionValue,
				run,
				"",
				DriverEvent{
					Kind:        DriverEventToolSettled,
					OperationID: call.OriginOperationID,
					ToolCallID:  call.ID,
				},
				&effects,
			)
		}

		r.checkpointLocked(
			&effects,
			RuntimeEventToolCallFailed,
			sessionValue,
			run,
		)
		r.mu.Unlock()
		r.flushEffects(evaluationCtx, effects)
		return
	}

	advance := false
	switch evaluation.Decision {
	case ToolEvaluationWait:
		if evaluation.Wait == nil {
			call.State = ToolCallStateFailed
			call.LastFailure = "tool evaluator required a wait without wait details"
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
				ToolCallFailedEventData{Error: call.LastFailure},
			)
			advance = true
			break
		}

		wait := WaitingRequest{
			ID:                 WaitID(r.newID()),
			Kind:               evaluation.Wait.Kind,
			Status:             WaitStatusPending,
			OperationID:        call.OriginOperationID,
			ToolCallID:         call.ID,
			Summary:            cloneRaw(evaluation.Wait.Summary),
			InputSchema:        cloneRaw(evaluation.Wait.InputSchema),
			AllowedResolutions: append([]WaitResolution(nil), evaluation.Wait.AllowedResolutions...),
			CreatedAt:          r.now(),
			UpdatedAt:          r.now(),
			ExpiresAt:          cloneTime(evaluation.Wait.ExpiresAt),
		}
		call.State = ToolCallStateWaiting
		call.WaitID = wait.ID
		run.WaitingRequests = append(run.WaitingRequests, wait)
		if !run.Controls.Paused {
			run.State = RunStateWaiting
		}
		r.waitReferences[wait.ID] = waitReference{
			ExecutorReference: evaluation.Wait.ExecutorReference,
		}
		r.touchRunLocked(sessionValue, run)

		r.recordRunEventLocked(
			&effects,
			sessionValue,
			run,
			"",
			RuntimeEventWaitRequested,
			wait.OperationID,
			call.ID,
			"",
			wait.ID,
			wait,
		)

		r.scheduleWaitExpiryLocked(
			&effects,
			sessionValue.ID,
			run.ID,
			wait,
		)

	case ToolEvaluationDenied:
		result := ToolExecutionResult{
			HasResult: true,
			Kind:      ToolResultKindError,
			Text:      evaluation.Reason,
			Error:     evaluation.Reason,
		}
		if evaluation.Result != nil {
			result = *evaluation.Result
		}

		r.recordExecutionResultLocked(
			sessionValue,
			run,
			call,
			ToolAttemptID(""),
			result,
			&effects,
		)
		r.advanceDriverLocked(
			sessionValue,
			run,
			"",
			DriverEvent{
				Kind:        DriverEventToolSettled,
				OperationID: call.OriginOperationID,
				ToolCallID:  call.ID,
			},
			&effects,
		)

	case ToolEvaluationAllowed:
		call.Authorization = cloneRaw(evaluation.Authorization)
		call.State = ToolCallStateReady
		r.touchRunLocked(sessionValue, run)
		if call.Required {
			advance = true
		} else {
			r.queueAppToolExecutionLocked(
				sessionValue,
				run,
				call,
				&effects,
			)
		}

	default:
		call.State = ToolCallStateFailed
		call.LastFailure = "tool evaluator returned an unsupported decision"
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
			ToolCallFailedEventData{Error: call.LastFailure},
		)
		advance = true
	}

	if advance && call.Required {
		r.advanceDriverLocked(
			sessionValue,
			run,
			"",
			DriverEvent{
				Kind:        DriverEventToolSettled,
				OperationID: call.OriginOperationID,
				ToolCallID:  call.ID,
			},
			&effects,
		)
	}

	r.checkpointLocked(
		&effects,
		RuntimeEventToolEvaluationStarted,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(evaluationCtx, effects)
}

func (r *Runtime) dispatchToolExecution(
	ctx context.Context,
	work runtimeWork,
) {
	executionCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}

	sessionValue, run, err := r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		r.mu.Unlock()
		return
	}

	call := findToolCall(run, work.ToolCallID)
	if call == nil || call.State != ToolCallStateExecutionQueued {
		r.mu.Unlock()
		return
	}
	if run.Controls.CancellationRequested || run.State.Terminal() {
		r.mu.Unlock()
		return
	}
	if run.Controls.Paused ||
		hasPendingRecoveryReview(run) ||
		(work.Automatic &&
			automationHoldApplies(
				run.Controls.AutomaticTools,
				call.OriginOperationID,
			)) {
		changed := r.releaseQueuedToolWorkLocked(
			sessionValue,
			run,
			call,
		)
		effects := runtimeEffects{}
		if changed {
			if !run.Controls.Paused && !hasPendingRecoveryReview(run) {
				r.advanceDriverLocked(
					sessionValue,
					run,
					"",
					DriverEvent{Kind: DriverEventControlsChanged},
					&effects,
				)
			}
			r.checkpointLocked(
				&effects,
				RuntimeEventRunStateChanged,
				sessionValue,
				run,
			)
		}
		r.mu.Unlock()
		r.flushEffects(executionCtx, effects)
		return
	}

	if run.Limits.MaxToolAttempts > 0 &&
		run.Usage.ToolAttempts >= run.Limits.MaxToolAttempts {
		r.mu.Unlock()
		r.failToolWithoutResult(
			ctx,
			work,
			"run tool-attempt limit reached",
		)
		return
	}

	attempt := ToolAttempt{
		ID:        ToolAttemptID(r.newID()),
		CallID:    call.ID,
		State:     ToolAttemptStateRunning,
		Automatic: work.Automatic,
		StartedAt: r.now(),
	}
	call.Attempts = append(call.Attempts, attempt)
	call.State = ToolCallStateRunning
	removeRequestedCall(run, call.ID)
	run.Usage.ToolAttempts++
	r.cancels[toolExecutionCancelKey(run.ID, attempt.ID)] = cancel
	r.touchRunLocked(sessionValue, run)

	request := ToolExecutionRequest{
		SessionID:      sessionValue.ID,
		RunID:          run.ID,
		ToolCall:       cloneJSON(*call),
		Attempt:        cloneJSON(attempt),
		SkillSessionID: run.SkillSession.ID,
	}
	request.Authorization = cloneRaw(call.Authorization)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		"",
		RuntimeEventToolAttemptStarted,
		call.OriginOperationID,
		call.ID,
		attempt.ID,
		"",
		attempt,
	)
	r.checkpointLocked(
		&effects,
		RuntimeEventToolAttemptStarted,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(executionCtx, effects)

	result, executionErr := r.deps.Tools.ExecuteTool(
		executionCtx,
		request,
	)

	r.mu.Lock()
	sessionValue, run, err = r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		r.mu.Unlock()
		return
	}

	call = findToolCall(run, work.ToolCallID)
	if call == nil {
		r.mu.Unlock()
		return
	}

	attemptRecord := findToolAttempt(call, attempt.ID)
	delete(r.cancels, toolExecutionCancelKey(run.ID, attempt.ID))
	if attemptRecord == nil ||
		attemptRecord.State != ToolAttemptStateRunning {
		r.mu.Unlock()
		return
	}

	effects = runtimeEffects{}
	switch {
	case executionErr != nil && !result.HasResult:
		if run.Controls.CancellationRequested ||
			errors.Is(executionErr, context.Canceled) ||
			errors.Is(executionErr, context.DeadlineExceeded) {
			attemptRecord.State = ToolAttemptStateUncertain
			attemptRecord.Uncertain = true
		} else {
			attemptRecord.State = ToolAttemptStateFailed
		}

		attemptRecord.Error = executionErr.Error()
		attemptRecord.FinishedAt = r.now()
		call.State = ToolCallStateFailed
		call.LastFailure = executionErr.Error()
		call.Authorization = nil
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
			attempt.ID,
			"",
			ToolCallFailedEventData{Error: executionErr.Error()},
		)

	default:
		if executionErr != nil && result.Error == "" {
			result.Error = executionErr.Error()
		}

		attemptRecord.State = ToolAttemptStateCompleted
		attemptRecord.FinishedAt = r.now()

		r.recordExecutionResultLocked(
			sessionValue,
			run,
			call,
			attempt.ID,
			result,
			&effects,
		)
	}

	if run.Controls.CancellationRequested {
		r.markRunTerminalLocked(
			sessionValue,
			run,
			"",
			RunOutcome{
				Kind:    RunOutcomeCancelled,
				Code:    string(RunStateCancelled),
				Message: "run cancellation was requested",
			},
			&effects,
		)
	} else if call.Required {
		r.advanceDriverLocked(
			sessionValue,
			run,
			"",
			DriverEvent{
				Kind:        DriverEventToolSettled,
				OperationID: call.OriginOperationID,
				ToolCallID:  call.ID,
			},
			&effects,
		)
	}

	r.checkpointLocked(
		&effects,
		RuntimeEventToolResultRecorded,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(executionCtx, effects)
}

func (r *Runtime) dispatchWaitResolution(
	ctx context.Context,
	work runtimeWork,
) {
	resolutionCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}

	sessionValue, run, err := r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		r.mu.Unlock()
		return
	}

	wait := findWaitingRequest(run, work.WaitID)
	if wait == nil || wait.Status != WaitStatusResolving {
		r.mu.Unlock()
		return
	}

	call := findToolCall(run, wait.ToolCallID)
	reference, found := r.waitReferences[wait.ID]
	if call == nil || !found {
		r.mu.Unlock()
		r.markWaitInvalid(ctx, work)
		return
	}
	if run.Controls.CancellationRequested || run.State.Terminal() {
		r.mu.Unlock()
		return
	}

	request := ToolWaitResolutionRequest{
		SessionID:         sessionValue.ID,
		RunID:             run.ID,
		Wait:              cloneJSON(*wait),
		ToolCall:          cloneJSON(*call),
		Resolution:        work.Resolution,
		Input:             cloneRaw(work.Input),
		ExecutorReference: reference.ExecutorReference,
	}
	r.cancels[waitResolutionCancelKey(run.ID, wait.ID)] = cancel
	r.mu.Unlock()

	resolution, resolutionErr := r.deps.Tools.ResolveToolWait(
		resolutionCtx,
		request,
	)

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	sessionValue, run, err = r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		r.mu.Unlock()
		return
	}

	delete(r.cancels, waitResolutionCancelKey(run.ID, work.WaitID))
	wait = findWaitingRequest(run, work.WaitID)
	if wait == nil || wait.Status != WaitStatusResolving {
		r.mu.Unlock()
		return
	}

	call = findToolCall(run, wait.ToolCallID)
	if call == nil {
		wait.Status = WaitStatusInvalid
		wait.Error = "waiting request no longer has an owning tool call"
		wait.UpdatedAt = r.now()
		r.clearWaitReferenceLocked(wait.ID)

		effects := runtimeEffects{}
		r.recordRunEventLocked(
			&effects,
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
		r.markRunTerminalLocked(
			sessionValue,
			run,
			"",
			RunOutcome{
				Kind:    RunOutcomeFailed,
				Code:    waitOwnerMissingStr,
				Message: "waiting request no longer has an owning tool call",
			},
			&effects,
		)
		r.checkpointLocked(
			&effects,
			RuntimeEventRunFailed,
			sessionValue,
			run,
		)
		r.mu.Unlock()
		r.flushEffects(resolutionCtx, effects)
		return
	}

	effects := runtimeEffects{}
	if resolutionErr != nil {
		wait.Status = WaitStatusPending
		wait.Error = resolutionErr.Error()
		wait.UpdatedAt = r.now()
		r.touchRunLocked(sessionValue, run)

		r.recordRunEventLocked(
			&effects,
			sessionValue,
			run,
			"",
			RuntimeEventWaitRequested,
			wait.OperationID,
			wait.ToolCallID,
			"",
			wait.ID,
			wait,
		)
		r.checkpointLocked(
			&effects,
			RuntimeEventWaitRequested,
			sessionValue,
			run,
		)
		r.mu.Unlock()
		r.flushEffects(resolutionCtx, effects)
		return
	}

	wait.Status = WaitStatusResolved
	wait.Error = ""
	wait.UpdatedAt = r.now()
	r.clearWaitReferenceLocked(wait.ID)
	call.WaitID = ""

	switch resolution.Status {
	case ToolWaitResolutionApproved:
		call.Authorization = cloneRaw(resolution.Authorization)
		call.State = ToolCallStateReady
		if !call.Required {
			r.queueAppToolExecutionLocked(
				sessionValue,
				run,
				call,
				&effects,
			)
		}

	case ToolWaitResolutionDenied:
		result := ToolExecutionResult{
			HasResult: true,
			Kind:      ToolResultKindError,
			Text:      "tool execution was denied",
			Error:     "tool execution was denied",
		}
		if resolution.Result != nil {
			result = *resolution.Result
		}
		r.recordExecutionResultLocked(
			sessionValue,
			run,
			call,
			"",
			result,
			&effects,
		)

	default:
		call.State = ToolCallStateFailed
		call.LastFailure = "wait resolver returned an unsupported resolution state"
	}

	r.touchRunLocked(sessionValue, run)
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		"",
		RuntimeEventWaitResolved,
		wait.OperationID,
		call.ID,
		"",
		wait.ID,
		wait,
	)

	if call.Required {
		r.advanceDriverLocked(
			sessionValue,
			run,
			"",
			DriverEvent{
				Kind:        DriverEventToolWaitResolved,
				OperationID: call.OriginOperationID,
				ToolCallID:  call.ID,
			},
			&effects,
		)
	}
	r.checkpointLocked(
		&effects,
		RuntimeEventWaitResolved,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(resolutionCtx, effects)
}

func (r *Runtime) dispatchWaitExpiry(
	ctx context.Context,
	work runtimeWork,
) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}

	_, run, err := r.sessionAndRunLocked(
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
			wait.Status != WaitStatusResolving) ||
		wait.ExpiresAt == nil ||
		r.waitExpiryGenerations[wait.ID] != work.WaitExpiryGeneration {
		r.mu.Unlock()
		return
	}

	if r.waitExpirers == nil {
		r.waitExpirers = make(map[WaitID]context.CancelFunc)
	}
	if _, found := r.waitExpirers[wait.ID]; found {
		r.mu.Unlock()
		return
	}

	expiryCtx, cancel := context.WithCancel(r.lifecycleCtx)
	r.waitExpirers[wait.ID] = cancel
	deadline := wait.ExpiresAt.UTC()
	r.mu.Unlock()

	delay := deadline.Sub(r.now())
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-expiryCtx.Done():
			return
		case <-timer.C:
		}
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}

	var sessionValue *Session
	sessionValue, run, err = r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		r.mu.Unlock()
		return
	}

	wait = findWaitingRequest(run, work.WaitID)
	if wait == nil ||
		(wait.Status != WaitStatusPending &&
			wait.Status != WaitStatusResolving) ||
		wait.ExpiresAt == nil ||
		r.waitExpiryGenerations[wait.ID] != work.WaitExpiryGeneration {
		r.mu.Unlock()
		return
	}

	expiryMessage := "waiting request expired before a resolution was received"
	wait.Status = WaitStatusExpired
	wait.Error = expiryMessage
	wait.UpdatedAt = r.now()

	effects := runtimeEffects{}
	releaseDispatchCancel(
		r.cancels,
		waitResolutionCancelKey(run.ID, wait.ID),
		&effects,
	)
	r.clearWaitReferenceLocked(wait.ID)

	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		"",
		RuntimeEventWaitExpired,
		wait.OperationID,
		wait.ToolCallID,
		"",
		wait.ID,
		wait,
	)

	call := findToolCall(run, wait.ToolCallID)
	switch {
	case wait.Kind == WaitKindRecovery:
		r.markRunTerminalLocked(
			sessionValue,
			run,
			"",
			RunOutcome{
				Kind:    RunOutcomeFailed,
				Code:    "recovery_wait_expired",
				Message: expiryMessage,
			},
			&effects,
		)

	case call == nil:
		r.markRunTerminalLocked(
			sessionValue,
			run,
			"",
			RunOutcome{
				Kind:    RunOutcomeFailed,
				Code:    waitOwnerMissingStr,
				Message: "waiting request expired without an owning tool call",
			},
			&effects,
		)

	default:
		call.WaitID = ""
		r.recordExecutionResultLocked(
			sessionValue,
			run,
			call,
			"",
			ToolExecutionResult{
				HasResult: true,
				Kind:      ToolResultKindError,
				Text:      expiryMessage,
				Error:     expiryMessage,
			},
			&effects,
		)
		if call.Required {
			r.advanceDriverLocked(
				sessionValue,
				run,
				"",
				DriverEvent{
					Kind:        DriverEventToolSettled,
					OperationID: call.OriginOperationID,
					ToolCallID:  call.ID,
				},
				&effects,
			)
		}
	}

	r.checkpointLocked(
		&effects,
		RuntimeEventWaitExpired,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)
}

func (r *Runtime) markWaitInvalid(
	ctx context.Context,
	work runtimeWork,
) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}

	sessionValue, run, err := r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		r.mu.Unlock()
		return
	}

	wait := findWaitingRequest(run, work.WaitID)
	if wait == nil {
		r.mu.Unlock()
		return
	}

	wait.Status = WaitStatusInvalid
	wait.Error = "wait cannot be resolved after recovery because executor state is unavailable"
	wait.UpdatedAt = r.now()
	r.clearWaitReferenceLocked(wait.ID)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
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

	call := findToolCall(run, wait.ToolCallID)
	if call == nil {
		r.markRunTerminalLocked(
			sessionValue,
			run,
			"",
			RunOutcome{
				Kind:    RunOutcomeFailed,
				Code:    waitOwnerMissingStr,
				Message: "wait executor state was lost and the owning tool call no longer exists",
			},
			&effects,
		)
		r.checkpointLocked(
			&effects,
			RuntimeEventRunFailed,
			sessionValue,
			run,
		)
		r.mu.Unlock()
		r.flushEffects(ctx, effects)
		return
	}

	call.WaitID = ""
	call.Authorization = nil
	call.State = ToolCallStateFailed
	call.LastFailure = "wait cannot be resolved because executor state is unavailable"
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
		wait.ID,
		ToolCallFailedEventData{
			Error: call.LastFailure,
		},
	)
	if call.Required {
		r.advanceDriverLocked(
			sessionValue,
			run,
			"",
			DriverEvent{
				Kind:        DriverEventToolSettled,
				OperationID: call.OriginOperationID,
				ToolCallID:  call.ID,
			},
			&effects,
		)
	}
	r.checkpointLocked(
		&effects,
		RuntimeEventToolCallFailed,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)
}

func (r *Runtime) failToolWithoutResult(
	ctx context.Context,
	work runtimeWork,
	message string,
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

	call.State = ToolCallStateFailed
	call.LastFailure = message
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
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
		ToolCallFailedEventData{Error: message},
	)
	if call.Required {
		r.advanceDriverLocked(
			sessionValue,
			run,
			"",
			DriverEvent{
				Kind:        DriverEventToolSettled,
				OperationID: call.OriginOperationID,
				ToolCallID:  call.ID,
			},
			&effects,
		)
	}
	r.checkpointLocked(
		&effects,
		RuntimeEventToolCallFailed,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)
}

func (r *Runtime) recordExecutionResultLocked(
	sessionValue *Session,
	run *Run,
	call *ToolCall,
	attemptID ToolAttemptID,
	result ToolExecutionResult,
	effects *runtimeEffects,
) {
	if !result.HasResult {
		call.State = ToolCallStateFailed
		call.LastFailure = result.Error
		if call.LastFailure == "" {
			call.LastFailure = "tool execution ended without a usable result"
		}
		r.touchRunLocked(sessionValue, run)

		r.recordRunEventLocked(
			effects,
			sessionValue,
			run,
			"",
			RuntimeEventToolCallFailed,
			call.OriginOperationID,
			call.ID,
			attemptID,
			"",
			ToolCallFailedEventData{Error: call.LastFailure},
		)
		return
	}

	kind := result.Kind
	if kind == "" {
		if result.Error != "" {
			kind = ToolResultKindError
		} else {
			kind = ToolResultKindSuccess
		}
	}

	toolResult := ToolResult{
		ID:             ToolResultID(r.newID()),
		AttemptID:      attemptID,
		CallID:         call.ID,
		ProviderCallID: call.ProviderCallID,
		Name:           call.Name,
		ChoiceID:       call.ChoiceID,
		Type:           call.Type,
		Kind:           kind,
		Text:           result.Text,
		Contents:       cloneJSON(result.Contents),
		Structured:     cloneRaw(result.Structured),
		App:            cloneRaw(result.App),
		Error:          result.Error,
		CreatedAt:      r.now(),
	}

	if result.AppRoute != nil {
		r.recordAppInstanceLocked(sessionValue, run, call, &toolResult, result.AppRoute, effects)
	}

	if result.SkillSessionActiveKnown {
		run.SkillSession.Active = filterReferences(
			result.SkillSessionActive,
			run.SkillSession.Enabled,
		)
	}

	selectToolResult(call, toolResult)
	call.State = ToolCallStateResolved
	call.LastFailure = ""
	call.Authorization = nil
	removeRequestedCall(run, call.ID)
	r.touchRunLocked(sessionValue, run)

	r.recordRunEventLocked(
		effects,
		sessionValue,
		run,
		"",
		RuntimeEventToolResultRecorded,
		call.OriginOperationID,
		call.ID,
		attemptID,
		"",
		toolResult,
	)
}

func (r *Runtime) dispatchSkillClose(
	ctx context.Context,
	work runtimeWork,
) {
	if work.SkillID == "" {
		return
	}
	r.closeSkillSession(
		ctx,
		work.SkillID,
		work.SessionID,
		work.RunID,
	)
}

func (r *Runtime) closeSkillSession(
	ctx context.Context,
	skillID string,
	sessionID SessionID,
	runID RunID,
) {
	closeCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		skillSessionCloseTimeout,
	)
	defer cancel()

	var closeErr error
	backgroundutil.RunSafely(
		"Session Skill-session cleanup",
		func() {
			closeErr = r.deps.Skills.CloseSkillSession(closeCtx, skillID)
		},
		func(failure backgroundutil.PanicFailureError) {
			closeErr = failure
		},
	)
	if closeErr != nil {
		r.reportDiagnostic(ctx, RuntimeDiagnostic{
			SessionID: sessionID,
			RunID:     runID,
			Operation: "close skill session",
			Message:   "skill session close failed",
			Cause:     closeErr.Error(),
			At:        r.now(),
		})
	}
}

func findModelOperation(
	run *Run,
	id OperationID,
) *ModelOperation {
	for index := range run.ModelOperations {
		if run.ModelOperations[index].ID == id {
			return &run.ModelOperations[index]
		}
	}
	return nil
}

func findToolAttempt(
	call *ToolCall,
	id ToolAttemptID,
) *ToolAttempt {
	for index := range call.Attempts {
		if call.Attempts[index].ID == id {
			return &call.Attempts[index]
		}
	}
	return nil
}

func transcriptMessageIndex(
	messages []TranscriptMessage,
	id string,
) int {
	for index := range messages {
		if messages[index].ID == id {
			return index
		}
	}
	return -1
}

func cloneOperationError(
	value *OperationError,
) *OperationError {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cloneInferenceUsage(value *InferenceUsage) *InferenceUsage {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cloneTime(
	value *time.Time,
) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func canonicalModelOutput(
	finalValue string,
	streamedValue string,
	preferStreamed bool,
) string {
	if streamedValue == "" {
		return finalValue
	}
	if finalValue == "" {
		return streamedValue
	}
	if strings.HasPrefix(finalValue, streamedValue) {
		return finalValue
	}
	if strings.HasPrefix(streamedValue, finalValue) {
		return streamedValue
	}
	if preferStreamed {
		return streamedValue
	}
	return finalValue
}

func isContextCancellationError(value *OperationError) bool {
	if value == nil {
		return true
	}

	text := strings.ToLower(value.Code + "\n" + value.Message)
	return strings.Contains(text, "context canceled") ||
		strings.Contains(text, "context cancelled") ||
		strings.Contains(text, "request canceled") ||
		strings.Contains(text, "request cancelled") ||
		strings.Contains(text, "operation cancelled") ||
		strings.Contains(text, "operation canceled")
}

func modelCancelKey(
	runID RunID,
	operationID OperationID,
) string {
	return "model:" + string(runID) + ":" + string(operationID)
}

func toolEvaluationCancelKey(
	runID RunID,
	callID ToolCallID,
) string {
	return "tool-evaluation:" + string(runID) + ":" + string(callID)
}

func toolExecutionCancelKey(
	runID RunID,
	attemptID ToolAttemptID,
) string {
	return "tool-execution:" + string(runID) + ":" + string(attemptID)
}

func waitResolutionCancelKey(
	runID RunID,
	waitID WaitID,
) string {
	return "wait-resolution:" + string(runID) + ":" + string(waitID)
}
