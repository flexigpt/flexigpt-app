package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	skillAggregate "github.com/flexigpt/flexigpt-app/internal/skill/aggregate"
)

type SkillAggregateWrapper struct {
	service *skillAggregate.Service
}

func withSkillAggregate[T any](
	w *SkillAggregateWrapper,
	fn func(*skillAggregate.Service) (T, error),
) (T, error) {
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.service == nil {
			return zero, basespec.ErrClosed
		}
		return fn(w.service)
	})
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

// ResolveArtifactSkill maps one durable ArtifactRef to the runtime-owned
// provider.SkillDef. The aggregate resyncs the owning Root catalog before
// returning the value.
func (w *SkillAggregateWrapper) ResolveArtifactSkill(
	ref artifact.ArtifactRef,
) (skillAggregate.ResolvedArtifactSkill, error) {
	return withSkillAggregate(
		w,
		func(service *skillAggregate.Service) (skillAggregate.ResolvedArtifactSkill, error) {
			return service.ResolveArtifactSkill(
				context.Background(),
				ref,
			)
		},
	)
}

// ResolveArtifactSkills resolves several durable ArtifactRefs in one call.
// The aggregate synchronizes each owning Root once instead of once per Skill.
func (w *SkillAggregateWrapper) ResolveArtifactSkills(
	refs []artifact.ArtifactRef,
) ([]skillAggregate.ResolvedArtifactSkill, error) {
	return withSkillAggregate(
		w,
		func(service *skillAggregate.Service) ([]skillAggregate.ResolvedArtifactSkill, error) {
			return service.ResolveArtifactSkills(
				context.Background(),
				refs,
			)
		},
	)
}

// GetArtifactSkillsPrompt renders an ArtifactRef-scoped Skill prompt.
// The caller supplies Artifact Store identities; the aggregate performs
// runtime-definition resolution and root catalog synchronization.
func (w *SkillAggregateWrapper) GetArtifactSkillsPrompt(
	filter skillAggregate.ArtifactSkillFilter,
) (string, error) {
	return withSkillAggregate(
		w,
		func(service *skillAggregate.Service) (string, error) {
			return service.GetArtifactSkillsPrompt(
				context.Background(),
				filter,
			)
		},
	)
}

// ListArtifactSkillRefs returns ArtifactRefs for runtime records matching an
// Artifact-aware filter. It is intentionally distinct from runtime.ListSkills,
// whose results use runtime-native SkillDef identities.
func (w *SkillAggregateWrapper) ListArtifactSkillRefs(
	filter skillAggregate.ArtifactSkillFilter,
) ([]artifact.ArtifactRef, error) {
	return withSkillAggregate(
		w,
		func(service *skillAggregate.Service) ([]artifact.ArtifactRef, error) {
			return service.ListArtifactSkillRefs(
				context.Background(),
				filter,
			)
		},
	)
}

// DescribeArtifactSkill returns the aggregate's compact Artifact-aware Skill
// summary. It is useful for management availability checks without requiring
// callers to parse runtime-native SkillDef values.
func (w *SkillAggregateWrapper) DescribeArtifactSkill(
	ref artifact.ArtifactRef,
) (skillAggregate.ArtifactSkillSummary, error) {
	return withSkillAggregate(
		w,
		func(service *skillAggregate.Service) (skillAggregate.ArtifactSkillSummary, error) {
			return service.DescribeArtifactSkill(
				context.Background(),
				ref,
			)
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
