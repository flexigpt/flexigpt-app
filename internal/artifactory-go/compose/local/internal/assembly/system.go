package assembly

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/iofs"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/jsonschema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/sqlite"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/idprovider"
	artifactimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/impl"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	managedpackageimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managedpackage/impl"
	refreshimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh/impl"
	resourceimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/impl"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	rootimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/impl"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	secretimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/impl"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/value"
	sourceimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/impl"
	ingestimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/impl"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

type Config struct {
	BaseDirectory     string
	EmbeddedProviders map[string]fs.FS
	AdditionalSources []sourceimpl.Adapter

	ArtifactProviders []provider.Provider
	// ArtifactIDProvider creates IDs for deterministic source synchronization.
	// It never reaches a provider or consumer.
	ArtifactIDProvider        idprovider.Provider
	Clock                     clockutil.Clock
	RootMutationPolicy        rootModel.RootPolicy
	FilesystemTraversalPolicy *fsdir.TraversalPolicy

	ProtectedOverlayNamespaces []overlayModel.Namespace
	StoreOverlayNamespaces     []overlayModel.Namespace
	SecretValues               value.ValueStore
}

type ManagedPackageResult struct {
	Source     sourceModel.Summary
	Generation string
}

type Components struct {
	Roots            *rootimpl.Service
	Sources          *sourceimpl.Service
	Artifacts        *artifactimpl.Service
	Definitions      definition.API
	Refresh          *refreshimpl.Service
	Resources        *resourceimpl.Service
	ShareableSchemas *jsonschema.Registry
	LocatorResolvers []provider.LocatorResolverFactory

	ManagedArtifacts *managedpackageimpl.Service
	SourceRuntime    sourceimpl.Runtime
	LocalState       *secretimpl.Service

	metadata           *sqlite.Store
	managedSources     *sourceimpl.Registry
	rootMutationPolicy rootModel.RootPolicy
}

func Open(
	ctx context.Context,
	config Config,
) (*Components, error) {
	secretValuesTransferred := false
	defer func() {
		if !secretValuesTransferred &&
			config.SecretValues != nil {
			_ = config.SecretValues.Close()
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
	providerRegistry, err := providerRegistryFromConfig(config)
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

	metadata, err := sqlite.Open(
		ctx,
		filepath.Join(
			base,
			spec.ArtifactStoreMetadataFileName,
		),
	)
	if err != nil {
		return nil, err
	}

	registeredSchemas := providerRegistry.Schemas()
	registeredDecoders := providerRegistry.Decoders()

	shareableRegistry, err := jsonschema.NewRegistry(
		registeredSchemas...,
	)
	if err != nil {
		_ = metadata.Close()
		return nil, err
	}

	if err := bindProviderSchemas(registeredDecoders, shareableRegistry); err != nil {
		_ = metadata.Close()
		return nil, err
	}

	filesystemAdapter, err := fsdir.NewWithTraversalPolicy(
		config.FilesystemTraversalPolicy,
	)
	if err != nil {
		_ = metadata.Close()
		return nil, err
	}

	managedAdapter, err := managedfs.New(
		filepath.Join(
			base,
			spec.ArtifactStoreContentDirectoryName,
		),
		filepath.Join(
			base,
			spec.ArtifactStoreStagingDirectoryName,
		),
	)
	if err != nil {
		_ = metadata.Close()
		return nil, err
	}

	embeddedAdapter, err := iofs.New(config.EmbeddedProviders)
	if err != nil {
		_ = metadata.Close()
		return nil, err
	}

	sourceAdapters := make([]sourceimpl.Adapter, 0, 3+len(config.AdditionalSources))
	sourceAdapters = append(
		sourceAdapters,
		filesystemAdapter,
		embeddedAdapter,
		managedAdapter,
	)
	sourceAdapters = append(sourceAdapters, config.AdditionalSources...)

	sourceRegistry, err := sourceimpl.NewRegistry(sourceAdapters...)
	if err != nil {

		_ = metadata.Close()
		return nil, err
	}
	decoderRegistry, err := ingestimpl.NewDecoderRegistry(
		registeredSchemas,
		registeredDecoders...,
	)
	if err != nil {

		_ = metadata.Close()
		return nil, err
	}

	sourceRepository := metadata.Sources()
	rootRepository := metadata.Roots()
	artifactRepository := metadata.Artifacts()
	definitionRepository := metadata.Definitions()
	sourceRuntime, err := sourceimpl.NewRuntime(
		sourceRepository,
		sourceRegistry,
	)
	if err != nil {

		_ = metadata.Close()
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

		_ = metadata.Close()
		return nil, err
	}
	rootService, err := rootimpl.NewService(
		rootRepository,
		config.Clock,
		config.RootMutationPolicy,
	)
	if err != nil {

		_ = metadata.Close()
		return nil, err
	}
	artifactService, err := artifactimpl.NewService(
		artifactRepository,
		definitionRepository,
		config.Clock,
		config.RootMutationPolicy,
	)
	if err != nil {
		_ = metadata.Close()
		return nil, err
	}

	localStateService, err := secretimpl.NewService(
		metadata.LocalState(),
		artifactRepository,
		config.Clock,
		config.RootMutationPolicy,
		config.ProtectedOverlayNamespaces,
		config.StoreOverlayNamespaces,
		config.SecretValues,
	)
	if err != nil {
		_ = metadata.Close()
		return nil, err
	}
	if err := localStateService.RecoverPending(ctx); err != nil {
		_ = localStateService.Close()
		_ = metadata.Close()
		return nil, err
	}

	discoveryEngine, err := ingestimpl.NewEngine(
		decoderRegistry,
	)
	if err != nil {
		_ = localStateService.Close()
		_ = metadata.Close()
		return nil, err
	}
	synchronizer, err := artifactimpl.NewSynchronizer(
		config.Clock,
		config.ArtifactIDProvider,
	)
	if err != nil {
		_ = localStateService.Close()
		_ = metadata.Close()
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
		_ = localStateService.Close()
		_ = metadata.Close()
		return nil, err
	}

	resourceService, err := resourceimpl.NewService(
		artifactRepository,
		definitionRepository,
		refreshService,
		sourceRuntime,
	)
	if err != nil {
		_ = localStateService.Close()
		_ = metadata.Close()
		return nil, err
	}

	components := &Components{
		Roots:              rootService,
		Sources:            sourceService,
		Artifacts:          artifactService,
		Definitions:        definitionRepository,
		Refresh:            refreshService,
		Resources:          resourceService,
		ShareableSchemas:   shareableRegistry,
		LocatorResolvers:   providerRegistry.LocatorResolvers(),
		SourceRuntime:      sourceRuntime,
		LocalState:         localStateService,
		metadata:           metadata,
		managedSources:     sourceRegistry,
		rootMutationPolicy: config.RootMutationPolicy,
	}
	managedArtifacts, err := managedpackageimpl.NewService(
		managedpackageimpl.Dependencies{
			Artifacts: artifactService,
			Refresh:   refreshService,
			Policy:    config.RootMutationPolicy,
			GetSourceState: func(
				ctx context.Context,
				rootID rootModel.RootID,
				sourceID sourceModel.SourceID,
			) (managedpackageimpl.SourceState, error) {
				result, err := components.getManagedSourceState(
					ctx,
					rootID,
					sourceID,
				)
				if err != nil {
					return managedpackageimpl.SourceState{}, err
				}
				return managedpackageimpl.SourceState{
					Source:     result.Source,
					Generation: result.Generation,
				}, nil
			},
			PublishPackage: func(
				ctx context.Context,
				rootID rootModel.RootID,
				sourceID sourceModel.SourceID,
				expectedRevision uint64,
				publication sourceModel.ManagedPackagePublication,
			) (managedpackageimpl.SourceState, error) {
				result, err := components.publishManagedPackageForMutableRoot(
					ctx,
					rootID,
					sourceID,
					expectedRevision,
					publication,
				)
				if err != nil {
					return managedpackageimpl.SourceState{}, err
				}
				return managedpackageimpl.SourceState{
					Source:     result.Source,
					Generation: result.Generation,
				}, nil
			},
			PublishProtectedPackage: func(
				ctx context.Context,
				rootID rootModel.RootID,
				sourceID sourceModel.SourceID,
				expectedRevision uint64,
				publication sourceModel.ManagedPackagePublication,
			) (managedpackageimpl.SourceState, error) {
				result, err := components.publishProtectedManagedPackage(
					ctx,
					rootID,
					sourceID,
					expectedRevision,
					publication,
				)
				if err != nil {
					return managedpackageimpl.SourceState{}, err
				}
				return managedpackageimpl.SourceState{
					Source:     result.Source,
					Generation: result.Generation,
				}, nil
			},
			RemovePackage:          components.removeManagedArtifactPackage,
			RemoveProtectedPackage: components.removeProtectedManagedArtifactPackage,
			PruneDiscoveryLocator:  components.pruneManagedDeclarationDiscovery,
		},
	)
	if err != nil {
		secretValuesTransferred = true
		_ = components.Close()
		return nil, err
	}
	components.ManagedArtifacts = managedArtifacts
	secretValuesTransferred = true
	return components, nil
}

func (c *Components) Close() error {
	if c == nil {
		return nil
	}
	var closeErrors []error
	if c.LocalState != nil {
		if err := c.LocalState.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
		c.LocalState = nil
	}
	if c.metadata != nil {
		if err := c.metadata.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	return errors.Join(closeErrors...)
}

// getManagedSourceState returns the current confirmed snapshot generation used
// as the optimistic token for managed package publication and removal. It does
// not expose Source configuration or the private acknowledged-generation
// metadata field.
func (c *Components) getManagedSourceState(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) (ManagedPackageResult, error) {
	if c == nil ||
		c.SourceRuntime == nil ||
		c.managedSources == nil {
		return ManagedPackageResult{}, spec.ErrClosed
	}
	if ctx == nil {
		return ManagedPackageResult{}, fmt.Errorf(
			"%w: managed Source state context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return ManagedPackageResult{}, err
	}
	v, err := c.SourceRuntime.Get(ctx, rootID, sourceID)
	if err != nil {
		return ManagedPackageResult{}, err
	}
	if !c.managedSources.SupportsManagedPackages(v.Kind) {
		return ManagedPackageResult{}, fmt.Errorf(
			"%w: source kind %q is not writable",
			spec.ErrUnsupported,
			v.Kind,
		)
	}
	generation, err := sourceSnapshotGeneration(
		ctx,
		c.SourceRuntime,
		v,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}
	return ManagedPackageResult{
		Source:     v.Summary(),
		Generation: generation,
	}, nil
}

// PublishManagedPackage publishes a package and advances the Source revision
// only when the resulting snapshot generation changed. The revision advance
// makes prior Source refresh state stale.
//
// Source-side publication and SQLite metadata publication intentionally remain
// separate operations. If the package write succeeds but the revision advance
// conflicts, the caller receives the conflict and must reload before retrying.
func (c *Components) publishManagedPackageForMutableRoot(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
	publication sourceModel.ManagedPackagePublication,
) (ManagedPackageResult, error) {
	return c.publishManagedPackage(
		ctx,
		rootID,
		sourceID,
		expectedSourceRevision,
		publication,
		false,
	)
}

// PublishProtectedManagedPackage is the trusted protected-topology package
// publication path. Artifact Store assigns no built-in meaning to this
// method. Application installers use it only for a Root declared protected by
// the application RootPolicy.
func (c *Components) publishProtectedManagedPackage(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
	publication sourceModel.ManagedPackagePublication,
) (ManagedPackageResult, error) {
	if c == nil || !c.isProtectedRoot(rootID) {
		return ManagedPackageResult{}, fmt.Errorf(
			"%w: Root %q is not a declared protected topology Root",
			spec.ErrProtected,
			rootID,
		)
	}
	if err := installFlow.RequirePrivileged(ctx); err != nil {
		return ManagedPackageResult{}, err
	}
	return c.publishManagedPackage(
		ctx,
		rootID,
		sourceID,
		expectedSourceRevision,
		publication,
		true,
	)
}

// RemoveManagedPackage removes one complete package and advances the Source
// revision after successful source-side removal.
func (c *Components) removeManagedPackageForMutableRoot(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
	address sourceModel.ManagedPackageAddress,
	expectedGeneration string,
) (ManagedPackageResult, error) {
	return c.removeManagedPackage(
		ctx,
		rootID,
		sourceID,
		expectedSourceRevision,
		address,
		expectedGeneration,
		false,
	)
}

// RemoveProtectedManagedPackage is the trusted protected-topology removal
// path. It is reserved for an explicit installer or update workflow.
func (c *Components) removeProtectedManagedPackage(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
	address sourceModel.ManagedPackageAddress,
	expectedGeneration string,
) (ManagedPackageResult, error) {
	if c == nil || !c.isProtectedRoot(rootID) {
		return ManagedPackageResult{}, fmt.Errorf(
			"%w: Root %q is not a declared protected topology Root",
			spec.ErrProtected,
			rootID,
		)
	}
	if err := installFlow.RequirePrivileged(ctx); err != nil {
		return ManagedPackageResult{}, err
	}
	return c.removeManagedPackage(
		ctx,
		rootID,
		sourceID,
		expectedSourceRevision,
		address,
		expectedGeneration,
		true,
	)
}

func (c *Components) publishManagedPackage(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
	publication sourceModel.ManagedPackagePublication,
	allowProtected bool,
) (ManagedPackageResult, error) {
	if c == nil {
		return ManagedPackageResult{}, spec.ErrClosed
	}
	if c.isProtectedRoot(rootID) && !allowProtected {
		return ManagedPackageResult{}, fmt.Errorf(
			"%w: managed package publication for protected Root %q requires the protected installer path",
			spec.ErrProtected,
			rootID,
		)
	}
	if err := rootimpl.RequireMutableRoot(ctx, c.rootMutationPolicy, rootID); err != nil {
		return ManagedPackageResult{}, err
	}

	v, err := c.managedSource(
		ctx,
		rootID,
		sourceID,
		expectedSourceRevision,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}

	beforeGeneration, err := sourceSnapshotGeneration(
		ctx,
		c.SourceRuntime,
		v,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}

	// An empty ExpectedGeneration is intentional. It means create-only
	// publication: an exact same-content package is an idempotent replay,
	// while different content at an existing address must conflict.
	//
	// "managedartifact.Service" supplies an expected generation when the caller
	// explicitly allows automatic replacement. Other callers such as managed
	// Collection updates provide their own compare-and-swap generation.
	generation, err := c.managedSources.PublishPackage(
		ctx,
		v,
		publication,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}

	result := ManagedPackageResult{
		Source:     v.Summary(),
		Generation: generation,
	}
	contentChanged := generation != beforeGeneration
	if !contentChanged {
		return result, nil
	}

	updated, err := c.Sources.MarkContentChanged(
		ctx,
		rootID,
		sourceID,
		expectedSourceRevision,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}
	result.Source = updated
	return result, nil
}

func (c *Components) removeManagedPackage(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
	address sourceModel.ManagedPackageAddress,
	expectedGeneration string,
	allowProtected bool,
) (ManagedPackageResult, error) {
	if c == nil {
		return ManagedPackageResult{}, spec.ErrClosed
	}
	if c.isProtectedRoot(rootID) && !allowProtected {
		return ManagedPackageResult{}, fmt.Errorf(
			"%w: managed package removal for protected Root %q requires the protected installer path",
			spec.ErrProtected,
			rootID,
		)
	}
	if err := rootimpl.RequireMutableRoot(ctx, c.rootMutationPolicy, rootID); err != nil {
		return ManagedPackageResult{}, err
	}
	if err := address.Validate(); err != nil {
		return ManagedPackageResult{}, err
	}
	if err := spec.ValidateSourceGeneration(expectedGeneration); err != nil {
		return ManagedPackageResult{}, err
	}

	v, err := c.managedSource(
		ctx,
		rootID,
		sourceID,
		expectedSourceRevision,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}

	beforeGeneration, err := sourceSnapshotGeneration(
		ctx,
		c.SourceRuntime,
		v,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}
	if beforeGeneration != expectedGeneration {
		exists, err := managedPackageExists(
			ctx,
			c.SourceRuntime,
			v,
			address,
		)
		if err != nil {
			return ManagedPackageResult{}, err
		}
		if exists {
			return ManagedPackageResult{}, fmt.Errorf(
				"%w: managed Source changed before package removal",
				spec.ErrConflict,
			)
		}
		updated, err := c.Sources.MarkContentChanged(
			ctx,
			rootID,
			sourceID,
			expectedSourceRevision,
		)
		if err != nil {
			return ManagedPackageResult{}, err
		}
		return ManagedPackageResult{
			Source:     updated,
			Generation: beforeGeneration,
		}, nil
	}

	if err := c.managedSources.RemovePackage(
		ctx,
		v,
		address,
		expectedGeneration,
	); err != nil {
		return ManagedPackageResult{}, err
	}

	generation, err := sourceSnapshotGeneration(ctx, c.SourceRuntime, v)
	if err != nil {
		return ManagedPackageResult{}, err
	}
	if generation == beforeGeneration {
		return ManagedPackageResult{
			Source:     v.Summary(),
			Generation: generation,
		}, nil
	}
	updated, err := c.Sources.MarkContentChanged(
		ctx,
		rootID,
		sourceID,
		expectedSourceRevision,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}

	return ManagedPackageResult{
		Source:     updated,
		Generation: generation,
	}, nil
}

func managedPackageExists(
	ctx context.Context,
	runtime sourceimpl.Runtime,
	v sourceModel.Source,
	address sourceModel.ManagedPackageAddress,
) (bool, error) {
	snapshot, err := runtime.Open(ctx, v)
	if err != nil {
		return false, err
	}
	dir, err := address.Directory()
	if err != nil {
		return false, err
	}
	entry, statErr := snapshot.Stat(ctx, dir)
	confirmErr := snapshot.Confirm(ctx)
	closeErr := snapshot.Close()
	if confirmErr != nil || closeErr != nil {
		return false, errors.Join(statErr, confirmErr, closeErr)
	}
	if errors.Is(statErr, spec.ErrNotFound) {
		return false, nil
	}
	if statErr != nil {
		return false, statErr
	}
	if !entry.IsDirectory {
		return false, fmt.Errorf(
			"%w: managed package %q is not a directory",
			spec.ErrInvalid,
			address,
		)
	}
	return true, nil
}

func (c *Components) isProtectedRoot(
	rootID rootModel.RootID,
) bool {
	return c != nil &&
		c.rootMutationPolicy != nil &&
		c.rootMutationPolicy.IsProtectedRoot(rootID)
}

func (c *Components) managedSource(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
) (sourceModel.Source, error) {
	if c == nil ||
		c.Sources == nil ||
		c.SourceRuntime == nil ||
		c.managedSources == nil {
		return sourceModel.Source{}, spec.ErrClosed
	}
	if ctx == nil {
		return sourceModel.Source{}, fmt.Errorf(
			"%w: managed Source context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return sourceModel.Source{}, err
	}
	if expectedSourceRevision == 0 {
		return sourceModel.Source{}, fmt.Errorf(
			"%w: expected source revision is required",
			spec.ErrInvalid,
		)
	}
	v, err := c.SourceRuntime.Get(ctx, rootID, sourceID)
	if err != nil {
		return sourceModel.Source{}, err
	}
	if v.Revision != expectedSourceRevision {
		return sourceModel.Source{}, spec.ErrConflict
	}
	if !c.managedSources.SupportsManagedPackages(v.Kind) {
		return sourceModel.Source{}, fmt.Errorf(
			"%w: source kind %q is not writable",
			spec.ErrUnsupported,
			v.Kind,
		)
	}
	return v, nil
}

func sourceSnapshotGeneration(
	ctx context.Context,
	runtime sourceimpl.Runtime,
	v sourceModel.Source,
) (string, error) {
	snapshot, err := runtime.Open(ctx, v)
	if err != nil {
		return "", err
	}
	generation := snapshot.Generation()
	confirmErr := snapshot.Confirm(ctx)
	closeErr := snapshot.Close()
	if err := errors.Join(confirmErr, closeErr); err != nil {
		return "", err
	}
	return generation, nil
}

func (c *Components) removeManagedArtifactPackage(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedRevision uint64,
	address sourceModel.ManagedPackageAddress,
	expectedGeneration string,
) (managedpackageimpl.SourceState, error) {
	result, err := c.removeManagedPackageForMutableRoot(
		ctx,
		rootID,
		sourceID,
		expectedRevision,
		address,
		expectedGeneration,
	)
	if err != nil {
		return managedpackageimpl.SourceState{}, err
	}
	return managedpackageimpl.SourceState{
		Source:     result.Source,
		Generation: result.Generation,
	}, nil
}

func (c *Components) removeProtectedManagedArtifactPackage(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedRevision uint64,
	address sourceModel.ManagedPackageAddress,
	expectedGeneration string,
) (managedpackageimpl.SourceState, error) {
	result, err := c.removeProtectedManagedPackage(
		ctx,
		rootID,
		sourceID,
		expectedRevision,
		address,
		expectedGeneration,
	)
	if err != nil {
		return managedpackageimpl.SourceState{}, err
	}
	return managedpackageimpl.SourceState{
		Source:     result.Source,
		Generation: result.Generation,
	}, nil
}
