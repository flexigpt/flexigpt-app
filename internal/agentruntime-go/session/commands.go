//nolint:nilerr // Ok.
package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/backgroundutil"
)

const (
	toolCallNotFound = "tool call was not found"
)

func (r *Runtime) CreateSession(
	ctx context.Context,
	command CreateSessionCommand,
) (*CommandReceipt, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		command.Meta,
		command,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionID := command.SessionID
	if sessionID == "" {
		sessionID = SessionID(r.newID())
	}
	if _, retired := r.retiredSessionIDs[sessionID]; retired {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: sessionID,
				Message:   "session ID was previously deleted and cannot be reused",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if _, exists := r.sessions[sessionID]; exists {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: sessionID,
				Message:   "session already exists",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	r.mu.Unlock()

	preparedConfiguration, preparationErr := r.prepareConfiguration(
		ctx,
		sessionID,
		command.Configuration,
	)

	r.mu.Lock()
	if r.closed {
		delete(r.commandReservations, command.Meta.ID)
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}
	if _, retired := r.retiredSessionIDs[sessionID]; retired {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: sessionID,
				Message:   "session ID was previously deleted and cannot be reused",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if _, exists := r.sessions[sessionID]; exists {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: sessionID,
				Message:   "session already exists",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if preparationErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: sessionID,
				Message:   preparationErr.Error(),
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	now := r.now()
	sessionValue := &Session{
		ID:            sessionID,
		Revision:      1,
		Title:         strings.TrimSpace(command.Title),
		CreatedAt:     now,
		UpdatedAt:     now,
		Configuration: cloneJSON(preparedConfiguration),
	}
	r.sessions[sessionID] = sessionValue

	receipt := CommandReceipt{
		CommandID:       command.Meta.ID,
		Accepted:        true,
		SessionID:       sessionID,
		SessionRevision: sessionValue.Revision,
	}
	r.finishCommandLocked(command.Meta, fingerprint, receipt)

	effects := runtimeEffects{}
	r.recordSessionEventLocked(
		&effects,
		sessionValue,
		command.Meta.ID,
		RuntimeEventSessionCreated,
		sessionValue,
	)
	r.checkpointLocked(
		&effects,
		RuntimeEventSessionCreated,
		sessionValue,
		nil,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func (r *Runtime) DeleteSession(
	ctx context.Context,
	command DeleteSessionCommand,
) (*CommandReceipt, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		command.Meta,
		command,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionValue := r.sessions[command.SessionID]
	if sessionValue == nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				Message:   ErrSessionNotFound.Error(),
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if message := expectedSessionRevisionMessage(
		sessionValue,
		command.Meta.ExpectedSessionRevision,
	); message != "" {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message:         message,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if active := r.activeRunLocked(sessionValue); active != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           active.ID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     active.Revision,
				Message:         "an active run must be cancelled or completed before deleting its session",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	effects := runtimeEffects{}
	r.recordSessionEventLocked(
		&effects,
		sessionValue,
		command.Meta.ID,
		RuntimeEventSessionDeleted,
		nil,
	)

	for _, runID := range sessionValue.RunIDs {
		run := r.runs[runID]
		if run == nil {
			continue
		}

		r.cancelOutstandingWaitsLocked(run, "session was deleted")
		if skillID := run.SkillSession.ID; skillID != "" {
			run.SkillSession.ID = ""
			effects.work = append(effects.work, runtimeWork{
				Kind:      workCloseSkill,
				SessionID: sessionValue.ID,
				RunID:     run.ID,
				SkillID:   skillID,
			})
		}

		for key, cancel := range r.cancels {
			if isRunDispatchCancelKey(key, run.ID) {
				effects.cancels = append(effects.cancels, cancel)
				delete(r.cancels, key)
			}
		}

		delete(r.runs, run.ID)
		delete(r.runToSession, run.ID)
	}

	delete(r.sessions, sessionValue.ID)
	if r.retiredSessionIDs == nil {
		r.retiredSessionIDs = make(map[SessionID]struct{})
	}
	r.retiredSessionIDs[sessionValue.ID] = struct{}{}
	effects.deletedSessions = append(effects.deletedSessions, sessionValue.ID)

	receipt := CommandReceipt{
		CommandID:       command.Meta.ID,
		Accepted:        true,
		SessionID:       sessionValue.ID,
		SessionRevision: sessionValue.Revision,
	}
	r.finishCommandLocked(command.Meta, fingerprint, receipt)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func (r *Runtime) UpdateSessionConfiguration(
	ctx context.Context,
	command UpdateSessionConfigurationCommand,
) (*CommandReceipt, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		command.Meta,
		command,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionValue := r.sessions[command.SessionID]
	if sessionValue == nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				Message:   ErrSessionNotFound.Error(),
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if message := expectedSessionRevisionMessage(
		sessionValue,
		command.Meta.ExpectedSessionRevision,
	); message != "" {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message:         message,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	configuration := cloneJSON(command.Configuration)
	sessionRevisionBeforePreparation := sessionValue.Revision
	r.mu.Unlock()

	preparedConfiguration, preparationErr := r.prepareConfiguration(
		ctx,
		command.SessionID,
		configuration,
	)

	r.mu.Lock()
	if r.closed {
		delete(r.commandReservations, command.Meta.ID)
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	sessionValue = r.sessions[command.SessionID]
	if sessionValue == nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				Message:   ErrSessionNotFound.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(receipt)
	}
	if sessionValue.Revision != sessionRevisionBeforePreparation {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message: "session changed while configuration was being prepared; " +
					"retry update",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if preparationErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:          command.Meta.ID,
				SessionID:          command.SessionID,
				SessionRevision:    sessionValue.Revision,
				Message:            preparationErr.Error(),
				PreparationFailure: preparationFailureDetails(preparationErr),
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	sessionValue.Configuration = cloneJSON(preparedConfiguration)
	r.touchSessionLocked(sessionValue)

	receipt := CommandReceipt{
		CommandID:       command.Meta.ID,
		Accepted:        true,
		SessionID:       command.SessionID,
		SessionRevision: sessionValue.Revision,
	}
	r.finishCommandLocked(command.Meta, fingerprint, receipt)

	effects := runtimeEffects{}
	r.recordSessionEventLocked(
		&effects,
		sessionValue,
		command.Meta.ID,
		RuntimeEventSessionConfigurationChanged,
		sessionValue.Configuration,
	)
	r.checkpointLocked(
		&effects,
		RuntimeEventSessionConfigurationChanged,
		sessionValue,
		nil,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func (r *Runtime) RenameSession(
	ctx context.Context,
	command RenameSessionCommand,
) (*CommandReceipt, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		command.Meta,
		command,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionValue := r.sessions[command.SessionID]
	if sessionValue == nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				Message:   ErrSessionNotFound.Error(),
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if message := expectedSessionRevisionMessage(
		sessionValue,
		command.Meta.ExpectedSessionRevision,
	); message != "" {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message:         message,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	title := strings.TrimSpace(command.Title)
	effects := runtimeEffects{}

	if sessionValue.Title != title {
		sessionValue.Title = title
		r.touchSessionLocked(sessionValue)
		r.recordSessionEventLocked(
			&effects,
			sessionValue,
			command.Meta.ID,
			RuntimeEventSessionRenamed,
			SessionRenamedEventData{Title: title},
		)
		r.checkpointLocked(
			&effects,
			RuntimeEventSessionRenamed,
			sessionValue,
			nil,
		)
	}

	receipt := CommandReceipt{
		CommandID:       command.Meta.ID,
		Accepted:        true,
		SessionID:       command.SessionID,
		SessionRevision: sessionValue.Revision,
	}
	r.finishCommandLocked(command.Meta, fingerprint, receipt)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func (r *Runtime) StartRun(
	ctx context.Context,
	command StartRunCommand,
) (*CommandReceipt, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		command.Meta,
		command,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionValue := r.sessions[command.SessionID]
	if sessionValue == nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				Message:   ErrSessionNotFound.Error(),
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if message := expectedSessionRevisionMessage(
		sessionValue,
		command.Meta.ExpectedSessionRevision,
	); message != "" {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message:         message,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if active := r.activeRunLocked(sessionValue); active != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           active.ID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     active.Revision,
				Message:         "an active run already owns this session",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if !hasUserInput(command.Input) {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message:         "run input must contain text, attachments, or MCP App context",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	effectiveConfiguration := cloneJSON(sessionValue.Configuration)
	persistConfiguration := command.Configuration != nil
	if command.Configuration != nil {
		effectiveConfiguration = cloneJSON(*command.Configuration)
	}

	plannedRunID := command.RunID
	if plannedRunID == "" {
		plannedRunID = RunID(r.newID())
	}
	plannedOperationID := OperationID(r.newID())
	plannedMessageID := r.newID()
	transcriptBeforePreparation := cloneJSON(sessionValue.Transcript)
	sessionRevisionBeforePreparation := sessionValue.Revision

	r.mu.Unlock()
	prepared, preparationErr := r.prepareSubmission(
		ctx,
		command.SessionID,
		effectiveConfiguration,
		command.Input,
		plannedOperationID,
		persistConfiguration,
	)
	var initial initialModelOperation
	if preparationErr == nil {
		initial, preparationErr = r.prepareInitialModelOperation(
			ctx,
			command.SessionID,
			plannedRunID,
			plannedOperationID,
			plannedMessageID,
			prepared.Configuration,
			transcriptBeforePreparation,
			prepared.Input,
			nil,
		)
	}
	r.mu.Lock()

	if r.closed {
		delete(r.commandReservations, command.Meta.ID)
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	sessionValue = r.sessions[command.SessionID]
	if sessionValue == nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				Message:   ErrSessionNotFound.Error(),
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if sessionValue.Revision != sessionRevisionBeforePreparation {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message:         "session changed while submission was being prepared; retry submission",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if active := r.activeRunLocked(sessionValue); active != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           active.ID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     active.Revision,
				Message:         "an active run already owns this session",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if preparationErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:          command.Meta.ID,
				SessionID:          command.SessionID,
				SessionRevision:    sessionValue.Revision,
				Message:            preparationErr.Error(),
				PreparationFailure: preparationFailureDetails(preparationErr),
			},
		)
		r.mu.Unlock()
		//nolint:nilerr // Ok
		return &receipt, nil
	}

	var runConfiguration *SessionConfiguration
	if prepared.PersistConfiguration {
		value := cloneJSON(prepared.Configuration)
		runConfiguration = &value
	}

	effects := runtimeEffects{}
	receipt, startErr := r.startRunLocked(
		sessionValue,
		command.Meta.ID,
		initial,
		prepared.Input,
		runConfiguration,
		command.Limits,
		&effects,
	)
	if startErr != nil {
		rejected := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message:         startErr.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(rejected)
	}

	r.finishCommandLocked(command.Meta, fingerprint, receipt)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func (r *Runtime) UpdateContinuationInput(
	ctx context.Context,
	command UpdateContinuationInputCommand,
) (*CommandReceipt, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		command.Meta,
		command,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionValue, run, lookupErr := r.sessionAndRunLocked(
		command.SessionID,
		command.RunID,
	)
	if lookupErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				RunID:     command.RunID,
				Message:   lookupErr.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(receipt)
	}
	if message := expectedRevisionsMessage(
		sessionValue,
		run,
		command.Meta,
	); message != "" {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         message,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if run.State.Terminal() ||
		run.Controls.CancellationRequested {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "cannot update continuation input for a terminal or cancelling run",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if command.ExpectedInputRevision != nil &&
		run.Continuation.Revision != *command.ExpectedInputRevision {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message: fmt.Sprintf(
					"continuation input revision mismatch: expected %d, current %d",
					*command.ExpectedInputRevision,
					run.Continuation.Revision,
				),
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	effectiveConfiguration := cloneJSON(run.Configuration)
	if command.Input.Configuration != nil {
		effectiveConfiguration = cloneJSON(*command.Input.Configuration)
	}
	sessionRevisionBeforePreparation := sessionValue.Revision
	runRevisionBeforePreparation := run.Revision
	continuationRevisionBeforePreparation := run.Continuation.Revision

	r.mu.Unlock()
	prepared, preparationErr := r.prepareSubmission(
		ctx,
		command.SessionID,
		effectiveConfiguration,
		UserInput{
			Text:          command.Input.Text,
			Attachments:   command.Input.Attachments,
			MCPAppContext: command.Input.MCPAppContext,
			Metadata:      command.Input.Metadata,
		},
		"",
		false,
	)
	r.mu.Lock()

	if r.closed {
		delete(r.commandReservations, command.Meta.ID)
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	sessionValue, run, lookupErr = r.sessionAndRunLocked(
		command.SessionID,
		command.RunID,
	)
	if lookupErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				RunID:     command.RunID,
				Message:   lookupErr.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(receipt)
	}
	if sessionValue.Revision != sessionRevisionBeforePreparation ||
		run.Revision != runRevisionBeforePreparation ||
		run.Continuation.Revision != continuationRevisionBeforePreparation {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "run changed while continuation input was being prepared; retry update",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if preparationErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         preparationErr.Error(),
			},
		)
		r.mu.Unlock()

		return &receipt, nil
	}

	input := cloneJSON(command.Input)
	input.Text = prepared.Input.Text
	input.Attachments = prepared.Input.Attachments
	input.MCPAppContext = prepared.Input.MCPAppContext
	input.Metadata = cloneRaw(prepared.Input.Metadata)
	if command.Input.Configuration != nil {
		input.Configuration = &prepared.Configuration
	}
	input.Revision = run.Continuation.Revision + 1
	input.ConsumedRevision = run.Continuation.ConsumedRevision
	input.UpdatedAt = r.now()

	run.Continuation = input
	r.touchRunLocked(sessionValue, run)

	receipt := CommandReceipt{
		CommandID:       command.Meta.ID,
		Accepted:        true,
		SessionID:       command.SessionID,
		RunID:           command.RunID,
		SessionRevision: sessionValue.Revision,
		RunRevision:     run.Revision,
	}
	r.finishCommandLocked(command.Meta, fingerprint, receipt)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		command.Meta.ID,
		RuntimeEventContinuationUpdated,
		"",
		"",
		"",
		"",
		ContinuationRevisionEventData{Revision: input.Revision},
	)
	r.checkpointLocked(
		&effects,
		RuntimeEventContinuationUpdated,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func (r *Runtime) ContinueRun(
	ctx context.Context,
	command ContinueRunCommand,
) (*CommandReceipt, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		command.Meta,
		command,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionValue, run, lookupErr := r.sessionAndRunLocked(
		command.SessionID,
		command.RunID,
	)
	if lookupErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				RunID:     command.RunID,
				Message:   lookupErr.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(receipt)
	}
	if message := expectedRevisionsMessage(sessionValue, run, command.Meta); message != "" {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         message,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if run.Controls.Paused {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "run is paused",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if run.Controls.CancellationRequested || run.State.Terminal() {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "run cannot continue",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if run.ActiveModelOperationID != "" || hasInFlightToolCalls(run) {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "model or tool work is still in progress",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if !allRequiredCallsResolved(run) {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "required tool calls must be resolved before continuation",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	effects := runtimeEffects{}
	work, continuationErr := r.beginContinuationModelLocked(
		sessionValue,
		run,
		command.Meta.ID,
		false,
		&effects,
	)
	if continuationErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         continuationErr.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(receipt)
	}

	effects.work = append(effects.work, work)
	receipt := CommandReceipt{
		CommandID:       command.Meta.ID,
		Accepted:        true,
		SessionID:       command.SessionID,
		RunID:           command.RunID,
		SessionRevision: sessionValue.Revision,
		RunRevision:     run.Revision,
		OperationIDs:    []OperationID{work.OperationID},
	}
	r.finishCommandLocked(command.Meta, fingerprint, receipt)
	r.checkpointLocked(
		&effects,
		RuntimeEventContinuationConsumed,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func (r *Runtime) ExecuteCalls(
	ctx context.Context,
	command ExecuteCallsCommand,
) (*CommandReceipt, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		command.Meta,
		command,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionValue, run, lookupErr := r.sessionAndRunLocked(
		command.SessionID,
		command.RunID,
	)
	if lookupErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				RunID:     command.RunID,
				Message:   lookupErr.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(receipt)
	}
	if message := expectedRevisionsMessage(sessionValue, run, command.Meta); message != "" {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         message,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if run.Controls.Paused ||
		run.Controls.CancellationRequested ||
		run.State.Terminal() ||
		run.ActiveModelOperationID != "" ||
		hasPendingRecoveryReview(run) {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "run is not dispatchable",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	callIDs := command.CallIDs
	if len(callIDs) == 0 {
		callIDs = unresolvedCallsForLatestOperation(run)
	}
	callIDs = uniqueToolCallIDs(callIDs)

	if len(callIDs) == 0 {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "no unresolved tool calls are available",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	for _, callID := range callIDs {
		call := findToolCall(run, callID)
		if call == nil {
			receipt := r.rejectReceiptLocked(
				command.Meta,
				fingerprint,
				CommandReceipt{
					CommandID:       command.Meta.ID,
					SessionID:       command.SessionID,
					RunID:           command.RunID,
					SessionRevision: sessionValue.Revision,
					RunRevision:     run.Revision,
					Message:         fmt.Sprintf("tool call %q was not found", callID),
				},
			)
			r.mu.Unlock()
			return &receipt, nil
		}
		if call.ProviderManaged || !call.Required {
			receipt := r.rejectReceiptLocked(
				command.Meta,
				fingerprint,
				CommandReceipt{
					CommandID:       command.Meta.ID,
					SessionID:       command.SessionID,
					RunID:           command.RunID,
					SessionRevision: sessionValue.Revision,
					RunRevision:     run.Revision,
					Message:         fmt.Sprintf("tool call %q is provider-managed", callID),
				},
			)
			r.mu.Unlock()
			return &receipt, nil
		}
		if message := toolMutationMessage(run, call); message != "" {
			receipt := r.rejectReceiptLocked(
				command.Meta,
				fingerprint,
				CommandReceipt{
					CommandID:       command.Meta.ID,
					SessionID:       command.SessionID,
					RunID:           command.RunID,
					SessionRevision: sessionValue.Revision,
					RunRevision:     run.Revision,
					Message:         message,
				},
			)
			r.mu.Unlock()
			return &receipt, nil
		}
		if call.State != ToolCallStatePending &&
			call.State != ToolCallStateReady {
			receipt := r.rejectReceiptLocked(
				command.Meta,
				fingerprint,
				CommandReceipt{
					CommandID:       command.Meta.ID,
					SessionID:       command.SessionID,
					RunID:           command.RunID,
					SessionRevision: sessionValue.Revision,
					RunRevision:     run.Revision,
					Message: fmt.Sprintf(
						"tool call %q is not runnable; retry it before execution",
						callID,
					),
				},
			)
			r.mu.Unlock()
			return &receipt, nil
		}
	}

	run.RequestedCallIDs = appendUniqueToolCallIDs(
		run.RequestedCallIDs,
		callIDs,
	)
	if command.Continue {
		run.ContinuationRequested = true
	}
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		command.Meta.ID,
		RuntimeEventToolCallsExecutionRequested,
		"",
		"",
		"",
		"",
		ToolCallsExecutionRequestedEventData{
			CallIDs:  append([]ToolCallID(nil), callIDs...),
			Continue: command.Continue,
		},
	)
	r.advanceDriverLocked(
		sessionValue,
		run,
		command.Meta.ID,
		DriverEvent{
			Kind: DriverEventExplicitExecution,
		},
		&effects,
	)

	receipt := CommandReceipt{
		CommandID:       command.Meta.ID,
		Accepted:        true,
		SessionID:       command.SessionID,
		RunID:           command.RunID,
		SessionRevision: sessionValue.Revision,
		RunRevision:     run.Revision,
		ToolCallIDs:     callIDs,
	}
	r.finishCommandLocked(command.Meta, fingerprint, receipt)
	r.checkpointLocked(
		&effects,
		RuntimeEventRunStateChanged,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func (r *Runtime) RetryCall(
	ctx context.Context,
	command RetryCallCommand,
) (*CommandReceipt, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		command.Meta,
		command,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionValue, run, lookupErr := r.sessionAndRunLocked(
		command.SessionID,
		command.RunID,
	)
	if lookupErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				RunID:     command.RunID,
				Message:   lookupErr.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(receipt)
	}
	if message := expectedRevisionsMessage(sessionValue, run, command.Meta); message != "" {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         message,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	call := findToolCall(run, command.CallID)
	if call == nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         toolCallNotFound,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if call.State == ToolCallStateRunning ||
		call.State == ToolCallStateEvaluating ||
		call.State == ToolCallStateWaiting {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "tool call is already active",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if call.ProviderManaged || !call.Required {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "provider-managed tool calls cannot be retried by the local runtime",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if message := toolMutationMessage(run, call); message != "" {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         message,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if call.State != ToolCallStateFailed &&
		call.State != ToolCallStateResolved {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "tool call is not in a retryable state",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if call.State == ToolCallStateResolved {
		result := selectedToolResult(*call)
		if result == nil || result.Kind == ToolResultKindSuccess {
			receipt := r.rejectReceiptLocked(
				command.Meta,
				fingerprint,
				CommandReceipt{
					CommandID:       command.Meta.ID,
					SessionID:       command.SessionID,
					RunID:           command.RunID,
					SessionRevision: sessionValue.Revision,
					RunRevision:     run.Revision,
					Message:         "successful tool results cannot be retried",
				},
			)
			r.mu.Unlock()
			return &receipt, nil
		}
	}

	call.SelectedResultID = ""
	call.LastFailure = ""
	call.Authorization = nil
	call.WaitID = ""
	call.State = ToolCallStatePending
	run.RequestedCallIDs = appendUniqueToolCallIDs(
		run.RequestedCallIDs,
		[]ToolCallID{call.ID},
	)
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		command.Meta.ID,
		RuntimeEventToolCallRetried,
		"",
		call.ID,
		"",
		"",
		nil,
	)
	r.advanceDriverLocked(
		sessionValue,
		run,
		command.Meta.ID,
		DriverEvent{
			Kind:       DriverEventExplicitExecution,
			ToolCallID: call.ID,
		},
		&effects,
	)

	receipt := CommandReceipt{
		CommandID:       command.Meta.ID,
		Accepted:        true,
		SessionID:       command.SessionID,
		RunID:           command.RunID,
		SessionRevision: sessionValue.Revision,
		RunRevision:     run.Revision,
		ToolCallIDs:     []ToolCallID{call.ID},
	}
	r.finishCommandLocked(command.Meta, fingerprint, receipt)
	r.checkpointLocked(
		&effects,
		RuntimeEventRunStateChanged,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func (r *Runtime) SkipCall(
	ctx context.Context,
	command SkipCallCommand,
) (*CommandReceipt, error) {
	return r.completeCallWithResult(
		ctx,
		command.Meta,
		command.SessionID,
		command.RunID,
		command.CallID,
		ToolResultKindSkipped,
		"The tool call was intentionally skipped by the user.",
	)
}

func (r *Runtime) CompleteCallAsError(
	ctx context.Context,
	command CompleteCallAsErrorCommand,
) (*CommandReceipt, error) {
	message := strings.TrimSpace(command.Error)
	if message == "" {
		message = "The tool call failed without a usable result."
	}

	return r.completeCallWithResult(
		ctx,
		command.Meta,
		command.SessionID,
		command.RunID,
		command.CallID,
		ToolResultKindError,
		message,
	)
}

func (r *Runtime) ResolveWait(
	ctx context.Context,
	command ResolveWaitCommand,
) (*CommandReceipt, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		command.Meta,
		command,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionValue, run, lookupErr := r.sessionAndRunLocked(
		command.SessionID,
		command.RunID,
	)
	if lookupErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				RunID:     command.RunID,
				Message:   lookupErr.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(receipt)
	}
	if message := expectedRevisionsMessage(sessionValue, run, command.Meta); message != "" {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         message,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if run.State.Terminal() || run.Controls.CancellationRequested {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "terminal or cancelling runs cannot resolve waiting requests",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	wait := findWaitingRequest(run, command.WaitID)
	if wait == nil || wait.Status != WaitStatusPending {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "waiting request is not pending",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if !waitAllowsResolution(*wait, command.Resolution) {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "wait resolution is not allowed for this waiting request",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	if wait.Kind == WaitKindRecovery {
		wait.Status = WaitStatusResolved
		wait.Error = ""
		wait.UpdatedAt = r.now()
		r.clearWaitReferenceLocked(wait.ID)

		effects := runtimeEffects{}
		r.recordRunEventLocked(
			&effects,
			sessionValue,
			run,
			command.Meta.ID,
			RuntimeEventWaitResolved,
			wait.OperationID,
			wait.ToolCallID,
			"",
			wait.ID,
			wait,
		)

		if command.Resolution == WaitResolutionCancel {
			r.markRunTerminalLocked(
				sessionValue,
				run,
				command.Meta.ID,
				RunOutcome{
					Kind:    RunOutcomeCancelled,
					Code:    "recovery_cancelled",
					Message: "the recovered run was cancelled by the user",
				},
				&effects,
			)
		} else {
			work, resumed, resumeErr := r.resumeRecoveredModelLocked(
				sessionValue,
				run,
				command.Meta.ID,
				&effects,
			)
			//nolint:gocritic // No switch.
			if resumeErr != nil {
				r.markRunTerminalLocked(
					sessionValue,
					run,
					command.Meta.ID,
					RunOutcome{
						Kind:    RunOutcomeFailed,
						Code:    "recovery_resume_failed",
						Message: resumeErr.Error(),
					},
					&effects,
				)
			} else if resumed {
				effects.work = append(effects.work, work)
			} else {
				run.State = RunStateWaiting
				r.touchRunLocked(sessionValue, run)
				r.recordRunEventLocked(
					&effects,
					sessionValue,
					run,
					command.Meta.ID,
					RuntimeEventRunStateChanged,
					"",
					"",
					"",
					"",
					RunStateChangedEventData{State: RunStateWaiting},
				)
			}
		}

		receipt := CommandReceipt{
			CommandID:       command.Meta.ID,
			Accepted:        true,
			SessionID:       command.SessionID,
			RunID:           command.RunID,
			SessionRevision: sessionValue.Revision,
			RunRevision:     run.Revision,
			WaitIDs:         []WaitID{command.WaitID},
		}
		r.finishCommandLocked(command.Meta, fingerprint, receipt)
		r.checkpointLocked(
			&effects,
			RuntimeEventWaitResolved,
			sessionValue,
			run,
		)
		r.mu.Unlock()
		r.flushEffects(ctx, effects)
		return &receipt, nil
	}

	wait.Status = WaitStatusResolving
	wait.UpdatedAt = r.now()
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		command.Meta.ID,
		RuntimeEventWaitResolutionRequested,
		wait.OperationID,
		wait.ToolCallID,
		"",
		wait.ID,
		WaitResolutionRequestedEventData{
			Resolution: command.Resolution,
		},
	)
	effects.work = append(effects.work, runtimeWork{
		Kind:       workResolveWait,
		SessionID:  command.SessionID,
		RunID:      command.RunID,
		WaitID:     command.WaitID,
		Resolution: command.Resolution,
		Input:      cloneRaw(command.Input),
	})

	receipt := CommandReceipt{
		CommandID:       command.Meta.ID,
		Accepted:        true,
		SessionID:       command.SessionID,
		RunID:           command.RunID,
		SessionRevision: sessionValue.Revision,
		RunRevision:     run.Revision,
		WaitIDs:         []WaitID{command.WaitID},
	}
	r.finishCommandLocked(command.Meta, fingerprint, receipt)
	r.checkpointLocked(
		&effects,
		RuntimeEventWaitResolved,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func (r *Runtime) SetAutomationHold(
	ctx context.Context,
	command SetAutomationHoldCommand,
) (*CommandReceipt, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		command.Meta,
		command,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionValue, run, lookupErr := r.sessionAndRunLocked(
		command.SessionID,
		command.RunID,
	)
	if lookupErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				RunID:     command.RunID,
				Message:   lookupErr.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(receipt)
	}
	if message := expectedRevisionsMessage(sessionValue, run, command.Meta); message != "" {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         message,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if run.State.Terminal() || run.Controls.CancellationRequested {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "terminal or cancelling runs cannot change automation controls",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if !command.Tools && !command.Continuation {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "automation hold must target tools, continuation, or both",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	scope := command.Scope
	if scope == "" {
		scope = AutomationHoldOneBatch
	}
	if scope != AutomationHoldOneBatch &&
		scope != AutomationHoldIndefinite {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "automation hold scope is unsupported",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	target := latestBatchOperationID(run)
	if run.ActiveModelOperationID != "" {
		target = run.ActiveModelOperationID
	}
	hold := AutomationHold{
		Enabled:           command.Enabled,
		Scope:             scope,
		TargetOperationID: target,
		SetByCommandID:    command.Meta.ID,
	}
	if !command.Enabled {
		hold = AutomationHold{}
	}
	if command.Tools {
		run.Controls.AutomaticTools = hold
	}
	if command.Continuation {
		run.Controls.AutomaticContinuation = hold
	}
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		command.Meta.ID,
		RuntimeEventAutomationChanged,
		target,
		"",
		"",
		"",
		run.Controls,
	)

	if !command.Enabled {
		active := findModelOperation(run, run.ActiveModelOperationID)
		if active != nil &&
			active.State == OperationStateQueued &&
			active.Automatic &&
			run.State == RunStateWaiting &&
			!run.Controls.Paused &&
			!automationHoldApplies(
				run.Controls.AutomaticContinuation,
				active.ID,
			) {
			run.State = RunStateRunning
			r.touchRunLocked(sessionValue, run)
			r.recordRunEventLocked(
				&effects,
				sessionValue,
				run,
				command.Meta.ID,
				RuntimeEventRunStateChanged,
				active.ID,
				"",
				"",
				"",
				RunStateChangedEventData{State: run.State},
			)
			effects.work = append(effects.work, runtimeWork{
				Kind:        workModel,
				SessionID:   sessionValue.ID,
				RunID:       run.ID,
				OperationID: active.ID,
				Automatic:   true,
			})
		} else {
			r.advanceDriverLocked(
				sessionValue,
				run,
				command.Meta.ID,
				DriverEvent{Kind: DriverEventControlsChanged},
				&effects,
			)
		}
	}

	receipt := CommandReceipt{
		CommandID:       command.Meta.ID,
		Accepted:        true,
		SessionID:       command.SessionID,
		RunID:           command.RunID,
		SessionRevision: sessionValue.Revision,
		RunRevision:     run.Revision,
	}
	r.finishCommandLocked(command.Meta, fingerprint, receipt)
	r.checkpointLocked(
		&effects,
		RuntimeEventAutomationChanged,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func (r *Runtime) PauseRun(
	ctx context.Context,
	command PauseRunCommand,
) (*CommandReceipt, error) {
	return r.setPause(
		ctx,
		command.Meta,
		command.SessionID,
		command.RunID,
		true,
	)
}

func (r *Runtime) ResumeRun(
	ctx context.Context,
	command ResumeRunCommand,
) (*CommandReceipt, error) {
	return r.setPause(
		ctx,
		command.Meta,
		command.SessionID,
		command.RunID,
		false,
	)
}

func (r *Runtime) CancelRun(
	ctx context.Context,
	command CancelRunCommand,
) (*CommandReceipt, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		command.Meta,
		command,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionValue, run, lookupErr := r.sessionAndRunLocked(
		command.SessionID,
		command.RunID,
	)
	if lookupErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				RunID:     command.RunID,
				Message:   lookupErr.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(receipt)
	}

	// Cancellation is an idempotent intent for this exact Run. A model stream
	// advances Session and Run revisions frequently, so revision preconditions
	// would make the Stop button fail during its normal use case.
	if run.State.Terminal() {
		receipt := CommandReceipt{
			CommandID:       command.Meta.ID,
			Accepted:        true,
			SessionID:       command.SessionID,
			RunID:           command.RunID,
			SessionRevision: sessionValue.Revision,
			RunRevision:     run.Revision,
			Message:         "run is already terminal",
		}
		r.finishCommandLocked(
			command.Meta,
			fingerprint,
			receipt,
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	run.Controls.CancellationRequested = true
	run.State = RunStateCancelling
	r.cancelUndispatchedLocked(run)
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
	for key, cancel := range r.cancels {
		if isRunDispatchCancelKey(key, run.ID) {
			effects.cancels = append(effects.cancels, cancel)
			delete(r.cancels, key)
		}
	}

	if !hasActiveExternalWork(run) {
		r.markRunTerminalLocked(
			sessionValue,
			run,
			command.Meta.ID,
			RunOutcome{
				Kind: RunOutcomeCancelled,
				Code: string(RunStateCancelled),
				Message: firstNonEmpty(
					strings.TrimSpace(command.Reason),
					"run cancellation was requested",
				),
			},
			&effects,
		)
	} else {
		r.recordRunEventLocked(
			&effects,
			sessionValue,
			run,
			command.Meta.ID,
			RuntimeEventRunStateChanged,
			run.ActiveModelOperationID,
			"",
			"",
			"",
			RunStateChangedEventData{State: RunStateCancelling},
		)
	}

	receipt := CommandReceipt{
		CommandID:       command.Meta.ID,
		Accepted:        true,
		SessionID:       command.SessionID,
		RunID:           command.RunID,
		SessionRevision: sessionValue.Revision,
		RunRevision:     run.Revision,
	}
	r.finishCommandLocked(command.Meta, fingerprint, receipt)
	r.checkpointLocked(
		&effects,
		RuntimeEventRunCancelled,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func (r *Runtime) EditAndResend(
	ctx context.Context,
	command EditAndResendCommand,
) (*CommandReceipt, error) {
	type preparationSnapshot struct {
		ready       bool
		revision    Revision
		config      SessionConfiguration
		persistConf bool
		toolResults []ToolResult
		carryErr    error
		transcript  []TranscriptMessage
		runID       RunID
		operationID OperationID
		messageID   string
	}

	// Do not hold the coordinator lock while attachment/model/tool/MCP
	// preparation runs. Preparation is potentially slow and may perform I/O.
	snapshot := preparationSnapshot{}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, ErrRuntimeClosed
	}

	sessionValue := r.sessions[command.SessionID]
	if sessionValue != nil &&
		(hasUserInput(command.Input) ||
			len(command.CarryForwardToolResultIDs) != 0) &&
		expectedSessionRevisionMessage(
			sessionValue,
			command.Meta.ExpectedSessionRevision,
		) == "" &&
		r.activeRunLocked(sessionValue) == nil {
		index := -1
		for i, message := range sessionValue.Transcript {
			if message.ID == command.ReplaceMessageID {
				index = i
				break
			}
		}

		if index >= 0 &&
			sessionValue.Transcript[index].Role == MessageRoleUser {
			snapshot.ready = true
			snapshot.revision = sessionValue.Revision
			snapshot.config = cloneJSON(sessionValue.Configuration)
			snapshot.transcript = cloneJSON(sessionValue.Transcript[:index])
			snapshot.runID = command.RunID
			if snapshot.runID == "" {
				snapshot.runID = RunID(r.newID())
			}
			snapshot.operationID = OperationID(r.newID())
			snapshot.messageID = r.newID()
			snapshot.persistConf = command.Configuration != nil
			snapshot.toolResults, snapshot.carryErr = carryForwardToolResults(
				sessionValue.Transcript[index].ToolResults,
				command.CarryForwardToolResultIDs,
			)
			if command.Configuration != nil {
				snapshot.config = cloneJSON(*command.Configuration)
			}
		}
	}
	r.mu.RUnlock()

	var (
		prepared       PreparedSubmission
		preparationErr error
		initial        initialModelOperation
	)
	if snapshot.ready && snapshot.carryErr == nil {
		prepared, preparationErr = r.prepareSubmission(
			ctx,
			command.SessionID,
			snapshot.config,
			command.Input,
			snapshot.operationID,
			snapshot.persistConf,
		)
		if preparationErr == nil {
			initial, preparationErr = r.prepareInitialModelOperation(
				ctx,
				command.SessionID,
				snapshot.runID,
				snapshot.operationID,
				snapshot.messageID,
				prepared.Configuration,
				snapshot.transcript,
				prepared.Input,
				snapshot.toolResults,
			)
		}
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		command.Meta,
		command,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionValue = r.sessions[command.SessionID]
	if sessionValue == nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				Message:   ErrSessionNotFound.Error(),
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	if message := expectedSessionRevisionMessage(
		sessionValue,
		command.Meta.ExpectedSessionRevision,
	); message != "" {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message:         message,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	if active := r.activeRunLocked(sessionValue); active != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           active.ID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     active.Revision,
				Message:         "editing requires an explicit cancellation or a safe terminal boundary",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	if !hasUserInput(command.Input) &&
		len(command.CarryForwardToolResultIDs) == 0 {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message:         "run input must contain text, attachments, or MCP App context",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	index := -1
	for i, message := range sessionValue.Transcript {
		if message.ID == command.ReplaceMessageID {
			index = i
			break
		}
	}
	if index < 0 {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message:         "replacement message was not found",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if sessionValue.Transcript[index].Role != MessageRoleUser {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message:         "only a user transcript message can be replaced",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	// We only accept preparation against the exact Session version that was
	// inspected before releasing the lock. Otherwise an intervening mutation
	// could make us replace a different visible suffix/configuration.
	if !snapshot.ready || sessionValue.Revision != snapshot.revision {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message:         "session changed while edit-and-resend input was being prepared; retry the edit",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if snapshot.carryErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message:         snapshot.carryErr.Error(),
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	if preparationErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:          command.Meta.ID,
				SessionID:          command.SessionID,
				SessionRevision:    sessionValue.Revision,
				Message:            preparationErr.Error(),
				PreparationFailure: preparationFailureDetails(preparationErr),
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	var runConfiguration *SessionConfiguration
	if prepared.PersistConfiguration {
		value := cloneJSON(prepared.Configuration)
		runConfiguration = &value
	}

	// "startRunLocked" has its own rollback for Run construction. This outer
	// snapshot restores the visible Session suffix if accepting replacement
	// work fails after the transcript was truncated.
	previousSession := cloneJSON(*sessionValue)

	sessionValue.Transcript = append(
		[]TranscriptMessage(nil),
		sessionValue.Transcript[:index]...,
	)
	r.touchSessionLocked(sessionValue)

	effects := runtimeEffects{}
	receipt, startErr := r.startRunLocked(
		sessionValue,
		command.Meta.ID,
		initial,
		prepared.Input,
		runConfiguration,
		command.Limits,
		&effects,
	)
	if startErr != nil {
		*sessionValue = previousSession

		rejected := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				SessionRevision: sessionValue.Revision,
				Message:         startErr.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(rejected)
	}

	r.finishCommandLocked(command.Meta, fingerprint, receipt)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func carryForwardToolResults(
	available []ToolResult,
	requested []ToolResultID,
) ([]ToolResult, error) {
	if len(requested) == 0 {
		return nil, nil
	}

	want := make(map[ToolResultID]struct{}, len(requested))
	for _, id := range requested {
		if id == "" {
			return nil, errors.New(
				"carried Tool result identity is required",
			)
		}
		want[id] = struct{}{}
	}

	output := make([]ToolResult, 0, len(want))
	for _, result := range available {
		if _, found := want[result.ID]; !found {
			continue
		}
		output = append(output, cloneJSON(result))
		delete(want, result.ID)
	}

	if len(want) != 0 {
		return nil, errors.New(
			"carried Tool result does not belong to the message being replaced",
		)
	}

	return output, nil
}

func (r *Runtime) beginCommandLocked(
	meta CommandMeta,
	payload any,
) (
	receipt CommandReceipt,
	duplicate bool,
	fingerprint string,
	err error,
) {
	if meta.ID == "" {
		return CommandReceipt{}, false, "", errors.New("command ID is required")
	}

	fingerprint, err = commandFingerprint(payload)
	if err != nil {
		return CommandReceipt{}, false, "", err
	}
	if previous, found := r.commandRecords[meta.ID]; found {
		if previous.Fingerprint != fingerprint {
			return CommandReceipt{}, false, "", ErrCommandIDConflict
		}
		return cloneJSON(previous.Receipt), true, fingerprint, nil
	}

	if r.commandReservations == nil {
		r.commandReservations = make(map[CommandID]string)
	}
	if reservedFingerprint, found := r.commandReservations[meta.ID]; found {
		if reservedFingerprint != fingerprint {
			return CommandReceipt{}, false, "", ErrCommandIDConflict
		}
		return CommandReceipt{}, false, fingerprint, ErrCommandInProgress
	}
	r.commandReservations[meta.ID] = fingerprint
	return CommandReceipt{}, false, fingerprint, nil
}

func (r *Runtime) finishCommandLocked(
	meta CommandMeta,
	fingerprint string,
	receipt CommandReceipt,
) {
	delete(r.commandReservations, meta.ID)
	r.commandRecords[meta.ID] = commandRecord{
		Fingerprint: fingerprint,
		Receipt:     cloneJSON(receipt),
	}
}

func (r *Runtime) rejectReceiptLocked(
	meta CommandMeta,
	fingerprint string,
	receipt CommandReceipt,
) CommandReceipt {
	receipt.Accepted = false
	if receipt.CommandID == "" {
		receipt.CommandID = meta.ID
	}
	r.finishCommandLocked(meta, fingerprint, receipt)
	return receipt
}

// commandReceiptResult preserves rejected command receipts as application
// results rather than turning expected user-validation failures into Wails
// transport failures.
func commandReceiptResult(
	receipt CommandReceipt,
) (*CommandReceipt, error) {
	return &receipt, nil
}

func preparationFailureDetails(
	cause error,
) *OperationPreparationFailureDetails {
	var failure *OperationPreparationFailureError
	if !errors.As(cause, &failure) || failure == nil {
		return nil
	}

	return &OperationPreparationFailureDetails{
		Code:       failure.Code,
		Message:    failure.Error(),
		Resolution: cloneJSON(failure.Summary),
	}
}

func commandFingerprint(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode command payload: %w", err)
	}
	// Include the command type so different command kinds cannot share an ID
	// merely because their JSON object shapes happen to be identical.
	typed := append(fmt.Appendf(nil, "%T\x00", value), raw...)
	sum := sha256.Sum256(typed)
	return hex.EncodeToString(sum[:]), nil
}

func expectedSessionRevisionMessage(
	sessionValue *Session,
	expected *Revision,
) string {
	if expected == nil {
		return ""
	}
	if sessionValue.Revision == *expected {
		return ""
	}

	return fmt.Sprintf(
		"session revision mismatch: expected %d, current %d",
		*expected,
		sessionValue.Revision,
	)
}

func expectedRevisionsMessage(
	sessionValue *Session,
	run *Run,
	meta CommandMeta,
) string {
	if message := expectedSessionRevisionMessage(
		sessionValue,
		meta.ExpectedSessionRevision,
	); message != "" {
		return message
	}
	if meta.ExpectedRunRevision == nil {
		return ""
	}
	if run.Revision == *meta.ExpectedRunRevision {
		return ""
	}

	return fmt.Sprintf(
		"run revision mismatch: expected %d, current %d",
		*meta.ExpectedRunRevision,
		run.Revision,
	)
}

func (r *Runtime) startRunLocked(
	sessionValue *Session,
	commandID CommandID,
	initial initialModelOperation,
	input UserInput,
	configuration *SessionConfiguration,
	limits *RunLimits,
	effects *runtimeEffects,
) (CommandReceipt, error) {
	if !hasUserInput(input) {
		return CommandReceipt{}, errors.New(
			"run input must contain text, attachments, tool results, or MCP App context",
		)
	}
	normalizedLimits, err := normalizeRunLimits(limits)
	if err != nil {
		return CommandReceipt{}, err
	}

	runID := initial.RunID
	if _, exists := r.runs[runID]; exists {
		return CommandReceipt{}, fmt.Errorf("run %q already exists", runID)
	}

	previousSession := cloneJSON(*sessionValue)
	previousEffects := *effects

	runConfiguration := cloneJSON(sessionValue.Configuration)
	if configuration != nil {
		runConfiguration = cloneJSON(*configuration)
		sessionValue.Configuration = cloneJSON(*configuration)
	}

	driver := r.drivers[SequentialChatDriverKind]
	if driver == nil {
		return CommandReceipt{}, errors.New("sequential chat driver is unavailable")
	}

	now := r.now()
	run := &Run{
		ID:            runID,
		SessionID:     sessionValue.ID,
		Revision:      1,
		State:         RunStateQueued,
		CreatedAt:     now,
		UpdatedAt:     now,
		Driver:        driver.InitialState(),
		Configuration: runConfiguration,
		SkillSession: SkillSessionState{
			Generation: 1,
			Enabled:    cloneJSON(initial.Prepared.EffectiveSkills),
			Active:     cloneJSON(initial.Prepared.ActiveSkills),
		},
		Limits: normalizedLimits,
	}

	inputMessage := initial.Current

	sessionValue.Transcript = append(sessionValue.Transcript, inputMessage)
	sessionValue.RunIDs = append(sessionValue.RunIDs, runID)
	r.runs[runID] = run
	r.runToSession[runID] = sessionValue.ID
	r.touchSessionLocked(sessionValue)

	r.recordRunEventLocked(
		effects,
		sessionValue,
		run,
		commandID,
		RuntimeEventRunAccepted,
		"",
		"",
		"",
		"",
		RunStateChangedEventData{State: RunStateQueued},
	)
	r.recordRunEventLocked(
		effects,
		sessionValue,
		run,
		commandID,
		RuntimeEventRunStateChanged,
		"",
		"",
		"",
		"",
		RunStateChangedEventData{State: RunStateRunning},
	)

	work, err := r.startModelOperationLocked(
		sessionValue,
		run,
		commandID,
		initial.OperationID,
		inputMessage.ID,
		nil,
		&initial.Prepared,
		false,
		effects,
	)
	if err != nil {
		*sessionValue = previousSession
		*effects = previousEffects
		delete(r.runs, runID)
		delete(r.runToSession, runID)
		return CommandReceipt{}, err
	}

	effects.work = append(effects.work, work)
	r.checkpointLocked(
		effects,
		RuntimeEventRunAccepted,
		sessionValue,
		run,
	)

	return CommandReceipt{
		CommandID:       commandID,
		Accepted:        true,
		SessionID:       sessionValue.ID,
		RunID:           runID,
		SessionRevision: sessionValue.Revision,
		RunRevision:     run.Revision,
		OperationIDs:    []OperationID{work.OperationID},
	}, nil
}

func (r *Runtime) startModelOperationLocked(
	sessionValue *Session,
	run *Run,
	commandID CommandID,
	requestedOperationID OperationID,
	currentMessageID string,
	operationConfiguration *SessionConfiguration,
	prepared *OperationPreparation,
	automatic bool,
	effects *runtimeEffects,
) (runtimeWork, error) {
	if run.Limits.MaxModelOperations > 0 &&
		run.Usage.ModelOperations >= run.Limits.MaxModelOperations {
		return runtimeWork{}, errors.New("run model operation limit reached")
	}

	current, found := transcriptMessageByID(
		sessionValue.Transcript,
		currentMessageID,
	)
	if !found {
		return runtimeWork{}, errors.New("model operation current message was not found")
	}
	if current.Role != MessageRoleUser {
		return runtimeWork{}, errors.New("model operation current message must be user-authored")
	}

	effectiveConfiguration := cloneJSON(run.Configuration)
	if operationConfiguration != nil {
		effectiveConfiguration = cloneJSON(*operationConfiguration)
	}

	history := selectHistory(
		sessionValue.Transcript,
		currentMessageID,
		effectiveConfiguration.History,
	)
	operationID := requestedOperationID
	if operationID == "" {
		operationID = OperationID(r.newID())
	}

	// The transcript keeps one durable association to the operation that
	// prepared it. Resolution data remains canonical on ModelOperation and is
	// not copied into TranscriptMessage.
	current.OperationID = operationID
	for index := range sessionValue.Transcript {
		if sessionValue.Transcript[index].ID == currentMessageID {
			sessionValue.Transcript[index].OperationID = operationID
			break
		}
	}

	operation := ModelOperation{
		ID:        operationID,
		State:     OperationStateQueued,
		Automatic: automatic,
		Request: ModelOperationInput{
			Configuration:        effectiveConfiguration,
			History:              cloneJSON(history),
			Current:              cloneJSON(current),
			ContinuationRevision: run.Continuation.ConsumedRevision,
		},
		Preparation: OperationPreparation{
			State: PreparationStatePending,
		},
	}
	if prepared != nil {
		operation.Preparation = cloneOperationPreparation(*prepared)
	}

	run.ModelOperations = append(run.ModelOperations, operation)
	run.ActiveModelOperationID = operationID
	run.State = RunStateRunning
	run.Usage.ModelOperations++
	r.touchRunLocked(sessionValue, run)

	r.recordRunEventLocked(
		effects,
		sessionValue,
		run,
		commandID,
		RuntimeEventModelOperationQueued,
		operationID,
		"",
		"",
		"",
		nil,
	)

	if operation.Preparation.State == PreparationStatePrepared {
		r.recordRunEventLocked(
			effects,
			sessionValue,
			run,
			commandID,
			RuntimeEventModelOperationPrepared,
			operationID,
			"",
			"",
			"",
			ModelOperationPreparedEventData{
				Fingerprint: operation.Preparation.Fingerprint,
				Resolution:  cloneJSON(operation.Preparation.Resolution),
			},
		)
	}

	return runtimeWork{
		Kind:        workModel,
		SessionID:   sessionValue.ID,
		RunID:       run.ID,
		OperationID: operationID,
		Automatic:   automatic,
	}, nil
}

func (r *Runtime) beginContinuationModelLocked(
	sessionValue *Session,
	run *Run,
	commandID CommandID,
	automatic bool,
	effects *runtimeEffects,
) (runtimeWork, error) {
	if hasPendingRecoveryReview(run) {
		return runtimeWork{}, errors.New(
			"recovery review must be resolved before continuation",
		)
	}
	if run.ActiveModelOperationID != "" {
		return runtimeWork{}, errors.New("a model operation is already active")
	}
	if !allRequiredCallsResolved(run) {
		return runtimeWork{}, errors.New("required tool calls remain unresolved")
	}
	if run.Limits.MaxModelOperations > 0 &&
		run.Usage.ModelOperations >= run.Limits.MaxModelOperations {
		return runtimeWork{}, errors.New("run model operation limit reached")
	}

	staged := ContinuationInput{}
	hasNewRevision := run.Continuation.Revision >
		run.Continuation.ConsumedRevision
	if hasNewRevision {
		staged = cloneJSON(run.Continuation)
	}

	results := collectUnconsumedToolResults(run)
	hasInput := strings.TrimSpace(staged.Text) != "" ||
		len(staged.Attachments) != 0 ||
		len(staged.MCPAppContext) != 0 ||
		len(results) != 0

	if !hasInput {
		return runtimeWork{}, errors.New(
			"continuation requires tool results, text, attachments, MCP App context, or configuration that produces an input",
		)
	}

	// Keep rollback local. Rejected continuation must not change live state.
	previousSession := cloneJSON(*sessionValue)
	previousRun := cloneJSON(*run)
	previousEffects := *effects

	var operationConfiguration *SessionConfiguration
	if staged.Configuration != nil {
		value := cloneJSON(*staged.Configuration)
		operationConfiguration = &value
	}

	inputMessage := TranscriptMessage{
		ID:            r.newID(),
		CreatedAt:     r.now(),
		Role:          MessageRoleUser,
		Status:        "continuation",
		Text:          staged.Text,
		Attachments:   cloneJSON(staged.Attachments),
		ToolResults:   cloneJSON(results),
		MCPAppContext: staged.MCPAppContext,
		Metadata:      cloneRaw(staged.Metadata),
	}
	sessionValue.Transcript = append(sessionValue.Transcript, inputMessage)

	if hasNewRevision {
		run.Continuation.ConsumedRevision = staged.Revision
	}
	run.ContinuationRequested = false

	work, err := r.startModelOperationLocked(
		sessionValue,
		run,
		commandID,
		"",
		inputMessage.ID,
		operationConfiguration,
		nil,
		automatic,
		effects,
	)
	if err != nil {
		*sessionValue = previousSession
		*run = previousRun
		*effects = previousEffects
		return runtimeWork{}, err
	}

	for callIndex := range run.ToolCalls {
		call := &run.ToolCalls[callIndex]
		if !isContinuationToolResult(*call) ||
			call.ConsumedByOperationID != "" ||
			call.SelectedResultID == "" {
			continue
		}
		call.ConsumedByOperationID = work.OperationID
	}
	r.touchRunLocked(sessionValue, run)

	r.recordRunEventLocked(
		effects,
		sessionValue,
		run,
		commandID,
		RuntimeEventContinuationConsumed,
		work.OperationID,
		"",
		"",
		"",
		ContinuationRevisionEventData{
			Revision: run.Continuation.ConsumedRevision,
		},
	)

	return work, nil
}

// isContinuationToolResult reports whether a selected result has a matching
// provider-side Tool call and may therefore be emitted as functionToolOutput.
//
// App-originated Tool calls are intentionally non-required and must be staged
// through App context rather than injected as a model Tool result.
func isContinuationToolResult(call ToolCall) bool {
	return call.Required && !call.ProviderManaged
}

func (r *Runtime) completeCallWithResult(
	ctx context.Context,
	meta CommandMeta,
	sessionID SessionID,
	runID RunID,
	callID ToolCallID,
	kind ToolResultKind,
	message string,
) (*CommandReceipt, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	payload := struct {
		Meta      CommandMeta
		SessionID SessionID
		RunID     RunID
		CallID    ToolCallID
		Kind      ToolResultKind
		Message   string
	}{
		Meta:      meta,
		SessionID: sessionID,
		RunID:     runID,
		CallID:    callID,
		Kind:      kind,
		Message:   message,
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		meta,
		payload,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionValue, run, lookupErr := r.sessionAndRunLocked(
		sessionID,
		runID,
	)
	if lookupErr != nil {
		receipt := r.rejectReceiptLocked(
			meta,
			fingerprint,
			CommandReceipt{
				CommandID: meta.ID,
				SessionID: sessionID,
				RunID:     runID,
				Message:   lookupErr.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(receipt)
	}
	if revisionMessage := expectedRevisionsMessage(
		sessionValue,
		run,
		meta,
	); revisionMessage != "" {
		receipt := r.rejectReceiptLocked(
			meta,
			fingerprint,
			CommandReceipt{
				CommandID:       meta.ID,
				SessionID:       sessionID,
				RunID:           runID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         revisionMessage,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	call := findToolCall(run, callID)
	if call == nil {
		receipt := r.rejectReceiptLocked(
			meta,
			fingerprint,
			CommandReceipt{
				CommandID:       meta.ID,
				SessionID:       sessionID,
				RunID:           runID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         toolCallNotFound,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if call.State == ToolCallStateRunning ||
		call.State == ToolCallStateEvaluating ||
		call.State == ToolCallStateWaiting {
		receipt := r.rejectReceiptLocked(
			meta,
			fingerprint,
			CommandReceipt{
				CommandID:       meta.ID,
				SessionID:       sessionID,
				RunID:           runID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "tool call is active and cannot be completed manually",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if call.ProviderManaged || !call.Required {
		receipt := r.rejectReceiptLocked(
			meta,
			fingerprint,
			CommandReceipt{
				CommandID:       meta.ID,
				SessionID:       sessionID,
				RunID:           runID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "provider-managed tool calls do not accept local results",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if message := toolMutationMessage(run, call); message != "" {
		receipt := r.rejectReceiptLocked(
			meta,
			fingerprint,
			CommandReceipt{
				CommandID:       meta.ID,
				SessionID:       sessionID,
				RunID:           runID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         message,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	result := ToolResult{
		ID:             ToolResultID(r.newID()),
		CallID:         call.ID,
		ProviderCallID: call.ProviderCallID,
		Name:           call.Name,
		ChoiceID:       call.ChoiceID,
		Type:           call.Type,
		Kind:           kind,
		Text:           message,
		Error:          message,
		CreatedAt:      r.now(),
	}
	selectToolResult(call, result)
	call.State = ToolCallStateResolved
	call.LastFailure = ""
	removeRequestedCall(run, call.ID)
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		meta.ID,
		RuntimeEventToolResultRecorded,
		call.OriginOperationID,
		call.ID,
		"",
		"",
		result,
	)
	r.advanceDriverLocked(
		sessionValue,
		run,
		meta.ID,
		DriverEvent{
			Kind:        DriverEventToolSettled,
			OperationID: call.OriginOperationID,
			ToolCallID:  call.ID,
		},
		&effects,
	)

	receipt := CommandReceipt{
		CommandID:       meta.ID,
		Accepted:        true,
		SessionID:       sessionID,
		RunID:           runID,
		SessionRevision: sessionValue.Revision,
		RunRevision:     run.Revision,
		ToolCallIDs:     []ToolCallID{call.ID},
	}
	r.finishCommandLocked(meta, fingerprint, receipt)
	r.checkpointLocked(
		&effects,
		RuntimeEventToolResultRecorded,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func (r *Runtime) setPause(
	ctx context.Context,
	meta CommandMeta,
	sessionID SessionID,
	runID RunID,
	paused bool,
) (*CommandReceipt, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	payload := struct {
		Meta      CommandMeta
		SessionID SessionID
		RunID     RunID
		Paused    bool
	}{
		Meta:      meta,
		SessionID: sessionID,
		RunID:     runID,
		Paused:    paused,
	}

	previous, duplicate, fingerprint, err := r.beginCommandLocked(
		meta,
		payload,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if duplicate {
		r.mu.Unlock()
		previous.Duplicate = true
		return &previous, nil
	}

	sessionValue, run, lookupErr := r.sessionAndRunLocked(
		sessionID,
		runID,
	)
	if lookupErr != nil {
		receipt := r.rejectReceiptLocked(
			meta,
			fingerprint,
			CommandReceipt{
				CommandID: meta.ID,
				SessionID: sessionID,
				RunID:     runID,
				Message:   lookupErr.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(receipt)
	}
	if revisionMessage := expectedRevisionsMessage(
		sessionValue,
		run,
		meta,
	); revisionMessage != "" {
		receipt := r.rejectReceiptLocked(
			meta,
			fingerprint,
			CommandReceipt{
				CommandID:       meta.ID,
				SessionID:       sessionID,
				RunID:           runID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         revisionMessage,
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if run.State.Terminal() || run.Controls.CancellationRequested {
		receipt := r.rejectReceiptLocked(
			meta,
			fingerprint,
			CommandReceipt{
				CommandID:       meta.ID,
				SessionID:       sessionID,
				RunID:           runID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "terminal or cancelling runs cannot change pause state",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	run.Controls.Paused = paused
	if paused {
		run.State = RunStatePaused
	} else if run.State == RunStatePaused {
		run.State = RunStateRunning
	}
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		meta.ID,
		RuntimeEventRunStateChanged,
		run.ActiveModelOperationID,
		"",
		"",
		"",
		RunStateChangedEventData{State: run.State},
	)
	if !paused {
		active := findModelOperation(run, run.ActiveModelOperationID)
		if active != nil && active.State == OperationStateQueued {
			effects.work = append(effects.work, runtimeWork{
				Kind:        workModel,
				SessionID:   sessionID,
				RunID:       runID,
				OperationID: active.ID,
				Automatic:   active.Automatic,
			})
		} else {
			r.advanceDriverLocked(
				sessionValue,
				run,
				meta.ID,
				DriverEvent{Kind: DriverEventResumed},
				&effects,
			)
		}
	}

	receipt := CommandReceipt{
		CommandID:       meta.ID,
		Accepted:        true,
		SessionID:       sessionID,
		RunID:           runID,
		SessionRevision: sessionValue.Revision,
		RunRevision:     run.Revision,
	}
	r.finishCommandLocked(meta, fingerprint, receipt)
	r.checkpointLocked(
		&effects,
		RuntimeEventRunStateChanged,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)

	return &receipt, nil
}

func (r *Runtime) advanceDriverLocked(
	sessionValue *Session,
	run *Run,
	commandID CommandID,
	event DriverEvent,
	effects *runtimeEffects,
) {
	if hasPendingRecoveryReview(run) {
		return
	}

	driver := r.drivers[run.Driver.Kind]
	if driver == nil {
		r.markRunTerminalLocked(
			sessionValue,
			run,
			commandID,
			RunOutcome{
				Kind:    RunOutcomeFailed,
				Code:    "driver_unavailable",
				Message: "run driver is unavailable",
			},
			effects,
		)
		return
	}

	event.View = buildDriverView(run)
	var (
		transition DriverTransition
		err        error
	)
	backgroundutil.RunSafely(
		"session driver transition",
		func() {
			transition, err = driver.Advance(run.Driver, event)
		},
		func(failure backgroundutil.PanicFailureError) {
			err = failure
		},
	)
	if err != nil {
		r.markRunTerminalLocked(
			sessionValue,
			run,
			commandID,
			RunOutcome{
				Kind:    RunOutcomeFailed,
				Code:    "driver_transition_failed",
				Message: err.Error(),
			},
			effects,
		)
		return
	}

	run.Driver = transition.NextState
	if transition.Outcome != nil {
		r.markRunTerminalLocked(
			sessionValue,
			run,
			commandID,
			*transition.Outcome,
			effects,
		)
		return
	}

	if transition.SetLifecycle != nil &&
		!run.Controls.Paused &&
		!run.Controls.CancellationRequested {
		run.State = *transition.SetLifecycle
		r.touchRunLocked(sessionValue, run)
		r.recordRunEventLocked(
			effects,
			sessionValue,
			run,
			commandID,
			RuntimeEventRunStateChanged,
			event.OperationID,
			event.ToolCallID,
			"",
			"",
			RunStateChangedEventData{State: run.State},
		)
	}

	for _, intent := range transition.Operations {
		switch intent.Kind {
		case OperationIntentStartModel:
			work, startErr := r.beginContinuationModelLocked(
				sessionValue,
				run,
				commandID,
				intent.Automatic,
				effects,
			)
			if startErr != nil {
				r.markRunTerminalLocked(
					sessionValue,
					run,
					commandID,
					RunOutcome{
						Kind:    RunOutcomeFailed,
						Code:    "continuation_failed",
						Message: startErr.Error(),
					},
					effects,
				)
				return
			}
			effects.work = append(effects.work, work)

		case OperationIntentExecuteTool:
			call := findToolCall(run, intent.ToolCallID)
			if call == nil {
				continue
			}
			kind := workToolEvaluate
			switch call.State {
			case ToolCallStatePending:
				call.State = ToolCallStateEvaluationQueued
			case ToolCallStateReady:
				call.State = ToolCallStateExecutionQueued
				kind = workToolExecute
			default:
				continue
			}
			r.touchRunLocked(sessionValue, run)
			effects.work = append(effects.work, runtimeWork{
				Kind:       kind,
				SessionID:  sessionValue.ID,
				RunID:      run.ID,
				ToolCallID: intent.ToolCallID,
				Automatic:  intent.Automatic,
			})

		case OperationIntentCompleteRun:
			r.markRunTerminalLocked(
				sessionValue,
				run,
				commandID,
				RunOutcome{Kind: RunOutcomeCompleted},
				effects,
			)

		default:
			r.markRunTerminalLocked(
				sessionValue,
				run,
				commandID,
				RunOutcome{
					Kind:    RunOutcomeFailed,
					Code:    "unsupported_operation_intent",
					Message: fmt.Sprintf("driver proposed unsupported operation intent %q", intent.Kind),
				},
				effects,
			)
		}
	}
}

func waitAllowsResolution(
	wait WaitingRequest,
	resolution WaitResolution,
) bool {
	return slices.Contains(
		wait.AllowedResolutions,
		resolution,
	)
}

func (r *Runtime) markRunTerminalLocked(
	sessionValue *Session,
	run *Run,
	commandID CommandID,
	outcome RunOutcome,
	effects *runtimeEffects,
) {
	if run.State.Terminal() {
		return
	}

	r.cancelOutstandingWaitsLocked(
		run,
		firstNonEmpty(outcome.Message, "run reached a terminal state"),
	)
	run.Outcome = &outcome
	switch outcome.Kind {
	case RunOutcomeCancelled:
		run.State = RunStateCancelled
	case RunOutcomeFailed:
		run.State = RunStateFailed
	default:
		run.State = RunStateCompleted
	}

	run.Controls.CancellationRequested = outcome.Kind == RunOutcomeCancelled
	skillID := run.SkillSession.ID
	run.SkillSession.ID = ""
	run.SkillSession.EstablishedGeneration = 0
	r.touchRunLocked(sessionValue, run)

	eventKind := RuntimeEventRunCompleted
	if outcome.Kind == RunOutcomeFailed {
		eventKind = RuntimeEventRunFailed
	}
	if outcome.Kind == RunOutcomeCancelled {
		eventKind = RuntimeEventRunCancelled
	}

	r.recordRunEventLocked(
		effects,
		sessionValue,
		run,
		commandID,
		eventKind,
		run.ActiveModelOperationID,
		"",
		"",
		"",
		outcome,
	)
	r.checkpointLocked(
		effects,
		eventKind,
		sessionValue,
		run,
	)

	if skillID != "" {
		effects.work = append(effects.work, runtimeWork{
			Kind:      workCloseSkill,
			SessionID: sessionValue.ID,
			RunID:     run.ID,
			SkillID:   skillID,
		})
	}
}

func buildDriverView(run *Run) DriverView {
	view := DriverView{
		RunState:               run.State,
		ActiveModelOperationID: run.ActiveModelOperationID,
		Controls:               cloneJSON(run.Controls),
		RequestedCallIDs:       append([]ToolCallID(nil), run.RequestedCallIDs...),
		ContinuationRequested:  run.ContinuationRequested,
		ToolCalls:              make([]DriverToolCallView, 0, len(run.ToolCalls)),
	}

	for _, call := range run.ToolCalls {
		var selected *ToolResultKind
		if result := selectedToolResult(call); result != nil {
			kind := result.Kind
			selected = &kind
		}

		view.ToolCalls = append(view.ToolCalls, DriverToolCallView{
			ID:                call.ID,
			OriginOperationID: call.OriginOperationID,
			Required:          call.Required,
			ProviderManaged:   call.ProviderManaged,
			Automatic:         call.Automatic,
			State:             call.State,
			SelectedResult:    selected,
		})
	}

	return view
}

func (r *Runtime) activeRunLocked(
	sessionValue *Session,
) *Run {
	if sessionValue == nil {
		return nil
	}

	for _, runID := range sessionValue.RunIDs {
		run := r.runs[runID]
		if run != nil && !run.State.Terminal() {
			return run
		}
	}

	return nil
}

func transcriptMessageByID(
	messages []TranscriptMessage,
	id string,
) (TranscriptMessage, bool) {
	for _, message := range messages {
		if message.ID == id {
			return message, true
		}
	}
	return TranscriptMessage{}, false
}

func selectHistory(
	messages []TranscriptMessage,
	currentID string,
	selection HistorySelection,
) []TranscriptMessage {
	currentIndex := -1
	for index, message := range messages {
		if message.ID == currentID {
			currentIndex = index
			break
		}
	}
	if currentIndex <= 0 {
		return nil
	}

	beforeCurrent := messages[:currentIndex]
	if selection.All {
		return cloneJSON(beforeCurrent)
	}

	previousTurns := max(selection.IncludePreviousUserTurns, 0)
	targetPureUsers := previousTurns + 1
	pureUsers := 0
	// If fewer than the requested number exist, retain all available history.
	start := 0

	for index := currentIndex; index >= 0; index-- {
		message := messages[index]
		if message.Role != MessageRoleUser || len(message.ToolResults) != 0 {
			continue
		}

		pureUsers++
		if pureUsers >= targetPureUsers {
			start = index
			break
		}
	}

	if start == 0 {
		return cloneJSON(beforeCurrent)
	}

	history := make([]TranscriptMessage, 0, currentIndex)
	for _, message := range messages[:start] {
		if message.Role == MessageRoleSystem ||
			message.Role == MessageRoleDeveloper {
			history = append(history, cloneJSON(message))
		}
	}
	history = append(
		history,
		cloneJSON(messages[start:currentIndex])...,
	)

	return history
}

func hasUserInput(input UserInput) bool {
	return strings.TrimSpace(input.Text) != "" ||
		len(input.Attachments) != 0 ||
		len(input.MCPAppContext) != 0
}

func hasInFlightToolCalls(run *Run) bool {
	for _, call := range run.ToolCalls {
		switch call.State {
		case ToolCallStateEvaluationQueued,
			ToolCallStateEvaluating,
			ToolCallStateExecutionQueued,
			ToolCallStateWaiting,
			ToolCallStateRunning:
			return true
		default:
		}
	}
	return false
}

func allRequiredCallsResolved(run *Run) bool {
	for _, call := range run.ToolCalls {
		if !call.Required ||
			call.ProviderManaged ||
			call.ConsumedByOperationID != "" {
			continue
		}
		if call.State != ToolCallStateResolved ||
			call.SelectedResultID == "" {
			return false
		}
	}
	return true
}

func unresolvedCallsForLatestOperation(run *Run) []ToolCallID {
	operationID := latestBatchOperationID(run)
	output := make([]ToolCallID, 0)

	for _, call := range run.ToolCalls {
		if call.OriginOperationID != operationID ||
			!call.Required ||
			call.ProviderManaged ||
			call.State == ToolCallStateResolved ||
			call.State == ToolCallStateObserved ||
			call.State == ToolCallStateCancelled {
			continue
		}
		output = append(output, call.ID)
	}

	return output
}

func latestBatchOperationID(run *Run) OperationID {
	if run == nil {
		return ""
	}

	for index := range slices.Backward(run.ModelOperations) {
		operation := run.ModelOperations[index]
		for _, call := range run.ToolCalls {
			if call.OriginOperationID == operation.ID {
				return operation.ID
			}
		}
	}

	return ""
}

func findToolCall(
	run *Run,
	id ToolCallID,
) *ToolCall {
	for index := range run.ToolCalls {
		if run.ToolCalls[index].ID == id {
			return &run.ToolCalls[index]
		}
	}
	return nil
}

func findWaitingRequest(
	run *Run,
	id WaitID,
) *WaitingRequest {
	for index := range run.WaitingRequests {
		if run.WaitingRequests[index].ID == id {
			return &run.WaitingRequests[index]
		}
	}
	return nil
}

func selectedToolResult(call ToolCall) *ToolResult {
	if call.SelectedResultID == "" {
		return nil
	}

	for index := range call.Results {
		if call.Results[index].ID == call.SelectedResultID {
			copyValue := call.Results[index]
			return &copyValue
		}
	}

	return nil
}

func selectToolResult(
	call *ToolCall,
	result ToolResult,
) {
	call.Results = append(call.Results, result)
	call.SelectedResultID = result.ID
}

func collectUnconsumedToolResults(run *Run) []ToolResult {
	output := make([]ToolResult, 0)

	for _, call := range run.ToolCalls {
		if !isContinuationToolResult(call) {
			continue
		}
		if call.ConsumedByOperationID != "" {
			continue
		}
		if result := selectedToolResult(call); result != nil {
			output = append(output, cloneJSON(*result))
		}
	}

	return output
}

func removeRequestedCall(
	run *Run,
	callID ToolCallID,
) {
	if len(run.RequestedCallIDs) == 0 {
		return
	}

	next := make([]ToolCallID, 0, len(run.RequestedCallIDs))
	for _, current := range run.RequestedCallIDs {
		if current != callID {
			next = append(next, current)
		}
	}
	run.RequestedCallIDs = next
}

func appendUniqueToolCallIDs(
	existing []ToolCallID,
	incoming []ToolCallID,
) []ToolCallID {
	seen := make(map[ToolCallID]struct{}, len(existing)+len(incoming))
	output := make([]ToolCallID, 0, len(existing)+len(incoming))

	for _, id := range existing {
		if id == "" {
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		output = append(output, id)
	}
	for _, id := range incoming {
		if id == "" {
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		output = append(output, id)
	}

	return output
}

func uniqueToolCallIDs(values []ToolCallID) []ToolCallID {
	return appendUniqueToolCallIDs(nil, values)
}

func artifactReferencesEqual(
	left []ArtifactReference,
	right []ArtifactReference,
) bool {
	left = uniqueArtifactReferences(left)
	right = uniqueArtifactReferences(right)
	if len(left) != len(right) {
		return false
	}

	leftKeys := make(map[string]struct{}, len(left))
	for _, reference := range left {
		leftKeys[reference.RootID+"\x00"+reference.ArtifactID] = struct{}{}
	}
	for _, reference := range right {
		key := reference.RootID + "\x00" + reference.ArtifactID
		if _, found := leftKeys[key]; !found {
			return false
		}
		delete(leftKeys, key)
	}
	return len(leftKeys) == 0
}

func filterReferences(
	values []ArtifactReference,
	allowed []ArtifactReference,
) []ArtifactReference {
	allowedByKey := make(map[string]struct{}, len(allowed))
	for _, reference := range allowed {
		allowedByKey[reference.RootID+"\x00"+reference.ArtifactID] = struct{}{}
	}

	output := make([]ArtifactReference, 0, len(values))
	for _, reference := range values {
		if _, found := allowedByKey[reference.RootID+"\x00"+reference.ArtifactID]; found {
			output = append(output, reference)
		}
	}

	return output
}

func cloneRaw(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}
