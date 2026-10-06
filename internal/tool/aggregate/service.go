package aggregate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	toolConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/consumerapi"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
	toolRuntime "github.com/flexigpt/flexigpt-app/internal/tool/runtime"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

type ToolSelection struct {
	ChoiceID              string                       `json:"choiceID"`
	Target                composition.CapabilityTarget `json:"target"`
	AutoExecute           bool                         `json:"autoExecute"`
	UserArgSchemaInstance jsonutil.JSONRawString       `json:"userArgSchemaInstance,omitempty"`
}

func (s ToolSelection) Validate() error {
	if err := spec.ValidateRequiredText(
		"Tool choice ID",
		s.ChoiceID,
		spec.MaxURIBytes,
	); err != nil {
		return err
	}
	return s.Target.Validate()
}

type Service struct {
	tools   *toolConsumerAPI.API
	runtime *toolRuntime.Service
}

type InvokeRequest struct {
	Target    composition.CapabilityTarget `json:"target"`
	Args      json.RawMessage              `json:"args"`
	TimeoutMS int                          `json:"timeoutMS,omitempty"`
}

func New(
	tools *toolConsumerAPI.API,
	runtimeService *toolRuntime.Service,
) (*Service, error) {
	if tools == nil || runtimeService == nil {
		return nil, fmt.Errorf(
			"%w: Tool Aggregate dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	return &Service{
		tools:   tools,
		runtime: runtimeService,
	}, nil
}

func (s *Service) Invoke(
	ctx context.Context,
	request InvokeRequest,
) (*toolRuntime.InvokeResponse, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	resolved, err := s.resolveToolTarget(ctx, request.Target)
	if err != nil {
		return nil, err
	}
	if resolved.Tool.Implementation.Kind != toolv1.ImplementationKindGo {
		return nil, fmt.Errorf(
			"%w: SDK Tools execute through provider inference",
			spec.ErrUnsupported,
		)
	}

	return s.runtime.Invoke(ctx, toolRuntime.InvokeRequest{
		Function:  resolved.Tool.Implementation.Function,
		Args:      request.Args,
		TimeoutMS: request.TimeoutMS,
	})
}

func (s *Service) HydrateInferenceToolChoice(
	ctx context.Context,
	selection ToolSelection,
) (inferenceSpec.ToolChoice, error) {
	if err := selection.Validate(); err != nil {
		return inferenceSpec.ToolChoice{}, err
	}

	resolved, err := s.resolveToolTarget(ctx, selection.Target)
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

func (s *Service) HydrateInferenceToolChoices(
	ctx context.Context,
	selections []ToolSelection,
) ([]inferenceSpec.ToolChoice, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
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

func (s *Service) ready(ctx context.Context) error {
	if s == nil || s.tools == nil || s.runtime == nil {
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Tool Aggregate context is nil",
			spec.ErrInvalid,
		)
	}
	return ctx.Err()
}

// resolveToolTarget resolves a source-backed Tool capability target for the
// Go Tool runtime. Direct Tool capability targets remain owned by their
// application-supplied runtime adapter and are not coerced into Artifacts.
func (s *Service) resolveToolTarget(
	ctx context.Context,
	target composition.CapabilityTarget,
) (toolConsumerAPI.ResolvedToolView, error) {
	if err := s.ready(ctx); err != nil {
		return toolConsumerAPI.ResolvedToolView{}, err
	}
	if err := target.Validate(); err != nil {
		return toolConsumerAPI.ResolvedToolView{}, err
	}
	if target.Type != declaration.TypeTool {
		return toolConsumerAPI.ResolvedToolView{}, fmt.Errorf(
			"%w: capability target type is %q, expected tool",
			spec.ErrUnsupported,
			target.Type,
		)
	}
	if target.Form != composition.TargetFormArtifact ||
		target.Artifact == nil {
		return toolConsumerAPI.ResolvedToolView{}, fmt.Errorf(
			"%w: direct Tool capability targets require an application runtime adapter",
			spec.ErrUnsupported,
		)
	}

	value, err := s.tools.ResolveEnabledTool(ctx, *target.Artifact)
	if err != nil {
		return toolConsumerAPI.ResolvedToolView{}, err
	}
	if value.Tool.Artifact.Ref() != *target.Artifact ||
		value.Tool.Name != target.Name {
		return toolConsumerAPI.ResolvedToolView{}, fmt.Errorf(
			"%w: Tool capability target no longer matches its Artifact",
			spec.ErrReferenceUnresolved,
		)
	}
	return value, nil
}

func hydrateSDKToolChoice(
	choice inferenceSpec.ToolChoice,
	document toolConsumerAPI.ToolView,
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
		return map[string]any{"type": "object"}, nil
	}

	var output map[string]any
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, fmt.Errorf(
			"%w: Tool inputSchema must project to an object",
			spec.ErrInvalid,
		)
	}
	if output == nil {
		output = map[string]any{"type": "object"}
	}
	return output, nil
}
