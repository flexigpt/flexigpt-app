package workflowruntime

import (
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow"
)

func validatePreparedTargetTransport(
	inventory workflowAPI.Inventory,
	targets map[string]sessionv3Runtime.WorkflowTarget,
) error {
	portUses := make(map[string][]workflowAPI.PortUse)
	for _, use := range inventory.PortUses {
		if use.Kind != workflowAPI.PortUseKindCall {
			continue
		}
		portUses[use.CallPath] = append(portUses[use.CallPath], use)
	}

	for _, call := range inventory.Calls {
		mode := sessionv3Runtime.WorkflowCallMode(call.Mode)
		if mode == "" {
			mode = sessionv3Runtime.WorkflowCallModeInvoke
		}
		for _, use := range call.TargetUses {
			target, found := targets[use.Name]
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
