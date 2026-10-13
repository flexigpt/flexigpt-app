package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
)

type ContextGoTargetInvoker interface {
	InvokeGoToolTarget(
		ctx context.Context,
		target composition.CapabilityTarget,
		args string,
		timeoutMS int,
	) (*llmtoolsutil.InvokeResponse, error)
}

// LegacyGoTargetInvoker keeps existing ToolStore wrappers source-compatible
// while newer wrappers receive the Run cancellation context.
type LegacyGoTargetInvoker interface {
	InvokeGoToolTarget(
		target composition.CapabilityTarget,
		args string,
		timeoutMS int,
	) (*llmtoolsutil.InvokeResponse, error)
}

type Adapter struct {
	invoker any
}

func NewAdapter(
	invoker any,
) (*Adapter, error) {
	if invoker == nil {
		return nil, errors.New("session go-tool adapter requires a target invoker")
	}
	if _, ok := invoker.(ContextGoTargetInvoker); !ok {
		if _, legacy := invoker.(LegacyGoTargetInvoker); !legacy {
			return nil, errors.New(
				"session go-tool adapter target invoker has an unsupported signature",
			)
		}
	}

	return &Adapter{invoker: invoker}, nil
}

func (a *Adapter) EvaluateTool(
	_ context.Context,
	_ sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolEvaluation, error) {
	return sessionRuntime.ToolEvaluation{
		Decision: sessionRuntime.ToolEvaluationAllowed,
	}, nil
}

func (a *Adapter) ExecuteTool(
	ctx context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (res sessionRuntime.ToolExecutionResult, err error) {
	var binding struct {
		Target    composition.CapabilityTarget `json:"target"`
		TimeoutMS int                          `json:"timeoutMS,omitempty"`
	}
	if err := json.Unmarshal(request.ToolCall.Binding.Data, &binding); err != nil {
		return sessionRuntime.ToolExecutionResult{}, fmt.Errorf(
			"decode Go-tool Session binding: %w",
			err,
		)
	}

	var response *llmtoolsutil.InvokeResponse
	switch invoker := a.invoker.(type) {
	case ContextGoTargetInvoker:
		response, err = invoker.InvokeGoToolTarget(
			ctx,
			binding.Target,
			string(request.ToolCall.Arguments),
			binding.TimeoutMS,
		)
	case LegacyGoTargetInvoker:
		response, err = invoker.InvokeGoToolTarget(
			binding.Target,
			string(request.ToolCall.Arguments),
			binding.TimeoutMS,
		)
	default:
		return sessionRuntime.ToolExecutionResult{}, errors.New(
			"go-tool target invoker is unavailable",
		)
	}
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}
	if response == nil {
		return sessionRuntime.ToolExecutionResult{}, errors.New(
			"go-tool invoker returned no response",
		)
	}

	rawOutputs, marshalErr := json.Marshal(response.Outputs)
	if marshalErr != nil {
		return sessionRuntime.ToolExecutionResult{}, fmt.Errorf(
			"marshal Go-tool outputs: %w",
			marshalErr,
		)
	}

	result := sessionRuntime.ToolExecutionResult{
		HasResult:  true,
		Kind:       sessionRuntime.ToolResultKindSuccess,
		Text:       string(rawOutputs),
		Structured: rawOutputs,
	}
	if response.IsError {
		result.Kind = sessionRuntime.ToolResultKindError
		result.Error = response.ErrorMessage
		if result.Error == "" {
			result.Error = "Go tool returned an error result"
		}
	}

	return result, nil
}

func (a *Adapter) ResolveToolWait(
	_ context.Context,
	_ sessionRuntime.ToolWaitResolutionRequest,
) (sessionRuntime.ToolWaitResolution, error) {
	return sessionRuntime.ToolWaitResolution{}, errors.New(
		"go-tool execution does not expose an approval wait",
	)
}
