package sessionv3

import (
	"errors"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"
)

// WorkflowTriggerKind identifies one declaration trigger form. It is kept
// separate from runtime occurrence state so the compiled program remains a
// stable typed declaration payload.
type WorkflowTriggerKind string

const (
	WorkflowTriggerKindEvent    WorkflowTriggerKind = "event"
	WorkflowTriggerKindSchedule WorkflowTriggerKind = "schedule"
)

type WorkflowTriggerBehavior string

const (
	WorkflowTriggerBehaviorStart         WorkflowTriggerBehavior = "start"
	WorkflowTriggerBehaviorSignal        WorkflowTriggerBehavior = "signal"
	WorkflowTriggerBehaviorStartOrSignal WorkflowTriggerBehavior = "start-or-signal"
)

type WorkflowTriggerOverlap string

const (
	WorkflowTriggerOverlapAllow   WorkflowTriggerOverlap = "allow"
	WorkflowTriggerOverlapSkip    WorkflowTriggerOverlap = "skip"
	WorkflowTriggerOverlapQueue   WorkflowTriggerOverlap = "queue"
	WorkflowTriggerOverlapReplace WorkflowTriggerOverlap = "replace"
)

type WorkflowTriggerMissedMode string

const (
	WorkflowTriggerMissedSkip    WorkflowTriggerMissedMode = "skip"
	WorkflowTriggerMissedLatest  WorkflowTriggerMissedMode = "latest"
	WorkflowTriggerMissedCatchUp WorkflowTriggerMissedMode = "catch-up"
)

// WorkflowTrigger is the compiled form of one Workflow v1 declaration
// trigger. Input and filter values remain typed templates. They are evaluated
// only when a durable trigger occurrence is accepted.
type WorkflowTrigger struct {
	Name     string                   `json:"name"`
	Event    *WorkflowEventTrigger    `json:"event,omitempty"`
	Schedule *WorkflowScheduleTrigger `json:"schedule,omitempty"`
}

type WorkflowEventTrigger struct {
	Event       string                  `json:"event"`
	Filter      *WorkflowValue          `json:"filter,omitempty"`
	Input       WorkflowValue           `json:"input"`
	InstanceKey *WorkflowValue          `json:"instanceKey,omitempty"`
	Behavior    WorkflowTriggerBehavior `json:"behavior,omitempty"`
}

type WorkflowScheduleTrigger struct {
	Calendar string                 `json:"calendar"`
	Interval WorkflowDuration       `json:"interval"`
	TimeZone string                 `json:"timeZone"`
	Input    WorkflowValue          `json:"input"`
	Overlap  WorkflowTriggerOverlap `json:"overlap"`
	Missed   WorkflowTriggerMissed  `json:"missed"`
}

type WorkflowTriggerMissed struct {
	Mode           WorkflowTriggerMissedMode `json:"mode,omitempty"`
	MaxOccurrences int                       `json:"maxOccurrences,omitempty"`
}

// TriggerEventEnvelope is the typed generic event boundary exposed to a
// trigger's `event` CEL scope. The Event declaration validates Data and
// Correlation before this envelope reaches an evaluated trigger template.
type TriggerEventEnvelope struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Source      string         `json:"source"`
	Data        CanonicalValue `json:"data"`
	Correlation CanonicalValue `json:"correlation,omitempty"`
	Time        time.Time      `json:"time"`
}

type WorkflowTriggerEvaluation struct {
	Input       CanonicalValue
	InstanceKey string
	Matched     bool
}

func (t WorkflowTrigger) Kind() WorkflowTriggerKind {
	switch {
	case t.Event != nil:
		return WorkflowTriggerKindEvent
	case t.Schedule != nil:
		return WorkflowTriggerKindSchedule
	default:
		return ""
	}
}

func (t WorkflowTrigger) Validate() error {
	if err := validateID("workflow trigger name", t.Name); err != nil {
		return err
	}
	if (t.Event == nil) == (t.Schedule == nil) {
		return errors.New("workflow trigger must contain exactly one trigger form")
	}
	if t.Event != nil {
		if err := t.Event.Validate(); err != nil {
			return err
		}
	}
	if t.Schedule != nil {
		if err := t.Schedule.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (t WorkflowEventTrigger) Validate() error {
	if err := validateID("workflow event trigger event", t.Event); err != nil {
		return err
	}
	if err := t.Input.Validate(); err != nil {
		return err
	}
	if t.Filter != nil {
		if err := t.Filter.Validate(); err != nil {
			return err
		}
	}
	if t.InstanceKey != nil {
		if err := t.InstanceKey.Validate(); err != nil {
			return err
		}
	}
	switch t.Behavior {
	case "",
		WorkflowTriggerBehaviorStart,
		WorkflowTriggerBehaviorSignal,
		WorkflowTriggerBehaviorStartOrSignal:
		return nil
	default:
		return errors.New("workflow event trigger behavior is invalid")
	}
}

func (t WorkflowScheduleTrigger) Validate() error {
	if (strings.TrimSpace(t.Calendar) == "") ==
		(t.Interval == "") {
		return errors.New("workflow schedule trigger requires exactly one schedule")
	}
	if t.Calendar != "" {
		if _, err := parseWorkflowCron(t.Calendar); err != nil {
			return err
		}
		if strings.TrimSpace(t.TimeZone) == "" {
			return errors.New("workflow calendar trigger requires an iana time zone")
		}
		if _, err := time.LoadLocation(t.TimeZone); err != nil {
			return fmt.Errorf("workflow calendar trigger time zone: %w", err)
		}
	} else {
		if err := validateWorkflowTimeout(t.Interval); err != nil {
			return err
		}
		if t.TimeZone != "" {
			return errors.New("workflow interval trigger cannot declare a time zone")
		}
	}
	if err := t.Input.Validate(); err != nil {
		return err
	}
	switch t.Overlap {
	case "",
		WorkflowTriggerOverlapAllow,
		WorkflowTriggerOverlapSkip,
		WorkflowTriggerOverlapQueue,
		WorkflowTriggerOverlapReplace:
	default:
		return errors.New("workflow trigger overlap policy is invalid")
	}
	return t.Missed.Validate()
}

func (m WorkflowTriggerMissed) Validate() error {
	switch m.Mode {
	case "",
		WorkflowTriggerMissedSkip,
		WorkflowTriggerMissedLatest,
		WorkflowTriggerMissedCatchUp:
	default:
		return errors.New("workflow trigger missed-occurrence policy is invalid")
	}
	if m.MaxOccurrences < 0 {
		return errors.New("workflow trigger missed-occurrence maximum is negative")
	}
	if m.Mode != WorkflowTriggerMissedCatchUp &&
		m.MaxOccurrences != 0 {
		return errors.New("workflow trigger missed-occurrence maximum requires catch-up")
	}
	if m.Mode == WorkflowTriggerMissedCatchUp &&
		m.MaxOccurrences <= 0 {
		return errors.New("workflow trigger catch-up requires a positive maximum")
	}
	return nil
}

func validateWorkflowTriggers(values []WorkflowTrigger) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return err
		}
		if _, duplicate := seen[value.Name]; duplicate {
			return errors.New("workflow trigger name is duplicated")
		}
		seen[value.Name] = struct{}{}
	}
	return nil
}

func workflowTriggerForName(
	program WorkflowProgram,
	name string,
) (WorkflowTrigger, bool) {
	for _, trigger := range program.Triggers {
		if trigger.Name == name {
			return trigger, true
		}
	}
	return WorkflowTrigger{}, false
}

// PrepareWorkflowTrigger evaluates a trigger only after durable occurrence
// identity has been accepted. It does not mutate the Session or dispatch
// external work.
func PrepareWorkflowTrigger(
	snapshot ProgramSnapshot,
	triggerName string,
	event *TriggerEventEnvelope,
	logicalTime time.Time,
) (WorkflowTriggerEvaluation, error) {
	program, err := workflowProgramFromSnapshot(snapshot)
	if err != nil {
		return WorkflowTriggerEvaluation{}, err
	}
	trigger, found := workflowTriggerForName(program, triggerName)
	if !found {
		return WorkflowTriggerEvaluation{}, errors.New("workflow trigger is unavailable")
	}
	if err := trigger.Validate(); err != nil {
		return WorkflowTriggerEvaluation{}, err
	}

	interpreter, err := NewWorkflowInterpreter()
	if err != nil {
		return WorkflowTriggerEvaluation{}, err
	}
	program.Input = MustCanonicalValue(nil)
	program.InputBound = false
	state := WorkflowState{
		Version:     WorkflowInterpreterVersion,
		State:       MustCanonicalValue(map[string]CanonicalValue{}),
		Outcomes:    []WorkflowStepOutcome{},
		TargetSets:  []WorkflowTargetSetState{},
		LogicalTime: logicalTime.UTC(),
	}

	var eventValue CanonicalValue
	if event != nil {
		if event.ID == "" ||
			event.Name == "" ||
			event.Source == "" ||
			!event.Data.Valid() ||
			!event.Correlation.Valid() ||
			event.Time.IsZero() {
			return WorkflowTriggerEvaluation{}, errors.New("workflow trigger event is invalid")
		}
		eventValue = MustCanonicalValue(*event)
	}

	result := WorkflowTriggerEvaluation{Matched: true}
	switch trigger.Kind() {
	case WorkflowTriggerKindEvent:
		if event == nil {
			return WorkflowTriggerEvaluation{}, errors.New("event trigger has no event")
		}
		if trigger.Event.Filter != nil {
			matched, conditionErr := interpreter.conditionWithEvent(
				*trigger.Event.Filter,
				program,
				state,
				eventValue,
			)
			if conditionErr != nil {
				return WorkflowTriggerEvaluation{}, conditionErr
			}
			if !matched {
				result.Matched = false
				return result, nil
			}
		}
		input, inputErr := interpreter.evaluateWithEvent(
			trigger.Event.Input,
			program,
			state,
			eventValue,
		)
		if inputErr != nil {
			return WorkflowTriggerEvaluation{}, inputErr
		}
		result.Input = input
		if trigger.Event.InstanceKey != nil {
			key, keyErr := interpreter.evaluateWithEvent(
				*trigger.Event.InstanceKey,
				program,
				state,
				eventValue,
			)
			if keyErr != nil {
				return WorkflowTriggerEvaluation{}, keyErr
			}
			result.InstanceKey, keyErr = DecodeCanonicalValue[string](key)
			if keyErr != nil {
				return WorkflowTriggerEvaluation{}, errors.New("workflow trigger instance key must be a string")
			}
		}

	case WorkflowTriggerKindSchedule:
		input, inputErr := interpreter.evaluate(
			trigger.Schedule.Input,
			program,
			state,
		)
		if inputErr != nil {
			return WorkflowTriggerEvaluation{}, inputErr
		}
		result.Input = input

	default:
		return WorkflowTriggerEvaluation{}, errors.New("workflow trigger kind is invalid")
	}

	if err := validateWorkflowSchema(program.InputSchema, result.Input); err != nil {
		return WorkflowTriggerEvaluation{}, fmt.Errorf("validate workflow trigger input: %w", err)
	}
	return result, nil
}

func bindWorkflowTriggerInput(
	snapshot ProgramSnapshot,
	input CanonicalValue,
) (ProgramSnapshot, error) {
	return workflowSnapshotWithInput(snapshot, input)
}
