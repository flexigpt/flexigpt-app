package sessionv3

import (
	"errors"
	"fmt"
)

func validateWorkflowCallContext(
	contracts ...*WorkflowCallableInterface,
) func(CanonicalValue) error {
	return func(value CanonicalValue) error {
		context, err := DecodeCanonicalValue[map[string]CanonicalValue](value)
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

func validateWorkflowCallPorts(
	call WorkflowCall,
	target WorkflowTarget,
	setContract *WorkflowCallableInterface,
) error {
	contracts := []*WorkflowCallableInterface{
		target.Contract,
		call.Contract,
		setContract,
	}
	for _, binding := range call.Ports {
		if err := binding.Validate(); err != nil {
			return err
		}
		var expected *WorkflowCallablePort
		for _, contract := range contracts {
			if contract == nil {
				continue
			}
			port, found := workflowCallablePort(contract.Ports, binding.Name)
			if !found {
				return fmt.Errorf(
					"workflow callable interface has no port %q",
					binding.Name,
				)
			}
			if expected == nil {
				copyValue := port
				expected = &copyValue
				continue
			}
			if expected.Direction != port.Direction ||
				expected.Stream != port.Stream {
				return fmt.Errorf(
					"workflow callable port %q has incompatible contracts",
					binding.Name,
				)
			}
			if err := portSchemaCompatible(
				expected.Schema,
				port.Schema,
			); err != nil {
				return fmt.Errorf(
					"workflow callable port %q schema: %w",
					binding.Name,
					err,
				)
			}
		}
		if expected == nil {
			return fmt.Errorf(
				"workflow call port %q has no target contract",
				binding.Name,
			)
		}
		requiredAccess, err := portAccessForDirection(expected.Direction)
		if err != nil {
			return err
		}
		if binding.Access != requiredAccess {
			return fmt.Errorf(
				"workflow call port %q access does not match direction",
				binding.Name,
			)
		}
	}
	return nil
}

func (i *WorkflowInterpreter) workflowCallPortPlans(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	path string,
	call WorkflowCall,
	target WorkflowTarget,
) ([]PortPlan, error) {
	setContract := call.SetContract
	if effective := workflowTargetSetContract(program, call); effective != nil {
		setContract = effective
	}
	if err := validateWorkflowCallPorts(call, target, setContract); err != nil {
		return nil, err
	}

	contracts := []*WorkflowCallableInterface{
		target.Contract,
		call.Contract,
		setContract,
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
			if !workflowChannelAllows(channel.Readers, path) {
				return nil, fmt.Errorf(
					"workflow call port %q channel read is not allowed",
					binding.Name,
				)
			}
		case WorkflowPortDirectionOutput:
			if !workflowChannelAllows(channel.Writers, path) {
				return nil, fmt.Errorf(
					"workflow call port %q channel write is not allowed",
					binding.Name,
				)
			}
		default:
			return nil, errors.New("workflow callable port direction is invalid")
		}
		if err := portSchemaCompatible(port.Schema, channel.Schema); err != nil {
			return nil, fmt.Errorf(
				"workflow call port %q schema: %w",
				binding.Name,
				err,
			)
		}

		access, err := portAccessForDirection(port.Direction)
		if err != nil {
			return nil, err
		}
		output = append(output, PortPlan{
			Name:         binding.Name,
			Direction:    port.Direction,
			Stream:       port.Stream,
			Schema:       port.Schema.Clone(),
			Channel:      channel.Name,
			Access:       access,
			MaxMessages:  channel.MaxMessages,
			MaxBytes:     channel.MaxBytes,
			OwnerFrameID: frame.ID,
		})
	}
	if err := validateUniquePortPlans(output); err != nil {
		return nil, err
	}
	_ = state
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
