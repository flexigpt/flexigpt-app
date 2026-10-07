package install

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/internal"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	storeSecret "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
)

// Config contains the concrete named capabilities used by Install; it is not
// a generic component group.
type Config struct {
	Roots           root.API
	RootSystem      root.System
	Sources         source.API
	SourceRuntime   source.Runtime
	SourceContent   source.ContentMutation
	ManagedSources  managedpackage.Runtime
	Artifacts       artifact.API
	Refresh         refreshFlow.API
	RefreshCompiled refreshFlow.CompiledDocumentRegistrar
	Repository      Repository
	SecretLifecycle storeSecret.LifecycleAPI
	Policy          root.Policy
}

func NewService(config Config) (API, error) {
	return internal.NewService(
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
}
