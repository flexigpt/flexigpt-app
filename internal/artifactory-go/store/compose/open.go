package compose

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	artifactcleanupFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/artifactcleanup"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

// Open constructs the provider-independent entity and flow aggregate. It does
// not close resources when construction fails: a deployment retains ownership
// until this function successfully returns a Store.
func Open(ctx context.Context, config Config) (*Store, error) {
	if ctx == nil {
		return nil, fmt.Errorf("%w: Artifact Store composition context is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	if config.Clock == nil {
		config.Clock = clockutil.System{}
	}
	if config.ArtifactIDProvider == nil {
		config.ArtifactIDProvider = artifact.NewUUIDIDProvider()
	}
	codecs, err := schema.NormalizeCodecs(config.SchemaCodecs)
	if err != nil {
		return nil, err
	}
	decoders, err := ingest.NormalizeDecoders(config.Decoders)
	if err != nil {
		return nil, err
	}
	schemas, err := config.SchemaFactory.NewCatalog(codecs...)
	if err != nil {
		return nil, err
	}
	if err := ingest.BindSchemaCatalog(decoders, schemas); err != nil {
		return nil, err
	}
	sourceRegistry, err := source.NewRegistry(config.SourceDrivers...)
	if err != nil {
		return nil, err
	}
	decoderRegistry, err := ingest.NewDecoderRegistry(codecs, decoders...)
	if err != nil {
		return nil, err
	}
	discovery, err := ingest.NewEngine(decoderRegistry)
	if err != nil {
		return nil, err
	}
	definitionService, err := definition.NewService(config.DefinitionRepository)
	if err != nil {
		return nil, err
	}
	rootService, err := root.NewService(config.RootRepository, config.Clock, config.RootPolicy)
	if err != nil {
		return nil, err
	}
	sourceRuntime, err := source.NewRuntime(config.SourceRepository, sourceRegistry)
	if err != nil {
		return nil, err
	}
	artifactService, err := artifact.NewService(
		config.ArtifactRepository,
		definitionService,
		config.Clock,
		config.RootPolicy,
	)
	if err != nil {
		return nil, err
	}
	catalogService, err := catalog.NewService(config.CatalogRepository, definitionService)
	if err != nil {
		return nil, err
	}
	secretLifecycle, err := secret.NewLifecycleService(
		config.SecretLifecycleRepository,
		config.Clock,
		config.SecretValues,
	)
	if err != nil {
		return nil, err
	}
	secretBindings, err := secret.NewBindingService(
		config.SecretBindingRepository,
		artifactService,
		config.Clock,
		config.ProtectedOverlayNamespaces,
		config.SecretValues,
		secretLifecycle,
	)
	if err != nil {
		return nil, err
	}
	secretRuntime, err := secret.NewRuntimeService(
		config.SecretRuntimeRepository,
		artifactService,
		config.ProtectedOverlayNamespaces,
		config.SecretValues,
	)
	if err != nil {
		return nil, err
	}
	overlayService, err := overlay.NewService(
		config.OverlayRepository,
		artifactService,
		config.Clock,
		config.RootPolicy,
		config.ProtectedOverlayNamespaces,
		config.StoreOverlayNamespaces,
		secretLifecycle,
	)
	if err != nil {
		return nil, err
	}
	cleanupService, err := artifactcleanupFlow.NewService(
		config.ArtifactCleanupRepository,
		artifactService,
		config.RootPolicy,
		config.Clock,
		secretLifecycle,
	)
	if err != nil {
		return nil, err
	}
	if err := secretLifecycle.RecoverPending(ctx); err != nil {
		return nil, err
	}
	synchronizer, err := artifact.NewSynchronizer(config.Clock, config.ArtifactIDProvider)
	if err != nil {
		return nil, err
	}
	refreshService, err := refreshFlow.NewService(
		sourceRuntime,
		config.ArtifactRepository,
		config.RefreshStates,
		discovery,
		discovery,
		synchronizer,
		config.RefreshPublisher,
		config.Clock,
		config.RootPolicy,
	)
	if err != nil {
		return nil, err
	}
	// Refresh is the narrow injected aggregate publisher for Source lifecycle
	// transitions; Source itself does not import the flow implementation.
	sourceService, err := source.NewService(
		config.SourceRepository,
		sourceRegistry,
		rootService,
		config.Clock,
		config.RootPolicy,
		refreshService,
	)
	if err != nil {
		return nil, err
	}
	resources, nativeResources, err := resourceFlow.NewService(
		config.ArtifactRepository,
		definitionService,
		refreshService,
		sourceRuntime,
	)
	if err != nil {
		return nil, err
	}
	managedPackages, err := managepackageFlow.NewService(
		managepackageFlow.Config{
			Artifacts:       artifactService,
			Refresh:         refreshService,
			Sources:         sourceService,
			Runtime:         sourceRuntime,
			ContentMutation: sourceService,
			Packages:        sourceRegistry,
			Policy:          config.RootPolicy,
		},
	)
	if err != nil {
		return nil, err
	}
	topology, err := installFlow.NewService(
		installFlow.Config{
			Roots:           rootService,
			RootSystem:      rootService,
			Sources:         sourceService,
			SourceRuntime:   sourceRuntime,
			SourceContent:   sourceService,
			ManagedSources:  sourceRegistry,
			Artifacts:       artifactService,
			Refresh:         refreshService,
			RefreshCompiled: refreshService,
			Repository:      config.InstallationRepository,
			SecretLifecycle: secretLifecycle,
			Policy:          config.RootPolicy,
		},
	)
	if err != nil {
		return nil, err
	}
	return &Store{
		Roots:                  rootService,
		Sources:                sourceService,
		Artifacts:              artifactService,
		Catalog:                catalogService,
		Definitions:            definitionService,
		Schemas:                schemas,
		Refresh:                refreshService,
		Resources:              resources,
		TrustedNativeResources: nativeResources,
		ManagedPackages:        managedPackages,
		Topology:               topology,
		ArtifactCleanup:        cleanupService,
		ProtectedOverlays:      overlayService,
		StoreOverlays:          overlayService,
		SecretBindings:         secretBindings,
		SecretRuntime:          secretRuntime,
		SecretLifecycle:        secretLifecycle,
		Protection:             protection{policy: config.RootPolicy},
		shutdown:               config.Shutdown,
	}, nil
}

func validateConfig(config Config) error {
	if config.RootRepository == nil || config.SourceRepository == nil || config.ArtifactRepository == nil ||
		config.CatalogRepository == nil ||
		config.DefinitionRepository == nil ||
		config.RefreshStates == nil ||
		config.RefreshPublisher == nil ||
		config.OverlayRepository == nil ||
		config.SecretBindingRepository == nil ||
		config.SecretRuntimeRepository == nil ||
		config.SecretLifecycleRepository == nil ||
		config.ArtifactCleanupRepository == nil ||
		config.InstallationRepository == nil ||
		config.SchemaFactory == nil {
		return fmt.Errorf("%w: Artifact Store composition dependencies are incomplete", spec.ErrInvalid)
	}
	return nil
}
