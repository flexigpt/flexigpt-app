package sessionv3

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"cel.dev/cel-go/cel"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	WorkflowInterpreterID      = "artifact.workflow.v1"
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

type WorkflowProgram struct {
	Version uint32 `json:"version"`

	InputSchema  CanonicalValue `json:"inputSchema"`
	OutputSchema CanonicalValue `json:"outputSchema"`
	Input        CanonicalValue `json:"input"`

	Root    WorkflowBlock             `json:"root"`
	Targets map[string]WorkflowTarget `json:"targets"`
}

type WorkflowBlock struct {
	State  *WorkflowStateSpec `json:"state,omitempty"`
	Steps  []WorkflowStep     `json:"steps"`
	Output WorkflowValue      `json:"output"`
}

type WorkflowStateSpec struct {
	Schema  CanonicalValue `json:"schema"`
	Initial WorkflowValue  `json:"initial"`
}

type WorkflowStep struct {
	ID           string         `json:"id"`
	When         *WorkflowValue `json:"when,omitempty"`
	OutputSchema CanonicalValue `json:"outputSchema,omitempty"`

	Call      *WorkflowCall      `json:"call,omitempty"`
	Transform *WorkflowTransform `json:"transform,omitempty"`
	Set       *WorkflowSet       `json:"set,omitempty"`
	Return    *WorkflowReturn    `json:"return,omitempty"`
	Fail      *WorkflowFail      `json:"fail,omitempty"`
}

type WorkflowCall struct {
	Target         WorkflowValue `json:"target"`
	AllowedTargets []string      `json:"allowedTargets,omitempty"`
	Input          WorkflowValue `json:"input"`
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
	Value WorkflowValue `json:"value"`
}

type WorkflowFail struct {
	Code    string        `json:"code"`
	Message WorkflowValue `json:"message"`
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

	NestedProgram *ProgramSnapshot `json:"nestedProgram,omitempty"`
	ToolName      string           `json:"toolName,omitempty"`
}

type WorkflowMCP struct {
	Server   ArtifactPin `json:"server"`
	ToolName string      `json:"toolName"`
}

type workflowProgramState struct {
	Version uint32 `json:"version"`

	Index       int                       `json:"index"`
	PendingPath string                    `json:"pendingPath,omitempty"`
	State       CanonicalValue            `json:"state"`
	Outputs     map[string]CanonicalValue `json:"outputs"`
}

type WorkflowInterpreter struct {
	env *cel.Env
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

	value, err := DecodeCanonicalValue[workflowProgramState](state.Data)
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

	state := workflowProgramState{
		Version: WorkflowInterpreterVersion,
		State:   MustCanonicalValue(map[string]any{}),
		Outputs: map[string]CanonicalValue{},
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

	switch event.Kind {
	case ProgramEventCancelled:
		status := ProgramFrameStateCancelled
		return workflowTransition(state, &status, nil, nil)

	case ProgramEventWorkSettled:
		if err := i.settleWork(&state, execution, event.WorkItemID); err != nil {
			return workflowFailure(state, "workflow_work_failed", err.Error())
		}

	case ProgramEventChildSettled:
		if err := i.settleChild(&state, execution, event.ChildFrameID); err != nil {
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
	state workflowProgramState,
) (ProgramTransition, error) {
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

			status := ProgramFrameStateCompleted
			return workflowTransition(state, &status, output, nil)
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
				state.Index++
				continue
			}
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

			status := ProgramFrameStateCompleted
			return workflowTransition(state, &status, output, nil)

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
			return workflowFailure(
				state,
				firstNonEmpty(
					step.Fail.Code,
					"workflow_failed",
				),
				message,
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
	state workflowProgramState,
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
				Effect: EffectContract{
					Kind: EffectKindNonIdempotent,
				},
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
				Effect: EffectContract{
					Kind: EffectKindNonIdempotent,
				},
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
				Effect: EffectContract{
					Kind: EffectKindNonIdempotent,
				},
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
				Effect: EffectContract{
					Kind: EffectKindNonIdempotent,
				},
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
				Path:    path,
				Program: nested,
			}},
		}, nil

	case WorkflowTargetAgent,
		WorkflowTargetModel:
		return workflowFailure(
			state,
			"workflow_target_unsupported",
			"Agent and Model Workflow targets require the v3 Agent/Workspace plan compiler",
		)

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
	state workflowProgramState,
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
	if err := validateWorkflowSchema(
		step.Transform.OutputSchema,
		value,
	); err != nil {
		return workflowFailure(
			state,
			"workflow_transform_output_invalid",
			err.Error(),
		)
	}
	if err := validateWorkflowSchema(
		step.OutputSchema,
		value,
	); err != nil {
		return workflowFailure(
			state,
			"workflow_step_output_invalid",
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
					Input: value,
				},
			},
		},
	)
}

func (i *WorkflowInterpreter) set(
	program WorkflowProgram,
	state workflowProgramState,
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
					Assignments: assignments,
					RemovePaths: append([]string(nil), set.Remove...),
				},
			},
		},
	)
}

func (i *WorkflowInterpreter) settleWork(
	state *workflowProgramState,
	execution Execution,
	workID WorkItemID,
) error {
	if state == nil {
		return errors.New("Workflow state is unavailable")
	}

	item := executionWorkItem(execution, workID)
	if item == nil {
		return errors.New("Workflow WorkItem is unavailable")
	}
	if state.PendingPath == "" || item.ProgramPath != state.PendingPath {
		return nil
	}
	if item.State != WorkItemStateCompleted {
		return fmt.Errorf(
			"Workflow WorkItem %q did not complete successfully",
			workID,
		)
	}

	stepID, found := strings.CutPrefix(item.ProgramPath, "step/")
	if !found || strings.TrimSpace(stepID) == "" {
		return errors.New("Workflow WorkItem has an invalid program path")
	}

	switch item.Type {
	case WorkItemTypeStateUpdate:
		if item.Input.StateUpdate == nil {
			return errors.New("Workflow StateUpdate WorkItem has no payload")
		}
		if err := applyWorkflowState(
			state,
			*item.Input.StateUpdate,
		); err != nil {
			return err
		}
		state.Outputs[stepID] = state.State.Clone()

	default:
		value, err := workflowWorkValue(*item)
		if err != nil {
			return err
		}
		state.Outputs[stepID] = value
	}

	state.PendingPath = ""
	state.Index++
	return nil
}

func (i *WorkflowInterpreter) settleChild(
	state *workflowProgramState,
	execution Execution,
	childID ProgramFrameID,
) error {
	if state == nil {
		return errors.New("Workflow state is unavailable")
	}

	for _, child := range execution.ProgramFrames {
		if child.ID != childID {
			continue
		}
		if child.Path != state.PendingPath {
			return nil
		}
		if child.Status != ProgramFrameStateCompleted {
			return errors.New("nested workflow ProgramFrame did not complete")
		}

		stepID, found := strings.CutPrefix(child.Path, "step/")
		if !found || strings.TrimSpace(stepID) == "" {
			return errors.New("nested workflow ProgramFrame has an invalid path")
		}
		state.Outputs[stepID] = child.Output.Clone()
		state.PendingPath = ""
		state.Index++
		return nil
	}
	return errors.New("nested workflow ProgramFrame is unavailable")
}

func (i *WorkflowInterpreter) evaluate(
	value WorkflowValue,
	program WorkflowProgram,
	state workflowProgramState,
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
		result, _, err := executable.Eval(
			workflowActivation(program, state),
		)
		if err != nil {
			return nil, fmt.Errorf(
				"evaluate Workflow expression: %w",
				err,
			)
		}
		return canonicalWorkflowValue(result.Value())

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
		return nil, errors.New("Workflow value has an unsupported kind")
	}
}

func (i *WorkflowInterpreter) condition(
	value WorkflowValue,
	program WorkflowProgram,
	state workflowProgramState,
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
	state workflowProgramState,
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
	if len(p.InputSchema) == 0 ||
		len(p.OutputSchema) == 0 ||
		!p.InputSchema.Valid() ||
		!p.OutputSchema.Valid() ||
		!p.Input.Valid() {
		return errors.New("Workflow Program schemas or input are invalid")
	}
	if err := p.Root.Validate(); err != nil {
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
	return validateWorkflowSchema(p.InputSchema, p.Input)
}

func (b WorkflowBlock) Validate() error {
	if err := b.Output.Validate(); err != nil {
		return err
	}
	if b.State != nil {
		if !b.State.Schema.Valid() ||
			len(b.State.Schema) == 0 {
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

func (s WorkflowStep) Validate() error {
	count := 0
	for _, present := range []bool{
		s.Call != nil,
		s.Transform != nil,
		s.Set != nil,
		s.Return != nil,
		s.Fail != nil,
	} {
		if present {
			count++
		}
	}
	if count != 1 {
		return errors.New("Workflow step must contain exactly one supported operation")
	}
	if s.When != nil {
		if err := s.When.Validate(); err != nil {
			return err
		}
	}
	if len(s.OutputSchema) != 0 && !s.OutputSchema.Valid() {
		return errors.New("Workflow step output schema is invalid")
	}
	if s.Call != nil {
		return s.Call.Validate()
	}
	if s.Transform != nil {
		return s.Transform.Validate()
	}
	if s.Set != nil {
		return s.Set.Validate()
	}
	if s.Return != nil {
		return s.Return.Value.Validate()
	}
	if s.Fail != nil {
		if strings.TrimSpace(s.Fail.Code) == "" {
			return errors.New("Workflow fail code is required")
		}
		return s.Fail.Message.Validate()
	}
	return nil
}

func (c WorkflowCall) Validate() error {
	if err := c.Target.Validate(); err != nil {
		return err
	}
	if err := c.Input.Validate(); err != nil {
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
			if strings.TrimSpace(value.Key) == "" {
				return errors.New("Workflow object key is empty")
			}
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

func (t WorkflowTarget) Validate() error {
	switch t.Kind {
	case WorkflowTargetTool:
		if t.Artifact == nil ||
			strings.TrimSpace(t.ToolName) == "" ||
			t.MCP != nil {
			return errors.New("Workflow Tool target is invalid")
		}
		return validatePin(*t.Artifact)

	case WorkflowTargetMCPTool:
		if t.MCP == nil || t.Artifact != nil {
			return errors.New("Workflow MCP Tool target is invalid")
		}
		return t.MCP.Validate()

	case WorkflowTargetService:
		if strings.TrimSpace(t.ServiceID) == "" ||
			strings.TrimSpace(t.BindingVersion) == "" {
			return errors.New("Workflow Service target is invalid")
		}
		return nil

	case WorkflowTargetActivity:
		if strings.TrimSpace(t.ActivityID) == "" ||
			strings.TrimSpace(t.BindingVersion) == "" {
			return errors.New("Workflow Activity target is invalid")
		}
		return nil

	case WorkflowTargetWorkflow:
		if t.NestedProgram == nil {
			return errors.New("Workflow nested target is invalid")
		}
		return nil

	case WorkflowTargetAgent,
		WorkflowTargetModel:
		if t.Artifact == nil {
			return errors.New("Workflow Artifact target is invalid")
		}
		return validatePin(*t.Artifact)

	default:
		return errors.New("Workflow target kind is unsupported")
	}
}

func (m WorkflowMCP) Validate() error {
	if strings.TrimSpace(m.ToolName) == "" {
		return errors.New("Workflow MCP Tool name is required")
	}
	return validatePin(m.Server)
}

func (s workflowProgramState) Validate() error {
	if s.Version != WorkflowInterpreterVersion ||
		s.Index < 0 ||
		!s.State.Valid() ||
		s.Outputs == nil {
		return errors.New("Workflow Program state is invalid")
	}
	for key, value := range s.Outputs {
		if strings.TrimSpace(key) == "" || !value.Valid() {
			return errors.New("Workflow Program output state is invalid")
		}
	}
	return nil
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
) (WorkflowProgram, workflowProgramState, error) {
	program, err := workflowProgramFromSnapshot(frame.Program)
	if err != nil {
		return WorkflowProgram{}, workflowProgramState{}, err
	}

	state, err := DecodeCanonicalValue[workflowProgramState](
		frame.State.Data,
	)
	if err != nil {
		return WorkflowProgram{}, workflowProgramState{}, err
	}
	if err := state.Validate(); err != nil {
		return WorkflowProgram{}, workflowProgramState{}, err
	}
	return program, state, nil
}

func workflowTransition(
	state workflowProgramState,
	status *ProgramFrameState,
	output CanonicalValue,
	work []WorkItemIntent,
) (ProgramTransition, error) {
	next, err := encodeWorkflowProgramState(state)
	if err != nil {
		return ProgramTransition{}, err
	}
	return ProgramTransition{
		State:     next,
		Status:    status,
		Output:    output.Clone(),
		WorkItems: work,
	}, nil
}

func workflowFailure(
	state workflowProgramState,
	code string,
	message string,
) (ProgramTransition, error) {
	status := ProgramFrameStateFailed
	next, err := encodeWorkflowProgramState(state)
	if err != nil {
		return ProgramTransition{}, err
	}
	return ProgramTransition{
		State:  next,
		Status: &status,
		Error: &SanitizedError{
			Code:    firstNonEmpty(code, "workflow_failed"),
			Message: firstNonEmpty(message, "Workflow execution failed"),
		},
	}, nil
}

func workflowWork(
	state workflowProgramState,
	path string,
	intent WorkItemIntent,
) (ProgramTransition, error) {
	state.PendingPath = path
	return workflowTransition(
		state,
		nil,
		nil,
		[]WorkItemIntent{intent},
	)
}

func encodeWorkflowProgramState(
	state workflowProgramState,
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
	state workflowProgramState,
) map[string]any {
	input, _ := DecodeCanonicalValue[any](program.Input)
	currentState, _ := DecodeCanonicalValue[any](state.State)

	steps := make(map[string]any, len(state.Outputs))
	for name, output := range state.Outputs {
		value, _ := DecodeCanonicalValue[any](output)
		steps[name] = map[string]any{
			"output": value,
		}
	}

	return map[string]any{
		"input":     input,
		"state":     currentState,
		"steps":     steps,
		"context":   nil,
		"messages":  nil,
		"item":      nil,
		"index":     nil,
		"result":    nil,
		"event":     nil,
		"error":     nil,
		"outcome":   nil,
		"completed": nil,
		"operation": nil,
	}
}

func applyWorkflowState(
	state *workflowProgramState,
	update StateUpdateWork,
) error {
	if state == nil {
		return errors.New("Workflow state is unavailable")
	}

	values, err := DecodeCanonicalValue[map[string]CanonicalValue](
		state.State,
	)
	if err != nil {
		return err
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

	next, err := canonicalWorkflowValue(values)
	if err != nil {
		return err
	}
	state.State = next
	return nil
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
