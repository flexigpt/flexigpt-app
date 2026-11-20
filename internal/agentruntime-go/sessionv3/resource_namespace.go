package sessionv3

import (
	"crypto/sha256"
	"encoding/hex"
)

// workflowResourceNamespace returns a bounded deterministic namespace for one
// nested Workflow invocation. Attached nested Workflows can reuse declaration
// resource names without sharing unrelated leases or rate windows.
func workflowResourceNamespace(
	executionID ExecutionID,
	frameID ProgramFrameID,
) string {
	sum := sha256.Sum256([]byte(
		string(executionID) + "\x00" + string(frameID),
	))
	return "resource:" + hex.EncodeToString(sum[:])
}

func namespaceWorkflowProgramResources(
	snapshot ProgramSnapshot,
	namespace string,
) (ProgramSnapshot, error) {
	program, err := workflowProgramFromSnapshot(snapshot)
	if err != nil {
		return ProgramSnapshot{}, err
	}

	aliases := make(map[string]string, len(program.Resources))
	for _, resource := range program.Resources {
		aliases[resource.Name] = namespace + ":" + resource.Name
	}
	for index := range program.Resources {
		program.Resources[index].Name = aliases[program.Resources[index].Name]
	}
	if err := namespaceWorkflowBlockResources(&program.Root, aliases); err != nil {
		return ProgramSnapshot{}, err
	}

	payload, err := canonicalWorkflowValue(program)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	snapshot.Payload = payload
	return snapshot, nil
}

func namespaceWorkflowBlockResources(
	block *WorkflowBlock,
	aliases map[string]string,
) error {
	if block == nil {
		return nil
	}
	for index := range block.Steps {
		step := &block.Steps[index]
		for claimIndex := range step.Claims {
			if value, found := aliases[step.Claims[claimIndex].Resource]; found {
				step.Claims[claimIndex].Resource = value
			}
		}
		if step.Effect != nil {
			if err := namespaceWorkflowBlockResources(
				step.Effect.Reconcile,
				aliases,
			); err != nil {
				return err
			}
			if err := namespaceWorkflowBlockResources(
				step.Effect.Compensate,
				aliases,
			); err != nil {
				return err
			}
		}

		switch {
		case step.Scope != nil:
			if err := namespaceWorkflowBlockResources(step.Scope, aliases); err != nil {
				return err
			}
		case step.Choose != nil:
			for caseIndex := range step.Choose.Cases {
				if err := namespaceWorkflowBlockResources(
					&step.Choose.Cases[caseIndex].Do,
					aliases,
				); err != nil {
					return err
				}
			}
			if err := namespaceWorkflowBlockResources(
				step.Choose.Default,
				aliases,
			); err != nil {
				return err
			}
		case step.Parallel != nil:
			for branchIndex := range step.Parallel.Branches {
				if err := namespaceWorkflowBlockResources(
					&step.Parallel.Branches[branchIndex].Do,
					aliases,
				); err != nil {
					return err
				}
			}
		case step.Foreach != nil:
			if err := namespaceWorkflowBlockResources(&step.Foreach.Do, aliases); err != nil {
				return err
			}
		case step.Repeat != nil:
			if err := namespaceWorkflowBlockResources(&step.Repeat.Do, aliases); err != nil {
				return err
			}
		case step.Machine != nil:
			for stateIndex := range step.Machine.States {
				state := &step.Machine.States[stateIndex]
				if err := namespaceWorkflowBlockResources(state.Entry, aliases); err != nil {
					return err
				}
				if err := namespaceWorkflowBlockResources(state.Exit, aliases); err != nil {
					return err
				}
				for transitionIndex := range state.Transitions {
					if err := namespaceWorkflowBlockResources(
						state.Transitions[transitionIndex].Do,
						aliases,
					); err != nil {
						return err
					}
				}
			}
		case step.Saga != nil:
			if err := namespaceWorkflowBlockResources(&step.Saga.Do, aliases); err != nil {
				return err
			}
		}
	}
	for index := range block.Catch {
		if err := namespaceWorkflowBlockResources(
			&block.Catch[index].Do,
			aliases,
		); err != nil {
			return err
		}
	}
	if err := namespaceWorkflowBlockResources(block.Finally, aliases); err != nil {
		return err
	}
	return nil
}
