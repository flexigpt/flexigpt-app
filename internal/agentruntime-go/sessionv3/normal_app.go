package sessionv3

import (
	"context"
	"errors"
	"slices"
	"strings"
)

func (r *Runtime) OpenAppView(
	ctx context.Context,
	command OpenAppViewCommand,
) (*CommandReceipt, error) {
	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		command,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			app := mutableAppInstance(execution, command.AppInstanceID)
			if app == nil {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     "App instance is unavailable",
				}, nil
			}
			if strings.TrimSpace(string(command.ViewID)) == "" {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     "App view ID is required",
				}, nil
			}
			if slices.Contains(app.Views, command.ViewID) {
				return &CommandReceipt{
					CommandID:         command.Meta.ID,
					Accepted:          true,
					SessionID:         session.ID,
					ExecutionID:       execution.ID,
					SessionRevision:   session.Revision,
					ExecutionRevision: execution.Revision,
					AppInstanceIDs:    []AppInstanceID{app.ID},
				}, nil
			}
			app.Views = append(app.Views, command.ViewID)
			r.touchExecutionLocked(session, execution)
			return &CommandReceipt{
				CommandID:         command.Meta.ID,
				Accepted:          true,
				SessionID:         session.ID,
				ExecutionID:       execution.ID,
				SessionRevision:   session.Revision,
				ExecutionRevision: execution.Revision,
				AppInstanceIDs:    []AppInstanceID{app.ID},
			}, nil
		},
	)
}

func (r *Runtime) ExecuteAppTool(
	ctx context.Context,
	command ExecuteAppToolCommand,
) (*CommandReceipt, error) {
	if command.Meta.ID == "" {
		return nil, errors.New("command ID is required")
	}
	if !command.Arguments.Valid() {
		return &CommandReceipt{
			CommandID:   command.Meta.ID,
			SessionID:   command.SessionID,
			ExecutionID: command.ExecutionID,
			Message:     "App Tool arguments are not canonical JSON",
		}, nil
	}

	executor, err := r.appExecutor()
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}
	session, execution, err := r.executionLocked(
		command.SessionID,
		command.ExecutionID,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if receipt, found, receiptErr := r.commandReceiptLocked(
		session,
		command.Meta,
		command,
	); receiptErr != nil || found {
		r.mu.Unlock()
		return receipt, receiptErr
	}
	if message := expectedExecutionRevision(session, execution, command.Meta); message != "" {
		receipt := r.rejectLocked(session, command.Meta, command, message)
		r.mu.Unlock()
		return receipt, nil
	}
	if execution.State.Terminal() ||
		execution.Controls.CancellationRequested {
		receipt := r.rejectLocked(
			session,
			command.Meta,
			command,
			"terminal or cancelling executions cannot execute App Tools",
		)
		r.mu.Unlock()
		return receipt, nil
	}
	app := mutableAppInstance(execution, command.AppInstanceID)
	if app == nil || !containsAppView(*app, command.ViewID) {
		receipt := r.rejectLocked(
			session,
			command.Meta,
			command,
			"App instance is not attached to this view",
		)
		r.mu.Unlock()
		return receipt, nil
	}
	snapshot := cloneAppInstance(*app)
	sessionRevision := session.Revision
	executionRevision := execution.Revision
	r.mu.Unlock()

	call, prepareErr := executor.PrepareAppTool(
		ctx,
		AppToolPreparationRequest{
			SessionID:   command.SessionID,
			ExecutionID: command.ExecutionID,
			App:         snapshot,
			ToolName:    strings.TrimSpace(command.ToolName),
			Arguments:   command.Arguments.Clone(),
		},
	)

	var (
		ready           []readyExecutionWork
		launchSessionID SessionID
	)
	r.mu.Lock()
	defer func() {
		r.mu.Unlock()
		if len(ready) != 0 {
			r.launchReadyWork(ctx, launchSessionID, ready)
		}
	}()
	if r.closed {
		return nil, ErrRuntimeClosed
	}
	session, execution, err = r.executionLocked(
		command.SessionID,
		command.ExecutionID,
	)
	if err != nil {
		return nil, err
	}
	if receipt, found, receiptErr := r.commandReceiptLocked(
		session,
		command.Meta,
		command,
	); receiptErr != nil || found {
		return receipt, receiptErr
	}
	if session.Revision != sessionRevision ||
		execution.Revision != executionRevision {
		return r.rejectLocked(
			session,
			command.Meta,
			command,
			"execution changed while App Tool execution was prepared; retry",
		), nil
	}
	if prepareErr != nil {
		return r.rejectLocked(
			session,
			command.Meta,
			command,
			prepareErr.Error(),
		), nil
	}
	if execution.State.Terminal() ||
		execution.Controls.CancellationRequested {
		return r.rejectLocked(
			session,
			command.Meta,
			command,
			"terminal or cancelling executions cannot execute App Tools",
		), nil
	}
	app = mutableAppInstance(execution, command.AppInstanceID)
	if app == nil || !containsAppView(*app, command.ViewID) {
		return r.rejectLocked(
			session,
			command.Meta,
			command,
			"App instance changed while App Tool execution was prepared",
		), nil
	}

	frame := mutableFrame(execution, app.OriginFrameID)
	if frame == nil {
		return r.rejectLocked(
			session,
			command.Meta,
			command,
			"App origin ProgramFrame is unavailable",
		), nil
	}

	if call.ID == "" || mutableWorkItem(execution, WorkItemID(call.ID)) != nil {
		call.ID = r.newID()
	}
	call.Source = ToolCallSourceApp
	call.AppInstanceID = app.ID
	if err := call.Validate(); err != nil {
		return r.rejectLocked(
			session,
			command.Meta,
			command,
			err.Error(),
		), nil
	}

	now := r.now()
	item := WorkItem{
		ID:          WorkItemID(call.ID),
		FrameID:     frame.ID,
		Type:        WorkItemTypeTool,
		State:       WorkItemStateReady,
		ProgramPath: frame.Path + "/app/" + call.ID,
		Effect: EffectContract{
			Kind: EffectKindNonIdempotent,
		},
		Automatic: true,
		Requested: true,
		Input: WorkItemPayload{
			Tool: &ToolWork{Call: call},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := item.Validate(); err != nil {
		return r.rejectLocked(
			session,
			command.Meta,
			command,
			err.Error(),
		), nil
	}

	execution.WorkItems = append(execution.WorkItems, item)
	frame.WorkItemIDs = append(frame.WorkItemIDs, item.ID)
	r.touchExecutionLocked(session, execution)

	receipt := CommandReceipt{
		CommandID:         command.Meta.ID,
		Accepted:          true,
		SessionID:         session.ID,
		ExecutionID:       execution.ID,
		SessionRevision:   session.Revision,
		ExecutionRevision: execution.Revision,
		WorkItemIDs:       []WorkItemID{item.ID},
	}
	r.rememberCommandLocked(session, command.Meta, command, receipt)
	if err := r.persistLocked(ctx, "appToolQueued", session); err != nil {
		return nil, err
	}
	ready = r.readyWorkForSessionLocked(session)
	launchSessionID = session.ID
	r.publishLocked(ctx, session, execution.ID, RuntimeEventWorkChanged)
	return &receipt, nil
}

func (r *Runtime) StageAppContext(
	ctx context.Context,
	command StageAppContextCommand,
) (*CommandReceipt, error) {
	if command.Meta.ID == "" {
		return nil, errors.New("command ID is required")
	}
	executor, err := r.appExecutor()
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}
	session, execution, err := r.executionLocked(
		command.SessionID,
		command.ExecutionID,
	)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if receipt, found, receiptErr := r.commandReceiptLocked(
		session,
		command.Meta,
		command,
	); receiptErr != nil || found {
		r.mu.Unlock()
		return receipt, receiptErr
	}
	if message := expectedExecutionRevision(session, execution, command.Meta); message != "" {
		receipt := r.rejectLocked(session, command.Meta, command, message)
		r.mu.Unlock()
		return receipt, nil
	}
	app := mutableAppInstance(execution, command.AppInstanceID)
	if app == nil || !containsAppView(*app, command.ViewID) {
		receipt := r.rejectLocked(
			session,
			command.Meta,
			command,
			"App instance is not attached to this view",
		)
		r.mu.Unlock()
		return receipt, nil
	}
	snapshot := cloneAppInstance(*app)
	sessionRevision := session.Revision
	executionRevision := execution.Revision
	r.mu.Unlock()

	update, validateErr := executor.ValidateAppContext(
		ctx,
		AppContextStagingRequest{
			SessionID:   command.SessionID,
			ExecutionID: command.ExecutionID,
			App:         snapshot,
			ViewID:      command.ViewID,
			Update:      cloneAppContextUpdate(command.Update),
		},
	)

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, ErrRuntimeClosed
	}
	session, execution, err = r.executionLocked(
		command.SessionID,
		command.ExecutionID,
	)
	if err != nil {
		return nil, err
	}
	if receipt, found, receiptErr := r.commandReceiptLocked(
		session,
		command.Meta,
		command,
	); receiptErr != nil || found {
		return receipt, receiptErr
	}
	if session.Revision != sessionRevision ||
		execution.Revision != executionRevision {
		return r.rejectLocked(
			session,
			command.Meta,
			command,
			"execution changed while App context was prepared; retry",
		), nil
	}
	if validateErr != nil {
		return r.rejectLocked(
			session,
			command.Meta,
			command,
			validateErr.Error(),
		), nil
	}
	app = mutableAppInstance(execution, command.AppInstanceID)
	if app == nil || !containsAppView(*app, command.ViewID) {
		return r.rejectLocked(
			session,
			command.Meta,
			command,
			"App instance changed while App context was prepared",
		), nil
	}

	update.AppInstanceID = app.ID
	update.ViewID = command.ViewID
	update.ServerID = app.ServerID
	if update.UpdatedAt.IsZero() {
		update.UpdatedAt = r.now()
	}

	execution.Continuation.AppContext = append(
		execution.Continuation.AppContext,
		cloneAppContextUpdate(update),
	)
	execution.Continuation.Revision++
	execution.Continuation.UpdatedAt = r.now()
	r.touchExecutionLocked(session, execution)

	receipt := CommandReceipt{
		CommandID:         command.Meta.ID,
		Accepted:          true,
		SessionID:         session.ID,
		ExecutionID:       execution.ID,
		SessionRevision:   session.Revision,
		ExecutionRevision: execution.Revision,
		AppInstanceIDs:    []AppInstanceID{app.ID},
	}
	r.rememberCommandLocked(session, command.Meta, command, receipt)
	if err := r.persistLocked(ctx, "appContextStaged", session); err != nil {
		return nil, err
	}
	r.publishLocked(ctx, session, execution.ID, RuntimeEventExecutionChanged)
	return &receipt, nil
}

func (r *Runtime) ReadAppResource(
	ctx context.Context,
	request AppResourceReadRequest,
) (AppResourceReadResult, error) {
	executor, err := r.appExecutor()
	if err != nil {
		return AppResourceReadResult{}, err
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return AppResourceReadResult{}, ErrRuntimeClosed
	}
	_, execution, err := r.executionLocked(
		request.SessionID,
		request.ExecutionID,
	)
	if err != nil {
		r.mu.RUnlock()
		return AppResourceReadResult{}, err
	}
	app := mutableAppInstance(execution, request.AppInstanceID)
	if app == nil || !containsAppView(*app, request.ViewID) {
		r.mu.RUnlock()
		return AppResourceReadResult{}, errors.New(
			"app instance is not attached to this view",
		)
	}
	snapshot := cloneAppInstance(*app)
	r.mu.RUnlock()

	return executor.ReadAppResource(
		ctx,
		AppResourceReadRequest{
			SessionID:     request.SessionID,
			ExecutionID:   request.ExecutionID,
			AppInstanceID: request.AppInstanceID,
			ViewID:        request.ViewID,
			ResourceURI:   strings.TrimSpace(request.ResourceURI),
			App:           snapshot,
		},
	)
}

func (r *Runtime) appExecutor() (AppExecutor, error) {
	if r == nil || r.deps.Binder == nil {
		return nil, errors.New("app executor is unavailable")
	}
	executor, found := r.deps.Binder.(AppExecutor)
	if !found || executor == nil {
		return nil, errors.New("app executor is unavailable")
	}
	return executor, nil
}

func mutableAppInstance(
	execution *Execution,
	id AppInstanceID,
) *AppInstance {
	if execution == nil {
		return nil
	}
	for index := range execution.Apps {
		if execution.Apps[index].ID == id {
			return &execution.Apps[index]
		}
	}
	return nil
}

func containsAppView(
	app AppInstance,
	view AppViewID,
) bool {
	return slices.Contains(app.Views, view)
}

func (r *Runtime) recordAppInstanceLocked(
	execution *Execution,
	item *WorkItem,
) {
	if execution == nil ||
		item == nil ||
		item.Output == nil ||
		item.Output.App == nil ||
		item.Output.AppInstanceID != "" ||
		item.Input.Tool == nil {
		return
	}

	call := item.Input.Tool.Call
	if call.Binding.Kind != ToolBindingKindMCP ||
		call.Binding.MCP == nil {
		return
	}

	serverID := strings.TrimSpace(item.Output.App.ServerID)
	if serverID == "" {
		serverID = call.Binding.MCP.ServerID
	}
	if serverID == "" {
		return
	}

	instance := AppInstance{
		ID:               AppInstanceID(r.newID()),
		ExecutionID:      execution.ID,
		OriginWorkItemID: item.ID,
		OriginFrameID:    item.FrameID,
		ServerID:         serverID,
		ResourceURI:      item.Output.App.ResourceURI,
		Binding:          cloneAppToolBinding(call.Binding),
		CreatedAt:        r.now(),
	}
	execution.Apps = append(execution.Apps, instance)
	item.Output.AppInstanceID = instance.ID
}

func cloneAppInstance(value AppInstance) AppInstance {
	output := value
	output.Views = append([]AppViewID(nil), value.Views...)
	output.Binding = cloneAppToolBinding(value.Binding)
	return output
}

func cloneAppToolBinding(value ToolBinding) ToolBinding {
	output := value
	if value.Go != nil {
		copyValue := *value.Go
		output.Go = &copyValue
	}
	if value.MCP != nil {
		copyValue := *value.MCP
		if value.MCP.ServerArtifact != nil {
			pin := *value.MCP.ServerArtifact
			copyValue.ServerArtifact = &pin
		}
		output.MCP = &copyValue
	}
	if value.Skill != nil {
		copyValue := *value.Skill
		output.Skill = &copyValue
	}
	if value.Provider != nil {
		copyValue := *value.Provider
		if value.Provider.Artifact != nil {
			pin := *value.Provider.Artifact
			copyValue.Artifact = &pin
		}
		output.Provider = &copyValue
	}
	if value.Unresolved != nil {
		copyValue := *value.Unresolved
		output.Unresolved = &copyValue
	}
	return output
}

func cloneAppContextUpdates(values []AppContextUpdate) []AppContextUpdate {
	if len(values) == 0 {
		return nil
	}
	output := make([]AppContextUpdate, len(values))
	for index := range values {
		output[index] = cloneAppContextUpdate(values[index])
	}
	return output
}

func cloneAppContextUpdate(value AppContextUpdate) AppContextUpdate {
	output := value
	output.StructuredContent = value.StructuredContent.Clone()
	output.Content = make([]AppContextContent, len(value.Content))
	for index := range value.Content {
		output.Content[index] = value.Content[index]
		output.Content[index].Data = append(
			[]byte(nil),
			value.Content[index].Data...,
		)
	}
	return output
}

type AppToolPreparationRequest struct {
	SessionID   SessionID
	ExecutionID ExecutionID
	App         AppInstance
	ToolName    string
	Arguments   CanonicalValue
}
