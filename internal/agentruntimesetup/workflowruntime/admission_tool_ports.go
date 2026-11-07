package workflowruntime

import (
	"errors"
	"fmt"
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

const (
	workflowToolContextTransportUnavailableMessage    = "workflow tool context transport is unavailable"
	workflowToolInputPortTransportUnavailableMessage  = "workflow tool input port transport is unavailable"
	workflowToolStreamPortTransportUnavailableMessage = "workflow tool streamed port transport is unavailable"
	workflowToolPortContractUnavailableMessage        = "workflow tool port contract is unavailable"
)

func rejectUnsupportedWorkflowToolTransport(
	program sessionv3Runtime.WorkflowProgram,
) error {
	return inspectWorkflowToolTransportBlock(program, program.Root)
}

func inspectWorkflowToolTransportBlock(
	program sessionv3Runtime.WorkflowProgram,
	block sessionv3Runtime.WorkflowBlock,
) error {
	for _, step := range block.Steps {
		if step.Call != nil {
			if err := inspectWorkflowToolCallTransport(
				program,
				*step.Call,
			); err != nil {
				return err
			}
		}
		if step.Effect != nil {
			if step.Effect.Reconcile != nil {
				if err := inspectWorkflowToolTransportBlock(
					program,
					*step.Effect.Reconcile,
				); err != nil {
					return err
				}
			}
			if step.Effect.Compensate != nil {
				if err := inspectWorkflowToolTransportBlock(
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
		if err := inspectWorkflowToolTransportBlock(
			program,
			handler.Do,
		); err != nil {
			return err
		}
	}
	if block.Finally != nil {
		return inspectWorkflowToolTransportBlock(program, *block.Finally)
	}
	return nil
}

func inspectWorkflowToolStepChildren(
	program sessionv3Runtime.WorkflowProgram,
	step sessionv3Runtime.WorkflowStep,
) error {
	switch {
	case step.Scope != nil:
		return inspectWorkflowToolTransportBlock(program, *step.Scope)

	case step.Saga != nil:
		return inspectWorkflowToolTransportBlock(program, step.Saga.Do)

	case step.Choose != nil:
		for _, candidate := range step.Choose.Cases {
			if err := inspectWorkflowToolTransportBlock(
				program,
				candidate.Do,
			); err != nil {
				return err
			}
		}
		if step.Choose.Default != nil {
			return inspectWorkflowToolTransportBlock(
				program,
				*step.Choose.Default,
			)
		}

	case step.Parallel != nil:
		for _, branch := range step.Parallel.Branches {
			if err := inspectWorkflowToolTransportBlock(
				program,
				branch.Do,
			); err != nil {
				return err
			}
		}

	case step.Foreach != nil:
		return inspectWorkflowToolTransportBlock(program, step.Foreach.Do)

	case step.Repeat != nil:
		return inspectWorkflowToolTransportBlock(program, step.Repeat.Do)

	case step.Machine != nil:
		for _, state := range step.Machine.States {
			if state.Entry != nil {
				if err := inspectWorkflowToolTransportBlock(
					program,
					*state.Entry,
				); err != nil {
					return err
				}
			}
			if state.Exit != nil {
				if err := inspectWorkflowToolTransportBlock(
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
				if err := inspectWorkflowToolTransportBlock(
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

func inspectWorkflowToolCallTransport(
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
				workflowToolContextTransportUnavailableMessage,
			)
		}
		for _, binding := range call.Ports {
			direction, stream, found := workflowToolPort(
				program,
				call,
				target,
				binding.Name,
			)
			if !found {
				return fmt.Errorf(
					"%w: %s",
					ErrBindingUnavailable,
					workflowToolPortContractUnavailableMessage,
				)
			}
			if direction == sessionv3Runtime.WorkflowPortDirectionInput {
				return fmt.Errorf(
					"%w: %s",
					ErrBindingUnavailable,
					workflowToolInputPortTransportUnavailableMessage,
				)
			}
			if stream {
				return fmt.Errorf(
					"%w: %s",
					ErrBindingUnavailable,
					workflowToolStreamPortTransportUnavailableMessage,
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

func workflowToolPort(
	program sessionv3Runtime.WorkflowProgram,
	call sessionv3Runtime.WorkflowCall,
	target sessionv3Runtime.WorkflowTarget,
	name string,
) (sessionv3Runtime.WorkflowPortDirection, bool, bool) {
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
				return port.Direction, port.Stream, true
			}
		}
	}
	return "", false, false
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
