package workflowruntime

import (
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func compileWorkflowCatchHandlers(
	document workflowv1.WorkflowDocument,
	values []workflowv1.Handler,
) ([]sessionv3Runtime.WorkflowCatchHandler, error) {
	if len(values) == 0 {
		return nil, nil
	}

	output := make(
		[]sessionv3Runtime.WorkflowCatchHandler,
		0,
		len(values),
	)
	for index := range values {
		match, err := compileWorkflowErrorMatch(
			document,
			values[index].Match,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"compile workflow catch handler %d: %w",
				index,
				err,
			)
		}
		block, err := compileWorkflowBlock(
			document,
			values[index].Do,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"compile workflow catch handler %d block: %w",
				index,
				err,
			)
		}
		output = append(output, sessionv3Runtime.WorkflowCatchHandler{
			Match: match,
			Do:    block,
		})
	}
	return output, nil
}

func compileWorkflowErrorMatch(
	document workflowv1.WorkflowDocument,
	input workflowv1.ErrorMatch,
) (sessionv3Runtime.WorkflowErrorMatch, error) {
	output := sessionv3Runtime.WorkflowErrorMatch{
		Codes:     append([]string(nil), input.Codes...),
		Retryable: input.Retryable,
	}
	for _, category := range input.Categories {
		value, err := compileWorkflowErrorCategory(category)
		if err != nil {
			return sessionv3Runtime.WorkflowErrorMatch{}, err
		}
		output.Categories = append(output.Categories, value)
	}
	for _, effect := range input.EffectStates {
		var value sessionv3Runtime.EffectState
		switch effect {
		case "not-applied":
			value = sessionv3Runtime.EffectStateConfirmedNotApplied
		case "applied":
			value = sessionv3Runtime.EffectStateConfirmedApplied
		case "unknown":
			value = sessionv3Runtime.EffectStateUncertain
		default:
			return sessionv3Runtime.WorkflowErrorMatch{}, fmt.Errorf(
				"unsupported workflow error effect state %q",
				effect,
			)
		}
		output.EffectStates = append(output.EffectStates, value)
	}
	if len(input.Schema) != 0 {
		schema, err := compileWorkflowSchema(document, input.Schema)
		if err != nil {
			return sessionv3Runtime.WorkflowErrorMatch{}, err
		}
		output.Schema = schema
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.WorkflowErrorMatch{}, err
	}
	return output, nil
}
