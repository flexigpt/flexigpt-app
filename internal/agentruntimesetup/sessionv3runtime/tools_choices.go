package sessionv3runtime

import (
	"context"
	"errors"
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/toolruntime"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func (b *Binder) toolChoices(
	ctx context.Context,
	req sessionv3Runtime.WorkDispatchRequest,
	defaults sessionv3Runtime.SessionDefaults,
	skillChoices []toolruntime.V3PreparedChoice,
) (
	[]inferenceSpec.ToolChoice,
	map[string]sessionv3Runtime.ToolCall,
	[]preparedChoice,
	error,
) {
	choices := []inferenceSpec.ToolChoice{}
	bindings := map[string]sessionv3Runtime.ToolCall{}
	prepared := []preparedChoice{}

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
		prepared = append(prepared, value)
		return nil
	}

	if len(defaults.Skills) != 0 {
		if len(skillChoices) == 0 {
			return nil, nil, nil, errors.New(
				"enabled Skills produced no lifecycle Tool choices",
			)
		}
		for _, c := range preparedChoicesFromDomain(skillChoices) {
			if err := register(c); err != nil {
				return nil, nil, nil, err
			}
		}
	}
	if payload := req.WorkItem.Input.LLM; payload != nil {
		for _, providerTool := range payload.ProviderTools {
			choice, err := toolruntime.ProviderManagedToolChoiceToInference(
				providerTool,
			)
			if err != nil {
				return nil, nil, nil, err
			}
			if err := register(preparedChoice{
				Choice:                choice,
				Call:                  providerTool.Call,
				ProviderConfiguration: providerTool.Configuration.Clone(),
			}); err != nil {
				return nil, nil, nil, err
			}
		}
	}
	if b.goTools == nil {
		return nil, nil, nil, errors.New(
			"session v3 Tool runtime is unavailable",
		)
	}
	goChoices, err := b.goTools.Choices(
		ctx,
		sessionv3Runtime.EffectiveToolSelections(defaults),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, c := range preparedChoicesFromDomain(goChoices) {
		if err := register(c); err != nil {
			return nil, nil, nil, err
		}
	}
	if b.mcpTools == nil {
		return nil, nil, nil, errors.New(
			"session v3 MCP runtime is unavailable",
		)
	}
	mcpChoices, err := b.mcpTools.Choices(ctx, defaults.MCP)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, c := range preparedChoicesFromDomain(mcpChoices) {
		if err := register(c); err != nil {
			return nil, nil, nil, err
		}
	}
	return choices, bindings, prepared, nil
}

type preparedChoice struct {
	Choice                inferenceSpec.ToolChoice
	Call                  sessionv3Runtime.ToolCall
	ProviderConfiguration jsonutil.CanonicalValue
}

func preparedChoicesFromDomain(
	values []toolruntime.V3PreparedChoice,
) []preparedChoice {
	output := make([]preparedChoice, len(values))
	for index := range values {
		output[index] = preparedChoice(values[index])
		output[index].ProviderConfiguration = values[index].ProviderConfiguration.Clone()
	}
	return output
}
