package sessionv3

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"cel.dev/cel-go/cel"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	WorkflowInterpreterID = "artifact.workflow.v1"

	// WorkflowInterpreterVersion is the current typed Workflow program
	// interpreter. This development-only payload replaces the prior
	// development payload in place and intentionally remains version 1.
	WorkflowInterpreterVersion = uint32(1)

	workflowChildCallFramePrefix = "workflow-child-frame"
	workflowChildCallWorkPrefix  = "workflow-child-call"
	workflowChildCallIDSeparator = ":"

	maxWorkflowProgramJSONBytes = 2 << 20
)

type WorkflowValueKind string

const (
	WorkflowValueLiteral    WorkflowValueKind = "literal"
	WorkflowValueExpression WorkflowValueKind = "expression"
	WorkflowValueObject     WorkflowValueKind = "object"
	WorkflowValueArray      WorkflowValueKind = "array"
)

// WorkflowValue is a typed durable expression template. Literal values remain
// jsonutil.CanonicalValue because Workflow input and output can be arbitrary JSON, but
// no runtime struct uses json.RawMessage as its public representation.
type WorkflowValue struct {
	Kind       WorkflowValueKind       `json:"kind"`
	Literal    jsonutil.CanonicalValue `json:"literal,omitempty"`
	Expression string                  `json:"expression,omitempty"`
	Object     []WorkflowValuePair     `json:"object,omitempty"`
	Array      []WorkflowValue         `json:"array,omitempty"`
}

type WorkflowValuePair struct {
	Key   string        `json:"key"`
	Value WorkflowValue `json:"value"`
}

type WorkflowCallMode string

const (
	WorkflowCallModeInvoke  WorkflowCallMode = "invoke"
	WorkflowCallModeStart   WorkflowCallMode = "start"
	WorkflowCallModeHandoff WorkflowCallMode = "handoff"
)

type WorkflowChildOwnership struct {
	Owner            WorkflowValue   `json:"owner"`
	CancelWithParent bool            `json:"cancelWithParent"`
	Budget           ExecutionBudget `json:"budget"`
}

func (o WorkflowChildOwnership) Validate() error {
	return o.Owner.Validate()
}

type WorkflowCallableInterface struct {
	Modes         []WorkflowCallMode           `json:"modes,omitempty"`
	InputSchema   jsonutil.CanonicalValue      `json:"inputSchema"`
	OutputSchema  jsonutil.CanonicalValue      `json:"outputSchema"`
	ErrorSchema   jsonutil.CanonicalValue      `json:"errorSchema,omitempty"`
	ReceiptSchema jsonutil.CanonicalValue      `json:"receiptSchema,omitempty"`
	Context       []WorkflowContextRequirement `json:"context,omitempty"`
	Ports         []WorkflowCallablePort       `json:"ports,omitempty"`
	Limits        *ExecutionBudget             `json:"limits,omitempty"`
}

type WorkflowProgram struct {
	Version uint32 `json:"version"`

	InputSchema  jsonutil.CanonicalValue `json:"inputSchema"`
	OutputSchema jsonutil.CanonicalValue `json:"outputSchema"`
	Input        jsonutil.CanonicalValue `json:"input"`
	InputBound   bool                    `json:"inputBound"`

	Root         WorkflowBlock                `json:"root"`
	Targets      map[string]WorkflowTarget    `json:"targets"`
	Requirements []WorkflowRequirementFact    `json:"requirements,omitempty"`
	TargetSets   map[string]WorkflowTargetSet `json:"targetSets,omitempty"`
	Events       []WorkflowEventContract      `json:"events,omitempty"`
	Resources    []WorkflowResourceDefinition `json:"resources,omitempty"`
	Channels     []WorkflowChannel            `json:"channels,omitempty"`
	Ports        []WorkflowPort               `json:"ports,omitempty"`
	Triggers     []WorkflowTrigger            `json:"triggers,omitempty"`
	Limits       ExecutionBudget              `json:"limits"`

	// Scope is empty for a root Workflow Program. Child block ProgramFrames
	// receive an immutable parent snapshot through this typed envelope.
	Scope WorkflowScopeContext `json:"scope"`
}

type WorkflowBlock struct {
	DeclarationPath string                 `json:"declarationPath"`
	State           *WorkflowStateSpec     `json:"state,omitempty"`
	Listen          []WorkflowSubscription `json:"listen,omitempty"`
	Steps           []WorkflowStep         `json:"steps"`
	Output          WorkflowValue          `json:"output"`
	Catch           []WorkflowCatchHandler `json:"catch,omitempty"`
	Finally         *WorkflowBlock         `json:"finally,omitempty"`
}

type WorkflowStateSpec struct {
	Schema  jsonutil.CanonicalValue `json:"schema"`
	Initial WorkflowValue           `json:"initial"`
}

type WorkflowStep struct {
	ID              string                  `json:"id"`
	DeclarationPath string                  `json:"declarationPath"`
	Timeout         WorkflowDuration        `json:"timeout,omitempty"`
	When            *WorkflowValue          `json:"when,omitempty"`
	Retry           *RetryPolicy            `json:"retry,omitempty"`
	Effect          *WorkflowStepEffect     `json:"effect,omitempty"`
	OutputSchema    jsonutil.CanonicalValue `json:"outputSchema,omitempty"`
	Claims          []WorkflowResourceClaim `json:"claims,omitempty"`
	Limits          *ExecutionBudget        `json:"limits,omitempty"`

	Call      *WorkflowCall      `json:"call,omitempty"`
	Transform *WorkflowTransform `json:"transform,omitempty"`
	Set       *WorkflowSet       `json:"set,omitempty"`
	Scope     *WorkflowBlock     `json:"scope,omitempty"`
	Saga      *WorkflowSaga      `json:"saga,omitempty"`
	Choose    *WorkflowChoose    `json:"choose,omitempty"`
	Parallel  *WorkflowParallel  `json:"parallel,omitempty"`
	Foreach   *WorkflowForeach   `json:"foreach,omitempty"`
	Repeat    *WorkflowRepeat    `json:"repeat,omitempty"`
	Machine   *WorkflowMachine   `json:"machine,omitempty"`
	Wait      *WorkflowWait      `json:"wait,omitempty"`
	Emit      *WorkflowEmit      `json:"emit,omitempty"`
	Return    *WorkflowReturn    `json:"return,omitempty"`
	Cancel    *WorkflowCancel    `json:"cancel,omitempty"`
	Fail      *WorkflowFail      `json:"fail,omitempty"`
	Break     bool               `json:"break,omitempty"`
	Continue  bool               `json:"continue,omitempty"`
}

type WorkflowCall struct {
	Target                 WorkflowValue              `json:"target"`
	AllowedTargets         []string                   `json:"allowedTargets,omitempty"`
	AllowedTargetSet       string                     `json:"allowedTargetSet,omitempty"`
	Input                  WorkflowValue              `json:"input"`
	Context                WorkflowValue              `json:"context"`
	DeclarationPath        string                     `json:"declarationPath"`
	Ports                  []WorkflowPortBinding      `json:"ports,omitempty"`
	Contract               *WorkflowCallableInterface `json:"contract,omitempty"`
	SetContract            *WorkflowCallableInterface `json:"setContract,omitempty"`
	Mode                   WorkflowCallMode           `json:"mode,omitempty"`
	Lifetime               WorkflowChildLifetime      `json:"lifetime,omitempty"`
	Conversation           *WorkflowValue             `json:"conversation,omitempty"`
	Ownership              *WorkflowChildOwnership    `json:"ownership,omitempty"`
	OnCompletion           string                     `json:"onCompletion,omitempty"`
	InternalLifecycleChild bool                       `json:"internalLifecycleChild,omitempty"`
	LLMInvocation          *WorkflowLLMInvocation     `json:"llmInvocation,omitempty"`
}

// WorkflowLLMInvocation is the setup-lowered typed Model or Agent invocation
// form. Workflow setup converts source data to this value before Runtime
// receives a ProgramSnapshot. Runtime evaluates only typed templates and does
// not decode a generic Workflow JSON request envelope.
type WorkflowLLMInvocation struct {
	Messages     []WorkflowLLMMessage     `json:"messages"`
	Instructions []WorkflowLLMInstruction `json:"instructions,omitempty"`
	Defaults     *SessionDefaultsOverride `json:"defaults,omitempty"`
}

type WorkflowLLMMessage struct {
	Role        MessageRole    `json:"role"`
	Text        *WorkflowValue `json:"text,omitempty"`
	Attachments []Attachment   `json:"attachments,omitempty"`
}

type WorkflowLLMInstruction struct {
	ID      string        `json:"id,omitempty"`
	Label   string        `json:"label,omitempty"`
	Content WorkflowValue `json:"content"`
}

type WorkflowTransform struct {
	Value        WorkflowValue           `json:"value"`
	OutputSchema jsonutil.CanonicalValue `json:"outputSchema,omitempty"`
}

type WorkflowCancel struct {
	Handle WorkflowValue  `json:"handle"`
	Reason *WorkflowValue `json:"reason,omitempty"`
}

type WorkflowSet struct {
	Values []WorkflowValuePair `json:"values,omitempty"`
	Remove []string            `json:"remove,omitempty"`
}

type WorkflowReturn struct {
	To    WorkflowReturnDestination `json:"to,omitempty"`
	Value WorkflowValue             `json:"value"`
}

type WorkflowFail struct {
	Code      string         `json:"code"`
	Category  ErrorCategory  `json:"category,omitempty"`
	Message   WorkflowValue  `json:"message"`
	Retryable bool           `json:"retryable,omitempty"`
	Details   *WorkflowValue `json:"details,omitempty"`
}

type WorkflowWait struct {
	After *WorkflowValue `json:"after,omitempty"`
	At    *WorkflowValue `json:"at,omitempty"`

	Event      *WorkflowEventWait      `json:"event,omitempty"`
	Channel    *WorkflowChannelWait    `json:"channel,omitempty"`
	Completion *WorkflowCompletionWait `json:"completion,omitempty"`
	Human      *WorkflowHumanWait      `json:"human,omitempty"`
}

type WorkflowTargetKind string

const (
	WorkflowTargetTool         WorkflowTargetKind = "tool"
	WorkflowTargetMCPTool      WorkflowTargetKind = "mcpTool"
	WorkflowTargetService      WorkflowTargetKind = "service"
	WorkflowTargetActivity     WorkflowTargetKind = "activity"
	WorkflowTargetWorkflow     WorkflowTargetKind = "workflow"
	WorkflowTargetAgent        WorkflowTargetKind = "agent"
	WorkflowTargetModel        WorkflowTargetKind = "model"
	WorkflowTargetProviderTool WorkflowTargetKind = "providerTool"
)

type WorkflowTarget struct {
	Kind WorkflowTargetKind `json:"kind"`

	Artifact     *ArtifactPin               `json:"artifact,omitempty"`
	MCP          *WorkflowMCP               `json:"mcp,omitempty"`
	ProviderTool *ProviderManagedToolTarget `json:"providerTool,omitempty"`
	Transport    WorkflowTargetTransport    `json:"transport"`

	ServiceID      string `json:"serviceID,omitempty"`
	ActivityID     string `json:"activityID,omitempty"`
	BindingVersion string `json:"bindingVersion,omitempty"`

	NestedProgram   *ProgramSnapshot           `json:"nestedProgram,omitempty"`
	ToolName        string                     `json:"toolName,omitempty"`
	Contract        *WorkflowCallableInterface `json:"contract,omitempty"`
	Effect          EffectContract             `json:"effect"`
	KeyedCapable    bool                       `json:"keyedCapable,omitempty"`
	DeclarationPath string                     `json:"declarationPath,omitempty"`
}

type WorkflowMCP struct {
	Server           ArtifactPin `json:"server"`
	ToolName         string      `json:"toolName"`
	ProviderToolName string      `json:"providerToolName"`
	ChoiceID         string      `json:"choiceID"`
	ToolDigest       string      `json:"toolDigest"`
	ApprovalRule     string      `json:"approvalRule"`
	ExecutionMode    string      `json:"executionMode"`
	DiscoveryDigest  string      `json:"discoveryDigest"`
}

type WorkflowStepOutcomeStatus string

const (
	WorkflowStepOutcomeSucceeded WorkflowStepOutcomeStatus = "succeeded"
	WorkflowStepOutcomeSkipped   WorkflowStepOutcomeStatus = "skipped"
	WorkflowStepOutcomeCancelled WorkflowStepOutcomeStatus = "cancelled"
	WorkflowStepOutcomeFailed    WorkflowStepOutcomeStatus = "failed"
	WorkflowStepOutcomeUncertain WorkflowStepOutcomeStatus = "uncertain"
)

type WorkflowStepOutcome struct {
	ID              string                    `json:"id"`
	Status          WorkflowStepOutcomeStatus `json:"status"`
	Output          jsonutil.CanonicalValue   `json:"output,omitempty"`
	Target          string                    `json:"target,omitempty"`
	Receipt         *ExternalReceipt          `json:"receipt,omitempty"`
	Error           *SanitizedError           `json:"error,omitempty"`
	Effect          EffectState               `json:"effectState,omitempty"`
	DeclarationPath string                    `json:"declarationPath,omitempty"`
}

// WorkflowTargetSetResolution is persisted while a resolver WorkItem is
// running. It prevents a later retry, pause, or restore from evaluating the
// target-set expression again and changing the accepted membership.
type WorkflowTargetSetResolution struct {
	Name       string     `json:"name"`
	WorkItemID WorkItemID `json:"workItemID"`
	Members    []string   `json:"members"`
}

// WorkflowState is the durable typed state of one sequential Workflow frame.
// It contains semantic state and step outcomes only. It never contains CEL
// programs, compiled schemas, Artifact services, clients, callbacks, or
// private authorization data.
type WorkflowState struct {
	Version uint32 `json:"version"`

	Index                   int                          `json:"index"`
	PendingPath             string                       `json:"pendingPath,omitempty"`
	PendingDeclarationPath  string                       `json:"pendingDeclarationPath,omitempty"`
	PendingTarget           string                       `json:"pendingTarget,omitempty"`
	Pending                 *WorkflowPending             `json:"pending,omitempty"`
	Exit                    WorkflowExitKind             `json:"exit,omitempty"`
	ExitValue               jsonutil.CanonicalValue      `json:"exitValue,omitempty"`
	LogicalTime             time.Time                    `json:"logicalTime,omitzero"`
	Deadline                *time.Time                   `json:"deadline,omitempty"`
	State                   jsonutil.CanonicalValue      `json:"state"`
	Messages                jsonutil.CanonicalValue      `json:"messages,omitempty"`
	Outcomes                []WorkflowStepOutcome        `json:"outcomes"`
	Subscriptions           []WorkflowSubscriptionState  `json:"subscriptions,omitempty"`
	CancelChildPaths        []string                     `json:"cancelChildPaths,omitempty"`
	CancelWorkItemIDs       []WorkItemID                 `json:"cancelWorkItemIDs,omitempty"`
	ReleaseFrameClaims      bool                         `json:"releaseFrameClaims,omitempty"`
	CancellationPending     bool                         `json:"cancellationPending,omitempty"`
	Terminal                *WorkflowTerminalState       `json:"terminal,omitempty"`
	Saga                    *WorkflowSagaScope           `json:"saga,omitempty"`
	TargetSets              []WorkflowTargetSetState     `json:"targetSets,omitempty"`
	TargetSetResolution     *WorkflowTargetSetResolution `json:"targetSetResolution,omitempty"`
	ChildCallWorkItemID     WorkItemID                   `json:"childCallWorkItemID,omitempty"`
	LastEffectWorkItemID    WorkItemID                   `json:"lastEffectWorkItemID,omitempty"`
	CausalEffectWorkItemIDs []WorkItemID                 `json:"causalEffectWorkItemIDs,omitempty"`
}

// WorkflowResourceClaim is a declaration-level resource claim template. Key
// becomes a concrete ResourceClaim when the controlling WorkItem is created.
type WorkflowResourceClaim struct {
	Resource string         `json:"resource"`
	Units    int            `json:"units"`
	Key      *WorkflowValue `json:"key,omitempty"`
}

func (c WorkflowResourceClaim) Validate() error {
	if err := validateID("workflow resource claim resource", c.Resource); err != nil {
		return err
	}
	if c.Units < 1 {
		return errors.New("workflow resource claim units are invalid")
	}
	if c.Key != nil {
		return c.Key.Validate()
	}
	return nil
}

type WorkflowInterpreter struct {
	env *cel.Env
}

func (s *WorkflowState) Validate() error {
	if s.Version != WorkflowInterpreterVersion ||
		s.Index < 0 ||
		!s.State.Valid() ||
		!s.Messages.Valid() ||
		s.Outcomes == nil {
		return errors.New("Workflow Program state is invalid")
	}
	seen := make(map[string]struct{}, len(s.Outcomes))
	for _, outcome := range s.Outcomes {
		if err := outcome.Validate(); err != nil {
			return fmt.Errorf("workflow step outcome: %w", err)
		}
		if _, duplicate := seen[outcome.ID]; duplicate {
			return errors.New("Workflow Program step outcome is duplicated")
		}
		seen[outcome.ID] = struct{}{}
	}
	if s.TargetSetResolution != nil {
		if err := s.TargetSetResolution.Validate(); err != nil {
			return err
		}
	}
	if s.ChildCallWorkItemID != "" {
		if err := validateID(
			"workflow child call WorkItem ID",
			string(s.ChildCallWorkItemID),
		); err != nil {
			return err
		}
	}
	if s.LastEffectWorkItemID != "" {
		if err := validateID(
			"workflow last effect work item ID",
			string(s.LastEffectWorkItemID),
		); err != nil {
			return err
		}
	}
	if err := validateWorkflowCausalEffectWorkItemIDs(
		s.CausalEffectWorkItemIDs,
	); err != nil {
		return err
	}
	for _, subscription := range s.Subscriptions {
		if err := subscription.Validate(); err != nil {
			return err
		}
	}
	for _, id := range s.CancelWorkItemIDs {
		if err := validateID(
			"workflow cancellation work item ID",
			string(id),
		); err != nil {
			return err
		}
	}
	if s.Terminal != nil {
		if err := s.Terminal.Validate(); err != nil {
			return err
		}
	}
	if s.Saga != nil {
		if err := s.Saga.Validate(); err != nil {
			return err
		}
	}
	if err := validateWorkflowTargetSetStates(s.TargetSets); err != nil {
		return err
	}
	if err := s.validateStructuredState(); err != nil {
		return err
	}
	return nil
}

func (s *WorkflowState) recordOutcome(
	outcome WorkflowStepOutcome,
) error {
	if s == nil || strings.TrimSpace(outcome.ID) == "" {
		return errors.New("workflow step outcome is invalid")
	}
	for _, current := range s.Outcomes {
		if current.ID == outcome.ID {
			return fmt.Errorf(
				"workflow step %q already has an outcome",
				outcome.ID,
			)
		}
	}
	s.Outcomes = append(s.Outcomes, cloneWorkflowStepOutcome(outcome))
	return nil
}

// clearPendingStep clears the source-step boundary only after that step has
// settled. Structured control flow retains the boundary while child work is
// active, then calls this method before it asks drive to select the next step.
func (s *WorkflowState) clearPendingStep() {
	if s == nil {
		return
	}
	s.PendingPath = ""
	s.PendingDeclarationPath = ""
	s.PendingTarget = ""
	s.Deadline = nil
	s.ChildCallWorkItemID = ""
}

func validateWorkflowCausalEffectWorkItemIDs(
	values []WorkItemID,
) error {
	seen := make(map[WorkItemID]struct{}, len(values))
	for _, value := range values {
		if err := validateID(
			"workflow causal effect work item ID",
			string(value),
		); err != nil {
			return err
		}
		if _, duplicate := seen[value]; duplicate {
			return errors.New("workflow causal effect work item ID is duplicated")
		}
		seen[value] = struct{}{}
	}
	return nil
}

func mergeWorkflowCausalEffectWorkItemIDs(
	groups ...[]WorkItemID,
) []WorkItemID {
	seen := make(map[WorkItemID]struct{})
	output := make([]WorkItemID, 0)
	for _, group := range groups {
		for _, value := range group {
			if value == "" {
				continue
			}
			if _, duplicate := seen[value]; duplicate {
				continue
			}
			seen[value] = struct{}{}
			output = append(output, value)
		}
	}
	return output
}

func workflowCausalEffectWorkItemIDs(
	state WorkflowState,
) []WorkItemID {
	return mergeWorkflowCausalEffectWorkItemIDs(
		[]WorkItemID{state.LastEffectWorkItemID},
		state.CausalEffectWorkItemIDs,
	)
}

func setWorkflowCausalEffectWorkItemIDs(
	state *WorkflowState,
	values []WorkItemID,
) {
	if state == nil {
		return
	}
	state.CausalEffectWorkItemIDs = mergeWorkflowCausalEffectWorkItemIDs(
		values,
	)
	state.LastEffectWorkItemID = ""
	if len(state.CausalEffectWorkItemIDs) != 0 {
		state.LastEffectWorkItemID = state.CausalEffectWorkItemIDs[len(state.CausalEffectWorkItemIDs)-1]
	}
}

func workflowFrameCausalEffectWorkItemIDs(
	frame ProgramFrame,
) []WorkItemID {
	switch frame.Program.InterpreterID {
	case WorkflowInterpreterID:
		_, state, err := decodeWorkflowProgram(frame)
		if err == nil {
			return workflowCausalEffectWorkItemIDs(state)
		}
	case BuiltinRetryBoundaryInterpreterID:
		state, err := jsonutil.DecodeCanonicalValue[retryProgramState](
			frame.State.Data,
		)
		if err == nil {
			return append(
				[]WorkItemID(nil),
				state.CausalEffectWorkItemIDs...,
			)
		}
	}
	return nil
}

func NewWorkflowInterpreter() (*WorkflowInterpreter, error) {
	env, err := cel.NewEnv(
		cel.Variable("input", cel.DynType),
		cel.Variable("state", cel.DynType),
		cel.Variable("steps", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("context", cel.DynType),
		cel.Variable("messages", cel.DynType),
		cel.Variable("item", cel.DynType),
		cel.Variable("index", cel.DynType),
		cel.Variable("result", cel.DynType),
		cel.Variable("event", cel.DynType),
		cel.Variable("error", cel.DynType),
		cel.Variable("outcome", cel.DynType),
		cel.Variable("completed", cel.DynType),
		cel.Variable("ports", cel.DynType),
		cel.Variable("operation", cel.DynType),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"construct Workflow expression environment: %w",
			err,
		)
	}
	return &WorkflowInterpreter{env: env}, nil
}

func (*WorkflowInterpreter) ID() string {
	return WorkflowInterpreterID
}

func (*WorkflowInterpreter) Version() uint32 {
	return WorkflowInterpreterVersion
}

func (i *WorkflowInterpreter) ValidateSnapshot(
	snapshot ProgramSnapshot,
) error {
	if snapshot.InterpreterID != WorkflowInterpreterID ||
		snapshot.InterpreterVersion != WorkflowInterpreterVersion {
		return errors.New("unsupported Workflow interpreter version")
	}
	if snapshot.Artifact == nil {
		return errors.New("Workflow Program snapshot has no Artifact pin")
	}
	if err := validatePin(*snapshot.Artifact); err != nil {
		return err
	}
	if snapshot.DefinitionDigest != snapshot.Artifact.Digest {
		return errors.New("Workflow Program definition digest is invalid")
	}

	program, err := jsonutil.DecodeCanonicalValue[WorkflowProgram](snapshot.Payload)
	if err != nil {
		return err
	}
	return program.Validate()
}

func (*WorkflowInterpreter) ValidateState(
	state ProgramState,
) error {
	if state.Type != WorkflowInterpreterID ||
		state.Version != WorkflowInterpreterVersion {
		return errors.New("unsupported Workflow state version")
	}

	value, err := jsonutil.DecodeCanonicalValue[WorkflowState](state.Data)
	if err != nil {
		return err
	}
	return value.Validate()
}

func (i *WorkflowInterpreter) InitialState(
	snapshot ProgramSnapshot,
	logicalTime time.Time,
) (ProgramState, error) {
	program, err := workflowProgramFromSnapshot(snapshot)
	if err != nil {
		return ProgramState{}, err
	}
	if logicalTime.IsZero() {
		return ProgramState{}, errors.New(
			"workflow logical initialization time is required",
		)
	}

	initialState := jsonutil.MustCanonicalValue(map[string]jsonutil.CanonicalValue{})
	if program.Scope.Child && len(program.Scope.OuterState) != 0 {
		initialState = program.Scope.OuterState.Clone()
	}
	state := WorkflowState{
		Version:     WorkflowInterpreterVersion,
		State:       initialState,
		Outcomes:    []WorkflowStepOutcome{},
		TargetSets:  program.Scope.TargetSets,
		Messages:    jsonutil.MustCanonicalValue(map[string][]WorkflowChannelMessageSnapshot{}),
		LogicalTime: logicalTime.UTC(),
	}
	setWorkflowCausalEffectWorkItemIDs(
		&state,
		program.Scope.CausalEffectWorkItemIDs,
	)
	if program.Root.State != nil {
		value, err := i.evaluate(
			program.Root.State.Initial,
			program,
			state,
		)
		if err != nil {
			return ProgramState{}, err
		}
		if err := validateWorkflowSchema(
			program.Root.State.Schema,
			value,
		); err != nil {
			return ProgramState{}, err
		}
		state.State = value
	}

	if program.Scope.Child &&
		program.Root.State == nil &&
		len(program.Scope.OuterState) != 0 {
		state.State = program.Scope.OuterState.Clone()
	}

	if err := i.activateWorkflowSubscriptions(program, &state); err != nil {
		return ProgramState{}, err
	}

	return encodeWorkflowProgramState(state)
}

func (i *WorkflowInterpreter) Advance(
	frame ProgramFrame,
	execution Execution,
	event ProgramEvent,
) (ProgramTransition, error) {
	program, state, err := decodeWorkflowProgram(frame)
	if err != nil {
		return ProgramTransition{}, err
	}

	//nolint:gocritic // If else is fine.
	if !event.At.IsZero() {
		state.LogicalTime = event.At.UTC()
	} else if !frame.UpdatedAt.IsZero() {
		state.LogicalTime = frame.UpdatedAt.UTC()
	} else {
		state.LogicalTime = frame.CreatedAt.UTC()
	}
	ensureWorkflowSubscriptionOwners(&state, frame.ID)

	if err := captureWorkflowMessages(program, &state, execution); err != nil {
		return ProgramTransition{}, err
	}

	if state.Terminal != nil {
		return i.advanceWorkflowTerminal(
			frame,
			execution,
			program,
			state,
			event,
		)
	}

	var transition ProgramTransition
	switch event.Kind {
	case ProgramEventCancelled:
		if state.Pending != nil && state.Pending.Kind == WorkflowPendingSaga {
			sagaTransition, _, sagaErr := i.beginSagaCompensation(
				frame,
				execution,
				program,
				state,
				ProgramFrameStateCancelled,
				workflowTerminalFailure(ProgramFrameStateCancelled, nil),
			)
			if sagaErr != nil {
				return ProgramTransition{}, sagaErr
			}
			return sagaTransition, nil
		}
		if state.Pending != nil &&
			state.Pending.Kind == WorkflowPendingReconciliation {
			state.CancellationPending = true
			return i.driveReconciliation(
				frame,
				execution,
				program,
				state,
			)
		}
		status := ProgramFrameStateCancelled
		transition, err = workflowTransition(
			state,
			&status,
			nil,
			nil,
		)
		if err == nil {
			transition.Error = &SanitizedError{
				Code:     "workflow_cancelled",
				Category: ErrorCategoryCancellation,
				Message:  "workflow execution was cancelled",
			}
		}

	case ProgramEventWorkSettled:
		if handled, mergeErr := i.settleParentStateMerge(
			frame,
			execution,
			program,
			&state,
			event.WorkItemID,
		); handled {
			if mergeErr != nil {
				if failure, found := errors.AsType[workflowChildError](mergeErr); found {
					transition, err = workflowFailureTransition(state, &failure.errorValue)
				} else {
					transition, err = workflowFailure(state, "workflow_state_merge_failed", mergeErr.Error())
				}
			} else {
				transition, err = i.drive(frame, execution, program, state)
			}
			break
		}
		if reconciliationTransition, handled, settleErr := i.settleReconciliationIntent(
			frame,
			execution,
			program,
			state,
			event.WorkItemID,
		); handled {
			if settleErr != nil {
				transition, err = workflowFailure(
					state,
					"workflow_reconciliation_failed",
					settleErr.Error(),
				)
			} else {
				transition = reconciliationTransition
			}
			break
		}
		if reviewTransition, handled, reviewErr := i.settleReconciliationReview(
			frame,
			execution,
			program,
			state,
			event.WorkItemID,
		); handled {
			if reviewErr != nil {
				transition, err = workflowFailure(state, "workflow_reconciliation_failed", reviewErr.Error())
			} else {
				transition = reviewTransition
			}
			break
		}
		if sagaTransition, handled, sagaErr := i.settleSagaWork(
			frame,
			execution,
			program,
			state,
			event.WorkItemID,
		); handled {
			if sagaErr != nil {
				transition, err = workflowFailure(
					state,
					"workflow_compensation_failed",
					sagaErr.Error(),
				)
			} else {
				transition = sagaTransition
			}
			break
		}
		item := executionWorkItem(execution, event.WorkItemID)
		handled, resolverErr := i.settleTargetSetResolver(
			&state,
			program,
			execution,
			frame.ID,
			event.WorkItemID,
		)
		if resolverErr != nil {
			transition, err = workflowFailure(
				state,
				"workflow_target_set_resolution_failed",
				resolverErr.Error(),
			)
			break
		}
		if handled {
			transition, err = i.resumeResolvedTargetSetCall(
				frame,
				execution,
				program,
				state,
			)
			break
		}
		if item != nil &&
			item.FrameID == frame.ID &&
			item.State == WorkItemStateReconciling &&
			state.PendingPath != "" &&
			item.ProgramPath == state.PendingPath {
			reconcileTransition, handled, reconcileErr := i.beginReconciliation(
				frame,
				execution,
				program,
				state,
				*item,
			)
			if handled {
				transition = reconcileTransition
				err = reconcileErr
				break
			}
		}
		if item != nil && item.FrameID == frame.ID &&
			state.PendingPath != "" && item.ProgramPath == state.PendingPath {
			if failure := workflowSettledFailure(program, state, *item); failure != nil {
				transition, err = workflowFailureTransition(
					state,
					failure,
				)
				break
			}
		}
		handled, settleErr := i.settleMachineWork(
			&state,
			program,
			execution,
			frame.ID,
			event.WorkItemID,
		)
		if settleErr != nil {
			transition, err = workflowFailure(
				state,
				"workflow_machine_failed",
				settleErr.Error(),
			)
			break
		}
		if !handled {
			if settleErr := i.settleWork(
				&state,
				program,
				execution,
				frame.ID,
				event.WorkItemID,
			); settleErr != nil {
				transition, err = workflowFailure(
					state,
					"workflow_work_failed",
					settleErr.Error(),
				)
				break
			}
		}
		transition, err = i.drive(frame, execution, program, state)

	case ProgramEventChildSettled:
		if reconciliationTransition, handled, settleErr := i.settleReconciliationChild(
			frame,
			execution,
			program,
			state,
			event.ChildFrameID,
		); handled {
			if settleErr != nil {
				transition, err = workflowFailure(
					state,
					"workflow_reconciliation_failed",
					settleErr.Error(),
				)
			} else {
				transition = reconciliationTransition
			}
			break
		}
		if sagaTransition, handled, sagaErr := i.settleSagaChild(
			frame,
			execution,
			program,
			state,
			event.ChildFrameID,
		); handled {
			if sagaErr != nil {
				transition, err = workflowFailure(state, "workflow_compensation_failed", sagaErr.Error())
			} else {
				transition = sagaTransition
			}
			break
		}
		err = i.settleChild(
			&state,
			program,
			execution,
			frame.ID,
			event.ChildFrameID,
		)
		if err != nil {
			if failure, ok := errors.AsType[workflowChildError](err); ok {
				transition, err = workflowFailureTransition(
					state,
					&failure.errorValue,
				)
			} else {
				transition, err = workflowFailure(
					state,
					"workflow_child_failed",
					err.Error(),
				)
			}
		} else {
			transition, err = i.drive(frame, execution, program, state)
		}

	case ProgramEventStarted,
		ProgramEventContinue,
		ProgramEventActivity:
		transition, err = i.drive(frame, execution, program, state)
	default:
		transition, err = i.drive(frame, execution, program, state)
	}

	if err != nil {
		return ProgramTransition{}, err
	}
	transition, err = i.recordTerminalWorkflowStepOutcome(
		program,
		state,
		transition,
		execution,
		frame,
		event,
	)
	if err != nil {
		return ProgramTransition{}, err
	}
	return i.prepareWorkflowTerminal(
		frame,
		execution,
		program,
		transition,
	)
}

func (i *WorkflowInterpreter) resumeResolvedTargetSetCall(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
) (ProgramTransition, error) {
	if state.PendingPath == "" {
		return i.drive(frame, execution, program, state)
	}
	step, err := workflowStepForPath(program.Root, state.PendingPath)
	if err != nil {
		return ProgramTransition{}, err
	}
	if step.Call == nil {
		return ProgramTransition{}, errors.New(
			"resolved workflow target set does not belong to a call step",
		)
	}
	return i.call(
		frame,
		program,
		state,
		state.PendingPath,
		step,
		*step.Call,
	)
}

func (i *WorkflowInterpreter) drive(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
) (ProgramTransition, error) {
	if err := captureWorkflowMessages(program, &state, execution); err != nil {
		return workflowFailure(
			state,
			"workflow_messages_capture_failed",
			err.Error(),
		)
	}
	if state.CancellationPending && state.Pending == nil {
		status := ProgramFrameStateCancelled
		transition, err := workflowTransition(
			state,
			&status,
			nil,
			nil,
		)
		if err != nil {
			return ProgramTransition{}, err
		}
		transition.Error = workflowTerminalFailure(status, nil)
		return transition, nil
	}
	if state.Pending != nil {
		return i.drivePending(frame, execution, program, state)
	}

	for {
		if state.Exit != "" {
			return i.completeWorkflowBlock(
				program,
				state,
				state.ExitValue,
				state.Exit,
			)
		}
		if state.PendingPath != "" {
			return workflowTransition(state, nil, nil, nil)
		}

		if state.Index >= len(program.Root.Steps) {
			if err := captureWorkflowMessages(program, &state, execution); err != nil {
				return workflowFailure(
					state,
					"workflow_messages_capture_failed",
					err.Error(),
				)
			}
			output, err := i.evaluate(
				program.Root.Output,
				program,
				state,
			)
			if err != nil {
				return workflowFailure(
					state,
					"workflow_output_failed",
					err.Error(),
				)
			}
			if err := validateWorkflowSchema(
				program.OutputSchema,
				output,
			); err != nil {
				return workflowFailure(
					state,
					"workflow_output_invalid",
					err.Error(),
				)
			}

			return i.completeWorkflowBlock(program, state, output, WorkflowExitNormal)
		}

		step := program.Root.Steps[state.Index]
		path := "step/" + step.ID
		if strings.TrimSpace(step.DeclarationPath) == "" {
			return workflowFailure(
				state,
				"workflow_declaration_path_missing",
				"workflow step has no prepared declaration path",
			)
		}
		state.PendingPath = path
		state.PendingDeclarationPath = step.DeclarationPath
		if err := captureWorkflowMessages(program, &state, execution); err != nil {
			return workflowFailure(
				state,
				"workflow_messages_capture_failed",
				err.Error(),
			)
		}

		if step.When != nil {
			allowed, err := i.condition(
				*step.When,
				program,
				state,
			)
			if err != nil {
				return workflowFailure(
					state,
					"workflow_condition_failed",
					err.Error(),
				)
			}
			if !allowed {
				if err := state.recordOutcome(WorkflowStepOutcome{
					ID:              step.ID,
					Status:          WorkflowStepOutcomeSkipped,
					DeclarationPath: step.DeclarationPath,
				}); err != nil {
					return workflowFailure(
						state,
						"workflow_state_invalid",
						err.Error(),
					)
				}
				state.Index++
				state.clearPendingStep()
				continue
			}
		}

		state.Deadline = cloneTime(frame.Deadline)
		if step.Timeout != "" {
			deadline, err := step.Timeout.Deadline(state.LogicalTime)
			if err != nil {
				return workflowFailure(state, "workflow_timeout_invalid", err.Error())
			}
			state.Deadline = earlierDeadline(state.Deadline, &deadline)
		}

		if state.Exit != "" {
			return i.completeWorkflowBlock(
				program,
				state,
				state.ExitValue,
				state.Exit,
			)
		}

		if step.Retry != nil && step.Retry.MaxAttempts > 1 {
			return i.startWorkflowRetry(
				frame,
				program,
				state,
				step,
				path,
			)
		}

		switch {
		case step.Call != nil:
			return i.call(
				frame,
				program,
				state,
				path,
				step,
				*step.Call,
			)

		case step.Transform != nil:
			return i.transform(
				program,
				state,
				path,
				step,
			)

		case step.Set != nil:
			return i.set(
				frame,
				program,
				state,
				path,
				step,
				*step.Set,
			)

		case step.Wait != nil:
			return i.wait(frame, program, state, path, *step.Wait)

		case step.Emit != nil:
			return i.emit(program, state, path, *step.Emit)

		case step.Scope != nil:
			return i.startScope(
				frame,
				program,
				state,
				step,
				path,
			)

		case step.Choose != nil:
			return i.startChoose(
				frame,
				program,
				state,
				step,
				path,
			)

		case step.Parallel != nil:
			return i.startParallel(
				frame,
				program,
				state,
				step,
				path,
			)

		case step.Foreach != nil:
			return i.startForeach(
				frame,
				program,
				state,
				step,
				path,
			)

		case step.Repeat != nil:
			return i.startRepeat(
				frame,
				program,
				state,
				step,
				path,
			)

		case step.Machine != nil:
			return i.startMachine(
				frame,
				execution,
				program,
				state,
				step,
				path,
			)

		case step.Saga != nil:
			return i.startSaga(
				frame,
				program,
				state,
				step,
				path,
			)

		case step.Return != nil:
			output, err := i.evaluate(
				step.Return.Value,
				program,
				state,
			)
			if err != nil {
				return workflowFailure(
					state,
					"workflow_return_failed",
					err.Error(),
				)
			}
			if err := recordInlineWorkflowStepOutcome(
				&state,
				step,
				WorkflowStepOutcomeSucceeded,
				output,
				nil,
			); err != nil {
				return workflowFailure(state, "workflow_state_invalid", err.Error())
			}
			state.Index++
			state.clearPendingStep()

			exit := WorkflowExitReturnBlock
			if step.Return.To == WorkflowReturnDestinationWorkflow {
				exit = WorkflowExitReturnWorkflow
			}
			return i.completeWorkflowBlock(program, state, output, exit)

		case step.Cancel != nil:
			return i.cancelCall(
				program,
				state,
				path,
				step,
				*step.Cancel,
			)

		case step.Fail != nil:
			message, err := i.stringValue(
				step.Fail.Message,
				program,
				state,
			)
			if err != nil {
				return workflowFailure(
					state,
					"workflow_fail_invalid",
					err.Error(),
				)
			}
			failure := &SanitizedError{
				Code:      step.Fail.Code,
				Category:  step.Fail.Category,
				Message:   message,
				Retryable: step.Fail.Retryable,
				StepPath:  state.PendingDeclarationPath,
			}
			if step.Fail.Details != nil {
				details, err := i.evaluate(*step.Fail.Details, program, state)
				if err != nil {
					return workflowFailure(state, "workflow_fail_invalid", err.Error())
				}
				failure.Details = details
			}
			if err := recordInlineWorkflowStepOutcome(
				&state,
				step,
				WorkflowStepOutcomeFailed,
				nil,
				failure,
			); err != nil {
				return workflowFailure(state, "workflow_state_invalid", err.Error())
			}
			state.Index++
			state.clearPendingStep()
			return workflowFailureTransition(state, failure)

		case step.Break:
			value := jsonutil.MustCanonicalValue(nil)
			if err := recordInlineWorkflowStepOutcome(
				&state,
				step,
				WorkflowStepOutcomeSucceeded,
				value,
				nil,
			); err != nil {
				return workflowFailure(state, "workflow_state_invalid", err.Error())
			}
			state.Index++
			state.clearPendingStep()
			return i.completeWorkflowBlock(
				program,
				state,
				value,
				WorkflowExitBreak,
			)

		case step.Continue:
			value := jsonutil.MustCanonicalValue(nil)
			if err := recordInlineWorkflowStepOutcome(
				&state,
				step,
				WorkflowStepOutcomeSucceeded,
				value,
				nil,
			); err != nil {
				return workflowFailure(state, "workflow_state_invalid", err.Error())
			}
			state.Index++
			state.clearPendingStep()
			return i.completeWorkflowBlock(
				program,
				state,
				value,
				WorkflowExitContinue,
			)

		default:
			return workflowFailure(
				state,
				"workflow_operation_unsupported",
				"Workflow program contains an unsupported operation",
			)
		}
	}
}

func (i *WorkflowInterpreter) call(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	path string,
	step WorkflowStep,
	call WorkflowCall,
) (ProgramTransition, error) {
	if transition, waiting, err := i.beginTargetSetResolution(
		frame,
		program,
		&state,
		path,
		call,
	); err != nil {
		return workflowFailure(state, "workflow_target_set_invalid", err.Error())
	} else if waiting {
		return transition, nil
	}

	targetName, err := i.stringValue(call.Target, program, state)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_target_invalid",
			err.Error(),
		)
	}
	allowedTargets, allowedErr := i.resolveWorkflowCallTargets(
		program,
		&state,
		call,
	)
	if allowedErr != nil {
		return workflowFailure(
			state,
			"workflow_target_set_invalid",
			allowedErr.Error(),
		)
	}
	if len(allowedTargets) != 0 &&
		!slices.Contains(allowedTargets, targetName) {
		return workflowFailure(
			state,
			"workflow_target_denied",
			"Workflow target is outside its allowed target set",
		)
	}

	target, found := program.Targets[targetName]
	if !found {
		return workflowFailure(
			state,
			"workflow_target_unavailable",
			"workflow target is unavailable",
		)
	}

	input, err := i.evaluate(call.Input, program, state)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_input_invalid",
			err.Error(),
		)
	}
	contextValue, err := i.evaluate(call.Context, program, state)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_context_invalid",
			err.Error(),
		)
	}
	setContract := call.SetContract
	if effective := workflowTargetSetContract(program, call); effective != nil {
		setContract = effective
	}
	if err := validateWorkflowCallContext(
		target.Contract,
		call.Contract,
		setContract,
	)(contextValue); err != nil {
		return workflowFailure(
			state,
			"workflow_context_invalid",
			err.Error(),
		)
	}
	ports, err := i.workflowCallPortPlans(
		frame,
		program,
		state,
		path,
		call,
		target,
	)
	if err != nil {
		return workflowFailure(state, "workflow_port_invalid", err.Error())
	}
	if err := validateWorkflowInterfaceInput(
		target.Contract,
		input,
	); err != nil {
		return workflowFailure(state, "workflow_target_input_invalid", err.Error())
	}
	if err := validateWorkflowInterfaceInput(call.Contract, input); err != nil {
		return workflowFailure(state, "workflow_call_input_invalid", err.Error())
	}
	if err := validateWorkflowInterfaceInput(setContract, input); err != nil {
		return workflowFailure(state, "workflow_target_set_input_invalid", err.Error())
	}

	effect, err := i.workflowCallEffect(
		program,
		state,
		step,
		target,
	)
	if err != nil {
		return workflowFailure(state, "workflow_effect_invalid", err.Error())
	}
	callBudget, err := workflowCallableBudget(
		target.Contract,
		call.Contract,
		setContract,
	)
	if err != nil {
		return workflowFailure(state, "workflow_limit_invalid", err.Error())
	}
	childBudget, err := workflowChildCallBudget(
		callBudget,
		step.Limits,
	)
	if err != nil {
		return workflowFailure(state, "workflow_limit_invalid", err.Error())
	}

	mode := call.Mode
	if mode == "" {
		mode = WorkflowCallModeInvoke
	}
	for _, contract := range []*WorkflowCallableInterface{target.Contract, call.Contract, call.SetContract} {
		if contract != nil && len(contract.Modes) != 0 && !slices.Contains(contract.Modes, mode) {
			if !call.InternalLifecycleChild {
				return workflowFailure(
					state,
					"workflow_call_mode_invalid",
					"target interface does not permit the requested call mode",
				)
			}
		}
	}
	if mode == WorkflowCallModeStart {
		return i.startCall(
			frame,
			program,
			state,
			path,
			step,
			targetName,
			target,
			input,
			effect,
			contextValue,
			call,
			ports,
			childBudget,
		)
	}
	if mode == WorkflowCallModeHandoff {
		return i.handoffWorkflowAgentTarget(
			frame,
			program,
			state,
			path,
			step,
			targetName,
			target,
			call,
			input,
			contextValue,
			effect,
			ports,
			childBudget,
		)
	}
	state.PendingTarget = targetName

	switch target.Kind {
	case WorkflowTargetTool:
		if target.Artifact == nil {
			return workflowFailure(
				state,
				"workflow_target_invalid",
				"Workflow Tool target has no Artifact pin",
			)
		}
		return i.workflowStepWork(
			program,
			state,
			path,
			WorkItemIntent{
				ProgramPath: path,
				Type:        WorkItemTypeTool,
				Automatic:   true,
				Effect:      effect,
				Budget:      cloneBudget(callBudget),
				Ports:       clonePortPlans(ports),
				Input: WorkItemPayload{
					Tool: &ToolWork{
						Call: ToolCall{
							ID:        string(frame.ID) + ":" + path,
							Name:      target.ToolName,
							Arguments: input,
							Binding: ToolBinding{
								Kind: ToolBindingKindUnresolved,
								Unresolved: &UnresolvedToolBinding{
									Reason: "Workflow Tool target requires runtime binding",
								},
							},
						},
						Target: target.Artifact,
					},
				},
			},
		)

	case WorkflowTargetMCPTool:
		if target.MCP == nil {
			return workflowFailure(
				state,
				"workflow_target_invalid",
				"Workflow MCP Tool target has no MCP binding",
			)
		}
		return i.workflowStepWork(
			program,
			state,
			path,
			WorkItemIntent{
				ProgramPath: path,
				Type:        WorkItemTypeTool,
				Automatic:   true,
				Effect:      effect,
				Budget:      cloneBudget(callBudget),
				Ports:       clonePortPlans(ports),
				Input: WorkItemPayload{
					Tool: &ToolWork{
						Call: ToolCall{
							ID:        string(frame.ID) + ":" + path,
							Name:      target.MCP.ToolName,
							Arguments: input,
							Binding: ToolBinding{
								Kind: ToolBindingKindUnresolved,
								Unresolved: &UnresolvedToolBinding{
									Reason: "Workflow MCP Tool target requires runtime binding",
								},
							},
						},
						MCP: &MCPToolTarget{
							Server:           target.MCP.Server,
							ToolName:         target.MCP.ToolName,
							ProviderToolName: target.MCP.ProviderToolName,
							ChoiceID:         target.MCP.ChoiceID,
							ToolDigest:       target.MCP.ToolDigest,
							ApprovalRule:     target.MCP.ApprovalRule,
							ExecutionMode:    target.MCP.ExecutionMode,
							DiscoveryDigest:  target.MCP.DiscoveryDigest,
						},
					},
				},
			},
		)

	case WorkflowTargetService:
		return i.workflowStepWork(
			program,
			state,
			path,
			WorkItemIntent{
				ProgramPath: path,
				Type:        WorkItemTypeService,
				Automatic:   true,
				Effect:      effect,
				Budget:      cloneBudget(callBudget),
				Ports:       clonePortPlans(ports),
				Input: WorkItemPayload{
					Service: &ServiceWork{
						ServiceID:      target.ServiceID,
						BindingVersion: target.BindingVersion,
						Input:          input,
						Context:        workflowExternalContext(contextValue),
					},
				},
			},
		)

	case WorkflowTargetActivity:
		return i.workflowStepWork(
			program,
			state,
			path,
			WorkItemIntent{
				ProgramPath: path,
				Type:        WorkItemTypeActivity,
				Automatic:   true,
				Effect:      effect,
				Budget:      cloneBudget(callBudget),
				Ports:       clonePortPlans(ports),
				Input: WorkItemPayload{
					Activity: &ActivityWork{
						ActivityID:     target.ActivityID,
						BindingVersion: target.BindingVersion,
						Input:          input,
						Context:        workflowExternalContext(contextValue),
					},
				},
			},
		)

	case WorkflowTargetWorkflow:
		if target.NestedProgram == nil {
			return workflowFailure(
				state,
				"workflow_target_invalid",
				"Nested Workflow program is unavailable",
			)
		}

		nested, err := workflowSnapshotWithInput(
			*target.NestedProgram,
			input,
		)
		if err != nil {
			return workflowFailure(
				state,
				"workflow_target_invalid",
				err.Error(),
			)
		}
		nested, err = workflowSnapshotWithContext(
			nested,
			contextValue,
		)
		if err != nil {
			return workflowFailure(
				state,
				"workflow_context_invalid",
				err.Error(),
			)
		}
		_, claimErr := i.workflowResourceClaims(
			program,
			state,
			step,
		)
		if claimErr != nil {
			return workflowFailure(
				state,
				"workflow_resource_claim_invalid",
				claimErr.Error(),
			)
		}

		return i.startWorkflowChildCall(
			frame,
			program,
			state,
			path,
			step,
			targetName,
			effect,
			input,
			contextValue,
			nested,
			ports,
			childBudget,
			ProgramFramePortBoundaryPublishedWorkflow,
			nil,
		)

	case WorkflowTargetAgent,
		WorkflowTargetModel:
		return i.invokeWorkflowLLMTarget(
			frame,
			state,
			program,
			path,
			step,
			targetName,
			target,
			call,
			input,
			contextValue,
			effect,
			nil,
			ports,
			childBudget,
		)

	case WorkflowTargetProviderTool:
		if target.ProviderTool == nil {
			return workflowFailure(
				state,
				"workflow_target_invalid",
				"provider-managed Tool target has no typed Model plan",
			)
		}
		if call.LLMInvocation == nil {
			return workflowFailure(
				state,
				"workflow_invocation_input_invalid",
				"provider-managed Tool target requires a typed Model invocation",
			)
		}
		modelTarget := WorkflowTarget{
			Kind:      WorkflowTargetModel,
			Artifact:  &target.ProviderTool.OwnerModel,
			Transport: target.Transport,
			Effect:    EffectContract{Kind: EffectKindRead},
		}
		invocation, invokeErr := i.prepareWorkflowLLMInvocation(
			frame,
			path,
			modelTarget,
			*call.LLMInvocation,
			program,
			state,
		)
		if invokeErr != nil {
			return workflowFailure(
				state,
				"workflow_invocation_input_invalid",
				invokeErr.Error(),
			)
		}
		choice, choiceErr := configuredProviderManagedToolChoice(
			target.ProviderTool.Choice,
			invocation.Defaults,
		)
		if choiceErr != nil {
			return workflowFailure(
				state,
				"workflow_provider_tool_configuration_invalid",
				choiceErr.Error(),
			)
		}
		invocation.Defaults = withoutProviderManagedToolSelection(
			invocation.Defaults,
			choice.Artifact,
		)
		child, childErr := NewSingleModelWorkflowInvocationProgram(
			invocation.Defaults,
			invocation.InputActivityID,
			invocation.HistoryActivityIDs,
			[]ProviderManagedToolChoice{choice},
		)
		if childErr != nil {
			return workflowFailure(
				state,
				"workflow_invocation_prepare_failed",
				childErr.Error(),
			)
		}
		transition, transitionErr := i.startWorkflowChildCall(
			frame,
			program,
			state,
			path,
			step,
			targetName,
			effect,
			input,
			contextValue,
			child,
			ports,
			childBudget,
			ProgramFramePortBoundaryNone,
			nil,
		)
		if transitionErr != nil {
			return ProgramTransition{}, transitionErr
		}
		transition.Activities = append(
			[]Activity(nil),
			invocation.Activities...,
		)
		return transition, nil

	default:
		return workflowFailure(
			state,
			"workflow_target_invalid",
			"Workflow target kind is invalid",
		)
	}
}

func (i *WorkflowInterpreter) workflowCallEffect(
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	target WorkflowTarget,
) (EffectContract, error) {
	effect := target.Effect
	if step.Effect == nil {
		if effect.Kind == EffectKindKeyed &&
			strings.TrimSpace(effect.IdempotencyKey) == "" ||
			effect.IdempotencyKey == "workflow-key-pending" {
			return EffectContract{}, errors.New(
				"keyed target requires a workflow operation key",
			)
		}
		return effect, nil
	}

	requested := step.Effect.Kind
	switch requested {
	case EffectKindRead:
		if target.Effect.Kind != EffectKindRead {
			return EffectContract{}, errors.New(
				"workflow call cannot strengthen a mutating target to read",
			)
		}

	case EffectKindIdempotent:
		if target.Effect.Kind != EffectKindRead &&
			target.Effect.Kind != EffectKindIdempotent {
			return EffectContract{}, errors.New(
				"workflow call exceeds the target idempotency guarantee",
			)
		}

	case EffectKindKeyed:
		switch target.Effect.Kind {
		case EffectKindRead, EffectKindIdempotent, EffectKindKeyed:
		default:
			return EffectContract{}, errors.New(
				"workflow call requests keying from a target without a keying guarantee",
			)
		}
		if step.Effect.Key == nil {
			return EffectContract{}, errors.New(
				"keyed workflow call has no operation key",
			)
		}
		key, err := i.stringValue(
			*step.Effect.Key,
			program,
			state,
		)
		if err != nil {
			return EffectContract{}, fmt.Errorf(
				"evaluate workflow operation key: %w",
				err,
			)
		}
		if strings.TrimSpace(key) == "" || strings.TrimSpace(key) != key {
			return EffectContract{}, errors.New(
				"workflow operation key must be a non-empty normalized string",
			)
		}
		effect.IdempotencyKey = key

	case EffectKindNonIdempotent:
		effect.IdempotencyKey = ""

	default:
		return EffectContract{}, fmt.Errorf(
			"unsupported workflow effect kind %q",
			requested,
		)
	}
	if step.Effect != nil && step.Effect.Reconcile != nil {
		if !target.Effect.Reconcile {
			return EffectContract{}, errors.New(
				"workflow reconciliation requires target reconciliation support",
			)
		}
		effect.Reconcile = true
	}

	effect.Kind = requested
	return effect, nil
}

func (i *WorkflowInterpreter) transform(
	program WorkflowProgram,
	state WorkflowState,
	path string,
	step WorkflowStep,
) (ProgramTransition, error) {
	value, err := i.evaluate(
		step.Transform.Value,
		program,
		state,
	)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_transform_failed",
			err.Error(),
		)
	}

	return i.workflowStepWork(
		program,
		state,
		path,
		WorkItemIntent{
			ProgramPath: path,
			Type:        WorkItemTypeTransform,
			Automatic:   true,
			Effect: EffectContract{
				Kind: EffectKindRead,
			},
			Input: WorkItemPayload{
				Transform: &TransformWork{
					Input:        value,
					OutputSchema: step.Transform.OutputSchema.Clone(),
				},
			},
		},
	)
}

func (i *WorkflowInterpreter) set(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	path string,
	step WorkflowStep,
	set WorkflowSet,
) (ProgramTransition, error) {
	owner, err := workflowStateOwnerForFrame(frame, program)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_state_unavailable",
			err.Error(),
		)
	}
	if owner == nil {
		return workflowFailure(
			state,
			"workflow_state_unavailable",
			"workflow set operation has no state-owning scope",
		)
	}

	assignments := make(
		[]StateAssignment,
		0,
		len(set.Values),
	)
	for _, assignment := range set.Values {
		value, evaluateErr := i.evaluate(
			assignment.Value,
			program,
			state,
		)
		if evaluateErr != nil {
			return workflowFailure(
				state,
				"workflow_state_update_invalid",
				evaluateErr.Error(),
			)
		}
		assignments = append(assignments, StateAssignment{
			Path:  assignment.Key,
			Value: value,
		})
	}

	return i.workflowStepWork(
		program,
		state,
		path,
		WorkItemIntent{
			ProgramPath: path,
			Type:        WorkItemTypeStateUpdate,
			Automatic:   true,
			Effect: EffectContract{
				Kind: EffectKindRead,
			},
			Input: WorkItemPayload{
				StateUpdate: &StateUpdateWork{
					StateOwnerID: owner.ID,
					Base:         state.State.Clone(),
					Schema:       owner.Schema.Clone(),
					Assignments:  assignments,
					RemovePaths:  append([]string(nil), set.Remove...),
				},
			},
		},
	)
}

func (i *WorkflowInterpreter) settleWork(
	state *WorkflowState,
	program WorkflowProgram,
	execution Execution,
	frameID ProgramFrameID,
	workID WorkItemID,
) error {
	if state == nil {
		return errors.New("Workflow state is unavailable")
	}

	item := executionWorkItem(execution, workID)
	if item == nil {
		return errors.New("Workflow WorkItem is unavailable")
	}
	if item.FrameID != frameID {
		return nil
	}
	if state.PendingPath == "" || item.ProgramPath != state.PendingPath {
		return nil
	}
	if item.State != WorkItemStateCompleted {
		return fmt.Errorf("Workflow WorkItem %q did not complete successfully", workID)
	}
	step, err := workflowStepForPath(program.Root, item.ProgramPath)
	if err != nil {
		return err
	}
	if err := validateWorkflowCallReceipt(program, *state, step, item.Output); err != nil {
		return err
	}
	if item.Output != nil &&
		item.Output.Tool != nil &&
		item.Output.Tool.Kind != ToolOutputKindSuccess {
		return fmt.Errorf(
			"Workflow Tool WorkItem %q returned an error result",
			workID,
		)
	}

	if item.EffectState == EffectStateCompensationFailed {
		return errors.New("workflow work item compensation failed")
	}

	value, err := workflowWorkValue(*item)
	if err != nil {
		return err
	}
	switch item.Type {
	case WorkItemTypeStateUpdate:
		if item.Input.StateUpdate == nil {
			return errors.New("workflow state update work item has no payload")
		}
		frame := mutableFrame(&execution, frameID)
		if frame == nil {
			return errors.New("workflow state update owner frame is unavailable")
		}
		owner, ownerErr := workflowStateOwnerForFrame(*frame, program)
		if ownerErr != nil {
			return ownerErr
		}
		if owner == nil {
			return errors.New("workflow state update has no state owner")
		}
		if item.Input.StateUpdate.StateOwnerID != owner.ID ||
			!canonicalValuesEqual(item.Input.StateUpdate.Schema, owner.Schema) {
			return errors.New(
				"workflow state update differs from its nearest state owner",
			)
		}
		state.State = value.Clone()
	default:
	}

	if err := i.validateStepOutput(program, *state, step, value); err != nil {
		return err
	}
	selectedTarget := state.PendingTarget
	outcome := WorkflowStepOutcome{
		ID:              step.ID,
		Status:          WorkflowStepOutcomeSucceeded,
		Output:          value,
		Target:          selectedTarget,
		Effect:          item.EffectState,
		DeclarationPath: state.PendingDeclarationPath,
	}
	if item.Output != nil {
		outcome.Receipt = cloneReconciliationReceipt(item.Output.Receipt)
	}
	if err := state.recordOutcome(outcome); err != nil {
		return err
	}
	if item.Effect.Kind != EffectKindRead &&
		item.EffectState == EffectStateConfirmedApplied {
		setWorkflowCausalEffectWorkItemIDs(
			state,
			[]WorkItemID{item.ID},
		)
	}
	if item.Type == WorkItemTypeChildCall &&
		item.Input.ChildCall != nil &&
		item.Input.ChildCall.Completion.FrameID != "" {
		if child := mutableFrame(&execution, item.Input.ChildCall.Completion.FrameID); child != nil {
			setWorkflowCausalEffectWorkItemIDs(state, mergeWorkflowCausalEffectWorkItemIDs(
				workflowCausalEffectWorkItemIDs(*state),
				workflowFrameCausalEffectWorkItemIDs(*child),
			))
		}
	}
	state.Index++
	state.clearPendingStep()
	return nil
}

func (i *WorkflowInterpreter) settleChild(
	state *WorkflowState,
	program WorkflowProgram,
	execution Execution,
	frameID ProgramFrameID,
	childID ProgramFrameID,
) error {
	if state == nil {
		return errors.New("Workflow state is unavailable")
	}
	if state.ChildCallWorkItemID != "" {
		// The durable child-call WorkItem is the authoritative outer call
		// boundary. Its completion wait settles the parent step with the
		// child output and effect facts.
		return nil
	}
	if state.Pending != nil {
		return i.settleStructuredChild(
			state,
			program,
			execution,
			frameID,
			childID,
		)
	}

	for _, child := range execution.ProgramFrames {
		if child.ID != childID || child.ParentFrameID != frameID {
			continue
		}
		if child.Path != state.PendingPath {
			return nil
		}
		if child.Status != ProgramFrameStateCompleted {
			return errors.New("nested workflow ProgramFrame did not complete")
		}
		step, err := workflowStepForPath(program.Root, child.Path)
		if err != nil {
			return err
		}
		if err := i.validateStepOutput(
			program,
			*state,
			step,
			child.Output,
		); err != nil {
			return err
		}
		if err := validateWorkflowCallReceiptForTarget(
			program,
			state.PendingTarget,
			step,
			child.Receipt,
		); err != nil {
			return err
		}
		selectedTarget := state.PendingTarget
		outcome := WorkflowStepOutcome{
			ID:              step.ID,
			Status:          WorkflowStepOutcomeSucceeded,
			Output:          child.Output,
			Target:          selectedTarget,
			Receipt:         cloneReconciliationReceipt(child.Receipt),
			DeclarationPath: state.PendingDeclarationPath,
		}
		if child.Receipt != nil {
			outcome.Effect = child.Receipt.EffectState
		}
		if err := state.recordOutcome(outcome); err != nil {
			return err
		}
		state.Index++
		state.clearPendingStep()
		return nil
	}
	return errors.New("nested workflow ProgramFrame is unavailable")
}

func workflowStepForPath(
	block WorkflowBlock,
	path string,
) (WorkflowStep, error) {
	stepID, found := strings.CutPrefix(path, "step/")
	if !found || strings.TrimSpace(stepID) == "" {
		return WorkflowStep{}, errors.New("Workflow WorkItem has an invalid program path")
	}
	for _, step := range block.Steps {
		if step.ID == stepID {
			return step, nil
		}
	}
	return WorkflowStep{}, fmt.Errorf("Workflow step %q is unavailable", stepID)
}

func (i *WorkflowInterpreter) validateStepOutput(
	program WorkflowProgram,
	state WorkflowState,
	step WorkflowStep,
	value jsonutil.CanonicalValue,
) error {
	if err := validateWorkflowSchema(step.OutputSchema, value); err != nil {
		return fmt.Errorf("validate Workflow step output: %w", err)
	}
	if step.Call == nil {
		return nil
	}

	targetName := state.PendingTarget
	if targetName == "" {
		return errors.New("workflow call has no persisted selected target")
	}
	target, found := program.Targets[targetName]
	if !found {
		return fmt.Errorf("Workflow target %q is unavailable", targetName)
	}
	if err := validateWorkflowInterfaceOutput(target.Contract, value); err != nil {
		return fmt.Errorf("validate Workflow target output: %w", err)
	}
	if err := validateWorkflowInterfaceOutput(step.Call.Contract, value); err != nil {
		return fmt.Errorf("validate Workflow call output: %w", err)
	}
	setContract := step.Call.SetContract
	if effective := workflowTargetSetContract(program, *step.Call); effective != nil {
		setContract = effective
	}
	if err := validateWorkflowInterfaceOutput(setContract, value); err != nil {
		return fmt.Errorf("validate workflow target set output: %w", err)
	}
	return nil
}

func validateWorkflowInterfaceInput(
	contract *WorkflowCallableInterface,
	value jsonutil.CanonicalValue,
) error {
	if contract == nil {
		return nil
	}
	return validateWorkflowSchema(contract.InputSchema, value)
}

func validateWorkflowInterfaceOutput(
	contract *WorkflowCallableInterface,
	value jsonutil.CanonicalValue,
) error {
	if contract == nil {
		return nil
	}
	return validateWorkflowSchema(contract.OutputSchema, value)
}

func workflowCallableBudget(
	contracts ...*WorkflowCallableInterface,
) (*ExecutionBudget, error) {
	var output *ExecutionBudget
	for _, contract := range contracts {
		if contract == nil || contract.Limits == nil {
			continue
		}

		if output == nil {
			value := cloneExecutionBudget(*contract.Limits)
			output = &value
			continue
		}

		value, err := narrowExecutionBudget(
			*output,
			*contract.Limits,
		)
		if err != nil {
			return nil, err
		}
		output = &value
	}
	return output, nil
}

func workflowChildCallBudget(
	callBudget *ExecutionBudget,
	stepBudget *ExecutionBudget,
) (*ExecutionBudget, error) {
	switch {
	case callBudget == nil:
		return cloneBudget(stepBudget), nil
	case stepBudget == nil:
		return cloneBudget(callBudget), nil
	}

	value, err := narrowExecutionBudget(
		*callBudget,
		*stepBudget,
	)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (i *WorkflowInterpreter) startWorkflowChildCall(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	path string,
	step WorkflowStep,
	targetName string,
	effect EffectContract,
	input jsonutil.CanonicalValue,
	contextValue jsonutil.CanonicalValue,
	child ProgramSnapshot,
	ports []PortPlan,
	budget *ExecutionBudget,
	portBoundary ProgramFramePortBoundary,
	handoff *ConversationHandoffIntent,
) (ProgramTransition, error) {
	claims, err := i.workflowResourceClaims(
		program,
		state,
		step,
	)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_resource_claim_invalid",
			err.Error(),
		)
	}

	childFrameID := workflowChildCallFrameID(frame.ID, path)
	completion := CompletionHandle{FrameID: childFrameID}
	boundary, err := i.workflowChildCallBoundary(
		frame.ID,
		program,
		state,
		path,
		step,
		targetName,
		effect,
		input,
		contextValue,
		completion,
	)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_child_call_invalid",
			err.Error(),
		)
	}

	state.PendingPath = path
	state.PendingTarget = targetName
	state.ChildCallWorkItemID = boundary.ID
	transition, err := workflowTransition(
		state,
		nil,
		nil,
		[]WorkItemIntent{boundary},
	)
	if err != nil {
		return ProgramTransition{}, err
	}
	transition.Frames = []ProgramFrameIntent{{
		ID:              childFrameID,
		Path:            path,
		DeclarationPath: state.PendingDeclarationPath,
		Program:         child,
		Deadline:        cloneTime(state.Deadline),
		Budget:          cloneBudget(budget),
		Claims:          claims,
		Ports:           clonePortPlans(ports),
		PortBoundary:    portBoundary,
		Handoff:         handoff,
	}}
	return transition, nil
}

func (i *WorkflowInterpreter) workflowChildCallBoundary(
	frameID ProgramFrameID,
	program WorkflowProgram,
	state WorkflowState,
	path string,
	step WorkflowStep,
	targetName string,
	effect EffectContract,
	input jsonutil.CanonicalValue,
	contextValue jsonutil.CanonicalValue,
	completion CompletionHandle,
) (WorkItemIntent, error) {
	if err := completion.Validate(); err != nil {
		return WorkItemIntent{}, err
	}
	if err := validateID("workflow child call target", targetName); err != nil {
		return WorkItemIntent{}, err
	}

	intent := WorkItemIntent{
		ID:              workflowChildCallWorkID(frameID, path),
		ProgramPath:     path,
		DeclarationPath: state.PendingDeclarationPath,
		Type:            WorkItemTypeChildCall,
		Automatic:       true,
		Cleanup:         false,
		Effect:          effect,
		Input: WorkItemPayload{
			ChildCall: &ChildCallWork{
				Target:     targetName,
				Input:      input.Clone(),
				Context:    contextValue.Clone(),
				Completion: completion,
			},
		},
	}
	if err := i.applyWorkflowStepEffectPolicy(
		program,
		state,
		step,
		&intent,
	); err != nil {
		return WorkItemIntent{}, err
	}
	if err := i.applyWorkflowStepWorkBudget(step, &intent); err != nil {
		return WorkItemIntent{}, err
	}
	return intent, nil
}

func workflowChildCallFrameID(
	frameID ProgramFrameID,
	path string,
) ProgramFrameID {
	return ProgramFrameID(
		workflowChildCallFramePrefix + ":" + string(frameID) + ":" + path,
	)
}

func workflowChildCallWorkID(frameID ProgramFrameID,
	path string,
) WorkItemID {
	return WorkItemID(
		workflowChildCallWorkPrefix +
			workflowChildCallIDSeparator +
			string(frameID) +
			workflowChildCallIDSeparator +
			path,
	)
}

func (i *WorkflowInterpreter) evaluate(
	value WorkflowValue,
	program WorkflowProgram,
	state WorkflowState,
) (jsonutil.CanonicalValue, error) {
	switch value.Kind {
	case WorkflowValueLiteral:
		return value.Literal.Clone(), nil

	case WorkflowValueExpression:
		if i == nil || i.env == nil {
			return nil, errors.New("Workflow expression runtime is unavailable")
		}

		compiled, issues := i.env.Compile(value.Expression)
		if issues != nil && issues.Err() != nil {
			return nil, fmt.Errorf(
				"compile Workflow expression: %w",
				issues.Err(),
			)
		}
		executable, err := i.env.Program(compiled)
		if err != nil {
			return nil, err
		}
		activation, err := workflowActivation(program, state)
		if err != nil {
			return nil, err
		}
		result, _, err := executable.Eval(activation)
		if err != nil {
			return nil, fmt.Errorf(
				"evaluate Workflow expression: %w",
				err,
			)
		}
		return canonicalWorkflowCELValue(result)

	case WorkflowValueObject:
		output := make(
			map[string]jsonutil.CanonicalValue,
			len(value.Object),
		)
		for _, field := range value.Object {
			child, err := i.evaluate(field.Value, program, state)
			if err != nil {
				return nil, err
			}
			output[field.Key] = child
		}
		return canonicalWorkflowValue(output)

	case WorkflowValueArray:
		output := make(
			[]jsonutil.CanonicalValue,
			len(value.Array),
		)
		for index := range value.Array {
			child, err := i.evaluate(
				value.Array[index],
				program,
				state,
			)
			if err != nil {
				return nil, err
			}
			output[index] = child
		}
		return canonicalWorkflowValue(output)

	default:
		return nil, errors.New("workflow value has an unsupported kind")
	}
}

func (i *WorkflowInterpreter) condition(
	value WorkflowValue,
	program WorkflowProgram,
	state WorkflowState,
) (bool, error) {
	result, err := i.evaluate(value, program, state)
	if err != nil {
		return false, err
	}
	return jsonutil.DecodeCanonicalValue[bool](result)
}

func (i *WorkflowInterpreter) stringValue(
	value WorkflowValue,
	program WorkflowProgram,
	state WorkflowState,
) (string, error) {
	result, err := i.evaluate(value, program, state)
	if err != nil {
		return "", err
	}
	return jsonutil.DecodeCanonicalValue[string](result)
}

func (p WorkflowProgram) Validate() error {
	if p.Version != WorkflowInterpreterVersion {
		return errors.New("unsupported Workflow Program version")
	}
	if err := p.Scope.Validate(); err != nil {
		return err
	}
	if len(p.InputSchema) == 0 ||
		len(p.OutputSchema) == 0 ||
		!p.InputSchema.Valid() ||
		!p.OutputSchema.Valid() ||
		!p.Input.Valid() {
		return errors.New("workflow Program schemas or input are invalid")
	}
	if p.InputBound && len(p.Input) == 0 {
		return errors.New("bound Workflow Program input is missing")
	}
	if strings.TrimSpace(p.Root.DeclarationPath) == "" {
		return errors.New("workflow root has no declaration path")
	}
	for _, requirement := range p.Requirements {
		if err := requirement.Validate(); err != nil {
			return err
		}
	}
	for _, target := range p.Targets {
		if err := target.Validate(); err != nil {
			return err
		}
	}
	for name, target := range p.Targets {
		if strings.TrimSpace(name) == "" ||
			strings.TrimSpace(target.DeclarationPath) == "" {
			return errors.New("Workflow Program target name is empty")
		}
	}
	return p.Limits.Validate()
}

func (p WorkflowProgram) ValidateInput() error {
	if !p.InputBound {
		return errors.New("Workflow Program input has not been bound")
	}
	return validateWorkflowSchema(p.InputSchema, p.Input)
}

func (s WorkflowStep) Validate() error {
	operationCount := 0
	for _, present := range []bool{
		s.Call != nil,
		s.Transform != nil,
		s.Set != nil,
		s.Scope != nil,
		s.Saga != nil,
		s.Choose != nil,
		s.Parallel != nil,
		s.Foreach != nil,
		s.Repeat != nil,
		s.Machine != nil,
		s.Emit != nil,
		s.Return != nil,
		s.Cancel != nil,
		s.Fail != nil,
		s.Wait != nil,
		s.Break,
		s.Continue,
	} {
		if present {
			operationCount++
		}
	}
	if operationCount != 1 {
		return errors.New("workflow step must contain exactly one supported operation")
	}
	if err := validateWorkflowTimeout(s.Timeout); err != nil {
		return err
	}
	if s.When != nil {
		if err := s.When.Validate(); err != nil {
			return err
		}
	}
	if len(s.OutputSchema) != 0 && !s.OutputSchema.Valid() {
		return errors.New("workflow step output schema is invalid")
	}
	if s.Retry != nil {
		if err := s.Retry.Validate(); err != nil {
			return err
		}
	}
	if s.Limits != nil {
		if err := s.Limits.Validate(); err != nil {
			return err
		}
	}
	if s.Effect != nil {
		if s.Call == nil {
			return errors.New(
				"workflow effect is supported only for callable work",
			)
		}
		if err := s.Effect.Validate(); err != nil {
			return err
		}
	}
	for _, claim := range s.Claims {
		if err := claim.Validate(); err != nil {
			return err
		}
	}

	switch {
	case s.Call != nil:
		return s.Call.Validate()

	case s.Transform != nil:
		return s.Transform.Validate()

	case s.Set != nil:
		return s.Set.Validate()

	case s.Scope != nil:
		return s.Scope.Validate()

	case s.Saga != nil:
		return s.Saga.Validate()

	case s.Choose != nil:
		return s.Choose.Validate()

	case s.Parallel != nil:
		return s.Parallel.Validate()

	case s.Foreach != nil:
		return s.Foreach.Validate()

	case s.Repeat != nil:
		return s.Repeat.Validate()

	case s.Machine != nil:
		return s.Machine.Validate()

	case s.Return != nil:
		return s.Return.Validate()

	case s.Cancel != nil:
		return s.Cancel.Validate()

	case s.Fail != nil:
		if strings.TrimSpace(s.Fail.Code) == "" {
			return errors.New("workflow fail code is required")
		}
		if s.Fail.Details != nil {
			if err := s.Fail.Details.Validate(); err != nil {
				return err
			}
		}
		return s.Fail.Message.Validate()
	case s.Wait != nil:
		if err := s.Wait.Validate(); err != nil {
			return err
		}
		return nil
	case s.Emit != nil:
		return s.Emit.Validate()
	case s.Break, s.Continue:
		return nil
	default:
		return errors.New("workflow step operation is unavailable")
	}
}

func (w WorkflowWait) Validate() error {
	count := 0
	for _, present := range []bool{
		w.After != nil || w.At != nil,
		w.Event != nil,
		w.Channel != nil,
		w.Completion != nil,
		w.Human != nil,
	} {
		if present {
			count++
		}
	}
	if count != 1 {
		return errors.New("workflow wait must contain exactly one wait kind")
	}
	if w.After != nil || w.At != nil {
		if (w.After == nil) == (w.At == nil) {
			return errors.New("timer wait requires exactly one of after or at")
		}
		if w.After != nil {
			return w.After.Validate()
		}
	}
	if w.Event != nil {
		return w.Event.Validate()
	}

	if w.Channel != nil {
		return w.Channel.Validate()
	}
	if w.Completion != nil {
		return w.Completion.Validate()
	}
	return w.Human.Validate()
}

func (c WorkflowCall) Validate() error {
	if err := c.Target.Validate(); err != nil {
		return err
	}
	if err := c.Input.Validate(); err != nil {
		return err
	}
	if err := c.Context.Validate(); err != nil {
		return err
	}
	if c.LLMInvocation != nil {
		if err := c.LLMInvocation.Validate(); err != nil {
			return err
		}
	}
	if err := validateUniqueWorkflowPortBindings(c.Ports); err != nil {
		return err
	}
	mode := c.Mode
	if mode == "" {
		mode = WorkflowCallModeInvoke
	}
	switch mode {
	case WorkflowCallModeInvoke, WorkflowCallModeStart, WorkflowCallModeHandoff:
	default:
		return errors.New("workflow call mode is invalid")
	}
	if c.Lifetime != "" &&
		c.Lifetime != WorkflowChildLifetimeAttached &&
		c.Lifetime != WorkflowChildLifetimeIndependent {
		return errors.New("workflow child lifetime is invalid")
	}
	if mode == WorkflowCallModeInvoke && c.Lifetime != "" {
		return errors.New("workflow invoke call cannot declare a child lifetime")
	}
	if mode == WorkflowCallModeHandoff {
		if c.Conversation == nil {
			return errors.New("workflow handoff requires a conversation")
		}
		if c.Lifetime == WorkflowChildLifetimeIndependent {
			return errors.New("workflow handoff cannot use an independent child lifetime")
		}
	}
	if c.Conversation != nil {
		if err := c.Conversation.Validate(); err != nil {
			return err
		}
	}
	if c.Ownership != nil {
		if err := c.Ownership.Validate(); err != nil {
			return err
		}
		if err := c.Ownership.Budget.Validate(); err != nil {
			return err
		}
	}
	if c.OnCompletion != "" && c.OnCompletion != "restore" && c.OnCompletion != "retain" {
		return errors.New("workflow handoff completion behavior is invalid")
	}
	if c.AllowedTargetSet != "" &&
		len(c.AllowedTargets) != 0 {
		return errors.New(
			"workflow call cannot contain static and resolver target sets",
		)
	}
	seen := make(map[string]struct{}, len(c.AllowedTargets))
	for _, target := range c.AllowedTargets {
		if strings.TrimSpace(target) == "" {
			return errors.New("Workflow allowed target is empty")
		}
		if _, duplicate := seen[target]; duplicate {
			return errors.New("Workflow allowed target is duplicated")
		}
		seen[target] = struct{}{}
	}
	if c.SetContract != nil {
		if err := c.SetContract.Validate(); err != nil {
			return err
		}
	}
	if c.Contract != nil {
		return c.Contract.Validate()
	}
	return nil
}

func (s *WorkflowState) validateStructuredState() error {
	if s.Pending != nil {
		if err := s.Pending.Validate(); err != nil {
			return err
		}
	}
	if s.Terminal != nil {
		if err := s.Terminal.Validate(); err != nil {
			return err
		}
	}
	if err := validateWorkflowTargetSetStates(s.TargetSets); err != nil {
		return err
	}
	if !s.ExitValue.Valid() {
		return errors.New("Workflow Program exit value is invalid")
	}
	return nil
}

func (c WorkflowCancel) Validate() error {
	if err := c.Handle.Validate(); err != nil {
		return err
	}
	if c.Reason != nil {
		return c.Reason.Validate()
	}
	return nil
}

func (t WorkflowTransform) Validate() error {
	if err := t.Value.Validate(); err != nil {
		return err
	}
	if len(t.OutputSchema) != 0 && !t.OutputSchema.Valid() {
		return errors.New("Workflow transform output schema is invalid")
	}
	return nil
}

func (s WorkflowSet) Validate() error {
	seen := make(map[string]struct{}, len(s.Values))
	for _, value := range s.Values {
		if strings.TrimSpace(value.Key) == "" {
			return errors.New("Workflow state assignment key is empty")
		}
		if _, duplicate := seen[value.Key]; duplicate {
			return errors.New("Workflow state assignment key is duplicated")
		}
		if err := value.Value.Validate(); err != nil {
			return err
		}
		seen[value.Key] = struct{}{}
	}
	for _, path := range s.Remove {
		if strings.TrimSpace(path) == "" {
			return errors.New("Workflow state removal path is empty")
		}
		if _, duplicate := seen[path]; duplicate {
			return errors.New("Workflow state assignment and removal overlap")
		}
		seen[path] = struct{}{}
	}
	return nil
}

func (v WorkflowValue) Validate() error {
	switch v.Kind {
	case WorkflowValueLiteral:
		if len(v.Literal) == 0 || !v.Literal.Valid() {
			return errors.New("Workflow literal is invalid")
		}
		return nil

	case WorkflowValueExpression:
		if strings.TrimSpace(v.Expression) == "" ||
			len(v.Expression) > 65536 {
			return errors.New("Workflow expression is invalid")
		}
		return nil

	case WorkflowValueObject:
		seen := make(map[string]struct{}, len(v.Object))
		for _, value := range v.Object {
			if _, duplicate := seen[value.Key]; duplicate {
				return errors.New("Workflow object key is duplicated")
			}
			if err := value.Value.Validate(); err != nil {
				return err
			}
			seen[value.Key] = struct{}{}
		}
		return nil

	case WorkflowValueArray:
		for _, value := range v.Array {
			if err := value.Validate(); err != nil {
				return err
			}
		}
		return nil

	default:
		return errors.New("Workflow value kind is unsupported")
	}
}

func (c WorkflowCallableInterface) Validate() error {
	for _, mode := range c.Modes {
		switch mode {
		case WorkflowCallModeInvoke, WorkflowCallModeStart, WorkflowCallModeHandoff:
		default:
			return errors.New("invalid workflow callable mode")
		}
	}
	if len(c.InputSchema) == 0 ||
		len(c.OutputSchema) == 0 ||
		!c.InputSchema.Valid() ||
		!c.OutputSchema.Valid() ||
		!c.ErrorSchema.Valid() ||
		!c.ReceiptSchema.Valid() {
		return errors.New("Workflow callable interface is invalid")
	}
	for _, requirement := range c.Context {
		if err := requirement.Validate(); err != nil {
			return err
		}
	}
	for _, port := range c.Ports {
		if err := port.Validate(); err != nil {
			return err
		}
	}
	if c.Limits != nil {
		if err := c.Limits.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (t WorkflowTarget) Validate() error {
	var err error
	switch t.Kind {
	case WorkflowTargetTool:
		if strings.TrimSpace(t.ToolName) == "" ||
			t.MCP != nil ||
			t.Artifact == nil {
			return errors.New("Workflow Tool target is invalid")
		}
		err = validatePin(*t.Artifact)

	case WorkflowTargetMCPTool:
		if t.MCP == nil || t.Artifact != nil {
			return errors.New("Workflow MCP Tool target is invalid")
		}
		err = t.MCP.Validate()

	case WorkflowTargetProviderTool:
		if t.ProviderTool == nil || t.Artifact != nil || t.MCP != nil {
			return errors.New("Workflow provider-managed Tool target is invalid")
		}
		err = t.ProviderTool.Validate()

	case WorkflowTargetService:
		if strings.TrimSpace(t.BindingVersion) == "" ||
			strings.TrimSpace(t.ServiceID) == "" ||
			t.Artifact != nil ||
			t.MCP != nil ||
			t.ProviderTool != nil {
			return errors.New("Workflow Service target is invalid")
		}

	case WorkflowTargetActivity:
		if strings.TrimSpace(t.BindingVersion) == "" ||
			strings.TrimSpace(t.ActivityID) == "" ||
			t.Artifact != nil ||
			t.MCP != nil ||
			t.ProviderTool != nil {
			return errors.New("Workflow Activity target is invalid")
		}

	case WorkflowTargetWorkflow:
		if t.NestedProgram == nil {
			return errors.New("Workflow nested target is invalid")
		}
		_, err = workflowProgramFromSnapshot(*t.NestedProgram)

	case WorkflowTargetAgent,
		WorkflowTargetModel:
		if t.Artifact == nil {
			return errors.New("Workflow Model or Agent target has no Artifact pin")
		}
		err = validatePin(*t.Artifact)

	default:
		return errors.New("Workflow target kind is unsupported")
	}
	if err != nil {
		return err
	}
	if t.Contract != nil {
		if err := t.Contract.Validate(); err != nil {
			return err
		}
	}
	if err := t.Transport.Validate(); err != nil {
		return err
	}
	if t.KeyedCapable {
		if t.Effect.Kind != EffectKindIdempotent {
			return errors.New(
				"workflow target keying capability requires an idempotent base effect",
			)
		}
		if t.Effect.IdempotencyKey != "" {
			return errors.New(
				"workflow target keying capability cannot contain an operation key",
			)
		}
	}
	return t.Effect.Validate()
}

func (m WorkflowMCP) Validate() error {
	if strings.TrimSpace(m.ToolName) == "" {
		return errors.New("Workflow MCP Tool name is required")
	}
	if err := validatePin(m.Server); err != nil {
		return err
	}
	return ToolBinding{
		Kind: ToolBindingKindMCP,
		MCP: &MCPToolBinding{
			ServerArtifact:   &m.Server,
			ServerID:         m.Server.RootID + ":" + m.Server.ArtifactID,
			ProviderToolName: m.ProviderToolName,
			ChoiceID:         m.ChoiceID,
			ToolName:         m.ToolName,
			ToolDigest:       m.ToolDigest,
			ApprovalRule:     m.ApprovalRule,
			ExecutionMode:    m.ExecutionMode,
		},
	}.Validate()
}

func workflowProgramFromSnapshot(
	snapshot ProgramSnapshot,
) (WorkflowProgram, error) {
	if snapshot.InterpreterID != WorkflowInterpreterID ||
		snapshot.InterpreterVersion != WorkflowInterpreterVersion {
		return WorkflowProgram{}, errors.New(
			"Workflow Program snapshot has another interpreter",
		)
	}
	program, err := jsonutil.DecodeCanonicalValue[WorkflowProgram](
		snapshot.Payload,
	)
	if err != nil {
		return WorkflowProgram{}, err
	}
	if err := program.Validate(); err != nil {
		return WorkflowProgram{}, err
	}
	return program, nil
}

func decodeWorkflowProgram(
	frame ProgramFrame,
) (WorkflowProgram, WorkflowState, error) {
	program, err := workflowProgramFromSnapshot(frame.Program)
	if err != nil {
		return WorkflowProgram{}, WorkflowState{}, err
	}

	state, err := jsonutil.DecodeCanonicalValue[WorkflowState](
		frame.State.Data,
	)
	if err != nil {
		return WorkflowProgram{}, WorkflowState{}, err
	}
	if err := state.Validate(); err != nil {
		return WorkflowProgram{}, WorkflowState{}, err
	}
	return program, state, nil
}

func workflowTransition(
	state WorkflowState,
	status *ProgramFrameState,
	output jsonutil.CanonicalValue,
	work []WorkItemIntent,
) (ProgramTransition, error) {
	releaseClaims := state.ReleaseFrameClaims
	state.ReleaseFrameClaims = false

	for index := range work {
		if work[index].DeclarationPath == "" {
			work[index].DeclarationPath = state.PendingDeclarationPath
		}
		if work[index].Evaluation == nil &&
			state.PendingDeclarationPath != "" {
			work[index].Evaluation = &EvaluationBoundary{
				DeclarationPath: state.PendingDeclarationPath,
				LogicalTime:     state.LogicalTime.UTC(),
			}
		}
	}

	if state.Pending != nil &&
		!state.Pending.ClaimsBound &&
		len(state.Pending.Claims) != 0 &&
		len(work) != 0 {
		work[0].Claims = append(
			work[0].Claims,
			state.Pending.Claims...,
		)
		work[0].RetainClaimsForFrame = true
		state.Pending.ClaimsBound = true
	}

	cancelPaths := append([]string(nil), state.CancelChildPaths...)
	cancelWork := append([]WorkItemID(nil), state.CancelWorkItemIDs...)
	state.CancelChildPaths = nil
	state.CancelWorkItemIDs = nil
	next, err := encodeWorkflowProgramState(state)
	if err != nil {
		return ProgramTransition{}, err
	}
	return ProgramTransition{
		State:              next,
		Status:             status,
		Output:             output.Clone(),
		WorkItems:          work,
		ReleaseFrameClaims: releaseClaims,
		CancelChildPaths:   cancelPaths,
		CancelWorkItemIDs:  cancelWork,
	}, nil
}

func workflowFailure(
	state WorkflowState,
	code string,
	message string,
) (ProgramTransition, error) {
	status := ProgramFrameStateFailed
	transition, err := workflowTransition(state, &status, nil, nil)
	if err != nil {
		return ProgramTransition{}, err
	}
	transition.Error = &SanitizedError{
		Code:    firstNonEmpty(code, "workflow_failed"),
		Message: firstNonEmpty(message, "workflow execution failed"),
	}
	return transition, nil
}

func workflowWork(
	state WorkflowState,
	path string,
	intent WorkItemIntent,
) (ProgramTransition, error) {
	if intent.Effect.Kind != EffectKindRead {
		for _, dependency := range workflowCausalEffectWorkItemIDs(state) {
			if !slices.Contains(intent.Dependencies, dependency) {
				intent.Dependencies = append(intent.Dependencies, dependency)
			}
		}
	}
	state.PendingPath = path
	intent.DeclarationPath = state.PendingDeclarationPath
	if state.PendingDeclarationPath != "" {
		intent.Evaluation = &EvaluationBoundary{
			DeclarationPath: state.PendingDeclarationPath,
			LogicalTime:     state.LogicalTime.UTC(),
		}
	}
	intent.Deadline = earlierDeadline(intent.Deadline, state.Deadline)
	return workflowTransition(
		state,
		nil,
		nil,
		[]WorkItemIntent{intent},
	)
}

func encodeWorkflowProgramState(
	state WorkflowState,
) (ProgramState, error) {
	data, err := canonicalWorkflowValue(state)
	if err != nil {
		return ProgramState{}, err
	}
	return ProgramState{
		Type:    WorkflowInterpreterID,
		Version: WorkflowInterpreterVersion,
		Data:    data,
	}, nil
}

func workflowActivation(
	program WorkflowProgram,
	state WorkflowState,
) (map[string]any, error) {
	input, err := workflowActivationValue(program.Input)
	if err != nil {
		return nil, err
	}
	currentState, err := workflowActivationValue(state.State)
	if err != nil {
		return nil, err
	}

	steps := make(map[string]any, len(state.Outcomes))
	for _, outcome := range state.Outcomes {
		value := map[string]any{
			"status": string(outcome.Status),
		}
		if len(outcome.Output) != 0 {
			output, err := workflowActivationValue(outcome.Output)
			if err != nil {
				return nil, err
			}
			value["output"] = output
		}
		if outcome.Target != "" {
			value["target"] = outcome.Target
		}
		if outcome.Effect != "" {
			value["effectState"] = string(outcome.Effect)
		}
		if outcome.Receipt != nil {
			receipt, err := canonicalWorkflowValue(outcome.Receipt)
			if err != nil {
				return nil, err
			}
			value["receipt"], err = workflowActivationValue(receipt)
			if err != nil {
				return nil, err
			}
		}
		if outcome.Error != nil {
			errorValue, err := canonicalWorkflowValue(outcome.Error)
			if err != nil {
				return nil, err
			}
			value["error"], err = workflowActivationValue(errorValue)
			if err != nil {
				return nil, err
			}
		}
		steps[outcome.ID] = value
	}

	for _, outcome := range program.Scope.OuterOutcomes {
		value := map[string]any{
			"status": string(outcome.Status),
		}
		if len(outcome.Output) != 0 {
			output, err := workflowActivationValue(outcome.Output)
			if err != nil {
				return nil, err
			}
			value["output"] = output
		}
		if outcome.Target != "" {
			value["target"] = outcome.Target
		}
		if outcome.Effect != "" {
			value["effectState"] = string(outcome.Effect)
		}
		if outcome.Receipt != nil {
			receipt, err := canonicalWorkflowValue(outcome.Receipt)
			if err != nil {
				return nil, err
			}
			value["receipt"], err = workflowActivationValue(receipt)
			if err != nil {
				return nil, err
			}
		}
		if outcome.Error != nil {
			errorValue, err := canonicalWorkflowValue(outcome.Error)
			if err != nil {
				return nil, err
			}
			value["error"], err = workflowActivationValue(errorValue)
			if err != nil {
				return nil, err
			}
		}
		steps[outcome.ID] = value
	}

	var item any
	if program.Scope.Item != nil {
		item, err = workflowActivationValue(*program.Scope.Item)
		if err != nil {
			return nil, err
		}
	}
	messages, err := workflowActivationValue(state.Messages)
	if err != nil {
		return nil, err
	}
	contextValue, err := workflowActivationContext(
		program.Scope.Context,
		state.LogicalTime,
	)
	if err != nil {
		return nil, err
	}
	return workflowActivationWithScope(
		program,
		input,
		currentState,
		steps,
		item,
		contextValue,
		messages,
	)
}

func workflowActivationWithScope(
	program WorkflowProgram,
	input any,
	currentState any,
	steps map[string]any,
	item any,
	contextValue any,
	messagesValue any,
) (map[string]any, error) {
	var (
		index        any
		result       any
		eventValue   any
		errorValue   any
		outcomeValue any
	)
	if program.Scope.Index != nil {
		index = *program.Scope.Index
	}
	if program.Scope.Result != nil {
		value, err := workflowActivationValue(*program.Scope.Result)
		if err != nil {
			return nil, err
		}
		result = value
	}
	if program.Scope.Event != nil {
		value, err := workflowActivationValue(*program.Scope.Event)
		if err != nil {
			return nil, err
		}
		eventValue = value
	}
	if program.Scope.Error != nil {
		value, err := workflowActivationValue(*program.Scope.Error)
		if err != nil {
			return nil, err
		}
		errorValue = value
	}
	if program.Scope.Outcome != nil {
		value, err := workflowActivationValue(*program.Scope.Outcome)
		if err != nil {
			return nil, err
		}
		outcomeValue = value
	}
	var operationValue any
	if program.Scope.Operation != nil {
		value, err := workflowActivationValue(*program.Scope.Operation)
		if err != nil {
			return nil, err
		}
		operationValue = value
	}
	var completedValue any
	if program.Scope.Completed != nil {
		value, err := workflowActivationValue(*program.Scope.Completed)
		if err != nil {
			return nil, err
		}
		completedValue = value
	}
	portsValue, err := workflowActivationValue(jsonutil.MustCanonicalValue(program.Scope.Ports))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"input":     input,
		"state":     currentState,
		"steps":     steps,
		"context":   contextValue,
		"messages":  messagesValue,
		"item":      item,
		"index":     index,
		"result":    result,
		"event":     eventValue,
		"error":     errorValue,
		"outcome":   outcomeValue,
		"completed": completedValue,
		"operation": operationValue,
		"ports":     portsValue,
	}, nil
}

func (i *WorkflowInterpreter) conditionWithEvent(
	value WorkflowValue,
	program WorkflowProgram,
	state WorkflowState,
	event jsonutil.CanonicalValue,
) (bool, error) {
	result, err := i.evaluateWithEvent(value, program, state, event)
	if err != nil {
		return false, err
	}
	return jsonutil.DecodeCanonicalValue[bool](result)
}

func (i *WorkflowInterpreter) evaluateWithEvent(
	value WorkflowValue,
	program WorkflowProgram,
	state WorkflowState,
	event jsonutil.CanonicalValue,
) (jsonutil.CanonicalValue, error) {
	activation, err := workflowActivation(program, state)
	if err != nil {
		return nil, err
	}
	if len(event) != 0 {
		eventValue, err := workflowActivationValue(event)
		if err != nil {
			return nil, err
		}
		activation["event"] = eventValue
	}
	return i.evaluateWithActivation(value, activation)
}

func applyWorkflowStateUpdate(
	update StateUpdateWork,
) (jsonutil.CanonicalValue, error) {
	values, err := jsonutil.DecodeCanonicalValue[map[string]jsonutil.CanonicalValue](
		update.Base,
	)
	if err != nil {
		return nil, err
	}
	if values == nil {
		values = map[string]jsonutil.CanonicalValue{}
	}

	for _, assignment := range update.Assignments {
		values[assignment.Path] = assignment.Value.Clone()
	}
	for _, path := range update.RemovePaths {
		delete(values, path)
	}

	return canonicalWorkflowValue(values)
}

func workflowWorkValue(
	item WorkItem,
) (jsonutil.CanonicalValue, error) {
	if item.Output == nil {
		return nil, errors.New("Workflow WorkItem has no output")
	}
	if len(item.Output.Value) != 0 {
		return item.Output.Value.Clone(), nil
	}
	if item.Output.Tool != nil &&
		len(item.Output.Tool.Structured) != 0 {
		return item.Output.Tool.Structured.Clone(), nil
	}
	return nil, errors.New(
		"Workflow WorkItem output has no durable semantic value",
	)
}

func workflowExternalContext(value jsonutil.CanonicalValue) jsonutil.CanonicalValue {
	context, err := jsonutil.DecodeCanonicalValue[map[string]jsonutil.CanonicalValue](value)
	if err == nil && len(context) == 0 {
		return nil
	}
	return value.Clone()
}

func workflowSnapshotWithInput(
	snapshot ProgramSnapshot,
	input jsonutil.CanonicalValue,
) (ProgramSnapshot, error) {
	program, err := workflowProgramFromSnapshot(snapshot)
	if err != nil {
		return ProgramSnapshot{}, err
	}

	program.Input = input.Clone()
	program.InputBound = true
	if err := program.ValidateInput(); err != nil {
		return ProgramSnapshot{}, err
	}

	payload, err := canonicalWorkflowValue(program)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	snapshot.Payload = payload
	return snapshot, nil
}

func workflowSnapshotWithContext(
	snapshot ProgramSnapshot,
	context jsonutil.CanonicalValue,
) (ProgramSnapshot, error) {
	if !context.Valid() {
		return ProgramSnapshot{}, errors.New(
			"workflow child context is not canonical json",
		)
	}

	program, err := workflowProgramFromSnapshot(snapshot)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	if len(context) == 0 {
		program.Scope.Context = nil
	} else {
		value := context.Clone()
		program.Scope.Context = &value
	}
	if err := program.Validate(); err != nil {
		return ProgramSnapshot{}, err
	}

	payload, err := canonicalWorkflowValue(program)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	snapshot.Payload = payload
	return snapshot, nil
}

func canonicalWorkflowValue(
	value any,
) (jsonutil.CanonicalValue, error) {
	raw, err := jsonutil.EncodeToJSONRaw(value)
	if err != nil {
		return nil, err
	}
	return jsonutil.CanonicalValueFrom(raw)
}

func validateWorkflowSchema(
	schema jsonutil.CanonicalValue,
	value jsonutil.CanonicalValue,
) error {
	if len(schema) == 0 {
		return nil
	}
	if !schema.Valid() || !value.Valid() {
		return errors.New("Workflow schema or value is invalid")
	}

	compiled, err := jsonutil.CompileJSONSchema([]byte(schema))
	if err != nil {
		return err
	}
	return jsonutil.ValidateJSONSchema(
		compiled,
		json.RawMessage(value.Clone()),
		maxWorkflowProgramJSONBytes,
	)
}
