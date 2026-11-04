package workflowruntime

import (
	"errors"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func compileWorkflowWait(
	document workflowv1.WorkflowDocument,
	input workflowv1.Wait,
) (sessionv3Runtime.WorkflowWait, error) {
	switch input.For {
	case "timer":
		if (len(input.After) == 0) == (len(input.At) == 0) {
			return sessionv3Runtime.WorkflowWait{}, errors.New(
				"timer wait requires exactly one of after or at",
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

	case "event":
		if input.Subscription == "" {
			return sessionv3Runtime.WorkflowWait{}, errors.New(
				"event wait requires a subscription",
			)
		}
		output := sessionv3Runtime.WorkflowEventWait{
			Subscription: input.Subscription,
		}
		if len(input.Match) != 0 {
			value, err := compileWorkflowValue(input.Match)
			if err != nil {
				return sessionv3Runtime.WorkflowWait{}, err
			}
			output.Match = &value
		}
		if input.Consume != nil {
			output.Consume = *input.Consume
		}
		return sessionv3Runtime.WorkflowWait{Event: &output}, nil

	case "channel":
		if input.Channel == "" {
			return sessionv3Runtime.WorkflowWait{}, errors.New(
				"channel wait requires a channel",
			)
		}
		output := sessionv3Runtime.WorkflowChannelWait{
			Channel: input.Channel,
		}
		if len(input.Cursor) != 0 {
			value, err := compileWorkflowValue(input.Cursor)
			if err != nil {
				return sessionv3Runtime.WorkflowWait{}, err
			}
			output.Cursor = &value
		}
		if len(input.Match) != 0 {
			value, err := compileWorkflowValue(input.Match)
			if err != nil {
				return sessionv3Runtime.WorkflowWait{}, err
			}
			output.Match = &value
		}
		if input.Consume != nil {
			output.Consume = *input.Consume
		}
		return sessionv3Runtime.WorkflowWait{Channel: &output}, nil

	case "completion":
		if len(input.Handle) == 0 {
			return sessionv3Runtime.WorkflowWait{}, errors.New(
				"completion wait requires a handle",
			)
		}
		handle, err := compileWorkflowValue(input.Handle)
		if err != nil {
			return sessionv3Runtime.WorkflowWait{}, err
		}
		return sessionv3Runtime.WorkflowWait{
			Completion: &sessionv3Runtime.WorkflowCompletionWait{
				Handle: handle,
			},
		}, nil

	case "human":
		if input.Assignees == nil || len(input.ResponseSchema) == 0 {
			return sessionv3Runtime.WorkflowWait{}, errors.New(
				"human wait requires assignees and a response schema",
			)
		}
		responseSchema, err := compileWorkflowSchema(document, input.ResponseSchema)
		if err != nil {
			return sessionv3Runtime.WorkflowWait{}, err
		}
		output := sessionv3Runtime.WorkflowHumanWait{
			ResponseSchema: responseSchema,
		}
		if len(input.Assignees.Users) != 0 {
			value, err := compileWorkflowValue(input.Assignees.Users)
			if err != nil {
				return sessionv3Runtime.WorkflowWait{}, err
			}
			output.Users = &value
		}
		if len(input.Assignees.Roles) != 0 {
			value, err := compileWorkflowValue(input.Assignees.Roles)
			if err != nil {
				return sessionv3Runtime.WorkflowWait{}, err
			}
			output.Roles = &value
		}
		request, err := compileWorkflowValue(input.Request)
		if err != nil {
			return sessionv3Runtime.WorkflowWait{}, err
		}
		output.Request = request
		return sessionv3Runtime.WorkflowWait{Human: &output}, nil

	default:
		return sessionv3Runtime.WorkflowWait{}, unsupportedWorkflowOperation(
			"wait",
			input.For,
		)
	}
}

func workflowCallModes(values []string) []sessionv3Runtime.WorkflowCallMode {
	output := make([]sessionv3Runtime.WorkflowCallMode, len(values))
	for index, value := range values {
		output[index] = sessionv3Runtime.WorkflowCallMode(value)
	}
	return output
}
