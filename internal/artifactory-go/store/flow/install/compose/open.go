package compose

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/internal"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type Config struct {
	Roots      root.API
	RootSystem root.System

	Sources       source.API
	SourceRuntime source.Runtime
	SourceContent source.ContentMutation

	ManagedSources managedpackage.Runtime

	Artifacts artifact.API

	Refresh         refreshFlow.API
	RefreshCompiled refreshFlow.CompiledDocumentRegistrar

	Repository installFlow.Repository

	SecretLifecycle secret.LifecycleAPI
	Policy          rootModel.RootPolicy
}

func Open(
	config Config,
) (installFlow.API, error) {
	if config.Roots == nil ||
		config.RootSystem == nil ||
		config.Sources == nil ||
		config.SourceRuntime == nil ||
		config.SourceContent == nil ||
		config.ManagedSources == nil ||
		config.Artifacts == nil ||
		config.Refresh == nil ||
		config.RefreshCompiled == nil ||
		config.Repository == nil ||
		config.SecretLifecycle == nil {
		return nil, fmt.Errorf(
			"%w: installation composition dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	service, err := internal.NewService(
		internal.Dependencies{
			Roots:           config.Roots,
			RootSystem:      config.RootSystem,
			Sources:         config.Sources,
			SourceRuntime:   config.SourceRuntime,
			SourceContent:   config.SourceContent,
			ManagedSources:  config.ManagedSources,
			Artifacts:       config.Artifacts,
			Refresh:         config.Refresh,
			RefreshCompiled: config.RefreshCompiled,
			Repository:      config.Repository,
			SecretLifecycle: config.SecretLifecycle,
			Policy:          config.Policy,
		},
	)
	if err != nil {
		return nil, err
	}

	var _ installFlow.API = service

	return service, nil
}
