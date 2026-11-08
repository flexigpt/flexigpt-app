package sessionv3

import (
	"errors"
	"fmt"
)

// validateWorkflowFramePortBoundary gives root Workflow ports their durable
// parent-child boundary behavior. A Workflow with no root port declarations
// remains compatible with callers that declare no child port mapping.
func validateWorkflowFramePortBoundary(
	snapshot ProgramSnapshot,
	plans []PortPlan,
) error {
	if len(plans) == 0 || snapshot.InterpreterID != WorkflowInterpreterID {
		return nil
	}

	program, err := workflowProgramFromSnapshot(snapshot)
	if err != nil {
		return err
	}
	if len(program.Ports) == 0 {
		return nil
	}

	published := make(map[string]WorkflowPort, len(program.Ports))
	for _, port := range program.Ports {
		published[port.Name] = port
	}
	for _, plan := range plans {
		port, found := published[plan.Name]
		if !found {
			return errors.New("workflow child port is not published by the target workflow")
		}
		switch plan.Direction {
		case WorkflowPortDirectionInput:
			if port.Access != WorkflowPortAccessRead &&
				port.Access != WorkflowPortAccessReadWrite {
				return errors.New("workflow child input port is not readable in the target workflow")
			}
		case WorkflowPortDirectionOutput:
			if port.Access != WorkflowPortAccessWrite &&
				port.Access != WorkflowPortAccessReadWrite {
				return errors.New("workflow child output port is not writable in the target workflow")
			}
		default:
			return errors.New("workflow child port direction is invalid")
		}
		channel, found := workflowChannelForName(program, port.Channel)
		if !found {
			return errors.New("workflow published port has no declared channel")
		}
		if err := portSchemaCompatible(plan.ChannelSchema, channel.Schema); err != nil {
			return fmt.Errorf(
				"workflow child port %q caller channel schema is incompatible with target channel schema: %w",
				plan.Name,
				err,
			)
		}
		if err := portSchemaCompatible(plan.Schema, channel.Schema); err != nil {
			return fmt.Errorf(
				"workflow child port %q callable schema is incompatible with target channel schema: %w",
				plan.Name,
				err,
			)
		}
	}
	return nil
}
