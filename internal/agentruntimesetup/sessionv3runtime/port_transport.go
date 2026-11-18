package sessionv3runtime

import (
	"errors"
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

func validatePortTransport(
	plans []sessionv3Runtime.PortPlan,
	transport sessionv3Runtime.WorkflowTargetTransport,
) error {
	if err := transport.Validate(); err != nil {
		return err
	}
	for _, plan := range plans {
		switch plan.Direction {
		case sessionv3Runtime.WorkflowPortDirectionInput:
			if !transport.InputPorts {
				return errors.New("target does not support workflow input ports")
			}
		case sessionv3Runtime.WorkflowPortDirectionOutput:
			if plan.Stream && !transport.StreamOutputPorts {
				return errors.New("target does not support streamed output ports")
			}
			if !plan.Stream && !transport.FinalOutputPorts {
				return errors.New("target does not support final output ports")
			}
		default:
			return fmt.Errorf("unsupported workflow port direction %q", plan.Direction)
		}
	}
	return nil
}

func validateLLMPortTransport(
	item sessionv3Runtime.WorkItem,
) error {
	for _, plan := range item.Ports {
		if plan.Direction == sessionv3Runtime.WorkflowPortDirectionInput {
			return errors.New("model and agent WorkItems do not accept workflow input ports")
		}
		if plan.Direction == sessionv3Runtime.WorkflowPortDirectionOutput &&
			!plan.Stream {
			return errors.New(
				"model and agent final ports belong to the child ProgramFrame, not the LLM WorkItem",
			)
		}
	}
	return nil
}

func validateToolPortTransport(
	item sessionv3Runtime.WorkItem,
) error {
	for _, plan := range item.Ports {
		if plan.Direction == sessionv3Runtime.WorkflowPortDirectionInput {
			return errors.New("tool workItems do not accept workflow input ports")
		}
		if plan.Direction == sessionv3Runtime.WorkflowPortDirectionOutput &&
			plan.Stream {
			return errors.New("tool workItems do not support streamed output ports")
		}
	}
	return nil
}
