package session

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
)

type AppToolPreparationRequest struct {
	SessionID SessionID
	RunID     RunID
	App       AppInstance
	ToolName  string
	Arguments json.RawMessage
}

type AppToolPreparation struct {
	Proposal ToolCallProposal
}

type AppResourceReadRequest struct {
	SessionID   SessionID
	RunID       RunID
	App         AppInstance
	ViewID      AppViewID
	ResourceURI string
}

type AppResourceReadResult struct {
	Content json.RawMessage `json:"content"`
}

type AppContextStagingRequest struct {
	SessionID SessionID
	RunID     RunID
	App       AppInstance
	ViewID    AppViewID
	Update    AppContextUpdate
}

type AppExecutor interface {
	PrepareAppTool(
		ctx context.Context,
		request AppToolPreparationRequest,
	) (AppToolPreparation, error)

	ReadAppResource(
		ctx context.Context,
		request AppResourceReadRequest,
	) (AppResourceReadResult, error)

	ValidateAppContext(
		ctx context.Context,
		request AppContextStagingRequest,
	) (AppContextUpdate, error)
}

type OpenAppViewCommand struct {
	Meta          CommandMeta   `json:"meta"`
	SessionID     SessionID     `json:"sessionID"`
	RunID         RunID         `json:"runID"`
	AppInstanceID AppInstanceID `json:"appInstanceID"`
	ViewID        AppViewID     `json:"viewID"`
}

type ExecuteAppToolCommand struct {
	Meta          CommandMeta     `json:"meta"`
	SessionID     SessionID       `json:"sessionID"`
	RunID         RunID           `json:"runID"`
	AppInstanceID AppInstanceID   `json:"appInstanceID"`
	ViewID        AppViewID       `json:"viewID"`
	ToolName      string          `json:"toolName"`
	Arguments     json.RawMessage `json:"arguments,omitempty"`
}

type StageAppContextCommand struct {
	Meta          CommandMeta      `json:"meta"`
	SessionID     SessionID        `json:"sessionID"`
	RunID         RunID            `json:"runID"`
	AppInstanceID AppInstanceID    `json:"appInstanceID"`
	ViewID        AppViewID        `json:"viewID"`
	Update        AppContextUpdate `json:"update"`
}

type ReadAppResourceRequest struct {
	SessionID     SessionID     `json:"sessionID"`
	RunID         RunID         `json:"runID"`
	AppInstanceID AppInstanceID `json:"appInstanceID"`
	ViewID        AppViewID     `json:"viewID"`
	ResourceURI   string        `json:"resourceURI,omitempty"`
}

func (r *Runtime) OpenAppView(
	ctx context.Context,
	command OpenAppViewCommand,
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

	sessionValue, run, err := r.sessionAndRunLocked(
		command.SessionID,
		command.RunID,
	)
	if err != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				RunID:     command.RunID,
				Message:   err.Error(),
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

	app := findAppInstance(run, command.AppInstanceID)
	if app == nil || app.Route == nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "app instance is unavailable",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if strings.TrimSpace(string(command.ViewID)) == "" {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "app view ID is required",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	if !slices.Contains(app.Views, command.ViewID) {
		app.Views = append(app.Views, command.ViewID)
		r.touchRunLocked(sessionValue, run)
	}

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		command.Meta.ID,
		RuntimeEventAppViewOpened,
		app.OriginOperationID,
		app.OriginToolCallID,
		"",
		"",
		AppViewEventData{
			AppInstanceID: app.ID,
			ViewID:        command.ViewID,
		},
	)

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
		RuntimeEventAppViewOpened,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)
	return &receipt, nil
}

func (r *Runtime) ExecuteAppTool(
	ctx context.Context,
	command ExecuteAppToolCommand,
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

	sessionValue, run, err := r.sessionAndRunLocked(
		command.SessionID,
		command.RunID,
	)
	if err != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				RunID:     command.RunID,
				Message:   err.Error(),
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
	if run.State.Terminal() ||
		run.Controls.CancellationRequested ||
		run.Controls.Paused ||
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
				Message:         "run is not dispatchable for app tool execution",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	app := findAppInstance(run, command.AppInstanceID)
	if app == nil ||
		app.Route == nil ||
		!slices.Contains(app.Views, command.ViewID) {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "app instance is not attached to this view",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	appSnapshot := cloneAppInstanceForRuntime(*app)
	sessionRevision := sessionValue.Revision
	runRevision := run.Revision
	r.mu.Unlock()

	prepared, prepareErr := r.deps.Apps.PrepareAppTool(
		ctx,
		AppToolPreparationRequest{
			SessionID: command.SessionID,
			RunID:     command.RunID,
			App:       appSnapshot,
			ToolName:  strings.TrimSpace(command.ToolName),
			Arguments: cloneRaw(command.Arguments),
		},
	)

	r.mu.Lock()
	if r.closed {
		delete(r.commandReservations, command.Meta.ID)
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	sessionValue, run, err = r.sessionAndRunLocked(
		command.SessionID,
		command.RunID,
	)
	if err != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				RunID:     command.RunID,
				Message:   err.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(receipt)
	}
	if sessionValue.Revision != sessionRevision ||
		run.Revision != runRevision ||
		run.Controls.Paused ||
		run.State.Terminal() ||
		run.Controls.CancellationRequested ||
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
				Message:         "run changed while app tool execution was prepared; retry",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if prepareErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         prepareErr.Error(),
			},
		)
		r.mu.Unlock()
		//nolint:nilerr // Explicit.
		return &receipt, nil
	}

	app = findAppInstance(run, command.AppInstanceID)
	if app == nil ||
		app.Route == nil ||
		!slices.Contains(app.Views, command.ViewID) {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "app instance changed while tool execution was prepared",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	origin := findModelOperation(run, app.OriginOperationID)
	if origin == nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "app origin operation is unavailable",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	proposal := prepared.Proposal
	callID := proposal.ID
	if callID == "" || findToolCall(run, callID) != nil {
		callID = ToolCallID(r.newID())
	}
	call := ToolCall{
		ID:                callID,
		OriginOperationID: origin.ID,
		ProviderCallID:    proposal.ProviderCallID,
		Name:              proposal.Name,
		ChoiceID:          proposal.ChoiceID,
		Type:              proposal.Type,
		Arguments:         cloneRaw(proposal.Arguments),
		Binding:           cloneJSON(proposal.Binding),
		Required:          false,
		ProviderManaged:   false,
		Automatic:         false,
		Source:            ToolCallSourceApp,
		AppInstanceID:     app.ID,
		State:             ToolCallStateEvaluationQueued,
	}
	run.ToolCalls = append(run.ToolCalls, call)
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		command.Meta.ID,
		RuntimeEventToolCallProposed,
		call.OriginOperationID,
		call.ID,
		"",
		"",
		call,
	)
	effects.work = append(effects.work, runtimeWork{
		Kind:       workToolEvaluate,
		SessionID:  sessionValue.ID,
		RunID:      run.ID,
		ToolCallID: call.ID,
	})

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
		RuntimeEventToolCallProposed,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)
	return &receipt, nil
}

func (r *Runtime) StageAppContext(
	ctx context.Context,
	command StageAppContextCommand,
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

	sessionValue, run, err := r.sessionAndRunLocked(
		command.SessionID,
		command.RunID,
	)
	if err != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				RunID:     command.RunID,
				Message:   err.Error(),
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
				Message:         "terminal or cancelling runs cannot stage App context",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	app := findAppInstance(run, command.AppInstanceID)
	if app == nil ||
		app.Route == nil ||
		!slices.Contains(app.Views, command.ViewID) {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "app instance is not attached to this view",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	appSnapshot := cloneAppInstanceForRuntime(*app)
	sessionRevision := sessionValue.Revision
	runRevision := run.Revision
	r.mu.Unlock()

	update, validateErr := r.deps.Apps.ValidateAppContext(
		ctx,
		AppContextStagingRequest{
			SessionID: command.SessionID,
			RunID:     command.RunID,
			App:       appSnapshot,
			ViewID:    command.ViewID,
			Update:    cloneJSON(command.Update),
		},
	)

	r.mu.Lock()
	if r.closed {
		delete(r.commandReservations, command.Meta.ID)
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}

	sessionValue, run, err = r.sessionAndRunLocked(
		command.SessionID,
		command.RunID,
	)
	if err != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				RunID:     command.RunID,
				Message:   err.Error(),
			},
		)
		r.mu.Unlock()
		return commandReceiptResult(receipt)
	}
	if sessionValue.Revision != sessionRevision || run.Revision != runRevision {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "run changed while app context was validated; retry",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}
	if validateErr != nil {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         validateErr.Error(),
			},
		)
		r.mu.Unlock()
		//nolint:nilerr // Explicit.
		return &receipt, nil
	}

	app = findAppInstance(run, command.AppInstanceID)
	if app == nil ||
		app.Route == nil ||
		!slices.Contains(app.Views, command.ViewID) {
		receipt := r.rejectReceiptLocked(
			command.Meta,
			fingerprint,
			CommandReceipt{
				CommandID:       command.Meta.ID,
				SessionID:       command.SessionID,
				RunID:           command.RunID,
				SessionRevision: sessionValue.Revision,
				RunRevision:     run.Revision,
				Message:         "app instance changed while context was validated",
			},
		)
		r.mu.Unlock()
		return &receipt, nil
	}

	update.AppInstanceID = app.ID
	update.ViewID = command.ViewID
	update.ServerID = app.ServerID
	if update.UpdatedAt.IsZero() {
		update.UpdatedAt = r.now()
	}

	run.Continuation.MCPAppContext = append(
		run.Continuation.MCPAppContext,
		cloneJSON(update),
	)
	run.Continuation.Revision++
	run.Continuation.UpdatedAt = r.now()
	r.touchRunLocked(sessionValue, run)

	effects := runtimeEffects{}
	r.recordRunEventLocked(
		&effects,
		sessionValue,
		run,
		command.Meta.ID,
		RuntimeEventAppContextStaged,
		app.OriginOperationID,
		app.OriginToolCallID,
		"",
		"",
		AppContextEventData{
			AppInstanceID: app.ID,
			ViewID:        command.ViewID,
			Revision:      run.Continuation.Revision,
		},
	)

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
		RuntimeEventAppContextStaged,
		sessionValue,
		run,
	)
	r.mu.Unlock()
	r.flushEffects(ctx, effects)
	return &receipt, nil
}

func (r *Runtime) ReadAppResource(
	ctx context.Context,
	request ReadAppResourceRequest,
) (AppResourceReadResult, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return AppResourceReadResult{}, ErrRuntimeClosed
	}

	sessionValue, run, err := r.sessionAndRunLocked(
		request.SessionID,
		request.RunID,
	)
	if err != nil {
		r.mu.RUnlock()
		return AppResourceReadResult{}, err
	}
	_ = sessionValue

	app := findAppInstance(run, request.AppInstanceID)
	if app == nil ||
		app.Route == nil ||
		!slices.Contains(app.Views, request.ViewID) {
		r.mu.RUnlock()
		return AppResourceReadResult{}, errors.New(
			"app instance is not attached to this view",
		)
	}
	appSnapshot := cloneAppInstanceForRuntime(*app)
	r.mu.RUnlock()

	return r.deps.Apps.ReadAppResource(
		ctx,
		AppResourceReadRequest{
			SessionID:   request.SessionID,
			RunID:       request.RunID,
			App:         appSnapshot,
			ViewID:      request.ViewID,
			ResourceURI: strings.TrimSpace(request.ResourceURI),
		},
	)
}

func findAppInstance(run *Run, id AppInstanceID) *AppInstance {
	for index := range run.Apps {
		if run.Apps[index].ID == id {
			return &run.Apps[index]
		}
	}
	return nil
}

func (r *Runtime) recordAppInstanceLocked(
	sessionValue *Session,
	run *Run,
	call *ToolCall,
	result *ToolResult,
	route *AppRoute,
	effects *runtimeEffects,
) {
	if route == nil || strings.TrimSpace(route.ServerID) == "" {
		return
	}

	instance := AppInstance{
		ID:                AppInstanceID(r.newID()),
		SessionID:         sessionValue.ID,
		RunID:             run.ID,
		OriginOperationID: call.OriginOperationID,
		OriginToolCallID:  call.ID,
		ServerID:          route.ServerID,
		ResourceURI:       route.ResourceURI,
		CreatedAt:         r.now(),
		Route:             cloneAppRoute(route),
	}
	run.Apps = append(run.Apps, instance)
	result.AppInstanceID = instance.ID

	r.recordRunEventLocked(
		effects,
		sessionValue,
		run,
		"",
		RuntimeEventAppInstanceCreated,
		call.OriginOperationID,
		call.ID,
		"",
		"",
		AppInstanceEventData{
			AppInstance: instance,
		},
	)
}

func (r *Runtime) queueAppToolExecutionLocked(
	sessionValue *Session,
	run *Run,
	call *ToolCall,
	effects *runtimeEffects,
) {
	if call.Required || call.State != ToolCallStateReady {
		return
	}

	call.State = ToolCallStateExecutionQueued
	r.touchRunLocked(sessionValue, run)
	effects.work = append(effects.work, runtimeWork{
		Kind:       workToolExecute,
		SessionID:  sessionValue.ID,
		RunID:      run.ID,
		ToolCallID: call.ID,
	})
}

func cloneAppInstanceForRuntime(value AppInstance) AppInstance {
	output := value
	output.Views = append([]AppViewID(nil), value.Views...)
	output.Route = cloneAppRoute(value.Route)
	return output
}

func cloneAppRoute(value *AppRoute) *AppRoute {
	if value == nil {
		return nil
	}

	output := *value
	output.Binding = ToolBinding{
		Kind: value.Binding.Kind,
		Data: append(
			json.RawMessage(nil),
			value.Binding.Data...,
		),
	}
	return &output
}
