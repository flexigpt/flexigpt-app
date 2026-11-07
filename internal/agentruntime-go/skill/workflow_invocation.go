package skill

import (
	"context"
	"encoding/json"
)

type WorkflowPortInput struct {
	Name     string            `json:"name"`
	Channel  string            `json:"channel"`
	Cursor   uint64            `json:"cursor,omitempty"`
	Values   []json.RawMessage `json:"values,omitempty"`
	Streamed bool              `json:"streamed,omitempty"`
}

type WorkflowToolInvocation struct {
	Context     json.RawMessage                     `json:"context,omitempty"`
	PortInputs  []WorkflowPortInput                 `json:"portInputs,omitempty"`
	OutputPorts []string                            `json:"outputPorts,omitempty"`
	Emit        func(string, json.RawMessage) error `json:"-"`
}

type workflowToolInvocationContextKey struct{}

func WithWorkflowToolInvocation(
	ctx context.Context,
	value WorkflowToolInvocation,
) context.Context {
	return context.WithValue(
		ctx,
		workflowToolInvocationContextKey{},
		cloneWorkflowToolInvocation(value),
	)
}

func WorkflowToolInvocationFromContext(
	ctx context.Context,
) (WorkflowToolInvocation, bool) {
	if ctx == nil {
		return WorkflowToolInvocation{}, false
	}
	value, found := ctx.Value(workflowToolInvocationContextKey{}).(WorkflowToolInvocation)
	if !found {
		return WorkflowToolInvocation{}, false
	}
	return cloneWorkflowToolInvocation(value), true
}

func cloneWorkflowToolInvocation(
	value WorkflowToolInvocation,
) WorkflowToolInvocation {
	output := WorkflowToolInvocation{
		Context: append(json.RawMessage(nil), value.Context...),
		PortInputs: make(
			[]WorkflowPortInput,
			len(value.PortInputs),
		),
		OutputPorts: append([]string(nil), value.OutputPorts...),
		Emit:        value.Emit,
	}
	for index := range value.PortInputs {
		output.PortInputs[index] = WorkflowPortInput{
			Name:     value.PortInputs[index].Name,
			Channel:  value.PortInputs[index].Channel,
			Cursor:   value.PortInputs[index].Cursor,
			Streamed: value.PortInputs[index].Streamed,
			Values: make(
				[]json.RawMessage,
				len(value.PortInputs[index].Values),
			),
		}
		for valueIndex := range value.PortInputs[index].Values {
			output.PortInputs[index].Values[valueIndex] = append(
				json.RawMessage(nil),
				value.PortInputs[index].Values[valueIndex]...,
			)
		}
	}
	return output
}
