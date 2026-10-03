package assembly

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/iofs"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/jsonschema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/sqlite"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/idprovider"
	artifactimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/impl"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"

	artifactcleanupFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/artifactcleanup"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installCompose "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/compose"

	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	managepackageCompose "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/compose"

	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	refreshimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh/impl"

	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	resourceimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/impl"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/impl"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	secretimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/impl"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/value"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	sourceimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/impl"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	ingestimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/impl"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

type Config struct {
	BaseDirectory     string
	EmbeddedProviders map[string]fs.FS
	AdditionalSources []driver.Driver

	SchemaCodecs []schema.Codec
	Decoders     []ingest.Decoder

	ArtifactIDProvider        idprovider.Provider
	Clock                     clockutil.Clock
	RootMutationPolicy        rootModel.RootPolicy
	FilesystemTraversalPolicy *fsdir.TraversalPolicy

	ProtectedOverlayNamespaces []overlayModel.Namespace
	StoreOverlayNamespaces     []overlayModel.Namespace
	SecretValues               value.ValueStore
}

type Components struct {
	Roots       root.API
	Sources     source.API
	Artifacts   artifact.API
	Catalog     catalog.API
	Definitions definition.API
	Schemas     schema.API
	Refresh     refreshFlow.API
	Resources   resourceFlow.API

	ManagedArtifacts managepackageFlow.API
	Install          installFlow.API

	ProtectedOverlays overlay.API
	StoreOverlays     overlay.StoreAPI

	SecretBindings  secret.API
	SecretRuntime   secret.RuntimeAPI
	SecretLifecycle secret.LifecycleAPI

	ArtifactCleanup artifactcleanupFlow.API

	metadata   *sqlite.Store
	localState *secretimpl.Service
}

func Open(
	ctx context.Context,
	config Config,
) (output *Components, returnErr error) {
	secretValuesTransferred := false
	var (
		metadata   *sqlite.Store
		localState *secretimpl.Service
	)

	defer func() {
		if returnErr == nil {
			return
		}

		if localState != nil {
			_ = localState.Close()
		} else if !secretValuesTransferred &&
			config.SecretValues != nil {
			_ = config.SecretValues.Close()
		}

		if metadata != nil {
			_ = metadata.Close()
		}
	}()

	if config.BaseDirectory == "" {
		return nil, fmt.Errorf(
			"%w: artifact system base directory is empty",
			spec.ErrInvalid,
		)
	}
	if config.Clock == nil {
		config.Clock = clockutil.System{}
	}
	if config.ArtifactIDProvider == nil {
		config.ArtifactIDProvider = idprovider.NewUUIDProvider()
	}

	schemaCodecs, err := schema.NormalizeCodecs(config.SchemaCodecs)
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

	metadata, err = sqlite.Open(
		ctx,
		filepath.Join(base, spec.ArtifactStoreMetadataFileName),
	)
	if err != nil {
		return nil, err
	}

	shareableRegistry, err := jsonschema.NewRegistry(schemaCodecs...)
	if err != nil {
		return nil, err
	}

	if err := ingest.BindSchemaCatalog(decoders, shareableRegistry); err != nil {
		return nil, err
	}

	filesystemAdapter, err := fsdir.NewWithTraversalPolicy(
		config.FilesystemTraversalPolicy,
	)
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

	sourceDrivers := make(
		[]driver.Driver,
		0,
		3+len(config.AdditionalSources),
	)
	sourceDrivers = append(
		sourceDrivers,
		filesystemAdapter,
		embeddedAdapter,
		managedAdapter,
	)
	sourceDrivers = append(sourceDrivers, config.AdditionalSources...)

	sourceRegistry, err := sourceimpl.NewRegistry(sourceDrivers...)
	if err != nil {
		return nil, err
	}

	decoderRegistry, err := ingestimpl.NewDecoderRegistry(
		schemaCodecs,
		decoders...,
	)
	if err != nil {
		return nil, err
	}

	rootRepository := metadata.Roots()
	sourceRepository := metadata.Sources()
	artifactRepository := metadata.Artifacts()
	definitionRepository := metadata.Definitions()

	rootService, err := rootimpl.NewService(
		rootRepository,
		config.Clock,
		config.RootMutationPolicy,
	)
	if err != nil {
		return nil, err
	}

	sourceRuntime, err := sourceimpl.NewRuntime(
		sourceRepository,
		sourceRegistry,
	)
	if err != nil {
		return nil, err
	}

	sourceService, err := sourceimpl.NewService(
		sourceRepository,
		sourceRegistry,
		rootRepository,
		config.Clock,
		config.RootMutationPolicy,
	)
	if err != nil {
		return nil, err
	}

	artifactService, err := artifactimpl.NewService(
		artifactRepository,
		definitionRepository,
		config.Clock,
		config.RootMutationPolicy,
	)
	if err != nil {
		return nil, err
	}

	localState, err = secretimpl.NewService(
		metadata.LocalState(),
		artifactRepository,
		config.Clock,
		config.RootMutationPolicy,
		config.ProtectedOverlayNamespaces,
		config.StoreOverlayNamespaces,
		config.SecretValues,
	)
	if err != nil {
		return nil, err
	}
	secretValuesTransferred = true

	if err := localState.RecoverPending(ctx); err != nil {
		return nil, err
	}

	discoveryEngine, err := ingestimpl.NewEngine(decoderRegistry)
	if err != nil {
		return nil, err
	}

	synchronizer, err := artifactimpl.NewSynchronizer(
		config.Clock,
		config.ArtifactIDProvider,
	)
	if err != nil {
		return nil, err
	}

	refreshService, err := refreshimpl.NewService(
		sourceRuntime,
		artifactRepository,
		metadata.RefreshStates(),
		discoveryEngine,
		synchronizer,
		metadata.Publisher(),
		config.Clock,
		config.RootMutationPolicy,
	)
	if err != nil {
		return nil, err
	}

	resourceService, err := resourceimpl.NewService(
		artifactRepository,
		definitionRepository,
		refreshService,
		sourceRuntime,
	)
	if err != nil {
		return nil, err
	}

	managedArtifacts, err := managepackageCompose.Open(
		managepackageCompose.Config{
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

	installer, err := installCompose.Open(
		installCompose.Config{
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
			SecretLifecycle: localState,
			Policy:          config.RootMutationPolicy,
		},
	)
	if err != nil {
		return nil, err
	}

	return &Components{
		Roots:       rootService,
		Sources:     sourceService,
		Artifacts:   artifactService,
		Catalog:     artifactService,
		Definitions: definitionRepository,
		Schemas:     shareableRegistry,
		Refresh:     refreshService,
		Resources:   resourceService,

		ManagedArtifacts: managedArtifacts,
		Install:          installer,

		ProtectedOverlays: localState,
		StoreOverlays:     localState,

		SecretBindings:  localState,
		SecretRuntime:   localState,
		SecretLifecycle: localState,

		ArtifactCleanup: localState,

		metadata:   metadata,
		localState: localState,
	}, nil
}

func (c *Components) Close() error {
	if c == nil {
		return nil
	}

	var output error

	if c.localState != nil {
		output = errors.Join(output, c.localState.Close())
		c.localState = nil
	}

	if c.metadata != nil {
		output = errors.Join(output, c.metadata.Close())
		c.metadata = nil
	}

	return output
}
