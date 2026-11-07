package toolruntime

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/tool"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

var ErrWorkflowGoToolPortTransportUnavailable = errors.New(
	"workflow go tool port transport is unavailable",
)

type WorkflowGoToolCaller interface {
	GoToolCaller

	CallGoToolWithWorkflowInvocation(
		ctx context.Context,
		function string,
		args json.RawMessage,
		timeout time.Duration,
		invocation tool.Invocation,
	) ([]llmtoolsSpec.ToolOutputUnion, error)
}

// WorkflowGoToolCallerAdapter is an application-side wrapper around an
// existing GoToolCaller. V2 callers continue to use CallGoTool unchanged.
// V3 Workflow callers use the additive invocation sidecar.
type WorkflowGoToolCallerAdapter struct {
	caller GoToolCaller
}

func NewWorkflowGoToolCaller(
	caller GoToolCaller,
) (*WorkflowGoToolCallerAdapter, error) {
	if caller == nil {
		return nil, errors.New("workflow go tool caller is unavailable")
	}
	return &WorkflowGoToolCallerAdapter{caller: caller}, nil
}

func (a *WorkflowGoToolCallerAdapter) CallGoTool(
	ctx context.Context,
	function string,
	args json.RawMessage,
	timeout time.Duration,
) ([]llmtoolsSpec.ToolOutputUnion, error) {
	if a == nil || a.caller == nil {
		return nil, ErrWorkflowGoToolPortTransportUnavailable
	}
	return a.caller.CallGoTool(ctx, function, args, timeout)
}

func (a *WorkflowGoToolCallerAdapter) CallGoToolWithWorkflowInvocation(
	ctx context.Context,
	function string,
	args json.RawMessage,
	timeout time.Duration,
	invocation tool.Invocation,
) ([]llmtoolsSpec.ToolOutputUnion, error) {
	if a == nil || a.caller == nil {
		return nil, ErrWorkflowGoToolPortTransportUnavailable
	}

	invocation.Version = tool.Version
	return a.caller.CallGoTool(
		tool.WithInvocation(ctx, invocation),
		function,
		append(json.RawMessage(nil), args...),
		timeout,
	)
}

func NewWorkflowGoToolInvocation(
	request sessionv3Runtime.WorkDispatchRequest,
	payload *sessionv3Runtime.ToolWork,
) tool.Invocation {
	if payload == nil {
		return tool.Invocation{}
	}

	output := tool.Invocation{
		Version: tool.Version,
		Context: append(json.RawMessage(nil), payload.Context...),
	}
	for _, input := range request.WorkItem.PortInputs {
		value := tool.PortInput{
			Name:     input.Name,
			Channel:  input.Channel,
			Cursor:   input.Cursor,
			Streamed: input.Streamed,
			Values:   make([]json.RawMessage, 0, len(input.Values)),
		}
		for _, current := range input.Values {
			value.Values = append(
				value.Values,
				json.RawMessage(current.Clone()),
			)
		}
		output.PortInputs = append(output.PortInputs, value)
	}
	for _, plan := range request.WorkItem.Ports {
		if plan.Direction != sessionv3Runtime.WorkflowPortDirectionOutput ||
			!plan.Stream {
			continue
		}
		output.OutputPorts = append(output.OutputPorts, plan.Name)
	}
	if len(output.OutputPorts) != 0 && request.PortStream.Emit != nil {
		output.Emit = func(
			name string,
			value json.RawMessage,
		) error {
			canonical, err := sessionv3Runtime.CanonicalValueFrom(value)
			if err != nil {
				return err
			}
			return request.PortStream.Emit(name, canonical)
		}
	}
	return output
}

func ValidateWorkflowGoToolTransport(
	caller GoToolCaller,
	invocation tool.Invocation,
) error {
	if !invocation.RequiresTransport() {
		return nil
	}
	if _, found := caller.(WorkflowGoToolCaller); !found {
		return ErrWorkflowGoToolPortTransportUnavailable
	}
	return nil
}

func CallGoToolWithWorkflowInvocation(
	ctx context.Context,
	caller GoToolCaller,
	function string,
	args json.RawMessage,
	timeout time.Duration,
	invocation tool.Invocation,
) ([]llmtoolsSpec.ToolOutputUnion, error) {
	if err := ValidateWorkflowGoToolTransport(caller, invocation); err != nil {
		return nil, err
	}
	if extended, found := caller.(WorkflowGoToolCaller); found {
		return extended.CallGoToolWithWorkflowInvocation(
			ctx,
			function,
			args,
			timeout,
			invocation,
		)
	}
	return caller.CallGoTool(ctx, function, args, timeout)
}
