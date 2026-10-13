package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

var (
	ErrRuntimeClosed     = errors.New("session runtime is closed")
	ErrSessionNotFound   = errors.New("session not found")
	ErrRunNotFound       = errors.New("run not found")
	ErrCommandIDConflict = errors.New("command ID was reused with a different payload")
	ErrCommandInProgress = errors.New("command ID is already being processed")
)

type commandRecord struct {
	Fingerprint string
	Receipt     CommandReceipt
}

type waitReference struct {
	ExecutorReference string
}

type workKind string

const (
	workModel        workKind = "model"
	workToolEvaluate workKind = "toolEvaluate"
	workToolExecute  workKind = "toolExecute"
	workResolveWait  workKind = "resolveWait"
	workExpireWait   workKind = "expireWait"
	workCloseSkill   workKind = "closeSkill"
)

type runtimeWork struct {
	Kind                 workKind
	SessionID            SessionID
	RunID                RunID
	OperationID          OperationID
	ToolCallID           ToolCallID
	WaitID               WaitID
	Automatic            bool
	Resolution           WaitResolution
	Input                json.RawMessage
	WaitExpiryGeneration uint64
	SkillID              string
}

type runtimeEffects struct {
	events          []RuntimeEvent
	checkpoints     []RuntimeCheckpointView
	work            []runtimeWork
	cancels         []context.CancelFunc
	deletedSessions []SessionID
}

type RuntimeOption func(*Runtime) error

func WithDriver(driver ExecutionDriver) RuntimeOption {
	return func(runtime *Runtime) error {
		if driver == nil {
			return errors.New("session runtime driver is nil")
		}
		if driver.Kind() == "" {
			return errors.New("session runtime driver kind is empty")
		}

		runtime.drivers[driver.Kind()] = driver
		return nil
	}
}

type Runtime struct {
	mu sync.RWMutex

	deps RuntimeDependencies

	sessions          map[SessionID]*Session
	runs              map[RunID]*Run
	runToSession      map[RunID]SessionID
	retiredSessionIDs map[SessionID]struct{}

	drivers map[string]ExecutionDriver

	commandRecords        map[CommandID]commandRecord
	commandReservations   map[CommandID]string
	sessionSequences      map[SessionID]uint64
	waitReferences        map[WaitID]waitReference
	waitExpirers          map[WaitID]context.CancelFunc
	waitExpiryGenerations map[WaitID]uint64
	waitExpirySequence    uint64
	cancels               map[string]context.CancelFunc

	subscribers      map[uint64]chan RuntimeEvent
	nextSubscriberID uint64

	lifecycleCtx    context.Context
	lifecycleCancel context.CancelFunc

	closed bool

	effectMu      sync.Mutex
	effectWG      sync.WaitGroup
	effectClosing bool
}

func NewRuntime(
	dependencies RuntimeDependencies,
	options ...RuntimeOption,
) (*Runtime, error) {
	if dependencies.Model == nil {
		return nil, errors.New("session runtime requires a model executor")
	}

	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())

	runtime := &Runtime{
		deps:                  dependencies,
		sessions:              make(map[SessionID]*Session),
		runs:                  make(map[RunID]*Run),
		runToSession:          make(map[RunID]SessionID),
		retiredSessionIDs:     make(map[SessionID]struct{}),
		drivers:               make(map[string]ExecutionDriver),
		commandRecords:        make(map[CommandID]commandRecord),
		commandReservations:   make(map[CommandID]string),
		sessionSequences:      make(map[SessionID]uint64),
		waitReferences:        make(map[WaitID]waitReference),
		waitExpirers:          make(map[WaitID]context.CancelFunc),
		waitExpiryGenerations: make(map[WaitID]uint64),
		cancels:               make(map[string]context.CancelFunc),
		subscribers:           make(map[uint64]chan RuntimeEvent),
		lifecycleCtx:          lifecycleCtx,
		lifecycleCancel:       lifecycleCancel,
	}

	runtime.drivers[SequentialChatDriverKind] = NewSequentialChatDriver()

	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(runtime); err != nil {
			return nil, err
		}
	}

	return runtime, nil
}

func (r *Runtime) Restore(ctx context.Context) error {
	if r == nil {
		return ErrRuntimeClosed
	}
	if r.deps.Recovery == nil {
		return nil
	}

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

	cancels := make([]context.CancelFunc, 0, len(r.cancels))
	for _, cancel := range r.cancels {
		cancels = append(cancels, cancel)
	}
	clear(r.cancels)

	waitCancels := make([]context.CancelFunc, 0, len(r.waitExpirers))
	for _, cancel := range r.waitExpirers {
		waitCancels = append(waitCancels, cancel)
	}
	clear(r.waitExpirers)
	clear(r.waitReferences)
	clear(r.waitExpiryGenerations)

	skillClosures := make([]runtimeWork, 0)
	for _, run := range r.runs {
		if run == nil || run.SkillSession.ID == "" {
			continue
		}
		skillClosures = append(skillClosures, runtimeWork{
			Kind:      workCloseSkill,
			SessionID: run.SessionID,
			RunID:     run.ID,
			SkillID:   run.SkillSession.ID,
		})
		run.SkillSession.ID = ""
	}

	lifecycleCancel := r.lifecycleCancel
	r.lifecycleCancel = nil

	subscribers := r.subscribers
	r.subscribers = make(map[uint64]chan RuntimeEvent)
	r.mu.Unlock()

	r.effectMu.Lock()
	r.effectClosing = true
	r.effectMu.Unlock()

	if lifecycleCancel != nil {
		lifecycleCancel()
	}
	for _, cancel := range cancels {
		cancel()
	}
	for _, cancel := range waitCancels {
		cancel()
	}

	r.effectWG.Wait()

	for _, closeWork := range skillClosures {
		closeCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		r.closeSkillSession(
			closeCtx,
			closeWork.SkillID,
			closeWork.SessionID,
			closeWork.RunID,
		)
		cancel()
	}

	for _, subscriber := range subscribers {
		close(subscriber)
	}
}

func (r *Runtime) Subscribe(
	buffer int,
) (events <-chan RuntimeEvent, unsubscribe func()) {
	if buffer < 1 {
		buffer = 64
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	channel := make(chan RuntimeEvent, buffer)
	if r.closed {
		close(channel)
		return channel, func() {}
	}

	r.nextSubscriberID++
	id := r.nextSubscriberID
	r.subscribers[id] = channel

	return channel, func() {
		r.mu.Lock()
		current, found := r.subscribers[id]
		if found {
			delete(r.subscribers, id)
			close(current)
		}
		r.mu.Unlock()
	}
}

func (r *Runtime) GetSession(
	_ context.Context,
	id SessionID,
) (Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessionValue := r.sessions[id]
	if sessionValue == nil {
		return Session{}, ErrSessionNotFound
	}

	return cloneJSON(*sessionValue), nil
}

func (r *Runtime) GetSessionSnapshot(
	_ context.Context,
	id SessionID,
) (SessionSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessionValue := r.sessions[id]
	if sessionValue == nil {
		return SessionSnapshot{}, ErrSessionNotFound
	}

	snapshot := SessionSnapshot{
		Session: cloneJSON(*sessionValue),
		Runs:    make([]Run, 0, len(sessionValue.RunIDs)),
	}
	for _, runID := range sessionValue.RunIDs {
		run := r.runs[runID]
		if run == nil {
			continue
		}
		snapshot.Runs = append(snapshot.Runs, cloneJSON(*run))
	}

	return snapshot, nil
}

func (r *Runtime) GetRunSnapshot(
	_ context.Context,
	id RunID,
) (RunSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	run := r.runs[id]
	if run == nil {
		return RunSnapshot{}, ErrRunNotFound
	}

	sessionValue := r.sessions[run.SessionID]
	if sessionValue == nil {
		return RunSnapshot{}, ErrSessionNotFound
	}

	return RunSnapshot{
		SessionID:       run.SessionID,
		SessionRevision: sessionValue.Revision,
		Run:             cloneJSON(*run),
	}, nil
}

func (r *Runtime) ListSessions(
	_ context.Context,
) ([]SessionListItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	output := make([]SessionListItem, 0, len(r.sessions))
	for _, sessionValue := range r.sessions {
		item := SessionListItem{
			ID:        sessionValue.ID,
			Revision:  sessionValue.Revision,
			Title:     sessionValue.Title,
			UpdatedAt: sessionValue.UpdatedAt,
		}

		for _, runID := range sessionValue.RunIDs {
			run := r.runs[runID]
			if run != nil && !run.State.Terminal() {
				item.ActiveRuns = append(item.ActiveRuns, runID)
			}
		}

		output = append(output, item)
	}

	return output, nil
}

func (r *Runtime) now() time.Time {
	if r.deps.Clock != nil {
		return clockutil.NowUTC(r.deps.Clock)
	}

	return clockutil.System{}.Now()
}

func (r *Runtime) newID() string {
	if r.deps.IDs != nil {
		return r.deps.IDs.NewID()
	}

	return uuidutil.NewUUIDv7()
}

func (r *Runtime) sessionAndRunLocked(
	sessionID SessionID,
	runID RunID,
) (*Session, *Run, error) {
	sessionValue := r.sessions[sessionID]
	if sessionValue == nil {
		return nil, nil, ErrSessionNotFound
	}

	run := r.runs[runID]
	if run == nil || run.SessionID != sessionID {
		return nil, nil, ErrRunNotFound
	}

	return sessionValue, run, nil
}

func (r *Runtime) touchSessionLocked(sessionValue *Session) {
	if sessionValue == nil {
		return
	}
	sessionValue.Revision++
	sessionValue.UpdatedAt = r.now()
}

func (r *Runtime) touchRunLocked(
	sessionValue *Session,
	run *Run,
) {
	r.touchSessionLocked(sessionValue)
	run.Revision++
	run.UpdatedAt = sessionValue.UpdatedAt
}

func (r *Runtime) recordSessionEventLocked(
	effects *runtimeEffects,
	sessionValue *Session,
	commandID CommandID,
	kind RuntimeEventKind,
	data runtimeEventData,
) {
	r.sessionSequences[sessionValue.ID]++
	effects.events = append(effects.events, RuntimeEvent{
		SessionID: sessionValue.ID,
		Sequence:  r.sessionSequences[sessionValue.ID],
		CommandID: commandID,
		Kind:      kind,
		At:        r.now(),
		Data:      eventData(data),
	})
}

func (r *Runtime) recordRunEventLocked(
	effects *runtimeEffects,
	sessionValue *Session,
	run *Run,
	commandID CommandID,
	kind RuntimeEventKind,
	operationID OperationID,
	toolCallID ToolCallID,
	toolAttemptID ToolAttemptID,
	waitID WaitID,
	data runtimeEventData,
) {
	run.EventSequence++
	effects.events = append(effects.events, RuntimeEvent{
		SessionID:     sessionValue.ID,
		RunID:         run.ID,
		Sequence:      run.EventSequence,
		CommandID:     commandID,
		OperationID:   operationID,
		ToolCallID:    toolCallID,
		ToolAttemptID: toolAttemptID,
		WaitID:        waitID,
		Kind:          kind,
		At:            r.now(),
		Data:          eventData(data),
	})
}

func (r *Runtime) checkpointLocked(
	effects *runtimeEffects,
	reason RuntimeEventKind,
	sessionValue *Session,
	run *Run,
) {
	if r.deps.Checkpoints == nil {
		return
	}

	view := RuntimeCheckpointView{
		Reason:    reason,
		Session:   cloneJSON(*sessionValue),
		CreatedAt: r.now(),
	}
	if run != nil {
		cloned := cloneJSON(*run)
		view.Run = &cloned
	}
	effects.checkpoints = append(effects.checkpoints, view)
}

func (r *Runtime) launchEffect(fn func()) {
	r.effectMu.Lock()
	if r.effectClosing {
		r.effectMu.Unlock()
		return
	}
	r.effectWG.Add(1)
	r.effectMu.Unlock()

	go func() {
		defer r.effectWG.Done()
		fn()
	}()
}

func (r *Runtime) flushEffects(
	ctx context.Context,
	effects runtimeEffects,
) {
	effectCtx := context.WithoutCancel(ctx)

	for _, cancel := range effects.cancels {
		cancel()
	}

	for _, event := range effects.events {
		r.publishEvent(effectCtx, event)
	}

	for _, checkpoint := range effects.checkpoints {
		r.handleCheckpoint(effectCtx, checkpoint)
	}

	for _, sessionID := range effects.deletedSessions {
		r.handleSessionDeletion(effectCtx, sessionID)
	}

	for _, work := range effects.work {
		workCopy := work
		r.launchEffect(func() {
			r.dispatchWork(effectCtx, workCopy)
		})
	}
}

func (r *Runtime) publishEvent(
	ctx context.Context,
	event RuntimeEvent,
) {
	r.mu.RLock()
	publisher := r.deps.Events

	for _, subscriber := range r.subscribers {
		select {
		case subscriber <- event:
		default:
			r.reportDiagnostic(ctx, RuntimeDiagnostic{
				SessionID: event.SessionID,
				RunID:     event.RunID,
				Operation: "runtime event subscriber",
				Message:   "runtime event subscriber buffer is full; client must reattach with a snapshot",
				At:        r.now(),
			})
		}
	}
	r.mu.RUnlock()

	if publisher != nil {
		f := func() {
			publishCtx := context.WithoutCancel(ctx)
			eventPublisher := publisher
			runtimeEvent := event
			if err := eventPublisher.PublishRuntimeEvent(publishCtx, runtimeEvent); err != nil {
				r.reportDiagnostic(publishCtx, RuntimeDiagnostic{
					SessionID: event.SessionID,
					RunID:     event.RunID,
					Operation: "publish runtime event",
					Message:   "runtime event publication failed",
					Cause:     err.Error(),
					At:        r.now(),
				})
			}
		}
		r.launchEffect(f)
	}
}

func (r *Runtime) handleCheckpoint(
	ctx context.Context,
	checkpoint RuntimeCheckpointView,
) {
	if r.deps.Checkpoints == nil {
		return
	}

	r.launchEffect(func() {
		checkpointCtx := context.WithoutCancel(ctx)
		sink := r.deps.Checkpoints
		checkpointView := checkpoint
		if err := sink.HandleCheckpoint(
			checkpointCtx,
			checkpoint,
		); err != nil {
			r.reportDiagnostic(checkpointCtx, RuntimeDiagnostic{
				SessionID: checkpointView.Session.ID,
				Operation: "checkpoint",
				Message:   "checkpoint sink failed; live runtime state remains active",
				Cause:     err.Error(),
				At:        r.now(),
			})
		}
	})
}

func (r *Runtime) handleSessionDeletion(
	ctx context.Context,
	sessionID SessionID,
) {
	deletionSink, ok := r.deps.Checkpoints.(SessionDeletionSink)
	if !ok || deletionSink == nil {
		return
	}

	r.launchEffect(func() {
		if err := deletionSink.HandleSessionDeleted(
			context.WithoutCancel(ctx),
			sessionID,
		); err != nil {
			r.reportDiagnostic(ctx, RuntimeDiagnostic{
				SessionID: sessionID,
				Operation: "delete session checkpoint",
				Message:   "session deletion sink failed; live runtime state remains deleted",
				Cause:     err.Error(),
				At:        r.now(),
			})
		}
	})
}

func (r *Runtime) reportDiagnostic(
	ctx context.Context,
	diagnostic RuntimeDiagnostic,
) {
	sink := r.deps.Diagnostics
	if sink == nil {
		return
	}
	r.launchEffect(func() {
		sink.ReportRuntimeDiagnostic(
			context.WithoutCancel(ctx),
			diagnostic,
		)
	})
}

func eventData(value runtimeEventData) json.RawMessage {
	if value == nil {
		return nil
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}

	return raw
}

func cloneJSON[T any](value T) T {
	return jsonutil.CloneJSONInValue(value)
}

type runtimeRestorer struct {
	runtime *Runtime
}

func (r runtimeRestorer) RestoreSession(
	_ context.Context,
	sessionValue Session,
) error {
	if r.runtime == nil {
		return ErrRuntimeClosed
	}

	if sessionValue.ID == "" {
		return errors.New("recovery session has an empty ID")
	}

	r.runtime.mu.Lock()
	defer r.runtime.mu.Unlock()

	if r.runtime.closed {
		return ErrRuntimeClosed
	}

	copyValue := cloneJSON(sessionValue)
	if copyValue.CreatedAt.IsZero() {
		copyValue.CreatedAt = r.runtime.now()
	}
	if copyValue.UpdatedAt.IsZero() {
		copyValue.UpdatedAt = copyValue.CreatedAt
	}
	if _, found := r.runtime.sessions[copyValue.ID]; found {
		return fmt.Errorf("recovery session %q is duplicated", copyValue.ID)
	}
	r.runtime.sessions[copyValue.ID] = &copyValue
	r.runtime.sessionSequences[copyValue.ID] = uint64(copyValue.Revision)

	return nil
}

func (r runtimeRestorer) RestoreRun(
	_ context.Context,
	runValue Run,
) error {
	if r.runtime == nil {
		return ErrRuntimeClosed
	}

	if runValue.ID == "" || runValue.SessionID == "" {
		return errors.New("recovery run has an empty ID or session ID")
	}

	r.runtime.mu.Lock()
	defer r.runtime.mu.Unlock()

	if r.runtime.closed {
		return ErrRuntimeClosed
	}

	sessionValue := r.runtime.sessions[runValue.SessionID]
	if sessionValue == nil {
		return fmt.Errorf(
			"recovery run %q references unknown session %q",
			runValue.ID,
			runValue.SessionID,
		)
	}

	copyValue := cloneJSON(runValue)
	normalizeRecoveredRun(&copyValue, r.runtime.now())

	if _, found := r.runtime.runs[copyValue.ID]; found {
		return fmt.Errorf("recovery run %q is duplicated", copyValue.ID)
	}
	if active := r.runtime.activeRunLocked(sessionValue); active != nil &&
		!copyValue.State.Terminal() {
		return fmt.Errorf(
			"recovery run %q conflicts with active run %q in session %q",
			copyValue.ID,
			active.ID,
			copyValue.SessionID,
		)
	}

	r.runtime.runs[copyValue.ID] = &copyValue
	r.runtime.runToSession[copyValue.ID] = copyValue.SessionID

	found := slices.Contains(sessionValue.RunIDs, copyValue.ID)
	if !found {
		sessionValue.RunIDs = append(sessionValue.RunIDs, copyValue.ID)
	}
	r.runtime.touchSessionLocked(sessionValue)

	return nil
}

func normalizeRecoveredRun(
	run *Run,
	now time.Time,
) {
	if run == nil {
		return
	}
	run.SkillSession.ID = ""
	run.SkillSession.Generation++

	requiresRecoveryReview := false

	for operationIndex := range run.ModelOperations {
		operation := &run.ModelOperations[operationIndex]
		switch operation.State {
		case OperationStateQueued,
			OperationStatePreparing,
			OperationStateRunning:
			requiresRecoveryReview = true
			operation.State = OperationStateInterrupted
			operation.FinishedAt = now
			operation.Error = &OperationError{
				Code:    "recovered_model_interrupted",
				Message: "model operation was in flight when the runtime stopped",
			}
		default:
		}
	}

	for callIndex := range run.ToolCalls {
		call := &run.ToolCalls[callIndex]
		for attemptIndex := range call.Attempts {
			attempt := &call.Attempts[attemptIndex]
			if attempt.State == ToolAttemptStateQueued ||
				attempt.State == ToolAttemptStateRunning {
				requiresRecoveryReview = true
				attempt.State = ToolAttemptStateUncertain
				attempt.Uncertain = true
				attempt.FinishedAt = now
				if attempt.Error == "" {
					attempt.Error = "external tool outcome is uncertain after recovery"
				}
			}
		}

		switch call.State {
		case ToolCallStateEvaluationQueued,
			ToolCallStateEvaluating,
			ToolCallStateExecutionQueued,
			ToolCallStateWaiting,
			ToolCallStateRunning:
			requiresRecoveryReview = true
			call.State = ToolCallStateFailed
			call.LastFailure = "external tool outcome requires recovery decision"
		default:
		}
	}

	for waitIndex := range run.WaitingRequests {
		wait := &run.WaitingRequests[waitIndex]
		if wait.Status != WaitStatusPending &&
			wait.Status != WaitStatusResolving {
			continue
		}

		if wait.Kind == WaitKindRecovery {
			requiresRecoveryReview = true
			continue
		}

		requiresRecoveryReview = true
		wait.Status = WaitStatusInvalid
		wait.Error = "wait executor state is unavailable after runtime recovery"
		wait.UpdatedAt = now
	}

	if !run.State.Terminal() &&
		(run.State == RunStateQueued ||
			run.State == RunStateRunning ||
			run.State == RunStateCancelling ||
			requiresRecoveryReview) {
		run.State = RunStateWaiting
		run.Outcome = nil
		ensureRecoveryWait(run, now)
	}

	run.ActiveModelOperationID = ""
	run.Controls.Paused = run.State == RunStatePaused
	run.Controls.CancellationRequested = false
	run.Revision++
	run.UpdatedAt = now
}

func ensureRecoveryWait(
	run *Run,
	now time.Time,
) {
	for _, wait := range run.WaitingRequests {
		if wait.Kind == WaitKindRecovery &&
			(wait.Status == WaitStatusPending ||
				wait.Status == WaitStatusResolving) {
			return
		}
	}

	summary, _ := json.Marshal(RecoveryWaitSummary{
		Message: "recovered work requires review before new external work is dispatched",
	})

	run.WaitingRequests = append(run.WaitingRequests, WaitingRequest{
		ID: WaitID(fmt.Sprintf(
			"recovery-%s-%d",
			run.ID,
			now.UnixNano(),
		)),
		Kind:    WaitKindRecovery,
		Status:  WaitStatusPending,
		Summary: summary,
		AllowedResolutions: []WaitResolution{
			WaitResolutionContinue,
			WaitResolutionCancel,
		},
		CreatedAt: now,
		UpdatedAt: now,
	})
}
