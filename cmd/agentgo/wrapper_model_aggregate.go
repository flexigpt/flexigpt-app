package main

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	modelAggregate "github.com/flexigpt/flexigpt-app/internal/model/aggregate"
)

type ModelAggregateWrapper struct {
	service *modelAggregate.Service
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
