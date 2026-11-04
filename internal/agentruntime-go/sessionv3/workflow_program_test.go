package sessionv3

import "testing"

func TestWorkflowTemplateBindsInputOnlyForChildInvocation(t *testing.T) {
	program := WorkflowProgram{
		Version:      WorkflowInterpreterVersion,
		InputSchema:  MustCanonicalValue(map[string]any{"type": "object"}),
		OutputSchema: MustCanonicalValue(true),
		Root: WorkflowBlock{
			Steps: []WorkflowStep{},
			Output: WorkflowValue{
				Kind:    WorkflowValueLiteral,
				Literal: MustCanonicalValue(nil),
			},
		},
		Targets: map[string]WorkflowTarget{},
	}
	if err := program.Validate(); err != nil {
		t.Fatal(err)
	}

	snapshot := ProgramSnapshot{
		InterpreterID:      WorkflowInterpreterID,
		InterpreterVersion: WorkflowInterpreterVersion,
		Artifact: &ArtifactPin{
			RootID:     "root",
			ArtifactID: "workflow",
			Revision:   1,
			Digest:     "sha256:test",
		},
		DefinitionDigest: "sha256:test",
		Payload:          MustCanonicalValue(program),
	}

	interpreter, err := NewWorkflowInterpreter()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := interpreter.InitialState(snapshot); err == nil {
		t.Fatal("unbound nested Workflow template was accepted as an executable frame")
	}

	bound, err := workflowSnapshotWithInput(
		snapshot,
		MustCanonicalValue(map[string]any{"request": "value"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := interpreter.InitialState(bound); err != nil {
		t.Fatalf("bound nested Workflow input was rejected: %v", err)
	}
}

func TestWorkflowStateUpdateProducesValidatedState(t *testing.T) {
	request := WorkDispatchRequest{
		WorkItem: WorkItem{
			ID:      "state-update",
			Type:    WorkItemTypeStateUpdate,
			FrameID: "frame",
			Input: WorkItemPayload{
				StateUpdate: &StateUpdateWork{
					Base: MustCanonicalValue(map[string]any{
						"attempts": 1,
						"obsolete": true,
					}),
					Schema: MustCanonicalValue(map[string]any{
						"type": "object",
						"required": []string{
							"attempts",
						},
						"properties": map[string]any{
							"attempts": map[string]any{
								"type":    "integer",
								"minimum": 0,
							},
						},
						"additionalProperties": false,
					}),
					Assignments: []StateAssignment{{
						Path:  "attempts",
						Value: MustCanonicalValue(2),
					}},
					RemovePaths: []string{"obsolete"},
				},
			},
		},
	}

	result, handled, err := (DeterministicExecutor{}).Execute(
		t.Context(),
		request,
		BoundWork{Private: deterministicBinding{}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !handled || result.Output == nil {
		t.Fatal("StateUpdate WorkItem was not handled")
	}

	value, err := DecodeCanonicalValue[map[string]int](result.Output.Value)
	if err != nil {
		t.Fatal(err)
	}
	if value["attempts"] != 2 || len(value) != 1 {
		t.Fatalf("unexpected updated state: %#v", value)
	}
}
