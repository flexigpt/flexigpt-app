package sessionv3

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maximumWorkflowMachineMicrosteps = 1024

type WorkflowMachine struct {
	Initial string                 `json:"initial"`
	State   *WorkflowStateSpec     `json:"state,omitempty"`
	Listen  []WorkflowSubscription `json:"listen,omitempty"`
	States  []WorkflowMachineState `json:"states"`
}

type WorkflowMachineState struct {
	Name        string                      `json:"name"`
	Entry       *WorkflowBlock              `json:"entry,omitempty"`
	Exit        *WorkflowBlock              `json:"exit,omitempty"`
	Transitions []WorkflowMachineTransition `json:"transitions,omitempty"`
	Final       bool                        `json:"final,omitempty"`
	Output      *WorkflowValue              `json:"output,omitempty"`
}

type WorkflowMachineTransition struct {
	Event string           `json:"event,omitempty"`
	After WorkflowDuration `json:"after,omitempty"`
	When  *WorkflowValue   `json:"when,omitempty"`
	Do    *WorkflowBlock   `json:"do,omitempty"`
	To    string           `json:"to"`
}

type WorkflowMachinePhase string

const (
	WorkflowMachinePhaseEntry         WorkflowMachinePhase = "entry"
	WorkflowMachinePhaseEntryRunning  WorkflowMachinePhase = "entryRunning"
	WorkflowMachinePhaseSelect        WorkflowMachinePhase = "select"
	WorkflowMachinePhaseWaiting       WorkflowMachinePhase = "waiting"
	WorkflowMachinePhaseExit          WorkflowMachinePhase = "exit"
	WorkflowMachinePhaseExitRunning   WorkflowMachinePhase = "exitRunning"
	WorkflowMachinePhaseAction        WorkflowMachinePhase = "action"
	WorkflowMachinePhaseActionRunning WorkflowMachinePhase = "actionRunning"
)

type WorkflowMachineWait struct {
	Key               string     `json:"key"`
	WorkItemID        WorkItemID `json:"workItemID"`
	TransitionIndexes []int      `json:"transitionIndexes"`
}

type WorkflowMachineProgress struct {
	Current            string                `json:"current"`
	State              CanonicalValue        `json:"state"`
	LocalState         bool                  `json:"localState"`
	Phase              WorkflowMachinePhase  `json:"phase"`
	SelectedTransition int                   `json:"selectedTransition"`
	Generation         uint64                `json:"generation"`
	EnteredAt          time.Time             `json:"enteredAt"`
	Event              CanonicalValue        `json:"event,omitempty"`
	ChildPath          string                `json:"childPath,omitempty"`
	ChildMergesState   bool                  `json:"childMergesState,omitempty"`
	Waits              []WorkflowMachineWait `json:"waits,omitempty"`
	Subscriptions      []string              `json:"subscriptions,omitempty"`
	Microsteps         int                   `json:"microsteps"`
}

func (m WorkflowMachine) Validate() error {
	if err := validateID("workflow machine initial state", m.Initial); err != nil {
		return err
	}
	if m.State != nil {
		if len(m.State.Schema) == 0 ||
			!m.State.Schema.Valid() {
			return errors.New("workflow machine state schema is invalid")
		}
		if err := m.State.Initial.Validate(); err != nil {
			return err
		}
	}
	if len(m.States) == 0 {
		return errors.New("workflow machine has no states")
	}

	names := make(map[string]struct{}, len(m.States))
	for _, state := range m.States {
		if err := state.Validate(); err != nil {
			return err
		}
		if _, duplicate := names[state.Name]; duplicate {
			return fmt.Errorf(
				"workflow machine state %q is duplicated",
				state.Name,
			)
		}
		names[state.Name] = struct{}{}
	}
	if _, found := names[m.Initial]; !found {
		return fmt.Errorf(
			"workflow machine initial state %q is unavailable",
			m.Initial,
		)
	}

	subscriptions := make(map[string]struct{}, len(m.Listen))
	for _, subscription := range m.Listen {
		if err := subscription.Validate(); err != nil {
			return err
		}
		if _, duplicate := subscriptions[subscription.Name]; duplicate {
			return fmt.Errorf(
				"workflow machine subscription %q is duplicated",
				subscription.Name,
			)
		}
		subscriptions[subscription.Name] = struct{}{}
	}

	for _, state := range m.States {
		for _, transition := range state.Transitions {
			if _, found := names[transition.To]; !found {
				return fmt.Errorf(
					"workflow machine transition target %q is unavailable",
					transition.To,
				)
			}
			if transition.Event != "" {
				if _, found := subscriptions[transition.Event]; !found {
					return fmt.Errorf(
						"workflow machine transition subscription %q is unavailable",
						transition.Event,
					)
				}
			}
		}
	}
	return nil
}

func (s WorkflowMachineState) Validate() error {
	if err := validateID("workflow machine state name", s.Name); err != nil {
		return err
	}
	if s.Entry != nil {
		if err := s.Entry.Validate(); err != nil {
			return err
		}
	}
	if s.Exit != nil {
		if err := s.Exit.Validate(); err != nil {
			return err
		}
	}
	if s.Final {
		if s.Output == nil {
			return fmt.Errorf(
				"final workflow machine state %q has no output",
				s.Name,
			)
		}
		if s.Exit != nil || len(s.Transitions) != 0 {
			return fmt.Errorf(
				"final workflow machine state %q has exit or transitions",
				s.Name,
			)
		}
		return s.Output.Validate()
	}
	if s.Output != nil {
		return fmt.Errorf(
			"non-final workflow machine state %q has an output",
			s.Name,
		)
	}
	if len(s.Transitions) == 0 {
		return fmt.Errorf(
			"non-final workflow machine state %q has no transitions",
			s.Name,
		)
	}
	for index := range s.Transitions {
		if err := s.Transitions[index].Validate(); err != nil {
			return fmt.Errorf(
				"workflow machine transition %d: %w",
				index,
				err,
			)
		}
	}
	return nil
}

func (t WorkflowMachineTransition) Validate() error {
	if err := validateID("workflow machine transition target", t.To); err != nil {
		return err
	}
	if t.Event != "" {
		if err := validateID(
			"workflow machine transition event",
			t.Event,
		); err != nil {
			return err
		}
	}
	if t.After != "" {
		if err := validateWorkflowTimeout(t.After); err != nil {
			return err
		}
	}
	if t.Event != "" && t.After != "" {
		return errors.New(
			"workflow machine transition has event and timer",
		)
	}
	if t.Event == "" && t.After == "" && t.When == nil {
		return errors.New(
			"workflow machine transition has no event, timer, or condition",
		)
	}
	if t.When != nil {
		if err := t.When.Validate(); err != nil {
			return err
		}
	}
	if t.Do != nil {
		return t.Do.Validate()
	}
	return nil
}

func (p WorkflowMachineProgress) Validate() error {
	if err := validateID("workflow machine current state", p.Current); err != nil {
		return err
	}
	if !p.State.Valid() {
		return errors.New("workflow machine state is invalid")
	}
	switch p.Phase {
	case WorkflowMachinePhaseEntry,
		WorkflowMachinePhaseEntryRunning,
		WorkflowMachinePhaseSelect,
		WorkflowMachinePhaseWaiting,
		WorkflowMachinePhaseExit,
		WorkflowMachinePhaseExitRunning,
		WorkflowMachinePhaseAction,
		WorkflowMachinePhaseActionRunning:
	default:
		return errors.New("workflow machine phase is invalid")
	}
	if p.SelectedTransition < -1 {
		return errors.New("workflow machine selected transition is invalid")
	}
	if p.EnteredAt.IsZero() {
		return errors.New("workflow machine state entry time is required")
	}
	if p.ChildPath == "" && p.ChildMergesState {
		return errors.New(
			"workflow machine child merge state exists without a child path",
		)
	}
	if !p.Event.Valid() {
		return errors.New("workflow machine event is invalid")
	}
	for _, wait := range p.Waits {
		if strings.TrimSpace(wait.Key) == "" ||
			wait.WorkItemID == "" ||
			len(wait.TransitionIndexes) == 0 {
			return errors.New("workflow machine wait is invalid")
		}
	}
	for _, subscription := range p.Subscriptions {
		if err := validateID(
			"workflow machine subscription",
			subscription,
		); err != nil {
			return err
		}
	}
	return nil
}

func machineStateByName(
	machine WorkflowMachine,
	name string,
) (*WorkflowMachineState, bool) {
	for index := range machine.States {
		if machine.States[index].Name != name {
			continue
		}
		value := machine.States[index]
		return &value, true
	}
	return nil, false
}

func (i *WorkflowInterpreter) startMachine(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	path string,
) (ProgramTransition, error) {
	if step.Machine == nil {
		return workflowFailure(
			state,
			"workflow_machine_invalid",
			"workflow machine is missing",
		)
	}
	if err := step.Machine.Validate(); err != nil {
		return workflowFailure(
			state,
			"workflow_machine_invalid",
			err.Error(),
		)
	}

	machineState := state.State.Clone()
	localState := step.Machine.State != nil
	if step.Machine.State != nil {
		value, err := i.evaluate(
			step.Machine.State.Initial,
			program,
			state,
		)
		if err != nil {
			return workflowFailure(
				state,
				"workflow_machine_state_invalid",
				err.Error(),
			)
		}
		if err := validateWorkflowSchema(
			step.Machine.State.Schema,
			value,
		); err != nil {
			return workflowFailure(
				state,
				"workflow_machine_state_invalid",
				err.Error(),
			)
		}
		machineState = value
	}

	subscriptions, err := i.machineSubscriptions(
		program,
		state,
		step.Machine.Listen,
		machineState,
	)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_machine_subscription_invalid",
			err.Error(),
		)
	}
	state.Subscriptions = append(state.Subscriptions, subscriptions...)

	names := make([]string, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		names = append(names, subscription.Name)
	}
	state.Pending = &WorkflowPending{
		Kind:   WorkflowPendingMachine,
		StepID: step.ID,
		Path:   path,
		Machine: &WorkflowMachineProgress{
			Current:            step.Machine.Initial,
			State:              machineState,
			LocalState:         localState,
			Phase:              WorkflowMachinePhaseEntry,
			SelectedTransition: -1,
			Generation:         1,
			EnteredAt:          state.LogicalTime.UTC(),
			Subscriptions:      names,
		},
	}
	return i.drivePending(frame, execution, program, state)
}

func (i *WorkflowInterpreter) machineSubscriptions(
	program WorkflowProgram,
	state WorkflowState,
	declarations []WorkflowSubscription,
	machineState CanonicalValue,
) ([]WorkflowSubscriptionState, error) {
	if len(declarations) == 0 {
		return nil, nil
	}

	active := make(map[string]struct{}, len(state.Subscriptions))
	for _, subscription := range state.Subscriptions {
		active[subscription.Name] = struct{}{}
	}
	evaluationState := state
	evaluationState.State = machineState.Clone()
	output := make(
		[]WorkflowSubscriptionState,
		0,
		len(declarations),
	)

	for _, declaration := range declarations {
		if _, duplicate := active[declaration.Name]; duplicate {
			return nil, fmt.Errorf(
				"workflow machine subscription %q shadows an active subscription",
				declaration.Name,
			)
		}
		event, found := workflowEventForName(
			program,
			declaration.Event,
		)
		if !found {
			return nil, fmt.Errorf(
				"workflow machine subscription %q has no event",
				declaration.Name,
			)
		}
		filter, err := i.workflowWaitFilter(
			program,
			evaluationState,
			declaration.Filter,
		)
		if err != nil {
			return nil, err
		}

		correlation := CanonicalValue(nil)
		if declaration.Correlation != nil {
			value, err := i.evaluate(
				*declaration.Correlation,
				program,
				evaluationState,
			)
			if err != nil {
				return nil, err
			}
			if err := validateWorkflowSchema(
				event.CorrelationSchema,
				value,
			); err != nil {
				return nil, fmt.Errorf(
					"validate workflow machine correlation: %w",
					err,
				)
			}
			correlation = value
		}
		capacity, maxBytes := workflowSubscriptionLimits(declaration)
		output = append(output, WorkflowSubscriptionState{
			Name:              declaration.Name,
			Event:             event.Name,
			Source:            event.Source,
			Correlation:       correlation,
			CorrelationSchema: event.CorrelationSchema.Clone(),
			DataSchema:        event.Schema.Clone(),
			Filter:            filter,
			Retention:         declaration.Retention,
			Capacity:          capacity,
			MaxBytes:          maxBytes,
			ActivatedAt:       state.LogicalTime.UTC(),
		})
		active[declaration.Name] = struct{}{}
	}
	return output, nil
}

func (i *WorkflowInterpreter) driveMachine(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
) (ProgramTransition, error) {
	pending := state.Pending
	if pending == nil || pending.Machine == nil {
		return workflowFailure(
			state,
			"workflow_machine_invalid",
			"workflow machine state is unavailable",
		)
	}
	step, err := workflowStepForPath(program.Root, pending.Path)
	if err != nil || step.Machine == nil {
		return workflowFailure(
			state,
			"workflow_machine_invalid",
			"workflow machine step is unavailable",
		)
	}
	progress := pending.Machine
	current, found := machineStateByName(
		*step.Machine,
		progress.Current,
	)
	if !found {
		return workflowFailure(
			state,
			"workflow_machine_invalid",
			"workflow machine current state is unavailable",
		)
	}

	switch progress.Phase {
	case WorkflowMachinePhaseEntry:
		if current.Entry != nil {
			return i.startMachineChild(
				frame,
				program,
				state,
				*current.Entry,
				"entry",
				WorkflowMachinePhaseEntryRunning,
			)
		}
		progress.Phase = WorkflowMachinePhaseSelect
		return i.driveMachine(frame, execution, program, state)

	case WorkflowMachinePhaseEntryRunning,
		WorkflowMachinePhaseExitRunning,
		WorkflowMachinePhaseActionRunning:
		return workflowTransition(state, nil, nil, nil)

	case WorkflowMachinePhaseSelect:
		if current.Final {
			return i.finishMachine(program, state, step)
		}
		for index, transition := range current.Transitions {
			if transition.Event != "" || transition.After != "" {
				continue
			}
			matched, err := i.machineTransitionMatches(
				transition,
				program,
				state,
				*progress,
				nil,
			)
			if err != nil {
				return workflowFailure(
					state,
					"workflow_machine_condition_failed",
					err.Error(),
				)
			}
			if !matched {
				continue
			}
			if err := selectMachineTransition(
				&state,
				index,
				nil,
			); err != nil {
				return workflowFailure(
					state,
					"workflow_machine_transition_invalid",
					err.Error(),
				)
			}
			return i.driveMachine(frame, execution, program, state)
		}
		return i.armMachineWaits(
			frame,
			execution,
			program,
			state,
			step,
			*current,
		)

	case WorkflowMachinePhaseWaiting:
		return i.armMachineWaits(
			frame,
			execution,
			program,
			state,
			step,
			*current,
		)

	case WorkflowMachinePhaseExit:
		if current.Exit != nil {
			return i.startMachineChild(
				frame,
				program,
				state,
				*current.Exit,
				"exit",
				WorkflowMachinePhaseExitRunning,
			)
		}
		progress.Phase = WorkflowMachinePhaseAction
		return i.driveMachine(frame, execution, program, state)

	case WorkflowMachinePhaseAction:
		if progress.SelectedTransition < 0 ||
			progress.SelectedTransition >= len(current.Transitions) {
			return workflowFailure(
				state,
				"workflow_machine_invalid",
				"workflow machine transition is unavailable",
			)
		}
		transition := current.Transitions[progress.SelectedTransition]
		if transition.Do != nil {
			return i.startMachineChild(
				frame,
				program,
				state,
				*transition.Do,
				"transition-"+strconv.Itoa(
					progress.SelectedTransition,
				),
				WorkflowMachinePhaseActionRunning,
			)
		}
		return i.enterMachineTarget(
			frame,
			execution,
			program,
			state,
			transition,
		)

	default:
		return workflowFailure(
			state,
			"workflow_machine_invalid",
			"workflow machine phase is invalid",
		)
	}
}

func (i *WorkflowInterpreter) startMachineChild(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	block WorkflowBlock,
	role string,
	phase WorkflowMachinePhase,
) (ProgramTransition, error) {
	pending := state.Pending
	if pending == nil || pending.Machine == nil {
		return ProgramTransition{}, errors.New(
			"workflow machine state is unavailable",
		)
	}

	progress := pending.Machine
	childName := role + "-" + strconv.FormatUint(
		progress.Generation,
		10,
	)
	childPath := pending.Path + "/machine/" +
		progress.Current + "/" + childName

	scope := WorkflowScopeContext{
		Child:              true,
		MergeState:         block.State == nil,
		OuterState:         progress.State.Clone(),
		OuterOutcomes:      workflowScopeOutcomes(program, state),
		OuterSubscriptions: cloneWorkflowSubscriptionStates(state.Subscriptions),
		Event:              workflowCanonicalPointer(progress.Event),
		Outcome:            nil,
	}
	childState := state
	childState.State = progress.State.Clone()
	snapshot, err := i.childBlockProgramWithScope(
		frame,
		program,
		childState,
		block,
		block.State == nil,
		scope,
	)
	if err != nil {
		return ProgramTransition{}, err
	}

	pending.Children = append(pending.Children, WorkflowChild{
		Name:  childName,
		Path:  childPath,
		State: WorkflowChildStatePending,
	})
	progress.ChildPath = childPath
	progress.ChildMergesState = block.State == nil
	progress.Phase = phase
	return workflowStructuredTransition(
		state,
		[]ProgramFrameIntent{{
			Path:     childPath,
			Program:  snapshot,
			Deadline: cloneTime(state.Deadline),
		}},
	)
}

func (i *WorkflowInterpreter) armMachineWaits(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	current WorkflowMachineState,
) (ProgramTransition, error) {
	pending := state.Pending
	if pending == nil || pending.Machine == nil {
		return ProgramTransition{}, errors.New(
			"workflow machine state is unavailable",
		)
	}
	progress := pending.Machine

	active := make([]WorkflowMachineWait, 0, len(progress.Waits))
	for _, wait := range progress.Waits {
		item := executionWorkItem(execution, wait.WorkItemID)
		if item != nil && !item.State.Terminal() {
			active = append(active, wait)
		}
	}
	if len(active) != 0 {
		progress.Waits = active
		progress.Phase = WorkflowMachinePhaseWaiting
		return workflowTransition(state, nil, nil, nil)
	}
	progress.Waits = active

	eventGroups := make(map[string][]int)
	timerIndexes := make([]int, 0)
	for index, transition := range current.Transitions {
		switch {
		case transition.Event != "":
			eventGroups[transition.Event] = append(
				eventGroups[transition.Event],
				index,
			)
		case transition.After != "":
			timerIndexes = append(timerIndexes, index)
		}
	}
	if len(eventGroups) == 0 && len(timerIndexes) == 0 {
		return workflowFailure(
			state,
			"workflow_machine_wait_unavailable",
			"workflow machine has no eligible event or timer transition",
		)
	}

	events := make([]string, 0, len(eventGroups))
	for name := range eventGroups {
		events = append(events, name)
	}
	sort.Strings(events)
	for _, name := range events {
		sort.Ints(eventGroups[name])
	}
	sort.Ints(timerIndexes)

	intents := make([]WorkItemIntent, 0, len(events)+len(timerIndexes))
	for _, name := range events {
		subscription, found := workflowSubscriptionForName(
			state,
			name,
		)
		if !found {
			return workflowFailure(
				state,
				"workflow_machine_subscription_invalid",
				"workflow machine transition subscription is inactive",
			)
		}
		key := "event-" + name
		id := machineWaitWorkItemID(
			frame.ID,
			pending.Path,
			progress.Generation,
			key,
		)
		progress.Waits = append(progress.Waits, WorkflowMachineWait{
			Key:        key,
			WorkItemID: id,
			TransitionIndexes: append(
				[]int(nil),
				eventGroups[name]...,
			),
		})
		intents = append(intents, WorkItemIntent{
			ID: id,
			ProgramPath: pending.Path + "/machine/" +
				progress.Current + "/wait/" + key,
			Type:      WorkItemTypeWait,
			Automatic: true,
			Effect:    EffectContract{Kind: EffectKindRead},
			Input: WorkItemPayload{Wait: &WaitWork{Request: WaitRequest{
				SubscriptionFrameID: frame.ID,
				Kind:                WaitKindEvent,
				Subscription:        subscription.Name,
				Signal:              subscription.Event,
				EventSource:         subscription.Source,
				Correlation:         subscription.Correlation.Clone(),
				CorrelationSchema:   subscription.CorrelationSchema.Clone(),
				DataSchema:          subscription.DataSchema.Clone(),
				Consume:             true,
				Deadline:            cloneTime(state.Deadline),
			}}},
		})
	}

	for _, index := range timerIndexes {
		transition := current.Transitions[index]
		fireAt, err := transition.After.Deadline(progress.EnteredAt)
		if err != nil {
			return workflowFailure(
				state,
				"workflow_machine_timer_invalid",
				err.Error(),
			)
		}
		key := "timer-" + strconv.Itoa(index)
		id := machineWaitWorkItemID(
			frame.ID,
			pending.Path,
			progress.Generation,
			key,
		)
		progress.Waits = append(progress.Waits, WorkflowMachineWait{
			Key:               key,
			WorkItemID:        id,
			TransitionIndexes: []int{index},
		})
		intents = append(intents, WorkItemIntent{
			ID: id,
			ProgramPath: pending.Path + "/machine/" +
				progress.Current + "/wait/" + key,
			Type:      WorkItemTypeWait,
			Automatic: true,
			Effect:    EffectContract{Kind: EffectKindRead},
			Input: WorkItemPayload{Wait: &WaitWork{Request: WaitRequest{
				Kind: WaitKindTimer,
				Timer: &TimerWait{
					ScheduledAt: progress.EnteredAt,
					FireAt:      fireAt,
				},
				Deadline: cloneTime(state.Deadline),
			}}},
		})
	}

	progress.Phase = WorkflowMachinePhaseWaiting
	return workflowTransition(state, nil, nil, intents)
}

func machineWaitWorkItemID(
	frameID ProgramFrameID,
	path string,
	generation uint64,
	key string,
) WorkItemID {
	return WorkItemID(
		"machine:" + string(frameID) + ":" +
			path + ":" + strconv.FormatUint(generation, 10) +
			":" + key,
	)
}

func (i *WorkflowInterpreter) settleMachineWork(
	state *WorkflowState,
	program WorkflowProgram,
	execution Execution,
	frameID ProgramFrameID,
	workItemID WorkItemID,
) (bool, error) {
	if state == nil || state.Pending == nil ||
		state.Pending.Kind != WorkflowPendingMachine ||
		state.Pending.Machine == nil {
		return false, nil
	}

	pending := state.Pending
	progress := pending.Machine
	waitIndex := -1
	for index := range progress.Waits {
		if progress.Waits[index].WorkItemID == workItemID {
			waitIndex = index
			break
		}
	}
	if waitIndex < 0 {
		return false, nil
	}

	item := executionWorkItem(execution, workItemID)
	if item == nil || item.FrameID != frameID {
		return false, nil
	}
	if item.State != WorkItemStateCompleted || item.Output == nil {
		return true, fmt.Errorf(
			"workflow machine wait %q did not complete successfully",
			workItemID,
		)
	}

	wait := progress.Waits[waitIndex]
	progress.Waits = append(
		progress.Waits[:waitIndex],
		progress.Waits[waitIndex+1:]...,
	)

	step, err := workflowStepForPath(program.Root, pending.Path)
	if err != nil || step.Machine == nil {
		return true, errors.New("workflow machine step is unavailable")
	}
	current, found := machineStateByName(
		*step.Machine,
		progress.Current,
	)
	if !found {
		return true, errors.New("workflow machine current state is unavailable")
	}

	event := item.Output.Value.Clone()
	for _, index := range wait.TransitionIndexes {
		if index < 0 || index >= len(current.Transitions) {
			return true, errors.New(
				"workflow machine transition index is invalid",
			)
		}
		transition := current.Transitions[index]
		matched, err := i.machineTransitionMatches(
			transition,
			program,
			*state,
			*progress,
			event,
		)
		if err != nil {
			return true, err
		}
		if !matched {
			continue
		}
		if err := selectMachineTransition(
			state,
			index,
			event,
		); err != nil {
			return true, err
		}
		return true, nil
	}

	// The accepted event was consumed by its subscription. Re-arm this state
	// for a future event without re-reading the same retained signal.
	progress.Phase = WorkflowMachinePhaseWaiting
	return true, nil
}

func (i *WorkflowInterpreter) machineTransitionMatches(
	transition WorkflowMachineTransition,
	program WorkflowProgram,
	state WorkflowState,
	progress WorkflowMachineProgress,
	event CanonicalValue,
) (bool, error) {
	if transition.When == nil {
		return true, nil
	}
	evaluationState := state
	evaluationState.State = progress.State.Clone()
	return i.conditionWithEvent(
		*transition.When,
		program,
		evaluationState,
		event,
	)
}

func selectMachineTransition(
	state *WorkflowState,
	index int,
	event CanonicalValue,
) error {
	if state == nil || state.Pending == nil ||
		state.Pending.Machine == nil {
		return errors.New("workflow machine state is unavailable")
	}
	progress := state.Pending.Machine
	progress.SelectedTransition = index
	progress.Event = event.Clone()
	progress.Phase = WorkflowMachinePhaseExit
	progress.Microsteps++
	if progress.Microsteps > maximumWorkflowMachineMicrosteps {
		return errors.New(
			"workflow machine exceeded its immediate transition limit",
		)
	}

	for _, wait := range progress.Waits {
		state.CancelWorkItemIDs = append(
			state.CancelWorkItemIDs,
			wait.WorkItemID,
		)
	}
	progress.Waits = nil
	return nil
}

func (i *WorkflowInterpreter) settleMachineChild(
	state *WorkflowState,
	program WorkflowProgram,
) error {
	if state == nil || state.Pending == nil ||
		state.Pending.Kind != WorkflowPendingMachine ||
		state.Pending.Machine == nil {
		return errors.New("workflow machine state is unavailable")
	}
	pending := state.Pending
	progress := pending.Machine
	if progress.ChildPath == "" {
		return nil
	}

	var child *WorkflowChild
	for index := range pending.Children {
		if pending.Children[index].Path == progress.ChildPath {
			child = &pending.Children[index]
			break
		}
	}
	if child == nil || !child.State.terminal() {
		return nil
	}

	if child.State != WorkflowChildStateSucceeded ||
		child.Result == nil {
		return workflowChildError{
			errorValue: workflowAggregateChildFailure(
				"workflow_machine_child_failed",
				"workflow machine child block did not complete",
				[]WorkflowChild{*child},
			),
		}
	}

	result := *child.Result
	switch result.Exit {
	case WorkflowExitNormal,
		WorkflowExitReturnBlock:
	case WorkflowExitReturnWorkflow,
		WorkflowExitBreak,
		WorkflowExitContinue:
		state.Exit = result.Exit
		state.ExitValue = result.Value.Clone()
		state.Pending = nil
		return nil
	default:
		return errors.New(
			"workflow machine child returned an invalid control result",
		)
	}

	if progress.ChildMergesState {
		progress.State = result.State.Clone()
	}
	progress.ChildPath = ""
	progress.ChildMergesState = false

	switch progress.Phase {
	case WorkflowMachinePhaseEntryRunning:
		progress.Phase = WorkflowMachinePhaseSelect

	case WorkflowMachinePhaseExitRunning:
		progress.Phase = WorkflowMachinePhaseAction

	case WorkflowMachinePhaseActionRunning:
		step, err := workflowStepForPath(program.Root, pending.Path)
		if err != nil || step.Machine == nil {
			return errors.New("workflow machine step is unavailable")
		}
		current, found := machineStateByName(
			*step.Machine,
			progress.Current,
		)
		if !found ||
			progress.SelectedTransition < 0 ||
			progress.SelectedTransition >= len(current.Transitions) {
			return errors.New(
				"workflow machine selected transition is unavailable",
			)
		}
		transition := current.Transitions[progress.SelectedTransition]
		progress.Current = transition.To
		progress.Generation++
		progress.EnteredAt = state.LogicalTime.UTC()
		progress.Event = nil
		progress.SelectedTransition = -1
		progress.Phase = WorkflowMachinePhaseEntry

	default:
		return errors.New("workflow machine child settled in an invalid phase")
	}
	return nil
}

func (i *WorkflowInterpreter) enterMachineTarget(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	transition WorkflowMachineTransition,
) (ProgramTransition, error) {
	if state.Pending == nil || state.Pending.Machine == nil {
		return ProgramTransition{}, errors.New(
			"workflow machine state is unavailable",
		)
	}
	progress := state.Pending.Machine
	progress.Current = transition.To
	progress.Generation++
	progress.EnteredAt = state.LogicalTime.UTC()
	progress.Event = nil
	progress.SelectedTransition = -1
	progress.Phase = WorkflowMachinePhaseEntry
	return i.driveMachine(frame, execution, program, state)
}

func (i *WorkflowInterpreter) finishMachine(
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
) (ProgramTransition, error) {
	if state.Pending == nil || state.Pending.Machine == nil ||
		step.Machine == nil {
		return ProgramTransition{}, errors.New(
			"workflow machine state is unavailable",
		)
	}

	pending := state.Pending
	progress := pending.Machine
	current, found := machineStateByName(
		*step.Machine,
		progress.Current,
	)
	if !found || !current.Final || current.Output == nil {
		return workflowFailure(
			state,
			"workflow_machine_invalid",
			"workflow machine reached a non-final state without a transition",
		)
	}

	evaluationState := state
	evaluationState.State = progress.State.Clone()
	value, err := i.evaluateWithEvent(
		*current.Output,
		program,
		evaluationState,
		progress.Event,
	)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_machine_output_failed",
			err.Error(),
		)
	}
	if !progress.LocalState {
		state.State = progress.State.Clone()
	}

	if err := i.validateStepOutput(
		program,
		state,
		step,
		value,
	); err != nil {
		return workflowFailure(
			state,
			"workflow_machine_output_invalid",
			err.Error(),
		)
	}
	if err := state.recordOutcome(WorkflowStepOutcome{
		ID:     step.ID,
		Status: WorkflowStepOutcomeSucceeded,
		Output: value,
	}); err != nil {
		return workflowFailure(
			state,
			"workflow_state_invalid",
			err.Error(),
		)
	}

	for _, wait := range progress.Waits {
		state.CancelWorkItemIDs = append(
			state.CancelWorkItemIDs,
			wait.WorkItemID,
		)
	}
	progress.Waits = nil
	state.Subscriptions = removeWorkflowSubscriptions(
		state.Subscriptions,
		progress.Subscriptions,
	)
	state.Pending = nil
	state.Index++
	state.PendingPath = ""
	state.PendingTarget = ""
	state.Deadline = nil
	return workflowTransition(state, nil, nil, nil)
}

func removeWorkflowSubscriptions(
	values []WorkflowSubscriptionState,
	names []string,
) []WorkflowSubscriptionState {
	if len(names) == 0 {
		return values
	}
	remove := make(map[string]struct{}, len(names))
	for _, name := range names {
		remove[name] = struct{}{}
	}
	output := make(
		[]WorkflowSubscriptionState,
		0,
		len(values),
	)
	for _, value := range values {
		if _, found := remove[value.Name]; found {
			continue
		}
		output = append(output, value)
	}
	return output
}
