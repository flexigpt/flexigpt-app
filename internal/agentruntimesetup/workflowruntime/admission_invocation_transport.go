package workflowruntime

import (
	"fmt"
	"slices"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow"
)

const (
	workflowInvocationContextInstructions = "instructions"

	workflowInvocationPortDirectionInput  = "input"
	workflowInvocationPortDirectionOutput = "output"
)

// verifyWorkflowInvocationTransport consumes planning-owned call, context,
// and port facts. It does not traverse source blocks or interpret source
// directives. The adapter owns current Model and Agent transport capability.
func (a *Admission) verifyWorkflowInvocationTransport(
	inventory workflowAPI.Inventory,
) error {
	if a == nil || a.invocations == nil {
		return nil
	}

	contextByCall := make(map[string][]string, len(inventory.ContextUses))
	for _, use := range inventory.ContextUses {
		contextByCall[use.CallPath] = append(
			contextByCall[use.CallPath],
			use.Name,
		)
	}

	for _, call := range inventory.Calls {
		for _, target := range call.TargetUses {
			capabilities, supported := a.workflowInvocationCapabilities(
				target.Kind,
			)
			if !supported {
				continue
			}

			mode := sessionv3Runtime.WorkflowCallMode(call.Mode)
			if !slices.Contains(capabilities.Modes, mode) {
				return fmt.Errorf(
					"%w: workflow %s target at %q does not support mode %q",
					ErrBindingUnavailable,
					target.Kind,
					call.Path,
					call.Mode,
				)
			}

			for _, name := range contextByCall[call.Path] {
				if !slices.Contains(capabilities.ContextForms, name) {
					return fmt.Errorf(
						"%w: workflow %s target at %q does not support context form %q",
						ErrBindingUnavailable,
						target.Kind,
						call.Path,
						name,
					)
				}
			}

			for _, port := range inventory.PortUses {
				if port.Kind != workflowAPI.PortUseKindCall ||
					port.CallPath != call.Path ||
					port.TargetKind != target.Kind {
					continue
				}
				if !port.ContractKnown {
					return fmt.Errorf(
						"%w: workflow %s target port %q has no typed callable contract",
						ErrBindingUnavailable,
						target.Kind,
						port.Name,
					)
				}

				switch port.Direction {
				case workflowInvocationPortDirectionInput:
					if !capabilities.InputPorts {
						return fmt.Errorf(
							"%w: workflow %s target input port %q is unavailable",
							ErrBindingUnavailable,
							target.Kind,
							port.Name,
						)
					}

				case workflowInvocationPortDirectionOutput:
					if !capabilities.OutputPorts {
						return fmt.Errorf(
							"%w: workflow %s target output port %q is unavailable",
							ErrBindingUnavailable,
							target.Kind,
							port.Name,
						)
					}
					if port.Stream && !capabilities.StreamOutputPorts {
						return fmt.Errorf(
							"%w: workflow %s target streamed port %q is unavailable",
							ErrBindingUnavailable,
							target.Kind,
							port.Name,
						)
					}

				default:
					return fmt.Errorf(
						"%w: workflow %s target port %q has an unsupported direction",
						ErrBindingUnavailable,
						target.Kind,
						port.Name,
					)
				}
			}
		}
	}
	return nil
}

func (a *Admission) workflowInvocationCapabilities(
	kind workflowAPI.TargetUseKind,
) (WorkflowInvocationTargetCapabilities, bool) {
	switch kind {
	case workflowAPI.TargetUseKindModel:
		return a.invocations.ModelInvocationCapabilities(), true
	case workflowAPI.TargetUseKindAgent:
		return a.invocations.AgentInvocationCapabilities(), true
	default:
		return WorkflowInvocationTargetCapabilities{}, false
	}
}
