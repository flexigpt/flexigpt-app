package sessionv3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/backgroundutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

const (
	maximumCommandFingerprintBytes = 32 << 20
	frameReceiptSystem             = "session-v3-frame"
)

type RuntimeDependencies struct {
	Binder       WorkBinder
	Checkpoints  CheckpointSink
	Recovery     RecoverySource
	Events       EventPublisher
	Attachments  AttachmentAdmission
	Interpreters []ProgramInterpreter
	Now          func() time.Time
	NewID        func() string
}

type Runtime struct {
	mu                      sync.RWMutex
	deps                    RuntimeDependencies
	sessions                map[SessionID]*Session
	retired                 map[SessionID]SessionTombstone
	interpreters            map[string]ProgramInterpreter
	privateBindings         map[WorkItemID]BoundWork
	privateWaits            map[WaitID]any
	cancels                 map[WorkItemID]context.CancelFunc
	waitResolutionCancels   map[WaitID]context.CancelFunc
	dueTimers               map[dueTimeKey]context.CancelFunc
	checkpointPending       map[SessionID]bool
	closedFrameBindings     map[string]struct{}
	closedExecutionBindings map[ExecutionID]struct{}
	resourceWaitSequence    uint64

	lifecycleCtx    context.Context
	lifecycleCancel context.CancelFunc

	asyncMu      sync.Mutex
	asyncWG      sync.WaitGroup
	asyncClosing bool

	restoring bool
	closed    bool
}

func NewRuntime(dependencies RuntimeDependencies) (*Runtime, error) {
	if dependencies.Binder == nil || dependencies.Checkpoints == nil || dependencies.Recovery == nil {
		return nil, errors.New("session runtime v3 dependencies are incomplete")
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
	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())

	r := &Runtime{
		deps:                    dependencies,
		sessions:                map[SessionID]*Session{},
		retired:                 map[SessionID]SessionTombstone{},
		interpreters:            map[string]ProgramInterpreter{},
		privateBindings:         map[WorkItemID]BoundWork{},
		privateWaits:            map[WaitID]any{},
		cancels:                 map[WorkItemID]context.CancelFunc{},
		waitResolutionCancels:   make(map[WaitID]context.CancelFunc),
		dueTimers:               map[dueTimeKey]context.CancelFunc{},
		checkpointPending:       map[SessionID]bool{},
		closedFrameBindings:     map[string]struct{}{},
		closedExecutionBindings: map[ExecutionID]struct{}{},
		lifecycleCtx:            lifecycleCtx,
		lifecycleCancel:         lifecycleCancel,
	}
	for _, interpreter := range dependencies.Interpreters {
		if interpreter == nil || interpreter.ID() == "" || interpreter.Version() == 0 {
			return nil, errors.New("invalid program frame interpreter")
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
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return ErrRuntimeClosed
	}
	r.restoring = true
	r.mu.Unlock()
	if err := r.deps.Recovery.Restore(ctx, runtimeRestorer{runtime: r}); err != nil {
		r.mu.Lock()
		r.restoring = false
		r.mu.Unlock()
		return err
	}
	r.resumeRestoredWork(ctx)

	// Restore is the one explicit readiness wave that is outside
	// persistLocked. At this point every restored Session, trigger, wait,
	// reconciliation, and due-time fact is stable.
	r.mu.Lock()
	r.restoring = false
	ready := make(map[SessionID][]readyExecutionWork, len(r.sessions))
	for id, session := range r.sessions {
		if r.checkpointPending[id] {
			continue
		}
		ready[id] = r.readyWorkForSessionLocked(session)
	}
	r.mu.Unlock()

	for id, values := range ready {
		r.launchReadyWork(context.WithoutCancel(ctx), id, values)
	}
	return nil
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

	lifecycleCancel := r.lifecycleCancel
	r.lifecycleCancel = nil

	for _, cancel := range r.cancels {
		cancel()
	}

	clear(r.cancels)
	resolutionCancels := make([]context.CancelFunc, 0, len(r.waitResolutionCancels))
	for _, cancel := range r.waitResolutionCancels {
		resolutionCancels = append(resolutionCancels, cancel)
	}
	clear(r.waitResolutionCancels)
	for _, cancel := range resolutionCancels {
		cancel()
	}

	dueCancels := r.takeAllDueTimesLocked()

	executionIDs := make([]ExecutionID, 0)
	for _, session := range r.sessions {
		for _, execution := range session.Executions {
			executionIDs = append(executionIDs, execution.ID)
		}
	}
	clear(r.privateBindings)
	clear(r.privateWaits)
	clear(r.closedFrameBindings)
	clear(r.closedExecutionBindings)
	r.mu.Unlock()

	r.asyncMu.Lock()
	r.asyncClosing = true
	r.asyncMu.Unlock()
	if lifecycleCancel != nil {
		lifecycleCancel()
	}
	for _, cancel := range dueCancels {
		cancel()
	}
	r.asyncWG.Wait()

	closer, supported := r.deps.Binder.(ExecutionBindingCloser)
	if !supported {
		return
	}
	for _, executionID := range executionIDs {
		closeCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		_ = closer.CloseExecutionBinding(closeCtx, executionID)
		cancel()
	}
}

func (r *Runtime) CreateSession(ctx context.Context, command CreateSessionCommand) (*CommandReceipt, error) {
	if command.Meta.ID == "" {
		return nil, errors.New(errCommandIDRequired)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, ErrRuntimeClosed
	}
	for _, existing := range r.sessions {
		if receipt, found, err := r.commandReceiptLocked(
			existing,
			command.Meta,
			command,
		); err != nil || found {
			return receipt, err
		}
	}
	for _, tombstone := range r.retired {
		if receipt, found, err := commandReceiptFromRecords(
			tombstone.Commands,
			command.Meta,
			command,
		); err != nil || found {
			return receipt, err
		}
	}

	id := command.SessionID
	if id == "" {
		id = SessionID(r.newID())
	}
	if _, retired := r.retired[id]; retired {
		return &CommandReceipt{
			CommandID: command.Meta.ID,
			SessionID: id,
			Message:   "session id was previously deleted and cannot be reused",
		}, nil
	}
	if _, found := r.sessions[id]; found {
		return &CommandReceipt{CommandID: command.Meta.ID, SessionID: id, Message: "session already exists"}, nil
	}
	if !command.Metadata.Valid() || validateDefaults(command.Defaults) != nil {
		return &CommandReceipt{
			CommandID: command.Meta.ID,
			SessionID: id,
			Message:   "invalid session defaults or metadata",
		}, nil
	}
	now := r.now()
	value := &Session{
		ID:         id,
		Revision:   1,
		Title:      command.Title,
		CreatedAt:  now,
		UpdatedAt:  now,
		Defaults:   cloneSessionDefaults(command.Defaults),
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
	if err := r.persistLocked(ctx, checkpointReasonSessionCreated, value); err != nil {
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
				return &CommandReceipt{
					CommandID: command.Meta.ID,
					SessionID: session.ID,
					Message:   err.Error(),
				}, nil
			}
			session.Defaults = cloneSessionDefaults(command.Defaults)
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
		return nil, errors.New(errCommandIDRequired)
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}
	session := r.sessions[command.SessionID]
	if session == nil {
		tombstone, found := r.retired[command.SessionID]
		if found {
			receipt, duplicate, receiptErr := commandReceiptFromRecords(
				tombstone.Commands,
				command.Meta,
				command,
			)
			r.mu.Unlock()
			if receiptErr != nil {
				return nil, receiptErr
			}
			if duplicate {
				return receipt, nil
			}
			return nil, ErrSessionNotFound
		}
		r.mu.Unlock()
		return nil, ErrSessionNotFound
	}
	if receipt, found, receiptErr := r.commandReceiptLocked(
		session,
		command.Meta,
		command,
	); receiptErr != nil || found {
		r.mu.Unlock()
		return receipt, receiptErr
	}
	if message := expectedSessionRevision(session, command.Meta); message != "" {
		receipt := r.rejectLocked(session, command.Meta, command, message)
		r.mu.Unlock()
		return receipt, nil
	}

	for _, execution := range session.Executions {
		if !execution.State.Terminal() &&
			!idleNormalConversationExecution(execution) {
			receipt := r.rejectLocked(
				session,
				command.Meta,
				command,
				"active executions must be terminal before deletion",
			)
			r.mu.Unlock()
			return receipt, nil
		}
	}

	previous := cloneSession(*session)
	now := r.now()
	receipt := CommandReceipt{
		CommandID:       command.Meta.ID,
		Accepted:        true,
		SessionID:       session.ID,
		SessionRevision: session.Revision,
	}
	r.rememberCommandLocked(session, command.Meta, command, receipt)

	tombstone := SessionTombstone{
		SessionID: session.ID,
		DeletedAt: now,
		Commands:  append([]CommandRecord(nil), session.Commands...),
	}
	if err := r.persistSessionTombstoneLocked(
		context.WithoutCancel(ctx),
		tombstone,
	); err != nil {
		var committed CheckpointCommitStatus
		if !errors.As(err, &committed) ||
			committed == nil ||
			!committed.CheckpointCommitted() {
			*session = previous
			r.mu.Unlock()
			return nil, err
		}
	}

	executionIDs := make(
		[]ExecutionID,
		0,
		len(session.Executions),
	)
	for _, execution := range session.Executions {
		executionIDs = append(executionIDs, execution.ID)
		r.cancelExecutionDueTimesLocked(execution)
		for _, item := range execution.WorkItems {
			if cancel := r.cancels[item.ID]; cancel != nil {
				cancel()
				delete(r.cancels, item.ID)
			}
			delete(r.privateBindings, item.ID)
		}
		for _, wait := range execution.Waits {
			delete(r.privateWaits, wait.ID)
		}
		delete(r.closedExecutionBindings, execution.ID)
	}

	delete(r.sessions, session.ID)
	r.retired[session.ID] = cloneSessionTombstone(tombstone)
	delete(r.checkpointPending, session.ID)
	r.mu.Unlock()

	if closer, supported := r.deps.Binder.(ExecutionBindingCloser); supported {
		for _, executionID := range executionIDs {
			id := executionID
			r.launchAsync(
				"session-v3 deleted execution binding cleanup",
				func() {
					_ = closer.CloseExecutionBinding(
						context.WithoutCancel(ctx),
						id,
					)
				},
				nil,
			)
		}
	}

	return &receipt, nil
}

func idleNormalConversationExecution(execution Execution) bool {
	if !isNormalConversationExecution(execution) ||
		execution.Controls.CancellationRequested ||
		execution.Cleanup.Active ||
		len(execution.ResourceLeases) != 0 {
		return false
	}
	for _, item := range execution.WorkItems {
		if !item.State.Terminal() ||
			item.State == WorkItemStateUncertain {
			return false
		}
	}
	for _, wait := range execution.Waits {
		if wait.Status == WaitStatusPending ||
			wait.Status == WaitStatusResolving {
			return false
		}
	}
	for _, frame := range execution.ProgramFrames {
		if frame.Fork != nil || len(frame.Joins) != 0 {
			return false
		}
		if frame.ParentFrameID == "" &&
			frame.Program.InterpreterID == BuiltinConversationInterpreterID {
			continue
		}
		if !frame.Status.Terminal() {
			return false
		}
	}
	return true
}

func (r *Runtime) SubmitMessage(ctx context.Context, command SubmitMessageCommand) (*CommandReceipt, error) {
	if command.Meta.ID == "" {
		return nil, errors.New(errCommandIDRequired)
	}
	if err := command.Input.Validate(); err != nil {
		return &CommandReceipt{
			CommandID: command.Meta.ID,
			SessionID: command.SessionID,
			Message:   err.Error(),
		}, nil
	}
	if receipt, found, lookupErr := r.FindCommandReceipt(
		ctx,
		command.SessionID,
		command.Meta,
		command,
	); lookupErr != nil {
		return nil, lookupErr
	} else if found {
		return receipt, nil
	}

	admittedAttachments, admitErr := r.admitMessageAttachments(
		ctx,
		command.SessionID,
		command.Input.AttachmentSources,
	)
	if admitErr != nil {
		return &CommandReceipt{
			CommandID: command.Meta.ID,
			SessionID: command.SessionID,
			Message:   admitErr.Error(),
		}, nil
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

	input := command.Input
	input.Attachments = appendMessageAttachments(
		input.Attachments,
		admittedAttachments,
	)
	input.AttachmentSources = nil
	resolvedDefaults, defaultsErr := resolveSessionInvocationDefaults(
		session.Defaults,
		input.Defaults,
	)
	if defaultsErr != nil {
		receipt := r.rejectLocked(
			session,
			command.Meta,
			command,
			defaultsErr.Error(),
		)
		r.mu.Unlock()
		return receipt, nil
	}

	previous := cloneSession(*session)
	execution, err := r.conversationMessageExecutionLocked(session)
	if err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	stagedAppContext, appSources := stagedConversationAppContextLocked(
		session,
		*execution,
	)

	activity, err := r.submitNormalMessageWithDefaultsLocked(
		session,
		execution,
		input,
		nil,
		&resolvedDefaults,
		stagedAppContext,
	)
	if err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	current := mutableSessionExecution(session, execution.ID)
	if current == nil {
		*session = previous
		r.mu.Unlock()
		return nil, errors.New("conversation execution disappeared during message submission")
	}
	r.clearConsumedAppContextLocked(session, current.ID, appSources)
	r.refreshExecutionStateLocked(current)
	receipt := CommandReceipt{
		CommandID:         command.Meta.ID,
		Accepted:          true,
		SessionID:         session.ID,
		ExecutionID:       current.ID,
		SessionRevision:   session.Revision,
		ExecutionRevision: current.Revision,
		ActivityIDs:       []ActivityID{activity.ID},
	}
	r.rememberCommandLocked(session, command.Meta, command, receipt)
	if err := r.persistLocked(ctx, checkpointReasonMessageSubmitted, session); err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	r.armSessionExecutionTimersLocked(
		ctx,
		session,
	)
	r.publishLocked(ctx, session, current.ID, RuntimeEventExecutionChanged)
	r.mu.Unlock()
	return &receipt, nil
}

func (r *Runtime) StartWorkflow(ctx context.Context, command StartWorkflowCommand) (*CommandReceipt, error) {
	if command.Meta.ID == "" {
		return nil, errors.New("invalid workflow start command")
	}
	workflowProgram, err := validatePreparedWorkflowStartProgram(command.Program)
	if err != nil {
		return nil, err
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
	executionID := command.ExecutionID
	if executionID == "" {
		executionID = ExecutionID(r.newID())
	}

	interpreter := r.interpreterLocked(command.Program)
	if interpreter == nil {
		receipt := r.rejectLocked(
			session,
			command.Meta,
			command,
			"workflow interpreter is unavailable",
		)
		r.mu.Unlock()
		return receipt, nil
	}
	budget, deadline, err := initialExecutionBudget(
		workflowProgram.Limits,
		r.now(),
	)
	if err != nil {
		receipt := r.rejectLocked(session, command.Meta, command, err.Error())
		r.mu.Unlock()
		return receipt, nil
	}
	resources, err := initialExecutionResources(workflowProgram.Resources)
	if err != nil {
		receipt := r.rejectLocked(session, command.Meta, command, err.Error())
		r.mu.Unlock()
		return receipt, nil
	}
	for _, existing := range session.Executions {
		if existing.ID == executionID {
			receipt := r.rejectLocked(session, command.Meta, command, "execution id already exists")
			r.mu.Unlock()
			return receipt, nil
		}
	}
	previous := cloneSession(*session)
	now := r.now()
	state, err := interpreter.InitialState(command.Program, now)
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	execution := Execution{
		ID:              executionID,
		SessionID:       session.ID,
		RootExecutionID: executionID,
		Lifetime:        ExecutionChildLifetimeRoot,
		Revision:        1,
		State:           ExecutionStateRunning,
		CreatedAt:       now,
		UpdatedAt:       now,
		Deadline:        deadline,
		Budget:          budget,
		Usage:           ExecutionBudgetUsage{},
		Cleanup:         CleanupState{},
		ProgramFrames: []ProgramFrame{
			{
				ID:              ProgramFrameID(r.newID()),
				Depth:           1,
				Program:         command.Program,
				State:           state,
				Status:          ProgramFrameStateRunning,
				Path:            "workflow",
				DeclarationPath: workflowProgram.Root.DeclarationPath,
				CreatedAt:       now,
				UpdatedAt:       now,
				Deadline:        cloneTime(deadline),
				Evaluation: &EvaluationBoundary{
					DeclarationPath: workflowProgram.Root.DeclarationPath,
					LogicalTime:     now,
				},
			},
		},
		WorkItems:      []WorkItem{},
		Attempts:       []Attempt{},
		Waits:          []Wait{},
		Resources:      resources,
		ResourceLeases: []ResourceLease{},
		ResourceRates:  []ResourceRateWindow{},
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
			Workflow: &WorkflowActivity{
				Phase:   WorkflowActivityPhaseStarted,
				Summary: "Workflow execution was admitted.",
			},
		},
	)
	if err := r.reserveActivityBudgetLocked(active, session.Timeline[len(session.Timeline)-1]); err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	if err := r.advanceFrameLocked(
		session,
		active,
		active.ProgramFrames[0].ID,
		ProgramEvent{Kind: ProgramEventStarted},
	); err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	active = mutableSessionExecution(session, executionID)
	r.refreshExecutionStateLocked(active)
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
	if err := r.persistLocked(ctx, checkpointReasonWorkflowStarted, session); err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	r.armExecutionDueTimesLocked(
		ctx,
		session.ID,
		active.ID,
		*active,
	)
	r.publishLocked(ctx, session, active.ID, RuntimeEventExecutionChanged)
	r.mu.Unlock()
	return &receipt, nil
}

// RegisterWorkflowTriggers admits an input-unbound Workflow template and
// persists its trigger registrations without creating an initial Execution.
// Schedule timers remain Runtime-owned and are armed only after persistence.
func (r *Runtime) RegisterWorkflowTriggers(
	ctx context.Context,
	command RegisterWorkflowTriggersCommand,
) (*CommandReceipt, error) {
	if command.Meta.ID == "" {
		return nil, errors.New(errCommandIDRequired)
	}
	workflowProgram, err := validatePreparedWorkflowTriggersProgram(command.Program)
	if err != nil {
		return nil, err
	}

	receipt, err := r.changeSession(
		ctx,
		command.Meta,
		command.SessionID,
		command,
		func(session *Session) (*CommandReceipt, error) {
			interpreter := r.interpreterLocked(command.Program)
			if interpreter == nil {
				return &CommandReceipt{
					CommandID: command.Meta.ID,
					SessionID: session.ID,
					Message:   "workflow trigger interpreter is unavailable",
				}, nil
			}
			if err := r.registerWorkflowTriggersLocked(
				session,
				command.Program,
				workflowProgram,
			); err != nil {
				return nil, err
			}

			r.touchSessionLocked(session)
			return &CommandReceipt{
				CommandID:       command.Meta.ID,
				Accepted:        true,
				SessionID:       session.ID,
				SessionRevision: session.Revision,
			}, nil
		},
	)
	if err != nil || receipt == nil || !receipt.Accepted {
		return receipt, err
	}

	r.mu.Lock()
	if !r.closed {
		if session := r.sessions[command.SessionID]; session != nil {
			r.armSessionTriggersLocked(context.WithoutCancel(ctx), session)
		}
	}
	r.mu.Unlock()
	return receipt, nil
}

func (r *Runtime) SetTriggerRegistrationState(
	ctx context.Context,
	command SetTriggerRegistrationStateCommand,
) (*CommandReceipt, error) {
	receipt, err := r.changeSession(
		ctx,
		command.Meta,
		command.SessionID,
		command,
		func(session *Session) (*CommandReceipt, error) {
			switch command.State {
			case TriggerRegistrationStateActive, TriggerRegistrationStateDisabled:
			default:
				return &CommandReceipt{
					CommandID: command.Meta.ID,
					SessionID: session.ID,
					Message:   "trigger registration state is invalid",
				}, nil
			}

			registration := mutableTriggerRegistration(session, command.TriggerID)
			if registration == nil {
				return &CommandReceipt{
					CommandID: command.Meta.ID,
					SessionID: session.ID,
					Message:   "trigger registration is unavailable",
				}, nil
			}

			if command.State == TriggerRegistrationStateDisabled {
				r.cancelTriggerLocked(session.ID, registration.ID)
			}
			registration.State = command.State
			registration.UpdatedAt = r.now()

			if command.State == TriggerRegistrationStateActive {
				if _, err := r.recoverSessionTriggersLocked(
					ctx,
					session,
				); err != nil {
					return nil, err
				}
				_ = r.drainTriggerQueuesLocked(
					ctx,
					session,
					"",
				)
			}

			r.touchSessionLocked(session)
			return &CommandReceipt{
				CommandID:       command.Meta.ID,
				Accepted:        true,
				SessionID:       session.ID,
				SessionRevision: session.Revision,
			}, nil
		},
	)
	if err != nil || receipt == nil || !receipt.Accepted ||
		command.State != TriggerRegistrationStateActive {
		return receipt, err
	}

	r.mu.Lock()
	if !r.closed {
		if session := r.sessions[command.SessionID]; session != nil {
			r.armSessionTriggersLocked(context.WithoutCancel(ctx), session)
		}
	}
	r.mu.Unlock()
	return receipt, nil
}

func (r *Runtime) SignalExecution(ctx context.Context, command SignalExecutionCommand) (*CommandReceipt, error) {
	if command.Name == "" || !command.Data.Valid() || !command.Correlation.Valid() {
		return nil, errors.New("invalid workflow signal")
	}
	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		command,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			if len(execution.Signals) >= maximumCoordinationRecords {
				return nil, errors.New("execution signal retention limit exceeded")
			}
			ownerFrameID := ProgramFrameID("")
			if len(execution.ProgramFrames) != 0 {
				ownerFrameID = execution.ProgramFrames[0].ID
			}
			if err := r.reserveEventBudgetForOwnerLocked(
				execution,
				WorkItemID(""),
				ownerFrameID,
				len(command.Data)+len(command.Correlation),
			); err != nil {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     err.Error(),
				}, nil
			}
			source := strings.TrimSpace(command.Source)
			if source == "" {
				source = "command"
			}
			signalID := strings.TrimSpace(command.SignalID)
			if signalID == "" {
				signalID = string(command.Meta.ID)
			}
			signal := ExecutionSignal{
				ID:           signalID,
				Name:         command.Name,
				Source:       source,
				Data:         command.Data.Clone(),
				Correlation:  command.Correlation.Clone(),
				CreatedAt:    r.now(),
				OwnerFrameID: ownerFrameID,
			}
			for _, existing := range execution.Signals {
				if existing.ID == signal.ID &&
					existing.Source == signal.Source {
					return nil, errors.New(
						"execution signal source and identity already exist",
					)
				}
			}
			execution.Signals = append(execution.Signals, signal)
			if err := r.acceptSignalForSubscriptionsLocked(
				execution,
				signal,
			); err != nil {
				return nil, err
			}
			if err := r.wakeEventWaitsLocked(
				session,
				execution,
				signal,
			); err != nil {
				return nil, err
			}
			executionID := execution.ID
			frameIDs := make([]ProgramFrameID, 0, len(execution.ProgramFrames))
			for _, frame := range execution.ProgramFrames {
				frameIDs = append(frameIDs, frame.ID)
			}
			for _, frameID := range frameIDs {
				execution = mutableSessionExecution(session, executionID)
				if execution == nil || execution.State.Terminal() {
					break
				}
				frame := mutableFrame(execution, frameID)
				if frame == nil {
					continue
				}
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
						Phase:   WorkflowActivityPhaseSignal,
						Summary: command.Name,
						Output: jsonutil.MustCanonicalValue(struct {
							Correlation jsonutil.CanonicalValue `json:"correlation,omitempty"`
							Data        jsonutil.CanonicalValue `json:"data"`
						}{
							Correlation: command.Correlation.Clone(),
							Data:        command.Data.Clone(),
						}),
					},
				}
				if err := r.appendBudgetedActivityLocked(session, execution, activity); err != nil {
					return nil, err
				}
				if err := r.advanceFrameLocked(
					session,
					execution,
					frame.ID,
					ProgramEvent{Kind: ProgramEventActivity, ActivityID: activity.ID},
				); err != nil {
					return nil, err
				}
			}
			execution = mutableSessionExecution(session, executionID)
			if execution == nil {
				return nil, ErrExecutionNotFound
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

// AcceptTriggerEvent records one external event occurrence and routes it to
// durable Workflow trigger registrations in this Session. The trigger
// registry, deduplication facts, selected Execution, and queue state remain
// inside the Session document.
func (r *Runtime) AcceptTriggerEvent(
	ctx context.Context,
	command AcceptTriggerEventCommand,
) (*CommandReceipt, error) {
	return r.acceptTriggerEvent(ctx, command)
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
						Message:     "requested work item is not ready",
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
	identity := command
	r.mu.RLock()
	if !r.closed {
		if session := r.sessions[command.SessionID]; session != nil {
			command.ExecutionID = r.resolveConversationControlExecutionIDLocked(
				session,
				command.ExecutionID,
			)
		}
	}
	r.mu.RUnlock()

	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		identity,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			if activityIDs, consumed, err := r.consumeContinuationLocked(
				session,
				execution,
			); err != nil {
				return nil, err
			} else if consumed {
				r.refreshExecutionStateLocked(execution)
				return &CommandReceipt{
					CommandID:         command.Meta.ID,
					Accepted:          true,
					SessionID:         session.ID,
					ExecutionID:       execution.ID,
					SessionRevision:   session.Revision,
					ExecutionRevision: execution.Revision,
					ActivityIDs:       activityIDs,
				}, nil
			}

			execution.Controls.ContinueRequested = true
			if err := r.requestAgentContinuationLocked(session, execution); err != nil {
				return nil, err
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

func (r *Runtime) CompleteConversationHandoff(
	ctx context.Context,
	command CompleteConversationHandoffCommand,
) (*CommandReceipt, error) {
	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		command,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			for _, frame := range execution.ProgramFrames {
				if frame.HandoffID == "" ||
					frame.HandoffID != command.HandoffID ||
					frame.Status.Terminal() {
					continue
				}
				if err := r.advanceFrameLocked(
					session,
					execution,
					frame.ID,
					ProgramEvent{Kind: ProgramEventHandoffComplete},
				); err != nil {
					return nil, err
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
			}
			return &CommandReceipt{
				CommandID:   command.Meta.ID,
				SessionID:   session.ID,
				ExecutionID: execution.ID,
				Message:     "active conversation handoff is unavailable",
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
			r.beginStoppingCleanupLocked(
				execution,
				&SanitizedError{
					Code:     "execution_cancelled",
					Category: ErrorCategoryCancellation,
					Message:  firstText(command.Reason, "execution was cancelled"),
				},
			)
			r.cancelOwnedChildExecutionsLocked(
				ctx,
				session,
				execution.ID,
				firstText(command.Reason, "execution was cancelled"),
			)
			executionID := execution.ID
			workItemIDs := make([]WorkItemID, 0, len(execution.WorkItems))
			for _, item := range execution.WorkItems {
				if item.State.Terminal() {
					continue
				}
				workItemIDs = append(workItemIDs, item.ID)
			}
			for _, workItemID := range workItemIDs {
				current := mutableSessionExecution(session, executionID)
				if current == nil {
					return nil, ErrExecutionNotFound
				}
				item := mutableWorkItem(current, workItemID)
				if item == nil || item.State.Terminal() {
					continue
				}
				r.cancelFrameWorkItemLocked(session, current, item)
			}
			execution = mutableSessionExecution(session, executionID)
			if execution == nil {
				return nil, ErrExecutionNotFound
			}

			rootFrames := make([]ProgramFrameID, 0, len(execution.ProgramFrames))

			for _, frame := range execution.ProgramFrames {
				if frame.ParentFrameID == "" &&
					!frame.Status.Terminal() &&
					!workflowFrameRunsCleanup(frame) {
					rootFrames = append(rootFrames, frame.ID)
				}
			}
			for _, frameID := range rootFrames {
				if err := r.cancelFrameTreeLocked(
					session,
					execution,
					frameID,
				); err != nil {
					return nil, err
				}
			}
			execution, err := r.finalizeExecutionSettlementLocked(
				ctx,
				session,
				executionID,
				nil,
			)
			if err != nil {
				return nil, err
			}
			if !execution.State.Terminal() {
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
	return r.acceptWaitResolution(ctx, command)
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

// FindCommandReceipt returns an already accepted receipt before a caller
// performs setup-owned preparation. It does not reserve a command or mutate
// Runtime state.
//
// Workflow wrappers use this method before source resolution. Message command
// paths use the same check before attachment admission. This lets an accepted
// duplicate replay after source files, selected Artifacts, or setup resources
// are no longer available.
func (r *Runtime) FindCommandReceipt(
	_ context.Context,
	sessionID SessionID,
	meta CommandMeta,
	payload any,
) (*CommandReceipt, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return nil, false, ErrRuntimeClosed
	}
	if session := r.sessions[sessionID]; session != nil {
		return r.commandReceiptLocked(session, meta, payload)
	}

	tombstone, found := r.retired[sessionID]
	if !found {
		return nil, false, ErrSessionNotFound
	}
	receipt, accepted, err := commandReceiptFromRecords(
		tombstone.Commands,
		meta,
		payload,
	)
	if err != nil || accepted {
		return receipt, accepted, err
	}
	return nil, false, ErrSessionNotFound
}

func (r *Runtime) PrepareUserMessageSkillTemplate(
	ctx context.Context,
	request UserMessageSkillTemplatePreparationRequest,
) (UserMessageSkillTemplateResult, error) {
	if r == nil {
		return UserMessageSkillTemplateResult{}, ErrRuntimeClosed
	}

	r.mu.RLock()
	closed := r.closed
	binder := r.deps.Binder
	r.mu.RUnlock()
	if closed {
		return UserMessageSkillTemplateResult{}, ErrRuntimeClosed
	}

	preparer, found := binder.(UserMessageSkillTemplatePreparer)
	if !found || preparer == nil {
		return UserMessageSkillTemplateResult{}, errors.New(
			"user-message skill template preparation is unavailable",
		)
	}
	return preparer.PrepareUserMessageSkillTemplate(ctx, request)
}

func (r *Runtime) PreviewInstructionSkill(
	ctx context.Context,
	request InstructionSkillPreviewRequest,
) (InstructionSkillPreviewResult, error) {
	if r == nil {
		return InstructionSkillPreviewResult{}, ErrRuntimeClosed
	}

	r.mu.RLock()
	closed := r.closed
	binder := r.deps.Binder
	r.mu.RUnlock()
	if closed {
		return InstructionSkillPreviewResult{}, ErrRuntimeClosed
	}

	previewer, found := binder.(InstructionSkillPreviewer)
	if !found || previewer == nil {
		return InstructionSkillPreviewResult{}, errors.New(
			"instruction skill preview is unavailable",
		)
	}
	return previewer.PreviewInstructionSkill(ctx, request)
}

func (r *Runtime) Conversation(ctx context.Context, id SessionID) (ConversationProjection, error) {
	session, err := r.GetSession(ctx, id)
	if err != nil {
		return ConversationProjection{}, err
	}
	return BuildConversationProjection(session), nil
}

func (r *Runtime) Activities(ctx context.Context, id SessionID) (ActivityProjection, error) {
	session, err := r.GetSession(ctx, id)
	if err != nil {
		return ActivityProjection{}, err
	}
	return BuildActivityProjection(session), nil
}

func (r *Runtime) Workflow(ctx context.Context, id SessionID) (WorkflowProjection, error) {
	session, err := r.GetSession(ctx, id)
	if err != nil {
		return WorkflowProjection{}, err
	}
	return BuildWorkflowProjection(session), nil
}

func (r *Runtime) persistSessionTombstoneLocked(
	ctx context.Context,
	tombstone SessionTombstone,
) error {
	if sink, ok := r.deps.Checkpoints.(SessionTombstoneDeletionSink); ok {
		return sink.HandleSessionTombstone(ctx, tombstone)
	}
	if sink, ok := r.deps.Checkpoints.(SessionDeletionSink); ok {
		return sink.HandleSessionDeleted(ctx, tombstone.SessionID)
	}
	return errors.New("session runtime checkpoint sink cannot persist tombstones")
}

func (r *Runtime) admitMessageAttachments(
	ctx context.Context,
	sessionID SessionID,
	sources []AttachmentSource,
) ([]Attachment, error) {
	if len(sources) == 0 {
		return nil, nil
	}
	if r == nil || r.deps.Attachments == nil {
		return nil, errors.New("attachment admission is unavailable")
	}

	return r.deps.Attachments.AdmitAttachments(
		ctx,
		AttachmentAdmissionRequest{
			SessionID: sessionID,
			Sources:   append([]AttachmentSource(nil), sources...),
		},
	)
}

func appendMessageAttachments(
	persisted []Attachment,
	admitted []Attachment,
) []Attachment {
	output := make([]Attachment, 0, len(persisted)+len(admitted))
	output = append(output, persisted...)
	output = append(output, admitted...)
	if len(output) == 0 {
		return nil
	}
	return output
}

func (r *Runtime) launchAsync(
	operation string,
	work func(),
	onPanic func(backgroundutil.PanicFailureError),
) {
	if r == nil {
		return
	}
	r.asyncMu.Lock()
	defer r.asyncMu.Unlock()
	if r.asyncClosing {
		return
	}
	backgroundutil.Launch(&r.asyncWG, operation, work, onPanic)
}

func (r *Runtime) launchWork(
	_ context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	items []WorkItemID,
) {
	for _, itemID := range items {
		id := itemID
		r.launchAsync(
			"session-v3 WorkItem dispatch",
			func() {
				r.dispatchOne(r.lifecycleCtx, sessionID, executionID, id)
			},
			func(failure backgroundutil.PanicFailureError) {
				r.finishPanickedWork(sessionID, executionID, id, failure)
			},
		)
	}
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
					Message:     "execution is terminal",
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
		return nil, errors.New(errCommandIDRequired)
	}
	r.mu.Lock()
	defer func() {
		r.mu.Unlock()
	}()
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
		*session = previous
		return receipt, err
	}
	r.rememberCommandLocked(session, meta, payload, *receipt)
	if err := r.persistLocked(ctx, checkpointReasonSessionChanged, session); err != nil {
		*session = previous
		return nil, err
	}
	r.armSessionExecutionTimersLocked(ctx, session)
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
		return nil, errors.New(errCommandIDRequired)
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
		*session = previous
		r.mu.Unlock()
		return receipt, err
	}
	execution, err = r.finalizeExecutionSettlementLocked(
		ctx,
		session,
		executionID,
		nil,
	)
	if err != nil || execution == nil {
		*session = previous
		r.mu.Unlock()
		return nil, errors.New("execution disappeared during command mutation")
	}
	receipt.SessionID = session.ID
	receipt.ExecutionID = execution.ID
	receipt.SessionRevision = session.Revision
	receipt.ExecutionRevision = execution.Revision
	r.rememberCommandLocked(session, meta, payload, *receipt)
	if err := r.persistLocked(ctx, checkpointReasonExecutionChanged, session); err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	r.armSessionExecutionTimersLocked(ctx, session)
	r.publishLocked(ctx, session, execution.ID, RuntimeEventExecutionChanged)
	r.mu.Unlock()
	return receipt, nil
}

func (r *Runtime) conversationExecutionLocked(session *Session) (*Execution, error) {
	for index := range session.Executions {
		execution := &session.Executions[index]
		if !execution.State.Terminal() && len(execution.ProgramFrames) != 0 &&
			execution.ProgramFrames[0].Program.InterpreterID ==
				BuiltinConversationInterpreterID {
			return execution, nil
		}
	}
	program, err := NewConversationProgram(session.Defaults)
	if err != nil {
		return nil, err
	}

	budget := ExecutionBudget{}
	var deadline *time.Time
	if session.Defaults.ConversationBudget != nil {
		budget, deadline, err = initialExecutionBudget(
			*session.Defaults.ConversationBudget,
			r.now(),
		)
		if err != nil {
			return nil, err
		}
	}

	interpreter := r.interpreterLocked(program)
	if interpreter == nil {
		return nil, errors.New("conversation interpreter unavailable")
	}
	now := r.now()
	state, err := interpreter.InitialState(program, now)
	if err != nil {
		return nil, err
	}
	execution := Execution{
		ID:        ExecutionID(r.newID()),
		SessionID: session.ID,
		Lifetime:  ExecutionChildLifetimeRoot,
		Revision:  1,
		State:     ExecutionStateWaiting,
		CreatedAt: now,
		UpdatedAt: now,
		Deadline:  deadline,
		Budget:    budget,
		Usage:     ExecutionBudgetUsage{},
		Cleanup:   CleanupState{},
		ProgramFrames: []ProgramFrame{
			{
				ID:        ProgramFrameID(r.newID()),
				Depth:     1,
				Program:   program,
				State:     state,
				Status:    ProgramFrameStateWaiting,
				Path:      "conversation",
				CreatedAt: now,
				UpdatedAt: now,
				Deadline:  cloneTime(deadline),
			},
		},
		WorkItems: []WorkItem{},
		Attempts:  []Attempt{},
		Waits:     []Wait{},
	}
	execution.RootExecutionID = execution.ID
	session.Executions = append(session.Executions, execution)
	return &session.Executions[len(session.Executions)-1], nil
}

func (r *Runtime) advanceFrameLocked(
	session *Session,
	execution *Execution,
	frameID ProgramFrameID,
	event ProgramEvent,
) (err error) {
	if session == nil || execution == nil {
		return errors.New("program frame execution is unavailable")
	}

	parentExecutionID := execution.ID
	previousExecution := cloneExecution(*execution)
	previousExecutionCount := len(session.Executions)
	previousTimelineLength := len(session.Timeline)
	previousConversationOwnership := cloneSession(*session).ConversationOwnership
	defer func() {
		if err != nil {
			if current := mutableSessionExecution(session, parentExecutionID); current != nil {
				*current = previousExecution
			}
			if len(session.Executions) > previousExecutionCount {
				session.Executions = session.Executions[:previousExecutionCount]
			}
			session.Timeline = session.Timeline[:previousTimelineLength]
			session.ConversationOwnership = previousConversationOwnership
		}
	}()

	frame := mutableFrame(execution, frameID)
	if frame == nil || frame.Status.Terminal() {
		return nil
	}
	reacquire := func() error {
		execution = mutableSessionExecution(session, parentExecutionID)
		if execution == nil {
			return errors.New(
				"program frame execution disappeared during transition",
			)
		}
		frame = mutableFrame(execution, frameID)
		if frame == nil {
			return errors.New(
				"program frame disappeared during transition",
			)
		}
		return nil
	}

	if event.Kind == ProgramEventWorkSettled {
		if handled, forkErr := r.settleForkBoundaryLocked(
			session,
			execution,
			frame,
			event,
		); handled {
			return forkErr
		}
		if handled, joinErr := r.settleJoinBoundaryLocked(
			session,
			execution,
			frame,
			event,
		); handled {
			return joinErr
		}
	}
	interpreter := r.interpreterLocked(frame.Program)
	if interpreter == nil {
		return fmt.Errorf("interpreter %q is unavailable", frame.Program.InterpreterID)
	}
	if event.At.IsZero() {
		event.At = r.now()
	}
	transition, err := interpreter.Advance(cloneFrame(*frame), cloneExecution(*execution), event)
	if err != nil {
		return err
	}
	if err := interpreter.ValidateState(transition.State); err != nil {
		return err
	}
	now := r.now()
	frame.State = transition.State
	frame.UpdatedAt = now
	if transition.Status != nil {
		frame.Status = *transition.Status
	}
	if frame.Program.InterpreterID == WorkflowInterpreterID {
		if _, workflowState, decodeErr := decodeWorkflowProgram(*frame); decodeErr == nil {
			if frame.Evaluation == nil {
				frame.Evaluation = &EvaluationBoundary{}
			}
			frame.Evaluation.DeclarationPath = firstText(
				workflowState.PendingDeclarationPath,
				frame.DeclarationPath,
			)
			frame.Evaluation.LogicalTime = workflowState.LogicalTime.UTC()
			frame.Evaluation.Messages = workflowState.Messages.Clone()
		}
	}
	frame.Output = transition.Output.Clone()
	if transition.Error != nil {
		frame.Error = transition.Error
	}
	if frame.Status == ProgramFrameStateCompleted &&
		transition.Error == nil &&
		frame.Receipt == nil {
		frame.Receipt = frameTerminalReceipt(
			*execution,
			event,
			frame.ID,
			now,
		)
	}
	reconciliationFailures, err := r.applyReconciliationSettlementsLocked(
		session,
		execution,
		transition.Reconciliations,
	)
	if err != nil {
		return err
	}
	if err := reacquire(); err != nil {
		return err
	}
	if err := r.applyCompensationSettlementsLocked(
		execution,
		transition.Compensations,
	); err != nil {
		return err
	}
	if err := reacquire(); err != nil {
		return err
	}
	for _, activity := range transition.Activities {
		if err := r.reserveActivityBudgetLocked(execution, activity); err != nil {
			return err
		}
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
	parentRunsCleanup := frame.Cleanup || workflowFrameRunsCleanup(*frame)
	if parentRunsCleanup {
		for index := range transition.Frames {
			transition.Frames[index].Cleanup = true
			transition.Frames[index].Deadline = r.cleanupOperationDeadlineLocked(
				*execution,
				transition.Frames[index].Deadline,
			)
		}
		for index := range transition.WorkItems {
			transition.WorkItems[index].Cleanup = true
			transition.WorkItems[index].Deadline = r.cleanupOperationDeadlineLocked(
				*execution,
				transition.WorkItems[index].Deadline,
			)
		}
	}
	if len(transition.Frames) != 0 {
		forkDependencies := childCallForkDependencies(
			transition.Frames,
			transition.WorkItems,
		)
		if err := r.deferFrameForkLocked(
			execution,
			frame,
			transition.Frames,
			transition.ContinueParent,
			transition.ForkClaims,
			forkDependencies,
		); err != nil {
			return err
		}
		transition.Frames = nil
		transition.ContinueParent = false
	}
	for _, intent := range transition.WorkItems {
		itemBudget, budgetErr := workItemBudgetForFrame(
			*execution,
			*frame,
			intent.Budget,
		)
		if budgetErr != nil {
			return budgetErr
		}
		now := r.now()
		item := WorkItem{
			ID:               intent.ID,
			FrameID:          frame.ID,
			Type:             intent.Type,
			OriginWorkItemID: intent.OriginWorkItemID,
			State:            WorkItemStateReady,
			Claims: append(
				[]ResourceClaim(nil),
				intent.Claims...,
			),
			ProgramPath:          intent.ProgramPath,
			DeclarationPath:      firstText(intent.DeclarationPath, frame.DeclarationPath),
			Deadline:             earlierDeadline(frame.Deadline, intent.Deadline),
			Effect:               intent.Effect,
			Retry:                cloneWorkflowRetryPolicy(intent.Retry),
			Automatic:            intent.Automatic,
			Dependencies:         append([]WorkItemID(nil), intent.Dependencies...),
			Budget:               itemBudget,
			BudgetScopeID:        firstText(intent.BudgetScopeID, frame.BudgetScopeID),
			Cleanup:              intent.Cleanup,
			Evaluation:           cloneEvaluationBoundary(intent.Evaluation),
			SagaID:               intent.SagaID,
			CompensationPolicy:   intent.CompensationPolicy,
			RetainClaimsForFrame: intent.RetainClaimsForFrame,
			Ports:                bindWorkPortPlans(intent.Ports, frame.ID),
			Input:                intent.Input,
			PortInputs:           clonePortInputs(intent.PortInputs),
			CreatedAt:            now,
			UpdatedAt:            now,
			Usage:                ExecutionBudgetUsage{},
		}
		if item.ID == "" {
			item.ID = WorkItemID(r.newID())
		}
		item.Ports = bindWorkPortPlans(intent.Ports, frame.ID)
		if item.Type == WorkItemTypeLLM {
			item.Ports, err = mergeWorkPortPlans(
				item.Ports,
				llmPortPlansForFrame(*frame),
			)
			if err != nil {
				return err
			}
		}
		item.EffectPlan = cloneEffectPlan(intent.EffectPlan)
		if mutableWorkItem(execution, item.ID) != nil {
			return fmt.Errorf(
				"work item %q already exists in execution",
				item.ID,
			)
		}
		if err := item.Validate(); err != nil {
			return err
		}
		execution.WorkItems = append(execution.WorkItems, item)
		stored := &execution.WorkItems[len(execution.WorkItems)-1]
		if err := r.markCompensationIntentLocked(execution, stored); err != nil {
			return err
		}
		frame = mutableFrame(execution, frameID)
		if frame == nil {
			return errors.New("parent program frame disappeared while adding work")
		}
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
		if item.Type == WorkItemTypeLLM && item.Input.LLM != nil {
			for _, consumedID := range item.Input.LLM.ContinuationWorkItemIDs {
				consumed := mutableWorkItem(execution, consumedID)
				if consumed == nil ||
					consumed.Type != WorkItemTypeTool ||
					consumed.ConsumedByWorkItemID != "" {
					continue
				}
				consumed.ConsumedByWorkItemID = item.ID
				consumed.UpdatedAt = now
			}
		}
	}
	if err := r.cancelChildFramesLocked(
		session,
		execution,
		frameID,
		transition.CancelChildPaths,
	); err != nil {
		return err
	}
	if err := reacquire(); err != nil {
		return err
	}
	if err := r.cancelChildFrameIDsLocked(
		session,
		execution,
		frameID,
		transition.CancelChildFrameIDs,
	); err != nil {
		return err
	}
	if err := reacquire(); err != nil {
		return err
	}
	if err := r.cancelWorkItemsLocked(
		session,
		execution,
		frameID,
		transition.CancelWorkItemIDs,
	); err != nil {
		return err
	}
	if err := reacquire(); err != nil {
		return err
	}
	if err := r.cancelChildExecutionIDsLocked(
		context.Background(),
		session,
		execution,
		frameID,
		transition.CancelChildExecutionIDs,
		workflowChildCancelled,
	); err != nil {
		return err
	}
	if err := reacquire(); err != nil {
		return err
	}
	if transition.ContinueParent {
		if transition.ReleaseFrameClaims {
			released := r.releaseFrameResourcesLocked(execution, frame.ID)
			r.wakeResourceWaitersLocked(execution, released)
		}

		frame = mutableFrame(execution, frameID)
		if frame == nil {
			return errors.New(
				"parent program frame disappeared during continuation",
			)
		}
		if !frame.Status.Terminal() {
			return r.advanceFrameLocked(
				session,
				execution,
				frameID,
				ProgramEvent{Kind: ProgramEventContinue},
			)
		}
	}
	for _, workItemID := range reconciliationFailures {
		r.advanceSettledWorkLocked(
			session,
			execution,
			frameID,
			workItemID,
		)
		if err := reacquire(); err != nil {
			return err
		}
	}
	if err := reacquire(); err != nil {
		return err
	}
	if frame.Status.Terminal() && frame.HandoffID != "" {
		r.settleConversationHandoffLocked(session, frame.HandoffID)
	}
	if frame.Status.Terminal() {
		released := r.releaseFrameResourcesLocked(execution, frame.ID)
		r.wakeResourceWaitersLocked(execution, released)
	}
	if transition.ReleaseFrameClaims && !frame.Status.Terminal() {
		released := r.releaseFrameResourcesLocked(execution, frame.ID)
		r.wakeResourceWaitersLocked(execution, released)
	}
	if frame.Status == ProgramFrameStateCompleted {
		deliveryErr := r.deliverFramePortsLocked(session, execution, frame)
		if err := reacquire(); err != nil {
			return err
		}
		if deliveryErr != nil {
			frame.Status = ProgramFrameStateFailed
			frame.Error = portPlanError(
				"port_delivery_failed",
				deliveryErr.Error(),
			)
			frame.UpdatedAt = r.now()
		}
	}
	if frame.Status.Terminal() && frame.ParentFrameID != "" {
		r.wakeCompletionFrameWaitsLocked(session, execution, *frame)
		return r.queueChildJoinLocked(
			session,
			execution,
			frame.ParentFrameID,
			frame.ID,
		)
	}
	r.beginFailureDrivenCleanupLocked(execution, frame.Error)
	execution = mutableSessionExecution(session, parentExecutionID)
	if execution == nil {
		return errors.New("program frame execution disappeared after cleanup classification")
	}
	if err := r.settlePendingFrameCancellationsLocked(session, execution); err != nil {
		return err
	}
	wasTerminal := execution.State.Terminal()
	r.refreshExecutionStateLocked(execution)
	if !wasTerminal && execution.State.Terminal() {
		r.settleTerminalExecutionLocked(session, execution)
	}
	return nil
}

func frameTerminalReceipt(
	execution Execution,
	event ProgramEvent,
	frameID ProgramFrameID,
	now time.Time,
) *ExternalReceipt {
	if event.Kind == ProgramEventWorkSettled {
		item := executionWorkItem(execution, event.WorkItemID)
		if item != nil &&
			item.FrameID == frameID &&
			item.Output != nil &&
			item.Output.Receipt != nil {
			value := *item.Output.Receipt
			return &value
		}
	}

	return &ExternalReceipt{
		System:      frameReceiptSystem,
		Reference:   string(frameID),
		ConfirmedAt: now,
		EffectState: EffectStateNotApplicable,
	}
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
	if item == nil || item.State != WorkItemStateReady ||
		!r.dispatchAllowedForWorkLocked(*execution, *item) ||
		(!item.Automatic && !item.Requested) ||
		(item.Type == WorkItemTypeTool &&
			!normalToolDispatchAllowed(*execution, *item)) ||
		unresolvedDependencies(*execution, *item) {
		r.mu.Unlock()
		return
	}
	if !r.reserveWorkStepLocked(ctx, session, execution, item) {
		r.mu.Unlock()
		return
	}
	if !item.Cleanup &&
		!r.workConcurrencyAvailableLocked(
			*execution,
			*item,
		) {
		previous := cloneSession(*session)
		r.setBudgetConcurrencyWaitingLocked(item)
		r.touchExecutionLocked(session, execution)
		if err := r.persistLocked(
			ctx,
			checkpointReasonWorkConcurrencyWaiting,
			session,
		); err != nil {
			*session = previous
			r.mu.Unlock()
			return
		}
		r.publishLocked(
			ctx,
			session,
			execution.ID,
			RuntimeEventWorkChanged,
		)
		r.mu.Unlock()
		return
	}
	if budgetErr := r.checkDispatchBudgetLocked(*execution, *item); budgetErr != nil {
		failure, found := budgetFailure(budgetErr)
		if !found {
			failure = &SanitizedError{
				Code:     budgetErrorCode,
				Category: ErrorCategoryLimit,
				Message:  budgetErr.Error(),
			}
		}
		r.blockWorkForBudgetLocked(
			ctx,
			session,
			execution,
			item,
			failure,
			checkpointReasonWorkBudgetBlocked,
		)
		r.mu.Unlock()
		return
	}
	if err := r.stageWorkPortInputsLocked(session, execution, item); err != nil {
		execution = r.settleUndispatchedWorkLocked(
			session,
			execution,
			item,
			WorkItemStateBlocked,
			portPlanError("port_input_invalid", err.Error()),
			true,
		)
		if execution != nil {
			_ = r.commitTerminalMutationLocked(
				ctx,
				session,
				execution.ID,
				checkpointReasonPortInputBlocked,
				RuntimeEventWorkChanged,
			)
		}
		r.mu.Unlock()
		return
	}
	previous := cloneSession(*session)
	acquired, reason, wakeAt, resourceErr := r.acquireWorkResourcesLocked(
		execution,
		item,
	)
	if resourceErr != nil {
		failure := &SanitizedError{
			Code:     "resource_claim_invalid",
			Category: ErrorCategoryValidation,
			Message:  resourceErr.Error(),
		}
		execution = r.settleUndispatchedWorkLocked(
			session,
			execution,
			item,
			WorkItemStateBlocked,
			failure,
			true,
		)
		if execution == nil ||
			!r.commitTerminalMutationLocked(
				ctx,
				session,
				execution.ID,
				checkpointReasonResourceClaimInvalid,
				RuntimeEventWorkChanged,
			) {
			*session = previous
		}
		r.mu.Unlock()
		return
	}
	if !acquired {
		r.setResourceWaitingLocked(item, reason, wakeAt)
		r.touchExecutionLocked(session, execution)
		if err := r.persistLocked(ctx, checkpointReasonResourceWait, session); err != nil {
			*session = previous
			r.mu.Unlock()
			return
		}
		r.armResourceWaitLocked(ctx, session.ID, execution.ID, *item)
		r.publishLocked(ctx, session, execution.ID, RuntimeEventWorkChanged)
		r.mu.Unlock()
		return
	}
	if item.Type == WorkItemTypeEmit {
		r.mu.Unlock()
		r.dispatchEmit(ctx, sessionID, executionID, itemID)
		return
	}
	prebound, hasPrebound := r.privateBindings[item.ID]
	item.State = WorkItemStateBinding
	item.UpdatedAt = r.now()
	r.touchExecutionLocked(session, execution)
	if err := r.persistLocked(ctx, checkpointReasonWorkBinding, session); err != nil {
		*session = previous
		r.mu.Unlock()
		return
	}
	bindCtx, cancelBinding := r.workContext(
		r.effectiveWorkDeadlineLocked(*execution, *item),
	)
	r.cancels[item.ID] = cancelBinding
	defer cancelBinding()
	request := WorkDispatchRequest{
		Session:   cloneSession(*session),
		Execution: cloneExecution(*execution),
		WorkItem:  *item,
	}
	portErr := r.validatePortPlansForDispatch(bindCtx, request)
	r.mu.Unlock()
	if portErr != nil {
		r.mu.Lock()
		session, execution, lookupErr := r.executionLocked(sessionID, executionID)
		if lookupErr == nil {
			current := mutableWorkItem(execution, itemID)
			if current != nil && current.State == WorkItemStateBinding {
				execution = r.settleUndispatchedWorkLocked(
					session,
					execution,
					current,
					WorkItemStateBlocked,
					portPlanError(
						"port_adapter_unavailable",
						portErr.Error(),
					),
					true,
				)
				if execution != nil {
					_ = r.commitTerminalMutationLocked(
						ctx,
						session,
						execution.ID,
						checkpointReasonPortBindingUnavailable,
						RuntimeEventWorkChanged,
					)
				}
			}
		}
		r.mu.Unlock()
		return
	}
	var (
		bound   BindResult
		bindErr error
	)
	if hasPrebound {
		bound = BindResult{SemanticBinding: prebound.SemanticBinding, Private: prebound.Private}
	} else {
		bound, bindErr = r.deps.Binder.BindWork(bindCtx, request)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	session, execution, err = r.executionLocked(sessionID, executionID)
	if err != nil {
		return
	}
	delete(r.cancels, itemID)
	item = mutableWorkItem(execution, itemID)
	if item == nil || item.State != WorkItemStateBinding {
		return
	}
	if bound.BoundToolCall != nil {
		if item.Input.Tool == nil {
			failure := &SanitizedError{
				Code:     "bound_tool_payload_missing",
				Category: ErrorCategoryExecution,
				Message:  "resolved tool binding has no tool work item payload",
			}
			execution = r.settleUndispatchedWorkLocked(
				session,
				execution,
				item,
				WorkItemStateBlocked,
				failure,
				true,
			)
			if execution != nil {
				_ = r.commitTerminalMutationLocked(
					ctx, session, execution.ID, checkpointReasonWorkBindingInvalid, RuntimeEventWorkChanged,
				)
			}
			return
		}
		item.Input.Tool.Call = jsonutil.CloneJSONInValue(*bound.BoundToolCall)
	}
	if !r.dispatchAllowedForWorkLocked(*execution, *item) {
		revertSnapshot := cloneSession(*session)
		item.State = WorkItemStateReady
		item.UpdatedAt = r.now()
		released := r.releaseWorkResourcesLocked(execution, item.ID)
		r.wakeResourceWaitersLocked(execution, released)
		r.touchExecutionLocked(session, execution)
		if err := r.persistLocked(ctx, checkpointReasonWorkBindingReverted, session); err != nil {
			*session = revertSnapshot
			return
		}
		r.publishLocked(ctx, session, execution.ID, RuntimeEventWorkChanged)
		return
	}
	if deadline := r.effectiveWorkDeadlineLocked(*execution, *item); deadline != nil &&
		!r.now().Before(deadline.UTC()) {
		r.finishWorkLocked(session, execution, item, nil, WorkExecutionResult{
			Error: deadlineError(), Confirmed: true,
		})
		execution = mutableSessionExecution(session, executionID)
		if execution != nil {
			_ = r.commitTerminalMutationLocked(
				ctx, session, execution.ID, checkpointReasonWorkDeadlineElapsed, RuntimeEventWorkChanged,
			)
		}
		return
	}
	if bindErr != nil {
		delete(r.privateBindings, item.ID)
		failure := ClassifyDomainError(
			bindErr, "runtime_binding_failed", ErrorCategoryExecution,
		)
		execution = r.settleUndispatchedWorkLocked(
			session,
			execution,
			item,
			WorkItemStateBlocked,
			failure,
			true,
		)
		if execution != nil {
			_ = r.commitTerminalMutationLocked(
				ctx, session, execution.ID, checkpointReasonWorkBlocked, RuntimeEventWorkChanged,
			)
		}
		return
	}
	if bound.ImmediateOutput != nil {
		delete(r.privateBindings, item.ID)
		item.Binding = &bound.SemanticBinding
		r.finishWorkLocked(
			session,
			execution,
			item,
			activeAttempt(execution, item.ID),
			WorkExecutionResult{
				Output:      bound.ImmediateOutput,
				Confirmed:   true,
				EffectState: bound.ImmediateEffectState,
			},
		)
		_ = r.commitTerminalMutationLocked(
			ctx,
			session,
			executionID,
			checkpointReasonWorkFinished,
			RuntimeEventWorkChanged,
		)
		return
	}
	if err := bound.SemanticBinding.Validate(); err != nil {
		r.finishWorkLocked(session, execution, item, nil, WorkExecutionResult{
			Error: &SanitizedError{
				Code: "invalid_semantic_binding", Category: ErrorCategoryValidation,
				Message: err.Error(),
			},
			Confirmed: true,
		})
		_ = r.commitTerminalMutationLocked(
			ctx,
			session,
			executionID,
			checkpointReasonWorkBindingInvalid,
			RuntimeEventWorkChanged,
		)
		return
	}
	if bound.Wait != nil {
		if item.Type == WorkItemTypeChildCall {
			if err := r.beginChildCallAttemptLocked(execution, item); err != nil {
				failure, found := budgetFailure(err)
				if !found {
					failure = &SanitizedError{
						Code:     budgetErrorCode,
						Category: ErrorCategoryLimit,
						Message:  err.Error(),
					}
				}
				r.blockWorkForBudgetLocked(
					ctx,
					session,
					execution,
					item,
					failure,
					checkpointReasonChildCallBudgetBlocked,
				)
				return
			}
		}
		effectiveDeadline := r.effectiveWorkDeadlineLocked(*execution, *item)
		bound.Wait.Deadline = earlierDeadline(
			bound.Wait.Deadline,
			effectiveDeadline,
		)
		now := r.now()
		wait := mutableWait(execution, item.WaitID)
		resumeAcceptedIntent := wait != nil &&
			wait.Status == WaitStatusPending &&
			wait.ResolutionIntent != nil
		if resumeAcceptedIntent {
			wait.Status = WaitStatusResolving
			wait.Request = *bound.Wait
			wait.UpdatedAt = now
		} else {
			value := Wait{
				ID:         WaitID(r.newID()),
				WorkItemID: item.ID,
				Status:     WaitStatusPending,
				Request:    *bound.Wait,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			execution.Waits = append(execution.Waits, value)
			wait = &execution.Waits[len(execution.Waits)-1]
		}
		item.State = WorkItemStateWaiting
		item.WaitID = wait.ID
		item.Binding = &bound.SemanticBinding
		item.UpdatedAt = now
		r.wakeResourceWaitersLocked(execution, nil)
		r.privateWaits[wait.ID] = bound.Private
		r.touchExecutionLocked(session, execution)
		if !resumeAcceptedIntent {
			r.appendWaitActivityLocked(session, execution, *wait)

			if err := r.resolveReadyWaitLocked(session, execution, wait.ID); err != nil {
				storedWait := mutableWait(execution, wait.ID)
				if storedWait != nil {
					storedWait.Status = WaitStatusInvalid
					storedWait.Error = &SanitizedError{
						Code:     "wait_replay_failed",
						Category: ErrorCategoryExecution,
						Message:  err.Error(),
					}
					storedWait.UpdatedAt = r.now()
				}
				item.WaitID = ""
				delete(r.privateWaits, wait.ID)
				r.finishWorkLocked(
					session,
					execution,
					item,
					nil,
					WorkExecutionResult{
						Error: &SanitizedError{
							Code:     "wait_replay_failed",
							Category: ErrorCategoryExecution,
							Message:  err.Error(),
						},
						Confirmed: true,
					},
				)
			}
		}

		if finalized, finalizeErr := r.finalizeExecutionSettlementLocked(
			ctx,
			session,
			executionID,
			nil,
		); finalizeErr == nil && finalized != nil {
			r.touchExecutionLocked(session, finalized)
		}
		if err := r.persistLocked(ctx, checkpointReasonWaitRequested, session); err != nil {
			return
		}
		r.armSessionExecutionTimersLocked(ctx, session)
		r.publishLocked(ctx, session, execution.ID, RuntimeEventWaitChanged)
		if resumeAcceptedIntent {
			r.launchAcceptedWaitResolution(ctx, session.ID, execution.ID, wait.ID)
		}
		return
	}
	previous = cloneSession(*session)
	if err := r.startDispatchedLocked(
		ctx,
		session,
		execution,
		item,
		BoundWork{SemanticBinding: bound.SemanticBinding, Private: bound.Private},
	); err != nil {
		*session = previous
		currentSession, currentExecution, lookupErr := r.executionLocked(
			sessionID,
			executionID,
		)
		if lookupErr != nil {
			return
		}
		currentItem := mutableWorkItem(currentExecution, itemID)
		if currentItem == nil || currentItem.State != WorkItemStateBinding {
			return
		}
		if failure, found := budgetFailure(err); found {
			r.blockWorkForBudgetLocked(
				ctx,
				currentSession,
				currentExecution,
				currentItem,
				failure,
				checkpointReasonWorkBudgetBlocked,
			)
			return
		}
		reverted := cloneSession(*currentSession)
		currentItem.State = WorkItemStateReady
		currentItem.UpdatedAt = r.now()
		r.touchExecutionLocked(currentSession, currentExecution)
		if persistErr := r.persistLocked(
			ctx,
			checkpointReasonWorkDispatchReverted,
			currentSession,
		); persistErr != nil {
			*currentSession = reverted
			return
		}
		r.publishLocked(ctx, currentSession, currentExecution.ID, RuntimeEventWorkChanged)
		return
	}
	r.publishLocked(ctx, session, execution.ID, RuntimeEventWorkChanged)
	r.launchAsync(
		"session-v3 WorkItem execution",
		func() {
			r.executeOne(
				r.lifecycleCtx,
				sessionID,
				executionID,
				item.ID,
				request,
				BoundWork{SemanticBinding: bound.SemanticBinding, Private: bound.Private},
			)
		},
		func(failure backgroundutil.PanicFailureError) {
			r.finishPanickedWork(sessionID, executionID, item.ID, failure)
		},
	)
}

func (r *Runtime) reserveWorkStepLocked(
	ctx context.Context,
	session *Session,
	execution *Execution,
	item *WorkItem,
) bool {
	if item == nil || item.StepExecutionReserved {
		return item != nil
	}
	if err := r.reserveStepExecutionLocked(execution, item); err != nil {
		failure, found := budgetFailure(err)
		if !found {
			failure = &SanitizedError{
				Code:     budgetErrorCode,
				Category: ErrorCategoryLimit,
				Message:  err.Error(),
			}
		}
		r.blockWorkForBudgetLocked(
			ctx,
			session,
			execution,
			item,
			failure,
			checkpointReasonWorkStepBudgetBlocked,
		)
		return false
	}
	item.StepExecutionReserved = true
	return true
}

func (r *Runtime) blockWorkForBudgetLocked(
	ctx context.Context,
	session *Session,
	execution *Execution,
	item *WorkItem,
	failure *SanitizedError,
	reason CheckpointReason,
) {
	if session == nil || execution == nil || item == nil || failure == nil {
		return
	}

	execution = r.settleUndispatchedWorkLocked(
		session,
		execution,
		item,
		WorkItemStateBlocked,
		failure,
		true,
	)
	if execution == nil {
		return
	}
	_ = r.commitTerminalMutationLocked(
		ctx,
		session,
		execution.ID,
		reason,
		RuntimeEventWorkChanged,
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

	attemptDeadlineValue, err := r.effectiveAttemptDeadlineLocked(
		*execution,
		*item,
		now,
	)
	if err != nil {
		return err
	}
	if deadline := r.effectiveWorkDeadlineLocked(*execution, *item); deadline != nil &&
		!now.Before(deadline.UTC()) {
		return errors.New("work deadline elapsed before dispatch")
	}
	if err := r.reserveCallAttemptLocked(execution, item); err != nil {
		//nolint:errorlint // We are taking error val.
		if failure, ok := err.(*SanitizedError); ok {
			return failure
		}
		return err
	}

	attempt := Attempt{
		ID:             AttemptID(r.newID()),
		WorkItemID:     item.ID,
		State:          AttemptStateDispatched,
		Number:         nextAttempt(*execution, item.ID),
		IdempotencyKey: item.Effect.IdempotencyKey,
		EffectState:    effectStateForDispatch(item.Effect),
		StartedAt:      now,
		Deadline:       attemptDeadlineValue,
	}
	item.State = WorkItemStateDispatched
	item.EffectState = effectStateForDispatch(item.Effect)
	item.RetryState = nil
	item.Binding = &bound.SemanticBinding
	item.AttemptIDs = append(item.AttemptIDs, attempt.ID)
	item.DispatchedAt = &now
	item.UpdatedAt = now
	execution.Attempts = append(execution.Attempts, attempt)
	r.privateBindings[item.ID] = bound
	r.touchExecutionLocked(session, execution)
	return r.persistLocked(ctx, checkpointReasonWorkDispatched, session)
}

// beginChildCallAttemptLocked creates the durable outer call attempt before
// the WorkItem waits for its child frame. The child frame owns native domain
// execution. This boundary records orchestration effect facts only.
func (r *Runtime) beginChildCallAttemptLocked(
	execution *Execution,
	item *WorkItem,
) error {
	if execution == nil ||
		item == nil ||
		item.Type != WorkItemTypeChildCall ||
		activeAttempt(execution, item.ID) != nil {
		return nil
	}
	if err := r.reserveCallAttemptLocked(execution, item); err != nil {
		return err
	}

	now := r.now()
	attempt := Attempt{
		ID:             AttemptID(r.newID()),
		WorkItemID:     item.ID,
		State:          AttemptStateDispatched,
		Number:         nextAttempt(*execution, item.ID),
		IdempotencyKey: item.Effect.IdempotencyKey,
		EffectState:    effectStateForDispatch(item.Effect),
		StartedAt:      now,
		Deadline:       cloneTime(item.Deadline),
	}
	item.AttemptIDs = append(item.AttemptIDs, attempt.ID)
	item.EffectState = effectStateForDispatch(item.Effect)
	item.DispatchedAt = &now
	item.UpdatedAt = now
	execution.Attempts = append(execution.Attempts, attempt)
	return nil
}

func (r *Runtime) executeOne(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	itemID WorkItemID,
	_req WorkDispatchRequest,
	bound BoundWork,
) {
	executionCtx, cancel := context.WithCancel(r.lifecycleCtx)
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		cancel()
		return
	}
	session, execution, err := r.executionLocked(sessionID, executionID)
	if err != nil {
		r.mu.Unlock()
		cancel()
		return
	}
	item := mutableWorkItem(execution, itemID)
	cleanupDispatch := item != nil &&
		item.Cleanup &&
		r.cleanupDispatchAllowedLocked(*execution, *item)
	if item != nil &&
		item.State == WorkItemStateDispatched &&
		((execution.Controls.CancellationRequested && !cleanupDispatch) ||
			execution.State.Terminal() ||
			(!item.Cleanup &&
				item.Deadline != nil &&
				!r.now().Before(*item.Deadline))) {
		r.settleUnstartedAttemptLocked(ctx, session, execution, item)
		r.mu.Unlock()
		cancel()
		return
	}
	if item == nil || item.State != WorkItemStateDispatched ||
		(execution.Controls.CancellationRequested && !cleanupDispatch) ||
		execution.State.Terminal() {
		r.mu.Unlock()
		cancel()
		return
	}
	attempt := activeAttempt(execution, itemID)
	dispatchDeadline := r.effectiveWorkDeadlineLocked(*execution, *item)
	if attempt != nil {
		dispatchDeadline = earlierDeadline(
			dispatchDeadline,
			attempt.Deadline,
		)
	}
	if dispatchDeadline != nil {
		cancel()
		executionCtx, cancel = context.WithDeadline(
			r.lifecycleCtx,
			dispatchDeadline.UTC(),
		)
	}
	r.cancels[itemID] = cancel
	request := WorkDispatchRequest{
		Session:   cloneSession(*session),
		Execution: cloneExecution(*execution),
		WorkItem:  *item,
	}
	request.PortStream = WorkPortStream{
		Emit: func(name string, value jsonutil.CanonicalValue) error {
			return r.appendWorkPortStream(
				ctx,
				sessionID,
				executionID,
				itemID,
				name,
				value,
			)
		},
	}
	if item.Type == WorkItemTypeLLM {
		request.Stream = WorkStream{
			OnText: func(value string) error {
				return r.appendLLMStream(
					ctx,
					sessionID,
					executionID,
					itemID,
					value,
					false,
				)
			},
			OnThinking: func(value string) error {
				return r.appendLLMStream(
					ctx,
					sessionID,
					executionID,
					itemID,
					value,
					true,
				)
			},
		}
	}
	r.mu.Unlock()
	result, err := r.deps.Binder.ExecuteWork(executionCtx, request, bound)
	cancel()
	if err != nil && result.Error == nil {
		result.Error = ClassifyDomainError(
			err,
			"external_execution_failed",
			ErrorCategoryExecution,
		)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}

	session, execution, err = r.executionLocked(sessionID, executionID)
	if err != nil {
		return
	}
	item = mutableWorkItem(execution, itemID)
	if item == nil || item.State != WorkItemStateDispatched {
		return
	}

	delete(r.cancels, itemID)
	delete(r.privateBindings, itemID)
	r.finishWorkLocked(session, execution, item, activeAttempt(execution, itemID), result)
	_ = r.commitTerminalMutationLocked(
		ctx,
		session,
		executionID,
		checkpointReasonWorkFinished,
		RuntimeEventWorkChanged,
	)
}

func (r *Runtime) finishWorkLocked(
	session *Session,
	execution *Execution,
	item *WorkItem,
	att *Attempt,
	result WorkExecutionResult,
) {
	executionID := execution.ID
	workItemID := item.ID
	result = mergeLLMStreamResult(*item, att, result)
	result = normalizeWorkResult(*item, result)
	preserveUsableNormalLLMError := normalConversationLLMOutputUsable(
		session,
		*execution,
		*item,
		result,
	)

	now := r.now()
	item.Output = result.Output
	item.Error = result.Error
	item.CompletedAt = &now
	item.UpdatedAt = now

	if item.Type == WorkItemTypeLLM {
		if err := r.consumeLLMUsageLocked(
			execution,
			item,
			item.Output,
		); err != nil {
			if failure, found := budgetFailure(err); found {
				result.Error = failure
			} else {
				result.Error = &SanitizedError{
					Code:     "model_usage_invalid",
					Category: ErrorCategoryValidation,
					Message:  err.Error(),
				}
			}
			item.Error = result.Error
			preserveUsableNormalLLMError = false
		}
	}
	if result.Error == nil &&
		result.Confirmed &&
		item.Output != nil {
		if err := r.deliverWorkPortsLocked(session, execution, item); err != nil {
			result.Error = portPlanError("port_delivery_failed", err.Error())
			item.Error = result.Error
		}
	}
	execution = mutableSessionExecution(session, executionID)
	if execution == nil {
		return
	}
	item = mutableWorkItem(execution, workItemID)
	if item == nil {
		return
	}
	attempt := activeAttempt(execution, workItemID)

	if execution.Controls.CancellationRequested &&
		!item.Cleanup &&
		(result.Confirmed || item.Effect.Kind == EffectKindRead) {
		item.State = WorkItemStateCancelled
		if attempt != nil {
			attempt.State = AttemptStateCancelled
			attempt.FinishedAt = &now
			attempt.Error = result.Error
			if result.Output != nil {
				attempt.Receipt = cloneReconciliationReceipt(result.Output.Receipt)
			}
		}
		recordEffectSettlement(item, attempt, result)
		if item.EffectState == EffectStateConfirmedApplied {
			if err := r.recordConfirmedEffectLocked(execution, item); err != nil {
				execution.Cleanup.Primary = appendCleanupCause(
					execution.Cleanup.Primary,
					SanitizedError{
						Code:     "effect_record_failed",
						Category: ErrorCategoryExecution,
						Message:  err.Error(),
					},
				)
			}
		}
		released := r.releaseWorkResourcesLocked(execution, item.ID)
		r.wakeResourceWaitersLocked(execution, released)
		r.cancelResourceWaitLocked(item.ID)
		r.advanceSettledWorkLocked(
			session,
			execution,
			item.FrameID,
			item.ID,
		)
		execution = mutableSessionExecution(session, executionID)
		if execution == nil {
			return
		}
		item = mutableWorkItem(execution, workItemID)
		if item == nil {
			return
		}
		r.wakeCompletionWorkWaitsLocked(session, execution, *item)
		execution, err := r.finalizeExecutionSettlementLocked(
			context.Background(),
			session,
			executionID,
			item.Error,
		)
		if err != nil || execution == nil {
			return
		}
		r.touchExecutionLocked(session, execution)
		return
	}

	if attempt != nil {
		attempt.FinishedAt = &now
		attempt.Error = result.Error
		attempt.Deadline = cloneTime(attempt.Deadline)
		if result.Output != nil {
			attempt.Receipt = result.Output.Receipt
		}
	}
	switch {
	case result.Confirmed &&
		(result.Error == nil || preserveUsableNormalLLMError):
		if workflowMutatingToolObservationIsUncertain(
			session,
			*execution,
			*item,
			result,
		) {
			item.State = WorkItemStateUncertain
			if attempt != nil {
				attempt.State = AttemptStateUncertain
			}
		} else {
			item.State = WorkItemStateCompleted
			if attempt != nil {
				attempt.State = AttemptStateConfirmed
			}
		}
		if attempt != nil {
			attempt.FinishedAt = &now
		}
	case result.Confirmed:
		item.State = WorkItemStateFailed
		if attempt != nil {
			attempt.State = AttemptStateFailed
		}
	case item.Effect.Kind == EffectKindRead:
		item.State = WorkItemStateFailed
		if attempt != nil {
			attempt.State = AttemptStateFailed
		}
		if item.Error == nil {
			item.Error = &SanitizedError{
				Code:    "confirmed_work_failed",
				Message: "a read-only work item completed without a confirmed output",
			}
		}
	case result.Error != nil &&
		result.Error.Category == ErrorCategoryUncertainty:
		item.State = WorkItemStateUncertain
		if attempt != nil {
			attempt.State = AttemptStateUncertain
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

	recordEffectSettlement(item, attempt, result)
	if err := r.recordConfirmedEffectLocked(execution, item); err != nil {
		item.State = WorkItemStateFailed
		item.Error = portPlanError("effect_record_failed", err.Error())
	}

	if item.State.Terminal() {
		if item.RetainClaimsForFrame &&
			item.State == WorkItemStateCompleted {
			owner := item.FrameID
			if item.RetainClaimsForFrameID != "" {
				owner = item.RetainClaimsForFrameID
			}
			r.transferWorkResourcesToFrameLocked(
				execution,
				item.ID,
				owner,
			)
		} else {
			released := r.releaseWorkResourcesLocked(execution, item.ID)
			r.wakeResourceWaitersLocked(execution, released)
		}
		r.cancelResourceWaitLocked(item.ID)
	}

	if item.State == WorkItemStateUncertain &&
		r.reconciliationRequiredLocked(execution, item) {
		item.State = WorkItemStateReconciling
		item.CompletedAt = nil
		item.UpdatedAt = now
		r.advanceSettledWorkLocked(
			session,
			execution,
			item.FrameID,
			item.ID,
		)
	}

	if item.State == WorkItemStateFailed &&
		r.prepareRetryLocked(execution, item, attempt, now) {
		r.refreshExecutionStateLocked(execution)
		r.touchExecutionLocked(session, execution)
		return
	}

	if item.Output != nil && item.Output.App != nil {
		r.recordAppInstanceLocked(execution, item)
	}

	if item.Type == WorkItemTypeTool &&
		item.Output != nil &&
		item.Output.Tool != nil &&
		r.reserveActivityBudgetLocked(execution, Activity{
			ExecutionID: execution.ID, FrameID: item.FrameID, WorkItemID: item.ID,
		}) == nil {
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
					ActionKind: ActionKindTool,
					Outcome:    string(item.Output.Tool.Kind),
					Output:     item.Output.Tool.Structured.Clone(),
					Error:      item.Error,
					Receipt:    item.Output.Receipt,
				},
			},
		)
	}
	if item.Type == WorkItemTypeLLM &&
		item.State != WorkItemStateCompleted {
		frame := mutableFrame(execution, item.FrameID)
		if frame != nil && frame.Status.Terminal() {
			r.appendInterruptedAssistantActivityLocked(
				session,
				execution,
				*item,
				attempt,
				now,
			)
		}
	}
	if item.State == WorkItemStateUncertain {
		if !execution.Controls.CancellationRequested {
			if err := r.recordUncertainWorkflowStepOutcomeLocked(
				execution,
				item,
			); err != nil {
				item.State = WorkItemStateFailed
				item.Error = &SanitizedError{
					Code:     "workflow_outcome_record_failed",
					Category: ErrorCategoryExecution,
					Message:  err.Error(),
				}
				r.advanceSettledWorkLocked(
					session,
					execution,
					item.FrameID,
					item.ID,
				)
			} else {
				r.addRecoveryWaitLocked(session, execution, item)
			}
		}
	} else {
		r.advanceSettledWorkLocked(
			session,
			execution,
			item.FrameID,
			item.ID,
		)
	}
	execution = mutableSessionExecution(session, executionID)
	if execution == nil {
		return
	}
	item = mutableWorkItem(execution, workItemID)
	if item == nil {
		return
	}
	r.wakeCompletionWorkWaitsLocked(session, execution, *item)
	execution, err := r.finalizeExecutionSettlementLocked(
		context.Background(),
		session,
		executionID,
		item.Error,
	)
	if err != nil || execution == nil {
		return
	}
	r.touchExecutionLocked(session, execution)
}

// resumeSagaCompensationFramesLocked wakes a pending Saga after any relevant
// WorkItem settles. This includes late confirmations after cancellation and
// ordinary effects that finish after the Saga body has already reached its
// primary failure boundary.
func workflowMutatingToolObservationIsUncertain(
	session *Session,
	execution Execution,
	item WorkItem,
	result WorkExecutionResult,
) bool {
	if item.Type != WorkItemTypeTool ||
		item.Effect.Kind == EffectKindRead ||
		result.Output == nil ||
		result.Output.Tool == nil ||
		result.Output.Tool.Kind == ToolOutputKindSuccess ||
		result.EffectState == nil ||
		*result.EffectState != EffectStateUncertain {
		return false
	}
	return !conversationToolObservation(session, execution, item)
}

func conversationToolObservation(
	session *Session,
	execution Execution,
	item WorkItem,
) bool {
	if item.Input.Tool == nil {
		return false
	}
	return normalConversationAgentLoopFrame(session, execution, item.FrameID)
}

func (r *Runtime) resumeSagaCompensationFramesLocked(
	session *Session,
	execution *Execution,
) {
	if session == nil || execution == nil {
		return
	}

	frameIDs := make([]ProgramFrameID, 0)
	for _, frame := range execution.ProgramFrames {
		if frame.Status.Terminal() ||
			frame.Program.InterpreterID != WorkflowInterpreterID {
			continue
		}
		_, state, err := decodeWorkflowProgram(frame)
		if err != nil ||
			state.Pending == nil ||
			state.Pending.Kind != WorkflowPendingSaga ||
			state.Pending.Saga == nil ||
			state.Pending.Saga.PrimaryStatus == "" {
			continue
		}
		frameIDs = append(frameIDs, frame.ID)
	}

	for _, frameID := range frameIDs {
		current := mutableSessionExecution(session, execution.ID)
		if current == nil {
			return
		}
		_ = r.advanceFrameLocked(
			session,
			current,
			frameID,
			ProgramEvent{Kind: ProgramEventContinue},
		)
	}
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
	if r.reserveActivityBudgetLocked(execution, session.Timeline[len(session.Timeline)-1]) != nil {
		session.Timeline = session.Timeline[:len(session.Timeline)-1]
	}
}

func (r *Runtime) addRecoveryWaitLocked(session *Session, execution *Execution, item *WorkItem) {
	now := r.now()
	wait := Wait{
		ID:         WaitID(r.newID()),
		WorkItemID: item.ID,
		Status:     WaitStatusPending,
		Request: WaitRequest{
			Kind: WaitKindRecovery,
			Summary: jsonutil.MustCanonicalValue(
				map[string]string{"message": "external work item outcome requires recovery review"},
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

func (r *Runtime) scheduleWaitDeadlineLocked(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	wait Wait,
) {
	wakeAt := waitWakeAt(wait)
	if wakeAt == nil || wait.ID == "" || wait.Status != WaitStatusPending {
		return
	}
	r.cancelWaitDeadlineLocked(wait.ID)
	waitCtx, armed := r.armDueTimeLocked(waitDueTimeKey(wait.ID))
	if !armed {
		return
	}
	deadline := wakeAt.UTC()
	r.launchAsync(
		"session-v3 wait deadline",
		func() {
			r.expireAt(waitCtx, sessionID, executionID, wait.ID, deadline)
		},
		nil,
	)
}

func (r *Runtime) cancelWaitDeadlineLocked(waitID WaitID) {
	r.cancelDueTimeLocked(waitDueTimeKey(waitID))
}

func (r *Runtime) expireAt(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	waitID WaitID,
	deadline time.Time,
) {
	delay := deadline.Sub(r.now())
	timer := time.NewTimer(max(delay, 0))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	session, execution, err := r.executionLocked(sessionID, executionID)
	if err != nil {
		return
	}
	if r.closed {
		return
	}
	r.consumeDueTimeLocked(waitDueTimeKey(waitID))
	wait := mutableWait(execution, waitID)
	if wait == nil || wait.Status != WaitStatusPending {
		return
	}
	now := r.now()
	if r.settleTimerLocked(ctx, session, execution, wait, now) {
		return
	}
	if wait.Request.Deadline == nil || now.Before(*wait.Request.Deadline) {
		return
	}
	wait.Status = WaitStatusExpired
	wait.Error = deadlineError()
	wait.UpdatedAt = now
	item := mutableWorkItem(execution, wait.WorkItemID)
	delete(r.privateWaits, wait.ID)
	if item != nil && !item.State.Terminal() {
		item.WaitID = ""
		r.finishWorkLocked(
			session,
			execution,
			item,
			activeAttempt(execution, item.ID),
			WorkExecutionResult{
				Error:     wait.Error,
				Confirmed: true,
			},
		)
	}
	_ = r.commitTerminalMutationLocked(
		context.WithoutCancel(ctx),
		session,
		executionID,
		checkpointReasonWaitExpired,
		RuntimeEventWaitChanged,
	)
}

func (r *Runtime) finishPanickedWork(
	sessionID SessionID,
	executionID ExecutionID,
	itemID WorkItemID,
	failure backgroundutil.PanicFailureError,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	session, execution, err := r.executionLocked(sessionID, executionID)
	if err != nil {
		return
	}
	item := mutableWorkItem(execution, itemID)
	if item == nil || item.State.Terminal() {
		return
	}

	result := WorkExecutionResult{
		Error: &SanitizedError{
			Code:    backgroundutil.PanicFailureCode,
			Message: failure.Error(),
		},
	}
	switch item.State {
	case WorkItemStateBinding:
		execution = r.settleUndispatchedWorkLocked(
			session,
			execution,
			item,
			WorkItemStateBlocked,
			result.Error,
			true,
		)
		if execution == nil {
			return
		}
	case WorkItemStateDispatched:
		r.finishWorkLocked(
			session,
			execution,
			item,
			activeAttempt(execution, item.ID),
			result,
		)
	default:
		return
	}
	execution = mutableSessionExecution(session, executionID)
	if execution == nil {
		return
	}
	item = mutableWorkItem(execution, itemID)
	if item == nil {
		return
	}
	r.wakeCompletionWorkWaitsLocked(session, execution, *item)
	execution = mutableSessionExecution(session, executionID)
	if execution == nil {
		return
	}
	delete(r.privateBindings, item.ID)
	if cancel := r.cancels[item.ID]; cancel != nil {
		cancel()
		delete(r.cancels, item.ID)
	}
	if !r.commitTerminalMutationLocked(
		context.Background(),
		session,
		execution.ID,
		checkpointReasonWorkPanicked,
		RuntimeEventWorkChanged,
	) {
		return
	}
}

func (r *Runtime) readyWorkLocked(execution Execution) []WorkItemID {
	if !r.executionDispatchAvailableLocked(execution) {
		return nil
	}

	values := []WorkItemID{}
	for _, item := range execution.WorkItems {
		if item.State == WorkItemStateReady && (item.Automatic || item.Requested) &&
			!unresolvedDependencies(execution, item) &&
			r.dispatchAllowedForWorkLocked(execution, item) {
			if item.Type == WorkItemTypeTool &&
				!normalToolDispatchAllowed(execution, item) {
				continue
			}
			values = append(values, item.ID)
		}
	}
	return values
}

func (r *Runtime) dispatchAllowedForWorkLocked(
	execution Execution,
	item WorkItem,
) bool {
	if !r.executionDispatchAvailableLocked(execution) {
		return false
	}

	if !execution.Controls.CancellationRequested {
		return true
	}
	return r.cleanupDispatchAllowedLocked(execution, item)
}

func (r *Runtime) refreshExecutionStateLocked(execution *Execution) {
	if execution == nil || execution.State.Terminal() {
		return
	}

	if outcome := r.frameOutcomeLocked(execution); outcome != nil {
		session := r.sessions[execution.SessionID]
		if session != nil && executionTerminalReadyLocked(session, *execution) {
			execution.Outcome = outcome
			switch outcome.Kind {
			case ExecutionOutcomeCancelled:
				execution.State = ExecutionStateCancelled
			case ExecutionOutcomeFailed:
				execution.State = ExecutionStateFailed
			default:
				execution.State = ExecutionStateCompleted
			}
			return
		}
	}

	if execution.Controls.CancellationRequested {
		if hasOutstandingCleanup(*execution) {
			execution.State = ExecutionStateRunning
			return
		}
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

	runnable := false
	for _, item := range execution.WorkItems {
		switch item.State {
		case WorkItemStateBinding, WorkItemStateDispatched:
			execution.State = ExecutionStateRunning
			return
		case WorkItemStateRetryWaiting:
			execution.State = ExecutionStateRunning
			return
		case WorkItemStateReady:
			if item.Automatic || item.Requested {
				runnable = true
			}
		case WorkItemStateResourceWaiting:
			execution.State = ExecutionStateWaiting
			return
		default:
		}
	}
	if runnable {
		execution.State = ExecutionStateRunning
		return
	}
	execution.State = ExecutionStateWaiting
}

func (r *Runtime) frameOutcomeLocked(execution *Execution) *ExecutionOutcome {
	if execution == nil {
		return nil
	}

	rootCount := 0
	allTerminal := true
	failed := false
	cancelled := false
	var terminalError *SanitizedError

	for _, frame := range execution.ProgramFrames {
		if frame.ParentFrameID != "" {
			continue
		}

		// A conversation frame is intentionally long-lived. It waits for the
		// next user Activity after every completed child Agent loop.
		if frame.Program.InterpreterID == BuiltinConversationInterpreterID {
			return nil
		}

		rootCount++
		if !frame.Status.Terminal() {
			allTerminal = false
			continue
		}
		switch frame.Status {
		case ProgramFrameStateFailed:
			failed = true
			if terminalError == nil {
				terminalError = cloneWorkflowSanitizedError(frame.Error)
			}
		case ProgramFrameStateCancelled:
			cancelled = true
			if terminalError == nil {
				terminalError = cloneWorkflowSanitizedError(frame.Error)
			}
		default:
		}
	}

	if rootCount == 0 || !allTerminal {
		return nil
	}
	if execution.Controls.CancellationRequested {
		cancelled = true
	}

	outcome := &ExecutionOutcome{
		Kind:        ExecutionOutcomeCompleted,
		CompletedAt: r.now(),
	}
	if cancelled {
		outcome.Kind = ExecutionOutcomeCancelled
		outcome.Code = workflowProgramFrameCancelledCode
		outcome.Message = workflowProgramFrameCancelMessage
		if execution.Cleanup.Primary != nil {
			outcome.Error = cloneWorkflowSanitizedError(execution.Cleanup.Primary)
			outcome.Code = firstText(outcome.Error.Code, outcome.Code)
			outcome.Message = firstText(outcome.Error.Message, outcome.Message)
		} else if terminalError != nil {
			outcome.Error = terminalError
		}
	} else if failed {
		outcome.Kind = ExecutionOutcomeFailed
		outcome.Code = workflowProgramFrameFailedCode
		outcome.Message = workflowProgramFrameFailedMessage
		if terminalError != nil {
			outcome.Error = terminalError
			outcome.Code = firstText(
				terminalError.Code,
				workflowProgramFrameFailedCode,
			)
			outcome.Message = firstText(
				terminalError.Message,
				workflowProgramFrameFailedMessage,
			)
		}
	}
	return outcome
}

func (r *Runtime) completeExecutionLocked(
	ctx context.Context,
	session *Session,
	execution *Execution,
	outcome ExecutionOutcome,
) {
	if !executionTerminalReadyLocked(session, *execution) {
		return
	}
	if outcome.Kind != ExecutionOutcomeCompleted && outcome.Error == nil &&
		execution.Cleanup.Primary != nil {
		outcome.Error = cloneWorkflowSanitizedError(execution.Cleanup.Primary)
		outcome.Code = firstText(outcome.Error.Code, outcome.Code)
		outcome.Message = firstText(outcome.Error.Message, outcome.Message)
	}
	if outcome.Error != nil {
		outcome.Error = cloneWorkflowSanitizedError(outcome.Error)
		if outcome.Code == "" {
			outcome.Code = outcome.Error.Code
		}
		if outcome.Message == "" {
			outcome.Message = outcome.Error.Message
		}
	}
	execution.Outcome = &outcome
	switch outcome.Kind {
	case ExecutionOutcomeCancelled:
		execution.State = ExecutionStateCancelled
	case ExecutionOutcomeFailed:
		execution.State = ExecutionStateFailed
	default:
		execution.State = ExecutionStateCompleted
	}
	r.settleTerminalExecutionLocked(session, execution)
	execution = mutableSessionExecution(session, execution.ID)
	if execution == nil {
		return
	}
	r.touchExecutionLocked(session, execution)
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

func (r *Runtime) persistLocked(ctx context.Context, reason CheckpointReason, session *Session) (err error) {
	if session == nil {
		return errors.New("cannot checkpoint a nil session")
	}

	// A prior terminal settlement can free a budget-concurrency slot without
	// releasing a named resource. Wake those durable waiters before building
	// the checkpoint candidate. The resulting Ready state is part of the same
	// authoritative committed Session mutation.
	for index := range session.Executions {
		execution := &session.Executions[index]
		if r.wakeResourceWaitersLocked(execution, nil) {
			r.touchExecutionLocked(session, execution)
		}
	}

	defer func() {
		if err != nil {
			r.checkpointPending[session.ID] = true
		} else {
			delete(r.checkpointPending, session.ID)
		}
	}()

	// Build and validate a durable candidate first. A failed checkpoint must
	// not mutate the live sequence, otherwise a retry would publish a gap.
	snapshot := cloneSession(*session)
	snapshot.EventSequence++

	if err := snapshot.Validate(); err != nil {
		return err
	}
	for _, execution := range snapshot.Executions {
		for _, frame := range execution.ProgramFrames {
			interpreter := r.interpreterLocked(frame.Program)
			if interpreter == nil {
				return fmt.Errorf("interpreter %q unavailable", frame.Program.InterpreterID)
			}
			if frame.Status.Terminal() {
				continue
			}
			if err := interpreter.ValidateState(frame.State); err != nil {
				return err
			}
		}
	}
	checkpointErr := r.deps.Checkpoints.HandleCheckpoint(
		context.WithoutCancel(ctx),
		CheckpointView{Reason: reason, Session: snapshot, CreatedAt: r.now()},
	)
	if checkpointErr != nil {
		var committed CheckpointCommitStatus
		if !errors.As(checkpointErr, &committed) ||
			committed == nil ||
			!committed.CheckpointCommitted() {
			return checkpointErr
		}
		if errors.As(checkpointErr, &committed) &&
			committed != nil &&
			committed.CheckpointCommitted() {
			session.EventSequence = snapshot.EventSequence
		}
	} else {
		session.EventSequence = snapshot.EventSequence
	}

	// A prior failed checkpoint must not suppress ready-work selection after
	// this checkpoint has committed. Clear the barrier before the common
	// post-commit dispatcher computes readiness.
	delete(r.checkpointPending, session.ID)

	// All normal command, settlement, retry, wait, resource, timer, trigger,
	// and port paths pass through persistLocked. This is the only normal
	// post-checkpoint ready-work launch boundary.
	r.scheduleCommittedFrameClosuresLocked(session)
	if !r.closed && !r.restoring {
		r.launchReadyWork(
			context.WithoutCancel(ctx),
			session.ID,
			r.readyWorkForSessionLocked(session),
		)
	}
	return nil
}

func (r *Runtime) publishLocked(ctx context.Context, session *Session, executionID ExecutionID, kind RuntimeEventKind) {
	if r.deps.Events == nil || r.checkpointPending[session.ID] {
		return
	}
	event := RuntimeEvent{
		SessionID:   session.ID,
		ExecutionID: executionID,
		Sequence:    session.EventSequence,
		Kind:        kind,
		At:          r.now(),
	}
	r.launchAsync("session-v3 event publication", func() {
		_ = r.deps.Events.PublishRuntimeEvent(r.lifecycleCtx, event)
	}, nil)
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
		return nil, false, errors.New(errCommandIDRequired)
	}
	if session == nil {
		return nil, false, errors.New("session is unavailable")
	}
	return commandReceiptFromRecords(session.Commands, meta, payload)
}

func commandReceiptFromRecords(
	records []CommandRecord,
	meta CommandMeta,
	payload any,
) (*CommandReceipt, bool, error) {
	if meta.ID == "" {
		return nil, false, errors.New(errCommandIDRequired)
	}
	fingerprint, err := fingerprintCommand(commandFingerprintPayload(payload))
	if err != nil {
		return nil, false, err
	}
	for _, record := range records {
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
	fingerprint, err := fingerprintCommand(
		commandFingerprintPayload(payload),
	)
	if err != nil {
		panic(err)
	}
	session.Commands = append(
		session.Commands,
		CommandRecord{ID: meta.ID, Fingerprint: fingerprint, Receipt: receipt, RecordedAt: r.now()},
	)
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

func commandFingerprintPayload(value any) any {
	switch command := value.(type) {
	case StartWorkflowCommand:
		if command.CommandIdentity != nil {
			return *command.CommandIdentity
		}
	case RegisterWorkflowTriggersCommand:
		if command.CommandIdentity != nil {
			return *command.CommandIdentity
		}
	}
	return value
}

func fingerprintCommand(value any) (string, error) {
	raw, err := jsonutil.MarshalCanonicalObject(
		value,
		maximumCommandFingerprintBytes,
	)
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
		"session revision mismatch: expected %d, current %d",
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
		"execution revision mismatch: expected %d, current %d",
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
			if item.Type == WorkItemTypeFork &&
				dependency != nil &&
				dependency.Type == WorkItemTypeChildCall &&
				childCallForkReservationReady(execution, *dependency) {
				continue
			}
			return true
		}
	}
	return false
}

// childCallForkReservationReady permits the Fork that materializes a child
// frame only after the owning ChildCall WorkItem has persisted its outer
// Attempt and durable completion wait. A child cannot start native work before
// its parent logical operation owns budget and effect accounting.
func childCallForkReservationReady(
	execution Execution,
	item WorkItem,
) bool {
	if item.Type != WorkItemTypeChildCall ||
		item.State != WorkItemStateWaiting {
		return false
	}
	for _, attempt := range execution.Attempts {
		if attempt.WorkItemID == item.ID &&
			attempt.State == AttemptStateDispatched {
			return true
		}
	}
	return false
}

func childCallForkDependencies(
	frames []ProgramFrameIntent,
	work []WorkItemIntent,
) []WorkItemID {
	frameIDs := make(map[ProgramFrameID]struct{}, len(frames))
	for _, frame := range frames {
		if frame.ID != "" {
			frameIDs[frame.ID] = struct{}{}
		}
	}

	output := make([]WorkItemID, 0)
	seen := make(map[WorkItemID]struct{})
	for _, intent := range work {
		if intent.ID == "" || intent.Type != WorkItemTypeChildCall ||
			intent.Input.ChildCall == nil {
			continue
		}
		if _, found := frameIDs[intent.Input.ChildCall.Completion.FrameID]; !found {
			continue
		}
		if _, duplicate := seen[intent.ID]; duplicate {
			continue
		}
		seen[intent.ID] = struct{}{}
		output = append(output, intent.ID)
	}
	return output
}

func containsResolution(values []WaitResolution, want WaitResolution) bool {
	return slices.Contains(values, want)
}

func cloneBudget(value *ExecutionBudget) *ExecutionBudget {
	if value == nil {
		return nil
	}
	output := cloneExecutionBudget(*value)
	return &output
}

func cloneEvaluationBoundary(
	value *EvaluationBoundary,
) *EvaluationBoundary {
	if value == nil {
		return nil
	}
	output := *value
	output.Messages = value.Messages.Clone()
	return &output
}

func cloneEffectPlan(value *WorkflowEffectPlan) *WorkflowEffectPlan {
	if value == nil {
		return nil
	}
	output := &WorkflowEffectPlan{
		Reconcile:  cloneWorkflowBlock(value.Reconcile),
		Compensate: cloneWorkflowBlock(value.Compensate),
	}
	return output
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
		return fmt.Errorf("duplicate restored session %q", value.ID)
	}
	copyValue := cloneSession(value)
	for _, execution := range copyValue.Executions {
		for _, frame := range execution.ProgramFrames {
			interpreter := r.runtime.interpreterLocked(frame.Program)
			if interpreter == nil {
				return fmt.Errorf("interpreter %q unavailable", frame.Program.InterpreterID)
			}
			if err := interpreter.ValidateSnapshot(frame.Program); err != nil {
				return err
			}
		}
	}
	for _, trigger := range copyValue.Triggers {
		if err := validateTriggerRegistrationProgram(trigger); err != nil {
			return err
		}
	}
	changed, recoverErr := r.runtime.recoverLocked(&copyValue)
	if recoverErr != nil {
		return recoverErr
	}
	if _, retired := r.runtime.retired[copyValue.ID]; retired {
		return fmt.Errorf("restored session %q is tombstoned", copyValue.ID)
	}
	r.runtime.sessions[copyValue.ID] = &copyValue
	if changed {
		if err := r.runtime.persistLocked(ctx, checkpointReasonRecoveryApplied, &copyValue); err != nil {
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
	return r.RestoreSessionTombstone(
		context.Background(),
		SessionTombstone{SessionID: id},
	)
}

func (r runtimeRestorer) RestoreSessionTombstone(
	_ context.Context,
	tombstone SessionTombstone,
) error {
	if r.runtime == nil {
		return ErrRuntimeClosed
	}
	if tombstone.SessionID == "" {
		return errors.New("restored session tombstone has an empty session id")
	}
	r.runtime.mu.Lock()
	defer r.runtime.mu.Unlock()
	if tombstone.DeletedAt.IsZero() {
		tombstone.DeletedAt = r.runtime.now()
	}
	if _, found := r.runtime.sessions[tombstone.SessionID]; found {
		return fmt.Errorf(
			"restored session tombstone %q conflicts with a live session",
			tombstone.SessionID,
		)
	}
	r.runtime.retired[tombstone.SessionID] = cloneSessionTombstone(tombstone)
	return nil
}

func (r *Runtime) recoverLocked(
	session *Session,
) (bool, error) {
	if session == nil {
		return false, nil
	}
	changed := false
	now := r.now()
	for executionIndex := range session.Executions {
		execution := &session.Executions[executionIndex]
		needsReview := false

		for itemIndex := range execution.WorkItems {
			item := &execution.WorkItems[itemIndex]
			retainedBinding := retainedLLMDispatchBinding(*item)
			switch item.State {
			case WorkItemStateBinding:
				item.State = WorkItemStateReady
				item.Requested = false
				item.UpdatedAt = now
				if item.Cleanup {
					item.Requested = true
				}
				item.Binding = retainedBinding
				changed = true
			case WorkItemStateDispatched:
				replaySafe := effectReplaySafe(item.Effect)

				item.State = WorkItemStateUncertain
				item.Error = &SanitizedError{
					Code:     "recovered_external_outcome_uncertain",
					Category: ErrorCategoryUncertainty,
					Effect:   EffectStateUncertain,
					Message:  "external work was dispatched when the runtime stopped",
				}
				item.EffectState = EffectStateUncertain
				item.CompletedAt = &now
				item.UpdatedAt = now
				changed = true
				for attemptIndex := range execution.Attempts {
					attempt := &execution.Attempts[attemptIndex]
					if attempt.WorkItemID == item.ID && attempt.State == AttemptStateDispatched {
						attempt.State = AttemptStateUncertain
						attempt.EffectState = EffectStateUncertain
						attempt.Error = item.Error
						attempt.FinishedAt = &now
					}
				}
				//nolint:gocritic // No switch.
				if replaySafe {
					r.appendRecoveredInterruptedAssistantActivitiesLocked(
						session,
						execution,
						*item,
						now,
					)
					item.State = WorkItemStateReady
					item.Binding = retainedBinding
					item.Output = nil
					item.Error = nil
					item.EffectState = EffectStateNotStarted
					item.CompletedAt = nil
					item.DispatchedAt = nil
					item.RetryState = nil
					item.LLMStream = nil
					item.Requested = item.Automatic
				} else if r.reconciliationRequiredLocked(execution, item) {
					item.State = WorkItemStateReconciling
					item.CompletedAt = nil
					item.RetryState = nil
					item.Requested = false
				} else {
					if outcomeErr := r.recordUncertainWorkflowStepOutcomeLocked(
						execution,
						item,
					); outcomeErr != nil {
						return changed, outcomeErr
					}
					r.addRecoveryWaitLocked(session, execution, item)
				}
			default:
				// No recovery action.
			}
		}
		released := normalizeRecoveredResources(execution)
		if len(released) != 0 {
			changed = true
		}
		r.wakeResourceWaitersLocked(
			execution,
			released,
		)
		for waitIndex := range execution.Waits {
			wait := &execution.Waits[waitIndex]
			if !wait.Request.RuntimeBound && wait.Status == WaitStatusResolving {
				if wait.ResolutionIntent == nil {
					wait.Status = WaitStatusPending
				}
				wait.UpdatedAt = now
				changed = true
			}
			if !wait.Request.RuntimeBound ||
				(wait.Status != WaitStatusPending &&
					wait.Status != WaitStatusResolving) {
				continue
			}

			item := mutableWorkItem(execution, wait.WorkItemID)
			if item != nil && item.State == WorkItemStateWaiting {
				if wait.Status == WaitStatusResolving &&
					wait.ResolutionIntent != nil {
					wait.Status = WaitStatusPending
					wait.Error = &SanitizedError{
						Code:     "recovered_wait_resolution_rebind",
						Category: ErrorCategoryCancellation,
						Message:  "accepted wait resolution will resume through its rebound owner",
					}
					wait.UpdatedAt = now
					item.State = WorkItemStateReady
					item.Output = nil
					item.Error = nil
					item.RetryState = nil
					item.ResourceWait = nil
					item.BudgetConcurrencyWait = false
					item.Requested = true
					item.UpdatedAt = now
					changed = true
					continue
				}
				wait.Status = WaitStatusCancelled
				wait.Error = &SanitizedError{
					Code:     "recovered_runtime_wait_rebound",
					Category: ErrorCategoryCancellation,
					Message:  "runtime-bound wait will be recreated from the captured operation",
				}
				wait.UpdatedAt = now
				r.cancelWaitDeadlineLocked(wait.ID)
				delete(r.privateWaits, wait.ID)

				item.WaitID = ""
				item.State = WorkItemStateReady
				item.Output = nil
				item.Error = nil
				item.RetryState = nil
				item.ResourceWait = nil
				item.BudgetConcurrencyWait = false
				item.EffectState = undispatchedEffectState(item.Effect)
				item.Requested = true
				item.UpdatedAt = now
				changed = true
				continue
			}

			if wait.Request.RuntimeBound {
				wait.Status = WaitStatusInvalid
				wait.Error = &SanitizedError{
					Code:    "recovered_runtime_wait_unavailable",
					Message: "runtime authorization state was not persisted",
				}
				wait.UpdatedAt = now
				if item != nil {
					r.addRecoveryWaitLocked(session, execution, item)
				} else {
					needsReview = true
				}
				changed = true
			}
		}
		if needsReview && !execution.State.Terminal() {
			recovery := Wait{
				ID: WaitID(r.newID()), WorkItemID: recoveryReviewWorkItemID, Status: WaitStatusPending,
				Request: WaitRequest{Kind: WaitKindRecovery, Summary: jsonutil.MustCanonicalValue(
					map[string]string{"message": "legacy recovery review is required"},
				), AllowedResolutions: []WaitResolution{WaitResolutionContinue, WaitResolutionCancel}},
				CreatedAt: now, UpdatedAt: now,
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
						Summary: "Legacy recovery review is required.",
					},
				},
			)
			changed = true
		}
	}
	if r.recoverConversationOwnershipLocked(session, now) {
		changed = true
	}
	if changed {
		r.touchSessionLocked(session)
	}
	return changed, nil
}

func (r *Runtime) appendRecoveredInterruptedAssistantActivitiesLocked(
	session *Session,
	execution *Execution,
	item WorkItem,
	now time.Time,
) {
	if r == nil ||
		session == nil ||
		execution == nil ||
		item.Type != WorkItemTypeLLM ||
		item.LLMStream == nil {
		return
	}

	attempt := latestAttemptForWorkItem(*execution, item.ID)
	for _, activity := range interruptedAssistantActivitiesForAttempt(
		item,
		attempt,
	) {
		activity.ID = ActivityID(r.newID())
		activity.CreatedAt = now
		activity.ExecutionID = execution.ID
		activity.FrameID = item.FrameID
		session.Timeline = append(session.Timeline, activity)
	}
}

// retainedLLMDispatchBinding preserves durable semantic LLM facts while
// recovery discards private setup resources. Other work types rebind through
// their owning domains and do not retain a provider dispatch plan.
func retainedLLMDispatchBinding(item WorkItem) *SemanticBinding {
	if item.Type != WorkItemTypeLLM ||
		item.Binding == nil ||
		item.Binding.Plan == nil {
		return nil
	}

	value := jsonutil.CloneJSONInValue(*item.Binding)
	return &value
}

func (r *Runtime) settleRecoveredReviewWorkLocked(
	session *Session,
	execution *Execution,
	workItemID WorkItemID,
) error {
	if session == nil || execution == nil {
		return errors.New("recovery review execution is unavailable")
	}

	executionID := execution.ID
	workItemIDs := make([]WorkItemID, 0)
	for _, item := range execution.WorkItems {
		if item.State != WorkItemStateUncertain {
			continue
		}
		if workItemID == recoveryReviewWorkItemID ||
			workItemID == "" || item.ID == workItemID {
			workItemIDs = append(workItemIDs, item.ID)
		}
	}

	for _, workItemID := range workItemIDs {
		current := mutableSessionExecution(session, executionID)
		if current == nil {
			return ErrExecutionNotFound
		}
		item := mutableWorkItem(current, workItemID)
		if item == nil || item.State != WorkItemStateUncertain {
			continue
		}
		if item.WaitID != "" {
			item.WaitID = ""
		}

		now := r.now()
		if item.Error == nil {
			item.Error = &SanitizedError{
				Code:     "recovery_outcome_uncertain",
				Category: ErrorCategoryUncertainty,
				Effect:   EffectStateUncertain,
				Message:  "external work remained uncertain after recovery review",
			}
		}
		item.State = WorkItemStateFailed
		item.CompletedAt = &now
		item.UpdatedAt = now

		r.advanceSettledWorkLocked(
			session,
			current,
			item.FrameID,
			item.ID,
		)

		current = mutableSessionExecution(session, executionID)
		if current == nil {
			return ErrExecutionNotFound
		}
		item = mutableWorkItem(current, workItemID)
		if item == nil {
			continue
		}
		r.wakeCompletionWorkWaitsLocked(session, current, *item)

		current = mutableSessionExecution(session, executionID)
		if current == nil {
			return ErrExecutionNotFound
		}
		r.resumeSagaCompensationFramesLocked(session, current)
	}

	return nil
}
