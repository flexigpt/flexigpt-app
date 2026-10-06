package inferencewrapper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	conversationSpec "github.com/flexigpt/flexigpt-app/internal/conversation/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func buildToolChoices(
	ctx context.Context,
	toolsSvc *toolAPI.Service,
	selections []conversationSpec.ToolSelection,
) ([]inferenceSpec.ToolChoice, error) {
	if len(selections) == 0 {
		return nil, nil
	}
	if toolsSvc == nil {
		return nil, errors.New(
			"tool aggregate is not configured for provider set",
		)
	}
	return hydrateInferenceToolChoices(ctx, toolsSvc, selections)
}

func hydrateInferenceToolChoices(
	ctx context.Context,
	toolsSvc *toolAPI.Service,
	selections []conversationSpec.ToolSelection,
) ([]inferenceSpec.ToolChoice, error) {
	output := make([]inferenceSpec.ToolChoice, 0, len(selections))
	seen := make(map[string]struct{}, len(selections))

	for index, selection := range selections {
		if _, duplicate := seen[selection.ChoiceID]; duplicate {
			return nil, fmt.Errorf(
				"%w: Tool choice %q is repeated",
				spec.ErrIdentityConflict,
				selection.ChoiceID,
			)
		}
		seen[selection.ChoiceID] = struct{}{}

		choice, err := hydrateInferenceToolChoice(
			ctx,
			toolsSvc,
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

// hydrateInferenceToolChoice converts either supported source-backed Tool
// implementation into an inference ToolChoice.
//
// Go Tools produce ordinary function choices. SDK Tools produce provider-native which are not invoked.
func hydrateInferenceToolChoice(
	ctx context.Context,
	toolsSvc *toolAPI.Service,
	selection conversationSpec.ToolSelection,
) (inferenceSpec.ToolChoice, error) {
	if err := selection.Validate(); err != nil {
		return inferenceSpec.ToolChoice{}, err
	}

	resolved, err := resolveToolTarget(ctx, toolsSvc, selection.Target)
	if err != nil {
		return inferenceSpec.ToolChoice{}, err
	}

	document := resolved.Tool
	arguments, err := toolArguments(document.InputSchema)
	if err != nil {
		return inferenceSpec.ToolChoice{}, err
	}

	choice := inferenceSpec.ToolChoice{
		ID:          selection.ChoiceID,
		Name:        string(document.Name),
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
			spec.ErrUnsupported,
			document.Implementation.Kind,
		)
	}
}

// resolveToolTarget resolves either Go or SDK source-backed Tool capability
// target. Direct Tool capability targets remain owned by their
// application-supplied runtime adapter and are not coerced into Artifacts.
func resolveToolTarget(
	ctx context.Context,
	toolsSvc *toolAPI.Service,
	target composition.CapabilityTarget,
) (toolAPI.ResolvedToolView, error) {
	if err := target.Validate(); err != nil {
		return toolAPI.ResolvedToolView{}, err
	}
	if target.Type != declaration.TypeTool {
		return toolAPI.ResolvedToolView{}, fmt.Errorf(
			"%w: capability target type is %q, expected tool",
			spec.ErrUnsupported,
			target.Type,
		)
	}
	if target.Form != composition.TargetFormArtifact ||
		target.Artifact == nil {
		return toolAPI.ResolvedToolView{}, fmt.Errorf(
			"%w: direct Tool capability targets require an application runtime adapter",
			spec.ErrUnsupported,
		)
	}

	value, err := toolsSvc.ResolveEnabledTool(ctx, *target.Artifact)
	if err != nil {
		return toolAPI.ResolvedToolView{}, err
	}
	if value.Tool.Artifact.Ref() != *target.Artifact ||
		value.Tool.Name != target.Name {
		return toolAPI.ResolvedToolView{}, fmt.Errorf(
			"%w: Tool capability target no longer matches its Artifact",
			spec.ErrReferenceUnresolved,
		)
	}
	return value, nil
}

func hydrateSDKToolChoice(
	choice inferenceSpec.ToolChoice,
	document toolAPI.ToolView,
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
			spec.ErrUnsupported,
			document.Implementation.SDKToolType,
		)
	}
}

func toolArguments(
	raw json.RawMessage,
) (map[string]any, error) {
	raw = bytes.TrimSpace(raw)
	if bytes.Equal(raw, []byte("false")) {
		return nil, fmt.Errorf(
			"%w: false Tool inputSchema cannot be advertised as inference arguments",
			spec.ErrUnsupported,
		)
	}
	if len(raw) == 0 ||
		bytes.Equal(raw, []byte("true")) {
		return getEmptySchema(), nil
	}

	var output map[string]any
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, fmt.Errorf(
			"%w: Tool inputSchema must project to an object",
			spec.ErrInvalid,
		)
	}
	if output == nil {
		output = getEmptySchema()
	}
	return output, nil
}

// decodeToolArgSchema remains shared by Skill Tool choice construction.
// Tool Store itself uses aggregate.toolArguments for Tool Artifact schemas.
func decodeToolArgSchema(
	raw jsonutil.JSONRawString,
) (map[string]any, error) {
	if len(raw) == 0 {
		return getEmptySchema(), nil
	}
	schema, err := jsonutil.DecodeJSONStringRawInto[map[string]any](raw)
	if err != nil {
		return nil, err
	}
	if len(schema) == 0 {
		return getEmptySchema(), nil
	}
	return schema, nil
}
