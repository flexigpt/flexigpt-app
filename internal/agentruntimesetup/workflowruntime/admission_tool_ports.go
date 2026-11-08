package workflowruntime

import (
	"fmt"

	workflowAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow"
)

const (
	workflowToolContextTransportUnavailableMessage    = "workflow tool context transport is unavailable"
	workflowToolInputPortTransportUnavailableMessage  = "workflow tool input port transport is unavailable"
	workflowToolStreamPortTransportUnavailableMessage = "workflow tool streamed port transport is unavailable"
	workflowToolPortContractUnavailableMessage        = "workflow tool port contract is unavailable"
)

// rejectUnsupportedWorkflowToolTransport consumes the Workflow-owned source
// inventory. It does not walk Workflow blocks, parse declaration values, or
// resolve source relationships.
//
// Setup owns the current adapter availability decision. Planning owns the
// declaration facts that identify which adapter capability is required.
func rejectUnsupportedWorkflowToolTransport(
	inventory workflowAPI.Inventory,
) error {
	contextByCall := make(map[string]bool, len(inventory.ContextUses))
	for _, use := range inventory.ContextUses {
		contextByCall[use.CallPath] = true
	}

	for _, call := range inventory.Calls {
		if !callUsesToolTarget(call) {
			continue
		}

		if contextByCall[call.Path] {
			return fmt.Errorf(
				"%w: %s",
				ErrBindingUnavailable,
				workflowToolContextTransportUnavailableMessage,
			)
		}

		for _, use := range inventory.PortUses {
			if use.Kind != workflowAPI.PortUseKindCall ||
				use.CallPath != call.Path {
				continue
			}

			// A call can target a mixed target set. Only the Tool target
			// transport restriction applies here.
			if use.TargetName != "" &&
				use.TargetKind != workflowAPI.TargetUseKindTool {
				continue
			}

			if !use.ContractKnown {
				return fmt.Errorf(
					"%w: %s",
					ErrBindingUnavailable,
					workflowToolPortContractUnavailableMessage,
				)
			}
			if use.Direction == "input" {
				return fmt.Errorf(
					"%w: %s",
					ErrBindingUnavailable,
					workflowToolInputPortTransportUnavailableMessage,
				)
			}
			if use.Stream {
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

func callUsesToolTarget(
	call workflowAPI.CallSite,
) bool {
	for _, target := range call.TargetUses {
		if target.Kind == workflowAPI.TargetUseKindTool {
			return true
		}
	}
	return false
}
