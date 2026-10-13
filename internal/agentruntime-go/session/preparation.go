package session

import (
	"context"
	"errors"
	"fmt"
)

type ModelOperationPreparationRequest struct {
	SessionID            SessionID
	RunID                RunID
	OperationID          OperationID
	Configuration        SessionConfiguration
	History              []TranscriptMessage
	Current              TranscriptMessage
	ContinuationRevision Revision
}

func RequirePreparedOperation(
	value OperationPreparation,
) error {
	if value.State != PreparationStatePrepared || value.Private == nil {
		return fmt.Errorf(
			"prepared operation %q is unavailable",
			value.Fingerprint,
		)
	}
	return nil
}

type ModelOperationPreparer interface {
	PrepareModelOperation(
		ctx context.Context,
		request ModelOperationPreparationRequest,
	) (OperationPreparation, error)
}

type OperationPreparationFailureError struct {
	Code    string
	Message string
	Summary OperationResolutionSummary
	Cause   error
}

func (failure *OperationPreparationFailureError) Error() string {
	if failure == nil {
		return ""
	}
	if failure.Message != "" {
		return failure.Message
	}
	if failure.Cause != nil {
		return failure.Cause.Error()
	}
	return "model operation preparation failed"
}

func (failure *OperationPreparationFailureError) Unwrap() error {
	if failure == nil {
		return nil
	}
	return failure.Cause
}

type initialModelOperation struct {
	RunID       RunID
	OperationID OperationID
	Current     TranscriptMessage
	Prepared    OperationPreparation
}

func (r *Runtime) prepareInitialModelOperation(
	ctx context.Context,
	sessionID SessionID,
	runID RunID,
	operationID OperationID,
	currentMessageID string,
	configuration SessionConfiguration,
	transcript []TranscriptMessage,
	input UserInput,
) (initialModelOperation, error) {
	current := TranscriptMessage{
		ID:            currentMessageID,
		CreatedAt:     r.now(),
		Role:          MessageRoleUser,
		Status:        "accepted",
		Text:          input.Text,
		Attachments:   cloneJSON(input.Attachments),
		MCPAppContext: cloneJSON(input.MCPAppContext),
		Metadata:      cloneRaw(input.Metadata),
	}

	prospective := append(cloneJSON(transcript), current)
	history := selectHistory(
		prospective,
		current.ID,
		configuration.History,
	)

	prepared, err := r.deps.Operations.PrepareModelOperation(
		ctx,
		ModelOperationPreparationRequest{
			SessionID:     sessionID,
			RunID:         runID,
			OperationID:   operationID,
			Configuration: cloneJSON(configuration),
			History:       cloneJSON(history),
			Current:       cloneJSON(current),
		},
	)
	if err != nil {
		return initialModelOperation{}, err
	}
	if prepared.State != PreparationStatePrepared || prepared.Private == nil {
		return initialModelOperation{}, errors.New(
			"operation preparer returned an unusable initial operation",
		)
	}

	return initialModelOperation{
		RunID:       runID,
		OperationID: operationID,
		Current:     current,
		Prepared:    prepared,
	}, nil
}

func (r *Runtime) prepareModelOperationForDispatch(
	ctx context.Context,
	work runtimeWork,
) (OperationPreparation, error) {
	r.mu.RLock()
	sessionValue, run, err := r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		r.mu.RUnlock()
		return OperationPreparation{}, err
	}

	operation := findModelOperation(run, work.OperationID)
	if operation == nil || operation.State != OperationStatePreparing {
		r.mu.RUnlock()
		return OperationPreparation{}, errStaleDispatch
	}
	if operation.Preparation.State == PreparationStatePrepared {
		prepared := cloneOperationPreparation(operation.Preparation)
		r.mu.RUnlock()
		if prepared.Private == nil {
			return OperationPreparation{}, &OperationPreparationFailureError{
				Code:    "prepared_operation_unavailable",
				Message: "the operation must be prepared again after recovery",
				Summary: prepared.Resolution,
			}
		}
		return prepared, nil
	}

	request := ModelOperationPreparationRequest{
		SessionID:            sessionValue.ID,
		RunID:                run.ID,
		OperationID:          operation.ID,
		Configuration:        cloneJSON(operation.Request.Configuration),
		History:              cloneJSON(operation.Request.History),
		Current:              cloneJSON(operation.Request.Current),
		ContinuationRevision: operation.Request.ContinuationRevision,
	}
	r.mu.RUnlock()

	prepared, prepareErr := r.deps.Operations.PrepareModelOperation(
		ctx,
		request,
	)
	if prepareErr != nil {
		r.recordPreparationFailure(ctx, work, prepareErr)
		return OperationPreparation{}, prepareErr
	}
	if prepared.State != PreparationStatePrepared || prepared.Private == nil {
		invalid := errors.New("operation preparer returned an unusable prepared operation")
		r.recordPreparationFailure(ctx, work, invalid)
		return OperationPreparation{}, invalid
	}

	r.mu.Lock()
	sessionValue, run, err = r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		r.mu.Unlock()
		return OperationPreparation{}, err
	}

	operation = findModelOperation(run, work.OperationID)
	if operation == nil ||
		operation.State != OperationStatePreparing ||
		run.Controls.CancellationRequested ||
		run.State.Terminal() {
		r.mu.Unlock()
		return OperationPreparation{}, errStaleDispatch
	}

	operation.Preparation = prepared
	run.SkillSession.Generation++
	run.SkillSession.EstablishedGeneration = 0
	run.SkillSession.Enabled = cloneJSON(prepared.EffectiveSkills)
	run.SkillSession.Active = filterReferences(
		prepared.ActiveSkills,
		prepared.EffectiveSkills,
	)
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		"",
		RuntimeEventModelOperationPrepared,
		operation.ID,
		"",
		"",
		"",
		ModelOperationPreparedEventData{
			Fingerprint: prepared.Fingerprint,
			Resolution:  cloneJSON(prepared.Resolution),
		},
	)
	r.checkpointLocked(
		&effects,
		RuntimeEventModelOperationPrepared,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return cloneOperationPreparation(prepared), nil
}

func (r *Runtime) recordPreparationFailure(
	_ context.Context,
	work runtimeWork,
	cause error,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sessionValue, run, err := r.sessionAndRunLocked(
		work.SessionID,
		work.RunID,
	)
	if err != nil {
		return
	}

	operation := findModelOperation(run, work.OperationID)
	if operation == nil || operation.State != OperationStatePreparing {
		return
	}

	if failure, ok := errors.AsType[*OperationPreparationFailureError](cause); ok {
		operation.Preparation = OperationPreparation{
			State:      PreparationStateFailed,
			Resolution: cloneJSON(failure.Summary),
		}
	} else {
		operation.Preparation = OperationPreparation{
			State: PreparationStateFailed,
			Resolution: OperationResolutionSummary{
				Version: 1,
				Diagnostics: []ResolutionDiagnostic{{
					Severity: DiagnosticSeverityError,
					Code:     "operation_preparation_failed",
					Message:  cause.Error(),
				}},
			},
		}
	}

	r.touchRunLocked(sessionValue, run)
}

func cloneOperationPreparation(
	value OperationPreparation,
) OperationPreparation {
	return OperationPreparation{
		State:           value.State,
		Fingerprint:     value.Fingerprint,
		PreparedAt:      value.PreparedAt,
		Resolution:      cloneJSON(value.Resolution),
		EffectiveSkills: cloneJSON(value.EffectiveSkills),
		ActiveSkills:    cloneJSON(value.ActiveSkills),
		Private:         value.Private,
	}
}
