package workflowruntime

import (
	"fmt"
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

// verifyTargetTransportCapabilities completes setup admission for Service and
// Activity context and port transport. It intentionally runs before a
// Workflow execution exists, so a known unavailable transport cannot surface
// after unrelated external work started.
func (a *Admission) verifyTargetTransportCapabilities(
	program sessionv3Runtime.WorkflowProgram,
) error {
	return a.verifyBlockTransport(program, program.Root)
}

func (a *Admission) verifyBlockTransport(
	program sessionv3Runtime.WorkflowProgram,
	block sessionv3Runtime.WorkflowBlock,
) error {
	for _, step := range block.Steps {
		if step.Call != nil {
			if err := a.verifyCallTransport(program, *step.Call); err != nil {
				return err
			}
		}
		if step.Effect != nil {
			if step.Effect.Reconcile != nil {
				if err := a.verifyBlockTransport(
					program,
					*step.Effect.Reconcile,
				); err != nil {
					return err
				}
			}
			if step.Effect.Compensate != nil {
				if err := a.verifyBlockTransport(
					program,
					*step.Effect.Compensate,
				); err != nil {
					return err
				}
			}
		}
		if err := a.verifyStepChildren(program, step); err != nil {
			return err
		}
	}
	for _, handler := range block.Catch {
		if err := a.verifyBlockTransport(program, handler.Do); err != nil {
			return err
		}
	}
	if block.Finally != nil {
		return a.verifyBlockTransport(program, *block.Finally)
	}
	return nil
}

func (a *Admission) verifyStepChildren(
	program sessionv3Runtime.WorkflowProgram,
	step sessionv3Runtime.WorkflowStep,
) error {
	switch {
	case step.Scope != nil:
		return a.verifyBlockTransport(program, *step.Scope)
	case step.Saga != nil:
		return a.verifyBlockTransport(program, step.Saga.Do)
	case step.Choose != nil:
		for _, candidate := range step.Choose.Cases {
			if err := a.verifyBlockTransport(program, candidate.Do); err != nil {
				return err
			}
		}
		if step.Choose.Default != nil {
			return a.verifyBlockTransport(program, *step.Choose.Default)
		}
	case step.Parallel != nil:
		for _, branch := range step.Parallel.Branches {
			if err := a.verifyBlockTransport(program, branch.Do); err != nil {
				return err
			}
		}
	case step.Foreach != nil:
		return a.verifyBlockTransport(program, step.Foreach.Do)
	case step.Repeat != nil:
		return a.verifyBlockTransport(program, step.Repeat.Do)
	case step.Machine != nil:
		for _, state := range step.Machine.States {
			if state.Entry != nil {
				if err := a.verifyBlockTransport(program, *state.Entry); err != nil {
					return err
				}
			}
			if state.Exit != nil {
				if err := a.verifyBlockTransport(program, *state.Exit); err != nil {
					return err
				}
			}
			for _, transition := range state.Transitions {
				if transition.Do != nil {
					if err := a.verifyBlockTransport(
						program,
						*transition.Do,
					); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (a *Admission) verifyCallTransport(
	program sessionv3Runtime.WorkflowProgram,
	call sessionv3Runtime.WorkflowCall,
) error {
	for _, target := range admissionCallTargets(program, call) {
		var binding WorkBinding
		switch target.Kind {
		case sessionv3Runtime.WorkflowTargetService:
			binding, _ = a.bindings.Service(target.ServiceID)
		case sessionv3Runtime.WorkflowTargetActivity:
			binding, _ = a.bindings.Activity(target.ActivityID)
		default:
			continue
		}
		if binding == nil {
			return fmt.Errorf("%w: selected target binding", ErrBindingUnavailable)
		}

		needsContext := !workflowCallContextIsEmpty(call.Context)
		needsInputPorts := false
		needsOutputStream := false
		for _, port := range call.Ports {
			direction, stream, found := admissionCallablePort(
				program,
				call,
				target,
				port.Name,
			)
			if !found {
				return fmt.Errorf(
					"%w: callable port %q",
					ErrBindingUnavailable,
					port.Name,
				)
			}
			if direction == sessionv3Runtime.WorkflowPortDirectionInput {
				needsInputPorts = true
			}
			if direction == sessionv3Runtime.WorkflowPortDirectionOutput &&
				stream {
				needsOutputStream = true
			}
		}
		if !needsContext && !needsInputPorts && !needsOutputStream {
			continue
		}

		capability, found := binding.(WorkBindingPortCapability)
		if !found {
			return fmt.Errorf(
				"%w: external binding does not support Workflow transport",
				ErrBindingUnavailable,
			)
		}
		support := capability.WorkflowPortCapabilities()
		switch {
		case needsContext && !support.Context:
			return fmt.Errorf(
				"%w: external binding does not support Workflow context",
				ErrBindingUnavailable,
			)
		case needsInputPorts && !support.InputPorts:
			return fmt.Errorf(
				"%w: external binding does not support Workflow input ports",
				ErrBindingUnavailable,
			)
		case needsOutputStream && !support.OutputStream:
			return fmt.Errorf(
				"%w: external binding does not support Workflow streamed output ports",
				ErrBindingUnavailable,
			)
		}
	}
	return nil
}

func admissionCallTargets(
	program sessionv3Runtime.WorkflowProgram,
	call sessionv3Runtime.WorkflowCall,
) []sessionv3Runtime.WorkflowTarget {
	names := make(map[string]struct{})
	if call.Target.Kind == sessionv3Runtime.WorkflowValueLiteral {
		if name, err := sessionv3Runtime.DecodeCanonicalValue[string](
			call.Target.Literal,
		); err == nil && name != "" {
			names[name] = struct{}{}
		}
	}
	if call.AllowedTargetSet != "" {
		if set, found := program.TargetSets[call.AllowedTargetSet]; found {
			for _, member := range set.Members {
				names[member] = struct{}{}
			}
			if set.Resolved != nil {
				for name := range program.Targets {
					names[name] = struct{}{}
				}
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
		if target, found := program.Targets[name]; found {
			output = append(output, target)
		}
	}
	return output
}

func admissionCallablePort(
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
