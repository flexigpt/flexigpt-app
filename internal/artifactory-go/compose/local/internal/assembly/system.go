package assembly

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/iofs"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/jsonschema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/sqlite"
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
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/value"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

type Config struct {
	BaseDirectory              string
	EmbeddedProviders          map[string]fs.FS
	AdditionalSources          []driver.Driver
	SchemaCodecs               []schema.Codec
	Decoders                   []ingest.Decoder
	ArtifactIDProvider         artifact.IDProvider
	Clock                      clockutil.Clock
	RootMutationPolicy         root.Policy
	FilesystemTraversalPolicy  *fsdir.TraversalPolicy
	ProtectedOverlayNamespaces []overlayModel.Namespace
	StoreOverlayNamespaces     []overlayModel.Namespace
	// SecretValues transfers to Components only after Open succeeds. A caller
	// retains ownership when Open returns an error.
	SecretValues value.ValueStore
}

// Components is local deployment assembly only. Each field is a named entity
// or flow capability, not a provider aggregate or a reusable component bag.
type Components struct {
	Roots             root.API
	Sources           source.API
	Artifacts         artifact.API
	Catalog           catalog.API
	Definitions       definition.API
	Schemas           schema.API
	Refresh           refreshFlow.API
	Resources         resourceFlow.API
	NativeResources   resourceFlow.NativePathAPI
	ManagedArtifacts  managepackageFlow.API
	Install           installFlow.API
	ProtectedOverlays overlay.API
	StoreOverlays     overlay.StoreAPI
	SecretBindings    secret.API
	SecretRuntime     secret.RuntimeAPI
	SecretLifecycle   secret.LifecycleAPI
	ArtifactCleanup   artifactcleanupFlow.API
	metadata          *sqlite.Store
	secretValues      value.ValueStore
	closeOnce         sync.Once
	closeErr          error
}

func Open(ctx context.Context, config Config) (output *Components, returnErr error) {
	if ctx == nil {
		return nil, fmt.Errorf("%w: artifact system context is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if config.BaseDirectory == "" {
		return nil, fmt.Errorf("%w: artifact system base directory is empty", spec.ErrInvalid)
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
	base, err := filepath.Abs(config.BaseDirectory)
	if err != nil {
		return nil, err
	}
	base = filepath.Clean(base)
	if err := ensureStoreLayout(base); err != nil {
		return nil, err
	}
	metadata, err := sqlite.Open(ctx, filepath.Join(base, spec.ArtifactStoreMetadataFileName))
	if err != nil {
		return nil, err
	}
	defer func() {
		if returnErr != nil && metadata != nil {
			_ = metadata.Close()
		}
	}()
	schemas, err := jsonschema.NewRegistry(codecs...)
	if err != nil {
		return nil, err
	}
	if err := ingest.BindSchemaCatalog(decoders, schemas); err != nil {
		return nil, err
	}
	filesystemAdapter, err := fsdir.NewWithTraversalPolicy(config.FilesystemTraversalPolicy)
	if err != nil {
		return nil, err
	}
	managedAdapter, err := managedfs.New(
		filepath.Join(base, spec.ArtifactStoreContentDirectoryName),
		filepath.Join(base, spec.ArtifactStoreStagingDirectoryName),
	)
	if err != nil {
		return nil, err
	}
	embeddedAdapter, err := iofs.New(config.EmbeddedProviders)
	if err != nil {
		return nil, err
	}
	drivers := make([]driver.Driver, 0, 3+len(config.AdditionalSources))
	drivers = append(drivers, filesystemAdapter, embeddedAdapter, managedAdapter)
	drivers = append(drivers, config.AdditionalSources...)
	sourceRegistry, err := source.NewRegistry(drivers...)
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
	rootRepository := metadata.Roots()
	sourceRepository := metadata.Sources()
	artifactRepository := metadata.Artifacts()
	definitionRepository := metadata.Definitions()
	definitionService, err := definition.NewService(definitionRepository)
	if err != nil {
		return nil, err
	}
	rootService, err := root.NewService(rootRepository, config.Clock, config.RootMutationPolicy)
	if err != nil {
		return nil, err
	}
	sourceRuntime, err := source.NewRuntime(sourceRepository, sourceRegistry)
	if err != nil {
		return nil, err
	}
	sourceService, err := source.NewService(
		sourceRepository,
		sourceRegistry,
		rootRepository,
		config.Clock,
		config.RootMutationPolicy,
	)
	if err != nil {
		return nil, err
	}
	artifactService, err := artifact.NewService(
		artifactRepository,
		definitionService,
		config.Clock,
		config.RootMutationPolicy,
	)
	if err != nil {
		return nil, err
	}
	catalogService, err := catalog.NewService(metadata.Catalog(), definitionService)
	if err != nil {
		return nil, err
	}
	secretLifecycle, err := secret.NewLifecycleService(metadata.Secrets(), config.Clock, config.SecretValues)
	if err != nil {
		return nil, err
	}
	secretBindings, err := secret.NewBindingService(
		metadata.Secrets(),
		artifactRepository,
		config.Clock,
		config.ProtectedOverlayNamespaces,
		config.SecretValues,
		secretLifecycle,
	)
	if err != nil {
		return nil, err
	}
	secretRuntime, err := secret.NewRuntimeService(
		metadata.Secrets(),
		artifactRepository,
		config.ProtectedOverlayNamespaces,
		config.SecretValues,
	)
	if err != nil {
		return nil, err
	}
	overlayService, err := overlay.NewService(
		metadata.Overlays(),
		artifactRepository,
		config.Clock,
		config.RootMutationPolicy,
		config.ProtectedOverlayNamespaces,
		config.StoreOverlayNamespaces,
		secretLifecycle,
	)
	if err != nil {
		return nil, err
	}
	cleanupService, err := artifactcleanupFlow.NewService(
		metadata.ArtifactCleanup(),
		artifactRepository,
		config.RootMutationPolicy,
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
		artifactRepository,
		metadata.RefreshStates(),
		discovery,
		synchronizer,
		metadata.Publisher(),
		config.Clock,
		config.RootMutationPolicy,
	)
	if err != nil {
		return nil, err
	}
	resources, nativeResources, err := resourceFlow.NewService(
		artifactRepository,
		definitionService,
		refreshService,
		sourceRuntime,
	)
	if err != nil {
		return nil, err
	}
	managedArtifacts, err := managepackageFlow.NewService(
		managepackageFlow.Config{
			Artifacts:       artifactService,
			Refresh:         refreshService,
			Sources:         sourceService,
			Runtime:         sourceRuntime,
			ContentMutation: sourceService,
			Packages:        sourceRegistry,
			Policy:          config.RootMutationPolicy,
		},
	)
	if err != nil {
		return nil, err
	}
	installer, err := installFlow.NewService(
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
			Repository:      metadata,
			SecretLifecycle: secretLifecycle,
			Policy:          config.RootMutationPolicy,
		},
	)
	if err != nil {
		return nil, err
	}
	return &Components{
		Roots:             rootService,
		Sources:           sourceService,
		Artifacts:         artifactService,
		Catalog:           catalogService,
		Definitions:       definitionService,
		Schemas:           schemas,
		Refresh:           refreshService,
		Resources:         resources,
		NativeResources:   nativeResources,
		ManagedArtifacts:  managedArtifacts,
		Install:           installer,
		ProtectedOverlays: overlayService,
		StoreOverlays:     overlayService,
		SecretBindings:    secretBindings,
		SecretRuntime:     secretRuntime,
		SecretLifecycle:   secretLifecycle,
		ArtifactCleanup:   cleanupService,
		metadata:          metadata,
		secretValues:      config.SecretValues,
	}, nil
}

func (c *Components) Close() error {
	if c == nil {
		return nil
	}
	c.closeOnce.Do(func() {
		if c.metadata != nil {
			c.closeErr = errors.Join(c.closeErr, c.metadata.Close())
			c.metadata = nil
		}
		if c.secretValues != nil {
			c.closeErr = errors.Join(c.closeErr, c.secretValues.Close())
			c.secretValues = nil
		}
	})
	return c.closeErr
}
