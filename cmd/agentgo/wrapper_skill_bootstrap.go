package main

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/skillruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/skillcatalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/llmsupport"
)

func NewSkillBuiltInInstaller(
	hydrator installModel.CompiledHydrationCoordinator,
) (installFlow.HydrationInstaller, error) {
	return skillcatalog.NewInstaller(skillcatalog.InstallerDependencies{
		Hydrator: hydrator,
	})
}

// initSkillRuntimeWrappers assembles the shared runtime and adapter before
// requests are served. Construction performs no catalog reads, so protected
// package hydration can still happen later in the existing bootstrap order.
func initSkillRuntimeWrappers(
	aggregateWrapper *SkillAggregateWrapper,
	artifacts artifact.API,
	cat catalog.API,
	resources resourceFlow.API,
	nativeResources resourceFlow.NativePathAPI,
	runtimeWrapper *SkillRuntimeWrapper,
) (*skillruntime.Resolver, error) {
	if aggregateWrapper == nil || runtimeWrapper == nil {
		return nil, fmt.Errorf(
			"%w: Skill wrappers are required",
			storeSpec.ErrInvalid,
		)
	}
	if runtimeWrapper.service != nil {
		return nil, fmt.Errorf(
			"%w: Skill runtime wrapper is already initialized",
			storeSpec.ErrConflict,
		)
	}
	support, err := llmsupport.Skill()
	if err != nil {
		return nil, err
	}

	catalogAdapter, err := skillruntime.NewCatalogAdapter(
		artifacts,
		cat,
		resources,
		nativeResources,
		support.Documents,
	)
	if err != nil {
		return nil, err
	}
	if err := InitSkillRuntimeWrapper(runtimeWrapper, catalogAdapter); err != nil {
		return nil, fmt.Errorf("initialize Skill runtime: %w", err)
	}

	resolver, err := skillruntime.NewResolver(
		catalogAdapter,
		runtimeWrapper.service,
	)
	if err != nil {
		runtimeWrapper.close()
		return nil, err
	}

	if err := InitSkillAggregateWrapper(
		aggregateWrapper,
		resolver,
		runtimeWrapper.service,
	); err != nil {
		runtimeWrapper.close()
		return nil, err
	}
	return resolver, nil
}
