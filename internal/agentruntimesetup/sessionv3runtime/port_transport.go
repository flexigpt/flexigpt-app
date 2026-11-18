package sessionv3runtime

import (
	"errors"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

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
