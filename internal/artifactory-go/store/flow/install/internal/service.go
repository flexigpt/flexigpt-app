package internal

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	storeSecret "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type Dependencies struct {
	Roots      root.API
	RootSystem root.System

	Sources       source.API
	SourceRuntime source.Runtime
	SourceContent source.ContentMutation

	ManagedSources managedpackage.Runtime

	Artifacts artifact.API

	Refresh         refreshFlow.API
	RefreshCompiled refreshFlow.CompiledDocumentRegistrar

	Repository Repository

	SecretLifecycle storeSecret.LifecycleAPI
	Policy          root.Policy
}

type Service struct {
	Roots      root.API
	rootSystem root.System

	Sources       source.API
	SourceRuntime source.Runtime
	sourceContent source.ContentMutation

	managedSources managedpackage.Runtime

	Artifacts artifact.API

	Refresh         refreshFlow.API
	refreshCompiled refreshFlow.CompiledDocumentRegistrar

	metadata Repository

	localState storeSecret.LifecycleAPI

	rootMutationPolicy root.Policy
}

func NewService(
	dependencies Dependencies,
) (*Service, error) {
	if dependencies.Roots == nil ||
		dependencies.RootSystem == nil ||
		dependencies.Sources == nil ||
		dependencies.SourceRuntime == nil ||
		dependencies.SourceContent == nil ||
		dependencies.ManagedSources == nil ||
		dependencies.Artifacts == nil ||
		dependencies.Refresh == nil ||
		dependencies.RefreshCompiled == nil ||
		dependencies.Repository == nil ||
		dependencies.SecretLifecycle == nil {
		return nil, fmt.Errorf(
			"%w: installation flow dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	return &Service{
		Roots:              dependencies.Roots,
		rootSystem:         dependencies.RootSystem,
		Sources:            dependencies.Sources,
		SourceRuntime:      dependencies.SourceRuntime,
		sourceContent:      dependencies.SourceContent,
		managedSources:     dependencies.ManagedSources,
		Artifacts:          dependencies.Artifacts,
		Refresh:            dependencies.Refresh,
		refreshCompiled:    dependencies.RefreshCompiled,
		metadata:           dependencies.Repository,
		localState:         dependencies.SecretLifecycle,
		rootMutationPolicy: dependencies.Policy,
	}, nil
}

func (c *Service) isProtectedRoot(
	rootID rootModel.RootID,
) bool {
	return c != nil &&
		c.rootMutationPolicy != nil &&
		c.rootMutationPolicy.IsProtectedRoot(rootID)
}
