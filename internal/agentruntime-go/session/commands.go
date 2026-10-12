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
)

const (
	runtimeEventStateKey = "state"
	runtimeEventErrorKey = "error"
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

	now := r.now()
	sessionValue := &Session{
		ID:            sessionID,
		Revision:      1,
		Title:         strings.TrimSpace(command.Title),
		CreatedAt:     now,
		UpdatedAt:     now,
		Configuration: cloneJSON(command.Configuration),
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

	sessionValue.Configuration = cloneJSON(command.Configuration)
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
			map[string]string{"title": title},
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

func (r *Runtime) ApplyAgent(
	ctx context.Context,
	command ApplyAgentCommand,
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

	nextConfiguration := cloneJSON(command.Application.Configuration)
	nextConfiguration.Agent = &command.Application.Agent
	if command.Application.PreserveWorkspace &&
		len(nextConfiguration.WorkspaceSelection) == 0 {
		nextConfiguration.WorkspaceSelection = cloneRaw(
			sessionValue.Configuration.WorkspaceSelection,
		)
	}

	sessionValue.Configuration = nextConfiguration
	r.touchSessionLocked(sessionValue)

	receipt := CommandReceipt{
		CommandID:       command.Meta.ID,
		Accepted:        true,
		SessionID:       command.SessionID,
		SessionRevision: sessionValue.Revision,
		StarterText:     command.Application.StarterText,
	}
	r.finishCommandLocked(command.Meta, fingerprint, receipt)

	effects := runtimeEffects{}
	r.recordSessionEventLocked(
		&effects,
		sessionValue,
		command.Meta.ID,
		RuntimeEventSessionConfigurationChanged,
		map[string]any{
			"agent": command.Application.Agent,
		},
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

	effects := runtimeEffects{}
	receipt, startErr := r.startRunLocked(
		sessionValue,
		command.Meta.ID,
		command.RunID,
		command.Input,
		command.Configuration,
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
	if run.State.Terminal() {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "cannot update continuation input for a terminal run",
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

	input := cloneJSON(command.Input)
	input.Revision = run.Continuation.Revision + 1
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
		map[string]Revision{"revision": input.Revision},
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

	run.ContinuationRequested = true
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
	work, continuationErr := r.beginContinuationModelLocked(
		sessionValue,
		run,
		command.Meta.ID,
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
	if run.Controls.Paused || run.Controls.CancellationRequested || run.State.Terminal() {
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
					Message:         fmt.Sprintf("tool call %q is already active", callID),
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
		RuntimeEventRunStateChanged,
		"",
		"",
		"",
		"",
		map[string]any{
			"executeCalls": callIDs,
			"continue":     command.Continue,
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
				Message:         "tool call was not found",
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

	call.SelectedResultID = ""
	call.LastFailure = ""
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
		RuntimeEventRunStateChanged,
		"",
		call.ID,
		"",
		"",
		map[string]string{"action": "retry"},
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
		delete(r.waitReferences, wait.ID)

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
				map[string]string{
					runtimeEventStateKey: string(RunStateWaiting),
				},
			)
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
		RuntimeEventWaitResolved,
		wait.OperationID,
		wait.ToolCallID,
		"",
		wait.ID,
		map[string]string{"resolution": string(command.Resolution)},
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
		r.advanceDriverLocked(
			sessionValue,
			run,
			command.Meta.ID,
			DriverEvent{Kind: DriverEventControlsChanged},
			&effects,
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
	if run.State.Terminal() {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "run is already terminal",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	run.Controls.CancellationRequested = true
	run.State = RunStateCancelling
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
	for key, cancel := range r.cancels {
		if strings.Contains(key, string(run.ID)) {
			effects.cancels = append(effects.cancels, cancel)
			delete(r.cancels, key)
		}
	}

	if run.ActiveModelOperationID == "" && !hasInFlightToolCalls(run) {
		r.markRunTerminalLocked(
			sessionValue,
			run,
			command.Meta.ID,
			RunOutcome{
				Kind:    RunOutcomeCancelled,
				Code:    string(RunStateCancelled),
				Message: strings.TrimSpace(command.Reason),
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
			map[string]string{
				runtimeEventStateKey: string(RunStateCancelling),
			},
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
				Message:         "editing requires an explicit cancellation or a safe terminal boundary",
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

	sessionValue.Transcript = append(
		[]TranscriptMessage(nil),
		sessionValue.Transcript[:index]...,
	)
	r.touchSessionLocked(sessionValue)

	effects := runtimeEffects{}
	receipt, startErr := r.startRunLocked(
		sessionValue,
		command.Meta.ID,
		command.RunID,
		command.Input,
		command.Configuration,
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

	fingerprint = commandFingerprint(payload)
	if previous, found := r.commandRecords[meta.ID]; found {
		if previous.Fingerprint != fingerprint {
			return CommandReceipt{}, false, "", ErrCommandIDConflict
		}
		return cloneJSON(previous.Receipt), true, fingerprint, nil
	}

	return CommandReceipt{}, false, fingerprint, nil
}

func (r *Runtime) finishCommandLocked(
	meta CommandMeta,
	fingerprint string,
	receipt CommandReceipt,
) {
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

func commandFingerprint(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
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
	requestedRunID RunID,
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

	runID := requestedRunID
	if runID == "" {
		runID = RunID(r.newID())
	}
	if _, exists := r.runs[runID]; exists {
		return CommandReceipt{}, fmt.Errorf("run %q already exists", runID)
	}

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
			Enabled:    cloneJSON(runConfiguration.InstalledSkills),
			Active:     nil,
		},
	}
	if limits != nil {
		run.Limits = cloneJSON(*limits)
	}

	inputMessage := TranscriptMessage{
		ID:            r.newID(),
		CreatedAt:     now,
		Role:          MessageRoleUser,
		Status:        "accepted",
		Text:          input.Text,
		Attachments:   cloneJSON(input.Attachments),
		MCPAppContext: cloneRaw(input.MCPAppContext),
		Metadata:      cloneRaw(input.Metadata),
	}

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
		map[string]string{
			runtimeEventStateKey: string(RunStateQueued),
		},
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
		map[string]string{
			runtimeEventStateKey: string(RunStateRunning),
		},
	)

	work, err := r.startModelOperationLocked(
		sessionValue,
		run,
		commandID,
		inputMessage.ID,
		effects,
	)
	if err != nil {
		delete(r.runs, runID)
		delete(r.runToSession, runID)
		sessionValue.RunIDs = sessionValue.RunIDs[:len(sessionValue.RunIDs)-1]
		sessionValue.Transcript = sessionValue.Transcript[:len(sessionValue.Transcript)-1]
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
	currentMessageID string,
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

	history := selectHistory(
		sessionValue.Transcript,
		currentMessageID,
		run.Configuration.History,
	)
	operationID := OperationID(r.newID())
	operation := ModelOperation{
		ID:    operationID,
		State: OperationStateQueued,
		Request: ModelOperationInput{
			Configuration:        cloneJSON(run.Configuration),
			History:              cloneJSON(history),
			Current:              cloneJSON(current),
			ContinuationRevision: run.Continuation.ConsumedRevision,
		},
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

	return runtimeWork{
		Kind:        workModel,
		SessionID:   sessionValue.ID,
		RunID:       run.ID,
		OperationID: operationID,
	}, nil
}

func (r *Runtime) beginContinuationModelLocked(
	sessionValue *Session,
	run *Run,
	commandID CommandID,
	effects *runtimeEffects,
) (runtimeWork, error) {
	if run.ActiveModelOperationID != "" {
		return runtimeWork{}, errors.New("a model operation is already active")
	}
	if !allRequiredCallsResolved(run) {
		return runtimeWork{}, errors.New("required tool calls remain unresolved")
	}

	if run.Continuation.Configuration != nil {
		r.applyRunConfigurationLocked(
			sessionValue,
			run,
			*run.Continuation.Configuration,
		)
	}

	results := collectUnconsumedToolResults(run)
	hasInput := strings.TrimSpace(run.Continuation.Text) != "" ||
		len(run.Continuation.Attachments) != 0 ||
		len(run.Continuation.MCPAppContext) != 0 ||
		len(results) != 0

	if !hasInput {
		return runtimeWork{}, errors.New(
			"continuation requires tool results, text, attachments, MCP App context, or configuration that produces an input",
		)
	}

	inputMessage := TranscriptMessage{
		ID:            r.newID(),
		CreatedAt:     r.now(),
		Role:          MessageRoleUser,
		Status:        "continuation",
		Text:          run.Continuation.Text,
		Attachments:   cloneJSON(run.Continuation.Attachments),
		ToolResults:   cloneJSON(results),
		MCPAppContext: cloneRaw(run.Continuation.MCPAppContext),
		Metadata:      cloneRaw(run.Continuation.Metadata),
	}
	sessionValue.Transcript = append(sessionValue.Transcript, inputMessage)

	run.Continuation.ConsumedRevision = run.Continuation.Revision
	run.ContinuationRequested = false

	work, err := r.startModelOperationLocked(
		sessionValue,
		run,
		commandID,
		inputMessage.ID,
		effects,
	)
	if err != nil {
		sessionValue.Transcript = sessionValue.Transcript[:len(sessionValue.Transcript)-1]
		return runtimeWork{}, err
	}

	for callIndex := range run.ToolCalls {
		call := &run.ToolCalls[callIndex]
		if call.ConsumedByOperationID != "" ||
			call.SelectedResultID == "" {
			continue
		}
		call.ConsumedByOperationID = work.OperationID
	}

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
		map[string]Revision{
			"revision": run.Continuation.ConsumedRevision,
		},
	)

	return work, nil
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
				Message:         "tool call was not found",
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
		map[string]bool{
			"paused": paused,
		},
	)
	if !paused {
		r.advanceDriverLocked(
			sessionValue,
			run,
			meta.ID,
			DriverEvent{
				Kind: DriverEventResumed,
			},
			&effects,
		)
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

func (r *Runtime) applyRunConfigurationLocked(
	sessionValue *Session,
	run *Run,
	configuration SessionConfiguration,
) {
	previousSkills := append(
		append([]ArtifactReference(nil), run.Configuration.InstalledSkills...),
		run.Configuration.WorkspaceSkills...,
	)
	nextSkills := append(
		append([]ArtifactReference(nil), configuration.InstalledSkills...),
		configuration.WorkspaceSkills...,
	)

	run.Configuration = cloneJSON(configuration)
	if artifactReferencesEqual(previousSkills, nextSkills) {
		return
	}

	run.SkillSession.Generation++
	run.SkillSession.Enabled = cloneJSON(nextSkills)
	run.SkillSession.Active = filterReferences(
		run.SkillSession.Active,
		run.SkillSession.Enabled,
	)
	r.touchRunLocked(sessionValue, run)
}

func (r *Runtime) advanceDriverLocked(
	sessionValue *Session,
	run *Run,
	commandID CommandID,
	event DriverEvent,
	effects *runtimeEffects,
) {
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
	transition, err := driver.Advance(run.Driver, event)
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
			map[string]string{
				runtimeEventStateKey: string(run.State),
			},
		)
	}

	for _, intent := range transition.Operations {
		switch intent.Kind {
		case OperationIntentStartModel:
			work, startErr := r.beginContinuationModelLocked(
				sessionValue,
				run,
				commandID,
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
			effects.work = append(effects.work, runtimeWork{
				Kind:       workToolEvaluate,
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

	targetPureUsers := selection.IncludePreviousUserTurns + 1
	pureUsers := 0
	start := currentIndex

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
		case ToolCallStateEvaluating,
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
			call.State == ToolCallStateObserved {
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
	if len(left) != len(right) {
		return false
	}

	leftKeys := make(map[string]struct{}, len(left))
	for _, reference := range left {
		leftKeys[reference.RootID+"\x00"+reference.ArtifactID] = struct{}{}
	}
	for _, reference := range right {
		if _, found := leftKeys[reference.RootID+"\x00"+reference.ArtifactID]; !found {
			return false
		}
	}

	return true
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
