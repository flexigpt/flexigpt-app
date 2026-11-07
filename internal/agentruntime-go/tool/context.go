package tool

import (
	"context"
	"encoding/json"
)

const Version = "v1"

type PortInput struct {
	Name     string            `json:"name"`
	Channel  string            `json:"channel"`
	Cursor   uint64            `json:"cursor,omitempty"`
	Values   []json.RawMessage `json:"values,omitempty"`
	Streamed bool              `json:"streamed,omitempty"`
}

type PortEmitter func(
	name string,
	value json.RawMessage,
) error

type Invocation struct {
	Version     string          `json:"version"`
	Context     json.RawMessage `json:"context,omitempty"`
	PortInputs  []PortInput     `json:"portInputs,omitempty"`
	OutputPorts []string        `json:"outputPorts,omitempty"`
	Emit        PortEmitter     `json:"-"`
}

func (i Invocation) RequiresTransport() bool {
	return len(i.Context) != 0 ||
		len(i.PortInputs) != 0 ||
		len(i.OutputPorts) != 0
}

type invocationContextKey struct{}

func WithInvocation(
	ctx context.Context,
	value Invocation,
) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(
		ctx,
		invocationContextKey{},
		Clone(value),
	)
}

func FromContext(
	ctx context.Context,
) (Invocation, bool) {
	if ctx == nil {
		return Invocation{}, false
	}
	value, found := ctx.Value(invocationContextKey{}).(Invocation)
	if !found {
		return Invocation{}, false
	}
	return Clone(value), true
}

func Clone(value Invocation) Invocation {
	output := Invocation{
		Version:     value.Version,
		Context:     append(json.RawMessage(nil), value.Context...),
		OutputPorts: append([]string(nil), value.OutputPorts...),
		Emit:        value.Emit,
		PortInputs:  make([]PortInput, len(value.PortInputs)),
	}
	for index := range value.PortInputs {
		output.PortInputs[index] = PortInput{
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
