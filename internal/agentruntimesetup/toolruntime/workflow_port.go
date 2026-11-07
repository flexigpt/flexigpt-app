package toolruntime

import (
	"context"
	"errors"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

var ErrWorkflowGoToolPortTransportUnavailable = errors.New(
	"workflow go tool port transport is unavailable",
)

type WorkflowGoToolInvocation struct {
	Context     sessionv3Runtime.CanonicalValue
	PortInputs  []sessionv3Runtime.PortInput
	OutputPorts []string
	Stream      sessionv3Runtime.WorkPortStream
}

type WorkflowGoToolCaller interface {
	GoToolCaller

	CallGoToolWithWorkflowInvocation(
		ctx context.Context,
		function string,
		args []byte,
		timeoutMillis int,
		invocation WorkflowGoToolInvocation,
	) ([]llmtoolsSpec.ToolOutputUnion, error)
}

func NewWorkflowGoToolInvocation(
	request sessionv3Runtime.WorkDispatchRequest,
	payload *sessionv3Runtime.ToolWork,
) WorkflowGoToolInvocation {
	if payload == nil {
		return WorkflowGoToolInvocation{}
	}

	output := WorkflowGoToolInvocation{
		Context:    payload.Context.Clone(),
		PortInputs: cloneWorkflowGoPortInputs(request.WorkItem.PortInputs),
		Stream:     request.PortStream,
	}
	for _, plan := range request.WorkItem.Ports {
		if plan.Direction != sessionv3Runtime.WorkflowPortDirectionOutput ||
			!plan.Stream {
			continue
		}
		output.OutputPorts = append(output.OutputPorts, plan.Name)
	}
	return output
}

func (i WorkflowGoToolInvocation) RequiresTransport() bool {
	return len(i.Context) != 0 ||
		len(i.PortInputs) != 0 ||
		len(i.OutputPorts) != 0
}

func ValidateWorkflowGoToolTransport(
	caller GoToolCaller,
	invocation WorkflowGoToolInvocation,
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
	args []byte,
	timeoutMillis int,
	invocation WorkflowGoToolInvocation,
) ([]llmtoolsSpec.ToolOutputUnion, error) {
	if err := ValidateWorkflowGoToolTransport(caller, invocation); err != nil {
		return nil, err
	}
	if extended, found := caller.(WorkflowGoToolCaller); found {
		return extended.CallGoToolWithWorkflowInvocation(
			ctx,
			function,
			args,
			timeoutMillis,
			invocation,
		)
	}
	return caller.CallGoTool(
		ctx,
		function,
		args,
		durationMilliseconds(timeoutMillis),
	)
}

func cloneWorkflowGoPortInputs(
	values []sessionv3Runtime.PortInput,
) []sessionv3Runtime.PortInput {
	if len(values) == 0 {
		return nil
	}

	output := make([]sessionv3Runtime.PortInput, len(values))
	for index := range values {
		output[index] = values[index]
		output[index].Values = make(
			[]sessionv3Runtime.CanonicalValue,
			len(values[index].Values),
		)
		for valueIndex := range values[index].Values {
			output[index].Values[valueIndex] = values[index].Values[valueIndex].Clone()
		}
	}
	return output
}
