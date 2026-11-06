package workflowruntime

import (
	"errors"
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func compileWorkflowSaga(
	document workflowv1.WorkflowDocument,
	input workflowv1.Saga,
) (sessionv3Runtime.WorkflowSaga, error) {
	block, err := compileWorkflowBlock(document, input.Do)
	if err != nil {
		return sessionv3Runtime.WorkflowSaga{}, err
	}

	policy := sessionv3Runtime.CompensationErrorPolicyContinue
	if input.CompensationErrors != "" {
		policy = sessionv3Runtime.CompensationErrorPolicy(
			input.CompensationErrors,
		)
	}
	concurrency := 1
	if input.Concurrency != nil {
		concurrency = *input.Concurrency
	}

	output := sessionv3Runtime.WorkflowSaga{
		CompensateOn: make(
			[]sessionv3Runtime.WorkflowSagaOutcome,
			0,
			len(input.CompensateOn),
		),
		CompensationErrors: policy,
		Concurrency:        concurrency,
		Do:                 block,
	}
	for _, value := range input.CompensateOn {
		switch value {
		case string(sessionv3Runtime.WorkflowSagaOutcomeFailed):
			output.CompensateOn = append(
				output.CompensateOn,
				sessionv3Runtime.WorkflowSagaOutcomeFailed,
			)
		case string(sessionv3Runtime.WorkflowSagaOutcomeCancelled):
			output.CompensateOn = append(
				output.CompensateOn,
				sessionv3Runtime.WorkflowSagaOutcomeCancelled,
			)
		case "uncertain":
			return sessionv3Runtime.WorkflowSaga{}, errors.New(
				"workflow saga does not compensate uncertain effects before reconciliation",
			)
		default:
			return sessionv3Runtime.WorkflowSaga{}, fmt.Errorf(
				"workflow saga compensation outcome %q is invalid",
				value,
			)
		}
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.WorkflowSaga{}, err
	}
	return output, nil
}
