package session

import (
	"context"
	"errors"
	"fmt"
)

type ToolRouter struct {
	executors map[ToolBindingKind]ToolExecutor
}

func NewToolRouter(
	executors map[ToolBindingKind]ToolExecutor,
) (*ToolRouter, error) {
	if len(executors) == 0 {
		return nil, errors.New("session tool router has no executors")
	}

	copied := make(map[ToolBindingKind]ToolExecutor, len(executors))
	for kind, executor := range executors {
		if kind == "" || executor == nil {
			continue
		}
		copied[kind] = executor
	}
	if len(copied) == 0 {
		return nil, errors.New("session tool router has no usable executors")
	}

	return &ToolRouter{executors: copied}, nil
}

func (r *ToolRouter) EvaluateTool(
	ctx context.Context,
	request ToolExecutionRequest,
) (ToolEvaluation, error) {
	executor, err := r.executorFor(request.ToolCall.Binding.Kind)
	if err != nil {
		return deniedToolEvaluation(err)
	}

	return executor.EvaluateTool(ctx, request)
}

func deniedToolEvaluation(err error) (ToolEvaluation, error) {
	return ToolEvaluation{
		Decision: ToolEvaluationDenied,
		Reason:   err.Error(),
		Result: &ToolExecutionResult{
			HasResult: true,
			Kind:      ToolResultKindError,
			Text:      err.Error(),
			Error:     err.Error(),
		},
	}, nil
}

func (r *ToolRouter) ExecuteTool(
	ctx context.Context,
	request ToolExecutionRequest,
) (ToolExecutionResult, error) {
	executor, err := r.executorFor(request.ToolCall.Binding.Kind)
	if err != nil {
		return ToolExecutionResult{}, err
	}

	return executor.ExecuteTool(ctx, request)
}

func (r *ToolRouter) ResolveToolWait(
	ctx context.Context,
	request ToolWaitResolutionRequest,
) (ToolWaitResolution, error) {
	executor, err := r.executorFor(request.ToolCall.Binding.Kind)
	if err != nil {
		return ToolWaitResolution{}, err
	}

	return executor.ResolveToolWait(ctx, request)
}

func (r *ToolRouter) executorFor(
	kind ToolBindingKind,
) (ToolExecutor, error) {
	executor := r.executors[kind]
	if executor == nil {
		return nil, fmt.Errorf(
			"no session tool executor is registered for binding kind %q",
			kind,
		)
	}
	return executor, nil
}
