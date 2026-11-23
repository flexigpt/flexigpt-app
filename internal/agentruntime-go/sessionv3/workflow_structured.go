package sessionv3

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type WorkflowCompletionMode string

const (
	WorkflowCompletionModeAll            WorkflowCompletionMode = "all"
	WorkflowCompletionModeFirstCompleted WorkflowCompletionMode = "first-completed"
	WorkflowCompletionModeFirstSuccess   WorkflowCompletionMode = "first-success"
	WorkflowCompletionModeQuorum         WorkflowCompletionMode = "quorum"
)

type WorkflowCollectionErrorPolicy string

const (
	WorkflowCollectionErrorFailFast WorkflowCollectionErrorPolicy = "fail-fast"
	WorkflowCollectionErrorCollect  WorkflowCollectionErrorPolicy = "collect"
)

type WorkflowRemainingPolicy string

const (
	WorkflowRemainingCancel WorkflowRemainingPolicy = "cancel"
	WorkflowRemainingWait   WorkflowRemainingPolicy = "wait"
)

type WorkflowRepeatLifetime string

const (
	WorkflowRepeatLifetimeUntilCancelled WorkflowRepeatLifetime = "until-cancelled"
)

type WorkflowRepeatExhaustedPolicy string

const (
	WorkflowRepeatOnExhaustedFail   WorkflowRepeatExhaustedPolicy = "fail"
	WorkflowRepeatOnExhaustedReturn WorkflowRepeatExhaustedPolicy = "return"
)

// WorkflowBlockRole identifies the structural position of a lowered
// WorkflowBlock in the runtime-neutral Workflow tree.
type WorkflowBlockRole string

const (
	WorkflowBlockRoleRoot              WorkflowBlockRole = "root"
	WorkflowBlockRoleScope             WorkflowBlockRole = "scope"
	WorkflowBlockRoleChooseCase        WorkflowBlockRole = "chooseCase"
	WorkflowBlockRoleChooseDefault     WorkflowBlockRole = "chooseDefault"
	WorkflowBlockRoleParallelBranch    WorkflowBlockRole = "parallelBranch"
	WorkflowBlockRoleForeach           WorkflowBlockRole = "foreach"
	WorkflowBlockRoleRepeat            WorkflowBlockRole = "repeat"
	WorkflowBlockRoleMachineEntry      WorkflowBlockRole = "machineEntry"
	WorkflowBlockRoleMachineExit       WorkflowBlockRole = "machineExit"
	WorkflowBlockRoleMachineTransition WorkflowBlockRole = "machineTransition"
	WorkflowBlockRoleSaga              WorkflowBlockRole = "saga"
	WorkflowBlockRoleEffectReconcile   WorkflowBlockRole = "effectReconcile"
	WorkflowBlockRoleEffectCompensate  WorkflowBlockRole = "effectCompensate"
	WorkflowBlockRoleCatch             WorkflowBlockRole = "catch"
	WorkflowBlockRoleFinally           WorkflowBlockRole = "finally"
)

// ErrStopWorkflowWalk terminates WalkWorkflowBlock early without error.
var ErrStopWorkflowWalk = errors.New("stop workflow tree walk")

type WorkflowBlockNode struct {
	Block       *WorkflowBlock
	ParentBlock *WorkflowBlock
	ParentStep  *WorkflowStep
	Role        WorkflowBlockRole
	Name        string
	Index       int
}

type WorkflowStepNode struct {
	Block *WorkflowBlock
	Step  *WorkflowStep
	Index int
}

// WorkflowTreeVisitor receives every lowered WorkflowBlock and WorkflowStep
// in deterministic pre-order. Runtime and setup share this one traversal
// contract without moving source parsing or Artifact planning into Runtime.
type WorkflowTreeVisitor struct {
	OnBlock func(node WorkflowBlockNode) error
	OnStep  func(node WorkflowStepNode) error
}

// WorkflowScopeContext is the durable parent visibility envelope for an
// interpreter-created child block frame. It contains semantic values only.
type WorkflowScopeContext struct {
	Child                   bool                        `json:"child,omitempty"`
	MergeState              bool                        `json:"mergeState,omitempty"`
	OuterState              jsonutil.CanonicalValue     `json:"outerState,omitempty"`
	StateOwner              *WorkflowStateOwner         `json:"stateOwner,omitempty"`
	TargetSets              []WorkflowTargetSetState    `json:"targetSets,omitempty"`
	OuterOutcomes           []WorkflowStepOutcome       `json:"outerOutcomes,omitempty"`
	CausalEffectWorkItemIDs []WorkItemID                `json:"causalEffectWorkItemIDs,omitempty"`
	OuterSubscriptions      []WorkflowSubscriptionState `json:"outerSubscriptions,omitempty"`
	Context                 *jsonutil.CanonicalValue    `json:"context,omitempty"`
	Item                    *jsonutil.CanonicalValue    `json:"item,omitempty"`
	Index                   *int                        `json:"index,omitempty"`
	Result                  *jsonutil.CanonicalValue    `json:"result,omitempty"`
	Event                   *jsonutil.CanonicalValue    `json:"event,omitempty"`
	Error                   *jsonutil.CanonicalValue    `json:"error,omitempty"`
	Outcome                 *jsonutil.CanonicalValue    `json:"outcome,omitempty"`
	Operation               *jsonutil.CanonicalValue    `json:"operation,omitempty"`
	Completed               *jsonutil.CanonicalValue    `json:"completed,omitempty"`
	Ports                   []PortInput                 `json:"ports,omitempty"`
	ChannelAliases          map[string]string           `json:"channelAliases,omitempty"`
	Saga                    *WorkflowSagaScope          `json:"saga,omitempty"`
	Cleanup                 bool                        `json:"cleanup,omitempty"`
}

func (s WorkflowScopeContext) Validate() error {
	if !s.OuterState.Valid() {
		return errors.New("workflow scope outer state is invalid")
	}
	if s.StateOwner != nil {
		if err := s.StateOwner.Validate(); err != nil {
			return err
		}
	}
	if err := validateWorkflowTargetSetStates(s.TargetSets); err != nil {
		return err
	}
	if err := validateWorkflowCausalEffectWorkItemIDs(s.CausalEffectWorkItemIDs); err != nil {
		return err
	}
	if s.Context != nil && !s.Context.Valid() {
		return errors.New("workflow scope context is invalid")
	}
	if s.Item != nil && !s.Item.Valid() {
		return errors.New("workflow scope item is invalid")
	}
	if s.Result != nil && !s.Result.Valid() {
		return errors.New("workflow scope result is invalid")
	}
	if s.Event != nil && !s.Event.Valid() {
		return errors.New("workflow scope event is invalid")
	}
	if s.Error != nil && !s.Error.Valid() {
		return errors.New("workflow scope error is invalid")
	}
	if s.Outcome != nil && !s.Outcome.Valid() {
		return errors.New("workflow scope outcome is invalid")
	}
	if s.Operation != nil && !s.Operation.Valid() {
		return errors.New("workflow scope operation is invalid")
	}
	if s.Completed != nil && !s.Completed.Valid() {
		return errors.New("workflow scope completed facts are invalid")
	}
	if err := validateUniquePortInputs(s.Ports); err != nil {
		return err
	}
	if err := validateWorkflowChannelAliases(s.ChannelAliases); err != nil {
		return err
	}
	if s.Saga != nil {
		if err := s.Saga.Validate(); err != nil {
			return err
		}
	}
	if s.Index != nil && *s.Index < 0 {
		return errors.New("workflow scope index is invalid")
	}

	seen := make(map[string]struct{}, len(s.OuterOutcomes))
	for _, outcome := range s.OuterOutcomes {
		if err := outcome.Validate(); err != nil {
			return fmt.Errorf("workflow scope outcome: %w", err)
		}
		if _, duplicate := seen[outcome.ID]; duplicate {
			return errors.New("workflow scope outcome is duplicated")
		}
		seen[outcome.ID] = struct{}{}
	}
	for _, subscription := range s.OuterSubscriptions {
		if err := subscription.Validate(); err != nil {
			return err
		}
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
	Exit                    WorkflowExitKind        `json:"exit"`
	Value                   jsonutil.CanonicalValue `json:"value"`
	State                   jsonutil.CanonicalValue `json:"state"`
	CausalEffectWorkItemIDs []WorkItemID            `json:"causalEffectWorkItemIDs,omitempty"`
}

func (r WorkflowBlockResult) Validate() error {
	switch r.Exit {
	case WorkflowExitNormal,
		WorkflowExitReturnBlock,
		WorkflowExitReturnWorkflow,
		WorkflowExitBreak,
		WorkflowExitContinue:
	default:
		return errors.New("workflow block result exit is invalid")
	}
	if !r.Value.Valid() || !r.State.Valid() {
		return errors.New("workflow block result contains invalid json")
	}
	if err := validateWorkflowCausalEffectWorkItemIDs(r.CausalEffectWorkItemIDs); err != nil {
		return err
	}
	return nil
}

func (r WorkflowReturn) Validate() error {
	switch r.To {
	case "",
		WorkflowReturnDestinationBlock,
		WorkflowReturnDestinationWorkflow:
	default:
		return errors.New("workflow return destination is invalid")
	}

	if err := r.Value.Validate(); err != nil {
		return err
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
	Mode   WorkflowCompletionMode `json:"mode,omitempty"`
	Quorum int                    `json:"quorum,omitempty"`
}

type WorkflowBranch struct {
	Name string         `json:"name"`
	When *WorkflowValue `json:"when,omitempty"`
	Do   WorkflowBlock  `json:"do"`
}

// WorkflowParentStateMerge is an explicit deterministic merge from isolated
// parallel or foreach child results into the nearest parent state owner.
type WorkflowParentStateMerge struct {
	Values []WorkflowValuePair `json:"values,omitempty"`
	Remove []string            `json:"remove,omitempty"`
}

// WorkflowParentStateMergeProgress persists the StateUpdate boundary after
// all selected child outcomes settle and before the parent receives success.
type WorkflowParentStateMergeProgress struct {
	WorkItemID WorkItemID              `json:"workItemID"`
	Owner      WorkflowStateOwner      `json:"owner"`
	Update     StateUpdateWork         `json:"update"`
	Output     jsonutil.CanonicalValue `json:"output"`
	Applied    bool                    `json:"applied,omitempty"`
}

type WorkflowParallel struct {
	Completion WorkflowCompletion            `json:"completion"`
	Errors     WorkflowCollectionErrorPolicy `json:"errors,omitempty"`
	Remaining  WorkflowRemainingPolicy       `json:"remaining,omitempty"`
	Merge      *WorkflowParentStateMerge     `json:"merge,omitempty"`
	Branches   []WorkflowBranch              `json:"branches"`
}

type WorkflowForeach struct {
	Items       WorkflowValue                 `json:"items"`
	ItemSchema  jsonutil.CanonicalValue       `json:"itemSchema"`
	Concurrency int                           `json:"concurrency,omitempty"`
	MaxItems    *int                          `json:"maxItems,omitempty"`
	Completion  WorkflowCompletion            `json:"completion"`
	Errors      WorkflowCollectionErrorPolicy `json:"errors,omitempty"`
	Merge       *WorkflowParentStateMerge     `json:"merge,omitempty"`
	Remaining   WorkflowRemainingPolicy       `json:"remaining,omitempty"`
	Do          WorkflowBlock                 `json:"do"`
}

type WorkflowRepeat struct {
	Count         *WorkflowValue                `json:"count,omitempty"`
	While         *WorkflowValue                `json:"while,omitempty"`
	Until         *WorkflowValue                `json:"until,omitempty"`
	Lifetime      WorkflowRepeatLifetime        `json:"lifetime,omitempty"`
	State         *WorkflowStateSpec            `json:"state,omitempty"`
	MaxIterations *int                          `json:"maxIterations,omitempty"`
	OnExhausted   WorkflowRepeatExhaustedPolicy `json:"onExhausted,omitempty"`
	Timeout       WorkflowDuration              `json:"timeout,omitempty"`
	Do            WorkflowBlock                 `json:"do"`
}

type WorkflowPendingKind string

const (
	WorkflowPendingScope          WorkflowPendingKind = "scope"
	WorkflowPendingChoose         WorkflowPendingKind = "choose"
	WorkflowPendingParallel       WorkflowPendingKind = "parallel"
	WorkflowPendingForeach        WorkflowPendingKind = "foreach"
	WorkflowPendingRepeat         WorkflowPendingKind = "repeat"
	WorkflowPendingMachine        WorkflowPendingKind = "machine"
	WorkflowPendingReconciliation WorkflowPendingKind = "reconciliation"
	WorkflowPendingSaga           WorkflowPendingKind = "saga"
)

type WorkflowChildState string

const (
	WorkflowChildStatePending   WorkflowChildState = "pending"
	WorkflowChildStateSucceeded WorkflowChildState = "succeeded"
	WorkflowChildStateFailed    WorkflowChildState = "failed"
	WorkflowChildStateCancelled WorkflowChildState = "cancelled"
	WorkflowChildStateUncertain WorkflowChildState = "uncertain"
	WorkflowChildStateSkipped   WorkflowChildState = "skipped"
)

func (s WorkflowChildState) terminal() bool {
	switch s {
	case WorkflowChildStateSucceeded,
		WorkflowChildStateFailed,
		WorkflowChildStateCancelled,
		WorkflowChildStateUncertain,
		WorkflowChildStateSkipped:
		return true
	default:
		return false
	}
}

type WorkflowChild struct {
	Name                    string               `json:"name"`
	Path                    string               `json:"path,omitempty"`
	State                   WorkflowChildState   `json:"state"`
	Result                  *WorkflowBlockResult `json:"result,omitempty"`
	Error                   *SanitizedError      `json:"error,omitempty"`
	CausalEffectWorkItemIDs []WorkItemID         `json:"causalEffectWorkItemIDs,omitempty"`
	CompletedOrder          uint64               `json:"completedOrder,omitempty"`
}

type WorkflowNamedOutcome struct {
	Name               string                  `json:"name"`
	State              WorkflowChildState      `json:"state"`
	Output             jsonutil.CanonicalValue `json:"output,omitempty"`
	Error              *SanitizedError         `json:"error,omitempty"`
	CompletionSequence uint64                  `json:"completionSequence,omitempty"`
}

type WorkflowParallelProgress struct {
	Completion         WorkflowCompletion            `json:"completion"`
	Errors             WorkflowCollectionErrorPolicy `json:"errors,omitempty"`
	Remaining          WorkflowRemainingPolicy       `json:"remaining,omitempty"`
	Winners            []string                      `json:"winners,omitempty"`
	Cancelling         bool                          `json:"cancelling,omitempty"`
	CompletionSequence uint64                        `json:"completionSequence,omitempty"`
}

type WorkflowItemOutcome struct {
	Index          int                     `json:"index"`
	State          WorkflowChildState      `json:"state"`
	Output         jsonutil.CanonicalValue `json:"output,omitempty"`
	Error          *SanitizedError         `json:"error,omitempty"`
	CompletedOrder uint64                  `json:"completedOrder,omitempty"`
}

type WorkflowForeachProgress struct {
	Items              []jsonutil.CanonicalValue     `json:"items"`
	Outcomes           []WorkflowItemOutcome         `json:"outcomes"`
	Next               int                           `json:"next"`
	Concurrency        int                           `json:"concurrency"`
	Completion         WorkflowCompletion            `json:"completion"`
	Errors             WorkflowCollectionErrorPolicy `json:"errors,omitempty"`
	Remaining          WorkflowRemainingPolicy       `json:"remaining,omitempty"`
	Merge              *WorkflowParentStateMerge     `json:"merge,omitempty"`
	Winners            []int                         `json:"winners,omitempty"`
	Cancelling         bool                          `json:"cancelling,omitempty"`
	CompletionSequence uint64                        `json:"completionSequence,omitempty"`
}

type WorkflowRepeatProgress struct {
	State      jsonutil.CanonicalValue `json:"state"`
	Result     jsonutil.CanonicalValue `json:"result,omitempty"`
	Iterations int                     `json:"iterations"`
	Count      *int                    `json:"count,omitempty"`
	LocalState bool                    `json:"localState"`
	LaunchNext bool                    `json:"launchNext"`
	Exhausted  bool                    `json:"exhausted,omitempty"`
}

type WorkflowPending struct {
	Kind             WorkflowPendingKind               `json:"kind"`
	StepID           string                            `json:"stepID"`
	Path             string                            `json:"path"`
	ParentStateMerge *WorkflowParentStateMerge         `json:"parentStateMerge,omitempty"`
	StateOwner       *WorkflowStateOwner               `json:"stateOwner,omitempty"`
	StateMerge       *WorkflowParentStateMergeProgress `json:"stateMerge,omitempty"`
	MergeState       bool                              `json:"mergeState,omitempty"`
	Claims           []ResourceClaim                   `json:"claims,omitempty"`
	ClaimsBound      bool                              `json:"claimsBound,omitempty"`
	Children         []WorkflowChild                   `json:"children,omitempty"`
	Parallel         *WorkflowParallelProgress         `json:"parallel,omitempty"`
	Foreach          *WorkflowForeachProgress          `json:"foreach,omitempty"`
	Repeat           *WorkflowRepeatProgress           `json:"repeat,omitempty"`
	Machine          *WorkflowMachineProgress          `json:"machine,omitempty"`
	Reconciliation   *WorkflowReconciliationProgress   `json:"reconciliation,omitempty"`
	Saga             *WorkflowSagaProgress             `json:"saga,omitempty"`
}

func (p WorkflowPending) Validate() error {
	if strings.TrimSpace(p.StepID) == "" || strings.TrimSpace(p.Path) == "" {
		return errors.New("workflow pending control has no step identity")
	}
	switch p.Kind {
	case WorkflowPendingScope, WorkflowPendingChoose:
		if len(p.Children) != 1 {
			return errors.New("workflow pending block has an invalid child count")
		}
	case WorkflowPendingParallel:
		if p.Parallel == nil {
			return errors.New("workflow pending parallel state is missing")
		}
	case WorkflowPendingForeach:
		if p.Foreach == nil {
			return errors.New("workflow pending foreach state is missing")
		}
	case WorkflowPendingRepeat:
		if p.Repeat == nil {
			return errors.New("workflow pending repeat state is missing")
		}
	case WorkflowPendingMachine:
		if p.Machine == nil {
			return errors.New("workflow pending machine state is missing")
		}
		if err := p.Machine.Validate(); err != nil {
			return err
		}
	case WorkflowPendingReconciliation:
		if p.Reconciliation == nil {
			return errors.New("workflow pending reconciliation state is missing")
		}
		if err := p.Reconciliation.Validate(); err != nil {
			return err
		}
	case WorkflowPendingSaga:
		if p.Saga == nil || len(p.Children) != 1 {
			return errors.New("workflow pending saga state is invalid")
		}
		if err := p.Saga.Validate(); err != nil {
			return err
		}
	default:
		return errors.New("workflow pending control kind is invalid")
	}

	if p.ParentStateMerge != nil {
		switch p.Kind {
		case WorkflowPendingParallel, WorkflowPendingForeach:
		default:
			return errors.New("workflow parent state merge has an invalid control owner")
		}
		if err := p.ParentStateMerge.Validate(); err != nil {
			return err
		}
		if p.StateOwner == nil {
			return errors.New("workflow parent state merge has no state owner")
		}
		if err := p.StateOwner.Validate(); err != nil {
			return err
		}
	}
	if p.StateMerge != nil {
		if p.ParentStateMerge == nil || p.StateOwner == nil {
			return errors.New("workflow state merge progress has no declaration owner")
		}
		if err := p.StateMerge.Validate(); err != nil {
			return err
		}
	}
	if p.StateOwner != nil && p.ParentStateMerge == nil {
		return errors.New("workflow pending state owner has no parent merge")
	}

	for _, claim := range p.Claims {
		if err := claim.Validate(); err != nil {
			return err
		}
	}
	seen := make(map[string]struct{}, len(p.Children))
	for _, child := range p.Children {
		if strings.TrimSpace(child.Name) == "" {
			return errors.New("workflow pending child has no name")
		}
		if child.State == "" {
			return errors.New("workflow pending child has no state")
		}
		if _, duplicate := seen[child.Name]; duplicate {
			return errors.New("workflow pending child is duplicated")
		}
		seen[child.Name] = struct{}{}
		if child.Result != nil {
			if err := child.Result.Validate(); err != nil {
				return err
			}
		}
		if child.Error != nil && !child.Error.Details.Valid() {
			return errors.New("workflow pending child error details are invalid")
		}
		if err := validateWorkflowCausalEffectWorkItemIDs(child.CausalEffectWorkItemIDs); err != nil {
			return err
		}
	}
	return nil
}

func (m WorkflowParentStateMerge) Validate() error {
	if len(m.Values) == 0 && len(m.Remove) == 0 {
		return errors.New("workflow parent state merge has no update")
	}
	seen := make(map[string]struct{}, len(m.Values)+len(m.Remove))
	for _, assignment := range m.Values {
		if strings.TrimSpace(assignment.Key) == "" {
			return errors.New("workflow parent state merge assignment key is empty")
		}
		if _, duplicate := seen[assignment.Key]; duplicate {
			return errors.New("workflow parent state merge assignment is duplicated")
		}
		if err := assignment.Value.Validate(); err != nil {
			return err
		}
		seen[assignment.Key] = struct{}{}
	}
	for _, path := range m.Remove {
		if strings.TrimSpace(path) == "" {
			return errors.New("workflow parent state merge removal path is empty")
		}
		if _, duplicate := seen[path]; duplicate {
			return errors.New("workflow parent state merge assignment and removal overlap")
		}
		seen[path] = struct{}{}
	}
	return nil
}

func (p WorkflowParentStateMergeProgress) Validate() error {
	if err := validateID(
		"workflow state merge work item ID",
		string(p.WorkItemID),
	); err != nil {
		return err
	}
	if err := p.Owner.Validate(); err != nil {
		return err
	}
	if p.Update.StateOwnerID != p.Owner.ID {
		return errors.New("workflow state merge owner differs from its update")
	}
	if err := p.Update.Validate(); err != nil {
		return err
	}
	if !p.Output.Valid() {
		return errors.New("workflow state merge output is invalid")
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
	value jsonutil.CanonicalValue,
	exit WorkflowExitKind,
) (ProgramTransition, error) {
	if !value.Valid() {
		return ProgramTransition{}, errors.New("workflow block completion value is invalid")
	}

	if exit == "" {
		exit = WorkflowExitNormal
	}

	if program.Scope.Child {
		result := WorkflowBlockResult{
			Exit:                    exit,
			Value:                   value.Clone(),
			State:                   state.State.Clone(),
			CausalEffectWorkItemIDs: workflowCausalEffectWorkItemIDs(state),
		}
		if err := result.Validate(); err != nil {
			return ProgramTransition{}, err
		}
		status := ProgramFrameStateCompleted
		return workflowTransition(
			state,
			&status,
			jsonutil.MustCanonicalValue(result),
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
	for index := range frames {
		if frames[index].Evaluation == nil &&
			state.PendingDeclarationPath != "" {
			frames[index].Evaluation = &EvaluationBoundary{
				DeclarationPath: state.PendingDeclarationPath,
				LogicalTime:     state.LogicalTime.UTC(),
				Messages:        state.Messages.Clone(),
			}
		}
	}
	forkClaims := []ResourceClaim(nil)
	if state.Pending != nil &&
		!state.Pending.ClaimsBound &&
		len(state.Pending.Claims) != 0 &&
		len(frames) != 0 {
		forkClaims = append(
			forkClaims,
			state.Pending.Claims...,
		)
		state.Pending.ClaimsBound = true
	}
	next, err := encodeWorkflowProgramState(state)
	if err != nil {
		return ProgramTransition{}, err
	}
	return ProgramTransition{
		State:      next,
		Frames:     frames,
		ForkClaims: forkClaims,
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
		values = append(values, cloneWorkflowStepOutcome(outcome))
	}
	for _, outcome := range state.Outcomes {
		values = append(values, cloneWorkflowStepOutcome(outcome))
	}
	return values
}

func workflowCanonicalPointer(value jsonutil.CanonicalValue) *jsonutil.CanonicalValue {
	if len(value) == 0 {
		return nil
	}
	copyValue := value.Clone()
	return &copyValue
}

func cloneWorkflowCanonicalPointer(
	value *jsonutil.CanonicalValue,
) *jsonutil.CanonicalValue {
	if value == nil {
		return nil
	}
	return workflowCanonicalPointer(*value)
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
	item *jsonutil.CanonicalValue,
	index *int,
	result *jsonutil.CanonicalValue,
) (ProgramSnapshot, error) {
	return i.childBlockProgramWithScope(
		frame,
		program,
		state,
		block,
		mergeState,
		WorkflowScopeContext{
			Item:   cloneWorkflowCanonicalPointer(item),
			Index:  cloneWorkflowIndex(index),
			Result: cloneWorkflowCanonicalPointer(result),
		},
	)
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
	item *jsonutil.CanonicalValue,
	index *int,
	result *jsonutil.CanonicalValue,
) (ProgramTransition, error) {
	claims, err := i.workflowResourceClaims(program, state, step)
	if err != nil {
		return workflowFailure(state, "workflow_resource_claim_invalid", err.Error())
	}

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
		Claims:     claims,
		Children: []WorkflowChild{{
			Name:  childName,
			Path:  childPath,
			State: WorkflowChildStatePending,
		}},
	}
	return workflowStructuredTransition(
		state,
		[]ProgramFrameIntent{{
			Path:            childPath,
			DeclarationPath: firstText(block.DeclarationPath, state.PendingDeclarationPath),
			Program:         snapshot,
			Deadline:        cloneTime(state.Deadline),
			Budget:          cloneBudget(step.Limits),
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

func workflowCompletionMode(value WorkflowCompletion) WorkflowCompletionMode {
	if value.Mode == "" {
		return WorkflowCompletionModeAll
	}
	return value.Mode
}

func workflowRemainingMode(value WorkflowRemainingPolicy) WorkflowRemainingPolicy {
	if value == "" {
		return WorkflowRemainingCancel
	}
	return value
}

func (i *WorkflowInterpreter) startParallel(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	path string,
) (ProgramTransition, error) {
	if step.Parallel == nil {
		return workflowFailure(state, "workflow_parallel_invalid", "Workflow parallel is missing")
	}

	claims, err := i.workflowResourceClaims(program, state, step)
	if err != nil {
		return workflowFailure(state, "workflow_resource_claim_invalid", err.Error())
	}

	pending := &WorkflowPending{
		Kind:   WorkflowPendingParallel,
		StepID: step.ID,
		Path:   path,
		Claims: claims,
		Parallel: &WorkflowParallelProgress{
			Completion: step.Parallel.Completion,
			Errors:     step.Parallel.Errors,
			Remaining:  step.Parallel.Remaining,
		},
	}
	if step.Parallel.Merge != nil {
		owner, ownerErr := workflowStateOwnerForFrame(frame, program)
		if ownerErr != nil {
			return workflowFailure(
				state,
				"workflow_state_merge_invalid",
				ownerErr.Error(),
			)
		}
		if owner == nil {
			return workflowFailure(
				state,
				"workflow_state_unavailable",
				"workflow parent state merge has no state owner",
			)
		}
		pending.ParentStateMerge = cloneWorkflowParentStateMerge(step.Parallel.Merge)
		pending.StateOwner = owner
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
				Path:            child.Path,
				DeclarationPath: firstText(branch.Do.DeclarationPath, state.PendingDeclarationPath),
				Program:         snapshot,
				Deadline:        cloneTime(state.Deadline),
				Budget:          cloneBudget(step.Limits),
			})
		}
		pending.Children = append(pending.Children, child)
	}

	state.Pending = pending
	if len(frames) == 0 {
		if err := i.finishParallel(frame, &state, program, step); err != nil {
			if failure, ok := errors.AsType[workflowChildError](err); ok {
				return workflowFailureTransition(state, &failure.errorValue)
			}
			return workflowFailure(state, "workflow_parallel_failed", err.Error())
		}
		return i.drive(frame, execution, program, state)
	}
	return workflowStructuredTransition(state, frames)
}

func (i *WorkflowInterpreter) startForeach(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	path string,
) (ProgramTransition, error) {
	if step.Foreach == nil {
		return workflowFailure(state, "workflow_foreach_invalid", "Workflow foreach is missing")
	}

	claims, err := i.workflowResourceClaims(program, state, step)
	if err != nil {
		return workflowFailure(state, "workflow_resource_claim_invalid", err.Error())
	}

	value, err := i.evaluate(step.Foreach.Items, program, state)
	if err != nil {
		return workflowFailure(state, "workflow_foreach_items_invalid", err.Error())
	}
	items, err := jsonutil.DecodeCanonicalValue[[]jsonutil.CanonicalValue](value)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_foreach_items_invalid",
			"Workflow foreach items must be an array",
		)
	}
	if step.Foreach.MaxItems != nil && len(items) > *step.Foreach.MaxItems {
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

	pending := &WorkflowPending{
		Kind:   WorkflowPendingForeach,
		StepID: step.ID,
		Path:   path,
		Claims: claims,
		Foreach: &WorkflowForeachProgress{
			Items:       append([]jsonutil.CanonicalValue(nil), items...),
			Outcomes:    outcomes,
			Concurrency: concurrency,
			Completion:  step.Foreach.Completion,
			Errors:      step.Foreach.Errors,
			Remaining:   step.Foreach.Remaining,
		},
	}
	if step.Foreach.Merge != nil {
		owner, ownerErr := workflowStateOwnerForFrame(frame, program)
		if ownerErr != nil {
			return workflowFailure(state, "workflow_state_merge_invalid", ownerErr.Error())
		}
		if owner == nil {
			return workflowFailure(
				state,
				"workflow_state_unavailable",
				"workflow parent state merge has no state owner",
			)
		}
		pending.ParentStateMerge = cloneWorkflowParentStateMerge(step.Foreach.Merge)
		pending.StateOwner = owner
	}
	state.Pending = pending
	return i.drivePending(frame, execution, program, state)
}

func (i *WorkflowInterpreter) startRepeat(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	path string,
) (ProgramTransition, error) {
	if step.Repeat == nil {
		return workflowFailure(state, "workflow_repeat_invalid", "Workflow repeat is missing")
	}

	claims, err := i.workflowResourceClaims(program, state, step)
	if err != nil {
		return workflowFailure(state, "workflow_resource_claim_invalid", err.Error())
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
		count, err := jsonutil.DecodeCanonicalValue[int](value)
		if err != nil || count < 0 {
			return workflowFailure(
				state,
				"workflow_repeat_count_invalid",
				"Workflow repeat count must be a non-negative integer",
			)
		}
		progress.Count = workflowIntegerPointer(count)
	}
	if step.Repeat.Timeout != "" {
		deadline, timeoutErr := step.Repeat.Timeout.Deadline(
			state.LogicalTime,
		)
		if timeoutErr != nil {
			return workflowFailure(
				state,
				"workflow_repeat_timeout_invalid",
				timeoutErr.Error(),
			)
		}
		state.Deadline = earlierDeadline(state.Deadline, &deadline)
	}

	state.Pending = &WorkflowPending{
		Kind:   WorkflowPendingRepeat,
		StepID: step.ID,
		Path:   path,
		Claims: claims,
		Repeat: progress,
	}
	return i.drivePending(frame, execution, program, state)
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
	if transition, handled, err := i.driveParentStateMerge(
		frame,
		execution,
		program,
		&state,
	); handled || err != nil {
		return transition, err
	}

	switch state.Pending.Kind {
	case WorkflowPendingForeach:
		return i.launchForeachChildren(frame, execution, program, state)
	case WorkflowPendingRepeat:
		return i.launchRepeatChild(frame, execution, program, state)
	case WorkflowPendingMachine:
		return i.driveMachine(frame, execution, program, state)
	case WorkflowPendingReconciliation:
		return i.driveReconciliation(frame, execution, program, state)
	case WorkflowPendingSaga:
		return i.driveSaga(frame, execution, program, state)
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
	execution Execution,
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
			Path:            path,
			DeclarationPath: firstText(step.Foreach.Do.DeclarationPath, state.PendingDeclarationPath),
			Program:         snapshot,
			Deadline:        cloneTime(state.Deadline),
			Budget:          cloneBudget(step.Limits),
		})
	}

	if len(frames) != 0 {
		return workflowStructuredTransition(state, frames)
	}
	if progress.Next == len(progress.Items) &&
		workflowActiveChildren(pending.Children) == 0 {
		if err := i.finishForeach(frame, &state, program, step); err != nil {
			if failure, ok := errors.AsType[workflowChildError](err); ok {
				return workflowFailureTransition(state, &failure.errorValue)
			}
			return workflowFailure(state, "workflow_foreach_failed", err.Error())
		}
		if state.Pending != nil {
			return i.drivePending(frame, execution, program, state)
		}
		return i.drive(frame, execution, program, state)
	}
	return workflowTransition(state, nil, nil, nil)
}

func (i *WorkflowInterpreter) launchRepeatChild(
	frame ProgramFrame,
	execution Execution,
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
	if state.Deadline != nil &&
		!state.LogicalTime.Before(state.Deadline.UTC()) {
		return workflowFailure(
			state,
			"workflow_repeat_timeout_elapsed",
			"workflow repeat timeout elapsed",
		)
	}
	progress.LaunchNext = false

	allowed, exhausted, err := i.repeatAllowsNext(program, state, step.Repeat, progress)
	if err != nil {
		return workflowFailure(state, "workflow_repeat_condition_failed", err.Error())
	}
	if !allowed {
		if exhausted {
			progress.Exhausted = true
			if step.Repeat.OnExhausted != WorkflowRepeatOnExhaustedReturn {
				return workflowFailure(
					state,
					"workflow_repeat_exhausted",
					"Workflow repeat exhausted its iteration limit",
				)
			}
		}
		if err := i.finishRepeat(&state, program, step); err != nil {
			return workflowFailure(state, "workflow_repeat_failed", err.Error())
		}
		return i.drive(frame, execution, program, state)
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
	if step.Repeat.State != nil {
		childProgram.Scope.StateOwner = workflowSyntheticStateOwner(
			frame.ID,
			pending.Path,
			"repeat",
			step.Repeat.State.Schema,
		)
	}
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
			Path:            path,
			DeclarationPath: firstText(step.Repeat.Do.DeclarationPath, state.PendingDeclarationPath),
			Program:         snapshot,
			Deadline:        cloneTime(state.Deadline),
			Budget:          cloneBudget(step.Limits),
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
	evaluationProgram := program
	evaluationProgram.Scope.Index = workflowIntegerPointer(
		progress.Iterations,
	)

	if spec.While != nil {
		allowed, err = i.condition(*spec.While, evaluationProgram, evaluationState)
		if err != nil || !allowed {
			return allowed, false, err
		}
	}
	if progress.Count != nil && progress.Iterations >= *progress.Count {
		return false, false, nil
	}
	if spec.MaxIterations != nil &&
		progress.Iterations >= *spec.MaxIterations {
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
		result, err := jsonutil.DecodeCanonicalValue[WorkflowBlockResult](frame.Output)
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
		if uncertainty := workflowFrameUncertainty(frame); uncertainty != nil {
			return WorkflowBlockResult{}, uncertainty
		}
		return WorkflowBlockResult{}, &SanitizedError{
			Code:     workflowChildCancelled,
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

// workflowFrameUncertainty preserves an already-recorded uncertain operation
// when structured cancellation settles a sibling branch. Cancellation stops
// remaining work, but it does not prove that an already-dispatched external
// effect was not applied.
func workflowFrameUncertainty(
	frame ProgramFrame,
) *SanitizedError {
	if frame.Program.InterpreterID != WorkflowInterpreterID {
		return nil
	}
	_, state, err := decodeWorkflowProgram(frame)
	if err != nil {
		return nil
	}
	for _, outcome := range slices.Backward(state.Outcomes) {
		if outcome.Status != WorkflowStepOutcomeUncertain {
			continue
		}
		if outcome.Error != nil {
			return cloneWorkflowSanitizedError(outcome.Error)
		}
		return &SanitizedError{
			Code:     "workflow_child_uncertain",
			Category: ErrorCategoryUncertainty,
			Effect:   outcome.Effect,
			Message:  "Workflow child contains an uncertain external outcome",
		}
	}
	return nil
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
	child.CausalEffectWorkItemIDs = workflowFrameCausalEffectWorkItemIDs(
		*childFrame,
	)

	result, childErr := workflowFrameResult(*childFrame)
	if childErr != nil {
		child.Error = childErr
		switch childErr.Category {
		case ErrorCategoryCancellation:
			child.State = WorkflowChildStateCancelled
		case ErrorCategoryUncertainty:
			child.State = WorkflowChildStateUncertain
		default:
			child.State = WorkflowChildStateFailed
		}
	} else {
		child.Result = &result
		child.State = WorkflowChildStateSucceeded
	}
	switch state.Pending.Kind {
	case WorkflowPendingParallel:
		state.Pending.Parallel.CompletionSequence++
		child.CompletedOrder = state.Pending.Parallel.CompletionSequence
	case WorkflowPendingForeach:
		state.Pending.Foreach.CompletionSequence++
		child.CompletedOrder = state.Pending.Foreach.CompletionSequence
	default:
	}

	if state.Pending.Kind == WorkflowPendingParallel &&
		state.Pending.Parallel != nil {
		progress := state.Pending.Parallel
		mode := workflowCompletionMode(progress.Completion)
		switch workflowCompletionMode(progress.Completion) {
		case WorkflowCompletionModeFirstCompleted:
			if len(progress.Winners) == 0 {
				progress.Winners = []string{child.Name}
			}
		case WorkflowCompletionModeFirstSuccess, WorkflowCompletionModeQuorum:
			if workflowParallelSucceeded(*child) &&
				(mode != WorkflowCompletionModeFirstSuccess || len(progress.Winners) == 0) &&
				!slices.Contains(progress.Winners, child.Name) {
				progress.Winners = append(progress.Winners, child.Name)
			}
		default:
		}
	}

	switch state.Pending.Kind {
	case WorkflowPendingScope, WorkflowPendingChoose:
		return i.settleSingleStructuredChild(state, program)
	case WorkflowPendingParallel:
		parent := mutableFrame(&execution, parentFrameID)
		if parent == nil {
			return errors.New("workflow parallel parent frame is unavailable")
		}
		return i.finishParallel(*parent, state, program, WorkflowStep{})

	case WorkflowPendingForeach:
		parent := mutableFrame(&execution, parentFrameID)
		if parent == nil {
			return errors.New("workflow foreach parent frame is unavailable")
		}
		return i.settleForeachChild(*parent, state, program)

	case WorkflowPendingRepeat:
		return i.settleRepeatChild(state, program)

	case WorkflowPendingMachine:
		return i.settleMachineChild(state, program)

	case WorkflowPendingSaga:
		return nil

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
	setWorkflowCausalEffectWorkItemIDs(state, mergeWorkflowCausalEffectWorkItemIDs(
		workflowCausalEffectWorkItemIDs(*state),
		child.CausalEffectWorkItemIDs,
	))
	if pending.MergeState {
		state.State = result.State.Clone()
	}
	switch result.Exit {
	case WorkflowExitReturnWorkflow,
		WorkflowExitBreak,
		WorkflowExitContinue:
		state.Exit = result.Exit
		state.ExitValue = result.Value.Clone()
		state.Pending = nil
		state.clearPendingStep()
		return nil
	case WorkflowExitNormal, WorkflowExitReturnBlock:
	default:
		return errors.New("Workflow child returned an invalid control value")
	}

	step, err := workflowStepForPath(program.Root, pending.Path)
	if err != nil {
		return err
	}
	if err := i.validateStepOutput(program, *state, step, result.Value); err != nil {
		return err
	}
	if err := state.recordOutcome(WorkflowStepOutcome{
		ID:              step.ID,
		Status:          WorkflowStepOutcomeSucceeded,
		Output:          result.Value,
		DeclarationPath: state.PendingDeclarationPath,
	}); err != nil {
		return err
	}
	workflowReleasePendingClaims(state, pending)
	state.Pending = nil
	state.Index++
	state.clearPendingStep()
	return nil
}

func workflowAggregateChildFailure(
	code string,
	message string,
	children []WorkflowChild,
) SanitizedError {
	category := ErrorCategoryExecution
	causes := make([]SanitizedError, 0)
	for _, child := range children {
		if child.Error == nil {
			continue
		}
		cause := *child.Error
		cause.Details = child.Error.Details.Clone()
		cause.Causes = append([]SanitizedError(nil), child.Error.Causes...)
		causes = append(causes, cause)
		if cause.Category == ErrorCategoryUncertainty {
			category = ErrorCategoryUncertainty
		}
	}
	return SanitizedError{
		Code:     code,
		Category: category,
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

func workflowChildrenCausalEffectWorkItemIDs(
	children []WorkflowChild,
) []WorkItemID {
	output := make([]WorkItemID, 0)
	for _, child := range children {
		output = mergeWorkflowCausalEffectWorkItemIDs(
			output,
			child.CausalEffectWorkItemIDs,
		)
	}
	return output
}

func workflowParallelSelectedCount(
	children []WorkflowChild,
) int {
	count := 0
	for _, child := range children {
		if child.State != WorkflowChildStateSkipped {
			count++
		}
	}
	return count
}

func workflowParallelHasSelectedFailure(
	children []WorkflowChild,
) bool {
	for _, child := range children {
		if child.State == WorkflowChildStateSkipped ||
			!child.State.terminal() ||
			workflowParallelSucceeded(child) {
			continue
		}
		return true
	}
	return false
}

func workflowParallelAllTerminal(children []WorkflowChild) bool {
	return workflowParallelTerminalCount(children) == len(children)
}

func workflowRequestParallelCancellation(
	state *WorkflowState,
	children []WorkflowChild,
) {
	if state == nil || state.Pending == nil || state.Pending.Parallel == nil {
		return
	}
	if state.Pending.Parallel.Cancelling {
		return
	}
	state.Pending.Parallel.Cancelling = true
	for _, child := range children {
		if child.State == WorkflowChildStatePending && child.Path != "" {
			state.CancelChildPaths = append(state.CancelChildPaths, child.Path)
		}
	}
}

func workflowParallelOutput(
	children []WorkflowChild,
	winners []string,
) (jsonutil.CanonicalValue, error) {
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

func workflowParallelAggregateFailure(
	children []WorkflowChild,
	winners []string,
	code string,
	message string,
) workflowChildError {
	failure := workflowAggregateChildFailure(code, message, children)
	if output, err := workflowParallelOutput(children, winners); err == nil {
		failure.Details = output
	}
	return workflowChildError{errorValue: failure}
}

func (i *WorkflowInterpreter) finishParallel(
	frame ProgramFrame,
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
	selected := workflowParallelSelectedCount(children)
	allTerminal := workflowParallelAllTerminal(children)

	switch mode {
	case WorkflowCompletionModeAll:
		hasFailure := workflowParallelHasSelectedFailure(children)
		if hasFailure && progress.Errors == WorkflowCollectionErrorFailFast {
			workflowRequestParallelCancellation(state, children)
		}
		if !allTerminal {
			return nil
		}
		if len(successes) != selected {
			return workflowParallelAggregateFailure(
				children, progress.Winners, "workflow_parallel_failed",
				"one or more parallel branches failed",
			)
		}
		progress.Winners = append([]string(nil), successes...)

	case WorkflowCompletionModeFirstCompleted:
		if len(progress.Winners) == 0 {
			if allTerminal {
				return workflowParallelAggregateFailure(
					children, nil, "workflow_parallel_no_selected_branch",
					"parallel selected no branch",
				)
			}
			return nil
		}
		if workflowRemainingMode(progress.Remaining) == WorkflowRemainingCancel {
			workflowRequestParallelCancellation(state, children)
		}
		if !allTerminal {
			return nil
		}
		for _, child := range children {
			if child.Name != progress.Winners[0] {
				continue
			}
			if !workflowParallelSucceeded(child) {
				return workflowParallelAggregateFailure(
					children, progress.Winners, "workflow_parallel_failed",
					"the first completed parallel branch failed",
				)
			}
			break
		}

	case WorkflowCompletionModeFirstSuccess:
		if len(progress.Winners) != 0 &&
			workflowRemainingMode(progress.Remaining) == WorkflowRemainingCancel {
			workflowRequestParallelCancellation(state, children)
		}
		if !allTerminal {
			return nil
		}
		if len(progress.Winners) == 0 {
			return workflowParallelAggregateFailure(
				children, nil, "workflow_parallel_failed",
				"no parallel branch succeeded",
			)
		}

	case WorkflowCompletionModeQuorum:
		if progress.Completion.Quorum <= 0 {
			return errors.New("workflow parallel quorum is invalid")
		}
		if len(progress.Winners) >= progress.Completion.Quorum &&
			workflowRemainingMode(progress.Remaining) == WorkflowRemainingCancel {
			workflowRequestParallelCancellation(state, children)
		}
		if !allTerminal {
			return nil
		}
		if len(progress.Winners) < progress.Completion.Quorum {
			return workflowParallelAggregateFailure(
				children, progress.Winners,
				"workflow_parallel_quorum_unreachable",
				"parallel success quorum was not reached",
			)
		}
		progress.Winners = append(
			[]string(nil),
			progress.Winners[:progress.Completion.Quorum]...,
		)

	default:
		return errors.New("workflow parallel completion mode is invalid")
	}

	value, err := workflowParallelOutput(children, progress.Winners)
	if err != nil {
		return err
	}
	setWorkflowCausalEffectWorkItemIDs(state, mergeWorkflowCausalEffectWorkItemIDs(
		workflowCausalEffectWorkItemIDs(*state),
		workflowChildrenCausalEffectWorkItemIDs(children),
	))
	waiting, err := i.ensureParentStateMerge(frame, program, state, value)
	if err != nil {
		return err
	}
	if waiting {
		return nil
	}
	step, err := workflowStepForPath(program.Root, pending.Path)
	if err != nil {
		return err
	}
	if err := i.validateStepOutput(program, *state, step, value); err != nil {
		return err
	}
	if err := state.recordOutcome(WorkflowStepOutcome{
		ID:              step.ID,
		Status:          WorkflowStepOutcomeSucceeded,
		Output:          value,
		DeclarationPath: state.PendingDeclarationPath,
	}); err != nil {
		return err
	}
	workflowReleasePendingClaims(state, pending)
	state.Pending = nil
	state.Index++
	state.clearPendingStep()
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

func workflowForeachAllTerminal(
	progress *WorkflowForeachProgress,
) bool {
	if progress == nil || progress.Next != len(progress.Items) {
		return false
	}
	return workflowForeachTerminalCount(progress.Outcomes) == len(progress.Outcomes)
}

func workflowRequestForeachCancellation(
	state *WorkflowState,
) {
	if state == nil || state.Pending == nil || state.Pending.Foreach == nil {
		return
	}
	pending := state.Pending
	progress := pending.Foreach
	if progress.Cancelling {
		return
	}
	progress.Cancelling = true

	for index := progress.Next; index < len(progress.Outcomes); index++ {
		progress.Outcomes[index].State = WorkflowChildStateSkipped
	}
	progress.Next = len(progress.Items)

	for _, child := range pending.Children {
		if child.State == WorkflowChildStatePending && child.Path != "" {
			state.CancelChildPaths = append(state.CancelChildPaths, child.Path)
		}
	}
}

func workflowForeachHasFailure(
	outcomes []WorkflowItemOutcome,
) bool {
	for _, outcome := range outcomes {
		switch outcome.State {
		case WorkflowChildStateFailed, WorkflowChildStateCancelled:
			return true
		case WorkflowChildStateUncertain:
			return true
		default:
		}
	}
	return false
}

func compareWorkflowItemCompletionOrder(
	left WorkflowItemOutcome,
	right WorkflowItemOutcome,
) int {
	switch {
	case left.CompletedOrder == 0 && right.CompletedOrder != 0:
		return 1
	case left.CompletedOrder != 0 && right.CompletedOrder == 0:
		return -1
	case left.CompletedOrder < right.CompletedOrder:
		return -1
	case left.CompletedOrder > right.CompletedOrder:
		return 1
	case left.Index < right.Index:
		return -1
	case left.Index > right.Index:
		return 1
	default:
		return 0
	}
}

func (i *WorkflowInterpreter) settleForeachChild(
	frame ProgramFrame,
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
		outcome.CompletedOrder = child.CompletedOrder
	}

	switch workflowCompletionMode(progress.Completion) {
	case WorkflowCompletionModeFirstCompleted:
		if len(progress.Winners) == 0 {
			outcomes := append([]WorkflowItemOutcome(nil), progress.Outcomes...)
			slices.SortFunc(outcomes, compareWorkflowItemCompletionOrder)
			for _, outcome := range outcomes {
				if outcome.State.terminal() {
					progress.Winners = []int{outcome.Index}
					break
				}
			}
		}
	case WorkflowCompletionModeFirstSuccess, WorkflowCompletionModeQuorum:
		outcomes := append([]WorkflowItemOutcome(nil), progress.Outcomes...)
		slices.SortFunc(outcomes, compareWorkflowItemCompletionOrder)
		for _, outcome := range outcomes {
			if outcome.State != WorkflowChildStateSucceeded ||
				slices.Contains(progress.Winners, outcome.Index) {
				continue
			}
			progress.Winners = append(progress.Winners, outcome.Index)
			if workflowCompletionMode(progress.Completion) == WorkflowCompletionModeFirstSuccess {
				break
			}
		}
	default:
	}

	step, err := workflowStepForPath(program.Root, pending.Path)
	if err != nil {
		return err
	}
	return i.finishForeach(frame, state, program, step)
}

func workflowForeachOutput(
	progress *WorkflowForeachProgress,
) (jsonutil.CanonicalValue, error) {
	if progress == nil {
		return nil, errors.New("Workflow foreach state is unavailable")
	}
	return canonicalWorkflowValue(struct {
		Items   []WorkflowItemOutcome `json:"items"`
		Winners []int                 `json:"winningIndexes,omitempty"`
	}{
		Items:   append([]WorkflowItemOutcome(nil), progress.Outcomes...),
		Winners: append([]int(nil), progress.Winners...),
	})
}

func workflowForeachAggregateFailure(
	progress *WorkflowForeachProgress,
) workflowChildError {
	failure := workflowAggregateForeachFailure(progress.Outcomes)
	if output, err := workflowForeachOutput(progress); err == nil {
		failure.Details = output
	}
	return workflowChildError{errorValue: failure}
}

func (i *WorkflowInterpreter) finishForeach(
	frame ProgramFrame,
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
	if len(progress.Outcomes) == 0 && mode != WorkflowCompletionModeAll {
		code := "workflow_foreach_no_selected_item"
		message := "foreach completion mode requires at least one item"
		if mode == WorkflowCompletionModeQuorum {
			code = "workflow_foreach_quorum_unreachable"
			message = "foreach success quorum cannot be reached by an empty collection"
		}
		failure := workflowAggregateForeachFailure(progress.Outcomes)
		failure.Code = code
		failure.Message = message
		if output, err := workflowForeachOutput(progress); err == nil {
			failure.Details = output
		}
		return workflowChildError{errorValue: failure}
	}

	switch mode {
	case WorkflowCompletionModeAll:
		if workflowForeachHasFailure(progress.Outcomes) &&
			progress.Errors == WorkflowCollectionErrorFailFast {
			workflowRequestForeachCancellation(state)
		}
		if !workflowForeachAllTerminal(progress) {
			return nil
		}
		if len(successes) != len(progress.Outcomes) {
			return workflowForeachAggregateFailure(progress)
		}
		progress.Winners = append([]int(nil), successes...)

	case WorkflowCompletionModeFirstCompleted:
		if len(progress.Winners) == 0 {
			return nil
		}
		if workflowRemainingMode(progress.Remaining) == WorkflowRemainingCancel {
			workflowRequestForeachCancellation(state)
		}
		if !workflowForeachAllTerminal(progress) {
			return nil
		}
		winner := progress.Winners[0]
		if winner < 0 || winner >= len(progress.Outcomes) ||
			progress.Outcomes[winner].State != WorkflowChildStateSucceeded {
			return workflowForeachAggregateFailure(progress)
		}

	case WorkflowCompletionModeFirstSuccess:
		if len(progress.Winners) != 0 &&
			workflowRemainingMode(progress.Remaining) == WorkflowRemainingCancel {
			workflowRequestForeachCancellation(state)
		}
		if !workflowForeachAllTerminal(progress) {
			return nil
		}
		if len(progress.Winners) == 0 {
			return workflowForeachAggregateFailure(progress)
		}
		progress.Winners = []int{progress.Winners[0]}

	case WorkflowCompletionModeQuorum:
		if progress.Completion.Quorum <= 0 {
			return errors.New("workflow foreach quorum is invalid")
		}
		if len(progress.Winners) >= progress.Completion.Quorum &&
			workflowRemainingMode(progress.Remaining) == WorkflowRemainingCancel {
			workflowRequestForeachCancellation(state)
		}
		if !workflowForeachAllTerminal(progress) {
			return nil
		}
		if len(progress.Winners) < progress.Completion.Quorum {
			return workflowForeachAggregateFailure(progress)
		}
		progress.Winners = append(
			[]int(nil),
			progress.Winners[:progress.Completion.Quorum]...,
		)

	default:
		return errors.New("workflow foreach completion mode is invalid")
	}

	value, err := workflowForeachOutput(progress)
	if err != nil {
		return err
	}
	setWorkflowCausalEffectWorkItemIDs(state, mergeWorkflowCausalEffectWorkItemIDs(
		workflowCausalEffectWorkItemIDs(*state),
		workflowChildrenCausalEffectWorkItemIDs(state.Pending.Children),
	))
	waiting, err := i.ensureParentStateMerge(frame, program, state, value)
	if err != nil {
		return err
	}
	if waiting {
		return nil
	}
	if err := i.validateStepOutput(program, *state, step, value); err != nil {
		return err
	}
	if err := state.recordOutcome(WorkflowStepOutcome{
		ID:              step.ID,
		Status:          WorkflowStepOutcomeSucceeded,
		Output:          value,
		DeclarationPath: state.PendingDeclarationPath,
	}); err != nil {
		return err
	}
	workflowReleasePendingClaims(state, state.Pending)
	state.Pending = nil
	state.Index++
	state.clearPendingStep()
	return nil
}

func workflowAggregateForeachFailure(
	outcomes []WorkflowItemOutcome,
) SanitizedError {
	category := ErrorCategoryExecution
	causes := make([]SanitizedError, 0)
	for _, outcome := range outcomes {
		if outcome.Error == nil {
			continue
		}
		cause := *outcome.Error
		cause.Details = outcome.Error.Details.Clone()
		cause.Causes = append([]SanitizedError(nil), outcome.Error.Causes...)
		causes = append(causes, cause)
		if cause.Category == ErrorCategoryUncertainty {
			category = ErrorCategoryUncertainty
		}
	}
	return SanitizedError{
		Code:     "workflow_foreach_failed",
		Category: category,
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
	step, err := workflowStepForPath(program.Root, pending.Path)
	if err != nil || step.Repeat == nil {
		return errors.New("Workflow repeat step is unavailable")
	}
	switch result.Exit {
	case WorkflowExitReturnWorkflow:
		state.Exit = WorkflowExitReturnWorkflow
		state.ExitValue = result.Value.Clone()
		state.Pending = nil
		state.clearPendingStep()
		return nil

	case WorkflowExitBreak:

		if step.Repeat.Do.State == nil {
			progress.State = result.State.Clone()
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

	setWorkflowCausalEffectWorkItemIDs(state, mergeWorkflowCausalEffectWorkItemIDs(
		workflowCausalEffectWorkItemIDs(*state),
		child.CausalEffectWorkItemIDs,
	))
	progress.Result = result.Value.Clone()
	progress.Iterations++

	if step.Repeat.Do.State == nil {
		// The body inherits the repeat carried state when it has no local
		// state declaration. Its completed state is the next iteration's
		// durable carried state, even when Repeat itself owns a local state
		// scope relative to its parent.
		progress.State = result.State.Clone()
	}

	if step.Repeat.Until != nil {
		evaluationState := *state
		evaluationState.State = progress.State.Clone()
		evaluationProgram := program
		resultValue := progress.Result.Clone()
		evaluationProgram.Scope.Result = &resultValue
		evaluationProgram.Scope.Index = workflowIntegerPointer(progress.Iterations - 1)
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
		State      jsonutil.CanonicalValue `json:"state"`
		Result     jsonutil.CanonicalValue `json:"result"`
		Iterations int                     `json:"iterations"`
		Exhausted  bool                    `json:"exhausted"`
	}{
		State:      progress.State.Clone(),
		Result:     progress.Result.Clone(),
		Iterations: progress.Iterations,
		Exhausted:  progress.Exhausted,
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
		ID:              step.ID,
		Status:          WorkflowStepOutcomeSucceeded,
		Output:          value,
		DeclarationPath: state.PendingDeclarationPath,
	}); err != nil {
		return err
	}
	workflowReleasePendingClaims(state, state.Pending)
	state.Pending = nil
	state.Index++
	state.clearPendingStep()
	return nil
}

func (b WorkflowBlock) Validate() error {
	if err := b.Output.Validate(); err != nil {
		return err
	}
	if err := validateWorkflowSubscriptions(b.Listen); err != nil {
		return err
	}
	if b.State != nil {
		if !b.State.Schema.Valid() || len(b.State.Schema) == 0 {
			return errors.New("workflow state schema is invalid")
		}
		if err := b.State.Initial.Validate(); err != nil {
			return err
		}
	}

	ids := make(map[string]struct{}, len(b.Steps))
	for index, step := range b.Steps {
		if strings.TrimSpace(step.ID) == "" {
			return fmt.Errorf("workflow step %d has no id", index)
		}
		if _, duplicate := ids[step.ID]; duplicate {
			return fmt.Errorf("workflow step %q is duplicated", step.ID)
		}
		ids[step.ID] = struct{}{}
		if err := step.Validate(); err != nil {
			return fmt.Errorf("workflow step %q: %w", step.ID, err)
		}
	}
	if err := validateWorkflowStepSagaDependencies(b.Steps); err != nil {
		return err
	}
	for index := range b.Catch {
		if err := b.Catch[index].Validate(); err != nil {
			return fmt.Errorf(
				"workflow catch handler %d: %w",
				index,
				err,
			)
		}
	}
	if b.Finally != nil {
		if err := b.Finally.Validate(); err != nil {
			return fmt.Errorf("workflow finally block: %w", err)
		}
	}
	return nil
}

func validateWorkflowStepSagaDependencies(
	steps []WorkflowStep,
) error {
	for _, step := range steps {
		if step.Saga == nil {
			continue
		}
		if err := step.Saga.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (i *WorkflowInterpreter) childBlockProgramWithScope(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	block WorkflowBlock,
	mergeState bool,
	scope WorkflowScopeContext,
) (ProgramSnapshot, error) {
	inherited, err := workflowInheritedChildScope(
		frame,
		program,
		state,
		mergeState,
		scope,
	)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	child := program
	child.Root = block
	child.Scope = inherited
	child.Input = program.Input.Clone()
	child.InputBound = true
	payload, err := canonicalWorkflowValue(child)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	snapshot := frame.Program
	snapshot.Payload = payload
	return snapshot, nil
}

func workflowInheritedChildScope(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	mergeState bool,
	override WorkflowScopeContext,
) (WorkflowScopeContext, error) {
	owner, err := workflowStateOwnerForFrame(frame, program)
	if err != nil {
		return WorkflowScopeContext{}, err
	}

	parent := cloneWorkflowScopeContext(program.Scope)
	output := WorkflowScopeContext{
		Child:                   true,
		MergeState:              mergeState,
		OuterState:              state.State.Clone(),
		StateOwner:              owner,
		TargetSets:              cloneWorkflowTargetSetStates(state.TargetSets),
		OuterOutcomes:           workflowScopeOutcomes(program, state),
		CausalEffectWorkItemIDs: workflowCausalEffectWorkItemIDs(state),
		OuterSubscriptions:      cloneWorkflowSubscriptionStates(state.Subscriptions),
		Context:                 cloneWorkflowCanonicalPointer(parent.Context),
		Item:                    cloneWorkflowCanonicalPointer(parent.Item),
		Index:                   cloneWorkflowIndex(parent.Index),
		Result:                  cloneWorkflowCanonicalPointer(parent.Result),
		Event:                   cloneWorkflowCanonicalPointer(parent.Event),
		Error:                   cloneWorkflowCanonicalPointer(parent.Error),
		Outcome:                 cloneWorkflowCanonicalPointer(parent.Outcome),
		Operation:               cloneWorkflowCanonicalPointer(parent.Operation),
		Completed:               cloneWorkflowCanonicalPointer(parent.Completed),
		Ports:                   clonePortInputs(parent.Ports),
		ChannelAliases:          cloneWorkflowChannelAliases(parent.ChannelAliases),
		Saga:                    cloneWorkflowSagaScope(parent.Saga),
		Cleanup:                 parent.Cleanup,
	}

	if len(override.OuterState) != 0 {
		output.OuterState = override.OuterState.Clone()
	}
	if override.StateOwner != nil {
		output.StateOwner = cloneWorkflowStateOwner(override.StateOwner)
	}
	if len(override.TargetSets) != 0 {
		output.TargetSets = cloneWorkflowTargetSetStates(override.TargetSets)
	}
	if len(override.CausalEffectWorkItemIDs) != 0 {
		output.CausalEffectWorkItemIDs = append([]WorkItemID(nil), override.CausalEffectWorkItemIDs...)
	}
	if override.Context != nil {
		output.Context = cloneWorkflowCanonicalPointer(override.Context)
	}
	if override.Item != nil {
		output.Item = cloneWorkflowCanonicalPointer(override.Item)
	}
	if override.Index != nil {
		output.Index = cloneWorkflowIndex(override.Index)
	}
	if override.Result != nil {
		output.Result = cloneWorkflowCanonicalPointer(override.Result)
	}
	if override.Event != nil {
		output.Event = cloneWorkflowCanonicalPointer(override.Event)
	}
	if override.Error != nil {
		output.Error = cloneWorkflowCanonicalPointer(override.Error)
	}
	if override.Outcome != nil {
		output.Outcome = cloneWorkflowCanonicalPointer(override.Outcome)
	}
	if override.Operation != nil {
		output.Operation = cloneWorkflowCanonicalPointer(override.Operation)
	}
	if override.Completed != nil {
		output.Completed = cloneWorkflowCanonicalPointer(override.Completed)
	}
	if len(override.Ports) != 0 {
		output.Ports = clonePortInputs(override.Ports)
	}
	if override.Saga != nil {
		output.Saga = cloneWorkflowSagaScope(override.Saga)
	}
	output.Cleanup = output.Cleanup || override.Cleanup
	return output, output.Validate()
}

func cloneWorkflowScopeContext(value WorkflowScopeContext) WorkflowScopeContext {
	output := value
	output.OuterState = value.OuterState.Clone()
	output.StateOwner = cloneWorkflowStateOwner(value.StateOwner)
	output.TargetSets = cloneWorkflowTargetSetStates(value.TargetSets)
	output.OuterOutcomes = cloneWorkflowStepOutcomes(value.OuterOutcomes)
	output.CausalEffectWorkItemIDs = append([]WorkItemID(nil), value.CausalEffectWorkItemIDs...)
	output.OuterSubscriptions = cloneWorkflowSubscriptionStates(value.OuterSubscriptions)
	output.Context = cloneWorkflowCanonicalPointer(value.Context)
	output.Item = cloneWorkflowCanonicalPointer(value.Item)
	output.Index = cloneWorkflowIndex(value.Index)
	output.Result = cloneWorkflowCanonicalPointer(value.Result)
	output.Event = cloneWorkflowCanonicalPointer(value.Event)
	output.Error = cloneWorkflowCanonicalPointer(value.Error)
	output.Outcome = cloneWorkflowCanonicalPointer(value.Outcome)
	output.Operation = cloneWorkflowCanonicalPointer(value.Operation)
	output.Completed = cloneWorkflowCanonicalPointer(value.Completed)
	output.Ports = clonePortInputs(value.Ports)
	output.ChannelAliases = cloneWorkflowChannelAliases(value.ChannelAliases)
	output.Saga = cloneWorkflowSagaScope(value.Saga)
	return output
}

func cloneWorkflowChannelAliases(
	values map[string]string,
) map[string]string {
	if len(values) == 0 {
		return nil
	}
	output := make(map[string]string, len(values))
	maps.Copy(output, values)
	return output
}

func validateWorkflowChannelAliases(values map[string]string) error {
	physicalNames := make(map[string]struct{}, len(values))
	for declared, physical := range values {
		if err := validateID("workflow declared channel alias", declared); err != nil {
			return err
		}
		if err := validateID("workflow private channel name", physical); err != nil {
			return err
		}
		if _, duplicate := physicalNames[physical]; duplicate {
			return errors.New("workflow private channel alias is duplicated")
		}
		physicalNames[physical] = struct{}{}
	}
	return nil
}

func cloneWorkflowTargetSetStates(
	values []WorkflowTargetSetState,
) []WorkflowTargetSetState {
	if len(values) == 0 {
		return nil
	}
	output := make([]WorkflowTargetSetState, len(values))
	for index := range values {
		output[index] = values[index]
		output[index].Members = append(
			[]string(nil),
			values[index].Members...,
		)
	}
	return output
}

func cloneWorkflowIndex(value *int) *int {
	if value == nil {
		return nil
	}
	output := *value
	return &output
}

func cloneWorkflowSagaScope(
	value *WorkflowSagaScope,
) *WorkflowSagaScope {
	if value == nil {
		return nil
	}
	output := *value
	return &output
}

func (c WorkflowChoose) Validate() error {
	for index, candidate := range c.Cases {
		if err := candidate.When.Validate(); err != nil {
			return fmt.Errorf("workflow choose case %d: %w", index, err)
		}
		if err := candidate.Do.Validate(); err != nil {
			return fmt.Errorf("workflow choose case %d: %w", index, err)
		}
	}
	if c.Default != nil {
		return c.Default.Validate()
	}
	return nil
}

func (c WorkflowCompletion) Validate() error {
	switch workflowCompletionMode(c) {
	case WorkflowCompletionModeAll,
		WorkflowCompletionModeFirstCompleted,
		WorkflowCompletionModeFirstSuccess:
		if c.Quorum != 0 {
			return errors.New("workflow completion quorum is valid only for quorum mode")
		}
	case WorkflowCompletionModeQuorum:
		if c.Quorum <= 0 {
			return errors.New("workflow completion quorum must be positive")
		}
	default:
		return errors.New("workflow completion mode is invalid")
	}
	return nil
}

func (p WorkflowParallel) Validate() error {
	if err := p.Completion.Validate(); err != nil {
		return err
	}
	switch p.Errors {
	case "", WorkflowCollectionErrorFailFast, WorkflowCollectionErrorCollect:
	default:
		return errors.New("workflow parallel error policy is invalid")
	}
	switch workflowRemainingMode(p.Remaining) {
	case WorkflowRemainingCancel, WorkflowRemainingWait:
	default:
		return errors.New("workflow parallel remaining policy is invalid")
	}
	if len(p.Branches) == 0 {
		return errors.New("workflow parallel has no branches")
	}

	seen := make(map[string]struct{}, len(p.Branches))
	for _, branch := range p.Branches {
		if strings.TrimSpace(branch.Name) == "" {
			return errors.New("workflow parallel branch has no name")
		}
		if _, duplicate := seen[branch.Name]; duplicate {
			return errors.New("workflow parallel branch is duplicated")
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
		return errors.New("workflow foreach item schema is invalid")
	}
	if f.Concurrency < 0 {
		return errors.New("workflow foreach limits are invalid")
	}
	if f.MaxItems != nil && *f.MaxItems < 0 {
		return errors.New("workflow foreach limits are invalid")
	}
	if err := f.Completion.Validate(); err != nil {
		return err
	}
	switch f.Errors {
	case "", WorkflowCollectionErrorFailFast, WorkflowCollectionErrorCollect:
	default:
		return errors.New("workflow foreach error policy is invalid")
	}
	switch workflowRemainingMode(f.Remaining) {
	case WorkflowRemainingCancel, WorkflowRemainingWait:
	default:
		return errors.New("workflow foreach remaining policy is invalid")
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
		return errors.New("workflow repeat requires exactly one control")
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
	if r.Lifetime != "" && r.Lifetime != WorkflowRepeatLifetimeUntilCancelled {
		return errors.New("workflow repeat lifetime is invalid")
	}
	if r.State != nil {
		if !r.State.Schema.Valid() || len(r.State.Schema) == 0 {
			return errors.New("workflow repeat state schema is invalid")
		}
		if err := r.State.Initial.Validate(); err != nil {
			return err
		}
	}
	if r.MaxIterations != nil && *r.MaxIterations < 0 {
		return errors.New("workflow repeat maxiterations is invalid")
	}
	switch r.OnExhausted {
	case "", WorkflowRepeatOnExhaustedFail, WorkflowRepeatOnExhaustedReturn:
	default:
		return errors.New("workflow repeat onexhausted is invalid")
	}
	if err := validateWorkflowTimeout(r.Timeout); err != nil {
		return err
	}
	return r.Do.Validate()
}

func workflowReleasePendingClaims(
	state *WorkflowState,
	pending *WorkflowPending,
) {
	if state == nil || pending == nil || !pending.ClaimsBound {
		return
	}
	state.ReleaseFrameClaims = true
}

// WalkWorkflowBlock visits a lowered WorkflowBlock tree in deterministic
// pre-order. It is the single traversal mechanism shared by channel
// namespacing, resource namespacing, and setup declaration-path application.
func WalkWorkflowBlock(
	root *WorkflowBlock,
	visitor WorkflowTreeVisitor,
) error {
	if root == nil {
		return nil
	}
	err := walkWorkflowBlock(
		WorkflowBlockNode{
			Block: root,
			Role:  WorkflowBlockRoleRoot,
		},
		visitor,
	)
	if errors.Is(err, ErrStopWorkflowWalk) {
		return nil
	}
	return err
}

func walkWorkflowBlock(
	node WorkflowBlockNode,
	visitor WorkflowTreeVisitor,
) error {
	if node.Block == nil {
		return nil
	}
	if visitor.OnBlock != nil {
		if err := visitor.OnBlock(node); err != nil {
			return err
		}
	}

	block := node.Block
	for stepIndex := range block.Steps {
		step := &block.Steps[stepIndex]
		if visitor.OnStep != nil {
			if err := visitor.OnStep(WorkflowStepNode{
				Block: block,
				Step:  step,
				Index: stepIndex,
			}); err != nil {
				return err
			}
		}
		if step.Effect != nil {
			if err := walkWorkflowBlock(WorkflowBlockNode{
				Block:       step.Effect.Reconcile,
				ParentBlock: block,
				ParentStep:  step,
				Role:        WorkflowBlockRoleEffectReconcile,
			}, visitor); err != nil {
				return err
			}
			if err := walkWorkflowBlock(WorkflowBlockNode{
				Block:       step.Effect.Compensate,
				ParentBlock: block,
				ParentStep:  step,
				Role:        WorkflowBlockRoleEffectCompensate,
			}, visitor); err != nil {
				return err
			}
		}

		switch {
		case step.Scope != nil:
			if err := walkWorkflowBlock(WorkflowBlockNode{
				Block:       step.Scope,
				ParentBlock: block,
				ParentStep:  step,
				Role:        WorkflowBlockRoleScope,
			}, visitor); err != nil {
				return err
			}
		case step.Choose != nil:
			for caseIndex := range step.Choose.Cases {
				if err := walkWorkflowBlock(WorkflowBlockNode{
					Block:       &step.Choose.Cases[caseIndex].Do,
					ParentBlock: block,
					ParentStep:  step,
					Role:        WorkflowBlockRoleChooseCase,
					Index:       caseIndex,
				}, visitor); err != nil {
					return err
				}
			}
			if err := walkWorkflowBlock(WorkflowBlockNode{
				Block:       step.Choose.Default,
				ParentBlock: block,
				ParentStep:  step,
				Role:        WorkflowBlockRoleChooseDefault,
			}, visitor); err != nil {
				return err
			}
		case step.Parallel != nil:
			for branchIndex := range step.Parallel.Branches {
				branch := &step.Parallel.Branches[branchIndex]
				if err := walkWorkflowBlock(WorkflowBlockNode{
					Block:       &branch.Do,
					ParentBlock: block,
					ParentStep:  step,
					Role:        WorkflowBlockRoleParallelBranch,
					Name:        branch.Name,
					Index:       branchIndex,
				}, visitor); err != nil {
					return err
				}
			}
		case step.Foreach != nil:
			if err := walkWorkflowBlock(WorkflowBlockNode{
				Block:       &step.Foreach.Do,
				ParentBlock: block,
				ParentStep:  step,
				Role:        WorkflowBlockRoleForeach,
			}, visitor); err != nil {
				return err
			}
		case step.Repeat != nil:
			if err := walkWorkflowBlock(WorkflowBlockNode{
				Block:       &step.Repeat.Do,
				ParentBlock: block,
				ParentStep:  step,
				Role:        WorkflowBlockRoleRepeat,
			}, visitor); err != nil {
				return err
			}
		case step.Machine != nil:
			for stateIndex := range step.Machine.States {
				machineState := &step.Machine.States[stateIndex]
				if err := walkWorkflowBlock(WorkflowBlockNode{
					Block:       machineState.Entry,
					ParentBlock: block,
					ParentStep:  step,
					Role:        WorkflowBlockRoleMachineEntry,
					Name:        machineState.Name,
				}, visitor); err != nil {
					return err
				}
				if err := walkWorkflowBlock(WorkflowBlockNode{
					Block:       machineState.Exit,
					ParentBlock: block,
					ParentStep:  step,
					Role:        WorkflowBlockRoleMachineExit,
					Name:        machineState.Name,
				}, visitor); err != nil {
					return err
				}
				for transitionIndex := range machineState.Transitions {
					if err := walkWorkflowBlock(WorkflowBlockNode{
						Block:       machineState.Transitions[transitionIndex].Do,
						ParentBlock: block,
						ParentStep:  step,
						Role:        WorkflowBlockRoleMachineTransition,
						Name:        machineState.Name,
						Index:       transitionIndex,
					}, visitor); err != nil {
						return err
					}
				}
			}
		case step.Saga != nil:
			if err := walkWorkflowBlock(WorkflowBlockNode{
				Block:       &step.Saga.Do,
				ParentBlock: block,
				ParentStep:  step,
				Role:        WorkflowBlockRoleSaga,
			}, visitor); err != nil {
				return err
			}
		}
	}
	for catchIndex := range block.Catch {
		if err := walkWorkflowBlock(WorkflowBlockNode{
			Block:       &block.Catch[catchIndex].Do,
			ParentBlock: block,
			Role:        WorkflowBlockRoleCatch,
			Index:       catchIndex,
		}, visitor); err != nil {
			return err
		}
	}
	return walkWorkflowBlock(WorkflowBlockNode{
		Block:       block.Finally,
		ParentBlock: block,
		Role:        WorkflowBlockRoleFinally,
	}, visitor)
}
