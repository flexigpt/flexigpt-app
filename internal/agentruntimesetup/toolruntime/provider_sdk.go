package toolruntime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
)

const (
	providerToolConfigurationUnsupportedMessage = "provider sdk tool does not accept user arguments"
	providerToolConfigurationInvalidMessage     = "provider sdk tool user arguments are invalid"
)

func v3ChoiceForResolvedToolWithConfiguration(
	resolved toolAPI.ResolvedToolView,
	userArguments jsonutil.CanonicalValue,
) (
	inferenceSpec.ToolChoice,
	sessionv3Runtime.ToolCall,
	jsonutil.CanonicalValue,
	error,
) {
	choice, call, err := v3ChoiceForResolvedTool(resolved)
	if err != nil {
		return inferenceSpec.ToolChoice{},
			sessionv3Runtime.ToolCall{},
			nil,
			err
	}

	schema, err := providerToolUserArgumentSchema(resolved)
	if err != nil {
		return inferenceSpec.ToolChoice{},
			sessionv3Runtime.ToolCall{},
			nil,
			err
	}
	configuration, err := normalizeProviderToolConfiguration(
		resolved,
		schema,
		userArguments,
	)
	if err != nil {
		return inferenceSpec.ToolChoice{},
			sessionv3Runtime.ToolCall{},
			nil,
			err
	}
	if err := applyProviderToolConfiguration(
		&choice,
		schema,
		configuration,
	); err != nil {
		return inferenceSpec.ToolChoice{},
			sessionv3Runtime.ToolCall{},
			nil,
			err
	}
	return choice, call, configuration, nil
}

func providerToolUserArgumentSchema(
	resolved toolAPI.ResolvedToolView,
) (jsonutil.CanonicalValue, error) {
	if resolved.Tool.UserArgSchema == nil ||
		len(bytes.TrimSpace(*resolved.Tool.UserArgSchema)) == 0 {
		return nil, nil
	}
	return jsonutil.CanonicalValueFrom(*resolved.Tool.UserArgSchema)
}

func normalizeProviderToolConfiguration(
	resolved toolAPI.ResolvedToolView,
	schema jsonutil.CanonicalValue,
	value jsonutil.CanonicalValue,
) (jsonutil.CanonicalValue, error) {
	if len(value) == 0 {
		return nil, nil
	}
	if !value.Valid() {
		return nil, errors.New(providerToolConfigurationInvalidMessage)
	}
	if resolved.Tool.Implementation.Kind != toolv1.ImplementationKindSDK {
		return nil, errors.New(providerToolConfigurationUnsupportedMessage)
	}
	if len(schema) == 0 {
		return nil, errors.New(providerToolConfigurationUnsupportedMessage)
	}

	compiled, err := jsonutil.CompileJSONSchema([]byte(schema))
	if err != nil {
		return nil, fmt.Errorf(
			"%s: %w",
			providerToolConfigurationInvalidMessage,
			err,
		)
	}
	if err := jsonutil.ValidateJSONSchema(
		compiled,
		json.RawMessage(value.Clone()),
		maximumToolSchemaBytes,
	); err != nil {
		return nil, fmt.Errorf(
			"%s: %w",
			providerToolConfigurationInvalidMessage,
			err,
		)
	}
	return value.Clone(), nil
}

func applyProviderToolConfiguration(
	choice *inferenceSpec.ToolChoice,
	schema jsonutil.CanonicalValue,
	configuration jsonutil.CanonicalValue,
) error {
	if len(configuration) == 0 {
		return nil
	}
	if choice == nil || len(schema) == 0 {
		return errors.New(providerToolConfigurationUnsupportedMessage)
	}

	switch choice.Type {
	case inferenceSpec.ToolTypeWebSearch:
		value, err := jsonutil.DecodeCanonicalValue[inferenceSpec.WebSearchToolChoiceItem](configuration)
		if err != nil {
			return fmt.Errorf(
				"%s: %w",
				providerToolConfigurationInvalidMessage,
				err,
			)
		}
		choice.WebSearchArguments = &value
		return nil

	case inferenceSpec.ToolTypeFunction,
		inferenceSpec.ToolTypeCustom:
		return fmt.Errorf(
			"%s for tool type %q",
			providerToolConfigurationUnsupportedMessage,
			choice.Type,
		)

	default:
		return fmt.Errorf(
			"%s for tool type %q",
			providerToolConfigurationUnsupportedMessage,
			choice.Type,
		)
	}
}

// ProviderManagedToolChoiceToInference converts the durable provider Tool
// choice into the provider request representation. It is used only by setup.
func ProviderManagedToolChoiceToInference(
	value sessionv3Runtime.ProviderManagedToolChoice,
) (inferenceSpec.ToolChoice, error) {
	if err := value.Validate(); err != nil {
		return inferenceSpec.ToolChoice{}, err
	}

	arguments, err := jsonutil.DecodeCanonicalValue[map[string]any](
		value.InputSchema,
	)
	if err != nil {
		return inferenceSpec.ToolChoice{}, err
	}

	choice := inferenceSpec.ToolChoice{
		Type:        inferenceSpec.ToolType(value.Type),
		ID:          value.Call.ChoiceID,
		Name:        value.Call.Name,
		Description: value.Call.Description,
		Arguments:   arguments,
	}
	if err := applyProviderToolConfiguration(
		&choice,
		value.UserArgumentSchema,
		value.Configuration,
	); err != nil {
		return inferenceSpec.ToolChoice{}, err
	}
	return choice, nil
}
