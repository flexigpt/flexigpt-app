package main

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/skillcatalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/skillruntime"
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
) (*skillruntime.RuntimeAdapter, error) {
	if aggregateWrapper == nil || runtimeWrapper == nil {
		return nil, fmt.Errorf(
			"%w: Skill wrappers are required",
			spec.ErrInvalid,
		)
	}
	if runtimeWrapper.service != nil {
		return nil, fmt.Errorf(
			"%w: Skill runtime wrapper is already initialized",
			spec.ErrConflict,
		)
	}

	adapter, err := skillruntime.NewRuntimeAdapter(
		artifacts,
		cat,
		resources,
		nativeResources,
	)
	if err != nil {
		return nil, err
	}
	if err := InitSkillRuntimeWrapper(runtimeWrapper, adapter); err != nil {
		return nil, fmt.Errorf("initialize Skill runtime: %w", err)
	}
	if err := adapter.BindRuntime(runtimeWrapper.service); err != nil {
		runtimeWrapper.close()
		return nil, err
	}
	if err := InitSkillAggregateWrapper(
		aggregateWrapper,
		adapter,
		runtimeWrapper.service,
	); err != nil {
		runtimeWrapper.close()
		return nil, err
	}
	return adapter, nil
}
