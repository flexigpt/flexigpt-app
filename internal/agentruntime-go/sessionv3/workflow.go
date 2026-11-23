package sessionv3

import "errors"

// ValidatePreparedWorkflowStart validates only the prepared runtime snapshot
// envelope. Source grammar, static schemas, CEL scope, target relationships,
// and interfaces were validated before this value reached Runtime v3.
func ValidatePreparedWorkflowStart(
	program ProgramSnapshot,
) error {
	_, err := validatePreparedWorkflowStartProgram(program)
	return err
}

func validatePreparedWorkflowStartProgram(
	program ProgramSnapshot,
) (WorkflowProgram, error) {
	value, err := validatePreparedWorkflowProgram(program)
	if err != nil {
		return WorkflowProgram{}, err
	}
	if !value.InputBound {
		return WorkflowProgram{}, errors.New(
			"prepared workflow start program has no bound input",
		)
	}
	return value, nil
}

func ValidatePreparedWorkflowTriggers(
	program ProgramSnapshot,
) error {
	_, err := validatePreparedWorkflowTriggersProgram(program)
	return err
}

func validatePreparedWorkflowTriggersProgram(
	program ProgramSnapshot,
) (WorkflowProgram, error) {
	value, err := validatePreparedWorkflowProgram(program)
	if err != nil {
		return WorkflowProgram{}, err
	}
	if len(value.Triggers) == 0 {
		return WorkflowProgram{}, errors.New(
			"workflow trigger registration returned no triggers",
		)
	}
	if value.InputBound {
		return WorkflowProgram{}, errors.New(
			"workflow trigger registration returned an input-bound program",
		)
	}
	return value, nil
}

func validatePreparedWorkflowProgram(
	program ProgramSnapshot,
) (WorkflowProgram, error) {
	if program.InterpreterID == "" || program.InterpreterVersion == 0 {
		return WorkflowProgram{}, errors.New(
			"prepared workflow has no program frame interpreter",
		)
	}
	if program.Artifact == nil {
		return WorkflowProgram{}, errors.New(
			"prepared workflow has no workflow artifact pin",
		)
	}
	if err := validatePin(*program.Artifact); err != nil {
		return WorkflowProgram{}, err
	}
	if program.DefinitionDigest == "" ||
		program.DefinitionDigest != program.Artifact.Digest {
		return WorkflowProgram{}, errors.New(
			"prepared workflow definition digest is invalid",
		)
	}
	if !program.Payload.Valid() {
		return WorkflowProgram{}, errors.New(
			"prepared workflow payload is not canonical json",
		)
	}
	return workflowProgramFromSnapshot(program)
}
