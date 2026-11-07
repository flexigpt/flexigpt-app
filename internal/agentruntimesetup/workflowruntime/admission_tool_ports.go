package workflowruntime

import (
	"errors"
	"fmt"
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

const (
	workflowToolPortBindingUnavailableCode = "workflow tool input-port binding is unavailable"
	workflowToolContextBindingUnavailable  = "workflow tool context binding is unavailable"
)

func rejectUnavailableWorkflowToolInputPorts(
	program sessionv3Runtime.WorkflowProgram,
) error {
	return inspectWorkflowToolPortBlock(program, program.Root)
}

func inspectWorkflowToolPortBlock(
	program sessionv3Runtime.WorkflowProgram,
	block sessionv3Runtime.WorkflowBlock,
) error {
	for _, step := range block.Steps {
		if step.Call != nil {
			if err := inspectWorkflowToolCallPorts(
				program,
				*step.Call,
			); err != nil {
				return err
			}
		}
		if step.Effect != nil {
			if step.Effect.Reconcile != nil {
				if err := inspectWorkflowToolPortBlock(
					program,
					*step.Effect.Reconcile,
				); err != nil {
					return err
				}
			}
			if step.Effect.Compensate != nil {
				if err := inspectWorkflowToolPortBlock(
					program,
					*step.Effect.Compensate,
				); err != nil {
					return err
				}
			}
		}
		if err := inspectWorkflowToolStepChildren(program, step); err != nil {
			return err
		}
	}
	for _, handler := range block.Catch {
		if err := inspectWorkflowToolPortBlock(program, handler.Do); err != nil {
			return err
		}
	}
	if block.Finally != nil {
		return inspectWorkflowToolPortBlock(program, *block.Finally)
	}
	return nil
}

func inspectWorkflowToolStepChildren(
	program sessionv3Runtime.WorkflowProgram,
	step sessionv3Runtime.WorkflowStep,
) error {
	switch {
	case step.Scope != nil:
		return inspectWorkflowToolPortBlock(program, *step.Scope)

	case step.Saga != nil:
		return inspectWorkflowToolPortBlock(program, step.Saga.Do)

	case step.Choose != nil:
		for _, candidate := range step.Choose.Cases {
			if err := inspectWorkflowToolPortBlock(program, candidate.Do); err != nil {
				return err
			}
		}
		if step.Choose.Default != nil {
			return inspectWorkflowToolPortBlock(program, *step.Choose.Default)
		}

	case step.Parallel != nil:
		for _, branch := range step.Parallel.Branches {
			if err := inspectWorkflowToolPortBlock(program, branch.Do); err != nil {
				return err
			}
		}

	case step.Foreach != nil:
		return inspectWorkflowToolPortBlock(program, step.Foreach.Do)

	case step.Repeat != nil:
		return inspectWorkflowToolPortBlock(program, step.Repeat.Do)

	case step.Machine != nil:
		for _, state := range step.Machine.States {
			if state.Entry != nil {
				if err := inspectWorkflowToolPortBlock(
					program,
					*state.Entry,
				); err != nil {
					return err
				}
			}
			if state.Exit != nil {
				if err := inspectWorkflowToolPortBlock(
					program,
					*state.Exit,
				); err != nil {
					return err
				}
			}
			for _, transition := range state.Transitions {
				if transition.Do == nil {
					continue
				}
				if err := inspectWorkflowToolPortBlock(
					program,
					*transition.Do,
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func inspectWorkflowToolCallPorts(
	program sessionv3Runtime.WorkflowProgram,
	call sessionv3Runtime.WorkflowCall,
) error {
	targets, err := workflowToolCallTargets(program, call)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if target.Kind != sessionv3Runtime.WorkflowTargetTool &&
			target.Kind != sessionv3Runtime.WorkflowTargetMCPTool {
			continue
		}
		if !workflowCallContextIsEmpty(call.Context) {
			return fmt.Errorf(
				"%w: %s",
				ErrBindingUnavailable,
				workflowToolContextBindingUnavailable,
			)
		}
		for _, binding := range call.Ports {
			direction, found := workflowToolPortDirection(
				program,
				call,
				target,
				binding.Name,
			)
			if !found || direction == sessionv3Runtime.WorkflowPortDirectionInput {
				return fmt.Errorf(
					"%w: %s",
					ErrBindingUnavailable,
					workflowToolPortBindingUnavailableCode,
				)
			}
		}
	}
	return nil
}

func workflowToolCallTargets(
	program sessionv3Runtime.WorkflowProgram,
	call sessionv3Runtime.WorkflowCall,
) ([]sessionv3Runtime.WorkflowTarget, error) {
	names := make(map[string]struct{})

	if value, err := sessionv3Runtime.DecodeCanonicalValue[string](
		call.Target.Literal,
	); err == nil && value != "" {
		names[value] = struct{}{}
	}

	if call.AllowedTargetSet != "" {
		set, found := program.TargetSets[call.AllowedTargetSet]
		if !found {
			return nil, errors.New("workflow target set is unavailable")
		}
		for _, name := range set.Members {
			names[name] = struct{}{}
		}
		if set.Resolved != nil {
			for name := range program.Targets {
				names[name] = struct{}{}
			}
		}
	}

	ordered := make([]string, 0, len(names))
	for name := range names {
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)

	output := make([]sessionv3Runtime.WorkflowTarget, 0, len(ordered))
	for _, name := range ordered {
		target, found := program.Targets[name]
		if !found {
			return nil, errors.New("workflow target is unavailable")
		}
		output = append(output, target)
	}
	return output, nil
}

func workflowToolPortDirection(
	program sessionv3Runtime.WorkflowProgram,
	call sessionv3Runtime.WorkflowCall,
	target sessionv3Runtime.WorkflowTarget,
	name string,
) (sessionv3Runtime.WorkflowPortDirection, bool) {
	contracts := []*sessionv3Runtime.WorkflowCallableInterface{
		target.Contract,
		call.Contract,
		call.SetContract,
	}
	if call.AllowedTargetSet != "" {
		if set, found := program.TargetSets[call.AllowedTargetSet]; found {
			if set.Resolved != nil {
				contracts = append(contracts, &set.Resolved.Contract)
			} else {
				contracts = append(contracts, set.Contract)
			}
		}
	}
	for _, contract := range contracts {
		if contract == nil {
			continue
		}
		for _, port := range contract.Ports {
			if port.Name == name {
				return port.Direction, true
			}
		}
	}
	return "", false
}

func workflowCallContextIsEmpty(
	value sessionv3Runtime.WorkflowValue,
) bool {
	switch value.Kind {
	case sessionv3Runtime.WorkflowValueObject:
		return len(value.Object) == 0
	case sessionv3Runtime.WorkflowValueLiteral:
		object, err := sessionv3Runtime.DecodeCanonicalValue[map[string]sessionv3Runtime.CanonicalValue](value.Literal)
		return err == nil && len(object) == 0
	default:
		return false
	}
}
