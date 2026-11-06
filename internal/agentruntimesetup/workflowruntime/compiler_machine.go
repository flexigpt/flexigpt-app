package workflowruntime

import (
	"errors"
	"fmt"
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func compileWorkflowMachine(
	document workflowv1.WorkflowDocument,
	input workflowv1.Machine,
) (sessionv3Runtime.WorkflowMachine, error) {
	listen, err := compileWorkflowSubscriptions(
		document,
		input.Listen,
	)
	if err != nil {
		return sessionv3Runtime.WorkflowMachine{}, err
	}
	if len(listen) != len(input.Listen) {
		return sessionv3Runtime.WorkflowMachine{}, errors.New(
			"workflow machine listener compilation was incomplete",
		)
	}

	state, err := compileWorkflowRepeatState(
		document,
		input.State,
	)
	if err != nil {
		return sessionv3Runtime.WorkflowMachine{}, err
	}

	names := make([]string, 0, len(input.States))
	for name := range input.States {
		names = append(names, name)
	}
	sort.Strings(names)

	output := sessionv3Runtime.WorkflowMachine{
		Initial: input.Initial,
		State:   state,
		Listen:  listen,
		States:  make([]sessionv3Runtime.WorkflowMachineState, 0, len(names)),
	}
	for _, name := range names {
		source := input.States[name]
		value, valueErr := compileWorkflowMachineState(
			document,
			name,
			source,
		)
		if valueErr != nil {
			return sessionv3Runtime.WorkflowMachine{}, valueErr
		}
		output.States = append(output.States, value)
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.WorkflowMachine{}, err
	}
	return output, nil
}

func compileWorkflowMachineState(
	document workflowv1.WorkflowDocument,
	name string,
	input workflowv1.MachineState,
) (sessionv3Runtime.WorkflowMachineState, error) {
	output := sessionv3Runtime.WorkflowMachineState{
		Name:  name,
		Final: input.Final,
	}
	if input.Entry != nil {
		value, err := compileWorkflowBlock(document, *input.Entry)
		if err != nil {
			return sessionv3Runtime.WorkflowMachineState{}, err
		}
		output.Entry = &value
	}
	if input.Exit != nil {
		value, err := compileWorkflowBlock(document, *input.Exit)
		if err != nil {
			return sessionv3Runtime.WorkflowMachineState{}, err
		}
		output.Exit = &value
	}
	if !input.Final && len(input.Output) != 0 {
		return sessionv3Runtime.WorkflowMachineState{}, fmt.Errorf(
			"non-final workflow machine state %q has an output",
			name,
		)
	}
	if len(input.Output) != 0 {
		value, err := compileWorkflowValue(input.Output)
		if err != nil {
			return sessionv3Runtime.WorkflowMachineState{}, err
		}
		output.Output = &value
	}
	output.Transitions = make(
		[]sessionv3Runtime.WorkflowMachineTransition,
		0,
		len(input.Transitions),
	)
	for index := range input.Transitions {
		transition, err := compileWorkflowMachineTransition(
			document,
			input.Transitions[index],
		)
		if err != nil {
			return sessionv3Runtime.WorkflowMachineState{}, fmt.Errorf(
				"transition %d: %w",
				index,
				err,
			)
		}
		output.Transitions = append(output.Transitions, transition)
	}
	return output, output.Validate()
}

func compileWorkflowMachineTransition(
	document workflowv1.WorkflowDocument,
	input workflowv1.Transition,
) (sessionv3Runtime.WorkflowMachineTransition, error) {
	output := sessionv3Runtime.WorkflowMachineTransition{
		Event: input.Event,
		To:    input.To,
	}
	if input.After != "" {
		value, err := sessionv3Runtime.ParseWorkflowDuration(
			input.After,
		)
		if err != nil {
			return sessionv3Runtime.WorkflowMachineTransition{}, err
		}
		output.After = value
	}
	if len(input.When) != 0 {
		value, err := compileWorkflowValue(input.When)
		if err != nil {
			return sessionv3Runtime.WorkflowMachineTransition{}, err
		}
		output.When = &value
	}
	if input.Do != nil {
		value, err := compileWorkflowBlock(document, *input.Do)
		if err != nil {
			return sessionv3Runtime.WorkflowMachineTransition{}, err
		}
		output.Do = &value
	}
	return output, output.Validate()
}
