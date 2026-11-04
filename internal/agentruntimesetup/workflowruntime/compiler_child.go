package workflowruntime

import (
	"errors"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func compileWorkflowChildOwnership(
	input workflowv1.Ownership,
) (sessionv3Runtime.WorkflowChildOwnership, error) {
	owner, err := compileWorkflowValue(input.Owner)
	if err != nil {
		return sessionv3Runtime.WorkflowChildOwnership{}, err
	}
	budget, err := compileWorkflowExecutionBudget(input.Budgets)
	if err != nil {
		return sessionv3Runtime.WorkflowChildOwnership{}, err
	}
	output := sessionv3Runtime.WorkflowChildOwnership{
		Owner:            owner,
		CancelWithParent: input.CancelWithParent,
		Budget:           budget,
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.WorkflowChildOwnership{}, err
	}
	return output, nil
}

func compileWorkflowExecutionBudget(
	input workflowv1.Limits,
) (sessionv3Runtime.ExecutionBudget, error) {
	output := sessionv3Runtime.ExecutionBudget{
		StepExecutions: valueOrZero(input.StepExecutions),
		CallAttempts:   valueOrZero(input.CallAttempts),
		WorkflowDepth:  valueOrZero(input.WorkflowDepth),
		Concurrency:    valueOrZero(input.Concurrency),
		ModelTokens:    valueOrZero(input.ModelTokens),
		MessageCount:   valueOrZero(input.MessageCount),
		MessageBytes:   valueOrZero(input.MessageBytes),
		EventCount:     valueOrZero(input.EventCount),
		EventBytes:     valueOrZero(input.EventBytes),
	}

	var err error
	if input.Duration != "" {
		output.Duration, err = sessionv3Runtime.ParseWorkflowDuration(input.Duration)
		if err != nil {
			return sessionv3Runtime.ExecutionBudget{}, err
		}
	}
	if input.CleanupDuration != "" {
		output.CleanupDuration, err = sessionv3Runtime.ParseWorkflowDuration(
			input.CleanupDuration,
		)
		if err != nil {
			return sessionv3Runtime.ExecutionBudget{}, err
		}
	}
	if input.Cost != nil {
		if input.Cost.Amount < 0 || input.Cost.Currency == "" {
			return sessionv3Runtime.ExecutionBudget{}, errors.New(
				"workflow child budget cost is invalid",
			)
		}
		output.Cost = &sessionv3Runtime.ExecutionBudgetCost{
			Amount:   input.Cost.Amount,
			Currency: input.Cost.Currency,
		}
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.ExecutionBudget{}, err
	}
	return output, nil
}

func valueOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
