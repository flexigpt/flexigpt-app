package workflowruntime

import (
	"encoding/json"
	"errors"
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

type Interpreter struct{}

func NewInterpreter() *Interpreter   { return &Interpreter{} }
func (*Interpreter) ID() string      { return InterpreterID }
func (*Interpreter) Version() uint32 { return InterpreterVersion }
func (*Interpreter) ValidateSnapshot(snapshot sessionv3Runtime.ProgramSnapshot) error {
	if snapshot.InterpreterID != InterpreterID || snapshot.InterpreterVersion != InterpreterVersion {
		return errors.New("unsupported Workflow interpreter version")
	}
	var p Program
	if err := json.Unmarshal(snapshot.Payload, &p); err != nil {
		return err
	}
	return validateProgram(p)
}

func (*Interpreter) ValidateState(state sessionv3Runtime.ProgramState) error {
	if state.Type != InterpreterID || state.Version != InterpreterVersion {
		return errors.New("unsupported Workflow state version")
	}
	var s State
	if err := json.Unmarshal(state.Data, &s); err != nil {
		return err
	}
	return s.Validate()
}

func (*Interpreter) InitialState(snapshot sessionv3Runtime.ProgramSnapshot) (sessionv3Runtime.ProgramState, error) {
	var p Program
	if err := json.Unmarshal(snapshot.Payload, &p); err != nil {
		return sessionv3Runtime.ProgramState{}, err
	}
	s := State{
		Version:   InterpreterVersion,
		Input:     p.Input.Clone(),
		Index:     0,
		Outputs:   map[string]sessionv3Runtime.CanonicalValue{},
		Pending:   "",
		Variables: sessionv3Runtime.MustCanonicalValue(map[string]any{}),
	}
	return encodeState(s), nil
}

func (i *Interpreter) Advance(
	frame sessionv3Runtime.ProgramFrame,
	execution sessionv3Runtime.Execution,
	event sessionv3Runtime.ProgramEvent,
) (sessionv3Runtime.ProgramTransition, error) {
	p, s, err := decode(frame)
	if err != nil {
		return sessionv3Runtime.ProgramTransition{}, err
	}
	switch event.Kind {
	case sessionv3Runtime.ProgramEventCancelled:
		status := sessionv3Runtime.ProgramFrameStateCancelled
		return sessionv3Runtime.ProgramTransition{State: encodeState(s), Status: &status}, nil
	case sessionv3Runtime.ProgramEventStarted, sessionv3Runtime.ProgramEventContinue:
		// No state change; continue to drive below.
	case sessionv3Runtime.ProgramEventWorkSettled:
		if err := s.settleWork(execution, event.WorkItemID); err != nil {
			return failure(s, err)
		}
	case sessionv3Runtime.ProgramEventChildSettled:
		if err := s.settleChild(execution, event.ChildFrameID); err != nil {
			return failure(s, err)
		}
	case sessionv3Runtime.ProgramEventActivity:
		if err := s.addSignal(execution, event.ActivityID); err != nil {
			return failure(s, err)
		}
	default:
	}
	return i.drive(frame, execution, p, s)
}

func (i *Interpreter) drive(
	frame sessionv3Runtime.ProgramFrame,
	execution sessionv3Runtime.Execution,
	p Program,
	s State,
) (sessionv3Runtime.ProgramTransition, error) {
	if s.Pending != "" {
		return sessionv3Runtime.ProgramTransition{State: encodeState(s)}, nil
	}
	if s.Index >= len(p.Document.Steps) {
		out := p.Document.Output
		if len(out) == 0 {
			out = workflowv1.Value(`null`)
		}
		status := sessionv3Runtime.ProgramFrameStateCompleted
		return sessionv3Runtime.ProgramTransition{
			State:  encodeState(s),
			Status: &status,
			Output: sessionv3Runtime.CanonicalValue(out),
			Activities: []sessionv3Runtime.Activity{
				{
					Kind:       sessionv3Runtime.ActivityKindWorkflow,
					Visibility: sessionv3Runtime.ActivityVisibilityBoth,
					Workflow: &sessionv3Runtime.WorkflowActivity{
						Phase:   "completed",
						Summary: "Workflow completed.",
						Output:  sessionv3Runtime.CanonicalValue(out),
					},
				},
			},
		}, nil
	}
	step := p.Document.Steps[s.Index]
	path := "step/" + step.ID
	raw, err := json.Marshal(step)
	if err != nil {
		return failure(s, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return failure(s, err)
	}
	if when, ok := fields["when"]; ok &&
		string(when) != "null" &&
		string(when) != "true" {
		// The current interpreter has no persisted branch-decision state.
		// Do not run the body after a condition as the old implementation did.
		return failure(
			s,
			fmt.Errorf(
				"workflow step %q uses conditional execution that is not implemented by workflow runtime v1",
				step.ID,
			),
		)
	}
	if call, ok := fields["call"]; ok {
		return i.call(frame, path, p, call, s)
	}
	switch {
	case fields["transform"] != nil:
	case fields["set"] != nil:
	case fields["wait"] != nil:
	case fields["emit"] != nil:
	case fields["parallel"] != nil || fields["foreach"] != nil:
	case fields["choose"] != nil || fields["repeat"] != nil || fields["machine"] != nil:
	case fields["saga"] != nil:
	case fields["scope"] != nil:
	case fields["cancel"] != nil:
		return failure(
			s,
			fmt.Errorf(
				"workflow step %q uses an operation that is represented by the contract but not executable by workflow runtime v1",
				step.ID,
			),
		)
	case fields["return"] != nil:
		status := sessionv3Runtime.ProgramFrameStateCompleted
		return sessionv3Runtime.ProgramTransition{
			State:  encodeState(s),
			Status: &status,
			Output: sessionv3Runtime.CanonicalValue(fields["return"]),
		}, nil
	case fields["fail"] != nil:
		return failure(s, errors.New("workflow fail step"))
	default:

	}
	return failure(s, fmt.Errorf("unsupported Workflow step %q", step.ID))
}

func (i *Interpreter) call(
	frame sessionv3Runtime.ProgramFrame,
	path string,
	p Program,
	raw json.RawMessage,
	s State,
) (sessionv3Runtime.ProgramTransition, error) {
	var call workflowv1.Call
	if err := json.Unmarshal(raw, &call); err != nil {
		return failure(s, err)
	}
	var name string
	if err := json.Unmarshal(call.Target, &name); err != nil {
		return failure(s, errors.New("dynamic Workflow target selection requires resolved target set"))
	}
	target, ok := p.Targets[name]
	if !ok {
		return failure(s, fmt.Errorf("workflow target %q is unavailable", name))
	}
	input := sessionv3Runtime.CanonicalValue(call.Input)
	switch target.Kind {
	case workflowv1.TargetAgent:
		activityID := sessionv3Runtime.ActivityID(frame.ID + ":" + sessionv3Runtime.ProgramFrameID(path) + ":input")
		program, err := sessionv3Runtime.NewAgentLoopProgram(
			sessionv3Runtime.SessionDefaults{
				Agent: &sessionv3Runtime.ArtifactSelection{
					RootID:           target.Artifact.RootID,
					ArtifactID:       target.Artifact.ArtifactID,
					ExpectedRevision: target.Artifact.Revision,
					ExpectedDigest:   target.Artifact.Digest,
				},
			},
			activityID,
		)
		if err != nil {
			return failure(s, err)
		}
		return childTransition(path, program, activityID, input, s)
	case workflowv1.TargetModel:
		activityID := sessionv3Runtime.ActivityID(frame.ID + ":" + sessionv3Runtime.ProgramFrameID(path) + ":input")
		program, err := sessionv3Runtime.NewSingleModelProgram(
			sessionv3Runtime.SessionDefaults{
				Model: &sessionv3Runtime.ArtifactSelection{
					RootID:           target.Artifact.RootID,
					ArtifactID:       target.Artifact.ArtifactID,
					ExpectedRevision: target.Artifact.Revision,
					ExpectedDigest:   target.Artifact.Digest,
				},
			},
			activityID,
		)
		if err != nil {
			return failure(s, err)
		}
		return childTransition(path, program, activityID, input, s)
	case workflowv1.TargetWorkflow:
		if target.NestedProgram == nil {
			return failure(s, errors.New("nested workflow programFrame is unavailable"))
		}
		nested, err := programWithInput(*target.NestedProgram, input)
		if err != nil {
			return failure(s, err)
		}
		return childTransition(path, nested, "", input, s)
	case workflowv1.TargetTool:
		if target.Artifact == nil {
			return failure(s, errors.New("workflow tool target is not pinned"))
		}
		return i.work(
			path,
			sessionv3Runtime.WorkItemTypeTool,
			sessionv3Runtime.EffectContract{Kind: sessionv3Runtime.EffectKindNonIdempotent},
			sessionv3Runtime.WorkItemPayload{
				Tool: &sessionv3Runtime.ToolWork{
					Call: sessionv3Runtime.ToolCall{
						ID:        frame.ID.String() + ":" + path,
						Name:      target.Name,
						Arguments: input,
						Binding: sessionv3Runtime.ToolBinding{
							Kind: sessionv3Runtime.ToolBindingKindUnresolved,
							Unresolved: &sessionv3Runtime.UnresolvedToolBinding{
								Reason: "Workflow Tool Artifact binding",
							},
						},
					},
					Target: target.Artifact,
				},
			},
			s,
		)
	case workflowv1.TargetService:
		return i.work(
			path,
			sessionv3Runtime.WorkItemTypeService,
			sessionv3Runtime.EffectContract{Kind: sessionv3Runtime.EffectKindNonIdempotent},
			sessionv3Runtime.WorkItemPayload{
				Service: &sessionv3Runtime.ServiceWork{
					ServiceID:      target.ServiceID,
					BindingVersion: target.BindingVersion,
					Input:          input,
				},
			},
			s,
		)
	case workflowv1.TargetActivity:
		return i.work(
			path,
			sessionv3Runtime.WorkItemTypeActivity,
			sessionv3Runtime.EffectContract{Kind: sessionv3Runtime.EffectKindNonIdempotent},
			sessionv3Runtime.WorkItemPayload{
				Activity: &sessionv3Runtime.ActivityWork{
					ActivityID:     target.ActivityID,
					BindingVersion: target.BindingVersion,
					Input:          input,
				},
			},
			s,
		)
	default:
		return failure(s, errors.New("unsupported Workflow call target"))
	}
}

func (i *Interpreter) work(
	path string,
	kind sessionv3Runtime.WorkItemType,
	effect sessionv3Runtime.EffectContract,
	input sessionv3Runtime.WorkItemPayload,
	s State,
) (sessionv3Runtime.ProgramTransition, error) {
	s.Pending = path
	return sessionv3Runtime.ProgramTransition{
		State: encodeState(s),
		WorkItems: []sessionv3Runtime.WorkItemIntent{
			{ProgramPath: path, Type: kind, Automatic: true, Effect: effect, Input: input},
		},
	}, nil
}

func childTransition(
	path string,
	program sessionv3Runtime.ProgramSnapshot,
	activityID sessionv3Runtime.ActivityID,
	input sessionv3Runtime.CanonicalValue,
	s State,
) (sessionv3Runtime.ProgramTransition, error) {
	s.Pending = path
	activities := []sessionv3Runtime.Activity{}
	if activityID != "" {
		activities = append(
			activities,
			sessionv3Runtime.Activity{
				ID:         activityID,
				Kind:       sessionv3Runtime.ActivityKindMessage,
				Visibility: sessionv3Runtime.ActivityVisibilityAudit,
				Message: &sessionv3Runtime.MessageActivity{
					Role: sessionv3Runtime.MessageRoleUser,
					Text: string(input),
				},
			},
		)
	}
	return sessionv3Runtime.ProgramTransition{
		State:      encodeState(s),
		Activities: activities,
		Frames:     []sessionv3Runtime.ProgramFrameIntent{{Path: path, Program: program}},
	}, nil
}

func programWithInput(
	snapshot sessionv3Runtime.ProgramSnapshot,
	input sessionv3Runtime.CanonicalValue,
) (sessionv3Runtime.ProgramSnapshot, error) {
	if !input.Valid() {
		return sessionv3Runtime.ProgramSnapshot{}, errors.New(
			"nested workflow input is not canonical JSON",
		)
	}

	var program Program
	if err := json.Unmarshal(snapshot.Payload, &program); err != nil {
		return sessionv3Runtime.ProgramSnapshot{}, err
	}
	program.Input = input.Clone()
	if err := validateProgram(program); err != nil {
		return sessionv3Runtime.ProgramSnapshot{}, err
	}
	snapshot.Payload = sessionv3Runtime.MustCanonicalValue(program)
	return snapshot, nil
}

type State struct {
	Version   uint32                                     `json:"version"`
	Input     sessionv3Runtime.CanonicalValue            `json:"input"`
	Index     int                                        `json:"index"`
	Pending   string                                     `json:"pending,omitempty"`
	Outputs   map[string]sessionv3Runtime.CanonicalValue `json:"outputs"`
	Variables sessionv3Runtime.CanonicalValue            `json:"variables"`
}

func (s *State) Validate() error {
	if s.Version != InterpreterVersion || !s.Input.Valid() || !s.Variables.Valid() || s.Outputs == nil {
		return errors.New("invalid Workflow interpreter state")
	}
	return nil
}

func encodeState(s State) sessionv3Runtime.ProgramState {
	return sessionv3Runtime.ProgramState{
		Type:    InterpreterID,
		Version: InterpreterVersion,
		Data:    sessionv3Runtime.MustCanonicalValue(s),
	}
}

func decode(frame sessionv3Runtime.ProgramFrame) (Program, State, error) {
	var p Program
	if err := json.Unmarshal(frame.Program.Payload, &p); err != nil {
		return Program{}, State{}, err
	}
	if err := validateProgram(p); err != nil {
		return Program{}, State{}, err
	}
	var s State
	if err := json.Unmarshal(frame.State.Data, &s); err != nil {
		return Program{}, State{}, err
	}
	if err := s.Validate(); err != nil {
		return Program{}, State{}, err
	}
	return p, s, nil
}

func (s *State) settleWork(e sessionv3Runtime.Execution, id sessionv3Runtime.WorkItemID) error {
	item := work(e, id)
	if item == nil {
		return errors.New("workflow WorkItem is unavailable")
	}
	if s.Pending != "" && item.ProgramPath != s.Pending {
		return nil
	}
	if item.State != sessionv3Runtime.WorkItemStateCompleted {
		return fmt.Errorf(
			"workflow WorkItem %q did not complete successfully: %s",
			id,
			item.State,
		)
	}
	if item.Output != nil {
		s.Outputs[item.ProgramPath] = item.Output.Value.Clone()
	}
	s.Pending = ""
	s.Index++
	return nil
}

func (s *State) settleChild(e sessionv3Runtime.Execution, id sessionv3Runtime.ProgramFrameID) error {
	for _, frame := range e.ProgramFrames {
		if frame.ID == id {
			if frame.Status == sessionv3Runtime.ProgramFrameStateFailed ||
				frame.Status == sessionv3Runtime.ProgramFrameStateCancelled {
				return errors.New("workflow child ProgramFrame failed")
			}
			if s.Pending != "" && frame.Path == s.Pending {
				s.Outputs[frame.Path] = frame.Output.Clone()
				s.Pending = ""
				s.Index++
			}
			return nil
		}
	}
	return errors.New("workflow child ProgramFrame is unavailable")
}

func (s *State) addSignal(e sessionv3Runtime.Execution, id sessionv3Runtime.ActivityID) error {
	return nil
}

func work(e sessionv3Runtime.Execution, id sessionv3Runtime.WorkItemID) *sessionv3Runtime.WorkItem {
	for _, v := range e.WorkItems {
		if v.ID == id {
			itemCopy := v
			return &itemCopy
		}
	}
	return nil
}

func failure(s State, err error) (sessionv3Runtime.ProgramTransition, error) {
	status := sessionv3Runtime.ProgramFrameStateFailed
	return sessionv3Runtime.ProgramTransition{
		State:  encodeState(s),
		Status: &status,
		Error:  &sessionv3Runtime.SanitizedError{Code: "workflow_execution_failed", Message: err.Error()},
	}, nil
}
