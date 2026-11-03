package toolruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

// GoToolCaller is the narrow application capability used to invoke a Go Tool.
// It contains no Artifact Store dependency and is shared by v2 and v3 setup.
type GoToolCaller interface {
	CallGoTool(
		ctx context.Context,
		function string,
		args json.RawMessage,
		timeout time.Duration,
	) ([]llmtoolsSpec.ToolOutputUnion, error)
}

// Router routes Session Tool work by its durable binding kind. Session owns
// Tool-call lifecycle state; Tool Runtime owns executor registration.
type Router struct {
	executors map[sessionRuntime.ToolBindingKind]sessionRuntime.ToolExecutor
}

func NewRouter(
	executors map[sessionRuntime.ToolBindingKind]sessionRuntime.ToolExecutor,
) (*Router, error) {
	if len(executors) == 0 {
		return nil, errors.New("tool runtime router has no executors")
	}

	copied := make(
		map[sessionRuntime.ToolBindingKind]sessionRuntime.ToolExecutor,
		len(executors),
	)
	for kind, executor := range executors {
		if kind == "" || executor == nil {
			continue
		}
		copied[kind] = executor
	}
	if len(copied) == 0 {
		return nil, errors.New("tool runtime router has no usable executors")
	}

	return &Router{executors: copied}, nil
}

func (r *Router) EvaluateTool(
	ctx context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolEvaluation, error) {
	executor, err := r.executorFor(request.ToolCall.Binding.Kind)
	if err != nil {
		return deniedToolEvaluation(err)
	}

	return executor.EvaluateTool(ctx, request)
}

func (r *Router) ExecuteTool(
	ctx context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolExecutionResult, error) {
	executor, err := r.executorFor(request.ToolCall.Binding.Kind)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}

	return executor.ExecuteTool(ctx, request)
}

func (r *Router) ResolveToolWait(
	ctx context.Context,
	request sessionRuntime.ToolWaitResolutionRequest,
) (sessionRuntime.ToolWaitResolution, error) {
	executor, err := r.executorFor(request.ToolCall.Binding.Kind)
	if err != nil {
		return sessionRuntime.ToolWaitResolution{}, err
	}

	return executor.ResolveToolWait(ctx, request)
}

func (r *Router) executorFor(
	kind sessionRuntime.ToolBindingKind,
) (sessionRuntime.ToolExecutor, error) {
	if r == nil {
		return nil, errors.New("tool runtime router is unavailable")
	}

	executor := r.executors[kind]
	if executor == nil {
		return nil, fmt.Errorf(
			"no tool runtime executor is registered for binding kind %q",
			kind,
		)
	}
	return executor, nil
}

func deniedToolEvaluation(
	err error,
) (sessionRuntime.ToolEvaluation, error) {
	return sessionRuntime.ToolEvaluation{
		Decision: sessionRuntime.ToolEvaluationDenied,
		Reason:   err.Error(),
		Result: &sessionRuntime.ToolExecutionResult{
			HasResult: true,
			Kind:      sessionRuntime.ToolResultKindError,
			Text:      err.Error(),
			Error:     err.Error(),
		},
	}, nil
}
