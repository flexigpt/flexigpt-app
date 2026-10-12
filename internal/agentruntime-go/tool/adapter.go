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

type GoTargetInvoker interface {
	InvokeGoToolTarget(
		target composition.CapabilityTarget,
		args string,
		timeoutMS int,
	) (*llmtoolsutil.InvokeResponse, error)
}

type Adapter struct {
	invoker GoTargetInvoker
}

func NewAdapter(
	invoker GoTargetInvoker,
) (*Adapter, error) {
	if invoker == nil {
		return nil, errors.New("session go-tool adapter requires a target invoker")
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
	_ context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolExecutionResult, error) {
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

	response, err := a.invoker.InvokeGoToolTarget(
		binding.Target,
		string(request.ToolCall.Arguments),
		binding.TimeoutMS,
	)
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
