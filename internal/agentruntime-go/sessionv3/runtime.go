package sessionv3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

const maxDurableCommandRecords = 1024

type RuntimeDependencies struct {
	Binder       WorkBinder
	Checkpoints  CheckpointSink
	Recovery     RecoverySource
	Events       EventPublisher
	Workflow     WorkflowAdmission
	Interpreters []ProgramInterpreter
	Now          func() time.Time
	NewID        func() string
}

type Runtime struct {
	mu              sync.RWMutex
	deps            RuntimeDependencies
	sessions        map[SessionID]*Session
	retired         map[SessionID]struct{}
	interpreters    map[string]ProgramInterpreter
	privateBindings map[WorkItemID]BoundWork
	privateWaits    map[WaitID]any
	cancels         map[WorkItemID]context.CancelFunc
	closed          bool
}

func NewRuntime(dependencies RuntimeDependencies) (*Runtime, error) {
	if dependencies.Binder == nil || dependencies.Checkpoints == nil || dependencies.Recovery == nil {
		return nil, errors.New("session Runtime v3 dependencies are incomplete")
	}
	if dependencies.Now == nil {
		dependencies.Now = func() time.Time { return time.Now().UTC() }
	}
	if dependencies.NewID == nil {
		dependencies.NewID = uuidutil.NewUUIDv7
	}
	if len(dependencies.Interpreters) == 0 {
		dependencies.Interpreters = NewBuiltinInterpreters()
	}
	r := &Runtime{
		deps:            dependencies,
		sessions:        map[SessionID]*Session{},
		retired:         map[SessionID]struct{}{},
		interpreters:    map[string]ProgramInterpreter{},
		privateBindings: map[WorkItemID]BoundWork{},
		privateWaits:    map[WaitID]any{},
		cancels:         map[WorkItemID]context.CancelFunc{},
	}
	for _, interpreter := range dependencies.Interpreters {
		if interpreter == nil || interpreter.ID() == "" || interpreter.Version() == 0 {
			return nil, errors.New("invalid ProgramFrame interpreter")
		}
		key := ProgramInterpreterKey(interpreter.ID(), interpreter.Version())
		if _, found := r.interpreters[key]; found {
			return nil, fmt.Errorf("duplicate interpreter %q", key)
		}
		r.interpreters[key] = interpreter
	}
	return r, nil
}

func (r *Runtime) Restore(ctx context.Context) error {
	return r.deps.Recovery.Restore(ctx, runtimeRestorer{runtime: r})
}

func (r *Runtime) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	for _, cancel := range r.cancels {
		cancel()
	}
	clear(r.cancels)
	clear(r.privateBindings)
	clear(r.privateWaits)
	r.mu.Unlock()
}

func (r *Runtime) CreateSession(ctx context.Context, command CreateSessionCommand) (*CommandReceipt, error) {
	if command.Meta.ID == "" {
		return nil, errors.New("command ID is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, ErrRuntimeClosed
	}
	id := command.SessionID
	if id == "" {
		id = SessionID(r.newID())
	}
	if _, retired := r.retired[id]; retired {
		return &CommandReceipt{
			CommandID: command.Meta.ID,
			SessionID: id,
			Message:   "Session ID was previously deleted and cannot be reused",
		}, nil
	}
	if _, found := r.sessions[id]; found {
		return &CommandReceipt{CommandID: command.Meta.ID, SessionID: id, Message: "Session already exists"}, nil
	}
	if !command.Metadata.Valid() || validateDefaults(command.Defaults) != nil {
		return &CommandReceipt{
			CommandID: command.Meta.ID,
			SessionID: id,
			Message:   "invalid Session defaults or metadata",
		}, nil
	}
	now := r.now()
	value := &Session{
		ID:         id,
		Revision:   1,
		Title:      command.Title,
		CreatedAt:  now,
		UpdatedAt:  now,
		Defaults:   command.Defaults,
		Metadata:   command.Metadata.Clone(),
		Timeline:   []Activity{},
		Executions: []Execution{},
	}
	receipt := CommandReceipt{
		CommandID:       command.Meta.ID,
		Accepted:        true,
		SessionID:       id,
		SessionRevision: value.Revision,
	}
	r.rememberCommandLocked(value, command.Meta, command, receipt)
	if err := r.persistLocked(ctx, "sessionCreated", value); err != nil {
		return nil, err
	}
	r.sessions[id] = value
	r.publishLocked(ctx, value, "", RuntimeEventSessionCreated)
	return &receipt, nil
}

func (r *Runtime) RenameSession(ctx context.Context, command RenameSessionCommand) (*CommandReceipt, error) {
	return r.changeSession(
		ctx,
		command.Meta,
		command.SessionID,
		command,
		func(session *Session) (*CommandReceipt, error) {
			session.Title = command.Title
			r.touchSessionLocked(session)
			return &CommandReceipt{
				CommandID:       command.Meta.ID,
				Accepted:        true,
				SessionID:       session.ID,
				SessionRevision: session.Revision,
			}, nil
		},
	)
}

func (r *Runtime) UpdateSessionDefaults(
	ctx context.Context,
	command UpdateSessionDefaultsCommand,
) (*CommandReceipt, error) {
	return r.changeSession(
		ctx,
		command.Meta,
		command.SessionID,
		command,
		func(session *Session) (*CommandReceipt, error) {
			if err := validateDefaults(command.Defaults); err != nil {
				//nolint:nilerr // Validation failure is encoded in receipt, not returned error.
				return &CommandReceipt{
					CommandID: command.Meta.ID,
					SessionID: session.ID,
					Message:   err.Error(),
				}, nil
			}
			session.Defaults = command.Defaults
			r.touchSessionLocked(session)
			return &CommandReceipt{
				CommandID:       command.Meta.ID,
				Accepted:        true,
				SessionID:       session.ID,
				SessionRevision: session.Revision,
			}, nil
		},
	)
}

func (r *Runtime) DeleteSession(ctx context.Context, command DeleteSessionCommand) (*CommandReceipt, error) {
	if command.Meta.ID == "" {
		return nil, errors.New("command ID is required")
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}
	session := r.sessions[command.SessionID]
	if session == nil {
		r.mu.Unlock()
		return nil, ErrSessionNotFound
	}
	if message := expectedSessionRevision(session, command.Meta); message != "" {
		receipt := r.rejectLocked(session, command.Meta, command, message)
		r.mu.Unlock()
		return receipt, nil
	}
	for _, execution := range session.Executions {
		if !execution.State.Terminal() {
			receipt := r.rejectLocked(
				session,
				command.Meta,
				command,
				"active Executions must be terminal before deletion",
			)
			r.mu.Unlock()
			return receipt, nil
		}
	}
	delete(r.sessions, session.ID)
	r.retired[session.ID] = struct{}{}
	r.mu.Unlock()
	if sink, ok := r.deps.Checkpoints.(SessionDeletionSink); ok {
		if err := sink.HandleSessionDeleted(context.WithoutCancel(ctx), command.SessionID); err != nil {
			return nil, err
		}
	}
	return &CommandReceipt{CommandID: command.Meta.ID, Accepted: true, SessionID: command.SessionID}, nil
}

func (r *Runtime) SubmitMessage(ctx context.Context, command SubmitMessageCommand) (*CommandReceipt, error) {
	if command.Meta.ID == "" {
		return nil, errors.New("command ID is required")
	}
	if !command.Input.Metadata.Valid() {
		return &CommandReceipt{
			CommandID: command.Meta.ID,
			SessionID: command.SessionID,
			Message:   "message metadata is invalid",
		}, nil
	}
	for _, attachment := range command.Input.Attachments {
		if err := attachment.Validate(); err != nil {
			//nolint:nilerr // Validation failure is encoded in receipt, not returned error.
			return &CommandReceipt{
				CommandID: command.Meta.ID,
				SessionID: command.SessionID,
				Message:   err.Error(),
			}, nil
		}
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}
	session := r.sessions[command.SessionID]
	if session == nil {
		r.mu.Unlock()
		return nil, ErrSessionNotFound
	}
	if receipt, found, err := r.commandReceiptLocked(session, command.Meta, command); err != nil || found {
		r.mu.Unlock()
		return receipt, err
	}
	if message := expectedSessionRevision(session, command.Meta); message != "" {
		receipt := r.rejectLocked(session, command.Meta, command, message)
		r.mu.Unlock()
		return receipt, nil
	}
	previous := cloneSession(*session)
	now := r.now()
	activity := Activity{
		ID:         ActivityID(r.newID()),
		Kind:       ActivityKindMessage,
		Visibility: ActivityVisibilityBoth,
		CreatedAt:  now,
		Message: &MessageActivity{
			Role:        MessageRoleUser,
			Status:      "accepted",
			Text:        command.Input.Text,
			Attachments: append([]Attachment(nil), command.Input.Attachments...),
			Metadata:    command.Input.Metadata.Clone(),
		},
	}
	session.Timeline = append(session.Timeline, activity)
	execution, err := r.conversationExecutionLocked(session)
	if err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	session.Timeline[len(session.Timeline)-1].ExecutionID = execution.ID
	if err := r.advanceFrameLocked(
		session,
		execution,
		execution.ProgramFrames[0].ID,
		ProgramEvent{Kind: ProgramEventActivity, ActivityID: activity.ID},
	); err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	r.refreshExecutionStateLocked(execution)
	r.touchSessionLocked(session)
	receipt := CommandReceipt{
		CommandID:         command.Meta.ID,
		Accepted:          true,
		SessionID:         session.ID,
		ExecutionID:       execution.ID,
		SessionRevision:   session.Revision,
		ExecutionRevision: execution.Revision,
		ActivityIDs:       []ActivityID{activity.ID},
	}
	r.rememberCommandLocked(session, command.Meta, command, receipt)
	if err := r.persistLocked(ctx, "messageSubmitted", session); err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	ready := r.readyWorkLocked(*execution)
	r.publishLocked(ctx, session, execution.ID, RuntimeEventExecutionChanged)
	r.mu.Unlock()
	r.launchWork(ctx, session.ID, execution.ID, ready)
	return &receipt, nil
}

func (r *Runtime) StartWorkflow(ctx context.Context, command StartWorkflowCommand) (*CommandReceipt, error) {
	if r.deps.Workflow == nil {
		return nil, errors.New("workflow admission is unavailable")
	}
	if command.Meta.ID == "" || !command.Input.Valid() {
		return nil, errors.New("invalid Workflow start command")
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}
	session := r.sessions[command.SessionID]
	if session == nil {
		r.mu.Unlock()
		return nil, ErrSessionNotFound
	}
	if receipt, found, err := r.commandReceiptLocked(session, command.Meta, command); err != nil || found {
		r.mu.Unlock()
		return receipt, err
	}
	if message := expectedSessionRevision(session, command.Meta); message != "" {
		receipt := r.rejectLocked(session, command.Meta, command, message)
		r.mu.Unlock()
		return receipt, nil
	}
	revision := session.Revision
	executionID := command.ExecutionID
	if executionID == "" {
		executionID = ExecutionID(r.newID())
	}
	r.mu.Unlock()
	admission, err := r.deps.Workflow.AdmitWorkflow(
		ctx,
		WorkflowAdmissionRequest{
			SessionID:   command.SessionID,
			ExecutionID: executionID,
			Workflow:    command.Workflow,
			Input:       command.Input.Clone(),
		},
	)
	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil {
		session = r.sessions[command.SessionID]
		if session == nil {
			return nil, ErrSessionNotFound
		}
		return r.rejectLocked(session, command.Meta, command, err.Error()), nil
	}
	if r.closed {
		return nil, ErrRuntimeClosed
	}
	session = r.sessions[command.SessionID]
	if session == nil {
		return nil, ErrSessionNotFound
	}
	if session.Revision != revision {
		return r.rejectLocked(
			session,
			command.Meta,
			command,
			"Session changed while Workflow admission was prepared; retry",
		), nil
	}
	if r.interpreterLocked(admission.Program) == nil {
		return r.rejectLocked(session, command.Meta, command, "Workflow interpreter is unavailable"), nil
	}
	for _, existing := range session.Executions {
		if existing.ID == executionID {
			return r.rejectLocked(session, command.Meta, command, "Execution ID already exists"), nil
		}
	}
	previous := cloneSession(*session)
	now := r.now()
	interpreter := r.interpreterLocked(admission.Program)
	state, err := interpreter.InitialState(admission.Program)
	if err != nil {
		return nil, err
	}
	execution := Execution{
		ID:        executionID,
		SessionID: session.ID,
		Revision:  1,
		State:     ExecutionStateRunning,
		CreatedAt: now,
		UpdatedAt: now,
		ProgramFrames: []ProgramFrame{
			{
				ID:        ProgramFrameID(r.newID()),
				Program:   admission.Program,
				State:     state,
				Status:    ProgramFrameStateRunning,
				Path:      "workflow",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		WorkItems: []WorkItem{},
		Attempts:  []Attempt{},
		Waits:     []Wait{},
	}
	session.Executions = append(session.Executions, execution)
	active := &session.Executions[len(session.Executions)-1]
	session.Timeline = append(
		session.Timeline,
		Activity{
			ID:          ActivityID(r.newID()),
			Kind:        ActivityKindWorkflow,
			Visibility:  ActivityVisibilityBoth,
			CreatedAt:   now,
			ExecutionID: active.ID,
			FrameID:     active.ProgramFrames[0].ID,
			Workflow:    &WorkflowActivity{Phase: "started", Summary: "Workflow execution was admitted."},
		},
	)
	if err := r.advanceFrameLocked(
		session,
		active,
		active.ProgramFrames[0].ID,
		ProgramEvent{Kind: ProgramEventStarted},
	); err != nil {
		*session = previous
		return nil, err
	}
	r.touchSessionLocked(session)
	receipt := CommandReceipt{
		CommandID:         command.Meta.ID,
		Accepted:          true,
		SessionID:         session.ID,
		ExecutionID:       active.ID,
		SessionRevision:   session.Revision,
		ExecutionRevision: active.Revision,
	}
	r.rememberCommandLocked(session, command.Meta, command, receipt)
	if err := r.persistLocked(ctx, "workflowStarted", session); err != nil {
		*session = previous
		return nil, err
	}
	ready := r.readyWorkLocked(*active)
	r.publishLocked(ctx, session, active.ID, RuntimeEventExecutionChanged)
	go r.launchWork(context.WithoutCancel(ctx), session.ID, active.ID, ready)
	return &receipt, nil
}

func (r *Runtime) SignalExecution(ctx context.Context, command SignalExecutionCommand) (*CommandReceipt, error) {
	if command.Name == "" || !command.Data.Valid() || !command.Correlation.Valid() {
		return nil, errors.New("invalid Workflow signal")
	}
	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		command,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			for _, frame := range execution.ProgramFrames {
				if frame.Status.Terminal() {
					continue
				}
				activity := Activity{
					ID:          ActivityID(r.newID()),
					Kind:        ActivityKindWorkflow,
					Visibility:  ActivityVisibilityAudit,
					CreatedAt:   r.now(),
					ExecutionID: execution.ID,
					FrameID:     frame.ID,
					Workflow: &WorkflowActivity{
						Phase:   "signal",
						Summary: command.Name,
						Output: MustCanonicalValue(
							map[string]CanonicalValue{"correlation": command.Correlation, "data": command.Data},
						),
					},
				}
				session.Timeline = append(session.Timeline, activity)
				if err := r.advanceFrameLocked(
					session,
					execution,
					frame.ID,
					ProgramEvent{Kind: ProgramEventActivity, ActivityID: activity.ID},
				); err != nil {
					return nil, err
				}
			}
			r.touchExecutionLocked(session, execution)
			return &CommandReceipt{
				CommandID:         command.Meta.ID,
				Accepted:          true,
				SessionID:         session.ID,
				ExecutionID:       execution.ID,
				SessionRevision:   session.Revision,
				ExecutionRevision: execution.Revision,
			}, nil
		},
	)
}

func (r *Runtime) ExecuteWorkItems(ctx context.Context, command ExecuteWorkItemsCommand) (*CommandReceipt, error) {
	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		command,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			for _, id := range command.WorkItemIDs {
				item := mutableWorkItem(execution, id)
				if item == nil || item.State != WorkItemStateReady {
					return &CommandReceipt{
						CommandID:   command.Meta.ID,
						SessionID:   session.ID,
						ExecutionID: execution.ID,
						Message:     "requested WorkItem is not ready",
					}, nil
				}
				item.Requested = true
				item.UpdatedAt = r.now()
			}
			r.touchExecutionLocked(session, execution)
			return &CommandReceipt{
				CommandID:         command.Meta.ID,
				Accepted:          true,
				SessionID:         session.ID,
				ExecutionID:       execution.ID,
				SessionRevision:   session.Revision,
				ExecutionRevision: execution.Revision,
				WorkItemIDs:       append([]WorkItemID(nil), command.WorkItemIDs...),
			}, nil
		},
	)
}

func (r *Runtime) ContinueExecution(ctx context.Context, command ContinueExecutionCommand) (*CommandReceipt, error) {
	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		command,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			execution.Controls.ContinueRequested = true
			for _, frame := range execution.ProgramFrames {
				if frame.Status.Terminal() {
					continue
				}
				if err := r.advanceFrameLocked(
					session,
					execution,
					frame.ID,
					ProgramEvent{Kind: ProgramEventContinue},
				); err != nil {
					return nil, err
				}
			}
			execution.Controls.ContinueRequested = false
			r.touchExecutionLocked(session, execution)
			return &CommandReceipt{
				CommandID:         command.Meta.ID,
				Accepted:          true,
				SessionID:         session.ID,
				ExecutionID:       execution.ID,
				SessionRevision:   session.Revision,
				ExecutionRevision: execution.Revision,
			}, nil
		},
	)
}

func (r *Runtime) PauseExecution(ctx context.Context, command PauseExecutionCommand) (*CommandReceipt, error) {
	return r.setPaused(ctx, command.Meta, command.SessionID, command.ExecutionID, command, true)
}

func (r *Runtime) ResumeExecution(ctx context.Context, command ResumeExecutionCommand) (*CommandReceipt, error) {
	return r.setPaused(ctx, command.Meta, command.SessionID, command.ExecutionID, command, false)
}

func (r *Runtime) CancelExecution(ctx context.Context, command CancelExecutionCommand) (*CommandReceipt, error) {
	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		command,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			execution.Controls.CancellationRequested = true
			execution.State = ExecutionStateCancelling
			now := r.now()
			for index := range execution.WorkItems {
				item := &execution.WorkItems[index]
				switch item.State {
				case WorkItemStateReady, WorkItemStateBinding, WorkItemStateWaiting:
					item.State = WorkItemStateCancelled
					item.UpdatedAt = now
				case WorkItemStateDispatched:
					if cancel := r.cancels[item.ID]; cancel != nil {
						cancel()
						delete(r.cancels, item.ID)
					}
				case WorkItemStateCompleted,
					WorkItemStateFailed,
					WorkItemStateUncertain,
					WorkItemStateCancelled,
					WorkItemStateBlocked:
					// Terminal or blocked; no action.
				}
			}
			for index := range execution.Waits {
				wait := &execution.Waits[index]
				if wait.Status == WaitStatusPending || wait.Status == WaitStatusResolving {
					wait.Status = WaitStatusCancelled
					wait.UpdatedAt = now
				}
			}
			for _, frame := range execution.ProgramFrames {
				if !frame.Status.Terminal() {
					_ = r.advanceFrameLocked(session, execution, frame.ID, ProgramEvent{Kind: ProgramEventCancelled})
				}
			}
			if !hasDispatched(*execution) {
				r.completeExecutionLocked(
					ctx,
					session,
					execution,
					ExecutionOutcome{
						Kind:        ExecutionOutcomeCancelled,
						Code:        "cancelled",
						Message:     firstText(command.Reason, "Execution was cancelled"),
						CompletedAt: now,
					},
				)
			} else {
				r.touchExecutionLocked(session, execution)
			}
			return &CommandReceipt{
				CommandID:         command.Meta.ID,
				Accepted:          true,
				SessionID:         session.ID,
				ExecutionID:       execution.ID,
				SessionRevision:   session.Revision,
				ExecutionRevision: execution.Revision,
			}, nil
		},
	)
}

func (r *Runtime) ResolveWait(ctx context.Context, command ResolveWaitCommand) (*CommandReceipt, error) {
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
	if receipt, found, err := r.commandReceiptLocked(session, command.Meta, command); err != nil || found {
		r.mu.Unlock()
		return receipt, err
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
	if !containsResolution(wait.Request.AllowedResolutions, command.Resolution) {
		receipt := r.rejectLocked(session, command.Meta, command, "wait resolution is not allowed")
		r.mu.Unlock()
		return receipt, nil
	}
	previous := cloneSession(*session)
	wait.Status = WaitStatusResolving
	wait.UpdatedAt = r.now()
	r.touchExecutionLocked(session, execution)
	if err := r.persistLocked(ctx, "waitResolutionRequested", session); err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	item := mutableWorkItem(execution, wait.WorkItemID)
	private := r.privateWaits[wait.ID]
	request := WaitResolutionRequest{
		Session:    cloneSession(*session),
		Execution:  cloneExecution(*execution),
		WorkItem:   *item,
		Wait:       *wait,
		Resolution: command.Resolution,
		Response:   command.Response.Clone(),
		Private:    private,
	}
	r.mu.Unlock()
	result, resolveErr := r.deps.Binder.ResolveWait(ctx, request)
	r.mu.Lock()
	defer r.mu.Unlock()
	session, execution, err = r.executionLocked(command.SessionID, command.ExecutionID)
	if err != nil {
		return nil, err
	}
	wait = mutableWait(execution, command.WaitID)
	item = mutableWorkItem(execution, wait.WorkItemID)
	if wait == nil || item == nil || wait.Status != WaitStatusResolving {
		return nil, errors.New("wait changed while resolution was in progress")
	}
	now := r.now()
	if resolveErr != nil {
		wait.Status = WaitStatusPending
		wait.Error = &SanitizedError{Code: "wait_resolution_failed", Message: resolveErr.Error()}
		wait.UpdatedAt = now
		r.touchExecutionLocked(session, execution)
		receipt := CommandReceipt{
			CommandID:         command.Meta.ID,
			Accepted:          true,
			SessionID:         session.ID,
			ExecutionID:       execution.ID,
			SessionRevision:   session.Revision,
			ExecutionRevision: execution.Revision,
			WaitIDs:           []WaitID{wait.ID},
			Message:           "wait remains pending",
		}
		r.rememberCommandLocked(session, command.Meta, command, receipt)
		if err := r.persistLocked(ctx, "waitResolutionFailed", session); err != nil {
			return nil, err
		}
		return &receipt, nil
	}
	wait.Status = WaitStatusResolved
	wait.Resolution = command.Resolution
	wait.Response = command.Response.Clone()
	wait.UpdatedAt = now
	delete(r.privateWaits, wait.ID)
	item.WaitID = ""
	switch {
	case result.Output != nil:
		r.finishWorkLocked(
			session,
			execution,
			item,
			activeAttempt(execution, item.ID),
			WorkExecutionResult{Output: result.Output, Error: result.Error, Confirmed: result.Confirmed},
		)
	case result.Resume && result.Bound != nil:
		r.privateBindings[item.ID] = *result.Bound
		if err := r.startDispatchedLocked(ctx, session, execution, item, *result.Bound); err != nil {
			return nil, err
		}
	default:
		item.State = WorkItemStateFailed
		item.Error = result.Error
		if item.Error == nil {
			item.Error = &SanitizedError{
				Code:    "wait_resolution_invalid",
				Message: "wait resolver returned no resume or result",
			}
		}
		item.CompletedAt = &now
		item.UpdatedAt = now
	}
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
	r.rememberCommandLocked(session, command.Meta, command, receipt)
	if err := r.persistLocked(ctx, "waitResolved", session); err != nil {
		return nil, err
	}
	ready := r.readyWorkLocked(*execution)
	r.publishLocked(ctx, session, execution.ID, RuntimeEventWaitChanged)
	go r.launchWork(context.WithoutCancel(ctx), session.ID, execution.ID, ready)
	return &receipt, nil
}

func (r *Runtime) GetSession(_ context.Context, id SessionID) (Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return Session{}, ErrRuntimeClosed
	}
	session := r.sessions[id]
	if session == nil {
		return Session{}, ErrSessionNotFound
	}
	return cloneSession(*session), nil
}

func (r *Runtime) Conversation(ctx context.Context, id SessionID) (ConversationProjection, error) {
	session, err := r.GetSession(ctx, id)
	if err != nil {
		return ConversationProjection{}, err
	}
	values := []Activity{}
	for _, activity := range session.Timeline {
		if activity.Kind == ActivityKindMessage && activity.Visibility != ActivityVisibilityAudit {
			values = append(values, activity)
		}
	}
	return ConversationProjection{
		SchemaVersion:   SchemaVersion,
		SessionID:       id,
		SessionRevision: session.Revision,
		Title:           session.Title,
		Messages:        values,
	}, nil
}

func (r *Runtime) Activities(ctx context.Context, id SessionID) (ActivityProjection, error) {
	session, err := r.GetSession(ctx, id)
	if err != nil {
		return ActivityProjection{}, err
	}
	return ActivityProjection{
		SchemaVersion:   SchemaVersion,
		SessionID:       id,
		SessionRevision: session.Revision,
		Activities:      append([]Activity(nil), session.Timeline...),
	}, nil
}

func (r *Runtime) Workflow(ctx context.Context, id SessionID) (WorkflowProjection, error) {
	session, err := r.GetSession(ctx, id)
	if err != nil {
		return WorkflowProjection{}, err
	}
	activities := []Activity{}
	executions := []Execution{}
	for _, activity := range session.Timeline {
		if activity.Kind == ActivityKindWorkflow || activity.Kind == ActivityKindAction ||
			activity.Kind == ActivityKindWait {
			activities = append(activities, activity)
		}
	}
	for _, execution := range session.Executions {
		for _, frame := range execution.ProgramFrames {
			if frame.Program.InterpreterID == "artifact.workflow.v1" {
				executions = append(executions, execution)
				break
			}
		}
	}
	return WorkflowProjection{
		SchemaVersion:   SchemaVersion,
		SessionID:       id,
		SessionRevision: session.Revision,
		Activities:      activities,
		Executions:      executions,
	}, nil
}

func (r *Runtime) setPaused(
	ctx context.Context,
	meta CommandMeta,
	sessionID SessionID,
	executionID ExecutionID,
	payload any,
	paused bool,
) (*CommandReceipt, error) {
	return r.changeExecution(
		ctx,
		meta,
		sessionID,
		executionID,
		payload,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			if execution.State.Terminal() {
				return &CommandReceipt{
					CommandID:   meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     "Execution is terminal",
				}, nil
			}
			execution.Controls.Paused = paused
			r.refreshExecutionStateLocked(execution)
			r.touchExecutionLocked(session, execution)
			return &CommandReceipt{
				CommandID:         meta.ID,
				Accepted:          true,
				SessionID:         session.ID,
				ExecutionID:       execution.ID,
				SessionRevision:   session.Revision,
				ExecutionRevision: execution.Revision,
			}, nil
		},
	)
}

func (r *Runtime) changeSession(
	ctx context.Context,
	meta CommandMeta,
	id SessionID,
	payload any,
	mutate func(*Session) (*CommandReceipt, error),
) (*CommandReceipt, error) {
	if meta.ID == "" {
		return nil, errors.New("command ID is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, ErrRuntimeClosed
	}
	session := r.sessions[id]
	if session == nil {
		return nil, ErrSessionNotFound
	}
	if receipt, found, err := r.commandReceiptLocked(session, meta, payload); err != nil || found {
		return receipt, err
	}
	if message := expectedSessionRevision(session, meta); message != "" {
		return r.rejectLocked(session, meta, payload, message), nil
	}
	previous := cloneSession(*session)
	receipt, err := mutate(session)
	if err != nil || !receipt.Accepted {
		return receipt, err
	}
	r.rememberCommandLocked(session, meta, payload, *receipt)
	if err := r.persistLocked(ctx, "sessionChanged", session); err != nil {
		*session = previous
		return nil, err
	}
	r.publishLocked(ctx, session, "", RuntimeEventSessionChanged)
	return receipt, nil
}

func (r *Runtime) changeExecution(
	ctx context.Context,
	meta CommandMeta,
	sessionID SessionID,
	executionID ExecutionID,
	payload any,
	mutate func(*Session, *Execution) (*CommandReceipt, error),
) (*CommandReceipt, error) {
	if meta.ID == "" {
		return nil, errors.New("command ID is required")
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}
	session, execution, err := r.executionLocked(sessionID, executionID)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if receipt, found, err := r.commandReceiptLocked(session, meta, payload); err != nil || found {
		r.mu.Unlock()
		return receipt, err
	}
	if message := expectedExecutionRevision(session, execution, meta); message != "" {
		receipt := r.rejectLocked(session, meta, payload, message)
		r.mu.Unlock()
		return receipt, nil
	}
	previous := cloneSession(*session)
	receipt, err := mutate(session, execution)
	if err != nil || !receipt.Accepted {
		r.mu.Unlock()
		return receipt, err
	}
	r.refreshExecutionStateLocked(execution)
	r.rememberCommandLocked(session, meta, payload, *receipt)
	if err := r.persistLocked(ctx, "executionChanged", session); err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	ready := r.readyWorkLocked(*execution)
	r.publishLocked(ctx, session, execution.ID, RuntimeEventExecutionChanged)
	r.mu.Unlock()
	r.launchWork(ctx, sessionID, executionID, ready)
	return receipt, nil
}

func (r *Runtime) conversationExecutionLocked(session *Session) (*Execution, error) {
	for index := range session.Executions {
		execution := &session.Executions[index]
		if !execution.State.Terminal() && len(execution.ProgramFrames) != 0 &&
			execution.ProgramFrames[0].Program.InterpreterID == BuiltinConversationInterpreterID {
			return execution, nil
		}
	}
	program, err := NewConversationProgram(session.Defaults)
	if err != nil {
		return nil, err
	}
	interpreter := r.interpreterLocked(program)
	if interpreter == nil {
		return nil, errors.New("conversation interpreter unavailable")
	}
	state, err := interpreter.InitialState(program)
	if err != nil {
		return nil, err
	}
	now := r.now()
	execution := Execution{
		ID:        ExecutionID(r.newID()),
		SessionID: session.ID,
		Revision:  1,
		State:     ExecutionStateWaiting,
		CreatedAt: now,
		UpdatedAt: now,
		ProgramFrames: []ProgramFrame{
			{
				ID:        ProgramFrameID(r.newID()),
				Program:   program,
				State:     state,
				Status:    ProgramFrameStateWaiting,
				Path:      "conversation",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		WorkItems: []WorkItem{},
		Attempts:  []Attempt{},
		Waits:     []Wait{},
	}
	session.Executions = append(session.Executions, execution)
	return &session.Executions[len(session.Executions)-1], nil
}

func (r *Runtime) advanceFrameLocked(
	session *Session,
	execution *Execution,
	frameID ProgramFrameID,
	event ProgramEvent,
) error {
	frame := mutableFrame(execution, frameID)
	if frame == nil || frame.Status.Terminal() {
		return nil
	}
	interpreter := r.interpreterLocked(frame.Program)
	if interpreter == nil {
		return fmt.Errorf("interpreter %q is unavailable", frame.Program.InterpreterID)
	}
	transition, err := interpreter.Advance(cloneFrame(*frame), cloneExecution(*execution), event)
	if err != nil {
		return err
	}
	if err := interpreter.ValidateState(transition.State); err != nil {
		return err
	}
	frame.State = transition.State
	frame.UpdatedAt = r.now()
	if transition.Status != nil {
		frame.Status = *transition.Status
	}
	frame.Output = transition.Output.Clone()
	if transition.Error != nil {
		frame.Error = transition.Error
	}
	for _, activity := range transition.Activities {
		if activity.ID == "" {
			activity.ID = ActivityID(r.newID())
		}
		if activity.CreatedAt.IsZero() {
			activity.CreatedAt = r.now()
		}
		activity.ExecutionID = execution.ID
		activity.FrameID = frame.ID
		session.Timeline = append(session.Timeline, activity)
	}
	childIDs := []ProgramFrameID{}
	for _, intent := range transition.Frames {
		childInterpreter := r.interpreterLocked(intent.Program)
		if childInterpreter == nil {
			return fmt.Errorf("child interpreter %q unavailable", intent.Program.InterpreterID)
		}
		if err := childInterpreter.ValidateSnapshot(intent.Program); err != nil {
			return err
		}
		state, err := childInterpreter.InitialState(intent.Program)
		if err != nil {
			return err
		}
		now := r.now()
		child := ProgramFrame{
			ID:            ProgramFrameID(r.newID()),
			ParentFrameID: frame.ID,
			Program:       intent.Program,
			State:         state,
			Status:        ProgramFrameStateRunning,
			Path:          intent.Path,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		execution.ProgramFrames = append(execution.ProgramFrames, child)
		frame.ChildFrameIDs = append(frame.ChildFrameIDs, child.ID)
		childIDs = append(childIDs, child.ID)
		if frame.Program.InterpreterID == BuiltinConversationInterpreterID {
			var cs conversationProgramState
			if json.Unmarshal(frame.State.Data, &cs) == nil {
				cs.ActiveFrameID = child.ID
				frame.State = encodeConversationState(cs)
			}
		}
	}
	for _, intent := range transition.WorkItems {
		now := r.now()
		item := WorkItem{
			ID:          WorkItemID(r.newID()),
			FrameID:     frame.ID,
			Type:        intent.Type,
			State:       WorkItemStateReady,
			ProgramPath: intent.ProgramPath,
			Deadline:    cloneTime(intent.Deadline),
			Effect:      intent.Effect,
			Automatic:   intent.Automatic,
			Input:       intent.Input,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := item.Validate(); err != nil {
			return err
		}
		execution.WorkItems = append(execution.WorkItems, item)
		frame.WorkItemIDs = append(frame.WorkItemIDs, item.ID)
		if frame.Program.InterpreterID == BuiltinAgentLoopInterpreterID {
			if item.Type == WorkItemTypeTool {
				var as agentLoopProgramState
				if json.Unmarshal(frame.State.Data, &as) == nil {
					as.PendingToolIDs = append(as.PendingToolIDs, item.ID)
					frame.State = encodeAgentLoopState(as)
				}
			}
			if item.Type == WorkItemTypeLLM {
				var as agentLoopProgramState
				if json.Unmarshal(frame.State.Data, &as) == nil {
					as.LastLLMWorkItemID = item.ID
					frame.State = encodeAgentLoopState(as)
				}
			}
		}
		if item.Type == WorkItemTypeLLM && frame.Program.InterpreterID == BuiltinSingleModelInterpreterID {
			var ss singleModelProgramState
			if json.Unmarshal(frame.State.Data, &ss) == nil {
				ss.WorkItemID = item.ID
				frame.State = encodeSingleModelState(ss)
			}
		}
	}
	for _, childID := range childIDs {
		if err := r.advanceFrameLocked(
			session,
			execution,
			childID,
			ProgramEvent{Kind: ProgramEventStarted},
		); err != nil {
			return err
		}
	}
	if frame.Status.Terminal() && frame.ParentFrameID != "" {
		return r.advanceFrameLocked(
			session,
			execution,
			frame.ParentFrameID,
			ProgramEvent{Kind: ProgramEventChildSettled, ChildFrameID: frame.ID},
		)
	}
	return nil
}

func (r *Runtime) dispatchOne(ctx context.Context, sessionID SessionID, executionID ExecutionID, itemID WorkItemID) {
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
	item := mutableWorkItem(execution, itemID)
	if item == nil || item.State != WorkItemStateReady || execution.Controls.Paused ||
		execution.Controls.CancellationRequested ||
		execution.State.Terminal() ||
		(!item.Automatic && !item.Requested) ||
		unresolvedDependencies(*execution, *item) {
		r.mu.Unlock()
		return
	}
	prebound, hasPrebound := r.privateBindings[item.ID]
	previous := cloneSession(*session)
	item.State = WorkItemStateBinding
	item.UpdatedAt = r.now()
	r.touchExecutionLocked(session, execution)
	if err := r.persistLocked(ctx, "workBinding", session); err != nil {
		*session = previous
		r.mu.Unlock()
		return
	}
	request := WorkDispatchRequest{
		Session:   cloneSession(*session),
		Execution: cloneExecution(*execution),
		WorkItem:  *item,
	}
	r.mu.Unlock()
	var bound BindResult
	if hasPrebound {
		bound = BindResult{SemanticBinding: prebound.SemanticBinding, Private: prebound.Private}
	} else {
		bound, err = r.deps.Binder.BindWork(ctx, request)
	}
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	session, execution, err = r.executionLocked(sessionID, executionID)
	item = mutableWorkItem(execution, itemID)
	if item == nil || item.State != WorkItemStateBinding {
		return
	}
	if err != nil {
		now := r.now()
		item.State = WorkItemStateBlocked
		item.Error = &SanitizedError{Code: "runtime_binding_failed", Message: err.Error()}
		item.CompletedAt = &now
		item.UpdatedAt = now
		r.touchExecutionLocked(session, execution)
		_ = r.persistLocked(ctx, "workBlocked", session)
		return
	}
	if bound.ImmediateOutput != nil {
		r.finishWorkLocked(
			session,
			execution,
			item,
			activeAttempt(execution, item.ID),
			WorkExecutionResult{Output: bound.ImmediateOutput, Confirmed: true},
		)
		_ = r.persistLocked(ctx, "workFinished", session)
		go r.launchWork(context.WithoutCancel(ctx), sessionID, executionID, r.readyWorkLocked(*execution))
		return
	}
	if bound.Wait != nil {
		now := r.now()
		wait := Wait{
			ID:         WaitID(r.newID()),
			WorkItemID: item.ID,
			Status:     WaitStatusPending,
			Request:    *bound.Wait,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		item.State = WorkItemStateWaiting
		item.WaitID = wait.ID
		item.Binding = &bound.SemanticBinding
		item.UpdatedAt = now
		execution.Waits = append(execution.Waits, wait)
		r.privateWaits[wait.ID] = bound.Private
		r.touchExecutionLocked(session, execution)
		r.appendWaitActivityLocked(session, execution, wait)
		_ = r.persistLocked(ctx, "waitRequested", session)
		if wait.Request.Deadline != nil {
			go r.expireAt(context.WithoutCancel(ctx), sessionID, executionID, wait.ID, *wait.Request.Deadline)
		}
		return
	}
	if err := r.startDispatchedLocked(
		ctx,
		session,
		execution,
		item,
		BoundWork{SemanticBinding: bound.SemanticBinding, Private: bound.Private},
	); err != nil {
		*session = previous
		return
	}
	go r.executeOne(
		context.WithoutCancel(ctx),
		sessionID,
		executionID,
		item.ID,
		request,
		BoundWork{SemanticBinding: bound.SemanticBinding, Private: bound.Private},
	)
}

func (r *Runtime) startDispatchedLocked(
	ctx context.Context,
	session *Session,
	execution *Execution,
	item *WorkItem,
	bound BoundWork,
) error {
	now := r.now()
	attempt := Attempt{
		ID:             AttemptID(r.newID()),
		WorkItemID:     item.ID,
		State:          AttemptStateDispatched,
		Number:         nextAttempt(*execution, item.ID),
		IdempotencyKey: item.Effect.IdempotencyKey,
		StartedAt:      now,
	}
	item.State = WorkItemStateDispatched
	item.Binding = &bound.SemanticBinding
	item.AttemptIDs = append(item.AttemptIDs, attempt.ID)
	item.DispatchedAt = &now
	item.UpdatedAt = now
	execution.Attempts = append(execution.Attempts, attempt)
	r.privateBindings[item.ID] = bound
	r.touchExecutionLocked(session, execution)
	return r.persistLocked(ctx, "workDispatched", session)
}

func (r *Runtime) executeOne(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	itemID WorkItemID,
	request WorkDispatchRequest,
	bound BoundWork,
) {
	executionCtx, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	r.cancels[itemID] = cancel
	r.mu.Unlock()
	result, err := r.deps.Binder.ExecuteWork(executionCtx, request, bound)
	cancel()
	if err != nil && result.Error == nil {
		result.Error = &SanitizedError{Code: "external_execution_failed", Message: err.Error()}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	session, execution, lookupErr := r.executionLocked(sessionID, executionID)
	if lookupErr != nil {
		return
	}
	item := mutableWorkItem(execution, itemID)
	if item == nil || item.State != WorkItemStateDispatched {
		return
	}
	delete(r.cancels, itemID)
	delete(r.privateBindings, itemID)
	r.finishWorkLocked(session, execution, item, activeAttempt(execution, itemID), result)
	_ = r.persistLocked(ctx, "workFinished", session)
	go r.launchWork(context.WithoutCancel(ctx), sessionID, executionID, r.readyWorkLocked(*execution))
}

func (r *Runtime) finishWorkLocked(
	session *Session,
	execution *Execution,
	item *WorkItem,
	attempt *Attempt,
	result WorkExecutionResult,
) {
	now := r.now()
	item.Output = result.Output
	item.Error = result.Error
	item.CompletedAt = &now
	item.UpdatedAt = now
	if attempt != nil {
		attempt.FinishedAt = &now
		attempt.Error = result.Error
		if result.Output != nil {
			attempt.Receipt = result.Output.Receipt
		}
	}
	switch {
	case result.Confirmed && result.Error == nil:
		item.State = WorkItemStateCompleted
		if attempt != nil {
			attempt.State = AttemptStateConfirmed
		}
	case result.Confirmed:
		item.State = WorkItemStateFailed
		if attempt != nil {
			attempt.State = AttemptStateFailed
		}
	default:
		item.State = WorkItemStateUncertain
		if attempt != nil {
			attempt.State = AttemptStateUncertain
		}
		if item.Error == nil {
			item.Error = &SanitizedError{
				Code:    "external_outcome_uncertain",
				Message: "external outcome was not durably confirmed",
			}
		}
	}
	if item.Type == WorkItemTypeTool && item.Output != nil && item.Output.Tool != nil {
		session.Timeline = append(
			session.Timeline,
			Activity{
				ID:          ActivityID(r.newID()),
				Kind:        ActivityKindAction,
				Visibility:  ActivityVisibilityBoth,
				CreatedAt:   now,
				ExecutionID: execution.ID,
				FrameID:     item.FrameID,
				WorkItemID:  item.ID,
				Action: &ActionActivity{
					Title:      firstText(item.Input.Tool.Call.DisplayName, item.Input.Tool.Call.Name),
					Summary:    item.Output.Tool.Text,
					ActionKind: "tool",
					Outcome:    string(item.Output.Tool.Kind),
					Output:     item.Output.Tool.Structured.Clone(),
					Error:      item.Error,
					Receipt:    item.Output.Receipt,
				},
			},
		)
	}
	if item.State == WorkItemStateUncertain {
		r.addRecoveryWaitLocked(session, execution, item)
	} else {
		_ = r.advanceFrameLocked(
			session,
			execution,
			item.FrameID,
			ProgramEvent{Kind: ProgramEventWorkSettled, WorkItemID: item.ID},
		)
	}
	r.refreshExecutionStateLocked(execution)
	r.touchExecutionLocked(session, execution)
}

func (r *Runtime) appendWaitActivityLocked(session *Session, execution *Execution, wait Wait) {
	session.Timeline = append(
		session.Timeline,
		Activity{
			ID:          ActivityID(r.newID()),
			Kind:        ActivityKindWait,
			Visibility:  ActivityVisibilityBoth,
			CreatedAt:   r.now(),
			ExecutionID: execution.ID,
			WorkItemID:  wait.WorkItemID,
			Wait:        &WaitActivity{WaitID: wait.ID, Status: wait.Status, Summary: string(wait.Request.Summary)},
		},
	)
}

func (r *Runtime) addRecoveryWaitLocked(session *Session, execution *Execution, item *WorkItem) {
	now := r.now()
	wait := Wait{
		ID:         WaitID(r.newID()),
		WorkItemID: item.ID,
		Status:     WaitStatusPending,
		Request: WaitRequest{
			Kind: WaitKindRecovery,
			Summary: MustCanonicalValue(
				map[string]string{"message": "external WorkItem outcome requires recovery review"},
			),
			AllowedResolutions: []WaitResolution{WaitResolutionContinue, WaitResolutionCancel},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	execution.Waits = append(execution.Waits, wait)
	item.WaitID = wait.ID
	r.appendWaitActivityLocked(session, execution, wait)
}

func (r *Runtime) expireAt(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	waitID WaitID,
	deadline time.Time,
) {
	delay := time.Until(deadline)
	if delay > 0 {
		time.Sleep(delay)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	session, execution, err := r.executionLocked(sessionID, executionID)
	if err != nil {
		return
	}
	wait := mutableWait(execution, waitID)
	if wait == nil || wait.Status != WaitStatusPending || wait.Request.Deadline == nil ||
		r.now().Before(*wait.Request.Deadline) {
		return
	}
	now := r.now()
	wait.Status = WaitStatusExpired
	wait.Error = &SanitizedError{Code: "wait_expired", Message: "wait deadline elapsed"}
	wait.UpdatedAt = now
	item := mutableWorkItem(execution, wait.WorkItemID)
	if item != nil && !item.State.Terminal() {
		item.State = WorkItemStateFailed
		item.Error = wait.Error
		item.CompletedAt = &now
		item.UpdatedAt = now
		_ = r.advanceFrameLocked(
			session,
			execution,
			item.FrameID,
			ProgramEvent{Kind: ProgramEventWorkSettled, WorkItemID: item.ID},
		)
	}
	r.touchExecutionLocked(session, execution)
	_ = r.persistLocked(context.WithoutCancel(ctx), "waitExpired", session)
}

func (r *Runtime) readyWorkLocked(execution Execution) []WorkItemID {
	if execution.Controls.Paused || execution.Controls.CancellationRequested || execution.State.Terminal() {
		return nil
	}
	values := []WorkItemID{}
	for _, item := range execution.WorkItems {
		if item.State == WorkItemStateReady && (item.Automatic || item.Requested) &&
			!unresolvedDependencies(execution, item) {
			values = append(values, item.ID)
		}
	}
	return values
}

func (r *Runtime) launchWork(ctx context.Context, sessionID SessionID, executionID ExecutionID, items []WorkItemID) {
	for _, id := range items {
		go r.dispatchOne(context.WithoutCancel(ctx), sessionID, executionID, id)
	}
}

func (r *Runtime) refreshExecutionStateLocked(execution *Execution) {
	if execution.State.Terminal() {
		return
	}
	if execution.Controls.CancellationRequested {
		execution.State = ExecutionStateCancelling
		return
	}
	if execution.Controls.Paused {
		execution.State = ExecutionStatePaused
		return
	}
	for _, wait := range execution.Waits {
		if wait.Status == WaitStatusPending || wait.Status == WaitStatusResolving {
			execution.State = ExecutionStateWaiting
			return
		}
	}
	for _, item := range execution.WorkItems {
		if item.State == WorkItemStateReady || item.State == WorkItemStateBinding ||
			item.State == WorkItemStateDispatched {
			execution.State = ExecutionStateRunning
			return
		}
	}
	execution.State = ExecutionStateWaiting
}

func (r *Runtime) completeExecutionLocked(
	ctx context.Context,
	session *Session,
	execution *Execution,
	outcome ExecutionOutcome,
) {
	execution.Outcome = &outcome
	switch outcome.Kind {
	case ExecutionOutcomeCancelled:
		execution.State = ExecutionStateCancelled
	case ExecutionOutcomeFailed:
		execution.State = ExecutionStateFailed
	default:
		execution.State = ExecutionStateCompleted
	}
	r.touchExecutionLocked(session, execution)
	if closer, ok := r.deps.Binder.(ExecutionBindingCloser); ok {
		id := execution.ID
		go func(ctx context.Context) { _ = closer.CloseExecutionBinding(ctx, id) }(context.WithoutCancel(ctx))
	}
}

func (r *Runtime) executionLocked(sessionID SessionID, executionID ExecutionID) (*Session, *Execution, error) {
	session := r.sessions[sessionID]
	if session == nil {
		return nil, nil, ErrSessionNotFound
	}
	for index := range session.Executions {
		if session.Executions[index].ID == executionID {
			return session, &session.Executions[index], nil
		}
	}
	return nil, nil, ErrExecutionNotFound
}

func (r *Runtime) interpreterLocked(snapshot ProgramSnapshot) ProgramInterpreter {
	return r.interpreters[ProgramInterpreterKey(snapshot.InterpreterID, snapshot.InterpreterVersion)]
}

func (r *Runtime) persistLocked(ctx context.Context, reason string, session *Session) error {
	if err := session.Validate(); err != nil {
		return err
	}
	for _, execution := range session.Executions {
		for _, frame := range execution.ProgramFrames {
			interpreter := r.interpreterLocked(frame.Program)
			if interpreter == nil {
				return fmt.Errorf("interpreter %q unavailable", frame.Program.InterpreterID)
			}
			if err := interpreter.ValidateSnapshot(frame.Program); err != nil {
				return err
			}
			if err := interpreter.ValidateState(frame.State); err != nil {
				return err
			}
		}
	}
	return r.deps.Checkpoints.HandleCheckpoint(
		context.WithoutCancel(ctx),
		CheckpointView{Reason: reason, Session: cloneSession(*session), CreatedAt: r.now()},
	)
}

func (r *Runtime) publishLocked(ctx context.Context, session *Session, executionID ExecutionID, kind RuntimeEventKind) {
	if r.deps.Events == nil {
		return
	}
	session.EventSequence++
	event := RuntimeEvent{
		SessionID:   session.ID,
		ExecutionID: executionID,
		Sequence:    session.EventSequence,
		Kind:        kind,
		At:          r.now(),
	}
	go func() { _ = r.deps.Events.PublishRuntimeEvent(context.WithoutCancel(ctx), event) }()
}

func (r *Runtime) touchSessionLocked(session *Session) {
	session.Revision++
	session.UpdatedAt = r.now()
}

func (r *Runtime) touchExecutionLocked(session *Session, execution *Execution) {
	r.touchSessionLocked(session)
	execution.Revision++
	execution.UpdatedAt = session.UpdatedAt
}
func (r *Runtime) now() time.Time { return r.deps.Now().UTC() }
func (r *Runtime) newID() string  { return r.deps.NewID() }
func (r *Runtime) commandReceiptLocked(session *Session, meta CommandMeta, payload any) (*CommandReceipt, bool, error) {
	if meta.ID == "" {
		return nil, false, errors.New("command ID is required")
	}
	fingerprint, err := fingerprintCommand(payload)
	if err != nil {
		return nil, false, err
	}
	for _, record := range session.Commands {
		if record.ID != meta.ID {
			continue
		}
		if record.Fingerprint != fingerprint {
			return nil, false, ErrCommandConflict
		}
		receipt := record.Receipt
		receipt.Duplicate = true
		return &receipt, true, nil
	}
	return nil, false, nil
}

func (r *Runtime) rememberCommandLocked(session *Session, meta CommandMeta, payload any, receipt CommandReceipt) {
	fingerprint, err := fingerprintCommand(payload)
	if err != nil {
		panic(err)
	}
	session.Commands = append(
		session.Commands,
		CommandRecord{ID: meta.ID, Fingerprint: fingerprint, Receipt: receipt, RecordedAt: r.now()},
	)
	if len(session.Commands) > maxDurableCommandRecords {
		session.Commands = append(
			[]CommandRecord(nil),
			session.Commands[len(session.Commands)-maxDurableCommandRecords:]...,
		)
	}
}

func (r *Runtime) rejectLocked(session *Session, meta CommandMeta, payload any, message string) *CommandReceipt {
	receipt := CommandReceipt{
		CommandID:       meta.ID,
		SessionID:       session.ID,
		SessionRevision: session.Revision,
		Message:         message,
	}
	r.rememberCommandLocked(session, meta, payload, receipt)
	return &receipt
}

func fingerprintCommand(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(append(fmt.Appendf(nil, "%T\x00", value), raw...))
	return hex.EncodeToString(sum[:]), nil
}

func expectedSessionRevision(session *Session, meta CommandMeta) string {
	if meta.ExpectedSessionRevision == nil || *meta.ExpectedSessionRevision == session.Revision {
		return ""
	}
	return fmt.Sprintf(
		"Session revision mismatch: expected %d, current %d",
		*meta.ExpectedSessionRevision,
		session.Revision,
	)
}

func expectedExecutionRevision(session *Session, execution *Execution, meta CommandMeta) string {
	if message := expectedSessionRevision(session, meta); message != "" {
		return message
	}
	if meta.ExpectedExecutionRevision == nil || *meta.ExpectedExecutionRevision == execution.Revision {
		return ""
	}
	return fmt.Sprintf(
		"Execution revision mismatch: expected %d, current %d",
		*meta.ExpectedExecutionRevision,
		execution.Revision,
	)
}

func mutableFrame(execution *Execution, id ProgramFrameID) *ProgramFrame {
	for index := range execution.ProgramFrames {
		if execution.ProgramFrames[index].ID == id {
			return &execution.ProgramFrames[index]
		}
	}
	return nil
}

func mutableWorkItem(execution *Execution, id WorkItemID) *WorkItem {
	for index := range execution.WorkItems {
		if execution.WorkItems[index].ID == id {
			return &execution.WorkItems[index]
		}
	}
	return nil
}

func mutableWait(execution *Execution, id WaitID) *Wait {
	for index := range execution.Waits {
		if execution.Waits[index].ID == id {
			return &execution.Waits[index]
		}
	}
	return nil
}

func activeAttempt(execution *Execution, itemID WorkItemID) *Attempt {
	for index := range execution.Attempts {
		if execution.Attempts[index].WorkItemID == itemID && execution.Attempts[index].State == AttemptStateDispatched {
			return &execution.Attempts[index]
		}
	}
	return nil
}

func nextAttempt(execution Execution, itemID WorkItemID) int {
	count := 0
	for _, attempt := range execution.Attempts {
		if attempt.WorkItemID == itemID {
			count++
		}
	}
	return count + 1
}

func unresolvedDependencies(execution Execution, item WorkItem) bool {
	for _, id := range item.Dependencies {
		dependency := executionWorkItem(execution, id)
		if dependency == nil || !dependency.State.Terminal() {
			return true
		}
	}
	return false
}

func hasDispatched(execution Execution) bool {
	for _, item := range execution.WorkItems {
		if item.State == WorkItemStateDispatched {
			return true
		}
	}
	return false
}

func containsResolution(values []WaitResolution, want WaitResolution) bool {
	return slices.Contains(values, want)
}

func firstText(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

type runtimeRestorer struct{ runtime *Runtime }

func (r runtimeRestorer) RestoreSession(ctx context.Context, value Session) error {
	if r.runtime == nil {
		return ErrRuntimeClosed
	}
	if err := value.Validate(); err != nil {
		return err
	}
	r.runtime.mu.Lock()
	defer r.runtime.mu.Unlock()
	if r.runtime.closed {
		return ErrRuntimeClosed
	}
	if _, found := r.runtime.sessions[value.ID]; found {
		return fmt.Errorf("duplicate restored Session %q", value.ID)
	}
	copyValue := cloneSession(value)
	changed := r.runtime.recoverLocked(&copyValue)
	r.runtime.sessions[copyValue.ID] = &copyValue
	if changed {
		if err := r.runtime.persistLocked(ctx, "recoveryApplied", &copyValue); err != nil {
			return err
		}
		r.runtime.publishLocked(ctx, &copyValue, "", RuntimeEventRecoveryApplied)
	}
	return nil
}

func (r runtimeRestorer) RestoreRetiredSession(_ context.Context, id SessionID) error {
	if r.runtime == nil {
		return ErrRuntimeClosed
	}
	r.runtime.mu.Lock()
	defer r.runtime.mu.Unlock()
	r.runtime.retired[id] = struct{}{}
	return nil
}

func (r *Runtime) recoverLocked(session *Session) bool {
	changed := false
	now := r.now()
	for executionIndex := range session.Executions {
		execution := &session.Executions[executionIndex]
		needsReview := false
		for itemIndex := range execution.WorkItems {
			item := &execution.WorkItems[itemIndex]
			switch item.State {
			case WorkItemStateBinding:
				item.State = WorkItemStateReady
				item.Requested = false
				item.UpdatedAt = now
				changed = true
			case WorkItemStateDispatched:
				item.State = WorkItemStateUncertain
				item.Error = &SanitizedError{
					Code:    "recovered_external_outcome_uncertain",
					Message: "external WorkItem was dispatched when the runtime stopped",
				}
				item.CompletedAt = &now
				item.UpdatedAt = now
				needsReview = true
				changed = true
				for attemptIndex := range execution.Attempts {
					attempt := &execution.Attempts[attemptIndex]
					if attempt.WorkItemID == item.ID && attempt.State == AttemptStateDispatched {
						attempt.State = AttemptStateUncertain
						attempt.Error = item.Error
						attempt.FinishedAt = &now
					}
				}
			case WorkItemStateReady,
				WorkItemStateWaiting,
				WorkItemStateCompleted,
				WorkItemStateFailed,
				WorkItemStateUncertain,
				WorkItemStateCancelled,
				WorkItemStateBlocked:
				// No recovery action.
			}
		}
		for waitIndex := range execution.Waits {
			wait := &execution.Waits[waitIndex]
			if wait.Request.RuntimeBound && (wait.Status == WaitStatusPending || wait.Status == WaitStatusResolving) {
				wait.Status = WaitStatusInvalid
				wait.Error = &SanitizedError{
					Code:    "recovered_runtime_wait_unavailable",
					Message: "runtime authorization state was not persisted",
				}
				wait.UpdatedAt = now
				needsReview = true
				changed = true
			}
		}
		if needsReview && !execution.State.Terminal() {
			recovery := Wait{
				ID:         WaitID(r.newID()),
				WorkItemID: "recovery",
				Status:     WaitStatusPending,
				Request: WaitRequest{
					Kind: WaitKindRecovery,
					Summary: MustCanonicalValue(
						map[string]string{"message": "recovery review is required before external work can continue"},
					),
					AllowedResolutions: []WaitResolution{WaitResolutionContinue, WaitResolutionCancel},
				},
				CreatedAt: now,
				UpdatedAt: now,
			}
			execution.Waits = append(execution.Waits, recovery)
			execution.State = ExecutionStateWaiting
			execution.Revision++
			execution.UpdatedAt = now
			session.Timeline = append(
				session.Timeline,
				Activity{
					ID:          ActivityID(r.newID()),
					Kind:        ActivityKindWait,
					Visibility:  ActivityVisibilityBoth,
					CreatedAt:   now,
					ExecutionID: execution.ID,
					Wait: &WaitActivity{
						WaitID:  recovery.ID,
						Status:  recovery.Status,
						Summary: "Recovery review is required.",
					},
				},
			)
			changed = true
		}
	}
	if changed {
		r.touchSessionLocked(session)
	}
	return changed
}
