package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/backgroundutil"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

const (
	defaultRunMaxModelOperations = 32
	defaultRunMaxToolAttempts    = 64
	maxRunMaxModelOperations     = 256
	maxRunMaxToolAttempts        = 512
	skillSessionCloseTimeout     = 5 * time.Second
)

var (
	ErrRuntimeClosed     = errors.New("session runtime is closed")
	ErrSessionNotFound   = errors.New("session not found")
	ErrRunNotFound       = errors.New("run not found")
	ErrCommandIDConflict = errors.New("command ID was reused with a different payload")
	ErrCommandInProgress = errors.New("command ID is already being processed")
)

const streamCheckpointMinInterval = time.Second

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
	streamCheckpointAt    map[OperationID]time.Time

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
	switch {
	case dependencies.Preparation == nil:
		return nil, errors.New("session runtime requires a submission preparer")
	case dependencies.Operations == nil:
		return nil, errors.New("session runtime requires an operation preparer")
	case dependencies.Templates == nil:
		return nil, errors.New("session runtime requires a user-message Skill template preparer")
	case dependencies.Model == nil:
		return nil, errors.New("session runtime requires a model executor")
	case dependencies.Tools == nil:
		return nil, errors.New("session runtime requires a Tool executor")
	case dependencies.Skills == nil:
		return nil, errors.New("session runtime requires a Skill session controller")
	case dependencies.Apps == nil:
		return nil, errors.New("session runtime requires an App executor")
	case dependencies.Diagnostics == nil:
		return nil, errors.New("session runtime requires a diagnostic sink")
	case dependencies.Attachments == nil:
		return nil, errors.New("session runtime requires an attachment preparer")
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
		streamCheckpointAt:    make(map[OperationID]time.Time),
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
	if r.deps.Recovery == nil {
		return nil
	}

	return r.deps.Recovery.Restore(ctx, runtimeRestorer{runtime: r})
}

func (r *Runtime) PrepareUserMessageSkillTemplate(ctx context.Context,
	request UserMessageSkillTemplatePreparationRequest,
) (UserMessageSkillTemplateResult, error) {
	if r.deps.Templates == nil {
		return UserMessageSkillTemplateResult{}, errors.New("no template engine available")
	}

	return r.deps.Templates.PrepareUserMessageSkillTemplate(ctx, request)
}

func (r *Runtime) PreviewInstructionSkill(
	ctx context.Context,
	request InstructionSkillPreviewRequest,
) (InstructionSkillPreviewResult, error) {
	if r.deps.InstructionSkills == nil {
		return InstructionSkillPreviewResult{}, errors.New(
			"no instruction Skill previewer is available",
		)
	}
	return r.deps.InstructionSkills.PreviewInstructionSkill(ctx, request)
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
	clear(r.streamCheckpointAt)

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
		Session:  cloneJSON(*sessionValue),
		Sequence: r.sessionSequences[id],
		Runs:     make([]Run, 0, len(sessionValue.RunIDs)),
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

	sort.Slice(output, func(left, right int) bool {
		if !output[left].UpdatedAt.Equal(output[right].UpdatedAt) {
			return output[left].UpdatedAt.After(output[right].UpdatedAt)
		}

		return output[left].ID < output[right].ID
	})

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
	sessionValue.EventSequence = r.sessionSequences[sessionValue.ID]
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
	r.sessionSequences[sessionValue.ID]++
	sessionValue.EventSequence = r.sessionSequences[sessionValue.ID]
	effects.events = append(effects.events, RuntimeEvent{
		SessionID:     sessionValue.ID,
		RunID:         run.ID,
		Sequence:      r.sessionSequences[sessionValue.ID],
		RunSequence:   run.EventSequence,
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

func (r *Runtime) shouldCheckpointStreamLocked(
	operationID OperationID,
) bool {
	if r.deps.Checkpoints == nil || operationID == "" {
		return false
	}

	now := r.now()
	last := r.streamCheckpointAt[operationID]
	if !last.IsZero() &&
		now.Sub(last) < streamCheckpointMinInterval {
		return false
	}

	r.streamCheckpointAt[operationID] = now
	return true
}

func (r *Runtime) launchEffect(fn func()) {
	r.launchEffectWithFailure(
		"session runtime effect",
		func(failure backgroundutil.PanicFailureError) {
			r.reportDiagnostic(r.lifecycleCtx, RuntimeDiagnostic{
				Operation: failure.Operation,
				Message:   "background effect panicked",
				Cause:     failure.Error(),
				At:        time.Now().UTC(),
			})
		},
		fn,
	)
}

func (r *Runtime) launchEffectWithFailure(
	operation string,
	onPanic func(backgroundutil.PanicFailureError),
	fn func(),
) {
	r.effectMu.Lock()
	if r.effectClosing {
		r.effectMu.Unlock()
		return
	}
	backgroundutil.Launch(
		&r.effectWG,
		operation,
		fn,
		onPanic,
	)
	r.effectMu.Unlock()
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
		r.launchWork(effectCtx, work)
	}
}

func (r *Runtime) publishEvent(
	ctx context.Context,
	event RuntimeEvent,
) {
	r.mu.RLock()
	publisher := r.deps.Events
	droppedSubscribers := 0
	for _, subscriber := range r.subscribers {
		// Keep the read lock while sending. Unsubscribe and Close acquire the
		// write lock before closing a subscriber channel, so this cannot race
		// into a send-on-closed-channel panic. Sending is non-blocking.
		select {
		case subscriber <- event:
		default:
			droppedSubscribers++
		}
	}
	r.mu.RUnlock()

	for index := 0; index < droppedSubscribers; index++ {
		r.reportDiagnostic(ctx, RuntimeDiagnostic{
			SessionID: event.SessionID,
			RunID:     event.RunID,
			Operation: "runtime event subscriber",
			Message: "runtime event subscriber buffer is full; " +
				"client must reattach with a snapshot",
			At: r.now(),
		})
	}

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
	if !ok {
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
	// A panicking diagnostic sink must not create an infinite sequence of
	// diagnostic-dispatch panics. The common boundary still logs its stack.
	r.launchEffectWithFailure(
		"session diagnostic sink",
		nil,
		func() {
			sink.ReportRuntimeDiagnostic(
				context.WithoutCancel(ctx),
				diagnostic,
			)
		},
	)
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
	r.runtime.sessionSequences[copyValue.ID] = copyValue.EventSequence

	return nil
}

func (r runtimeRestorer) RestoreRetiredSession(
	ctx context.Context,
	id SessionID,
) error {
	if r.runtime == nil {
		return ErrRuntimeClosed
	}

	effects := runtimeEffects{}
	r.runtime.mu.Lock()
	defer func() {
		r.runtime.mu.Unlock()
		if len(effects.events) != 0 || len(effects.checkpoints) != 0 {
			r.runtime.flushEffects(ctx, effects)
		}
	}()

	if r.runtime.closed {
		return ErrRuntimeClosed
	}
	if r.runtime.retiredSessionIDs == nil {
		r.runtime.retiredSessionIDs = make(map[SessionID]struct{})
	}
	r.runtime.retiredSessionIDs[id] = struct{}{}
	return nil
}

func (r runtimeRestorer) RestoreRun(
	ctx context.Context,
	runValue Run,
) error {
	if r.runtime == nil {
		return ErrRuntimeClosed
	}

	if runValue.ID == "" || runValue.SessionID == "" {
		return errors.New("recovery run has an empty ID or session ID")
	}

	effects := runtimeEffects{}
	recoveryCheckpoint := false

	restoreErr := func() error {
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
		recoveryApplied := normalizeRecoveredRun(
			&copyValue,
			r.runtime.now(),
		)
		for index := range copyValue.Apps {
			copyValue.Apps[index].Route = nil
			copyValue.Apps[index].UnavailableReason = "App runtime route is unavailable after recovery"
		}

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

		associationChanged := !slices.Contains(
			sessionValue.RunIDs,
			copyValue.ID,
		)
		if associationChanged {
			sessionValue.RunIDs = append(sessionValue.RunIDs, copyValue.ID)
		}
		if recoveryApplied || associationChanged {
			r.runtime.touchSessionLocked(sessionValue)
		}

		if recoveryApplied {
			r.runtime.recordRunEventLocked(
				&effects,
				sessionValue,
				&copyValue,
				"",
				RuntimeEventRecoveryApplied,
				"",
				"",
				"",
				"",
				RunStateChangedEventData{State: copyValue.State},
			)
			r.runtime.checkpointLocked(
				&effects,
				RuntimeEventRecoveryApplied,
				sessionValue,
				&copyValue,
			)
			recoveryCheckpoint = true
		}
		return nil
	}()
	if restoreErr != nil {
		return restoreErr
	}

	if recoveryCheckpoint {
		r.runtime.flushEffects(ctx, effects)
	}
	return nil
}

func (r *Runtime) prepareSubmission(
	ctx context.Context,
	sessionID SessionID,
	configuration SessionConfiguration,
	input UserInput,
	operationID OperationID,
	persistConfiguration bool,
) (PreparedSubmission, error) {
	input = cloneJSON(input)
	attachments, err := r.prepareAttachments(ctx, input.Attachments)
	if err != nil {
		return PreparedSubmission{}, err
	}
	input.Attachments = attachments

	prepared, err := r.deps.Preparation.PrepareSubmission(
		ctx,
		SubmissionPreparationRequest{
			SessionID:            sessionID,
			Configuration:        cloneJSON(configuration),
			Input:                cloneJSON(input),
			OperationID:          operationID,
			PersistConfiguration: persistConfiguration,
		},
	)
	if err != nil {
		return PreparedSubmission{}, err
	}

	// Persistence is a command decision. A preparer may normalize the
	// configuration, but it must not silently change whether it becomes the
	// Session default.
	prepared.PersistConfiguration = persistConfiguration
	return prepared, nil
}

func (r *Runtime) prepareAttachments(
	ctx context.Context,
	references []AttachmentReference,
) ([]AttachmentReference, error) {
	if len(references) == 0 {
		return nil, nil
	}
	values, err := r.deps.Attachments.PrepareAttachments(
		ctx,
		cloneJSON(references),
	)
	if err != nil {
		return nil, fmt.Errorf("prepare attachments: %w", err)
	}
	return values, nil
}

func normalizeRunLimits(input *RunLimits) (RunLimits, error) {
	output := RunLimits{
		MaxModelOperations: defaultRunMaxModelOperations,
		MaxToolAttempts:    defaultRunMaxToolAttempts,
	}
	if input == nil {
		return output, nil
	}

	output = cloneJSON(*input)
	if output.MaxModelOperations < 0 ||
		output.MaxToolAttempts < 0 {
		return RunLimits{}, errors.New("run limits cannot be negative")
	}
	if output.MaxModelOperations == 0 {
		output.MaxModelOperations = defaultRunMaxModelOperations
	}
	if output.MaxToolAttempts == 0 {
		output.MaxToolAttempts = defaultRunMaxToolAttempts
	}
	if output.MaxModelOperations > maxRunMaxModelOperations {
		return RunLimits{}, fmt.Errorf(
			"run model-operation limit cannot exceed %d",
			maxRunMaxModelOperations,
		)
	}
	if output.MaxToolAttempts > maxRunMaxToolAttempts {
		return RunLimits{}, fmt.Errorf(
			"run tool-attempt limit cannot exceed %d",
			maxRunMaxToolAttempts,
		)
	}
	return output, nil
}

func uniqueArtifactReferences(
	values []ArtifactReference,
) []ArtifactReference {
	seen := make(map[string]struct{}, len(values))
	output := make([]ArtifactReference, 0, len(values))

	for _, value := range values {
		key := value.RootID + "\x00" + value.ArtifactID
		if value.RootID == "" || value.ArtifactID == "" {
			continue
		}
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		output = append(output, value)
	}

	return output
}

func hasPendingRecoveryReview(run *Run) bool {
	if run == nil {
		return false
	}
	for _, wait := range run.WaitingRequests {
		if wait.Kind != WaitKindRecovery {
			continue
		}
		if wait.Status == WaitStatusPending ||
			wait.Status == WaitStatusResolving {
			return true
		}
	}
	return false
}

func toolMutationMessage(run *Run, call *ToolCall) string {
	if run == nil || call == nil {
		return toolCallNotFound
	}
	if run.State.Terminal() || run.Controls.CancellationRequested {
		return "terminal or cancelling runs cannot change tool calls"
	}
	if hasPendingRecoveryReview(run) {
		return "recovery review must be resolved before changing tool calls"
	}
	if run.ActiveModelOperationID != "" {
		return "tool calls cannot change while a model operation is active"
	}
	if call.ConsumedByOperationID != "" {
		return "the selected tool result has already been consumed"
	}

	latest := latestBatchOperationID(run)
	if latest != "" && call.OriginOperationID != latest {
		return "the tool call does not belong to the current tool batch"
	}

	switch call.State {
	case ToolCallStateEvaluationQueued,
		ToolCallStateEvaluating,
		ToolCallStateExecutionQueued,
		ToolCallStateWaiting,
		ToolCallStateRunning:
		return "tool call is active and cannot be changed manually"
	default:
		return ""
	}
}

func (r *Runtime) resumeRecoveredModelLocked(
	sessionValue *Session,
	run *Run,
	commandID CommandID,
	effects *runtimeEffects,
) (runtimeWork, bool, error) {
	if run == nil || sessionValue == nil {
		return runtimeWork{}, false, errors.New("recovered run is unavailable")
	}
	if run.ActiveModelOperationID != "" {
		return runtimeWork{}, false, errors.New("a model operation is already active")
	}
	if !allRequiredCallsResolved(run) {
		return runtimeWork{}, false, nil
	}
	if len(run.ModelOperations) == 0 {
		return runtimeWork{}, false, nil
	}

	latest := run.ModelOperations[len(run.ModelOperations)-1]
	if latest.State != OperationStateInterrupted {
		return runtimeWork{}, false, nil
	}
	if strings.TrimSpace(latest.Request.Current.ID) == "" {
		return runtimeWork{}, false, errors.New(
			"recovered interrupted model operation has no current message",
		)
	}

	operationConfiguration := cloneJSON(latest.Request.Configuration)
	work, err := r.startModelOperationLocked(
		sessionValue,
		run,
		commandID,
		"",
		latest.Request.Current.ID,
		&operationConfiguration,
		nil,
		false,
		effects,
	)
	if err != nil {
		return runtimeWork{}, false, err
	}

	return work, true, nil
}

func (r *Runtime) cancelOutstandingWaitsLocked(
	run *Run,
	reason string,
) {
	if run == nil {
		return
	}

	for index := range run.WaitingRequests {
		wait := &run.WaitingRequests[index]
		if wait.Status != WaitStatusPending &&
			wait.Status != WaitStatusResolving {
			continue
		}

		wait.Status = WaitStatusCancelled
		wait.Error = reason
		wait.UpdatedAt = r.now()
		r.clearWaitReferenceLocked(wait.ID)
	}
}

func (r *Runtime) cancelWaitExpiryLocked(waitID WaitID) {
	if cancel, found := r.waitExpirers[waitID]; found {
		delete(r.waitExpirers, waitID)
		cancel()
	}
	delete(r.waitExpiryGenerations, waitID)
}

func (r *Runtime) scheduleWaitExpiryLocked(
	effects *runtimeEffects,
	sessionID SessionID,
	runID RunID,
	wait WaitingRequest,
) {
	if effects == nil ||
		wait.ID == "" ||
		wait.ExpiresAt == nil ||
		wait.Status != WaitStatusPending {
		return
	}
	if r.waitExpiryGenerations == nil {
		r.waitExpiryGenerations = make(map[WaitID]uint64)
	}

	r.cancelWaitExpiryLocked(wait.ID)
	r.waitExpirySequence++
	r.waitExpiryGenerations[wait.ID] = r.waitExpirySequence
	effects.work = append(effects.work, runtimeWork{
		Kind:                 workExpireWait,
		SessionID:            sessionID,
		RunID:                runID,
		WaitID:               wait.ID,
		WaitExpiryGeneration: r.waitExpirySequence,
	})
}

func (r *Runtime) clearWaitReferenceLocked(waitID WaitID) {
	r.cancelWaitExpiryLocked(waitID)
	delete(r.waitReferences, waitID)
}

func (r *Runtime) cancelUndispatchedLocked(run *Run) {
	if run == nil {
		return
	}

	now := r.now()

	for index := range run.ModelOperations {
		operation := &run.ModelOperations[index]
		switch operation.State {
		case OperationStateQueued, OperationStatePreparing:
			operation.State = OperationStateCancelled
			operation.FinishedAt = now
			operation.Error = &OperationError{
				Code:    string(RunStateCancelled),
				Message: "cancelled before model invocation",
			}
			if run.ActiveModelOperationID == operation.ID {
				run.ActiveModelOperationID = ""
			}
		default:
		}
	}

	for index := range run.WaitingRequests {
		wait := &run.WaitingRequests[index]
		if wait.Status != WaitStatusPending &&
			wait.Status != WaitStatusResolving {
			continue
		}
		wait.Status = WaitStatusCancelled
		wait.Error = "run was cancelled"
		wait.UpdatedAt = now
		r.clearWaitReferenceLocked(wait.ID)
	}

	for index := range run.ToolCalls {
		call := &run.ToolCalls[index]
		switch call.State {
		case ToolCallStatePending,
			ToolCallStateEvaluationQueued,
			ToolCallStateEvaluating,
			ToolCallStateReady,
			ToolCallStateExecutionQueued,
			ToolCallStateWaiting:
			call.State = ToolCallStateCancelled
			call.Authorization = nil
			call.WaitID = ""
		default:
		}
	}

	run.RequestedCallIDs = nil
}

func hasActiveExternalWork(run *Run) bool {
	if run == nil {
		return false
	}

	for _, operation := range run.ModelOperations {
		if operation.State == OperationStateRunning {
			return true
		}
	}
	for _, call := range run.ToolCalls {
		if call.State == ToolCallStateRunning {
			return true
		}
	}
	return false
}

func (r *Runtime) releaseQueuedToolWorkLocked(
	sessionValue *Session,
	run *Run,
	call *ToolCall,
) bool {
	if sessionValue == nil || run == nil || call == nil {
		return false
	}

	switch call.State {
	case ToolCallStateEvaluationQueued:
		call.State = ToolCallStatePending
	case ToolCallStateExecutionQueued:
		call.State = ToolCallStateReady
	default:
		return false
	}

	r.touchRunLocked(sessionValue, run)
	return true
}

// prepareConfiguration applies the same authoritative normalization boundary
// used for submitted configuration without admitting user input or attachments.
// Session creation and Session-configuration updates must not persist a raw
// configuration that differs from the configuration later used by a Run.
func (r *Runtime) prepareConfiguration(
	ctx context.Context,
	sessionID SessionID,
	configuration SessionConfiguration,
) (SessionConfiguration, error) {
	prepared, err := r.prepareSubmission(
		ctx,
		sessionID,
		configuration,
		UserInput{},
		"",
		true,
	)
	if err != nil {
		return SessionConfiguration{}, err
	}
	return cloneJSON(prepared.Configuration), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeRecoveredRun(
	run *Run,
	now time.Time,
) bool {
	if run == nil {
		return false
	}
	changed := false
	if run.SkillSession.ID != "" ||
		run.SkillSession.EstablishedGeneration != 0 {
		run.SkillSession.ID = ""
		run.SkillSession.EstablishedGeneration = 0
		run.SkillSession.Generation++
		changed = true
	}

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
			changed = true
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
				changed = true
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
			changed = true
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
		changed = true
	}

	if !run.State.Terminal() &&
		(run.State == RunStateQueued ||
			run.State == RunStateRunning ||
			run.State == RunStateCancelling ||
			requiresRecoveryReview) {
		if run.State != RunStateWaiting || run.Outcome != nil {
			changed = true
		}
		run.State = RunStateWaiting
		run.Outcome = nil
		if ensureRecoveryWait(run, now) {
			changed = true
		}
	}

	if run.ActiveModelOperationID != "" {
		run.ActiveModelOperationID = ""
		changed = true
	}

	paused := run.State == RunStatePaused
	if run.Controls.Paused != paused {
		run.Controls.Paused = paused
		changed = true
	}
	if run.Controls.CancellationRequested {
		run.Controls.CancellationRequested = false
		changed = true
	}

	if changed {
		run.Revision++
		run.UpdatedAt = now
	}
	return changed
}

func ensureRecoveryWait(
	run *Run,
	now time.Time,
) bool {
	for _, wait := range run.WaitingRequests {
		if wait.Kind == WaitKindRecovery &&
			(wait.Status == WaitStatusPending ||
				wait.Status == WaitStatusResolving) {
			return false
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
	return true
}

func isRunDispatchCancelKey(
	key string,
	runID RunID,
) bool {
	run := string(runID) + ":"
	return strings.HasPrefix(key, "model:"+run) ||
		strings.HasPrefix(key, "tool-evaluation:"+run) ||
		strings.HasPrefix(key, "tool-execution:"+run) ||
		strings.HasPrefix(key, "wait-resolution:"+run)
}
