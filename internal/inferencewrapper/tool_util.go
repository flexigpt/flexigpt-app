package inferencewrapper

import (
	"context"
	"errors"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	toolAggregate "github.com/flexigpt/flexigpt-app/internal/tool/aggregate"
)

func buildToolChoices(
	ctx context.Context,
	tools *toolAggregate.Service,
	selections []toolAggregate.ToolSelection,
) ([]inferenceSpec.ToolChoice, error) {
	if len(selections) == 0 {
		return nil, nil
	}
	if tools == nil {
		return nil, errors.New(
			"tool aggregate is not configured for provider set",
		)
	}
	return tools.HydrateInferenceToolChoices(ctx, selections)
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
