package workflowruntime

import (
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func compileWorkflowValueObject(
	values map[string]workflowv1.Value,
) (sessionv3Runtime.WorkflowValue, error) {
	if len(values) == 0 {
		return sessionv3Runtime.WorkflowValue{
			Kind:    sessionv3Runtime.WorkflowValueLiteral,
			Literal: sessionv3Runtime.MustCanonicalValue(struct{}{}),
		}, nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	output := sessionv3Runtime.WorkflowValue{
		Kind:   sessionv3Runtime.WorkflowValueObject,
		Object: make([]sessionv3Runtime.WorkflowValuePair, 0, len(keys)),
	}
	for _, key := range keys {
		value, err := compileWorkflowValue(values[key])
		if err != nil {
			return sessionv3Runtime.WorkflowValue{}, err
		}
		output.Object = append(output.Object, sessionv3Runtime.WorkflowValuePair{
			Key:   key,
			Value: value,
		})
	}
	return output, nil
}
