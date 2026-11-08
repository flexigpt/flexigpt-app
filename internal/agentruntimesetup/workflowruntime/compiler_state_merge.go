package workflowruntime

import (
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func compileWorkflowParentStateMerge(
	input *workflowv1.StateMerge,
) (*sessionv3Runtime.WorkflowParentStateMerge, error) {
	if input == nil {
		return nil, nil
	}

	keys := make([]string, 0, len(input.Values))
	for key := range input.Values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	output := &sessionv3Runtime.WorkflowParentStateMerge{
		Values: make(
			[]sessionv3Runtime.WorkflowValuePair,
			0,
			len(keys),
		),
		Remove: append([]string(nil), input.Remove...),
	}
	for _, key := range keys {
		value, err := compileWorkflowValue(input.Values[key])
		if err != nil {
			return nil, err
		}
		output.Values = append(output.Values, sessionv3Runtime.WorkflowValuePair{
			Key:   key,
			Value: value,
		})
	}
	if err := output.Validate(); err != nil {
		return nil, err
	}
	return output, nil
}
