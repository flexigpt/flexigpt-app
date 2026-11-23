package workflowruntime

import (
	"errors"
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

const workflowCallPathSegment = "call"

// attachWorkflowDeclarationPaths applies planning-owned canonical declaration
// paths from Workflow Inventory onto the lowered runtime tree using the single
// typed lowered-tree traversal contract.
func attachWorkflowDeclarationPaths(
	program *sessionv3Runtime.WorkflowProgram,
	inventory workflowAPI.Inventory,
) error {
	if program == nil {
		return errors.New("workflow program is unavailable")
	}

	calls := make(map[string]struct{}, len(inventory.Calls))
	for _, value := range inventory.Calls {
		calls[value.Path] = struct{}{}
	}

	blockIndex := 0
	if err := sessionv3Runtime.WalkWorkflowBlock(
		&program.Root,
		sessionv3Runtime.WorkflowTreeVisitor{
			OnBlock: func(node sessionv3Runtime.WorkflowBlockNode) error {
				if blockIndex >= len(inventory.Blocks) {
					return errors.New("workflow lowering has more blocks than inventory")
				}
				planned := inventory.Blocks[blockIndex]
				blockIndex++
				if string(planned.Role) != string(node.Role) {
					return fmt.Errorf(
						"workflow lowering block %q role %q differs from inventory role %q",
						planned.Path,
						node.Role,
						planned.Role,
					)
				}
				node.Block.DeclarationPath = planned.Path
				return nil
			},
			OnStep: func(node sessionv3Runtime.WorkflowStepNode) error {
				stepPath := workflowv1.DeclarationStepPath(
					node.Block.DeclarationPath,
					node.Step.ID,
				)
				node.Step.DeclarationPath = stepPath
				if node.Step.Call != nil {
					callPath := workflowv1.JoinDeclarationPath(stepPath, workflowCallPathSegment)
					if _, found := calls[callPath]; !found {
						return fmt.Errorf(
							"workflow lowering call %q is absent from inventory",
							callPath,
						)
					}
					node.Step.Call.DeclarationPath = callPath
				}
				return nil
			},
		},
	); err != nil {
		return err
	}
	if blockIndex != len(inventory.Blocks) {
		return errors.New("workflow inventory has unconsumed blocks")
	}
	return nil
}
