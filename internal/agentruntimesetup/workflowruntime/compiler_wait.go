package workflowruntime

import (
	"errors"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func compileWorkflowWait(input workflowv1.Wait) (sessionv3Runtime.WorkflowWait, error) {
	if input.For != "timer" {
		return sessionv3Runtime.WorkflowWait{}, unsupportedWorkflowOperation(
			"wait",
			input.For,
		)
	}
	if (len(input.After) == 0) == (len(input.At) == 0) {
		return sessionv3Runtime.WorkflowWait{}, errors.New(
			"timer wait requires exactly one of after or at",
		)
	}
	if input.Subscription != "" || input.Channel != "" ||
		len(input.Cursor) != 0 || len(input.Match) != 0 ||
		input.Consume != nil || len(input.Handle) != 0 ||
		input.Assignees != nil || len(input.Request) != 0 ||
		len(input.ResponseSchema) != 0 {
		return sessionv3Runtime.WorkflowWait{}, errors.New(
			"timer wait contains fields for another wait kind",
		)
	}

	output := sessionv3Runtime.WorkflowWait{}
	if len(input.After) != 0 {
		value, err := compileWorkflowValue(input.After)
		if err != nil {
			return sessionv3Runtime.WorkflowWait{}, err
		}
		output.After = &value
	}
	if len(input.At) != 0 {
		value, err := compileWorkflowValue(input.At)
		if err != nil {
			return sessionv3Runtime.WorkflowWait{}, err
		}
		output.At = &value
	}
	return output, nil
}

func workflowCallModes(values []string) []sessionv3Runtime.WorkflowCallMode {
	output := make([]sessionv3Runtime.WorkflowCallMode, len(values))
	for index, value := range values {
		output[index] = sessionv3Runtime.WorkflowCallMode(value)
	}
	return output
}
