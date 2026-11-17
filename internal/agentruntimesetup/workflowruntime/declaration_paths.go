package workflowruntime

import (
	"errors"
	"fmt"
	"strconv"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

// attachWorkflowDeclarationPaths copies planning-owned canonical declaration
// paths into runtime-neutral descriptors. It walks the lowered runtime tree,
// not Workflow source. Source traversal remains owned by contract admission
// and Workflow Inventory construction.
func attachWorkflowDeclarationPaths(
	program *sessionv3Runtime.WorkflowProgram,
	inventory workflowAPI.Inventory,
) error {
	if program == nil {
		return errors.New("workflow program is unavailable")
	}

	blocks := make(map[string]struct{}, len(inventory.Blocks))
	for _, value := range inventory.Blocks {
		blocks[value.Path] = struct{}{}
	}
	calls := make(map[string]struct{}, len(inventory.Calls))
	for _, value := range inventory.Calls {
		calls[value.Path] = struct{}{}
	}

	if err := attachWorkflowBlockDeclarationPaths(
		&program.Root,
		workflowv1.RootDeclarationPath,
		blocks,
		calls,
	); err != nil {
		return err
	}

	for name, target := range program.Targets {
		target.DeclarationPath = workflowv1.JoinDeclarationPath(
			workflowv1.RootDeclarationPath,
			"targets",
			name,
		)
		program.Targets[name] = target
	}

	return nil
}

func attachWorkflowBlockDeclarationPaths(
	block *sessionv3Runtime.WorkflowBlock,
	path string,
	blocks map[string]struct{},
	calls map[string]struct{},
) error {
	if block == nil {
		return errors.New("workflow block is unavailable")
	}
	if _, found := blocks[path]; !found {
		return fmt.Errorf(
			"workflow lowering block %q is absent from Inventory",
			path,
		)
	}

	block.DeclarationPath = path
	for index := range block.Steps {
		step := &block.Steps[index]
		stepPath := workflowv1.DeclarationStepPath(path, step.ID)
		step.DeclarationPath = stepPath

		if step.Call != nil {
			callPath := workflowv1.JoinDeclarationPath(stepPath, "call")
			if _, found := calls[callPath]; !found {
				return fmt.Errorf(
					"workflow lowering call %q is absent from Inventory",
					callPath,
				)
			}
			step.Call.DeclarationPath = callPath
		}

		if step.Effect != nil {
			if err := attachOptionalWorkflowBlockDeclarationPath(
				step.Effect.Reconcile,
				workflowv1.JoinDeclarationPath(
					stepPath,
					"effect",
					"reconcile",
				),
				blocks,
				calls,
			); err != nil {
				return err
			}
			if err := attachOptionalWorkflowBlockDeclarationPath(
				step.Effect.Compensate,
				workflowv1.JoinDeclarationPath(
					stepPath,
					"effect",
					"compensate",
				),
				blocks,
				calls,
			); err != nil {
				return err
			}
		}

		switch {
		case step.Scope != nil:
			if err := attachWorkflowBlockDeclarationPaths(
				step.Scope,
				workflowv1.JoinDeclarationPath(stepPath, "scope"),
				blocks,
				calls,
			); err != nil {
				return err
			}

		case step.Choose != nil:
			for caseIndex := range step.Choose.Cases {
				if err := attachWorkflowBlockDeclarationPaths(
					&step.Choose.Cases[caseIndex].Do,
					workflowv1.JoinDeclarationPath(stepPath, "choose", "cases", strconv.Itoa(caseIndex), "do"),
					blocks,
					calls,
				); err != nil {
					return err
				}
			}
			if err := attachOptionalWorkflowBlockDeclarationPath(
				step.Choose.Default,
				workflowv1.JoinDeclarationPath(
					stepPath,
					"choose",
					"default",
				),
				blocks,
				calls,
			); err != nil {
				return err
			}

		case step.Parallel != nil:
			for branchIndex := range step.Parallel.Branches {
				branch := &step.Parallel.Branches[branchIndex]
				if err := attachWorkflowBlockDeclarationPaths(
					&branch.Do,
					workflowv1.JoinDeclarationPath(
						stepPath,
						"parallel",
						"branches",
						branch.Name,
						"do",
					),
					blocks,
					calls,
				); err != nil {
					return err
				}
			}

		case step.Foreach != nil:
			if err := attachWorkflowBlockDeclarationPaths(
				&step.Foreach.Do,
				workflowv1.JoinDeclarationPath(
					stepPath,
					"foreach",
					"do",
				),
				blocks,
				calls,
			); err != nil {
				return err
			}

		case step.Repeat != nil:
			if err := attachWorkflowBlockDeclarationPaths(
				&step.Repeat.Do,
				workflowv1.JoinDeclarationPath(
					stepPath,
					"repeat",
					"do",
				),
				blocks,
				calls,
			); err != nil {
				return err
			}

		case step.Machine != nil:
			for stateIndex := range step.Machine.States {
				state := &step.Machine.States[stateIndex]
				statePath := workflowv1.JoinDeclarationPath(
					stepPath,
					"machine",
					"states",
					state.Name,
				)
				if err := attachOptionalWorkflowBlockDeclarationPath(
					state.Entry,
					workflowv1.JoinDeclarationPath(statePath, "entry"),
					blocks,
					calls,
				); err != nil {
					return err
				}
				if err := attachOptionalWorkflowBlockDeclarationPath(
					state.Exit,
					workflowv1.JoinDeclarationPath(statePath, "exit"),
					blocks,
					calls,
				); err != nil {
					return err
				}
				for transitionIndex := range state.Transitions {
					if err := attachOptionalWorkflowBlockDeclarationPath(
						state.Transitions[transitionIndex].Do,
						workflowv1.JoinDeclarationPath(statePath, "transitions", strconv.Itoa(transitionIndex), "do"),
						blocks,
						calls,
					); err != nil {
						return err
					}
				}
			}

		case step.Saga != nil:
			if err := attachWorkflowBlockDeclarationPaths(
				&step.Saga.Do,
				workflowv1.JoinDeclarationPath(
					stepPath,
					"saga",
					"do",
				),
				blocks,
				calls,
			); err != nil {
				return err
			}
		}
	}

	for index := range block.Catch {
		if err := attachWorkflowBlockDeclarationPaths(
			&block.Catch[index].Do,
			workflowv1.JoinDeclarationPath(path, "catch", strconv.Itoa(index)),
			blocks,
			calls,
		); err != nil {
			return err
		}
	}

	return attachOptionalWorkflowBlockDeclarationPath(
		block.Finally,
		workflowv1.JoinDeclarationPath(path, "finally"),
		blocks,
		calls,
	)
}

func attachOptionalWorkflowBlockDeclarationPath(
	block *sessionv3Runtime.WorkflowBlock,
	path string,
	blocks map[string]struct{},
	calls map[string]struct{},
) error {
	if block == nil {
		return nil
	}
	return attachWorkflowBlockDeclarationPaths(
		block,
		path,
		blocks,
		calls,
	)
}
