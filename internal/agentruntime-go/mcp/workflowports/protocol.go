package workflowports

import (
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	ExtensionID = "io.flexigpt.workflow.ports/v1"
	Version     = "v1"

	CapabilityVersionKey  = "version"
	CapabilityRequestKey  = "request"
	CapabilityResponseKey = "response"
)

type PortInput struct {
	Name     string            `json:"name"`
	Channel  string            `json:"channel"`
	Cursor   uint64            `json:"cursor,omitempty"`
	Values   []json.RawMessage `json:"values,omitempty"`
	Streamed bool              `json:"streamed,omitempty"`
}

type Request struct {
	Version     string          `json:"version"`
	Context     json.RawMessage `json:"context,omitempty"`
	PortInputs  []PortInput     `json:"portInputs,omitempty"`
	OutputPorts []string        `json:"outputPorts,omitempty"`
}

func RequestMetaValue(value Request) (any, error) {
	value.Version = Version
	raw, err := jsonutil.MarshalCanonicalObject(
		value,
		2<<20,
	)
	if err != nil {
		return nil, err
	}

	var output any
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, err
	}
	return output, nil
}

func RequestFromMeta(value any) (*Request, error) {
	if value == nil {
		return nil, nil
	}
	raw, err := jsonutil.EncodeToJSONRaw(value)
	if err != nil {
		return nil, err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return nil, err
	}

	var output Request
	if err := json.Unmarshal(canonical, &output); err != nil {
		return nil, err
	}
	if output.Version != Version {
		return nil, nil
	}
	return &output, nil
}

func CloneRequest(value *Request) *Request {
	if value == nil {
		return nil
	}

	output := &Request{
		Version:     value.Version,
		Context:     append(json.RawMessage(nil), value.Context...),
		OutputPorts: append([]string(nil), value.OutputPorts...),
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
