package binder

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func (b *Binder) toolChoices(
	ctx context.Context,
	req sessionv3.WorkDispatchRequest,
) ([]inferenceSpec.ToolChoice, map[string]sessionv3.ToolCall, error) {
	choices := []inferenceSpec.ToolChoice{}
	bindings := map[string]sessionv3.ToolCall{}

	register := func(value preparedChoice) error {
		if value.Choice.ID == "" || value.Choice.Name == "" {
			return errors.New("prepared Tool choice has no ID or name")
		}
		if _, duplicate := bindings[value.Choice.ID]; duplicate {
			return fmt.Errorf("duplicate provider Tool choice ID %q", value.Choice.ID)
		}
		if _, duplicate := bindings[value.Choice.Name]; duplicate {
			return fmt.Errorf("duplicate provider Tool name %q", value.Choice.Name)
		}

		choices = append(choices, value.Choice)
		bindings[value.Choice.ID] = value.Call
		bindings[value.Choice.Name] = value.Call
		return nil
	}

	if len(req.Session.Defaults.Skills) != 0 {
		skillChoices, err := b.skillLifecycleChoices()
		if err != nil {
			return nil, nil, err
		}
		for _, c := range skillChoices {
			if err := register(c); err != nil {
				return nil, nil, err
			}
		}
	}
	goChoices, err := b.goToolChoices(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	for _, c := range goChoices {
		if err := register(c); err != nil {
			return nil, nil, err
		}
	}
	mcpChoices, err := b.mcpToolChoices(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	for _, c := range mcpChoices {
		if err := register(c); err != nil {
			return nil, nil, err
		}
	}
	return choices, bindings, nil
}

type preparedChoice struct {
	Choice inferenceSpec.ToolChoice
	Call   sessionv3.ToolCall
}
