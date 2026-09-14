package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	skillAggregate "github.com/flexigpt/flexigpt-app/internal/skill/aggregate"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
)

type SkillAggregateWrapper struct {
	service *skillAggregate.Service
}

func InitSkillAggregateWrapper(
	wrapper *SkillAggregateWrapper,
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
	runtimeWrapper *SkillRuntimeWrapper,
) error {
	if wrapper == nil ||
		artifacts == nil ||
		resources == nil ||
		runtimeWrapper == nil {
		return errors.New("skill aggregate wrapper dependencies are incomplete")
	}

	router, err := skillAggregate.NewArtifactRouter(
		artifacts,
		resources,
	)
	if err != nil {
		return fmt.Errorf("initialize Skill Artifact router: %w", err)
	}
	catalogSource, err := skillAggregate.NewCatalogSource(router)
	if err != nil {
		return fmt.Errorf("initialize Skill Root catalog source: %w", err)
	}
	if err := InitSkillRuntimeWrapper(runtimeWrapper, catalogSource); err != nil {
		return fmt.Errorf("initialize Skill runtime: %w", err)
	}
	service, err := skillAggregate.New(router, runtimeWrapper.service)
	if err != nil {
		runtimeWrapper.close()
		return fmt.Errorf("initialize Skill aggregate: %w", err)
	}
	wrapper.service = service
	return nil
}

func (w *SkillAggregateWrapper) RuntimeCatalogIDForRoot(
	rootID root.RootID,
) (skillRuntime.CatalogID, error) {
	return middleware.WithRecoveryResp(
		func() (skillRuntime.CatalogID, error) {
			return skillAggregate.RootCatalogID(rootID)
		},
	)
}

func (w *SkillAggregateWrapper) ResolveArtifactSkill(
	ref artifact.ArtifactRef,
) (skillAggregate.ResolvedArtifactSkill, error) {
	return middleware.WithRecoveryResp(
		func() (skillAggregate.ResolvedArtifactSkill, error) {
			if w == nil || w.service == nil {
				return skillAggregate.ResolvedArtifactSkill{}, basespec.ErrClosed
			}
			return w.service.ResolveArtifactSkill(context.Background(), ref)
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
