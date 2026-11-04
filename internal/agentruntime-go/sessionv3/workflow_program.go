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
// CanonicalValue because Workflow input and output can be arbitrary JSON, but
// no runtime struct uses json.RawMessage as its public representation.
type WorkflowValue struct {
	Kind       WorkflowValueKind   `json:"kind"`
	Literal    CanonicalValue      `json:"literal,omitempty"`
	Expression string              `json:"expression,omitempty"`
	Object     []WorkflowValuePair `json:"object,omitempty"`
	Array      []WorkflowValue     `json:"array,omitempty"`
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

type WorkflowCallableInterface struct {
	Modes         []WorkflowCallMode `json:"modes,omitempty"`
	InputSchema   CanonicalValue     `json:"inputSchema"`
	OutputSchema  CanonicalValue     `json:"outputSchema"`
	ErrorSchema   CanonicalValue     `json:"errorSchema,omitempty"`
	ReceiptSchema CanonicalValue     `json:"receiptSchema,omitempty"`
}

type WorkflowProgram struct {
	Version uint32 `json:"version"`

	InputSchema  CanonicalValue `json:"inputSchema"`
	OutputSchema CanonicalValue `json:"outputSchema"`
	Input        CanonicalValue `json:"input"`
	InputBound   bool           `json:"inputBound"`

	Root     WorkflowBlock             `json:"root"`
	Targets  map[string]WorkflowTarget `json:"targets"`
	Events   []WorkflowEventContract   `json:"events,omitempty"`
	Channels []WorkflowChannel         `json:"channels,omitempty"`

	// Scope is empty for a root Workflow Program. Child block ProgramFrames
	// receive an immutable parent snapshot through this typed envelope.
	Scope WorkflowScopeContext `json:"scope"`
}

type WorkflowBlock struct {
	State  *WorkflowStateSpec     `json:"state,omitempty"`
	Listen []WorkflowSubscription `json:"listen,omitempty"`
	Steps  []WorkflowStep         `json:"steps"`
	Output WorkflowValue          `json:"output"`
}

type WorkflowStateSpec struct {
	Schema  CanonicalValue `json:"schema"`
	Initial WorkflowValue  `json:"initial"`
}

type WorkflowStep struct {
	ID           string           `json:"id"`
	Timeout      WorkflowDuration `json:"timeout,omitempty"`
	When         *WorkflowValue   `json:"when,omitempty"`
	Retry        *RetryPolicy     `json:"retry,omitempty"`
	OutputSchema CanonicalValue   `json:"outputSchema,omitempty"`

	Call      *WorkflowCall      `json:"call,omitempty"`
	Transform *WorkflowTransform `json:"transform,omitempty"`
	Set       *WorkflowSet       `json:"set,omitempty"`
	Scope     *WorkflowBlock     `json:"scope,omitempty"`
	Choose    *WorkflowChoose    `json:"choose,omitempty"`
	Parallel  *WorkflowParallel  `json:"parallel,omitempty"`
	Foreach   *WorkflowForeach   `json:"foreach,omitempty"`
	Repeat    *WorkflowRepeat    `json:"repeat,omitempty"`
	Wait      *WorkflowWait      `json:"wait,omitempty"`
	Emit      *WorkflowEmit      `json:"emit,omitempty"`
	Return    *WorkflowReturn    `json:"return,omitempty"`
	Fail      *WorkflowFail      `json:"fail,omitempty"`
	Break     bool               `json:"break,omitempty"`
	Continue  bool               `json:"continue,omitempty"`
}

type WorkflowCall struct {
	Target         WorkflowValue              `json:"target"`
	AllowedTargets []string                   `json:"allowedTargets,omitempty"`
	Input          WorkflowValue              `json:"input"`
	Context        WorkflowValue              `json:"context"`
	Contract       *WorkflowCallableInterface `json:"contract,omitempty"`
	SetContract    *WorkflowCallableInterface `json:"setContract,omitempty"`
}

type WorkflowTransform struct {
	Value        WorkflowValue  `json:"value"`
	OutputSchema CanonicalValue `json:"outputSchema,omitempty"`
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
	WorkflowTargetTool     WorkflowTargetKind = "tool"
	WorkflowTargetMCPTool  WorkflowTargetKind = "mcpTool"
	WorkflowTargetService  WorkflowTargetKind = "service"
	WorkflowTargetActivity WorkflowTargetKind = "activity"
	WorkflowTargetWorkflow WorkflowTargetKind = "workflow"
	WorkflowTargetAgent    WorkflowTargetKind = "agent"
	WorkflowTargetModel    WorkflowTargetKind = "model"
)

type WorkflowTarget struct {
	Kind WorkflowTargetKind `json:"kind"`

	Artifact *ArtifactPin `json:"artifact,omitempty"`
	MCP      *WorkflowMCP `json:"mcp,omitempty"`

	ServiceID      string `json:"serviceID,omitempty"`
	ActivityID     string `json:"activityID,omitempty"`
	BindingVersion string `json:"bindingVersion,omitempty"`

	NestedProgram *ProgramSnapshot           `json:"nestedProgram,omitempty"`
	ToolName      string                     `json:"toolName,omitempty"`
	Contract      *WorkflowCallableInterface `json:"contract,omitempty"`
	Effect        EffectContract             `json:"effect"`
}

type WorkflowMCP struct {
	Server   ArtifactPin `json:"server"`
	ToolName string      `json:"toolName"`
}

type WorkflowStepOutcomeStatus string

const (
	WorkflowStepOutcomeSucceeded WorkflowStepOutcomeStatus = "succeeded"
	WorkflowStepOutcomeSkipped   WorkflowStepOutcomeStatus = "skipped"
)

type WorkflowStepOutcome struct {
	ID     string                    `json:"id"`
	Status WorkflowStepOutcomeStatus `json:"status"`
	Output CanonicalValue            `json:"output,omitempty"`
}

// WorkflowState is the durable typed state of one sequential Workflow frame.
// It contains semantic state and step outcomes only. It never contains CEL
// programs, compiled schemas, Artifact services, clients, callbacks, or
// private authorization data.
type WorkflowState struct {
	Version uint32 `json:"version"`

	Index            int                         `json:"index"`
	PendingPath      string                      `json:"pendingPath,omitempty"`
	PendingTarget    string                      `json:"pendingTarget,omitempty"`
	Pending          *WorkflowPending            `json:"pending,omitempty"`
	Exit             WorkflowExitKind            `json:"exit,omitempty"`
	ExitValue        CanonicalValue              `json:"exitValue,omitempty"`
	LogicalTime      time.Time                   `json:"logicalTime,omitzero"`
	Deadline         *time.Time                  `json:"deadline,omitempty"`
	State            CanonicalValue              `json:"state"`
	Outcomes         []WorkflowStepOutcome       `json:"outcomes"`
	Subscriptions    []WorkflowSubscriptionState `json:"subscriptions,omitempty"`
	CancelChildPaths []string                    `json:"cancelChildPaths,omitempty"`
}

type WorkflowInterpreter struct {
	env *cel.Env
}

func (s *WorkflowState) Validate() error {
	if s.Version != WorkflowInterpreterVersion ||
		s.Index < 0 ||
		!s.State.Valid() ||
		s.Outcomes == nil {
		return errors.New("Workflow Program state is invalid")
	}
	seen := make(map[string]struct{}, len(s.Outcomes))
	for _, outcome := range s.Outcomes {
		if strings.TrimSpace(outcome.ID) == "" ||
			!outcome.Output.Valid() {
			return errors.New("Workflow Program output state is invalid")
		}
		switch outcome.Status {
		case WorkflowStepOutcomeSucceeded:
			if len(outcome.Output) == 0 {
				return errors.New("successful Workflow step has no output")
			}
		case WorkflowStepOutcomeSkipped:
			if len(outcome.Output) != 0 {
				return errors.New("skipped Workflow step has an output")
			}
		default:
			return errors.New("Workflow step outcome status is invalid")
		}
		if _, duplicate := seen[outcome.ID]; duplicate {
			return errors.New("Workflow Program step outcome is duplicated")
		}
		seen[outcome.ID] = struct{}{}
	}
	for _, subscription := range s.Subscriptions {
		if err := subscription.Validate(); err != nil {
			return err
		}
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
	s.Outcomes = append(s.Outcomes, WorkflowStepOutcome{
		ID:     outcome.ID,
		Status: outcome.Status,
		Output: outcome.Output.Clone(),
	})
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

	program, err := DecodeCanonicalValue[WorkflowProgram](snapshot.Payload)
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

	value, err := DecodeCanonicalValue[WorkflowState](state.Data)
	if err != nil {
		return err
	}
	return value.Validate()
}

func (i *WorkflowInterpreter) InitialState(
	snapshot ProgramSnapshot,
) (ProgramState, error) {
	program, err := workflowProgramFromSnapshot(snapshot)
	if err != nil {
		return ProgramState{}, err
	}
	if err := program.ValidateInput(); err != nil {
		return ProgramState{}, err
	}

	initialState := MustCanonicalValue(map[string]CanonicalValue{})
	if program.Scope.Child && len(program.Scope.OuterState) != 0 {
		initialState = program.Scope.OuterState.Clone()
	}
	state := WorkflowState{
		Version:     WorkflowInterpreterVersion,
		State:       initialState,
		Outcomes:    []WorkflowStepOutcome{},
		LogicalTime: time.Now().UTC(),
	}
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
	if !program.Scope.Child && state.LogicalTime.IsZero() {
		state.LogicalTime = time.Now().UTC()
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

	switch event.Kind {
	case ProgramEventCancelled:
		status := ProgramFrameStateCancelled
		return workflowTransition(state, &status, nil, nil)

	case ProgramEventWorkSettled:
		item := executionWorkItem(execution, event.WorkItemID)
		if item != nil && item.FrameID == frame.ID &&
			state.PendingPath != "" && item.ProgramPath == state.PendingPath {
			if failure := workflowSettledFailure(program, state, *item); failure != nil {
				return workflowFailureTransition(state, failure)
			}
		}
		if err := i.settleWork(
			&state,
			program,
			execution,
			frame.ID,
			event.WorkItemID,
		); err != nil {
			return workflowFailure(state, "workflow_work_failed", err.Error())
		}

	case ProgramEventChildSettled:
		if err := i.settleChild(
			&state,
			program,
			execution,
			frame.ID,
			event.ChildFrameID,
		); err != nil {
			if failure, ok := errors.AsType[workflowChildError](err); ok {
				return workflowFailureTransition(
					state,
					&failure.errorValue,
				)
			}
			return workflowFailure(state, "workflow_child_failed", err.Error())
		}

	case ProgramEventStarted,
		ProgramEventContinue,
		ProgramEventActivity:
	default:
	}

	return i.drive(frame, execution, program, state)
}

func (i *WorkflowInterpreter) drive(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
) (ProgramTransition, error) {
	if state.Pending != nil {
		return i.drivePending(frame, execution, program, state)
	}

	for {
		if state.PendingPath != "" {
			return workflowTransition(state, nil, nil, nil)
		}

		if state.Index >= len(program.Root.Steps) {
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
					ID:     step.ID,
					Status: WorkflowStepOutcomeSkipped,
				}); err != nil {
					return workflowFailure(
						state,
						"workflow_state_invalid",
						err.Error(),
					)
				}
				state.Index++
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

		switch {
		case step.Call != nil:
			return i.call(
				frame,
				program,
				state,
				path,
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
				program,
				state,
				path,
				*step.Set,
			)

		case step.Wait != nil:
			return i.wait(program, state, path, *step.Wait)

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

			exit := WorkflowExitReturnBlock
			if step.Return.To == WorkflowReturnDestinationWorkflow {
				exit = WorkflowExitReturnWorkflow
			}
			return i.completeWorkflowBlock(program, state, output, exit)

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
			transition, err := workflowFailure(state, step.Fail.Code, message)
			if err != nil {
				return ProgramTransition{}, err
			}
			transition.Error.Category = step.Fail.Category
			transition.Error.Retryable = step.Fail.Retryable
			transition.Error.StepPath = path
			if step.Fail.Details != nil {
				details, err := i.evaluate(*step.Fail.Details, program, state)
				if err != nil {
					return workflowFailure(state, "workflow_fail_invalid", err.Error())
				}
				transition.Error.Details = details
			}
			return transition, nil

		case step.Break:
			return i.completeWorkflowBlock(
				program,
				state,
				MustCanonicalValue(nil),
				WorkflowExitBreak,
			)

		case step.Continue:
			return i.completeWorkflowBlock(
				program,
				state,
				MustCanonicalValue(nil),
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
	call WorkflowCall,
) (ProgramTransition, error) {
	targetName, err := i.stringValue(call.Target, program, state)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_target_invalid",
			err.Error(),
		)
	}
	if len(call.AllowedTargets) != 0 &&
		!slices.Contains(call.AllowedTargets, targetName) {
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
			"Workflow target is unavailable",
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
	if err := validateWorkflowInterfaceInput(
		target.Contract,
		input,
	); err != nil {
		return workflowFailure(state, "workflow_target_input_invalid", err.Error())
	}
	if err := validateWorkflowInterfaceInput(call.Contract, input); err != nil {
		return workflowFailure(state, "workflow_call_input_invalid", err.Error())
	}
	if err := validateWorkflowInterfaceInput(call.SetContract, input); err != nil {
		return workflowFailure(state, "workflow_target_set_input_invalid", err.Error())
	}
	for _, contract := range []*WorkflowCallableInterface{target.Contract, call.Contract, call.SetContract} {
		if contract != nil && len(contract.Modes) != 0 && !slices.Contains(contract.Modes, WorkflowCallModeInvoke) {
			return workflowFailure(state, "workflow_call_mode_invalid", "target interface does not permit invoke")
		}
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
		return workflowWork(
			state,
			path,
			WorkItemIntent{
				ProgramPath: path,
				Type:        WorkItemTypeTool,
				Automatic:   true,
				Effect:      target.Effect,
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
		return workflowWork(
			state,
			path,
			WorkItemIntent{
				ProgramPath: path,
				Type:        WorkItemTypeTool,
				Automatic:   true,
				Effect:      target.Effect,
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
							Server:   target.MCP.Server,
							ToolName: target.MCP.ToolName,
						},
					},
				},
			},
		)

	case WorkflowTargetService:
		return workflowWork(
			state,
			path,
			WorkItemIntent{
				ProgramPath: path,
				Type:        WorkItemTypeService,
				Automatic:   true,
				Effect:      target.Effect,
				Input: WorkItemPayload{
					Service: &ServiceWork{
						ServiceID:      target.ServiceID,
						BindingVersion: target.BindingVersion,
						Input:          input,
					},
				},
			},
		)

	case WorkflowTargetActivity:
		return workflowWork(
			state,
			path,
			WorkItemIntent{
				ProgramPath: path,
				Type:        WorkItemTypeActivity,
				Automatic:   true,
				Effect:      target.Effect,
				Input: WorkItemPayload{
					Activity: &ActivityWork{
						ActivityID:     target.ActivityID,
						BindingVersion: target.BindingVersion,
						Input:          input,
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

		next := state
		next.PendingPath = path
		nextState, err := encodeWorkflowProgramState(next)
		if err != nil {
			return ProgramTransition{}, err
		}
		return ProgramTransition{
			State: nextState,
			Frames: []ProgramFrameIntent{{
				Path:     path,
				Deadline: cloneTime(state.Deadline),
				Program:  nested,
			}},
		}, nil

	case WorkflowTargetAgent,
		WorkflowTargetModel:
		if target.Artifact == nil {
			return workflowFailure(
				state,
				"workflow_target_invalid",
				"workflow llm target has no artifact pin",
			)
		}

		invocation := LLMInvocationPlan{
			Input:   input,
			Context: contextValue,
		}
		defaults := SessionDefaults{}
		switch target.Kind {
		case WorkflowTargetModel:
			defaults.Model = workflowSelectionForPin(*target.Artifact)
		case WorkflowTargetAgent:
			defaults.Agent = workflowSelectionForPin(*target.Artifact)
		default:
		}

		var childProgram ProgramSnapshot
		if target.Kind == WorkflowTargetAgent {
			childProgram, err = NewWorkflowAgentLoopProgram(
				defaults,
				invocation,
			)
		} else {
			childProgram, err = NewWorkflowSingleModelProgram(
				defaults,
				invocation,
			)
		}
		if err != nil {
			return workflowFailure(
				state,
				"workflow_target_invalid",
				err.Error(),
			)
		}

		next := state
		next.PendingPath = path
		nextState, err := encodeWorkflowProgramState(next)
		if err != nil {
			return ProgramTransition{}, err
		}
		return ProgramTransition{
			State: nextState,
			Frames: []ProgramFrameIntent{{
				ID:       ProgramFrameID(string(frame.ID) + ":" + path),
				Path:     path,
				Deadline: cloneTime(state.Deadline),
				Program:  childProgram,
			}},
		}, nil

	default:
		return workflowFailure(
			state,
			"workflow_target_invalid",
			"Workflow target kind is invalid",
		)
	}
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

	return workflowWork(
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
	program WorkflowProgram,
	state WorkflowState,
	path string,
	set WorkflowSet,
) (ProgramTransition, error) {
	if program.Root.State == nil {
		return workflowFailure(
			state,
			"workflow_state_unavailable",
			"Workflow set operation has no state declaration",
		)
	}

	assignments := make(
		[]StateAssignment,
		0,
		len(set.Values),
	)
	for _, assignment := range set.Values {
		value, err := i.evaluate(
			assignment.Value,
			program,
			state,
		)
		if err != nil {
			return workflowFailure(
				state,
				"workflow_state_update_invalid",
				err.Error(),
			)
		}
		assignments = append(assignments, StateAssignment{
			Path:  assignment.Key,
			Value: value,
		})
	}

	return workflowWork(
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
					Base:        state.State.Clone(),
					Schema:      program.Root.State.Schema.Clone(),
					Assignments: assignments,
					RemovePaths: append([]string(nil), set.Remove...),
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

	value, err := workflowWorkValue(*item)
	if err != nil {
		return err
	}
	switch item.Type {
	case WorkItemTypeStateUpdate:
		if item.Input.StateUpdate == nil {
			return errors.New("Workflow StateUpdate WorkItem has no payload")
		}
		if program.Root.State == nil {
			return errors.New("Workflow StateUpdate WorkItem has no state declaration")
		}
		state.State = value.Clone()
	default:
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
	state.PendingPath = ""
	state.PendingTarget = ""
	state.Deadline = nil
	state.Index++
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
		if err := state.recordOutcome(WorkflowStepOutcome{
			ID:     step.ID,
			Status: WorkflowStepOutcomeSucceeded,
			Output: child.Output,
		}); err != nil {
			return err
		}
		state.PendingPath = ""
		state.PendingTarget = ""
		state.Deadline = nil
		state.Index++
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
	value CanonicalValue,
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
	if err := validateWorkflowInterfaceOutput(step.Call.SetContract, value); err != nil {
		return fmt.Errorf("validate workflow target set output: %w", err)
	}
	return nil
}

func validateWorkflowInterfaceInput(
	contract *WorkflowCallableInterface,
	value CanonicalValue,
) error {
	if contract == nil {
		return nil
	}
	return validateWorkflowSchema(contract.InputSchema, value)
}

func validateWorkflowInterfaceOutput(
	contract *WorkflowCallableInterface,
	value CanonicalValue,
) error {
	if contract == nil {
		return nil
	}
	return validateWorkflowSchema(contract.OutputSchema, value)
}

func (i *WorkflowInterpreter) evaluate(
	value WorkflowValue,
	program WorkflowProgram,
	state WorkflowState,
) (CanonicalValue, error) {
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
			map[string]CanonicalValue,
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
			[]CanonicalValue,
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
	return DecodeCanonicalValue[bool](result)
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
	return DecodeCanonicalValue[string](result)
}

func (p WorkflowProgram) Validate() error {
	if p.Version != WorkflowInterpreterVersion {
		return errors.New("unsupported Workflow Program version")
	}
	if err := p.Scope.Validate(); err != nil {
		return err
	}
	if p.InputBound && len(p.Input) == 0 {
		return errors.New("bound Workflow Program input is missing")
	}
	if len(p.InputSchema) == 0 ||
		len(p.OutputSchema) == 0 ||
		!p.InputSchema.Valid() ||
		!p.OutputSchema.Valid() ||
		!p.Input.Valid() {
		return errors.New("workflow Program schemas or input are invalid")
	}
	if err := p.Root.Validate(); err != nil {
		return err
	}
	if err := validateWorkflowCoordinationProgram(p); err != nil {
		return err
	}
	for name, target := range p.Targets {
		if strings.TrimSpace(name) == "" {
			return errors.New("Workflow Program target name is empty")
		}
		if err := target.Validate(); err != nil {
			return fmt.Errorf("Workflow target %q: %w", name, err)
		}
	}
	return nil
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
		s.Choose != nil,
		s.Parallel != nil,
		s.Foreach != nil,
		s.Repeat != nil,
		s.Return != nil,
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

	switch {
	case s.Call != nil:
		return s.Call.Validate()

	case s.Transform != nil:
		return s.Transform.Validate()

	case s.Set != nil:
		return s.Set.Validate()

	case s.Scope != nil:
		return s.Scope.Validate()

	case s.Choose != nil:
		return s.Choose.Validate()

	case s.Parallel != nil:
		return s.Parallel.Validate()

	case s.Foreach != nil:
		return s.Foreach.Validate()

	case s.Repeat != nil:
		return s.Repeat.Validate()

	case s.Return != nil:
		return s.Return.Validate()

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
	if !s.ExitValue.Valid() {
		return errors.New("Workflow Program exit value is invalid")
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
	return nil
}

func (t WorkflowTarget) Validate() error {
	var err error
	switch t.Kind {
	case WorkflowTargetTool:
		if t.Artifact == nil ||
			strings.TrimSpace(t.ToolName) == "" ||
			t.MCP != nil {
			return errors.New("Workflow Tool target is invalid")
		}
		err = validatePin(*t.Artifact)

	case WorkflowTargetMCPTool:
		if t.MCP == nil || t.Artifact != nil {
			return errors.New("Workflow MCP Tool target is invalid")
		}
		err = t.MCP.Validate()

	case WorkflowTargetService:
		if strings.TrimSpace(t.ServiceID) == "" ||
			strings.TrimSpace(t.BindingVersion) == "" {
			return errors.New("Workflow Service target is invalid")
		}

	case WorkflowTargetActivity:
		if strings.TrimSpace(t.ActivityID) == "" ||
			strings.TrimSpace(t.BindingVersion) == "" {
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
			return errors.New("Workflow Artifact target is invalid")
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
	return t.Effect.Validate()
}

func (m WorkflowMCP) Validate() error {
	if strings.TrimSpace(m.ToolName) == "" {
		return errors.New("Workflow MCP Tool name is required")
	}
	return validatePin(m.Server)
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
	program, err := DecodeCanonicalValue[WorkflowProgram](
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

	state, err := DecodeCanonicalValue[WorkflowState](
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
	output CanonicalValue,
	work []WorkItemIntent,
) (ProgramTransition, error) {
	cancelPaths := append([]string(nil), state.CancelChildPaths...)
	state.CancelChildPaths = nil
	next, err := encodeWorkflowProgramState(state)
	if err != nil {
		return ProgramTransition{}, err
	}
	return ProgramTransition{
		State:            next,
		Status:           status,
		Output:           output.Clone(),
		WorkItems:        work,
		CancelChildPaths: cancelPaths,
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
	state.PendingPath = path
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
		steps[outcome.ID] = value
	}

	var item any
	if program.Scope.Item != nil {
		item, err = workflowActivationValue(*program.Scope.Item)
		if err != nil {
			return nil, err
		}
	}
	return workflowActivationWithScope(
		program,
		input,
		currentState,
		steps,
		item,
	)
}

func workflowActivationWithScope(
	program WorkflowProgram,
	input any,
	currentState any,
	steps map[string]any,
	item any,
) (map[string]any, error) {
	var (
		index  any
		result any
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
	return map[string]any{
		"input":     input,
		"state":     currentState,
		"steps":     steps,
		"context":   nil,
		"messages":  nil,
		"item":      item,
		"index":     index,
		"result":    result,
		"event":     nil,
		"error":     nil,
		"outcome":   nil,
		"completed": nil,
		"operation": nil,
	}, nil
}

func applyWorkflowStateUpdate(
	update StateUpdateWork,
) (CanonicalValue, error) {
	values, err := DecodeCanonicalValue[map[string]CanonicalValue](
		update.Base,
	)
	if err != nil {
		return nil, err
	}
	if values == nil {
		values = map[string]CanonicalValue{}
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
) (CanonicalValue, error) {
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

func workflowSnapshotWithInput(
	snapshot ProgramSnapshot,
	input CanonicalValue,
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

func canonicalWorkflowValue(
	value any,
) (CanonicalValue, error) {
	raw, err := jsonutil.EncodeToJSONRaw(value)
	if err != nil {
		return nil, err
	}
	return CanonicalValueFrom(raw)
}

func validateWorkflowSchema(
	schema CanonicalValue,
	value CanonicalValue,
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
