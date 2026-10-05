package main

import (
	"errors"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

func artifactTargetMappers(
	tools *ToolAggregateWrapper,
	models *ModelAggregateWrapper,
) (map[declaration.Type]composition.ArtifactTargetMapper, error) {
	if tools == nil || models == nil {
		return nil, errors.New(
			"artifact target mapper aggregates are incomplete",
		)
	}

	toolMappers, err := tools.targetMappers()
	if err != nil {
		return nil, err
	}
	modelMappers, err := models.targetMappers()
	if err != nil {
		return nil, err
	}

	maps.Copy(toolMappers, modelMappers)
	return toolMappers, nil
}
