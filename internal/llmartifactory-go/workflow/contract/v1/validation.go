package workflowv1

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type validator struct {
	document WorkflowDocument
	steps    int
	paths    map[string]bool
}

func (v WorkflowDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: WorkflowType,
		RequireName:  true,
	}); err != nil {
		return err
	}
	if v.Locator != nil {
		return invalid("Workflow documents are concrete processes, not located aliases")
	}
	if len(v.InputSchema) == 0 || len(v.OutputSchema) == 0 || len(v.Output) == 0 || v.Steps == nil {
		return invalid("Workflow requires inputSchema, outputSchema, steps, and explicit output")
	}
	if err := v.validateSchemaFields(reflect.ValueOf(v), "workflow"); err != nil {
		return err
	}
	if err := validateLimits(v.Limits, nil); err != nil {
		return err
	}
	check := validator{document: v, paths: map[string]bool{}}
	if err := check.declarations(); err != nil {
		return err
	}
	scope := rootExpressionScope()
	if err := check.block(v.Block, scope, v.Limits); err != nil {
		return err
	}
	for _, name := range sortedKeys(v.Channels) {
		channel := v.Channels[name]
		for _, path := range append(append([]string(nil), channel.Readers...), channel.Writers...) {
			if !check.paths[path] {
				return invalid("channel %q authorizes unknown step path %q", name, path)
			}
		}
	}
	return v.validateLiteral(v.Output, v.OutputSchema)
}

func (v *validator) declarations() error {
	d := v.document
	for _, group := range []map[string]Target{d.Targets, d.Requirements} {
		for _, name := range sortedKeys(group) {
			if err := validateName("callable dependency name", name); err != nil {
				return err
			}
			target := group[name]
			if _, _, err := target.DeclarationMember(); err != nil {
				return fmt.Errorf("dependency %q: %w", name, err)
			}
		}
	}
	for _, name := range sortedKeys(d.TargetSets) {
		set := d.TargetSets[name]
		seen := map[string]bool{}
		for _, member := range set.Members {
			if _, found := d.Targets[member]; !found {
				return invalid("target set %q contains unknown target %q", name, member)
			}
			if seen[member] {
				return invalid("target set %q repeats target %q", name, member)
			}
			seen[member] = true
		}
	}
	for _, name := range sortedKeys(d.Ports) {
		if err := v.port(d.Ports[name], nil, ""); err != nil {
			return fmt.Errorf("published port %q: %w", name, err)
		}
	}
	for _, name := range sortedKeys(d.Resources) {
		resource := d.Resources[name]
		if resource.Rate != nil {
			if err := validateDuration(resource.Rate.Per, true); err != nil {
				return err
			}
		}
	}
	for _, name := range sortedKeys(d.Triggers) {
		trigger := d.Triggers[name]
		scope := rootExpressionScope()
		delete(scope.variables, "input")
		delete(scope.variables, "messages")
		scope.variables["event"] = true
		if trigger.Event != nil {
			value := trigger.Event
			if _, found := d.Events[value.Event]; !found {
				return invalid("trigger %q refers to unknown event %q", name, value.Event)
			}
			if err := scope.validate(value.Filter, "bool"); err != nil {
				return err
			}
			for _, binding := range []Value{value.Input, value.InstanceKey} {
				if err := scope.validate(binding, ""); err != nil {
					return err
				}
			}
			if err := d.validateLiteral(value.Input, d.InputSchema); err != nil {
				return fmt.Errorf("trigger %q input: %w", name, err)
			}
		}
		if trigger.Schedule != nil {
			value := trigger.Schedule
			delete(scope.variables, "event")
			if value.Calendar != "" {
				if err := validateCalendar(value.Calendar); err != nil {
					return err
				}
				if err := validateTimeZone(value.TimeZone); err != nil {
					return err
				}
			} else if err := validateDuration(value.Interval, true); err != nil {
				return err
			}
			if err := scope.validate(value.Input, ""); err != nil {
				return err
			}
			if err := d.validateLiteral(value.Input, d.InputSchema); err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *validator) state(value *State, scope expressionScope) (expressionScope, error) {
	if value == nil {
		return scope, nil
	}
	if err := scope.validate(value.Initial, ""); err != nil {
		return scope, err
	}
	if err := v.document.validateLiteral(value.Initial, value.Schema); err != nil {
		return scope, fmt.Errorf("state initial value: %w", err)
	}
	scope = scope.clone()
	scope.state = true
	return scope, nil
}

func (v *validator) listen(
	values map[string]Subscription,
	scope expressionScope,
) (expressionScope, error) {
	scope = scope.clone()
	for _, name := range sortedKeys(values) {
		value := values[name]
		if _, duplicate := scope.subscriptions[name]; duplicate {
			return scope, invalid("subscription %q shadows an enclosing subscription", name)
		}
		event, found := v.document.Events[value.Event]
		if !found {
			return scope, invalid("subscription %q refers to unknown event %q", name, value.Event)
		}
		if err := scope.validate(value.Correlation, ""); err != nil {
			return scope, err
		}
		if err := v.document.validateLiteral(value.Correlation, event.CorrelationSchema); err != nil {
			return scope, err
		}
		if err := scope.with("event").validate(value.Filter, "bool"); err != nil {
			return scope, err
		}
		if err := validateDuration(value.Retention, true); err != nil {
			return scope, err
		}
		scope.subscriptions[name] = value
	}
	return scope, nil
}

func (v *validator) block(block Block, scope expressionScope, limits *Limits) error {
	if scope.depth > 64 {
		return invalid("Workflow block nesting exceeds 64 levels")
	}
	if block.Steps == nil {
		return invalid("every block requires an explicit steps array")
	}
	scope = scope.clone()
	var err error
	scope, err = v.state(block.State, scope)
	if err != nil {
		return err
	}
	scope, err = v.listen(block.Listen, scope)
	if err != nil {
		return err
	}
	localIDs := map[string]bool{}
	for _, step := range block.Steps {
		if err := validateName("Workflow step ID", step.ID); err != nil {
			return err
		}
		if localIDs[step.ID] || scope.reserved[step.ID] {
			return invalid("step ID %q is duplicated or shadows an enclosing step", step.ID)
		}
		localIDs[step.ID] = true
	}
	for id := range localIDs {
		scope.reserved[id] = true
	}
	for _, step := range block.Steps {
		stepScope := scope.child(step.ID)
		v.paths[stepScope.path] = true
		v.steps++
		if v.steps > 16384 {
			return invalid("Workflow contains more than 16384 declared steps")
		}
		if err := v.step(step, stepScope, limits); err != nil {
			return fmt.Errorf("step %q: %w", stepScope.path, err)
		}
		scope.steps[step.ID] = true
	}
	if err := scope.validate(block.Output, ""); err != nil {
		return fmt.Errorf("block output: %w", err)
	}
	settled := scope.clone()
	for id := range localIDs {
		settled.steps[id] = true
	}
	for index, handler := range block.Catch {
		if err := v.block(handler.Do, settled.with("error").child(fmt.Sprintf("catch-%d", index)), limits); err != nil {
			return err
		}
	}
	if block.Finally != nil {
		if err := v.block(*block.Finally, settled.with("outcome").child("finally"), limits); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) step(step Step, scope expressionScope, parent *Limits) error {
	operations := []bool{
		step.Call != nil, step.Transform != nil, step.Set != nil,
		step.Scope != nil, step.Choose != nil, step.Parallel != nil,
		step.Foreach != nil, step.Repeat != nil, step.Machine != nil,
		step.Wait != nil, step.Emit != nil, step.Saga != nil,
		step.Cancel != nil, step.Return != nil, step.Fail != nil,
		step.Break != nil, step.Continue != nil,
	}
	count := 0
	for _, present := range operations {
		if present {
			count++
		}
	}
	if count != 1 {
		return invalid("a step requires exactly one operation")
	}
	if err := scope.validate(step.When, "bool"); err != nil {
		return err
	}
	if err := validateDuration(step.Timeout, true); err != nil {
		return err
	}
	if err := validateLimits(step.Limits, parent); err != nil {
		return err
	}
	limits := inheritLimits(parent, step.Limits)
	for _, claim := range step.Claims {
		resource, found := v.document.Resources[claim.Resource]
		if !found {
			return invalid("claim refers to unknown resource %q", claim.Resource)
		}
		units := 1
		if claim.Units != nil {
			units = *claim.Units
		}
		if resource.Concurrency != nil && units > *resource.Concurrency {
			return invalid("claim exceeds resource %q concurrency capacity", claim.Resource)
		}
		if resource.Keyed != (len(claim.Key) != 0) {
			return invalid("resource %q key requirement is not satisfied", claim.Resource)
		}
		if err := scope.validate(claim.Key, ""); err != nil {
			return err
		}
	}
	if step.Retry != nil {
		if err := validateDuration(step.Retry.PerAttemptTimeout, true); err != nil {
			return err
		}
		if step.Retry.Backoff != nil {
			if err := validateDuration(step.Retry.Backoff.Initial, false); err != nil {
				return err
			}
			if err := validateDuration(step.Retry.Backoff.Maximum, false); err != nil {
				return err
			}
		}
	}
	if step.Effect != nil {
		effect := step.Effect
		if err := scope.validate(effect.Key, ""); err != nil {
			return err
		}
		if effect.Kind == "keyed" && len(effect.Key) == 0 {
			return invalid("keyed effects require a key")
		}
		if effect.Reconcile != nil {
			if err := v.block(*effect.Reconcile, scope.with("operation").child("reconcile"), limits); err != nil {
				return err
			}
		}
		if effect.Compensate != nil {
			if err := v.block(*effect.Compensate, scope.with("completed").child("compensate"), limits); err != nil {
				return err
			}
		}
	}
	switch {
	case step.Call != nil:
		return v.call(*step.Call, step, scope)
	case step.Transform != nil:
		if err := scope.validate(step.Transform.Value, ""); err != nil {
			return err
		}
		if err := v.document.validateLiteral(step.Transform.Value, step.Transform.OutputSchema); err != nil {
			return err
		}
		return v.document.validateLiteral(step.Transform.Value, step.OutputSchema)
	case step.Set != nil:
		if !scope.state {
			return invalid("set requires an enclosing state-owning scope")
		}
		for _, name := range sortedKeys(step.Set.Values) {
			if slices.Contains(step.Set.Remove, name) {
				return invalid("set both assigns and removes field %q", name)
			}
			if err := scope.validate(step.Set.Values[name], ""); err != nil {
				return err
			}
		}
	case step.Scope != nil:
		return v.block(*step.Scope, scope, limits)
	case step.Choose != nil:
		for index, branch := range step.Choose.Cases {
			if err := scope.validate(branch.When, "bool"); err != nil {
				return err
			}
			if err := v.block(branch.Do, scope.child(fmt.Sprintf("case-%d", index)), limits); err != nil {
				return err
			}
		}
		if step.Choose.Default != nil {
			return v.block(*step.Choose.Default, scope.child("default"), limits)
		}
	case step.Parallel != nil:
		value := step.Parallel
		if err := completion(value.Completion, value.Errors, len(value.Branches)); err != nil {
			return err
		}
		for _, name := range sortedKeys(value.Branches) {
			branch := value.Branches[name]
			if err := scope.validate(branch.When, "bool"); err != nil {
				return err
			}
			if err := v.block(branch.Do, scope.child(name), limits); err != nil {
				return err
			}
		}
	case step.Foreach != nil:
		value := step.Foreach
		if err := scope.validate(value.Items, "list"); err != nil {
			return err
		}
		if err := completion(value.Completion, value.Errors, -1); err != nil {
			return err
		}
		if value.Concurrency != nil && limits != nil && limits.Concurrency != nil &&
			*value.Concurrency > *limits.Concurrency {
			return invalid("foreach concurrency exceeds inherited limit")
		}
		literal, known, err := literalValue(value.Items)
		if err != nil {
			return err
		}
		if known {
			items, ok := literal.([]any)
			if !ok {
				return invalid("foreach items must be a collection")
			}
			if value.MaxItems != nil && len(items) > *value.MaxItems {
				return invalid("literal foreach collection exceeds maxItems")
			}
			compiled, err := v.document.compileSchema(value.ItemSchema)
			if err != nil {
				return err
			}
			for index, item := range items {
				raw, err := json.Marshal(item)
				if err != nil {
					return err
				}
				if err := jsonutil.ValidateJSONSchema(
					compiled,
					json.RawMessage(raw),
					storeSpec.MaxDefinitionBodyBytes,
				); err != nil {
					return fmt.Errorf("foreach item %d: %w", index, err)
				}
			}
			if err := completion(value.Completion, value.Errors, len(items)); err != nil {
				return err
			}
		}
		return v.block(value.Do, scope.with("item").with("index").child("do"), limits)
	case step.Repeat != nil:
		value := step.Repeat
		controls := 0
		for _, present := range []bool{
			len(value.Count) != 0, len(value.While) != 0,
			len(value.Until) != 0, value.Lifetime != "",
		} {
			if present {
				controls++
			}
		}
		if controls != 1 {
			return invalid("repeat requires exactly one iteration control")
		}
		if err := scope.validate(value.Count, "int"); err != nil {
			return err
		}
		if err := validateDuration(value.Timeout, true); err != nil {
			return err
		}
		carried, err := v.state(value.State, scope)
		if err != nil {
			return err
		}
		carried.repeat = true
		carried = carried.with("index")
		if err := carried.validate(value.While, "bool"); err != nil {
			return err
		}
		if err := carried.with("result").validate(value.Until, "bool"); err != nil {
			return err
		}
		return v.block(value.Do, carried.child("do"), limits)
	case step.Machine != nil:
		return v.machine(*step.Machine, scope, limits)
	case step.Wait != nil:
		return v.wait(*step.Wait, scope)
	case step.Emit != nil:
		value := step.Emit
		for _, binding := range []Value{value.Data, value.Correlation, value.DeduplicationKey} {
			if err := scope.validate(binding, ""); err != nil {
				return err
			}
		}
		if value.To.Channel != "" {
			channel, err := v.channel(value.To.Channel, scope.path, true)
			if err != nil {
				return err
			}
			return v.document.validateLiteral(value.Data, channel.Schema)
		}
		event, found := v.document.Events[value.To.Event]
		if !found {
			return invalid("emit refers to unknown event %q", value.To.Event)
		}
		if err := v.document.validateLiteral(value.Correlation, event.CorrelationSchema); err != nil {
			return err
		}
		return v.document.validateLiteral(value.Data, event.Schema)
	case step.Saga != nil:
		return v.block(step.Saga.Do, scope.child("do"), limits)
	case step.Cancel != nil:
		if err := scope.validate(step.Cancel.Handle, ""); err != nil {
			return err
		}
		return scope.validate(step.Cancel.Reason, "")
	case step.Return != nil:
		if err := scope.validate(step.Return.Value, ""); err != nil {
			return err
		}
		if step.Return.To == "workflow" {
			return v.document.validateLiteral(step.Return.Value, v.document.OutputSchema)
		}
	case step.Fail != nil:
		if err := scope.validate(step.Fail.Message, "string"); err != nil {
			return err
		}
		return scope.validate(step.Fail.Details, "")
	case step.Break != nil, step.Continue != nil:
		if !scope.repeat {
			return invalid("break and continue require an enclosing repeat")
		}
	}
	return nil
}

func completion(value *Completion, errors string, count int) error {
	mode := "all"
	if value != nil {
		mode = value.Mode
	}
	if mode != "all" && errors != "" {
		return invalid("errors is configurable only for all completion")
	}
	if mode == "quorum" {
		if value.Quorum == nil || *value.Quorum <= 0 {
			return invalid("quorum completion requires a positive quorum")
		}
		if count >= 0 && *value.Quorum > count {
			return invalid("completion quorum exceeds the declared collection size")
		}
	}
	return nil
}

func (v *validator) machine(machine Machine, scope expressionScope, limits *Limits) error {
	if _, found := machine.States[machine.Initial]; !found {
		return invalid("machine initial state %q does not exist", machine.Initial)
	}
	scope, err := v.state(machine.State, scope)
	if err != nil {
		return err
	}
	scope, err = v.listen(machine.Listen, scope)
	if err != nil {
		return err
	}
	for _, name := range sortedKeys(machine.States) {
		state := machine.States[name]
		local := scope.child(name)
		if state.Entry != nil {
			if err := v.block(*state.Entry, local.child("entry"), limits); err != nil {
				return err
			}
		}
		if state.Final {
			if len(state.Output) == 0 || len(state.Transitions) != 0 || state.Exit != nil {
				return invalid("final machine state %q requires output and cannot exit or transition", name)
			}
			if err := local.validate(state.Output, ""); err != nil {
				return err
			}
			continue
		}
		if len(state.Transitions) == 0 || len(state.Output) != 0 {
			return invalid("non-final machine state %q requires transitions and cannot declare output", name)
		}
		if state.Exit != nil {
			if err := v.block(*state.Exit, local.child("exit"), limits); err != nil {
				return err
			}
		}
		for index, transition := range state.Transitions {
			if _, found := machine.States[transition.To]; !found {
				return invalid("machine transition destination %q does not exist", transition.To)
			}
			eventScope := local
			if transition.Event != "" {
				if _, found := scope.subscriptions[transition.Event]; !found {
					return invalid("machine transition uses unknown subscription %q", transition.Event)
				}
				eventScope = eventScope.with("event")
			}
			if transition.Event != "" && transition.After != "" ||
				transition.Event == "" && transition.After == "" && len(transition.When) == 0 {
				return invalid("transition requires an event, timer, or immediate condition")
			}
			if err := validateDuration(transition.After, false); err != nil {
				return err
			}
			if err := eventScope.validate(transition.When, "bool"); err != nil {
				return err
			}
			if transition.Do != nil {
				if err := v.block(
					*transition.Do,
					eventScope.child(fmt.Sprintf("transition-%d", index)),
					limits,
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *validator) wait(wait Wait, scope expressionScope) error {
	switch wait.For {
	case "timer":
		for _, value := range []Value{wait.After, wait.At} {
			if err := scope.validate(value, "string"); err != nil {
				return err
			}
		}
		if literal, known, _ := literalValue(wait.After); known {
			text, ok := literal.(string)
			if !ok {
				return invalid("timer after must be a duration")
			}
			return validateDuration(text, false)
		}
		if literal, known, _ := literalValue(wait.At); known {
			text, ok := literal.(string)
			if !ok {
				return invalid("timer at must be an RFC 3339 timestamp")
			}
			if _, err := time.Parse(time.RFC3339Nano, text); err != nil {
				return invalid("timer at must be an RFC 3339 timestamp")
			}
		}
	case "event":
		if _, found := scope.subscriptions[wait.Subscription]; !found {
			return invalid("wait refers to inactive subscription %q", wait.Subscription)
		}
		return scope.with("event").validate(wait.Match, "bool")
	case "channel":
		if _, err := v.channel(wait.Channel, scope.path, false); err != nil {
			return err
		}
		if err := scope.validate(wait.Cursor, ""); err != nil {
			return err
		}
		return scope.with("event").validate(wait.Match, "bool")
	case "completion":
		return scope.validate(wait.Handle, "")
	case "human":
		if wait.Assignees == nil {
			return invalid("human wait requires assignees")
		}
		if err := scope.validate(wait.Assignees.Users, "list"); err != nil {
			return err
		}
		if err := scope.validate(wait.Assignees.Roles, "list"); err != nil {
			return err
		}
		return scope.validate(wait.Request, "")
	default:
		return invalid("unknown wait kind %q", wait.For)
	}
	return nil
}

func (v *validator) channel(name, path string, write bool) (Channel, error) {
	channel, found := v.document.Channels[name]
	if !found {
		return Channel{}, invalid("unknown channel %q", name)
	}
	allowed := channel.Readers
	if write {
		allowed = channel.Writers
	}
	if len(allowed) != 0 && path != "" && !slices.Contains(allowed, path) {
		return Channel{}, invalid("step %q has no channel %q permission", path, name)
	}
	return channel, nil
}

func (v *validator) port(binding PortBinding, port *CallablePort, path string) error {
	read := binding.Access == "read" || binding.Access == "read-write"
	write := binding.Access == "write" || binding.Access == "read-write"
	channel, err := v.channel(binding.Channel, path, write)
	if err != nil {
		return err
	}
	if read {
		if _, err := v.channel(binding.Channel, path, false); err != nil {
			return err
		}
	}
	if port != nil {
		if port.Direction == "input" && (!read || write) ||
			port.Direction == "output" && (!write || read) {
			return invalid("call port direction and channel access disagree")
		}
		return rejectDisjointSchemas(port.Schema, channel.Schema)
	}
	return nil
}

func (v *validator) call(call Call, step Step, scope expressionScope) error {
	if err := scope.validate(call.Target, "string"); err != nil {
		return err
	}
	if err := scope.validate(call.Input, ""); err != nil {
		return err
	}
	for _, name := range sortedKeys(call.Context) {
		if err := scope.validate(call.Context[name], ""); err != nil {
			return err
		}
	}
	for _, value := range []Value{call.Conversation} {
		if err := scope.validate(value, ""); err != nil {
			return err
		}
	}
	if call.Ownership != nil {
		if err := scope.validate(call.Ownership.Owner, ""); err != nil {
			return err
		}
	}
	var allowed *TargetSet
	if call.Allowed != "" {
		set, found := v.document.TargetSets[call.Allowed]
		if !found {
			return invalid("unknown allowed target set %q", call.Allowed)
		}
		allowed = &set
		if set.Resolved != nil {
			if err := scope.validate(set.Resolved.Value, "list"); err != nil {
				return err
			}
			if err := scope.validate(set.Resolved.Constraints, ""); err != nil {
				return err
			}
			if err := v.callInterface(call, &set.Resolved.Interface, scope); err != nil {
				return err
			}
		}
		if set.Interface != nil {
			if err := v.callInterface(call, set.Interface, scope); err != nil {
				return err
			}
		}
	}

	var targets []Target
	literal, known, err := literalValue(call.Target)
	if err != nil {
		return err
	}
	if known {
		name, ok := literal.(string)
		if !ok {
			return invalid("call target must be a target name")
		}
		target, found := v.document.Targets[name]
		if !found {
			return invalid("call refers to unknown target %q", name)
		}
		targets = append(targets, target)
		if allowed != nil &&
			allowed.Resolved == nil &&
			!slices.Contains(allowed.Members, name) {
			return invalid("target %q does not belong to set %q", name, call.Allowed)
		}
	} else {
		if allowed == nil {
			return invalid("dynamic calls require a declared allowed target set")
		}
		for _, name := range allowed.Members {
			targets = append(targets, v.document.Targets[name])
		}
	}
	mode := call.Mode
	if mode == "" {
		mode = "invoke"
	}
	for _, target := range targets {
		if mode == "handoff" && target.Kind != TargetAgent {
			return invalid("handoff is valid only for Agent targets")
		}
		if err := v.callInterface(call, target.Interface, scope); err != nil {
			return err
		}
		if target.Interface != nil && target.Interface.Effect != nil &&
			step.Retry != nil && step.Retry.MaxAttempts > 1 &&
			target.Interface.Effect.Kind == "non-idempotent" {
			safe := step.Effect != nil &&
				(step.Effect.Reconcile != nil ||
					step.Effect.Kind == "keyed" && target.Interface.Effect.Keyed)
			if !safe {
				return invalid("non-idempotent target retry requires supported keying or reconciliation")
			}
		}
	}
	if err := v.callInterface(call, call.Interface, scope); err != nil {
		return err
	}
	for _, name := range sortedKeys(call.Ports) {
		if err := v.port(call.Ports[name], nil, scope.path); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) callInterface(call Call, contract *CallableInterface, scope expressionScope) error {
	if contract == nil {
		// Binding-owned interfaces are checked by future binding preparation,
		// not invented by declaration admission.
		return nil
	}
	mode := call.Mode
	if mode == "" {
		mode = "invoke"
	}
	if len(contract.Modes) != 0 && !slices.Contains(contract.Modes, mode) {
		return invalid("target interface does not support mode %q", mode)
	}
	if err := v.document.validateLiteral(call.Input, contract.InputSchema); err != nil {
		return fmt.Errorf("call input: %w", err)
	}
	for _, name := range sortedKeys(call.Context) {
		schema, found := contract.Context[name]
		if !found {
			return invalid("target interface does not accept context form %q", name)
		}
		if err := v.document.validateLiteral(call.Context[name], schema); err != nil {
			return err
		}
	}
	for _, name := range sortedKeys(call.Ports) {
		port, found := contract.Ports[name]
		if !found {
			return invalid("target interface has no port %q", name)
		}
		if err := v.port(call.Ports[name], &port, scope.path); err != nil {
			return err
		}
	}
	if call.Interface != nil && call.Interface != contract {
		if err := rejectDisjointSchemas(call.Interface.InputSchema, contract.InputSchema); err != nil {
			return err
		}
		if err := rejectDisjointSchemas(call.Interface.OutputSchema, contract.OutputSchema); err != nil {
			return err
		}
	}
	return nil
}

// DeclarationMember projects only real source-backed declaration dependencies.
// Service and activity references remain explicit binding requirements.
func (t Target) DeclarationMember() (declaration.Entry, bool, error) {
	if err := storeSpec.ValidateRequiredText("callable reference", t.Ref, storeSpec.MaxURIBytes); err != nil {
		return declaration.Entry{}, false, err
	}
	if t.Kind == TargetService || t.Kind == TargetActivity {
		if t.Scope != "" || t.Locator != nil || t.Server != nil {
			return declaration.Entry{}, false, invalid(
				"service/activity targets cannot contain Artifact selection fields",
			)
		}
		return declaration.Entry{}, false, nil
	}
	kind := declaration.Type(t.Kind)
	switch t.Kind {
	case TargetAgent, TargetTool, TargetModel, TargetWorkflow:
	default:
		return declaration.Entry{}, false, invalid("unsupported target kind %q", t.Kind)
	}
	reference := Reference{Name: t.Ref, Scope: t.Scope, Locator: t.Locator}
	if t.Server != nil {
		if t.Kind != TargetTool || t.Scope != "" || t.Locator != nil {
			return declaration.Entry{}, false, invalid("only Tool targets can select an MCP server")
		}
		kind = declaration.TypeMCP
		reference = *t.Server
	}
	if err := validateName("callable declaration reference", reference.Name); err != nil {
		return declaration.Entry{}, false, err
	}
	fields := map[string]any{"type": string(kind), "name": reference.Name}
	if reference.Scope != "" {
		fields["scope"] = string(reference.Scope)
	}
	if reference.Locator != nil {
		fields["locator"] = reference.Locator
	}
	if reference.Server != "" {
		if kind != declaration.TypeMCP {
			return declaration.Entry{}, false, invalid("server selector requires MCP")
		}
		fields["server"] = reference.Server
	}
	entry, err := declaration.NewEntry(fields)
	if err != nil {
		return declaration.Entry{}, false, err
	}
	if _, err := entry.MemberForm(); err != nil {
		return declaration.Entry{}, false, err
	}
	if strings.Contains(reference.Name, "/") {
		return declaration.Entry{}, false, invalid("declaration reference name is not portable")
	}
	return entry, true, nil
}
