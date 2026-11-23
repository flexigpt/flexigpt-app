package sessionv3

import (
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// captureWorkflowCallPortInputs freezes all currently visible input-port
// records at the logical call boundary. Runtime staging later reuses these
// facts and never observes channel records appended after this boundary.
func captureWorkflowCallPortInputs(
	execution Execution,
	plans []PortPlan,
) []PortInput {
	output := make([]PortInput, 0)
	for _, plan := range plans {
		if plan.Direction != WorkflowPortDirectionInput {
			continue
		}
		input := PortInput{
			Name:     plan.Name,
			Channel:  plan.Channel,
			Streamed: plan.Stream,
			Captured: true,
		}
		channel := executionChannel(&execution, plan.Channel)
		if channel != nil {
			input.Cursor = channel.NextSequence
			for _, message := range channel.Messages {
				input.Values = append(input.Values, message.Data.Clone())
			}
		}
		output = append(output, input)
	}
	return output
}

func capturedWorkflowCallPortInputs(
	execution Execution,
	call WorkflowCall,
	plans []PortPlan,
) []PortInput {
	if len(call.PortInputs) != 0 {
		return clonePortInputs(call.PortInputs)
	}
	return captureWorkflowCallPortInputs(execution, plans)
}

func validateWorkflowCallContext(
	contracts ...*WorkflowCallableInterface,
) func(jsonutil.CanonicalValue) error {
	return func(value jsonutil.CanonicalValue) error {
		context, err := jsonutil.DecodeCanonicalValue[map[string]jsonutil.CanonicalValue](value)
		if err != nil {
			return errors.New("workflow call context must be an object")
		}
		for _, contract := range contracts {
			if contract == nil {
				continue
			}
			for name, current := range context {
				requirement, found := workflowContextRequirement(
					contract.Context,
					name,
				)
				if !found {
					return fmt.Errorf(
						"workflow callable interface does not accept context form %q",
						name,
					)
				}
				if err := validateWorkflowSchema(
					requirement.Schema,
					current,
				); err != nil {
					return fmt.Errorf(
						"validate workflow context %q: %w",
						name,
						err,
					)
				}
			}
		}
		return nil
	}
}

func (i *WorkflowInterpreter) workflowCallPortPlans(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	path string,
	call WorkflowCall,
	target WorkflowTarget,
) ([]PortPlan, error) {
	contracts := []*WorkflowCallableInterface{
		target.Contract,
		call.Contract,
		workflowTargetSetContract(program, call),
	}
	output := make([]PortPlan, 0, len(call.Ports))
	for _, binding := range call.Ports {
		var port WorkflowCallablePort
		found := false
		for _, contract := range contracts {
			if contract == nil {
				continue
			}
			value, currentFound := workflowCallablePort(
				contract.Ports,
				binding.Name,
			)
			if !currentFound {
				continue
			}
			port = value
			found = true
			break
		}
		if !found {
			return nil, fmt.Errorf(
				"workflow call port %q is unavailable",
				binding.Name,
			)
		}

		channel, channelFound := workflowChannelForName(
			program,
			binding.Channel,
		)
		if !channelFound {
			return nil, fmt.Errorf(
				"workflow call port %q channel is unavailable",
				binding.Name,
			)
		}
		switch port.Direction {
		case WorkflowPortDirectionInput:
			if !workflowChannelAllows(
				channel.Readers,
				state.PendingDeclarationPath,
			) {
				return nil, fmt.Errorf(
					"workflow call port %q channel read is not allowed",
					binding.Name,
				)
			}
		case WorkflowPortDirectionOutput:
			if !workflowChannelAllows(
				channel.Writers,
				state.PendingDeclarationPath,
			) {
				return nil, fmt.Errorf(
					"workflow call port %q channel write is not allowed",
					binding.Name,
				)
			}
		default:
			return nil, errors.New("workflow callable port direction is invalid")
		}

		access, err := portAccessForDirection(port.Direction)
		if err != nil {
			return nil, err
		}
		output = append(output, PortPlan{
			Name:            binding.Name,
			Direction:       port.Direction,
			Stream:          port.Stream,
			Schema:          port.Schema.Clone(),
			ChannelSchema:   channel.Schema.Clone(),
			Channel:         channel.Name,
			Access:          access,
			MaxMessages:     channel.MaxMessages,
			MaxBytes:        channel.MaxBytes,
			OwnerFrameID:    frame.ID,
			DeclarationPath: state.PendingDeclarationPath,
		})
	}
	if err := validateUniquePortPlans(output); err != nil {
		return nil, err
	}
	_ = state
	_ = path
	return output, nil
}

func workflowProgramWithPortInputs(
	snapshot ProgramSnapshot,
	inputs []PortInput,
) (ProgramSnapshot, error) {
	if len(inputs) == 0 {
		return snapshot, nil
	}
	if snapshot.InterpreterID != WorkflowInterpreterID {
		return ProgramSnapshot{}, errors.New(
			"target port adapter is unavailable for this child interpreter",
		)
	}

	program, err := workflowProgramFromSnapshot(snapshot)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	program.Scope.Ports = clonePortInputs(inputs)
	payload, err := canonicalWorkflowValue(program)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	snapshot.Payload = payload
	return snapshot, nil
}
