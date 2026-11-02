package binder

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func (b *Binder) toolChoices(
	ctx context.Context,
	req sessionv3.WorkDispatchRequest,
) ([]inferenceSpec.ToolChoice, map[string]sessionv3.ToolCall, error) {
	choices := []inferenceSpec.ToolChoice{}
	bindings := map[string]sessionv3.ToolCall{}
	if len(req.Session.Defaults.Skills) != 0 {
		skillChoices, err := b.skillLifecycleChoices()
		if err != nil {
			return nil, nil, err
		}
		for _, c := range skillChoices {
			choices = append(choices, c.Choice)
			bindings[c.Choice.ID] = c.Call
			bindings[c.Choice.Name] = c.Call
		}
	}
	goChoices, err := b.goToolChoices(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	for _, c := range goChoices {
		choices = append(choices, c.Choice)
		bindings[c.Choice.ID] = c.Call
		bindings[c.Choice.Name] = c.Call
	}
	mcpChoices, err := b.mcpToolChoices(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	for _, c := range mcpChoices {
		choices = append(choices, c.Choice)
		bindings[c.Choice.ID] = c.Call
		bindings[c.Choice.Name] = c.Call
	}
	return choices, bindings, nil
}

type preparedChoice struct {
	Choice inferenceSpec.ToolChoice
	Call   sessionv3.ToolCall
}
