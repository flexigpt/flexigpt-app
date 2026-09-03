package main

import (
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	skillAggregate "github.com/flexigpt/flexigpt-app/internal/skill/aggregate"

	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
)

// SkillAggregateWrapper owns the application-level composition between the
// Skill store, runtime catalog source, runtime service, and aggregate service.
//
// Only RuntimeCatalogIDForCollection is exported as a Wails method.
type SkillAggregateWrapper struct {
	service *skillAggregate.Service
}

func InitSkillAggregateWrapper(
	wrapper *SkillAggregateWrapper,
	router *skillStore.ArtifactRouter,
	runtimeWrapper *SkillRuntimeWrapper,
) error {
	if wrapper == nil {
		return errors.New("skill aggregate wrapper is required")
	}
	if router == nil {
		return errors.New("skill artifact router is required")
	}
	if runtimeWrapper == nil {
		return errors.New("skill runtime wrapper is required")
	}

	catalogSource, err := skillAggregate.NewCatalogSource(router)
	if err != nil {
		return fmt.Errorf("initialize Skill catalog source: %w", err)
	}

	if err := InitSkillRuntimeWrapper(
		runtimeWrapper,
		catalogSource,
	); err != nil {
		return fmt.Errorf("initialize Skill runtime: %w", err)
	}

	service, err := skillAggregate.New(
		router,
		runtimeWrapper.service,
	)
	if err != nil {
		runtimeWrapper.close()
		return fmt.Errorf("initialize Skill aggregate: %w", err)
	}

	wrapper.service = service
	return nil
}

// RuntimeCatalogIDForCollection maps durable Collection identity to the
// runtime-owned opaque catalog identity. It does not read Skill content.
func (w *SkillAggregateWrapper) RuntimeCatalogIDForCollection(
	ref collection.CollectionRef,
) (skillRuntime.CatalogID, error) {
	return middleware.WithRecoveryResp(
		func() (skillRuntime.CatalogID, error) {
			return skillAggregate.CollectionCatalogID(ref)
		},
	)
}

func (w *SkillAggregateWrapper) close() {
	if w == nil {
		return
	}

	service := w.service
	w.service = nil

	if service != nil {
		service.Close()
	}
}
