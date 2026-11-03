package sessionv3

import "testing"

func TestWorkflowProgramRejectsDuplicateSteps(t *testing.T) {
	t.Parallel()

	program := WorkflowProgram{
		Version:      WorkflowInterpreterVersion,
		InputSchema:  MustCanonicalValue(true),
		OutputSchema: MustCanonicalValue(true),
		Input:        MustCanonicalValue(map[string]any{}),
		Root: WorkflowBlock{
			Steps: []WorkflowStep{
				{
					ID: "duplicate",
					Return: &WorkflowReturn{
						Value: WorkflowValue{
							Kind:    WorkflowValueLiteral,
							Literal: MustCanonicalValue(nil),
						},
					},
				},
				{
					ID: "duplicate",
					Return: &WorkflowReturn{
						Value: WorkflowValue{
							Kind:    WorkflowValueLiteral,
							Literal: MustCanonicalValue(nil),
						},
					},
				},
			},
			Output: WorkflowValue{
				Kind:    WorkflowValueLiteral,
				Literal: MustCanonicalValue(nil),
			},
		},
		Targets: map[string]WorkflowTarget{},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("Workflow Program accepted duplicate step IDs")
	}
}

func TestWorkflowValueLiteralDoesNotEvaluateDirectiveData(t *testing.T) {
	t.Parallel()

	interpreter, err := NewWorkflowInterpreter()
	if err != nil {
		t.Fatal(err)
	}

	program := WorkflowProgram{
		Version:      WorkflowInterpreterVersion,
		InputSchema:  MustCanonicalValue(true),
		OutputSchema: MustCanonicalValue(true),
		Input:        MustCanonicalValue(map[string]any{}),
		Root: WorkflowBlock{
			Output: WorkflowValue{
				Kind: WorkflowValueLiteral,
				Literal: MustCanonicalValue(map[string]any{
					"$expr": "not evaluated",
				}),
			},
		},
		Targets: map[string]WorkflowTarget{},
	}
	state := workflowProgramState{
		Version: WorkflowInterpreterVersion,
		State:   MustCanonicalValue(map[string]any{}),
		Outputs: map[string]CanonicalValue{},
	}

	value, err := interpreter.evaluate(
		program.Root.Output,
		program,
		state,
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(value) != `{"$expr":"not evaluated"}` {
		t.Fatalf("literal value = %s", value)
	}
}
