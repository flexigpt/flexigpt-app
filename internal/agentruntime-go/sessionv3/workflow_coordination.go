package sessionv3

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type WorkflowEventContract struct {
	Name              string         `json:"name"`
	Source            string         `json:"source"`
	Schema            CanonicalValue `json:"schema"`
	CorrelationSchema CanonicalValue `json:"correlationSchema,omitempty"`
}

type WorkflowChannel struct {
	Name        string         `json:"name"`
	Schema      CanonicalValue `json:"schema"`
	MaxMessages int            `json:"maxMessages,omitempty"`
	MaxBytes    int            `json:"maxBytes,omitempty"`
	Readers     []string       `json:"readers,omitempty"`
	Writers     []string       `json:"writers,omitempty"`
}

type WorkflowSubscription struct {
	Name        string         `json:"name"`
	Event       string         `json:"event"`
	Correlation *WorkflowValue `json:"correlation,omitempty"`
	Filter      *WorkflowValue `json:"filter,omitempty"`
}

type WorkflowSubscriptionState struct {
	Name              string         `json:"name"`
	Event             string         `json:"event"`
	Source            string         `json:"source"`
	Correlation       CanonicalValue `json:"correlation,omitempty"`
	CorrelationSchema CanonicalValue `json:"correlationSchema,omitempty"`
	DataSchema        CanonicalValue `json:"dataSchema,omitempty"`
	Filter            *WorkflowValue `json:"filter,omitempty"`
}

type WorkflowWaitFilter struct {
	Filters []WorkflowValue       `json:"filters"`
	Input   CanonicalValue        `json:"input"`
	State   CanonicalValue        `json:"state"`
	Steps   []WorkflowStepOutcome `json:"steps"`
}

type WorkflowEventWait struct {
	Subscription string         `json:"subscription"`
	Match        *WorkflowValue `json:"match,omitempty"`
	Consume      bool           `json:"consume,omitempty"`
}

type WorkflowChannelWait struct {
	Channel string         `json:"channel"`
	Cursor  *WorkflowValue `json:"cursor,omitempty"`
	Match   *WorkflowValue `json:"match,omitempty"`
	Consume bool           `json:"consume,omitempty"`
}

type WorkflowCompletionWait struct {
	Handle WorkflowValue `json:"handle"`
}

type WorkflowHumanWait struct {
	Users          *WorkflowValue `json:"users,omitempty"`
	Roles          *WorkflowValue `json:"roles,omitempty"`
	Request        WorkflowValue  `json:"request"`
	ResponseSchema CanonicalValue `json:"responseSchema"`
}

type WorkflowEmit struct {
	Destination      EmitDestination `json:"destination"`
	Kind             string          `json:"kind,omitempty"`
	Data             WorkflowValue   `json:"data"`
	Correlation      *WorkflowValue  `json:"correlation,omitempty"`
	DeduplicationKey *WorkflowValue  `json:"deduplicationKey,omitempty"`
}

func (e WorkflowEventContract) Validate() error {
	if err := validateID("workflow event name", e.Name); err != nil {
		return err
	}
	if err := validateID("workflow event source", e.Source); err != nil {
		return err
	}
	if len(e.Schema) == 0 || !e.Schema.Valid() || !e.CorrelationSchema.Valid() {
		return errors.New("workflow event schema is invalid")
	}
	return nil
}

func (c WorkflowChannel) Validate() error {
	if err := validateID("workflow channel name", c.Name); err != nil {
		return err
	}
	if len(c.Schema) == 0 || !c.Schema.Valid() {
		return errors.New("workflow channel schema is invalid")
	}
	if c.MaxMessages < 0 || c.MaxBytes < 0 {
		return errors.New("workflow channel limits cannot be negative")
	}
	if err := validateWorkflowChannelPaths(c.Readers); err != nil {
		return err
	}
	return validateWorkflowChannelPaths(c.Writers)
}

func validateWorkflowChannelPaths(values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := validateID("workflow channel step path", value); err != nil {
			return err
		}
		if _, duplicate := seen[value]; duplicate {
			return errors.New("workflow channel path is duplicated")
		}
		seen[value] = struct{}{}
	}
	return nil
}

func (s WorkflowSubscription) Validate() error {
	if err := validateID("workflow subscription name", s.Name); err != nil {
		return err
	}
	if err := validateID("workflow subscription event", s.Event); err != nil {
		return err
	}
	if s.Correlation != nil {
		if err := s.Correlation.Validate(); err != nil {
			return err
		}
	}
	if s.Filter != nil {
		if err := s.Filter.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func validateWorkflowSubscriptions(values []WorkflowSubscription) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return err
		}
		if _, duplicate := seen[value.Name]; duplicate {
			return errors.New("workflow subscription is duplicated")
		}
		seen[value.Name] = struct{}{}
	}
	return nil
}

func (s WorkflowSubscriptionState) Validate() error {
	if err := validateID("workflow subscription name", s.Name); err != nil {
		return err
	}
	if err := validateID("workflow subscription event", s.Event); err != nil {
		return err
	}
	if err := validateID("workflow subscription source", s.Source); err != nil {
		return err
	}
	if !s.Correlation.Valid() ||
		!s.CorrelationSchema.Valid() ||
		!s.DataSchema.Valid() {
		return errors.New("workflow subscription state is invalid")
	}
	if s.Filter != nil {
		return s.Filter.Validate()
	}
	return nil
}

func (a WaitAssignees) Validate() error {
	if len(a.Users) == 0 && len(a.Roles) == 0 {
		return errors.New("human wait has no assignees")
	}
	seenUsers := make(map[string]struct{}, len(a.Users))
	for _, user := range a.Users {
		if err := validateID("human wait user", user); err != nil {
			return err
		}
		if _, duplicate := seenUsers[user]; duplicate {
			return errors.New("human wait user is duplicated")
		}
		seenUsers[user] = struct{}{}
	}
	seenRoles := make(map[string]struct{}, len(a.Roles))
	for _, role := range a.Roles {
		if err := validateID("human wait role", role); err != nil {
			return err
		}
		if _, duplicate := seenRoles[role]; duplicate {
			return errors.New("human wait role is duplicated")
		}
		seenRoles[role] = struct{}{}
	}
	return nil
}

func (f WorkflowWaitFilter) Validate() error {
	if len(f.Filters) == 0 ||
		!f.Input.Valid() ||
		!f.State.Valid() {
		return errors.New("workflow wait filter is invalid")
	}
	for _, value := range f.Filters {
		if err := value.Validate(); err != nil {
			return err
		}
	}
	for _, outcome := range f.Steps {
		if strings.TrimSpace(outcome.ID) == "" || !outcome.Output.Valid() {
			return errors.New("workflow wait filter step outcome is invalid")
		}
	}
	return nil
}

func (f WorkflowWaitFilter) Matches(event CanonicalValue) (bool, error) {
	if err := f.Validate(); err != nil {
		return false, err
	}
	if !event.Valid() {
		return false, errors.New("workflow wait event is invalid")
	}

	interpreter, err := NewWorkflowInterpreter()
	if err != nil {
		return false, err
	}
	activationState := WorkflowState{
		Version:  WorkflowInterpreterVersion,
		State:    f.State.Clone(),
		Outcomes: cloneWorkflowStepOutcomes(f.Steps),
	}
	activation, err := workflowActivation(
		WorkflowProgram{Input: f.Input.Clone()},
		activationState,
	)
	if err != nil {
		return false, err
	}
	eventValue, err := workflowActivationValue(event)
	if err != nil {
		return false, err
	}
	activation["event"] = eventValue

	for _, filter := range f.Filters {
		value, err := interpreter.evaluateWithActivation(filter, activation)
		if err != nil {
			return false, err
		}
		matched, err := DecodeCanonicalValue[bool](value)
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}
	}
	return true, nil
}

func (i *WorkflowInterpreter) evaluateWithActivation(
	value WorkflowValue,
	activation map[string]any,
) (CanonicalValue, error) {
	switch value.Kind {
	case WorkflowValueLiteral:
		return value.Literal.Clone(), nil

	case WorkflowValueExpression:
		if i == nil || i.env == nil {
			return nil, errors.New("workflow expression runtime is unavailable")
		}
		compiled, issues := i.env.Compile(value.Expression)
		if issues != nil && issues.Err() != nil {
			return nil, fmt.Errorf("compile workflow expression: %w", issues.Err())
		}
		executable, err := i.env.Program(compiled)
		if err != nil {
			return nil, err
		}
		result, _, err := executable.Eval(activation)
		if err != nil {
			return nil, fmt.Errorf("evaluate workflow expression: %w", err)
		}
		return canonicalWorkflowCELValue(result)

	case WorkflowValueObject:
		output := make(map[string]CanonicalValue, len(value.Object))
		for _, field := range value.Object {
			child, err := i.evaluateWithActivation(field.Value, activation)
			if err != nil {
				return nil, err
			}
			output[field.Key] = child
		}
		return canonicalWorkflowValue(output)

	case WorkflowValueArray:
		output := make([]CanonicalValue, len(value.Array))
		for index := range value.Array {
			child, err := i.evaluateWithActivation(value.Array[index], activation)
			if err != nil {
				return nil, err
			}
			output[index] = child
		}
		return canonicalWorkflowValue(output)

	default:
		return nil, errors.New("workflow value kind is unsupported")
	}
}

func (w WorkflowEventWait) Validate() error {
	if err := validateID("workflow event wait subscription", w.Subscription); err != nil {
		return err
	}
	if w.Match != nil {
		return w.Match.Validate()
	}
	return nil
}

func (w WorkflowChannelWait) Validate() error {
	if err := validateID("workflow channel wait channel", w.Channel); err != nil {
		return err
	}
	if w.Cursor != nil {
		if err := w.Cursor.Validate(); err != nil {
			return err
		}
	}
	if w.Match != nil {
		return w.Match.Validate()
	}
	return nil
}

func (w WorkflowCompletionWait) Validate() error {
	return w.Handle.Validate()
}

func (w WorkflowHumanWait) Validate() error {
	if w.Users == nil && w.Roles == nil {
		return errors.New("workflow human wait has no assignee expression")
	}
	if w.Users != nil {
		if err := w.Users.Validate(); err != nil {
			return err
		}
	}
	if w.Roles != nil {
		if err := w.Roles.Validate(); err != nil {
			return err
		}
	}
	if err := w.Request.Validate(); err != nil {
		return err
	}
	if len(w.ResponseSchema) == 0 || !w.ResponseSchema.Valid() {
		return errors.New("workflow human wait response schema is invalid")
	}
	return nil
}

func (e WorkflowEmit) Validate() error {
	switch {
	case e.Destination.Channel == "" && e.Destination.Event == "":
		return errors.New("workflow emit has no destination")
	case e.Destination.Channel != "" && e.Destination.Event != "":
		return errors.New("workflow emit destination is ambiguous")
	}
	if err := e.Data.Validate(); err != nil {
		return err
	}
	if e.Correlation != nil {
		if err := e.Correlation.Validate(); err != nil {
			return err
		}
	}
	if e.DeduplicationKey != nil {
		return e.DeduplicationKey.Validate()
	}
	return nil
}

func validateWorkflowCoordinationProgram(program WorkflowProgram) error {
	events := make(map[string]WorkflowEventContract, len(program.Events))
	for _, event := range program.Events {
		if err := event.Validate(); err != nil {
			return err
		}
		if _, duplicate := events[event.Name]; duplicate {
			return errors.New("workflow event is duplicated")
		}
		events[event.Name] = event
	}

	channels := make(map[string]WorkflowChannel, len(program.Channels))
	for _, channel := range program.Channels {
		if err := channel.Validate(); err != nil {
			return err
		}
		if _, duplicate := channels[channel.Name]; duplicate {
			return errors.New("workflow channel is duplicated")
		}
		channels[channel.Name] = channel
	}

	active := make(map[string]struct{}, len(program.Scope.OuterSubscriptions))
	for _, subscription := range program.Scope.OuterSubscriptions {
		if _, found := events[subscription.Event]; !found {
			return errors.New("workflow inherited subscription has no event")
		}
		if _, duplicate := active[subscription.Name]; duplicate {
			return errors.New("workflow inherited subscription is duplicated")
		}
		active[subscription.Name] = struct{}{}
	}
	return validateWorkflowCoordinationBlock(program.Root, events, channels, active)
}

func validateWorkflowCoordinationBlock(
	block WorkflowBlock,
	events map[string]WorkflowEventContract,
	channels map[string]WorkflowChannel,
	inherited map[string]struct{},
) error {
	active := cloneWorkflowStringSet(inherited)
	for _, subscription := range block.Listen {
		if _, found := events[subscription.Event]; !found {
			return fmt.Errorf("workflow subscription %q has no event", subscription.Name)
		}
		if _, duplicate := active[subscription.Name]; duplicate {
			return fmt.Errorf("workflow subscription %q shadows an active subscription", subscription.Name)
		}
		active[subscription.Name] = struct{}{}
	}

	for _, step := range block.Steps {
		if step.Wait != nil {
			switch {
			case step.Wait.Event != nil:
				if _, found := active[step.Wait.Event.Subscription]; !found {
					return fmt.Errorf(
						"workflow event wait references inactive subscription %q",
						step.Wait.Event.Subscription,
					)
				}
			case step.Wait.Channel != nil:
				if _, found := channels[step.Wait.Channel.Channel]; !found {
					return fmt.Errorf("workflow channel wait references unknown channel %q", step.Wait.Channel.Channel)
				}
			}
		}
		if step.Emit != nil {
			switch {
			case step.Emit.Destination.Channel != "":
				if _, found := channels[step.Emit.Destination.Channel]; !found {
					return fmt.Errorf("workflow emit references unknown channel %q", step.Emit.Destination.Channel)
				}
			case step.Emit.Destination.Event != "":
				if _, found := events[step.Emit.Destination.Event]; !found {
					return fmt.Errorf("workflow emit references unknown event %q", step.Emit.Destination.Event)
				}
			}
		}

		for _, child := range workflowStepBlocks(step) {
			if err := validateWorkflowCoordinationBlock(child, events, channels, active); err != nil {
				return err
			}
		}
	}
	return nil
}

func workflowStepBlocks(step WorkflowStep) []WorkflowBlock {
	output := make([]WorkflowBlock, 0)
	if step.Scope != nil {
		output = append(output, *step.Scope)
	}
	if step.Choose != nil {
		for _, candidate := range step.Choose.Cases {
			output = append(output, candidate.Do)
		}
		if step.Choose.Default != nil {
			output = append(output, *step.Choose.Default)
		}
	}
	if step.Parallel != nil {
		for _, branch := range step.Parallel.Branches {
			output = append(output, branch.Do)
		}
	}
	if step.Foreach != nil {
		output = append(output, step.Foreach.Do)
	}
	if step.Repeat != nil {
		output = append(output, step.Repeat.Do)
	}
	return output
}

func (i *WorkflowInterpreter) activateWorkflowSubscriptions(
	program WorkflowProgram,
	state *WorkflowState,
) error {
	if state == nil {
		return errors.New("workflow state is unavailable")
	}
	state.Subscriptions = cloneWorkflowSubscriptionStates(program.Scope.OuterSubscriptions)
	active := make(map[string]struct{}, len(state.Subscriptions)+len(program.Root.Listen))
	for _, subscription := range state.Subscriptions {
		active[subscription.Name] = struct{}{}
	}

	for _, declaration := range program.Root.Listen {
		if _, duplicate := active[declaration.Name]; duplicate {
			return fmt.Errorf("workflow subscription %q shadows an active subscription", declaration.Name)
		}
		event, found := workflowEventForName(program, declaration.Event)
		if !found {
			return fmt.Errorf("workflow subscription %q has no event", declaration.Name)
		}

		correlation := CanonicalValue(nil)
		if declaration.Correlation != nil {
			value, err := i.evaluate(*declaration.Correlation, program, *state)
			if err != nil {
				return err
			}
			if err := validateWorkflowSchema(event.CorrelationSchema, value); err != nil {
				return fmt.Errorf("validate workflow subscription correlation: %w", err)
			}
			correlation = value
		}

		state.Subscriptions = append(state.Subscriptions, WorkflowSubscriptionState{
			Name:              declaration.Name,
			Event:             event.Name,
			Source:            event.Source,
			Correlation:       correlation,
			CorrelationSchema: event.CorrelationSchema.Clone(),
			DataSchema:        event.Schema.Clone(),
			Filter:            cloneWorkflowValuePointer(declaration.Filter),
		})
		active[declaration.Name] = struct{}{}
	}
	return nil
}

func (i *WorkflowInterpreter) workflowWaitFilter(
	program WorkflowProgram,
	state WorkflowState,
	values ...*WorkflowValue,
) (*WorkflowWaitFilter, error) {
	filters := make([]WorkflowValue, 0, len(values))
	for _, value := range values {
		if value != nil {
			filters = append(filters, cloneWorkflowValue(*value))
		}
	}
	if len(filters) == 0 {
		return nil, nil
	}
	return &WorkflowWaitFilter{
		Filters: filters,
		Input:   program.Input.Clone(),
		State:   state.State.Clone(),
		Steps:   cloneWorkflowStepOutcomes(workflowScopeOutcomes(program, state)),
	}, nil
}

func (i *WorkflowInterpreter) wait(
	program WorkflowProgram,
	state WorkflowState,
	path string,
	spec WorkflowWait,
) (ProgramTransition, error) {
	switch {
	case spec.After != nil || spec.At != nil:
		return i.timerWait(program, state, path, spec)

	case spec.Event != nil:
		subscription, found := workflowSubscriptionForName(state, spec.Event.Subscription)
		if !found {
			return workflowFailure(state, "workflow_wait_invalid", "workflow event subscription is inactive")
		}
		filter, err := i.workflowWaitFilter(program, state, subscription.Filter, spec.Event.Match)
		if err != nil {
			return workflowFailure(state, "workflow_wait_invalid", err.Error())
		}
		return i.workflowStepWork(program, state, path, WorkItemIntent{
			ProgramPath: path,
			Type:        WorkItemTypeWait,
			Automatic:   true,
			Effect:      EffectContract{Kind: EffectKindRead},
			Input: WorkItemPayload{Wait: &WaitWork{Request: WaitRequest{
				Kind:              WaitKindEvent,
				Subscription:      subscription.Name,
				Signal:            subscription.Event,
				EventSource:       subscription.Source,
				Correlation:       subscription.Correlation.Clone(),
				CorrelationSchema: subscription.CorrelationSchema.Clone(),
				DataSchema:        subscription.DataSchema.Clone(),
				Filter:            filter,
				Consume:           spec.Event.Consume,
			}}},
		})

	case spec.Channel != nil:
		channel, found := workflowChannelForName(program, spec.Channel.Channel)
		if !found {
			return workflowFailure(state, "workflow_wait_invalid", "workflow channel is unavailable")
		}
		if !workflowChannelAllows(channel.Readers, path) {
			return workflowFailure(state, "workflow_wait_denied", "workflow channel read is not allowed")
		}
		cursor := uint64(0)
		if spec.Channel.Cursor != nil {
			value, err := i.evaluate(*spec.Channel.Cursor, program, state)
			if err != nil {
				return workflowFailure(state, "workflow_wait_invalid", err.Error())
			}
			cursor, err = DecodeCanonicalValue[uint64](value)
			if err != nil {
				return workflowFailure(
					state,
					"workflow_wait_invalid",
					"workflow channel cursor must be a non-negative integer",
				)
			}
		}
		filter, err := i.workflowWaitFilter(program, state, spec.Channel.Match)
		if err != nil {
			return workflowFailure(state, "workflow_wait_invalid", err.Error())
		}
		return i.workflowStepWork(program, state, path, WorkItemIntent{
			ProgramPath: path,
			Type:        WorkItemTypeWait,
			Automatic:   true,
			Effect:      EffectContract{Kind: EffectKindRead},
			Input: WorkItemPayload{Wait: &WaitWork{Request: WaitRequest{
				Kind:       WaitKindChannel,
				Channel:    channel.Name,
				Cursor:     cursor,
				Consume:    spec.Channel.Consume,
				DataSchema: channel.Schema.Clone(),
				Filter:     filter,
			}}},
		})

	case spec.Completion != nil:
		value, err := i.evaluate(spec.Completion.Handle, program, state)
		if err != nil {
			return workflowFailure(state, "workflow_wait_invalid", err.Error())
		}
		handle, err := DecodeCanonicalValue[CompletionHandle](value)
		if err != nil {
			return workflowFailure(state, "workflow_wait_invalid", "workflow completion handle is invalid")
		}
		if err := handle.Validate(); err != nil {
			return workflowFailure(state, "workflow_wait_invalid", err.Error())
		}
		return i.workflowStepWork(program, state, path, WorkItemIntent{
			ProgramPath: path,
			Type:        WorkItemTypeWait,
			Automatic:   true,
			Effect:      EffectContract{Kind: EffectKindRead},
			Input: WorkItemPayload{Wait: &WaitWork{Request: WaitRequest{
				Kind:       WaitKindChild,
				Completion: &handle,
			}}},
		})

	case spec.Human != nil:
		users, err := i.workflowStringValues(program, state, spec.Human.Users)
		if err != nil {
			return workflowFailure(state, "workflow_wait_invalid", err.Error())
		}
		roles, err := i.workflowStringValues(program, state, spec.Human.Roles)
		if err != nil {
			return workflowFailure(state, "workflow_wait_invalid", err.Error())
		}
		assignees := WaitAssignees{Users: users, Roles: roles}
		if err := assignees.Validate(); err != nil {
			return workflowFailure(state, "workflow_wait_invalid", err.Error())
		}
		request, err := i.evaluate(spec.Human.Request, program, state)
		if err != nil {
			return workflowFailure(state, "workflow_wait_invalid", err.Error())
		}
		summary := MustCanonicalValue(struct {
			Assignees WaitAssignees  `json:"assignees"`
			Request   CanonicalValue `json:"request"`
		}{
			Assignees: assignees,
			Request:   request,
		})
		return i.workflowStepWork(program, state, path, WorkItemIntent{
			ProgramPath: path,
			Type:        WorkItemTypeWait,
			Automatic:   true,
			Effect:      EffectContract{Kind: EffectKindRead},
			Input: WorkItemPayload{Wait: &WaitWork{Request: WaitRequest{
				Kind:               WaitKindHuman,
				Summary:            summary,
				Request:            request,
				Assignees:          &assignees,
				InputSchema:        spec.Human.ResponseSchema.Clone(),
				AllowedResolutions: []WaitResolution{WaitResolutionRespond},
			}}},
		})

	default:
		return workflowFailure(state, "workflow_wait_invalid", "workflow wait is unavailable")
	}
}

func (i *WorkflowInterpreter) emit(
	program WorkflowProgram,
	state WorkflowState,
	path string,
	spec WorkflowEmit,
) (ProgramTransition, error) {
	data, err := i.evaluate(spec.Data, program, state)
	if err != nil {
		return workflowFailure(state, "workflow_emit_invalid", err.Error())
	}
	correlation := MustCanonicalValue(nil)
	if spec.Correlation != nil {
		correlation, err = i.evaluate(*spec.Correlation, program, state)
		if err != nil {
			return workflowFailure(state, "workflow_emit_invalid", err.Error())
		}
	}
	key := ""
	if spec.DeduplicationKey != nil {
		key, err = i.stringValue(*spec.DeduplicationKey, program, state)
		if err != nil {
			return workflowFailure(state, "workflow_emit_invalid", err.Error())
		}
	}

	payload := EmitWork{
		Destination:      spec.Destination,
		Kind:             spec.Kind,
		Data:             data,
		Correlation:      correlation,
		DeduplicationKey: key,
	}
	switch {
	case spec.Destination.Channel != "":
		channel, found := workflowChannelForName(program, spec.Destination.Channel)
		if !found {
			return workflowFailure(state, "workflow_emit_invalid", "workflow channel is unavailable")
		}
		if !workflowChannelAllows(channel.Writers, path) {
			return workflowFailure(state, "workflow_emit_denied", "workflow channel write is not allowed")
		}
		payload.DataSchema = channel.Schema.Clone()
		payload.MaxMessages = channel.MaxMessages
		payload.MaxBytes = channel.MaxBytes

	case spec.Destination.Event != "":
		event, found := workflowEventForName(program, spec.Destination.Event)
		if !found {
			return workflowFailure(state, "workflow_emit_invalid", "workflow event is unavailable")
		}
		payload.EventSource = event.Source
		payload.DataSchema = event.Schema.Clone()
		payload.CorrelationSchema = event.CorrelationSchema.Clone()

	default:
		return workflowFailure(state, "workflow_emit_invalid", "workflow emit destination is unavailable")
	}

	return i.workflowStepWork(program, state, path, WorkItemIntent{
		ProgramPath: path,
		Type:        WorkItemTypeEmit,
		Automatic:   true,
		Effect:      EffectContract{Kind: EffectKindRead},
		Input:       WorkItemPayload{Emit: &payload},
	})
}

func (i *WorkflowInterpreter) workflowStringValues(
	program WorkflowProgram,
	state WorkflowState,
	value *WorkflowValue,
) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	resolved, err := i.evaluate(*value, program, state)
	if err != nil {
		return nil, err
	}
	values, err := DecodeCanonicalValue[[]string](resolved)
	if err != nil {
		return nil, errors.New("workflow human assignees must be string arrays")
	}
	for _, current := range values {
		if strings.TrimSpace(current) == "" || strings.TrimSpace(current) != current {
			return nil, errors.New("workflow human assignee is invalid")
		}
	}
	return values, nil
}

func (i *WorkflowInterpreter) workflowStepWork(
	program WorkflowProgram,
	state WorkflowState,
	path string,
	intent WorkItemIntent,
) (ProgramTransition, error) {
	step, err := workflowStepForPath(program.Root, path)
	if err != nil {
		return workflowFailure(state, "workflow_step_invalid", err.Error())
	}
	intent.Retry = cloneWorkflowRetryPolicy(step.Retry)
	return workflowWork(state, path, intent)
}

func workflowEventForName(
	program WorkflowProgram,
	name string,
) (WorkflowEventContract, bool) {
	for _, event := range program.Events {
		if event.Name == name {
			return event, true
		}
	}
	return WorkflowEventContract{}, false
}

func workflowChannelForName(
	program WorkflowProgram,
	name string,
) (WorkflowChannel, bool) {
	for _, channel := range program.Channels {
		if channel.Name == name {
			return channel, true
		}
	}
	return WorkflowChannel{}, false
}

func workflowSubscriptionForName(
	state WorkflowState,
	name string,
) (WorkflowSubscriptionState, bool) {
	for _, subscription := range state.Subscriptions {
		if subscription.Name == name {
			return subscription, true
		}
	}
	return WorkflowSubscriptionState{}, false
}

func workflowChannelAllows(
	allowed []string,
	path string,
) bool {
	if len(allowed) == 0 {
		return true
	}
	return slices.Contains(allowed, path)
}

func cloneWorkflowRetryPolicy(value *RetryPolicy) *RetryPolicy {
	if value == nil {
		return nil
	}
	output := *value
	output.Match.Categories = append([]ErrorCategory(nil), value.Match.Categories...)
	output.Match.Codes = append([]string(nil), value.Match.Codes...)
	if value.Match.Retryable != nil {
		copyValue := *value.Match.Retryable
		output.Match.Retryable = &copyValue
	}
	return &output
}

func cloneWorkflowValue(value WorkflowValue) WorkflowValue {
	output := value
	output.Literal = value.Literal.Clone()
	output.Object = make([]WorkflowValuePair, len(value.Object))
	for index := range value.Object {
		output.Object[index] = WorkflowValuePair{
			Key:   value.Object[index].Key,
			Value: cloneWorkflowValue(value.Object[index].Value),
		}
	}
	output.Array = make([]WorkflowValue, len(value.Array))
	for index := range value.Array {
		output.Array[index] = cloneWorkflowValue(value.Array[index])
	}
	return output
}

func cloneWorkflowValuePointer(value *WorkflowValue) *WorkflowValue {
	if value == nil {
		return nil
	}
	output := cloneWorkflowValue(*value)
	return &output
}

func cloneWorkflowStepOutcomes(
	values []WorkflowStepOutcome,
) []WorkflowStepOutcome {
	output := make([]WorkflowStepOutcome, len(values))
	for index := range values {
		output[index] = WorkflowStepOutcome{
			ID:     values[index].ID,
			Status: values[index].Status,
			Output: values[index].Output.Clone(),
		}
	}
	return output
}

func cloneWorkflowSubscriptionStates(
	values []WorkflowSubscriptionState,
) []WorkflowSubscriptionState {
	output := make([]WorkflowSubscriptionState, len(values))
	for index := range values {
		output[index] = WorkflowSubscriptionState{
			Name:              values[index].Name,
			Event:             values[index].Event,
			Source:            values[index].Source,
			Correlation:       values[index].Correlation.Clone(),
			CorrelationSchema: values[index].CorrelationSchema.Clone(),
			DataSchema:        values[index].DataSchema.Clone(),
			Filter:            cloneWorkflowValuePointer(values[index].Filter),
		}
	}
	return output
}

func cloneWorkflowStringSet(
	input map[string]struct{},
) map[string]struct{} {
	output := make(map[string]struct{}, len(input))
	for value := range input {
		output[value] = struct{}{}
	}
	return output
}
