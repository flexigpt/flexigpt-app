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
	return WalkWorkflowBlock(block, WorkflowTreeVisitor{
		OnStep: func(node WorkflowStepNode) error {
			for claimIndex := range node.Step.Claims {
				if value, found := aliases[node.Step.Claims[claimIndex].Resource]; found {
					node.Step.Claims[claimIndex].Resource = value
				}
			}
			return nil
		},
	})
}
