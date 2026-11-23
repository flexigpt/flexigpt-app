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
	plan llmExecutionPlan,
	skillChoices []toolruntime.V3PreparedChoice,
) (
	[]inferenceSpec.ToolChoice,
	map[string]sessionv3Runtime.ToolCall,
	[]preparedChoice,
	error,
) {
	defaults := plan.defaults
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
	for _, selection := range defaults.MCP {
		if preparedMCP, found := plan.preparedMCP[selectionKey(selection.Server)]; found &&
			preparedMCP.Prepared {
			preparedMCPChoices, decodeErr := preparedChoicesFromDispatchPlans(
				preparedMCP.PreparedTools,
			)
			if decodeErr != nil {
				return nil, nil, nil, decodeErr
			}
			for _, c := range preparedMCPChoices {
				if err := register(c); err != nil {
					return nil, nil, nil, err
				}
			}
			continue
		}
		if b.mcpTools == nil {
			return nil, nil, nil, errors.New(
				"session v3 mcp runtime is unavailable",
			)
		}
		mcpChoices, err := b.mcpTools.Choices(
			ctx,
			[]sessionv3Runtime.MCPServerSelection{selection},
		)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, c := range preparedChoicesFromDomain(mcpChoices) {
			if err := register(c); err != nil {
				return nil, nil, nil, err
			}
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

func preparedChoicesFromDispatchPlans(
	values []sessionv3Runtime.ToolDispatchPlan,
) ([]preparedChoice, error) {
	output := make([]preparedChoice, 0, len(values))
	for _, value := range values {
		if len(value.Choice) == 0 {
			return nil, errors.New(
				"prepared tool dispatch plan has no provider choice",
			)
		}
		choice, err := jsonutil.DecodeCanonicalValue[inferenceSpec.ToolChoice](
			value.Choice,
		)
		if err != nil {
			return nil, err
		}
		if choice.ID == "" ||
			choice.Name == "" ||
			choice.ID != value.Call.ChoiceID ||
			choice.Name != value.Call.Name {
			return nil, errors.New(
				"prepared provider choice differs from its durable tool call",
			)
		}
		output = append(output, preparedChoice{
			Choice:                choice,
			Call:                  jsonutil.CloneJSONInValue(value.Call),
			ProviderConfiguration: value.ProviderConfiguration.Clone(),
		})
	}
	return output, nil
}
