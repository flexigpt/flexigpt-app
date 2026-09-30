package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	modelAggregate "github.com/flexigpt/flexigpt-app/internal/model/aggregate"
)

type ModelAggregateWrapper struct {
	service *modelAggregate.Service
}

func (w *ModelAggregateWrapper) GetDefaultModelProvider() (
	*artifact.ArtifactRef,
	error,
) {
	if w == nil || w.service == nil {
		return nil, basespec.ErrClosed
	}
	return w.service.GetDefaultModelProvider(context.Background())
}

func (w *ModelAggregateWrapper) SetDefaultModelProvider(
	provider *artifact.ArtifactRef,
) error {
	if w == nil || w.service == nil {
		return basespec.ErrClosed
	}
	return w.service.SetDefaultModelProvider(context.Background(), provider)
}

func (w *ModelAggregateWrapper) GetModelProviderDefaultModel(
	provider artifact.ArtifactRef,
) (artifact.ArtifactRef, error) {
	if w == nil || w.service == nil {
		return artifact.ArtifactRef{}, basespec.ErrClosed
	}
	return w.service.GetModelProviderDefaultModel(context.Background(), provider)
}

func (w *ModelAggregateWrapper) targetMappers() (
	map[declaration.Type]resolve.ArtifactTargetMapper,
	error,
) {
	if w == nil || w.service == nil {
		return nil, basespec.ErrClosed
	}
	return map[declaration.Type]resolve.ArtifactTargetMapper{
		declaration.TypeModel: w.service,
	}, nil
}

func (w *ModelAggregateWrapper) close() {
	if w == nil {
		return
	}
	w.service = nil
}
