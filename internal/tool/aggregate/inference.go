package aggregate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type ToolSelection struct {
	ChoiceID              string                 `json:"choiceID"`
	Target                resolve.MappedTarget   `json:"target"`
	AutoExecute           bool                   `json:"autoExecute"`
	UserArgSchemaInstance jsonutil.JSONRawString `json:"userArgSchemaInstance,omitempty"`
}

func (s ToolSelection) Validate() error {
	if err := basespec.ValidateRequiredText(
		"Tool choice ID",
		s.ChoiceID,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	return s.Target.Validate()
}

func (s *Service) HydrateInferenceToolChoice(
	ctx context.Context,
	selection ToolSelection,
) (inferenceSpec.ToolChoice, error) {
	if err := selection.Validate(); err != nil {
		return inferenceSpec.ToolChoice{}, err
	}

	resolved, err := s.ResolveMappedTool(ctx, selection.Target)
	if err != nil {
		return inferenceSpec.ToolChoice{}, err
	}

	document := resolved.Tool.Document
	arguments, err := toolArguments(document.InputSchema)
	if err != nil {
		return inferenceSpec.ToolChoice{}, err
	}

	choice := inferenceSpec.ToolChoice{
		ID:          selection.ChoiceID,
		Name:        document.Name,
		Description: document.Description,
	}

	switch document.Implementation.Kind {
	case toolv1.ImplementationKindGo:
		choice.Type = inferenceSpec.ToolType("function")
		choice.Arguments = arguments
		return choice, nil

	case toolv1.ImplementationKindSDK:
		return hydrateSDKToolChoice(
			choice,
			document,
			arguments,
			selection.UserArgSchemaInstance,
		)

	default:
		return inferenceSpec.ToolChoice{}, fmt.Errorf(
			"%w: Tool implementation %q is unsupported",
			basespec.ErrUnsupported,
			document.Implementation.Kind,
		)
	}
}

func (s *Service) HydrateInferenceToolChoices(
	ctx context.Context,
	selections []ToolSelection,
) ([]inferenceSpec.ToolChoice, error) {
	output := make([]inferenceSpec.ToolChoice, 0, len(selections))
	seen := make(map[string]struct{}, len(selections))

	for index, selection := range selections {
		if _, duplicate := seen[selection.ChoiceID]; duplicate {
			return nil, fmt.Errorf(
				"%w: Tool choice %q is repeated",
				basespec.ErrIdentityConflict,
				selection.ChoiceID,
			)
		}
		seen[selection.ChoiceID] = struct{}{}

		choice, err := s.HydrateInferenceToolChoice(
			ctx,
			selection,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"hydrate Tool choice %d: %w",
				index,
				err,
			)
		}
		output = append(output, choice)
	}
	return output, nil
}

func hydrateSDKToolChoice(
	choice inferenceSpec.ToolChoice,
	document toolv1.ToolDocument,
	arguments map[string]any,
	rawUserArgs jsonutil.JSONRawString,
) (inferenceSpec.ToolChoice, error) {
	choice.Type = inferenceSpec.ToolType(
		document.Implementation.SDKToolType,
	)

	switch document.Implementation.SDKToolType {
	case toolv1.SDKToolTypeWebSearch:
		var config inferenceSpec.WebSearchToolChoiceItem
		if raw := strings.TrimSpace(string(rawUserArgs)); raw != "" {
			decoded, err := jsonutil.DecodeJSONStringRawInto[inferenceSpec.WebSearchToolChoiceItem](rawUserArgs)
			if err != nil {
				return inferenceSpec.ToolChoice{}, fmt.Errorf(
					"decode SDK webSearch Tool configuration: %w",
					err,
				)
			}
			config = decoded
		}
		choice.Type = inferenceSpec.ToolTypeWebSearch
		choice.WebSearchArguments = &config
		return choice, nil

	case toolv1.SDKToolTypeFunction,
		toolv1.SDKToolTypeCustom:
		if raw := strings.TrimSpace(string(rawUserArgs)); raw != "" {
			if _, err := jsonutil.DecodeJSONStringRawInto[map[string]any](rawUserArgs); err != nil {
				return inferenceSpec.ToolChoice{}, fmt.Errorf(
					"decode SDK Tool configuration: %w",
					err,
				)
			}
		}
		choice.Arguments = arguments
		return choice, nil

	default:
		return inferenceSpec.ToolChoice{}, fmt.Errorf(
			"%w: SDK Tool type %q is unsupported",
			basespec.ErrUnsupported,
			document.Implementation.SDKToolType,
		)
	}
}

func toolArguments(
	raw json.RawMessage,
) (map[string]any, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 ||
		bytes.Equal(raw, []byte("true")) ||
		bytes.Equal(raw, []byte("false")) {
		return map[string]any{"type": "object"}, nil
	}

	var output map[string]any
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, fmt.Errorf(
			"%w: Tool inputSchema must project to an object",
			basespec.ErrInvalid,
		)
	}
	if output == nil {
		output = map[string]any{"type": "object"}
	}
	return output, nil
}
