package workflowruntime

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

type Interpreter struct{}

func NewInterpreter() *Interpreter   { return &Interpreter{} }
func (*Interpreter) ID() string      { return InterpreterID }
func (*Interpreter) Version() uint32 { return InterpreterVersion }
func (*Interpreter) ValidateSnapshot(snapshot sessionv3.ProgramSnapshot) error {
	if snapshot.InterpreterID != InterpreterID || snapshot.InterpreterVersion != InterpreterVersion {
		return errors.New("unsupported Workflow interpreter version")
	}
	var p Program
	if err := json.Unmarshal(snapshot.Payload, &p); err != nil {
		return err
	}
	return validateProgram(p)
}

func (*Interpreter) ValidateState(state sessionv3.ProgramState) error {
	if state.Type != InterpreterID || state.Version != InterpreterVersion {
		return errors.New("unsupported Workflow state version")
	}
	var s State
	if err := json.Unmarshal(state.Data, &s); err != nil {
		return err
	}
	return s.Validate()
}

func (*Interpreter) InitialState(snapshot sessionv3.ProgramSnapshot) (sessionv3.ProgramState, error) {
	var p Program
	if err := json.Unmarshal(snapshot.Payload, &p); err != nil {
		return sessionv3.ProgramState{}, err
	}
	s := State{
		Version:   InterpreterVersion,
		Input:     p.Input.Clone(),
		Index:     0,
		Outputs:   map[string]sessionv3.CanonicalValue{},
		Pending:   "",
		Variables: sessionv3.MustCanonicalValue(map[string]any{}),
	}
	return encodeState(s), nil
}

func (i *Interpreter) Advance(
	frame sessionv3.ProgramFrame,
	execution sessionv3.Execution,
	event sessionv3.ProgramEvent,
) (sessionv3.ProgramTransition, error) {
	p, s, err := decode(frame)
	if err != nil {
		return sessionv3.ProgramTransition{}, err
	}
	switch event.Kind {
	case sessionv3.ProgramEventCancelled:
		status := sessionv3.ProgramFrameStateCancelled
		return sessionv3.ProgramTransition{State: encodeState(s), Status: &status}, nil
	case sessionv3.ProgramEventStarted, sessionv3.ProgramEventContinue:
		// No state change; continue to drive below.
	case sessionv3.ProgramEventWorkSettled:
		if err := s.settleWork(execution, event.WorkItemID); err != nil {
			return failure(s, err)
		}
	case sessionv3.ProgramEventChildSettled:
		if err := s.settleChild(execution, event.ChildFrameID); err != nil {
			return failure(s, err)
		}
	case sessionv3.ProgramEventActivity:
		if err := s.addSignal(execution, event.ActivityID); err != nil {
			return failure(s, err)
		}
	default:
	}
	return i.drive(frame, execution, p, s)
}

func (i *Interpreter) drive(
	frame sessionv3.ProgramFrame,
	execution sessionv3.Execution,
	p Program,
	s State,
) (sessionv3.ProgramTransition, error) {
	if s.Pending != "" {
		return sessionv3.ProgramTransition{State: encodeState(s)}, nil
	}
	if s.Index >= len(p.Document.Steps) {
		out := p.Document.Output
		if len(out) == 0 {
			out = workflowv1.Value(`null`)
		}
		status := sessionv3.ProgramFrameStateCompleted
		return sessionv3.ProgramTransition{
			State:  encodeState(s),
			Status: &status,
			Output: sessionv3.CanonicalValue(out),
			Activities: []sessionv3.Activity{
				{
					Kind:       sessionv3.ActivityKindWorkflow,
					Visibility: sessionv3.ActivityVisibilityBoth,
					Workflow: &sessionv3.WorkflowActivity{
						Phase:   "completed",
						Summary: "Workflow completed.",
						Output:  sessionv3.CanonicalValue(out),
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
		status := sessionv3.ProgramFrameStateCompleted
		return sessionv3.ProgramTransition{
			State:  encodeState(s),
			Status: &status,
			Output: sessionv3.CanonicalValue(fields["return"]),
		}, nil
	case fields["fail"] != nil:
		return failure(s, errors.New("workflow fail step"))
	default:

	}
	return failure(s, fmt.Errorf("unsupported Workflow step %q", step.ID))
}

func (i *Interpreter) call(
	frame sessionv3.ProgramFrame,
	path string,
	p Program,
	raw json.RawMessage,
	s State,
) (sessionv3.ProgramTransition, error) {
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
	input := sessionv3.CanonicalValue(call.Input)
	switch target.Kind {
	case workflowv1.TargetAgent:
		activityID := sessionv3.ActivityID(frame.ID + ":" + sessionv3.ProgramFrameID(path) + ":input")
		program, err := sessionv3.NewAgentLoopProgram(
			sessionv3.SessionDefaults{
				Agent: &sessionv3.ArtifactSelection{
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
		activityID := sessionv3.ActivityID(frame.ID + ":" + sessionv3.ProgramFrameID(path) + ":input")
		program, err := sessionv3.NewSingleModelProgram(
			sessionv3.SessionDefaults{
				Model: &sessionv3.ArtifactSelection{
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
		return childTransition(path, *target.NestedProgram, "", input, s)
	case workflowv1.TargetTool:
		if target.Artifact == nil {
			return failure(s, errors.New("workflow tool target is not pinned"))
		}
		return i.work(
			path,
			sessionv3.WorkItemTypeTool,
			sessionv3.EffectContract{Kind: sessionv3.EffectKindNonIdempotent},
			sessionv3.WorkItemPayload{
				Tool: &sessionv3.ToolWork{
					Call: sessionv3.ToolCall{
						ID:        frame.ID.String() + ":" + path,
						Name:      target.Name,
						Arguments: input,
						Binding: sessionv3.ToolBinding{
							Kind:       sessionv3.ToolBindingKindUnresolved,
							Unresolved: &sessionv3.UnresolvedToolBinding{Reason: "Workflow Tool Artifact binding"},
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
			sessionv3.WorkItemTypeService,
			sessionv3.EffectContract{Kind: sessionv3.EffectKindNonIdempotent},
			sessionv3.WorkItemPayload{
				Service: &sessionv3.ServiceWork{
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
			sessionv3.WorkItemTypeActivity,
			sessionv3.EffectContract{Kind: sessionv3.EffectKindNonIdempotent},
			sessionv3.WorkItemPayload{
				Activity: &sessionv3.ActivityWork{
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
	kind sessionv3.WorkItemType,
	effect sessionv3.EffectContract,
	input sessionv3.WorkItemPayload,
	s State,
) (sessionv3.ProgramTransition, error) {
	s.Pending = path
	return sessionv3.ProgramTransition{
		State: encodeState(s),
		WorkItems: []sessionv3.WorkItemIntent{
			{ProgramPath: path, Type: kind, Automatic: true, Effect: effect, Input: input},
		},
	}, nil
}

func childTransition(
	path string,
	program sessionv3.ProgramSnapshot,
	activityID sessionv3.ActivityID,
	input sessionv3.CanonicalValue,
	s State,
) (sessionv3.ProgramTransition, error) {
	s.Pending = path
	activities := []sessionv3.Activity{}
	if activityID != "" {
		activities = append(
			activities,
			sessionv3.Activity{
				ID:         activityID,
				Kind:       sessionv3.ActivityKindMessage,
				Visibility: sessionv3.ActivityVisibilityAudit,
				Message:    &sessionv3.MessageActivity{Role: sessionv3.MessageRoleUser, Text: string(input)},
			},
		)
	}
	return sessionv3.ProgramTransition{
		State:      encodeState(s),
		Activities: activities,
		Frames:     []sessionv3.ProgramFrameIntent{{Path: path, Program: program}},
	}, nil
}

type State struct {
	Version   uint32                              `json:"version"`
	Input     sessionv3.CanonicalValue            `json:"input"`
	Index     int                                 `json:"index"`
	Pending   string                              `json:"pending,omitempty"`
	Outputs   map[string]sessionv3.CanonicalValue `json:"outputs"`
	Variables sessionv3.CanonicalValue            `json:"variables"`
}

func (s *State) Validate() error {
	if s.Version != InterpreterVersion || !s.Input.Valid() || !s.Variables.Valid() || s.Outputs == nil {
		return errors.New("invalid Workflow interpreter state")
	}
	return nil
}

func encodeState(s State) sessionv3.ProgramState {
	return sessionv3.ProgramState{
		Type:    InterpreterID,
		Version: InterpreterVersion,
		Data:    sessionv3.MustCanonicalValue(s),
	}
}

func decode(frame sessionv3.ProgramFrame) (Program, State, error) {
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

func (s *State) settleWork(e sessionv3.Execution, id sessionv3.WorkItemID) error {
	item := work(e, id)
	if item == nil {
		return errors.New("workflow WorkItem is unavailable")
	}
	if s.Pending != "" && item.ProgramPath != s.Pending {
		return nil
	}
	if item.State == sessionv3.WorkItemStateFailed || item.State == sessionv3.WorkItemStateUncertain ||
		item.State == sessionv3.WorkItemStateBlocked {
		return fmt.Errorf("workflow WorkItem %q failed", id)
	}
	if item.Output != nil {
		s.Outputs[item.ProgramPath] = item.Output.Value.Clone()
	}
	s.Pending = ""
	s.Index++
	return nil
}

func (s *State) settleChild(e sessionv3.Execution, id sessionv3.ProgramFrameID) error {
	for _, frame := range e.ProgramFrames {
		if frame.ID == id {
			if frame.Status == sessionv3.ProgramFrameStateFailed ||
				frame.Status == sessionv3.ProgramFrameStateCancelled {
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
func (s *State) addSignal(e sessionv3.Execution, id sessionv3.ActivityID) error { return nil }
func work(e sessionv3.Execution, id sessionv3.WorkItemID) *sessionv3.WorkItem {
	for _, v := range e.WorkItems {
		if v.ID == id {
			itemCopy := v
			return &itemCopy
		}
	}
	return nil
}

func failure(s State, err error) (sessionv3.ProgramTransition, error) {
	status := sessionv3.ProgramFrameStateFailed
	return sessionv3.ProgramTransition{
		State:  encodeState(s),
		Status: &status,
		Error:  &sessionv3.SanitizedError{Code: "workflow_execution_failed", Message: err.Error()},
	}, nil
}
