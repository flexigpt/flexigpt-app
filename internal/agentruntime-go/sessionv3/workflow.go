package sessionv3

import (
	"context"
	"errors"
)

func ValidateWorkflowTriggerRegistrationResult(
	result WorkflowAdmissionResult,
) error {
	if err := ValidateWorkflowAdmissionResult(result); err != nil {
		return err
	}
	program, err := workflowProgramFromSnapshot(result.Program)
	if err != nil {
		return err
	}
	if len(program.Triggers) == 0 {
		return errors.New("workflow trigger registration returned no triggers")
	}
	if program.InputBound {
		return errors.New(
			"workflow trigger registration returned an input-bound program",
		)
	}
	return nil
}

// StartWorkflow and SignalExecution are implemented in runtime.go so that
// all Session mutations share one lock, revision, idempotency, checkpoint,
// cancellation, and recovery path. This file retains compile-time assertions
// for the public Workflow admission contract.

var _ WorkflowAdmission = workflowAdmissionFunc(nil)

type workflowAdmissionFunc func(
	context.Context,
	WorkflowAdmissionRequest,
) (WorkflowAdmissionResult, error)

func (f workflowAdmissionFunc) AdmitWorkflow(
	ctx context.Context,
	request WorkflowAdmissionRequest,
) (WorkflowAdmissionResult, error) {
	if f == nil {
		return WorkflowAdmissionResult{}, errors.New(
			"Workflow admission function is unavailable",
		)
	}
	return f(ctx, request)
}

func ValidateWorkflowAdmissionResult(
	result WorkflowAdmissionResult,
) error {
	program := result.Program
	if program.InterpreterID == "" || program.InterpreterVersion == 0 {
		return errors.New("workflow admission returned no ProgramFrame interpreter")
	}
	if program.Artifact == nil {
		return errors.New("workflow admission returned no Workflow Artifact pin")
	}
	if err := validatePin(*program.Artifact); err != nil {
		return err
	}
	if program.DefinitionDigest == "" ||
		program.DefinitionDigest != program.Artifact.Digest {
		return errors.New("workflow ProgramFrame definition digest is invalid")
	}
	if !program.Payload.Valid() {
		return errors.New("workflow ProgramFrame payload is not canonical JSON")
	}
	if _, err := workflowProgramFromSnapshot(program); err != nil {
		return err
	}
	return nil
}
