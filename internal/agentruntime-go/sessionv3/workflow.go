package sessionv3

import "errors"

// ValidatePreparedWorkflowStart validates only the prepared runtime snapshot
// envelope. Source grammar, static schemas, CEL scope, target relationships,
// and interfaces were validated before this value reached Runtime v3.
func ValidatePreparedWorkflowStart(
	program ProgramSnapshot,
) error {
	if err := validatePreparedWorkflowProgram(program); err != nil {
		return err
	}
	value, err := workflowProgramFromSnapshot(program)
	if err != nil {
		return err
	}
	if !value.InputBound {
		return errors.New("workflow trigger registration returned no triggers")
	}
	return nil
}

func ValidatePreparedWorkflowTriggers(
	program ProgramSnapshot,
) error {
	if err := validatePreparedWorkflowProgram(program); err != nil {
		return err
	}
	value, err := workflowProgramFromSnapshot(program)
	if err != nil {
		return err
	}
	if len(value.Triggers) == 0 {
		return errors.New("workflow trigger registration returned no triggers")
	}
	if value.InputBound {
		return errors.New(
			"workflow trigger registration returned an input-bound program",
		)
	}
	return nil
}

func validatePreparedWorkflowProgram(
	program ProgramSnapshot,
) error {
	if program.InterpreterID == "" || program.InterpreterVersion == 0 {
		return errors.New("prepared workflow has no ProgramFrame interpreter")
	}
	if program.Artifact == nil {
		return errors.New("prepared workflow has no Workflow Artifact pin")
	}
	if err := validatePin(*program.Artifact); err != nil {
		return err
	}
	if program.DefinitionDigest == "" ||
		program.DefinitionDigest != program.Artifact.Digest {
		return errors.New("prepared workflow definition digest is invalid")
	}
	if !program.Payload.Valid() {
		return errors.New("prepared workflow payload is not canonical JSON")
	}
	if _, err := workflowProgramFromSnapshot(program); err != nil {
		return err
	}
	return nil
}
