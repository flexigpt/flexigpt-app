package install

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/internal"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	storeSecret "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// Config contains the concrete named capabilities used by Install; it is not
// a generic component bundle.
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
	if config.Roots == nil || config.RootSystem == nil || config.Sources == nil || config.SourceRuntime == nil ||
		config.SourceContent == nil ||
		config.ManagedSources == nil ||
		config.Artifacts == nil ||
		config.Refresh == nil ||
		config.RefreshCompiled == nil ||
		config.Repository == nil ||
		config.SecretLifecycle == nil {
		return nil, fmt.Errorf("%w: Install dependencies are incomplete", spec.ErrInvalid)
	}
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
