package workflowruntime

import (
	"fmt"
	"slices"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow"
)

func validatePreparedTargetTransport(
	inventory workflowAPI.Inventory,
	program *sessionv3Runtime.WorkflowProgram,
) error {
	if program == nil {
		return fmt.Errorf("%w: workflow program is unavailable", ErrBindingUnavailable)
	}

	portUses := make(map[string][]workflowAPI.PortUse)
	for _, use := range inventory.PortUses {
		if use.Kind != workflowAPI.PortUseKindCall {
			continue
		}
		portUses[use.CallPath] = append(portUses[use.CallPath], use)
	}
	contextUses := make(map[string][]workflowAPI.ContextUse)
	for _, use := range inventory.ContextUses {
		contextUses[use.CallPath] = append(contextUses[use.CallPath], use)
	}

	for _, call := range inventory.Calls {
		lowered := workflowCallAtDeclarationPath(
			&program.Root,
			call.Path,
		)
		if lowered == nil {
			return fmt.Errorf(
				"%w: call %q has no lowered runtime descriptor",
				ErrBindingUnavailable,
				call.Path,
			)
		}
		mode := sessionv3Runtime.WorkflowCallMode(call.Mode)
		if mode == "" {
			mode = sessionv3Runtime.WorkflowCallModeInvoke
		}
		for _, use := range call.TargetUses {
			target, found := program.Targets[use.Name]
			if !found {
				return fmt.Errorf(
					"%w: call %q target %q has no prepared target descriptor",
					ErrBindingUnavailable,
					call.Path,
					use.Name,
				)
			}
			if !target.Transport.SupportsMode(mode) {
				return fmt.Errorf(
					"%w: target %q does not support call mode %q",
					ErrBindingUnavailable,
					use.Name,
					mode,
				)
			}

			if workflowTargetRequiresTypedLLMInvocation(target) &&
				lowered.LLMInvocation == nil {
				return fmt.Errorf(
					"%w: call %q target %q requires the explicit typed model or agent invocation form",
					ErrBindingUnavailable,
					call.Path,
					use.Name,
				)
			}
			for _, contextUse := range contextUses[call.Path] {
				if workflowTransportAcceptsContext(
					target.Transport,
					contextUse.Name,
				) {
					continue
				}
				return fmt.Errorf(
					"%w: target %q does not support workflow context form %q",
					ErrBindingUnavailable,
					use.Name,
					contextUse.Name,
				)
			}

			for _, port := range portUses[call.Path] {
				if port.TargetName != "" && port.TargetName != use.Name {
					continue
				}
				if !port.ContractKnown {
					return fmt.Errorf(
						"%w: target %q port %q has no prepared callable transport fact",
						ErrBindingUnavailable,
						use.Name,
						port.Name,
					)
				}
				switch port.Direction {
				case string(sessionv3Runtime.WorkflowPortDirectionInput):
					if !target.Transport.InputPorts {
						return fmt.Errorf(
							"%w: target %q does not support workflow input ports",
							ErrBindingUnavailable,
							use.Name,
						)
					}
				case string(sessionv3Runtime.WorkflowPortDirectionOutput):
					if port.Stream {
						if !target.Transport.StreamOutputPorts {
							return fmt.Errorf(
								"%w: target %q does not support streamed output ports",
								ErrBindingUnavailable,
								use.Name,
							)
						}
					} else if !target.Transport.FinalOutputPorts {
						return fmt.Errorf(
							"%w: target %q does not support final output ports",
							ErrBindingUnavailable,
							use.Name,
						)
					}
				default:
					return fmt.Errorf(
						"%w: target %q port %q has an invalid direction",
						ErrBindingUnavailable,
						use.Name,
						port.Name,
					)
				}
			}
		}
	}
	return nil
}

func workflowTargetRequiresTypedLLMInvocation(
	target sessionv3Runtime.WorkflowTarget,
) bool {
	switch target.Kind {
	case sessionv3Runtime.WorkflowTargetModel,
		sessionv3Runtime.WorkflowTargetAgent,
		sessionv3Runtime.WorkflowTargetProviderTool:
		return true
	default:
		return false
	}
}

func workflowTransportAcceptsContext(
	transport sessionv3Runtime.WorkflowTargetTransport,
	name string,
) bool {
	return slices.Contains(transport.ContextForms, wildcardContextForm) ||
		slices.Contains(transport.ContextForms, name)
}
