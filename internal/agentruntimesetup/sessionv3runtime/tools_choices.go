package sessionv3runtime

import (
	"context"
	"errors"
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/toolruntime"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func (b *Binder) toolChoices(
	ctx context.Context,
	req sessionv3Runtime.WorkDispatchRequest,
) ([]inferenceSpec.ToolChoice, map[string]sessionv3Runtime.ToolCall, error) {
	choices := []inferenceSpec.ToolChoice{}
	bindings := map[string]sessionv3Runtime.ToolCall{}

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
		var (
			skillChoices []preparedChoice
			err          error
		)
		if b.skillTools != nil {
			skillChoices, err = b.skillLifecycleChoicesFromDomain(ctx, req)
		} else {
			skillChoices, err = b.skillLifecycleChoices()
		}
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
	Call   sessionv3Runtime.ToolCall
}

func (b *Binder) skillLifecycleChoicesFromDomain(
	ctx context.Context,
	req sessionv3Runtime.WorkDispatchRequest,
) ([]preparedChoice, error) {
	values, err := b.skillTools.Choices(ctx, req)
	if err != nil {
		return nil, err
	}
	return preparedChoicesFromDomain(values), nil
}

func preparedChoicesFromDomain(
	values []toolruntime.V3PreparedChoice,
) []preparedChoice {
	output := make([]preparedChoice, len(values))
	for index := range values {
		output[index] = preparedChoice(values[index])
	}
	return output
}
