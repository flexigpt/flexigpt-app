package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/skillcollectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/workspacecollectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	skillAggregate "github.com/flexigpt/flexigpt-app/internal/skill/aggregate"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/skill/store/workspaceadapter"
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
	bundles skillConsumerAPI.BundleReader,
	artifacts compositionapi.ArtifactAPI,
	collections compositionapi.CollectionAPI,
	catalogs compositionapi.CatalogAPI,
	resources compositionapi.ResourceAPI,
	workspaceSkills *workspaceadapter.Adapter,
	runtimeWrapper *SkillRuntimeWrapper,
) error {
	if wrapper == nil {
		return errors.New("skill aggregate wrapper is required")
	}
	if bundles == nil ||
		artifacts == nil ||
		collections == nil ||
		catalogs == nil ||
		resources == nil ||
		workspaceSkills == nil {
		return errors.New("skill aggregate wrapper dependencies are incomplete")
	}
	if runtimeWrapper == nil {
		return errors.New("skill runtime wrapper is required")
	}

	router, err := skillAggregate.NewArtifactRouter(
		artifacts,
		collections,
	)
	if err != nil {
		return fmt.Errorf("initialize Skill artifact router: %w", err)
	}

	workspaceLoader, err := workspaceadapter.NewStoreLoader(workspaceSkills)
	if err != nil {
		return fmt.Errorf("initialize Workspace Skill loader: %w", err)
	}

	bundleResolver, err := skillAggregate.NewBundleResolver(
		bundles,
		artifacts,
		catalogs,
		resources,
	)
	if err != nil {
		return fmt.Errorf("initialize Skill Bundle runtime resolver: %w", err)
	}

	bundleLoader, err := skillAggregate.NewBundleLoader(bundleResolver)
	if err != nil {
		return fmt.Errorf("initialize Skill Bundle runtime loader: %w", err)
	}

	if err := router.Register(
		workspacecollectionv1.WorkspaceCollectionKind,
		workspaceLoader,
	); err != nil {
		return fmt.Errorf("register Workspace Skill loader: %w", err)
	}
	if err := router.Register(
		skillcollectionv1.SkillCollectionKind,
		bundleLoader,
	); err != nil {
		return fmt.Errorf("register Skill Bundle runtime loader: %w", err)
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

// ResolveArtifactSkill is the durable ArtifactRef to runtime SkillDef bridge.
//
// It belongs here instead of SkillStoreWrapper because resolution also makes
// the owning runtime catalog current and verifies that the resolved runtime
// definition is registered.
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
