package sessionv3

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	quorumStr         = "quorum"
	allStr            = "all"
	cancelStr         = "cancel"
	firstCompletedStr = "first-completed"
	firstSuccessStr   = "first-success"
)

// WorkflowScopeContext is the durable parent visibility envelope for an
// interpreter-created child block frame. It contains semantic values only.
type WorkflowScopeContext struct {
	Child         bool                  `json:"child,omitempty"`
	MergeState    bool                  `json:"mergeState,omitempty"`
	OuterState    CanonicalValue        `json:"outerState,omitempty"`
	OuterOutcomes []WorkflowStepOutcome `json:"outerOutcomes,omitempty"`
	Item          *CanonicalValue       `json:"item,omitempty"`
	Index         *int                  `json:"index,omitempty"`
	Result        *CanonicalValue       `json:"result,omitempty"`
}

func (s WorkflowScopeContext) Validate() error {
	if !s.OuterState.Valid() {
		return errors.New("Workflow scope outer state is invalid")
	}
	if s.Item != nil && !s.Item.Valid() {
		return errors.New("Workflow scope item is invalid")
	}
	if s.Result != nil && !s.Result.Valid() {
		return errors.New("Workflow scope result is invalid")
	}
	if s.Index != nil && *s.Index < 0 {
		return errors.New("Workflow scope index is invalid")
	}

	seen := make(map[string]struct{}, len(s.OuterOutcomes))
	for _, outcome := range s.OuterOutcomes {
		if strings.TrimSpace(outcome.ID) == "" || !outcome.Output.Valid() {
			return errors.New("Workflow scope outcome is invalid")
		}
		if _, duplicate := seen[outcome.ID]; duplicate {
			return errors.New("Workflow scope outcome is duplicated")
		}
		seen[outcome.ID] = struct{}{}
	}
	return nil
}

type WorkflowReturnDestination string

const (
	WorkflowReturnDestinationBlock    WorkflowReturnDestination = "block"
	WorkflowReturnDestinationWorkflow WorkflowReturnDestination = "workflow"
)

type WorkflowExitKind string

const (
	WorkflowExitNormal         WorkflowExitKind = "normal"
	WorkflowExitReturnBlock    WorkflowExitKind = "returnBlock"
	WorkflowExitReturnWorkflow WorkflowExitKind = "returnWorkflow"
	WorkflowExitBreak          WorkflowExitKind = "break"
	WorkflowExitContinue       WorkflowExitKind = "continue"
)

type WorkflowBlockResult struct {
	Exit  WorkflowExitKind `json:"exit"`
	Value CanonicalValue   `json:"value"`
	State CanonicalValue   `json:"state"`
}

func (r WorkflowBlockResult) Validate() error {
	switch r.Exit {
	case WorkflowExitNormal,
		WorkflowExitReturnBlock,
		WorkflowExitReturnWorkflow,
		WorkflowExitBreak,
		WorkflowExitContinue:
	default:
		return errors.New("Workflow block result exit is invalid")
	}
	if !r.Value.Valid() || !r.State.Valid() {
		return errors.New("Workflow block result contains invalid JSON")
	}
	return nil
}

type WorkflowChoose struct {
	Cases   []WorkflowCase `json:"cases"`
	Default *WorkflowBlock `json:"default,omitempty"`
}

type WorkflowCase struct {
	When WorkflowValue `json:"when"`
	Do   WorkflowBlock `json:"do"`
}

type WorkflowCompletion struct {
	Mode   string `json:"mode,omitempty"`
	Quorum int    `json:"quorum,omitempty"`
}

type WorkflowBranch struct {
	Name string         `json:"name"`
	When *WorkflowValue `json:"when,omitempty"`
	Do   WorkflowBlock  `json:"do"`
}

type WorkflowParallel struct {
	Completion WorkflowCompletion `json:"completion"`
	Errors     string             `json:"errors,omitempty"`
	Remaining  string             `json:"remaining,omitempty"`
	Branches   []WorkflowBranch   `json:"branches"`
}

type WorkflowForeach struct {
	Items       WorkflowValue      `json:"items"`
	ItemSchema  CanonicalValue     `json:"itemSchema"`
	Concurrency int                `json:"concurrency,omitempty"`
	MaxItems    int                `json:"maxItems,omitempty"`
	Completion  WorkflowCompletion `json:"completion"`
	Errors      string             `json:"errors,omitempty"`
	Remaining   string             `json:"remaining,omitempty"`
	Do          WorkflowBlock      `json:"do"`
}

type WorkflowRepeat struct {
	Count         *WorkflowValue     `json:"count,omitempty"`
	While         *WorkflowValue     `json:"while,omitempty"`
	Until         *WorkflowValue     `json:"until,omitempty"`
	Lifetime      string             `json:"lifetime,omitempty"`
	State         *WorkflowStateSpec `json:"state,omitempty"`
	MaxIterations int                `json:"maxIterations,omitempty"`
	OnExhausted   string             `json:"onExhausted,omitempty"`
	Do            WorkflowBlock      `json:"do"`
}

type WorkflowPendingKind string

const (
	WorkflowPendingScope    WorkflowPendingKind = "scope"
	WorkflowPendingChoose   WorkflowPendingKind = "choose"
	WorkflowPendingParallel WorkflowPendingKind = "parallel"
	WorkflowPendingForeach  WorkflowPendingKind = "foreach"
	WorkflowPendingRepeat   WorkflowPendingKind = "repeat"
)

type WorkflowChildState string

const (
	WorkflowChildStatePending   WorkflowChildState = "pending"
	WorkflowChildStateSucceeded WorkflowChildState = "succeeded"
	WorkflowChildStateFailed    WorkflowChildState = "failed"
	WorkflowChildStateCancelled WorkflowChildState = "cancelled"
	WorkflowChildStateSkipped   WorkflowChildState = "skipped"
)

func (s WorkflowChildState) terminal() bool {
	switch s {
	case WorkflowChildStateSucceeded,
		WorkflowChildStateFailed,
		WorkflowChildStateCancelled,
		WorkflowChildStateSkipped:
		return true
	default:
		return false
	}
}

type WorkflowChild struct {
	Name   string               `json:"name"`
	Path   string               `json:"path,omitempty"`
	State  WorkflowChildState   `json:"state"`
	Result *WorkflowBlockResult `json:"result,omitempty"`
	Error  *SanitizedError      `json:"error,omitempty"`
}

type WorkflowNamedOutcome struct {
	Name   string             `json:"name"`
	State  WorkflowChildState `json:"state"`
	Output CanonicalValue     `json:"output,omitempty"`
	Error  *SanitizedError    `json:"error,omitempty"`
}

type WorkflowParallelProgress struct {
	Completion WorkflowCompletion `json:"completion"`
	Errors     string             `json:"errors,omitempty"`
	Remaining  string             `json:"remaining,omitempty"`
	Winners    []string           `json:"winners,omitempty"`
}

type WorkflowItemOutcome struct {
	Index  int                `json:"index"`
	State  WorkflowChildState `json:"state"`
	Output CanonicalValue     `json:"output,omitempty"`
	Error  *SanitizedError    `json:"error,omitempty"`
}

type WorkflowForeachProgress struct {
	Items       []CanonicalValue      `json:"items"`
	Outcomes    []WorkflowItemOutcome `json:"outcomes"`
	Next        int                   `json:"next"`
	Concurrency int                   `json:"concurrency"`
	Completion  WorkflowCompletion    `json:"completion"`
	Errors      string                `json:"errors,omitempty"`
	Remaining   string                `json:"remaining,omitempty"`
	Winners     []int                 `json:"winners,omitempty"`
}

type WorkflowRepeatProgress struct {
	State      CanonicalValue `json:"state"`
	Result     CanonicalValue `json:"result,omitempty"`
	Iterations int            `json:"iterations"`
	Count      *int           `json:"count,omitempty"`
	LocalState bool           `json:"localState"`
	LaunchNext bool           `json:"launchNext"`
}

type WorkflowPending struct {
	Kind       WorkflowPendingKind       `json:"kind"`
	StepID     string                    `json:"stepID"`
	Path       string                    `json:"path"`
	MergeState bool                      `json:"mergeState,omitempty"`
	Children   []WorkflowChild           `json:"children,omitempty"`
	Parallel   *WorkflowParallelProgress `json:"parallel,omitempty"`
	Foreach    *WorkflowForeachProgress  `json:"foreach,omitempty"`
	Repeat     *WorkflowRepeatProgress   `json:"repeat,omitempty"`
}

func (p WorkflowPending) Validate() error {
	if strings.TrimSpace(p.StepID) == "" || strings.TrimSpace(p.Path) == "" {
		return errors.New("Workflow pending control has no step identity")
	}
	switch p.Kind {
	case WorkflowPendingScope, WorkflowPendingChoose:
		if len(p.Children) != 1 {
			return errors.New("Workflow pending block has an invalid child count")
		}
	case WorkflowPendingParallel:
		if p.Parallel == nil {
			return errors.New("Workflow pending parallel state is missing")
		}
	case WorkflowPendingForeach:
		if p.Foreach == nil {
			return errors.New("Workflow pending foreach state is missing")
		}
	case WorkflowPendingRepeat:
		if p.Repeat == nil {
			return errors.New("Workflow pending repeat state is missing")
		}
	default:
		return errors.New("Workflow pending control kind is invalid")
	}

	seen := make(map[string]struct{}, len(p.Children))
	for _, child := range p.Children {
		if strings.TrimSpace(child.Name) == "" {
			return errors.New("Workflow pending child has no name")
		}
		if child.State == "" {
			return errors.New("Workflow pending child has no state")
		}
		if _, duplicate := seen[child.Name]; duplicate {
			return errors.New("Workflow pending child is duplicated")
		}
		seen[child.Name] = struct{}{}
		if child.Result != nil {
			if err := child.Result.Validate(); err != nil {
				return err
			}
		}
		if child.Error != nil && !child.Error.Details.Valid() {
			return errors.New("Workflow pending child error details are invalid")
		}
	}
	return nil
}

type workflowChildError struct {
	errorValue SanitizedError
}

func (e workflowChildError) Error() string {
	return e.errorValue.Message
}

func (i *WorkflowInterpreter) completeWorkflowBlock(
	program WorkflowProgram,
	state WorkflowState,
	value CanonicalValue,
	exit WorkflowExitKind,
) (ProgramTransition, error) {
	if !value.Valid() {
		return ProgramTransition{}, errors.New("Workflow block completion value is invalid")
	}

	if program.Scope.Child {
		result := WorkflowBlockResult{
			Exit:  exit,
			Value: value.Clone(),
			State: state.State.Clone(),
		}
		if err := result.Validate(); err != nil {
			return ProgramTransition{}, err
		}
		status := ProgramFrameStateCompleted
		return workflowTransition(
			state,
			&status,
			MustCanonicalValue(result),
			nil,
		)
	}

	switch exit {
	case WorkflowExitNormal,
		WorkflowExitReturnBlock,
		WorkflowExitReturnWorkflow:
	default:
		return workflowFailure(
			state,
			"workflow_control_invalid",
			"Workflow control escaped the root block",
		)
	}
	if err := validateWorkflowSchema(program.OutputSchema, value); err != nil {
		return workflowFailure(state, "workflow_output_invalid", err.Error())
	}

	status := ProgramFrameStateCompleted
	return workflowTransition(state, &status, value, nil)
}

func workflowStructuredTransition(
	state WorkflowState,
	frames []ProgramFrameIntent,
) (ProgramTransition, error) {
	next, err := encodeWorkflowProgramState(state)
	if err != nil {
		return ProgramTransition{}, err
	}
	return ProgramTransition{
		State:  next,
		Frames: frames,
	}, nil
}

func workflowScopeOutcomes(
	program WorkflowProgram,
	state WorkflowState,
) []WorkflowStepOutcome {
	values := make(
		[]WorkflowStepOutcome,
		0,
		len(program.Scope.OuterOutcomes)+len(state.Outcomes),
	)
	for _, outcome := range program.Scope.OuterOutcomes {
		values = append(values, WorkflowStepOutcome{
			ID:     outcome.ID,
			Status: outcome.Status,
			Output: outcome.Output.Clone(),
		})
	}
	for _, outcome := range state.Outcomes {
		values = append(values, WorkflowStepOutcome{
			ID:     outcome.ID,
			Status: outcome.Status,
			Output: outcome.Output.Clone(),
		})
	}
	return values
}

func workflowCanonicalPointer(value CanonicalValue) *CanonicalValue {
	if len(value) == 0 {
		return nil
	}
	copyValue := value.Clone()
	return &copyValue
}

func workflowIntegerPointer(value int) *int {
	copyValue := value
	return &copyValue
}

func (i *WorkflowInterpreter) childBlockProgram(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	block WorkflowBlock,
	mergeState bool,
	item *CanonicalValue,
	index *int,
	result *CanonicalValue,
) (ProgramSnapshot, error) {
	child := program
	child.Root = block
	child.Scope = WorkflowScopeContext{
		Child:         true,
		MergeState:    mergeState,
		OuterState:    state.State.Clone(),
		OuterOutcomes: workflowScopeOutcomes(program, state),
		Item:          item,
		Index:         index,
		Result:        result,
	}
	child.Input = program.Input.Clone()
	child.InputBound = true

	if err := child.Validate(); err != nil {
		return ProgramSnapshot{}, err
	}

	payload, err := canonicalWorkflowValue(child)
	if err != nil {
		return ProgramSnapshot{}, err
	}

	snapshot := frame.Program
	snapshot.Payload = payload
	return snapshot, nil
}

func (i *WorkflowInterpreter) startSingleChild(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	path string,
	kind WorkflowPendingKind,
	childName string,
	block WorkflowBlock,
	mergeState bool,
	item *CanonicalValue,
	index *int,
	result *CanonicalValue,
) (ProgramTransition, error) {
	snapshot, err := i.childBlockProgram(
		frame,
		program,
		state,
		block,
		mergeState,
		item,
		index,
		result,
	)
	if err != nil {
		return workflowFailure(state, "workflow_child_prepare_failed", err.Error())
	}

	childPath := path + "/" + childName
	state.Pending = &WorkflowPending{
		Kind:       kind,
		StepID:     step.ID,
		Path:       path,
		MergeState: mergeState,
		Children: []WorkflowChild{{
			Name:  childName,
			Path:  childPath,
			State: WorkflowChildStatePending,
		}},
	}
	return workflowStructuredTransition(
		state,
		[]ProgramFrameIntent{{
			Path:     childPath,
			Program:  snapshot,
			Deadline: cloneTime(state.Deadline),
		}},
	)
}

func (i *WorkflowInterpreter) startScope(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	path string,
) (ProgramTransition, error) {
	if step.Scope == nil {
		return workflowFailure(state, "workflow_scope_invalid", "Workflow scope is missing")
	}
	return i.startSingleChild(
		frame,
		program,
		state,
		step,
		path,
		WorkflowPendingScope,
		"scope",
		*step.Scope,
		step.Scope.State == nil,
		nil,
		nil,
		nil,
	)
}

func (i *WorkflowInterpreter) startChoose(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	path string,
) (ProgramTransition, error) {
	if step.Choose == nil {
		return workflowFailure(state, "workflow_choose_invalid", "Workflow choose is missing")
	}

	for index, candidate := range step.Choose.Cases {
		selected, err := i.condition(candidate.When, program, state)
		if err != nil {
			return workflowFailure(state, "workflow_choose_condition_failed", err.Error())
		}
		if !selected {
			continue
		}
		return i.startSingleChild(
			frame,
			program,
			state,
			step,
			path,
			WorkflowPendingChoose,
			"case-"+strconv.Itoa(index),
			candidate.Do,
			candidate.Do.State == nil,
			nil,
			nil,
			nil,
		)
	}

	if step.Choose.Default == nil {
		return workflowFailure(
			state,
			"routing_no_match",
			"Workflow choose had no matching case or default",
		)
	}
	return i.startSingleChild(
		frame,
		program,
		state,
		step,
		path,
		WorkflowPendingChoose,
		"default",
		*step.Choose.Default,
		step.Choose.Default.State == nil,
		nil,
		nil,
		nil,
	)
}

func workflowCompletionMode(value WorkflowCompletion) string {
	if value.Mode == "" {
		return allStr
	}
	return value.Mode
}

func workflowRemainingMode(value string) string {
	if value == "" {
		return cancelStr
	}
	return value
}

func (i *WorkflowInterpreter) startParallel(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	path string,
) (ProgramTransition, error) {
	if step.Parallel == nil {
		return workflowFailure(state, "workflow_parallel_invalid", "Workflow parallel is missing")
	}

	pending := &WorkflowPending{
		Kind:   WorkflowPendingParallel,
		StepID: step.ID,
		Path:   path,
		Parallel: &WorkflowParallelProgress{
			Completion: step.Parallel.Completion,
			Errors:     step.Parallel.Errors,
			Remaining:  step.Parallel.Remaining,
		},
	}
	frames := make([]ProgramFrameIntent, 0, len(step.Parallel.Branches))

	for _, branch := range step.Parallel.Branches {
		selected := true
		if branch.When != nil {
			value, err := i.condition(*branch.When, program, state)
			if err != nil {
				return workflowFailure(
					state,
					"workflow_parallel_condition_failed",
					err.Error(),
				)
			}
			selected = value
		}

		child := WorkflowChild{
			Name:  branch.Name,
			State: WorkflowChildStateSkipped,
		}
		if selected {
			snapshot, err := i.childBlockProgram(
				frame,
				program,
				state,
				branch.Do,
				false,
				nil,
				nil,
				nil,
			)
			if err != nil {
				return workflowFailure(
					state,
					"workflow_parallel_prepare_failed",
					err.Error(),
				)
			}
			child.Path = path + "/branch/" + branch.Name
			child.State = WorkflowChildStatePending
			frames = append(frames, ProgramFrameIntent{
				Path:     child.Path,
				Program:  snapshot,
				Deadline: cloneTime(state.Deadline),
			})
		}
		pending.Children = append(pending.Children, child)
	}

	state.Pending = pending
	if len(frames) == 0 {
		if err := i.finishParallel(&state, program, step); err != nil {
			if failure, ok := errors.AsType[workflowChildError](err); ok {
				return workflowFailureTransition(state, &failure.errorValue)
			}
			return workflowFailure(state, "workflow_parallel_failed", err.Error())
		}
		return i.drive(frame, Execution{}, program, state)
	}
	return workflowStructuredTransition(state, frames)
}

func (i *WorkflowInterpreter) startForeach(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	path string,
) (ProgramTransition, error) {
	if step.Foreach == nil {
		return workflowFailure(state, "workflow_foreach_invalid", "Workflow foreach is missing")
	}

	value, err := i.evaluate(step.Foreach.Items, program, state)
	if err != nil {
		return workflowFailure(state, "workflow_foreach_items_invalid", err.Error())
	}
	items, err := DecodeCanonicalValue[[]CanonicalValue](value)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_foreach_items_invalid",
			"Workflow foreach items must be an array",
		)
	}
	if step.Foreach.MaxItems > 0 && len(items) > step.Foreach.MaxItems {
		return workflowFailure(
			state,
			"workflow_foreach_limit_exceeded",
			"Workflow foreach item count exceeds maxItems",
		)
	}
	for index := range items {
		if err := validateWorkflowSchema(step.Foreach.ItemSchema, items[index]); err != nil {
			return workflowFailure(
				state,
				"workflow_foreach_item_invalid",
				fmt.Sprintf("validate foreach item %d: %v", index, err),
			)
		}
	}

	concurrency := step.Foreach.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	outcomes := make([]WorkflowItemOutcome, len(items))
	for index := range outcomes {
		outcomes[index] = WorkflowItemOutcome{
			Index: index,
			State: WorkflowChildStatePending,
		}
	}

	state.Pending = &WorkflowPending{
		Kind:   WorkflowPendingForeach,
		StepID: step.ID,
		Path:   path,
		Foreach: &WorkflowForeachProgress{
			Items:       append([]CanonicalValue(nil), items...),
			Outcomes:    outcomes,
			Concurrency: concurrency,
			Completion:  step.Foreach.Completion,
			Errors:      step.Foreach.Errors,
			Remaining:   step.Foreach.Remaining,
		},
	}
	return i.drivePending(frame, Execution{}, program, state)
}

func (i *WorkflowInterpreter) startRepeat(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	path string,
) (ProgramTransition, error) {
	if step.Repeat == nil {
		return workflowFailure(state, "workflow_repeat_invalid", "Workflow repeat is missing")
	}

	progress := &WorkflowRepeatProgress{
		State:      state.State.Clone(),
		LocalState: step.Repeat.State != nil,
		LaunchNext: true,
	}
	if step.Repeat.State != nil {
		value, err := i.evaluate(step.Repeat.State.Initial, program, state)
		if err != nil {
			return workflowFailure(
				state,
				"workflow_repeat_state_invalid",
				err.Error(),
			)
		}
		if err := validateWorkflowSchema(step.Repeat.State.Schema, value); err != nil {
			return workflowFailure(
				state,
				"workflow_repeat_state_invalid",
				err.Error(),
			)
		}
		progress.State = value
	}
	if step.Repeat.Count != nil {
		value, err := i.evaluate(*step.Repeat.Count, program, state)
		if err != nil {
			return workflowFailure(state, "workflow_repeat_count_invalid", err.Error())
		}
		count, err := DecodeCanonicalValue[int](value)
		if err != nil || count < 0 {
			return workflowFailure(
				state,
				"workflow_repeat_count_invalid",
				"Workflow repeat count must be a non-negative integer",
			)
		}
		progress.Count = workflowIntegerPointer(count)
	}

	state.Pending = &WorkflowPending{
		Kind:   WorkflowPendingRepeat,
		StepID: step.ID,
		Path:   path,
		Repeat: progress,
	}
	return i.drivePending(frame, Execution{}, program, state)
}

func (i *WorkflowInterpreter) drivePending(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
) (ProgramTransition, error) {
	if state.Pending == nil {
		return i.drive(frame, execution, program, state)
	}

	switch state.Pending.Kind {
	case WorkflowPendingForeach:
		return i.launchForeachChildren(frame, execution, program, state)
	case WorkflowPendingRepeat:
		return i.launchRepeatChild(frame, execution, program, state)
	default:
		return workflowTransition(state, nil, nil, nil)
	}
}

func workflowActiveChildren(values []WorkflowChild) int {
	count := 0
	for _, child := range values {
		if child.State == WorkflowChildStatePending {
			count++
		}
	}
	return count
}

func (i *WorkflowInterpreter) launchForeachChildren(
	frame ProgramFrame,
	_ Execution,
	program WorkflowProgram,
	state WorkflowState,
) (ProgramTransition, error) {
	pending := state.Pending
	if pending == nil || pending.Foreach == nil {
		return workflowFailure(state, "workflow_foreach_invalid", "Workflow foreach state is missing")
	}
	step, err := workflowStepForPath(program.Root, pending.Path)
	if err != nil || step.Foreach == nil {
		return workflowFailure(state, "workflow_foreach_invalid", "Workflow foreach step is unavailable")
	}

	progress := pending.Foreach
	frames := make([]ProgramFrameIntent, 0)
	for workflowActiveChildren(pending.Children) < progress.Concurrency &&
		progress.Next < len(progress.Items) {
		index := progress.Next
		progress.Next++

		item := progress.Items[index].Clone()
		snapshot, err := i.childBlockProgram(
			frame,
			program,
			state,
			step.Foreach.Do,
			false,
			&item,
			workflowIntegerPointer(index),
			nil,
		)
		if err != nil {
			return workflowFailure(state, "workflow_foreach_prepare_failed", err.Error())
		}

		path := pending.Path + "/item/" + strconv.Itoa(index)
		pending.Children = append(pending.Children, WorkflowChild{
			Name:  strconv.Itoa(index),
			Path:  path,
			State: WorkflowChildStatePending,
		})
		frames = append(frames, ProgramFrameIntent{
			Path:     path,
			Program:  snapshot,
			Deadline: cloneTime(state.Deadline),
		})
	}

	if len(frames) != 0 {
		return workflowStructuredTransition(state, frames)
	}
	if progress.Next == len(progress.Items) &&
		workflowActiveChildren(pending.Children) == 0 {
		if err := i.finishForeach(&state, program, step); err != nil {
			if failure, ok := errors.AsType[workflowChildError](err); ok {
				return workflowFailureTransition(state, &failure.errorValue)
			}
			return workflowFailure(state, "workflow_foreach_failed", err.Error())
		}
		return i.drive(frame, Execution{}, program, state)
	}
	return workflowTransition(state, nil, nil, nil)
}

func (i *WorkflowInterpreter) launchRepeatChild(
	frame ProgramFrame,
	_ Execution,
	program WorkflowProgram,
	state WorkflowState,
) (ProgramTransition, error) {
	pending := state.Pending
	if pending == nil || pending.Repeat == nil {
		return workflowFailure(state, "workflow_repeat_invalid", "Workflow repeat state is missing")
	}
	step, err := workflowStepForPath(program.Root, pending.Path)
	if err != nil || step.Repeat == nil {
		return workflowFailure(state, "workflow_repeat_invalid", "Workflow repeat step is unavailable")
	}
	progress := pending.Repeat
	if !progress.LaunchNext {
		return workflowTransition(state, nil, nil, nil)
	}
	progress.LaunchNext = false

	allowed, exhausted, err := i.repeatAllowsNext(program, state, step.Repeat, progress)
	if err != nil {
		return workflowFailure(state, "workflow_repeat_condition_failed", err.Error())
	}
	if !allowed {
		if exhausted && step.Repeat.OnExhausted != "return" {
			return workflowFailure(
				state,
				"workflow_repeat_exhausted",
				"Workflow repeat exhausted its iteration limit",
			)
		}
		if err := i.finishRepeat(&state, program, step); err != nil {
			return workflowFailure(state, "workflow_repeat_failed", err.Error())
		}
		return i.drive(frame, Execution{}, program, state)
	}

	index := progress.Iterations
	carry := progress.State.Clone()
	snapshot, err := i.childBlockProgram(
		frame,
		program,
		state,
		step.Repeat.Do,
		step.Repeat.Do.State == nil,
		nil,
		workflowIntegerPointer(index),
		workflowCanonicalPointer(progress.Result),
	)
	if err != nil {
		return workflowFailure(state, "workflow_repeat_prepare_failed", err.Error())
	}
	childProgram, err := workflowProgramFromSnapshot(snapshot)
	if err != nil {
		return workflowFailure(state, "workflow_repeat_prepare_failed", err.Error())
	}
	childProgram.Scope.OuterState = carry
	payload, err := canonicalWorkflowValue(childProgram)
	if err != nil {
		return workflowFailure(state, "workflow_repeat_prepare_failed", err.Error())
	}
	snapshot.Payload = payload

	path := pending.Path + "/iteration/" + strconv.Itoa(index)
	pending.Children = append(pending.Children, WorkflowChild{
		Name:  strconv.Itoa(index),
		Path:  path,
		State: WorkflowChildStatePending,
	})
	return workflowStructuredTransition(
		state,
		[]ProgramFrameIntent{{
			Path:     path,
			Program:  snapshot,
			Deadline: cloneTime(state.Deadline),
		}},
	)
}

func (i *WorkflowInterpreter) repeatAllowsNext(
	program WorkflowProgram,
	state WorkflowState,
	spec *WorkflowRepeat,
	progress *WorkflowRepeatProgress,
) (allowed, exhausted bool, err error) {
	if spec == nil || progress == nil {
		return false, false, errors.New("Workflow repeat state is unavailable")
	}

	evaluationState := state
	evaluationState.State = progress.State.Clone()

	if spec.While != nil {
		allowed, err = i.condition(*spec.While, program, evaluationState)
		if err != nil || !allowed {
			return allowed, false, err
		}
	}
	if progress.Count != nil && progress.Iterations >= *progress.Count {
		return false, false, nil
	}
	if spec.MaxIterations > 0 && progress.Iterations >= spec.MaxIterations {
		return false, true, nil
	}
	return true, false, nil
}

func workflowChildByPath(
	children []WorkflowChild,
	path string,
) (int, bool) {
	for index := range children {
		if children[index].Path == path {
			return index, true
		}
	}
	return 0, false
}

func workflowFrameResult(
	frame ProgramFrame,
) (WorkflowBlockResult, *SanitizedError) {
	switch frame.Status {
	case ProgramFrameStateCompleted:
		result, err := DecodeCanonicalValue[WorkflowBlockResult](frame.Output)
		if err != nil {
			return WorkflowBlockResult{}, &SanitizedError{
				Code:     "workflow_child_output_invalid",
				Category: ErrorCategoryValidation,
				Message:  err.Error(),
			}
		}
		if err := result.Validate(); err != nil {
			return WorkflowBlockResult{}, &SanitizedError{
				Code:     "workflow_child_output_invalid",
				Category: ErrorCategoryValidation,
				Message:  err.Error(),
			}
		}
		return result, nil

	case ProgramFrameStateCancelled:
		return WorkflowBlockResult{}, &SanitizedError{
			Code:     "workflow_child_cancelled",
			Category: ErrorCategoryCancellation,
			Message:  "Workflow child block was cancelled",
		}

	default:
		if frame.Error != nil {
			value := *frame.Error
			value.Details = frame.Error.Details.Clone()
			value.Causes = append([]SanitizedError(nil), frame.Error.Causes...)
			return WorkflowBlockResult{}, &value
		}
		return WorkflowBlockResult{}, &SanitizedError{
			Code:     "workflow_child_failed",
			Category: ErrorCategoryExecution,
			Message:  "Workflow child block failed",
		}
	}
}

func (i *WorkflowInterpreter) settleStructuredChild(
	state *WorkflowState,
	program WorkflowProgram,
	execution Execution,
	parentFrameID ProgramFrameID,
	childID ProgramFrameID,
) error {
	if state == nil || state.Pending == nil {
		return nil
	}

	var childFrame *ProgramFrame
	for index := range execution.ProgramFrames {
		frame := &execution.ProgramFrames[index]
		if frame.ID == childID && frame.ParentFrameID == parentFrameID {
			childFrame = frame
			break
		}
	}
	if childFrame == nil || !childFrame.Status.Terminal() {
		return nil
	}

	childIndex, found := workflowChildByPath(
		state.Pending.Children,
		childFrame.Path,
	)
	if !found {
		return nil
	}
	child := &state.Pending.Children[childIndex]
	if child.State.terminal() {
		return nil
	}

	result, childErr := workflowFrameResult(*childFrame)
	if childErr != nil {
		child.Error = childErr
		switch childErr.Category {
		case ErrorCategoryCancellation:
			child.State = WorkflowChildStateCancelled
		default:
			child.State = WorkflowChildStateFailed
		}
	} else {
		child.Result = &result
		child.State = WorkflowChildStateSucceeded
	}

	switch state.Pending.Kind {
	case WorkflowPendingScope, WorkflowPendingChoose:
		return i.settleSingleStructuredChild(state, program)

	case WorkflowPendingParallel:
		return i.finishParallel(state, program, WorkflowStep{})

	case WorkflowPendingForeach:
		return i.settleForeachChild(state, program)

	case WorkflowPendingRepeat:
		return i.settleRepeatChild(state, program)

	default:
		return errors.New("Workflow pending control kind is invalid")
	}
}

func (i *WorkflowInterpreter) settleSingleStructuredChild(
	state *WorkflowState,
	program WorkflowProgram,
) error {
	if state == nil || state.Pending == nil || len(state.Pending.Children) != 1 {
		return errors.New("Workflow single child state is invalid")
	}
	pending := state.Pending
	child := pending.Children[0]
	if !child.State.terminal() {
		return nil
	}
	if child.State != WorkflowChildStateSucceeded || child.Result == nil {
		return workflowChildError{
			errorValue: workflowAggregateChildFailure(
				"workflow_child_failed",
				"Workflow child block did not complete",
				pending.Children,
			),
		}
	}

	result := *child.Result
	switch result.Exit {
	case WorkflowExitReturnWorkflow,
		WorkflowExitBreak,
		WorkflowExitContinue:
		state.Exit = result.Exit
		state.ExitValue = result.Value.Clone()
		state.Pending = nil
		return nil
	case WorkflowExitNormal, WorkflowExitReturnBlock:
	default:
		return errors.New("Workflow child returned an invalid control value")
	}

	step, err := workflowStepForPath(program.Root, pending.Path)
	if err != nil {
		return err
	}
	if pending.MergeState {
		state.State = result.State.Clone()
	}
	if err := i.validateStepOutput(program, *state, step, result.Value); err != nil {
		return err
	}
	if err := state.recordOutcome(WorkflowStepOutcome{
		ID:     step.ID,
		Status: WorkflowStepOutcomeSucceeded,
		Output: result.Value,
	}); err != nil {
		return err
	}
	state.Pending = nil
	state.Index++
	return nil
}

func workflowAggregateChildFailure(
	code string,
	message string,
	children []WorkflowChild,
) SanitizedError {
	causes := make([]SanitizedError, 0)
	for _, child := range children {
		if child.Error != nil {
			cause := *child.Error
			cause.Details = child.Error.Details.Clone()
			cause.Causes = append([]SanitizedError(nil), child.Error.Causes...)
			causes = append(causes, cause)
		}
	}
	return SanitizedError{
		Code:     code,
		Category: ErrorCategoryExecution,
		Message:  message,
		Causes:   causes,
	}
}

func workflowParallelSucceeded(child WorkflowChild) bool {
	return child.State == WorkflowChildStateSucceeded &&
		child.Result != nil &&
		(child.Result.Exit == WorkflowExitNormal ||
			child.Result.Exit == WorkflowExitReturnBlock)
}

func workflowParallelTerminalCount(children []WorkflowChild) int {
	count := 0
	for _, child := range children {
		if child.State.terminal() {
			count++
		}
	}
	return count
}

func workflowParallelSuccessNames(children []WorkflowChild) []string {
	output := make([]string, 0)
	for _, child := range children {
		if workflowParallelSucceeded(child) {
			output = append(output, child.Name)
		}
	}
	return output
}

func workflowParallelOutput(
	children []WorkflowChild,
	winners []string,
) (CanonicalValue, error) {
	outcomes := make([]WorkflowNamedOutcome, 0, len(children))
	for _, child := range children {
		outcome := WorkflowNamedOutcome{
			Name:  child.Name,
			State: child.State,
		}
		if child.Result != nil {
			outcome.Output = child.Result.Value.Clone()
		}
		if child.Error != nil {
			copyValue := *child.Error
			copyValue.Details = child.Error.Details.Clone()
			copyValue.Causes = append([]SanitizedError(nil), child.Error.Causes...)
			outcome.Error = &copyValue
		}
		outcomes = append(outcomes, outcome)
	}
	return canonicalWorkflowValue(struct {
		Branches []WorkflowNamedOutcome `json:"branches"`
		Winners  []string               `json:"winners,omitempty"`
	}{
		Branches: outcomes,
		Winners:  append([]string(nil), winners...),
	})
}

func (i *WorkflowInterpreter) finishParallel(
	state *WorkflowState,
	program WorkflowProgram,
	_ WorkflowStep,
) error {
	if state == nil || state.Pending == nil || state.Pending.Parallel == nil {
		return errors.New("Workflow parallel state is invalid")
	}
	pending := state.Pending
	progress := pending.Parallel
	children := pending.Children
	if len(children) == 0 {
		return errors.New("Workflow parallel has no branches")
	}

	mode := workflowCompletionMode(progress.Completion)
	successes := workflowParallelSuccessNames(children)
	allTerminal := workflowParallelTerminalCount(children) == len(children)

	switch mode {
	case "all":
		if !allTerminal {
			return nil
		}
		if len(successes) != len(children) {
			return workflowChildError{
				errorValue: workflowAggregateChildFailure(
					"workflow_parallel_failed",
					"one or more parallel branches failed",
					children,
				),
			}
		}
		progress.Winners = append([]string(nil), successes...)

	case "first-completed":
		if len(progress.Winners) == 0 {
			for _, child := range children {
				if child.State.terminal() {
					progress.Winners = []string{child.Name}
					break
				}
			}
		}
		if !allTerminal {
			return nil
		}
		if len(progress.Winners) == 0 {
			return errors.New("Workflow parallel has no completed branch")
		}
		for _, child := range children {
			if child.Name != progress.Winners[0] {
				continue
			}
			if !workflowParallelSucceeded(child) {
				return workflowChildError{
					errorValue: workflowAggregateChildFailure(
						"workflow_parallel_failed",
						"the first completed parallel branch failed",
						[]WorkflowChild{child},
					),
				}
			}
			break
		}

	case "first-success":
		if len(progress.Winners) == 0 && len(successes) != 0 {
			progress.Winners = []string{successes[0]}
		}
		if !allTerminal {
			return nil
		}
		if len(progress.Winners) == 0 {
			return workflowChildError{
				errorValue: workflowAggregateChildFailure(
					"workflow_parallel_failed",
					"no parallel branch succeeded",
					children,
				),
			}
		}

	case quorumStr:
		if progress.Completion.Quorum <= 0 {
			return errors.New("Workflow parallel quorum is invalid")
		}
		if len(progress.Winners) == 0 &&
			len(successes) >= progress.Completion.Quorum {
			progress.Winners = append(
				[]string(nil),
				successes[:progress.Completion.Quorum]...,
			)
		}
		if !allTerminal {
			return nil
		}
		if len(successes) < progress.Completion.Quorum {
			return workflowChildError{
				errorValue: workflowAggregateChildFailure(
					"workflow_parallel_quorum_unreachable",
					"parallel success quorum was not reached",
					children,
				),
			}
		}

	default:
		return errors.New("Workflow parallel completion mode is invalid")
	}

	value, err := workflowParallelOutput(children, progress.Winners)
	if err != nil {
		return err
	}
	step, err := workflowStepForPath(program.Root, pending.Path)
	if err != nil {
		return err
	}
	if err := i.validateStepOutput(program, *state, step, value); err != nil {
		return err
	}
	if err := state.recordOutcome(WorkflowStepOutcome{
		ID:     step.ID,
		Status: WorkflowStepOutcomeSucceeded,
		Output: value,
	}); err != nil {
		return err
	}
	state.Pending = nil
	state.Index++
	return nil
}

func workflowForeachSucceeded(outcome WorkflowItemOutcome) bool {
	return outcome.State == WorkflowChildStateSucceeded
}

func workflowForeachTerminalCount(outcomes []WorkflowItemOutcome) int {
	count := 0
	for _, outcome := range outcomes {
		if outcome.State.terminal() {
			count++
		}
	}
	return count
}

func workflowForeachSuccessIndexes(outcomes []WorkflowItemOutcome) []int {
	output := make([]int, 0)
	for _, outcome := range outcomes {
		if workflowForeachSucceeded(outcome) {
			output = append(output, outcome.Index)
		}
	}
	return output
}

func (i *WorkflowInterpreter) settleForeachChild(
	state *WorkflowState,
	program WorkflowProgram,
) error {
	if state == nil || state.Pending == nil || state.Pending.Foreach == nil {
		return errors.New("Workflow foreach state is invalid")
	}
	pending := state.Pending
	progress := pending.Foreach

	for _, child := range pending.Children {
		if !child.State.terminal() {
			continue
		}
		index, err := strconv.Atoi(child.Name)
		if err != nil || index < 0 || index >= len(progress.Outcomes) {
			return errors.New("Workflow foreach child index is invalid")
		}
		outcome := &progress.Outcomes[index]
		if outcome.State.terminal() {
			continue
		}
		outcome.State = child.State
		if child.Result != nil {
			outcome.Output = child.Result.Value.Clone()
		}
		if child.Error != nil {
			copyValue := *child.Error
			copyValue.Details = child.Error.Details.Clone()
			copyValue.Causes = append([]SanitizedError(nil), child.Error.Causes...)
			outcome.Error = &copyValue
		}
	}

	if workflowForeachTerminalCount(progress.Outcomes) == len(progress.Outcomes) &&
		progress.Next == len(progress.Items) {
		step, err := workflowStepForPath(program.Root, pending.Path)
		if err != nil {
			return err
		}
		return i.finishForeach(state, program, step)
	}
	return nil
}

func (i *WorkflowInterpreter) finishForeach(
	state *WorkflowState,
	program WorkflowProgram,
	step WorkflowStep,
) error {
	if state == nil || state.Pending == nil || state.Pending.Foreach == nil {
		return errors.New("Workflow foreach state is invalid")
	}
	progress := state.Pending.Foreach
	successes := workflowForeachSuccessIndexes(progress.Outcomes)
	mode := workflowCompletionMode(progress.Completion)

	switch mode {
	case "all":
		if len(successes) != len(progress.Outcomes) {
			return workflowChildError{
				errorValue: workflowAggregateForeachFailure(progress.Outcomes),
			}
		}
		progress.Winners = append([]int(nil), successes...)

	case "first-completed":
		if len(progress.Outcomes) != 0 {
			progress.Winners = []int{0}
		}

	case "first-success":
		if len(successes) == 0 {
			return workflowChildError{
				errorValue: workflowAggregateForeachFailure(progress.Outcomes),
			}
		}
		progress.Winners = []int{successes[0]}

	case quorumStr:
		if progress.Completion.Quorum <= 0 ||
			len(successes) < progress.Completion.Quorum {
			return workflowChildError{
				errorValue: workflowAggregateForeachFailure(progress.Outcomes),
			}
		}
		progress.Winners = append(
			[]int(nil),
			successes[:progress.Completion.Quorum]...,
		)

	default:
		return errors.New("Workflow foreach completion mode is invalid")
	}

	value, err := canonicalWorkflowValue(struct {
		Items   []WorkflowItemOutcome `json:"items"`
		Winners []int                 `json:"winningIndexes,omitempty"`
	}{
		Items:   append([]WorkflowItemOutcome(nil), progress.Outcomes...),
		Winners: append([]int(nil), progress.Winners...),
	})
	if err != nil {
		return err
	}
	if err := i.validateStepOutput(program, *state, step, value); err != nil {
		return err
	}
	if err := state.recordOutcome(WorkflowStepOutcome{
		ID:     step.ID,
		Status: WorkflowStepOutcomeSucceeded,
		Output: value,
	}); err != nil {
		return err
	}
	state.Pending = nil
	state.Index++
	return nil
}

func workflowAggregateForeachFailure(
	outcomes []WorkflowItemOutcome,
) SanitizedError {
	causes := make([]SanitizedError, 0)
	for _, outcome := range outcomes {
		if outcome.Error == nil {
			continue
		}
		cause := *outcome.Error
		cause.Details = outcome.Error.Details.Clone()
		cause.Causes = append([]SanitizedError(nil), outcome.Error.Causes...)
		causes = append(causes, cause)
	}
	return SanitizedError{
		Code:     "workflow_foreach_failed",
		Category: ErrorCategoryExecution,
		Message:  "one or more foreach items failed",
		Causes:   causes,
	}
}

func (i *WorkflowInterpreter) settleRepeatChild(
	state *WorkflowState,
	program WorkflowProgram,
) error {
	if state == nil || state.Pending == nil || state.Pending.Repeat == nil {
		return errors.New("Workflow repeat state is invalid")
	}
	pending := state.Pending
	progress := pending.Repeat

	var child *WorkflowChild
	for index := range pending.Children {
		if pending.Children[index].State.terminal() {
			child = &pending.Children[index]
		}
	}
	if child == nil {
		return nil
	}
	if child.State != WorkflowChildStateSucceeded || child.Result == nil {
		return workflowChildError{
			errorValue: workflowAggregateChildFailure(
				"workflow_repeat_failed",
				"Workflow repeat iteration failed",
				[]WorkflowChild{*child},
			),
		}
	}

	result := *child.Result
	switch result.Exit {
	case WorkflowExitReturnWorkflow:
		state.Exit = WorkflowExitReturnWorkflow
		state.ExitValue = result.Value.Clone()
		state.Pending = nil
		return nil

	case WorkflowExitBreak:
		step, err := workflowStepForPath(program.Root, pending.Path)
		if err != nil {
			return err
		}
		progress.Result = result.Value.Clone()
		progress.Iterations++
		return i.finishRepeat(state, program, step)

	case WorkflowExitNormal,
		WorkflowExitReturnBlock,
		WorkflowExitContinue:
	default:
		return errors.New("Workflow repeat child returned an invalid control value")
	}

	if !progress.LocalState {
		progress.State = result.State.Clone()
	}
	progress.Result = result.Value.Clone()
	progress.Iterations++

	step, err := workflowStepForPath(program.Root, pending.Path)
	if err != nil || step.Repeat == nil {
		return errors.New("Workflow repeat step is unavailable")
	}

	if step.Repeat.Until != nil {
		evaluationState := *state
		evaluationState.State = progress.State.Clone()
		resultValue := progress.Result.Clone()
		evaluationProgram := program
		evaluationProgram.Scope.Result = &resultValue
		stop, err := i.condition(
			*step.Repeat.Until,
			evaluationProgram,
			evaluationState,
		)
		if err != nil {
			return err
		}
		if stop {
			return i.finishRepeat(state, program, step)
		}
	}

	progress.LaunchNext = true
	return nil
}

func (i *WorkflowInterpreter) finishRepeat(
	state *WorkflowState,
	program WorkflowProgram,
	step WorkflowStep,
) error {
	if state == nil || state.Pending == nil || state.Pending.Repeat == nil {
		return errors.New("Workflow repeat state is invalid")
	}
	progress := state.Pending.Repeat
	value, err := canonicalWorkflowValue(struct {
		State      CanonicalValue `json:"state"`
		Result     CanonicalValue `json:"result"`
		Iterations int            `json:"iterations"`
		Exhausted  bool           `json:"exhausted"`
	}{
		State:      progress.State.Clone(),
		Result:     progress.Result.Clone(),
		Iterations: progress.Iterations,
		Exhausted: step.Repeat.MaxIterations > 0 &&
			progress.Iterations >= step.Repeat.MaxIterations,
	})
	if err != nil {
		return err
	}

	if !progress.LocalState {
		state.State = progress.State.Clone()
	}
	if err := i.validateStepOutput(program, *state, step, value); err != nil {
		return err
	}
	if err := state.recordOutcome(WorkflowStepOutcome{
		ID:     step.ID,
		Status: WorkflowStepOutcomeSucceeded,
		Output: value,
	}); err != nil {
		return err
	}
	state.Pending = nil
	state.Index++
	return nil
}

func (b WorkflowBlock) Validate() error {
	if err := b.Output.Validate(); err != nil {
		return err
	}
	if b.State != nil {
		if !b.State.Schema.Valid() || len(b.State.Schema) == 0 {
			return errors.New("Workflow state schema is invalid")
		}
		if err := b.State.Initial.Validate(); err != nil {
			return err
		}
	}

	ids := make(map[string]struct{}, len(b.Steps))
	for index, step := range b.Steps {
		if strings.TrimSpace(step.ID) == "" {
			return fmt.Errorf("Workflow step %d has no ID", index)
		}
		if _, duplicate := ids[step.ID]; duplicate {
			return fmt.Errorf("Workflow step %q is duplicated", step.ID)
		}
		ids[step.ID] = struct{}{}
		if err := step.Validate(); err != nil {
			return fmt.Errorf("Workflow step %q: %w", step.ID, err)
		}
	}
	return nil
}

func (c WorkflowChoose) Validate() error {
	for index, candidate := range c.Cases {
		if err := candidate.When.Validate(); err != nil {
			return fmt.Errorf("Workflow choose case %d: %w", index, err)
		}
		if err := candidate.Do.Validate(); err != nil {
			return fmt.Errorf("Workflow choose case %d: %w", index, err)
		}
	}
	if c.Default != nil {
		return c.Default.Validate()
	}
	return nil
}

func (c WorkflowCompletion) Validate() error {
	switch workflowCompletionMode(c) {
	case allStr, firstCompletedStr, firstSuccessStr:
		if c.Quorum != 0 {
			return errors.New("Workflow completion quorum is valid only for quorum mode")
		}
	case quorumStr:
		if c.Quorum <= 0 {
			return errors.New("Workflow completion quorum must be positive")
		}
	default:
		return errors.New("Workflow completion mode is invalid")
	}
	return nil
}

func (p WorkflowParallel) Validate() error {
	if err := p.Completion.Validate(); err != nil {
		return err
	}
	switch p.Errors {
	case "", "fail-fast", "collect":
	default:
		return errors.New("Workflow parallel error policy is invalid")
	}
	switch workflowRemainingMode(p.Remaining) {
	case "cancel", "wait":
	default:
		return errors.New("Workflow parallel remaining policy is invalid")
	}
	if len(p.Branches) == 0 {
		return errors.New("Workflow parallel has no branches")
	}

	seen := make(map[string]struct{}, len(p.Branches))
	for _, branch := range p.Branches {
		if strings.TrimSpace(branch.Name) == "" {
			return errors.New("Workflow parallel branch has no name")
		}
		if _, duplicate := seen[branch.Name]; duplicate {
			return errors.New("Workflow parallel branch is duplicated")
		}
		seen[branch.Name] = struct{}{}
		if branch.When != nil {
			if err := branch.When.Validate(); err != nil {
				return err
			}
		}
		if err := branch.Do.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (f WorkflowForeach) Validate() error {
	if err := f.Items.Validate(); err != nil {
		return err
	}
	if !f.ItemSchema.Valid() || len(f.ItemSchema) == 0 {
		return errors.New("Workflow foreach item schema is invalid")
	}
	if f.Concurrency < 0 || f.MaxItems < 0 {
		return errors.New("Workflow foreach limits are invalid")
	}
	if err := f.Completion.Validate(); err != nil {
		return err
	}
	switch f.Errors {
	case "", "fail-fast", "collect":
	default:
		return errors.New("Workflow foreach error policy is invalid")
	}
	switch workflowRemainingMode(f.Remaining) {
	case "cancel", "wait":
	default:
		return errors.New("Workflow foreach remaining policy is invalid")
	}
	return f.Do.Validate()
}

func (r WorkflowRepeat) Validate() error {
	controls := 0
	for _, present := range []bool{
		r.Count != nil,
		r.While != nil,
		r.Until != nil,
		r.Lifetime != "",
	} {
		if present {
			controls++
		}
	}
	if controls != 1 {
		return errors.New("Workflow repeat requires exactly one control")
	}
	if r.Count != nil {
		if err := r.Count.Validate(); err != nil {
			return err
		}
	}
	if r.While != nil {
		if err := r.While.Validate(); err != nil {
			return err
		}
	}
	if r.Until != nil {
		if err := r.Until.Validate(); err != nil {
			return err
		}
	}
	if r.Lifetime != "" && r.Lifetime != "until-cancelled" {
		return errors.New("Workflow repeat lifetime is invalid")
	}
	if r.State != nil {
		if !r.State.Schema.Valid() || len(r.State.Schema) == 0 {
			return errors.New("Workflow repeat state schema is invalid")
		}
		if err := r.State.Initial.Validate(); err != nil {
			return err
		}
	}
	if r.MaxIterations < 0 {
		return errors.New("Workflow repeat maxIterations is invalid")
	}
	switch r.OnExhausted {
	case "", "fail", "return":
	default:
		return errors.New("Workflow repeat onExhausted is invalid")
	}
	return r.Do.Validate()
}
