package sessionv3

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	maximumTriggerOccurrences                              = 65_536
	maximumTriggerCalendarSearch                           = 3_200_000
	triggerTimerOperation                                  = "session-v3 trigger timer"
	triggerEventSourceSchedule                             = "schedule"
	triggerOccurrenceStateAccepted  TriggerOccurrenceState = "accepted"
	triggerOccurrenceStateStarted   TriggerOccurrenceState = "started"
	triggerOccurrenceStateSignalled TriggerOccurrenceState = "signalled"
	triggerOccurrenceStateQueued    TriggerOccurrenceState = "queued"
	triggerOccurrenceStateSkipped   TriggerOccurrenceState = "skipped"
	triggerOccurrenceStateReplaced  TriggerOccurrenceState = "replaced"
	triggerOccurrenceStateFailed    TriggerOccurrenceState = "failed"
)

type TriggerRegistrationState string

const (
	TriggerRegistrationStateActive   TriggerRegistrationState = "active"
	TriggerRegistrationStateDisabled TriggerRegistrationState = "disabled"
)

type TriggerOccurrenceState string

type TriggerRegistration struct {
	ID                  TriggerID                `json:"id"`
	Workflow            ArtifactPin              `json:"workflow"`
	Program             ProgramSnapshot          `json:"program"`
	TriggerName         string                   `json:"triggerName"`
	Kind                WorkflowTriggerKind      `json:"kind"`
	State               TriggerRegistrationState `json:"state"`
	NextOccurrence      *time.Time               `json:"nextOccurrence,omitempty"`
	LastOccurrence      *time.Time               `json:"lastOccurrence,omitempty"`
	LastLocalOccurrence string                   `json:"lastLocalOccurrence,omitempty"`
	Occurrences         []TriggerOccurrence      `json:"occurrences,omitempty"`
	Queue               []TriggerOccurrenceID    `json:"queue,omitempty"`
	CreatedAt           time.Time                `json:"createdAt"`
	UpdatedAt           time.Time                `json:"updatedAt"`
}

type TriggerOccurrence struct {
	ID            TriggerOccurrenceID    `json:"id"`
	Source        string                 `json:"source"`
	SourceID      string                 `json:"sourceID"`
	OccurredAt    time.Time              `json:"occurredAt"`
	AcceptedAt    time.Time              `json:"acceptedAt"`
	Data          CanonicalValue         `json:"data,omitempty"`
	Correlation   CanonicalValue         `json:"correlation,omitempty"`
	Input         CanonicalValue         `json:"input"`
	InstanceKey   string                 `json:"instanceKey,omitempty"`
	State         TriggerOccurrenceState `json:"state"`
	ExecutionID   ExecutionID            `json:"executionID,omitempty"`
	QueuePosition int                    `json:"queuePosition,omitempty"`
	Error         *SanitizedError        `json:"error,omitempty"`
}

// TriggerExecution links an Execution to the trigger occurrence that started
// it. The link is durable and lets signal and schedule overlap behavior select
// the exact current instance without a separate invocation store.
type TriggerExecution struct {
	RegistrationID TriggerID           `json:"registrationID"`
	OccurrenceID   TriggerOccurrenceID `json:"occurrenceID"`
	InstanceKey    string              `json:"instanceKey,omitempty"`
	OccurredAt     time.Time           `json:"occurredAt"`
}

func (r TriggerRegistration) Validate() error {
	if err := validateID("trigger registration ID", string(r.ID)); err != nil {
		return err
	}
	if err := validatePin(r.Workflow); err != nil {
		return err
	}
	if r.Program.Artifact == nil || *r.Program.Artifact != r.Workflow {
		return errors.New("trigger registration program pin differs from workflow pin")
	}
	if err := validateID("trigger registration name", r.TriggerName); err != nil {
		return err
	}
	switch r.Kind {
	case WorkflowTriggerKindEvent, WorkflowTriggerKindSchedule:
	default:
		return errors.New("trigger registration kind is invalid")
	}
	switch r.State {
	case TriggerRegistrationStateActive, TriggerRegistrationStateDisabled:
	default:
		return errors.New("trigger registration state is invalid")
	}
	if r.CreatedAt.IsZero() || r.UpdatedAt.IsZero() {
		return errors.New("trigger registration timestamps are required")
	}
	if r.NextOccurrence != nil && r.Kind != WorkflowTriggerKindSchedule {
		return errors.New("event trigger registration has a schedule occurrence")
	}
	if r.LastOccurrence != nil && r.LastOccurrence.IsZero() {
		return errors.New("trigger registration last occurrence is invalid")
	}

	program, err := workflowProgramFromSnapshot(r.Program)
	if err != nil {
		return err
	}
	trigger, found := workflowTriggerForName(program, r.TriggerName)
	if !found || trigger.Kind() != r.Kind {
		return errors.New("trigger registration declaration is unavailable")
	}

	occurrences := make(map[TriggerOccurrenceID]struct{}, len(r.Occurrences))
	for _, occurrence := range r.Occurrences {
		if err := occurrence.Validate(); err != nil {
			return err
		}
		if _, duplicate := occurrences[occurrence.ID]; duplicate {
			return errors.New("trigger occurrence identity is duplicated")
		}
		occurrences[occurrence.ID] = struct{}{}
	}
	queued := make(map[TriggerOccurrenceID]struct{}, len(r.Queue))
	for _, id := range r.Queue {
		if _, duplicate := queued[id]; duplicate {
			return errors.New("trigger queue identity is duplicated")
		}
		queued[id] = struct{}{}
		occurrence, found := triggerOccurrenceByID(r, id)
		if !found || occurrence.State != triggerOccurrenceStateQueued {
			return errors.New("trigger queue contains a non-queued occurrence")
		}
	}
	return nil
}

func (o TriggerOccurrence) Validate() error {
	if err := validateID("trigger occurrence ID", string(o.ID)); err != nil {
		return err
	}
	if err := validateID("trigger occurrence source", o.Source); err != nil {
		return err
	}
	if err := validateID("trigger occurrence source ID", o.SourceID); err != nil {
		return err
	}
	if o.OccurredAt.IsZero() || o.AcceptedAt.IsZero() {
		return errors.New("trigger occurrence timestamps are required")
	}
	if !o.Data.Valid() || !o.Correlation.Valid() || !o.Input.Valid() {
		return errors.New("trigger occurrence values are not canonical JSON")
	}
	switch o.State {
	case triggerOccurrenceStateAccepted,
		triggerOccurrenceStateStarted,
		triggerOccurrenceStateSignalled,
		triggerOccurrenceStateQueued,
		triggerOccurrenceStateSkipped,
		triggerOccurrenceStateReplaced,
		triggerOccurrenceStateFailed:
	default:
		return errors.New("trigger occurrence state is invalid")
	}
	if o.ExecutionID != "" {
		if err := validateID("trigger occurrence execution ID", string(o.ExecutionID)); err != nil {
			return err
		}
	}
	if o.Error != nil {
		if !o.Error.Details.Valid() {
			return errors.New("trigger occurrence error details are invalid")
		}
		if err := o.Error.Effect.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (t TriggerExecution) Validate() error {
	if err := validateID("trigger execution registration ID", string(t.RegistrationID)); err != nil {
		return err
	}
	if err := validateID("trigger execution occurrence ID", string(t.OccurrenceID)); err != nil {
		return err
	}
	if t.OccurredAt.IsZero() {
		return errors.New("trigger execution occurrence time is required")
	}
	return nil
}

func validateTriggerRegistrations(
	session Session,
	executionIDs map[ExecutionID]struct{},
) error {
	registrations := make(map[TriggerID]TriggerRegistration, len(session.Triggers))
	for _, registration := range session.Triggers {
		if err := registration.Validate(); err != nil {
			return err
		}
		if _, duplicate := registrations[registration.ID]; duplicate {
			return errors.New("trigger registration identity is duplicated")
		}
		registrations[registration.ID] = registration
	}

	for _, execution := range session.Executions {
		if execution.Trigger == nil {
			continue
		}
		registration, found := registrations[execution.Trigger.RegistrationID]
		if !found {
			return errors.New("execution trigger registration is unavailable")
		}
		occurrence, found := triggerOccurrenceByID(
			registration,
			execution.Trigger.OccurrenceID,
		)
		if !found || occurrence.ExecutionID != execution.ID {
			return errors.New("execution trigger occurrence is unavailable")
		}
		if _, found := executionIDs[occurrence.ExecutionID]; !found {
			return errors.New("trigger occurrence execution is unavailable")
		}
	}
	return nil
}

func triggerOccurrenceByID(
	registration TriggerRegistration,
	id TriggerOccurrenceID,
) (TriggerOccurrence, bool) {
	for _, occurrence := range registration.Occurrences {
		if occurrence.ID == id {
			return occurrence, true
		}
	}
	return TriggerOccurrence{}, false
}

func mutableTriggerRegistration(
	session *Session,
	id TriggerID,
) *TriggerRegistration {
	if session == nil {
		return nil
	}
	for index := range session.Triggers {
		if session.Triggers[index].ID == id {
			return &session.Triggers[index]
		}
	}
	return nil
}

func mutableTriggerOccurrence(
	registration *TriggerRegistration,
	id TriggerOccurrenceID,
) *TriggerOccurrence {
	if registration == nil {
		return nil
	}
	for index := range registration.Occurrences {
		if registration.Occurrences[index].ID == id {
			return &registration.Occurrences[index]
		}
	}
	return nil
}

func triggerTimerKey(sessionID SessionID, triggerID TriggerID) string {
	return string(sessionID) + "\x00" + string(triggerID)
}

func (r *Runtime) cancelTriggerLocked(
	sessionID SessionID,
	triggerID TriggerID,
) {
	key := triggerTimerKey(sessionID, triggerID)
	if cancel, found := r.triggerTimers[key]; found {
		delete(r.triggerTimers, key)
		cancel()
	}
}

func workflowTriggerRegistrationID(
	program ProgramSnapshot,
	name string,
) (TriggerID, error) {
	if program.Artifact == nil {
		return "", errors.New("workflow trigger program has no artifact pin")
	}
	value := TriggerID(strings.Join([]string{
		"workflow-trigger",
		program.Artifact.RootID,
		program.Artifact.ArtifactID,
		program.Artifact.Digest,
		name,
	}, ":"))
	if err := validateID("trigger registration ID", string(value)); err != nil {
		return "", err
	}
	return value, nil
}

func (r *Runtime) registerWorkflowTriggersLocked(
	session *Session,
	program ProgramSnapshot,
) error {
	if session == nil {
		return errors.New("trigger registration session is unavailable")
	}
	workflow, err := workflowProgramFromSnapshot(program)
	if err != nil {
		return err
	}
	if len(workflow.Triggers) == 0 {
		return nil
	}
	now := r.now()
	for _, trigger := range workflow.Triggers {
		id, idErr := workflowTriggerRegistrationID(program, trigger.Name)
		if idErr != nil {
			return idErr
		}
		existing := mutableTriggerRegistration(session, id)
		if existing != nil {
			if existing.Workflow != *program.Artifact {
				return errors.New("workflow trigger registration pin changed")
			}
			continue
		}
		registration := TriggerRegistration{
			ID:          id,
			Workflow:    *program.Artifact,
			Program:     cloneTriggerProgram(program),
			TriggerName: trigger.Name,
			Kind:        trigger.Kind(),
			State:       TriggerRegistrationStateActive,
			Occurrences: []TriggerOccurrence{},
			Queue:       []TriggerOccurrenceID{},
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if trigger.Schedule != nil {
			next, _, nextErr := workflowTriggerNextOccurrence(
				*trigger.Schedule,
				now,
				"",
			)
			if nextErr != nil {
				return nextErr
			}
			registration.NextOccurrence = &next
		}
		if err := registration.Validate(); err != nil {
			return err
		}
		session.Triggers = append(session.Triggers, registration)
	}
	return nil
}

func cloneTriggerProgram(value ProgramSnapshot) ProgramSnapshot {
	output := value
	output.Payload = value.Payload.Clone()
	if value.Artifact != nil {
		pin := *value.Artifact
		output.Artifact = &pin
	}
	return output
}

func (r *Runtime) armSessionTriggersLocked(
	ctx context.Context,
	session *Session,
) {
	if r == nil || session == nil || r.closed {
		return
	}
	for index := range session.Triggers {
		r.armTriggerLocked(ctx, session.ID, session.Triggers[index])
	}
}

func (r *Runtime) armTriggerLocked(
	ctx context.Context,
	sessionID SessionID,
	registration TriggerRegistration,
) {
	if r == nil ||
		registration.Kind != WorkflowTriggerKindSchedule ||
		registration.State != TriggerRegistrationStateActive ||
		registration.NextOccurrence == nil ||
		registration.ID == "" {
		return
	}
	key := triggerTimerKey(sessionID, registration.ID)
	if r.triggerTimers == nil {
		r.triggerTimers = make(map[string]context.CancelFunc)
	}
	if _, found := r.triggerTimers[key]; found {
		return
	}
	timerCtx, cancel := context.WithCancel(r.lifecycleCtx)
	r.triggerTimers[key] = cancel

	at := registration.NextOccurrence.UTC()
	r.launchAsync(
		triggerTimerOperation,
		func() {
			r.waitForTrigger(
				timerCtx,
				context.WithoutCancel(ctx),
				sessionID,
				registration.ID,
				at,
			)
		},
		nil,
	)
}

func (r *Runtime) waitForTrigger(
	ctx context.Context,
	persistContext context.Context,
	sessionID SessionID,
	triggerID TriggerID,
	at time.Time,
) {
	delay := at.Sub(r.now())
	timer := time.NewTimer(max(delay, 0))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	session := r.sessions[sessionID]
	if session == nil {
		r.mu.Unlock()
		return
	}
	registration := mutableTriggerRegistration(session, triggerID)
	if registration == nil ||
		registration.State != TriggerRegistrationStateActive ||
		registration.NextOccurrence == nil ||
		!registration.NextOccurrence.Equal(at) {
		r.mu.Unlock()
		return
	}
	delete(r.triggerTimers, triggerTimerKey(sessionID, triggerID))
	ready, changed, err := r.fireScheduledTriggerLocked(
		persistContext,
		session,
		registration,
		at,
	)
	if err != nil {
		r.mu.Unlock()
		return
	}
	if changed {
		r.touchSessionLocked(session)
		if err := r.persistLocked(persistContext, "triggerFired", session); err != nil {
			r.mu.Unlock()
			return
		}
	}
	r.armSessionTriggersLocked(persistContext, session)
	r.publishLocked(
		persistContext,
		session,
		"",
		RuntimeEventExecutionChanged,
	)
	r.mu.Unlock()
	r.launchReadyWork(persistContext, sessionID, ready)
}

func (r *Runtime) fireScheduledTriggerLocked(
	ctx context.Context,
	session *Session,
	registration *TriggerRegistration,
	at time.Time,
) ([]readyExecutionWork, bool, error) {
	if session == nil || registration == nil {
		return nil, false, errors.New("scheduled trigger owner is unavailable")
	}
	trigger, err := workflowTriggerFromRegistration(*registration)
	if err != nil {
		return nil, false, err
	}
	if trigger.Schedule == nil {
		return nil, false, errors.New("scheduled trigger declaration is unavailable")
	}

	evaluation, evaluateErr := PrepareWorkflowTrigger(
		registration.Program,
		registration.TriggerName,
		nil,
		at,
	)
	occurrence := TriggerOccurrence{
		ID:          scheduledTriggerOccurrenceID(registration.ID, at),
		Source:      triggerEventSourceSchedule,
		SourceID:    at.UTC().Format(time.RFC3339Nano),
		OccurredAt:  at.UTC(),
		AcceptedAt:  r.now(),
		Data:        MustCanonicalValue(nil),
		Correlation: MustCanonicalValue(nil),
		Input:       MustCanonicalValue(nil),
		State:       triggerOccurrenceStateAccepted,
	}
	if evaluateErr != nil {
		occurrence.State = triggerOccurrenceStateFailed
		occurrence.Error = &SanitizedError{
			Code:     "trigger_input_invalid",
			Category: ErrorCategoryValidation,
			Message:  evaluateErr.Error(),
		}
	} else {
		occurrence.Input = evaluation.Input.Clone()
		occurrence.InstanceKey = evaluation.InstanceKey
	}

	next, local, nextErr := workflowTriggerNextOccurrence(
		*trigger.Schedule,
		at,
		registration.LastLocalOccurrence,
	)
	if nextErr != nil {
		return nil, false, nextErr
	}
	registration.LastOccurrence = cloneTime(&at)
	registration.LastLocalOccurrence = local
	registration.NextOccurrence = &next
	registration.UpdatedAt = r.now()

	if existing := mutableTriggerOccurrence(registration, occurrence.ID); existing != nil {
		return nil, true, nil
	}
	if err := appendTriggerOccurrence(registration, occurrence); err != nil {
		return nil, false, err
	}
	stored := mutableTriggerOccurrence(registration, occurrence.ID)
	if stored == nil {
		return nil, false, errors.New("scheduled trigger occurrence disappeared")
	}
	if evaluateErr != nil {
		return r.readyWorkForSessionLocked(session), true, nil
	}
	if err := r.deliverScheduledTriggerLocked(ctx, session, registration, stored); err != nil {
		stored.State = triggerOccurrenceStateFailed
		stored.Error = &SanitizedError{
			Code:     "trigger_dispatch_failed",
			Category: ErrorCategoryExecution,
			Message:  err.Error(),
		}
	}
	return r.readyWorkForSessionLocked(session), true, nil
}

func scheduledTriggerOccurrenceID(
	triggerID TriggerID,
	at time.Time,
) TriggerOccurrenceID {
	return TriggerOccurrenceID(
		string(triggerID) + ":schedule:" + at.UTC().Format(time.RFC3339Nano),
	)
}

func workflowTriggerFromRegistration(
	registration TriggerRegistration,
) (WorkflowTrigger, error) {
	program, err := workflowProgramFromSnapshot(registration.Program)
	if err != nil {
		return WorkflowTrigger{}, err
	}
	trigger, found := workflowTriggerForName(program, registration.TriggerName)
	if !found {
		return WorkflowTrigger{}, errors.New("trigger declaration is unavailable")
	}
	if trigger.Kind() != registration.Kind {
		return WorkflowTrigger{}, errors.New("trigger declaration kind changed")
	}
	return trigger, nil
}

func appendTriggerOccurrence(
	registration *TriggerRegistration,
	occurrence TriggerOccurrence,
) error {
	if registration == nil {
		return errors.New("trigger registration is unavailable")
	}
	if len(registration.Occurrences) >= maximumTriggerOccurrences {
		return errors.New("trigger occurrence retention limit exceeded")
	}
	if err := occurrence.Validate(); err != nil {
		return err
	}
	registration.Occurrences = append(registration.Occurrences, occurrence)
	return nil
}

func (r *Runtime) acceptTriggerEvent(
	ctx context.Context,
	command AcceptTriggerEventCommand,
) (*CommandReceipt, error) {
	if command.Meta.ID == "" ||
		strings.TrimSpace(command.Source) == "" ||
		strings.TrimSpace(command.EventID) == "" ||
		strings.TrimSpace(command.Name) == "" ||
		!command.Data.Valid() ||
		!command.Correlation.Valid() {
		return nil, errors.New("invalid trigger event command")
	}
	if command.OccurredAt.IsZero() {
		command.OccurredAt = r.now()
	} else {
		command.OccurredAt = command.OccurredAt.UTC()
	}

	return r.changeSession(
		ctx,
		command.Meta,
		command.SessionID,
		command,
		func(session *Session) (*CommandReceipt, error) {
			event := TriggerEventEnvelope{
				ID:          command.EventID,
				Name:        command.Name,
				Source:      command.Source,
				Data:        command.Data.Clone(),
				Correlation: command.Correlation.Clone(),
				Time:        command.OccurredAt,
			}
			for index := range session.Triggers {
				registration := &session.Triggers[index]
				if registration.State != TriggerRegistrationStateActive ||
					registration.Kind != WorkflowTriggerKindEvent {
					continue
				}
				trigger, err := workflowTriggerFromRegistration(*registration)
				if err != nil {
					return nil, err
				}
				if trigger.Event == nil || trigger.Event.Event != event.Name {
					continue
				}
				if triggerOccurrenceBySource(*registration, event.Source, event.ID) {
					continue
				}
				program, err := workflowProgramFromSnapshot(registration.Program)
				if err != nil {
					return nil, err
				}
				contract, found := workflowEventForName(
					program,
					trigger.Event.Event,
				)
				if !found {
					return nil, errors.New("trigger event contract is unavailable")
				}
				if contract.Source != event.Source {
					continue
				}
				if err := validateWorkflowSchema(contract.Schema, event.Data); err != nil {
					return nil, fmt.Errorf("validate trigger event data: %w", err)
				}
				if err := validateWorkflowSchema(
					contract.CorrelationSchema,
					event.Correlation,
				); err != nil {
					return nil, fmt.Errorf("validate trigger event correlation: %w", err)
				}
				evaluation, err := PrepareWorkflowTrigger(
					registration.Program,
					registration.TriggerName,
					&event,
					event.Time,
				)
				if err != nil {
					return nil, err
				}
				if !evaluation.Matched {
					continue
				}
				occurrence := TriggerOccurrence{
					ID: TriggerOccurrenceID(
						string(registration.ID) + ":event:" +
							event.Source + ":" + event.ID,
					),
					Source:      event.Source,
					SourceID:    event.ID,
					OccurredAt:  event.Time,
					AcceptedAt:  r.now(),
					Data:        event.Data.Clone(),
					Correlation: event.Correlation.Clone(),
					Input:       evaluation.Input.Clone(),
					InstanceKey: evaluation.InstanceKey,
					State:       triggerOccurrenceStateAccepted,
				}
				if err := appendTriggerOccurrence(registration, occurrence); err != nil {
					return nil, err
				}
				stored := mutableTriggerOccurrence(registration, occurrence.ID)
				if stored == nil {
					return nil, errors.New("accepted trigger occurrence disappeared")
				}
				if err := r.deliverEventTriggerLocked(
					ctx,
					session,
					registration,
					trigger,
					stored,
					event,
				); err != nil {
					stored.State = triggerOccurrenceStateFailed
					stored.Error = &SanitizedError{
						Code:     "trigger_dispatch_failed",
						Category: ErrorCategoryExecution,
						Message:  err.Error(),
					}
				}
				registration.UpdatedAt = r.now()
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
}

func triggerOccurrenceBySource(
	registration TriggerRegistration,
	source string,
	sourceID string,
) bool {
	for _, occurrence := range registration.Occurrences {
		if occurrence.Source == source && occurrence.SourceID == sourceID {
			return true
		}
	}
	return false
}

func (r *Runtime) deliverEventTriggerLocked(
	ctx context.Context,
	session *Session,
	registration *TriggerRegistration,
	trigger WorkflowTrigger,
	occurrence *TriggerOccurrence,
	event TriggerEventEnvelope,
) error {
	if session == nil || registration == nil || occurrence == nil ||
		trigger.Event == nil {
		return errors.New("event trigger delivery is unavailable")
	}
	behavior := trigger.Event.Behavior
	if behavior == "" {
		behavior = WorkflowTriggerBehaviorStart
	}
	active := activeTriggerExecutions(
		*session,
		registration.ID,
		occurrence.InstanceKey,
	)

	switch behavior {
	case WorkflowTriggerBehaviorStart:
		return r.startTriggerExecutionLocked(
			session,
			registration,
			occurrence,
		)

	case WorkflowTriggerBehaviorSignal:
		if len(active) == 0 {
			occurrence.State = triggerOccurrenceStateSkipped
			occurrence.Error = &SanitizedError{
				Code:     "trigger_signal_target_missing",
				Category: ErrorCategoryRouting,
				Message:  "event trigger signal has no active execution",
			}
			return nil
		}
		target := mutableSessionExecution(session, active[0])
		if target == nil {
			return errors.New("event trigger signal execution is unavailable")
		}
		if err := r.signalTriggerExecutionLocked(
			session,
			target,
			event,
		); err != nil {
			return err
		}
		occurrence.ExecutionID = target.ID
		occurrence.State = triggerOccurrenceStateSignalled
		return nil

	case WorkflowTriggerBehaviorStartOrSignal:
		if len(active) == 0 {
			return r.startTriggerExecutionLocked(
				session,
				registration,
				occurrence,
			)
		}
		target := mutableSessionExecution(session, active[0])
		if target == nil {
			return errors.New("event trigger signal execution is unavailable")
		}
		if err := r.signalTriggerExecutionLocked(
			session,
			target,
			event,
		); err != nil {
			return err
		}
		occurrence.ExecutionID = target.ID
		occurrence.State = triggerOccurrenceStateSignalled
		return nil

	default:
		return errors.New("event trigger behavior is invalid")
	}
}

func (r *Runtime) deliverScheduledTriggerLocked(
	ctx context.Context,
	session *Session,
	registration *TriggerRegistration,
	occurrence *TriggerOccurrence,
) error {
	if session == nil || registration == nil || occurrence == nil {
		return errors.New("scheduled trigger delivery is unavailable")
	}
	trigger, err := workflowTriggerFromRegistration(*registration)
	if err != nil {
		return err
	}
	if trigger.Schedule == nil {
		return errors.New("scheduled trigger declaration is unavailable")
	}
	active := activeTriggerExecutions(
		*session,
		registration.ID,
		occurrence.InstanceKey,
	)
	overlap := trigger.Schedule.Overlap
	if overlap == "" {
		overlap = WorkflowTriggerOverlapAllow
	}
	switch overlap {
	case WorkflowTriggerOverlapAllow:
		return r.startTriggerExecutionLocked(session, registration, occurrence)

	case WorkflowTriggerOverlapSkip:
		if len(active) != 0 {
			occurrence.State = triggerOccurrenceStateSkipped
			return nil
		}
		return r.startTriggerExecutionLocked(session, registration, occurrence)

	case WorkflowTriggerOverlapQueue:
		if len(active) != 0 {
			occurrence.State = triggerOccurrenceStateQueued
			occurrence.QueuePosition = len(registration.Queue) + 1
			registration.Queue = append(registration.Queue, occurrence.ID)
			return nil
		}
		return r.startTriggerExecutionLocked(session, registration, occurrence)

	case WorkflowTriggerOverlapReplace:
		for _, executionID := range active {
			execution := mutableSessionExecution(session, executionID)
			if execution == nil {
				continue
			}
			r.cancelIndependentExecutionLocked(
				ctx,
				session,
				execution,
				"trigger occurrence replaced an active execution",
			)
		}
		return r.startTriggerExecutionLocked(session, registration, occurrence)

	default:
		return errors.New("scheduled trigger overlap policy is invalid")
	}
}

func activeTriggerExecutions(
	session Session,
	registrationID TriggerID,
	instanceKey string,
) []ExecutionID {
	values := make([]Execution, 0)
	for _, execution := range session.Executions {
		if execution.State.Terminal() ||
			execution.Trigger == nil ||
			execution.Trigger.RegistrationID != registrationID ||
			execution.Trigger.InstanceKey != instanceKey {
			continue
		}
		values = append(values, execution)
	}
	for left := 0; left < len(values); left++ {
		for right := left + 1; right < len(values); right++ {
			if values[right].CreatedAt.After(values[left].CreatedAt) {
				values[left], values[right] = values[right], values[left]
			}
		}
	}
	output := make([]ExecutionID, len(values))
	for index := range values {
		output[index] = values[index].ID
	}
	return output
}

func (r *Runtime) startTriggerExecutionLocked(
	session *Session,
	registration *TriggerRegistration,
	occurrence *TriggerOccurrence,
) error {
	if session == nil || registration == nil || occurrence == nil {
		return errors.New("trigger execution owner is unavailable")
	}
	program, err := bindWorkflowTriggerInput(
		registration.Program,
		occurrence.Input,
	)
	if err != nil {
		return err
	}
	workflow, err := workflowProgramFromSnapshot(program)
	if err != nil {
		return err
	}
	now := r.now()
	budget, deadline, err := initialExecutionBudget(workflow.Limits, now)
	if err != nil {
		return err
	}
	resources, err := initialExecutionResources(workflow.Resources)
	if err != nil {
		return err
	}

	interpreter := r.interpreterLocked(program)
	if interpreter == nil {
		return errors.New("workflow trigger interpreter is unavailable")
	}
	state, err := interpreter.InitialState(program)
	if err != nil {
		return err
	}
	executionID := ExecutionID(r.newID())
	frameID := ProgramFrameID(r.newID())
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
		Trigger: &TriggerExecution{
			RegistrationID: registration.ID,
			OccurrenceID:   occurrence.ID,
			InstanceKey:    occurrence.InstanceKey,
			OccurredAt:     occurrence.OccurredAt,
		},
		ProgramFrames: []ProgramFrame{{
			ID:        frameID,
			Program:   program,
			State:     state,
			Status:    ProgramFrameStateRunning,
			Path:      "workflow",
			CreatedAt: now,
			UpdatedAt: now,
			Deadline:  cloneTime(deadline),
		}},
		WorkItems:      []WorkItem{},
		Attempts:       []Attempt{},
		Waits:          []Wait{},
		Resources:      resources,
		ResourceLeases: []ResourceLease{},
		ResourceRates:  []ResourceRateWindow{},
	}
	session.Executions = append(session.Executions, execution)
	current := &session.Executions[len(session.Executions)-1]
	occurrence.ExecutionID = executionID
	occurrence.State = triggerOccurrenceStateStarted
	session.Timeline = append(session.Timeline, Activity{
		ID:          ActivityID(r.newID()),
		Kind:        ActivityKindWorkflow,
		Visibility:  ActivityVisibilityBoth,
		CreatedAt:   now,
		ExecutionID: executionID,
		FrameID:     frameID,
		Workflow: &WorkflowActivity{
			Phase:   "triggered",
			Summary: "workflow trigger occurrence accepted",
			Output: MustCanonicalValue(struct {
				TriggerID    TriggerID           `json:"triggerID"`
				OccurrenceID TriggerOccurrenceID `json:"occurrenceID"`
			}{
				TriggerID:    registration.ID,
				OccurrenceID: occurrence.ID,
			}),
		},
	})
	if err := r.advanceFrameLocked(
		session,
		current,
		frameID,
		ProgramEvent{Kind: ProgramEventStarted},
	); err != nil {
		session.Executions = session.Executions[:len(session.Executions)-1]
		occurrence.ExecutionID = ""
		occurrence.State = triggerOccurrenceStateFailed
		return err
	}
	current = mutableSessionExecution(session, executionID)
	if current == nil {
		return errors.New("trigger execution disappeared after startup")
	}
	r.refreshExecutionStateLocked(current)
	return nil
}

func (r *Runtime) signalTriggerExecutionLocked(
	session *Session,
	execution *Execution,
	event TriggerEventEnvelope,
) error {
	if session == nil || execution == nil || execution.State.Terminal() {
		return errors.New("trigger signal execution is unavailable")
	}
	for _, existing := range execution.Signals {
		if existing.Source == event.Source && existing.ID == event.ID {
			return nil
		}
	}
	signal := ExecutionSignal{
		ID:          event.ID,
		Name:        event.Name,
		Source:      event.Source,
		Data:        event.Data.Clone(),
		Correlation: event.Correlation.Clone(),
		CreatedAt:   event.Time,
	}
	execution.Signals = append(execution.Signals, signal)
	if err := r.acceptSignalForSubscriptionsLocked(
		execution,
		signal,
	); err != nil {
		return err
	}
	if err := r.wakeEventWaitsLocked(session, execution, signal); err != nil {
		return err
	}
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
				Phase:   "trigger-signal",
				Summary: event.Name,
				Output: MustCanonicalValue(struct {
					Source      string         `json:"source"`
					ID          string         `json:"id"`
					Data        CanonicalValue `json:"data"`
					Correlation CanonicalValue `json:"correlation,omitempty"`
				}{
					Source:      event.Source,
					ID:          event.ID,
					Data:        event.Data.Clone(),
					Correlation: event.Correlation.Clone(),
				}),
			},
		}
		session.Timeline = append(session.Timeline, activity)
		if err := r.advanceFrameLocked(
			session,
			execution,
			frame.ID,
			ProgramEvent{
				Kind:       ProgramEventActivity,
				ActivityID: activity.ID,
			},
		); err != nil {
			return err
		}
	}
	r.touchExecutionLocked(session, execution)
	return nil
}

func (r *Runtime) drainTriggerQueuesLocked(
	ctx context.Context,
	session *Session,
	terminalExecutionID ExecutionID,
) {
	_ = terminalExecutionID
	if session == nil || terminalExecutionID == "" {
		return
	}
	for index := range session.Triggers {
		registration := &session.Triggers[index]
		if len(registration.Queue) == 0 ||
			registration.State != TriggerRegistrationStateActive {
			continue
		}

		queued := append([]TriggerOccurrenceID(nil), registration.Queue...)
		registration.Queue = registration.Queue[:0]
		for _, nextID := range queued {
			occurrence := mutableTriggerOccurrence(registration, nextID)
			if occurrence == nil || occurrence.State != triggerOccurrenceStateQueued {
				continue
			}
			if len(activeTriggerExecutions(
				*session,
				registration.ID,
				occurrence.InstanceKey,
			)) != 0 {
				registration.Queue = append(registration.Queue, nextID)
				continue
			}
			if err := r.startTriggerExecutionLocked(
				session,
				registration,
				occurrence,
			); err != nil {
				occurrence.State = triggerOccurrenceStateFailed
				occurrence.Error = &SanitizedError{
					Code:     "trigger_queue_start_failed",
					Category: ErrorCategoryExecution,
					Message:  err.Error(),
				}
			}
			registration.UpdatedAt = r.now()
		}
		for queueIndex, occurrenceID := range registration.Queue {
			occurrence := mutableTriggerOccurrence(registration, occurrenceID)
			if occurrence != nil {
				occurrence.QueuePosition = queueIndex + 1
			}
		}
	}
}

func (r *Runtime) recoverSessionTriggersLocked(
	ctx context.Context,
	session *Session,
) (bool, error) {
	if session == nil {
		return false, nil
	}
	now := r.now()
	changed := false
	for index := range session.Triggers {
		registration := &session.Triggers[index]
		if registration.State != TriggerRegistrationStateActive ||
			registration.Kind != WorkflowTriggerKindSchedule ||
			registration.NextOccurrence == nil {
			continue
		}
		trigger, err := workflowTriggerFromRegistration(*registration)
		if err != nil {
			return changed, err
		}
		if trigger.Schedule == nil {
			return changed, errors.New("scheduled trigger declaration is unavailable")
		}
		mode := trigger.Schedule.Missed.Mode
		if mode == "" {
			mode = WorkflowTriggerMissedSkip
		}
		catchUpLimit := trigger.Schedule.Missed.MaxOccurrences
		if catchUpLimit <= 0 {
			catchUpLimit = 1
		}
		var latest *time.Time
		missed := 0
		for registration.NextOccurrence != nil &&
			!registration.NextOccurrence.After(now) {
			at := registration.NextOccurrence.UTC()
			next, local, nextErr := workflowTriggerNextOccurrence(
				*trigger.Schedule,
				at,
				registration.LastLocalOccurrence,
			)
			if nextErr != nil {
				return changed, nextErr
			}
			registration.LastOccurrence = &at
			registration.LastLocalOccurrence = local
			registration.NextOccurrence = &next
			registration.UpdatedAt = now
			changed = true
			switch mode {
			case WorkflowTriggerMissedSkip:
				occurrence := TriggerOccurrence{
					ID:          scheduledTriggerOccurrenceID(registration.ID, at),
					Source:      triggerEventSourceSchedule,
					SourceID:    at.Format(time.RFC3339Nano),
					OccurredAt:  at,
					AcceptedAt:  now,
					Data:        MustCanonicalValue(nil),
					Correlation: MustCanonicalValue(nil),
					Input:       MustCanonicalValue(nil),
					State:       triggerOccurrenceStateSkipped,
				}
				if mutableTriggerOccurrence(registration, occurrence.ID) == nil {
					if err := appendTriggerOccurrence(registration, occurrence); err != nil {
						return changed, err
					}
				}
			case WorkflowTriggerMissedLatest:
				value := at
				latest = &value
			case WorkflowTriggerMissedCatchUp:
				if missed < catchUpLimit {
					if _, _, fireErr := r.fireScheduledTriggerLocked(
						ctx,
						session,
						registration,
						at,
					); fireErr != nil {
						return changed, fireErr
					}
					missed++
				}
			default:
				return changed, errors.New("trigger missed-occurrence policy is invalid")
			}
		}
		if latest != nil {
			if _, _, fireErr := r.fireScheduledTriggerLocked(
				ctx,
				session,
				registration,
				*latest,
			); fireErr != nil {
				return changed, fireErr
			}
			changed = true
		}
	}
	return changed, nil
}

func workflowTriggerNextOccurrence(
	schedule WorkflowScheduleTrigger,
	after time.Time,
	lastLocal string,
) (time.Time, string, error) {
	if schedule.Interval != "" {
		next, err := schedule.Interval.Deadline(after.UTC())
		return next, "", err
	}
	cron, err := parseWorkflowCron(schedule.Calendar)
	if err != nil {
		return time.Time{}, "", err
	}
	location, err := time.LoadLocation(schedule.TimeZone)
	if err != nil {
		return time.Time{}, "", err
	}
	cursor := after.UTC().Truncate(time.Minute).Add(time.Minute)
	for range maximumTriggerCalendarSearch {
		local := cursor.In(location)
		if cron.matches(local) {
			localKey := local.Format("2006-01-02T15:04")
			if localKey != lastLocal {
				return cursor.UTC(), localKey, nil
			}
		}
		cursor = cursor.Add(time.Minute)
	}
	return time.Time{}, "", errors.New("workflow calendar trigger has no occurrence in the supported search horizon")
}

type workflowCronField struct {
	values []bool
	min    int
	max    int
}

type workflowCron struct {
	minute workflowCronField
	hour   workflowCronField
	day    workflowCronField
	month  workflowCronField
	week   workflowCronField
}

func parseWorkflowCron(value string) (workflowCron, error) {
	fields := strings.Fields(value)
	if len(fields) != 5 {
		return workflowCron{}, errors.New("workflow calendar trigger requires five cron fields")
	}
	minute, err := parseWorkflowCronField(fields[0], 0, 59, false)
	if err != nil {
		return workflowCron{}, err
	}
	hour, err := parseWorkflowCronField(fields[1], 0, 23, false)
	if err != nil {
		return workflowCron{}, err
	}
	day, err := parseWorkflowCronField(fields[2], 1, 31, false)
	if err != nil {
		return workflowCron{}, err
	}
	month, err := parseWorkflowCronField(fields[3], 1, 12, false)
	if err != nil {
		return workflowCron{}, err
	}
	week, err := parseWorkflowCronField(fields[4], 0, 7, true)
	if err != nil {
		return workflowCron{}, err
	}
	return workflowCron{
		minute: minute,
		hour:   hour,
		day:    day,
		month:  month,
		week:   week,
	}, nil
}

func parseWorkflowCronField(
	raw string,
	minimum int,
	maximum int,
	normalizeSunday bool,
) (workflowCronField, error) {
	output := workflowCronField{
		values: make([]bool, maximum-minimum+1),
		min:    minimum,
		max:    maximum,
	}
	if strings.TrimSpace(raw) == "" {
		return workflowCronField{}, errors.New("workflow calendar trigger has an empty cron field")
	}
	for part := range strings.SplitSeq(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return workflowCronField{}, errors.New("workflow calendar trigger has an empty cron item")
		}
		base := part
		stride := 1
		if left, right, found := strings.Cut(part, "/"); found {
			base = left
			value, err := strconv.Atoi(right)
			if err != nil || value <= 0 || value > maximum-minimum+1 {
				return workflowCronField{}, errors.New("workflow calendar trigger has an invalid cron stride")
			}
			stride = value
		}
		start := minimum
		end := maximum
		switch base {
		case "*":
		default:
			if left, right, found := strings.Cut(base, "-"); found {
				var err error
				start, err = strconv.Atoi(left)
				if err != nil {
					return workflowCronField{}, errors.New("workflow calendar trigger has an invalid cron range")
				}
				end, err = strconv.Atoi(right)
				if err != nil {
					return workflowCronField{}, errors.New("workflow calendar trigger has an invalid cron range")
				}
			} else {
				value, err := strconv.Atoi(base)
				if err != nil {
					return workflowCronField{}, errors.New("workflow calendar trigger has an invalid cron value")
				}
				start = value
				end = value
			}
		}
		if start < minimum || end > maximum || end < start {
			return workflowCronField{}, errors.New("workflow calendar trigger cron value is outside its field range")
		}
		for value := start; value <= end; value += stride {
			if normalizeSunday && value == 7 {
				output.values[0] = true
				continue
			}
			output.values[value-minimum] = true
		}
	}
	return output, nil
}

func (f workflowCronField) matches(value int) bool {
	if value < f.min || value > f.max {
		return false
	}
	if len(f.values) == 0 {
		return false
	}
	return f.values[value-f.min]
}

func (c workflowCron) matches(value time.Time) bool {
	return c.minute.matches(value.Minute()) &&
		c.hour.matches(value.Hour()) &&
		c.day.matches(value.Day()) &&
		c.month.matches(int(value.Month())) &&
		c.week.matches(int(value.Weekday()))
}
