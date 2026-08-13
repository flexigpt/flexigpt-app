package system

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/managedartifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/embedded"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/managed"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/sqlite"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type Config struct {
	BaseDirectory             string
	EmbeddedProviders         map[string]fs.FS
	AdditionalSources         []source.Adapter
	Decoders                  []discovery.Decoder
	Clock                     clockutil.Clock
	ShareableCodecs           []shareable.Codec
	RootMutationPolicy        protection.RootPolicy
	FilesystemTraversalPolicy *fsdir.TraversalPolicy
}

// shareableSchemaBinder is optional so Artifact Store remains generic. A
// decoder that emits shareable document-derived Definitions may require the
// registered schema registry before it accepts source bytes.
type shareableSchemaBinder interface {
	BindShareableSchemas(r *shareable.Registry) error
}

type ManagedPackageResult struct {
	Source     source.Summary
	Generation string
}

type Components struct {
	Roots            *root.Service
	Sources          *source.Service
	Collections      *collection.Service
	Artifacts        *artifact.Service
	Refresh          *refresh.Service
	ShareableSchemas *shareable.Registry

	ManagedArtifacts *managedartifact.Service
	CollectionReader collection.Reader
	ArtifactReader   artifact.Reader
	Catalogs         catalog.Reader
	SourceRuntime    source.Runtime
	discovery        *discovery.Engine

	metadata           *sqlite.Store
	managedSources     *source.Registry
	decoderIDs         map[basespec.DecoderID]struct{}
	rootMutationPolicy protection.RootPolicy
}

func Open(
	ctx context.Context,
	config Config,
) (*Components, error) {
	if config.BaseDirectory == "" {
		return nil, fmt.Errorf(
			"%w: artifact system base directory is empty",
			basespec.ErrInvalid,
		)
	}
	if config.Clock == nil {
		config.Clock = clockutil.System{}
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
			artifactbuiltin.ArtifactStoreMetadataFileName,
		),
	)
	if err != nil {
		return nil, err
	}

	shareableRegistry, err := shareable.NewRegistry(
		config.ShareableCodecs...,
	)
	if err != nil {
		_ = metadata.Close()
		return nil, err
	}

	if err := bindShareableSchemas(config.Decoders, shareableRegistry); err != nil {
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

	managedAdapter, err := managed.New(
		filepath.Join(
			base,
			artifactbuiltin.ArtifactStoreContentDirectoryName,
		),
		filepath.Join(
			base,
			artifactbuiltin.ArtifactStoreStagingDirectoryName,
		),
	)
	if err != nil {
		_ = metadata.Close()
		return nil, err
	}

	embeddedAdapter, err := embedded.New(config.EmbeddedProviders)
	if err != nil {
		_ = metadata.Close()
		return nil, err
	}

	sourceAdapters := make([]source.Adapter, 0, 3+len(config.AdditionalSources))
	sourceAdapters = append(
		sourceAdapters,
		filesystemAdapter,
		embeddedAdapter,
		managedAdapter,
	)
	sourceAdapters = append(sourceAdapters, config.AdditionalSources...)

	sourceRegistry, err := source.NewRegistry(sourceAdapters...)
	if err != nil {

		_ = metadata.Close()
		return nil, err
	}
	decoderRegistry, err := discovery.NewDecoderRegistry(config.Decoders...)
	if err != nil {

		_ = metadata.Close()
		return nil, err
	}

	sourceRepository := metadata.Sources()
	rootRepository := metadata.Roots()
	collectionRepository := metadata.Collections()
	catalogRepository := metadata.Catalogs()
	artifactRepository := metadata.Artifacts()
	sourceRuntime, err := source.NewRuntime(
		sourceRepository,
		sourceRegistry,
	)
	if err != nil {

		_ = metadata.Close()
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

		_ = metadata.Close()
		return nil, err
	}
	rootService, err := root.NewService(
		rootRepository,
		config.Clock,
		config.RootMutationPolicy,
	)
	if err != nil {

		_ = metadata.Close()
		return nil, err
	}
	collectionService, err := collection.NewService(
		collectionRepository,
		sourceService,
		config.Clock,
		config.RootMutationPolicy,
	)
	if err != nil {

		_ = metadata.Close()
		return nil, err
	}
	artifactService, err := artifact.NewService(
		artifactRepository,
		collectionRepository,
		catalogRepository,
		config.Clock,
		config.RootMutationPolicy,
	)
	if err != nil {

		_ = metadata.Close()
		return nil, err
	}
	discoveryEngine, err := discovery.NewEngine(
		decoderRegistry,
		config.Clock,
	)
	if err != nil {

		_ = metadata.Close()
		return nil, err
	}
	reconciler, err := artifact.NewReconciler(
		config.Clock,
	)
	if err != nil {

		_ = metadata.Close()
		return nil, err
	}
	decoderIDs := make(map[basespec.DecoderID]struct{}, len(config.Decoders))
	for _, decoder := range config.Decoders {
		decoderIDs[decoder.ID()] = struct{}{}
	}

	refreshService, err := refresh.NewService(
		collectionRepository,
		catalogRepository,
		sourceRuntime,
		artifactRepository,
		discoveryEngine,
		reconciler,
		metadata.Publisher(),
		config.Clock,
		config.RootMutationPolicy,
	)
	if err != nil {

		_ = metadata.Close()
		return nil, err
	}

	components := &Components{
		Roots:              rootService,
		Sources:            sourceService,
		Collections:        collectionService,
		Artifacts:          artifactService,
		Refresh:            refreshService,
		ShareableSchemas:   shareableRegistry,
		CollectionReader:   collectionRepository,
		ArtifactReader:     artifactRepository,
		Catalogs:           catalogRepository,
		SourceRuntime:      sourceRuntime,
		discovery:          discoveryEngine,
		metadata:           metadata,
		managedSources:     sourceRegistry,
		decoderIDs:         decoderIDs,
		rootMutationPolicy: config.RootMutationPolicy,
	}
	managedArtifacts, err := managedartifact.NewService(
		managedartifact.Dependencies{
			Artifacts:   artifactService,
			Collections: collectionRepository,
			Refresh:     refreshService,
			Policy:      config.RootMutationPolicy,
			GetSourceState: func(
				ctx context.Context,
				rootID basespec.RootID,
				sourceID basespec.SourceID,
			) (managedartifact.SourceState, error) {
				result, err := components.GetManagedSourceState(
					ctx,
					rootID,
					sourceID,
				)
				if err != nil {
					return managedartifact.SourceState{}, err
				}
				return managedartifact.SourceState{
					Source:     result.Source,
					Generation: result.Generation,
				}, nil
			},
			PublishPackage: func(
				ctx context.Context,
				rootID basespec.RootID,
				sourceID basespec.SourceID,
				expectedRevision uint64,
				publication source.ManagedPackagePublication,
			) (managedartifact.SourceState, error) {
				result, err := components.PublishManagedPackage(
					ctx,
					rootID,
					sourceID,
					expectedRevision,
					publication,
				)
				if err != nil {
					return managedartifact.SourceState{}, err
				}
				return managedartifact.SourceState{
					Source:     result.Source,
					Generation: result.Generation,
				}, nil
			},
			PublishProtectedPackage: func(
				ctx context.Context,
				rootID basespec.RootID,
				sourceID basespec.SourceID,
				expectedRevision uint64,
				publication source.ManagedPackagePublication,
			) (managedartifact.SourceState, error) {
				result, err := components.PublishProtectedManagedPackage(
					ctx,
					rootID,
					sourceID,
					expectedRevision,
					publication,
				)
				if err != nil {
					return managedartifact.SourceState{}, err
				}
				return managedartifact.SourceState{
					Source:     result.Source,
					Generation: result.Generation,
				}, nil
			},
			RemovePackage:          components.removeManagedArtifactPackage,
			RemoveProtectedPackage: components.removeProtectedManagedArtifactPackage,
		},
	)
	if err != nil {
		_ = components.Close()
		return nil, err
	}
	components.ManagedArtifacts = managedArtifacts
	return components, nil
}

func (c *Components) HasDecoder(id basespec.DecoderID) bool {
	if c == nil {
		return false
	}
	_, exists := c.decoderIDs[id]
	return exists
}

func (c *Components) RootMutationPolicy() protection.RootPolicy {
	if c == nil {
		return nil
	}
	return c.rootMutationPolicy
}

func bindShareableSchemas(
	decoders []discovery.Decoder,
	registry *shareable.Registry,
) error {
	for index, decoder := range decoders {
		binder, supported := decoder.(shareableSchemaBinder)
		if !supported {
			continue
		}
		if err := binder.BindShareableSchemas(registry); err != nil {
			return fmt.Errorf(
				"bind shareable schemas to decoder %d: %w",
				index,
				err,
			)
		}
	}
	return nil
}

// DecoderFingerprint returns the exact decoder capability fingerprint used by
// newly published catalogs. Consumers use it to mark old catalogs stale after
// a decoder implementation or revision changes.
func (c *Components) DecoderFingerprint() (cryptoutil.Digest, error) {
	if c == nil || c.discovery == nil {
		return "", fmt.Errorf(
			"%w: artifact decoder registry is unavailable",
			basespec.ErrClosed,
		)
	}
	return c.discovery.DecoderFingerprint()
}

// GetManagedSourceState returns the current confirmed snapshot generation used
// as the optimistic token for managed package publication and removal. It does
// not expose Source configuration or the private acknowledged-generation
// metadata field.
func (c *Components) GetManagedSourceState(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
) (ManagedPackageResult, error) {
	if c == nil ||
		c.SourceRuntime == nil ||
		c.managedSources == nil {
		return ManagedPackageResult{}, basespec.ErrClosed
	}
	if ctx == nil {
		return ManagedPackageResult{}, fmt.Errorf(
			"%w: managed Source state context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return ManagedPackageResult{}, err
	}
	value, err := c.SourceRuntime.Get(ctx, rootID, sourceID)
	if err != nil {
		return ManagedPackageResult{}, err
	}
	if !c.managedSources.SupportsManagedPackages(value.Kind) {
		return ManagedPackageResult{}, fmt.Errorf(
			"%w: source kind %q is not writable",
			basespec.ErrUnsupported,
			value.Kind,
		)
	}
	generation, err := sourceSnapshotGeneration(
		ctx,
		c.SourceRuntime,
		value,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}
	return ManagedPackageResult{
		Source:     value.Summary(),
		Generation: generation,
	}, nil
}

// PublishManagedPackage publishes a package and advances the Source revision
// only when the resulting snapshot generation changed. The revision advance
// invalidates catalogs that observed the prior generation.
//
// Source-side publication and SQLite metadata publication intentionally remain
// separate operations. If the package write succeeds but the revision advance
// conflicts, the caller receives the conflict and must reload before retrying.
func (c *Components) PublishManagedPackage(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	expectedSourceRevision uint64,
	publication source.ManagedPackagePublication,
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
func (c *Components) PublishProtectedManagedPackage(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	expectedSourceRevision uint64,
	publication source.ManagedPackagePublication,
) (ManagedPackageResult, error) {
	if c == nil || !c.isProtectedRoot(rootID) {
		return ManagedPackageResult{}, fmt.Errorf(
			"%w: Root %q is not a declared protected topology Root",
			basespec.ErrProtected,
			rootID,
		)
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
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
func (c *Components) RemoveManagedPackage(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	expectedSourceRevision uint64,
	address source.ManagedPackageAddress,
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
func (c *Components) RemoveProtectedManagedPackage(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	expectedSourceRevision uint64,
	address source.ManagedPackageAddress,
	expectedGeneration string,
) (ManagedPackageResult, error) {
	if c == nil || !c.isProtectedRoot(rootID) {
		return ManagedPackageResult{}, fmt.Errorf(
			"%w: Root %q is not a declared protected topology Root",
			basespec.ErrProtected,
			rootID,
		)
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
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

func (c *Components) Close() error {
	if c == nil {
		return nil
	}
	var closeErrors []error
	if c.metadata != nil {
		if err := c.metadata.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	return errors.Join(closeErrors...)
}

func (c *Components) publishManagedPackage(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	expectedSourceRevision uint64,
	publication source.ManagedPackagePublication,
	allowProtected bool,
) (ManagedPackageResult, error) {
	if c == nil {
		return ManagedPackageResult{}, basespec.ErrClosed
	}
	if c.isProtectedRoot(rootID) && !allowProtected {
		return ManagedPackageResult{}, fmt.Errorf(
			"%w: managed package publication for protected Root %q requires the protected installer path",
			basespec.ErrProtected,
			rootID,
		)
	}
	if err := protection.RequireMutableRoot(ctx, c.rootMutationPolicy, rootID); err != nil {
		return ManagedPackageResult{}, err
	}
	normalizedPublication, err := source.NormalizeManagedPackagePublication(
		publication,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}
	requestedGeneration := normalizedPublication.ExpectedGeneration
	value, err := c.managedSource(
		ctx,
		rootID,
		sourceID,
		expectedSourceRevision,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}
	publication = normalizedPublication

	beforeGeneration, err := sourceSnapshotGeneration(
		ctx,
		c.SourceRuntime,
		value,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}
	if publication.ExpectedGeneration == "" {
		publication.ExpectedGeneration = beforeGeneration
	}

	generation, err := c.managedSources.PublishPackage(
		ctx,
		value,
		publication,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}

	result := ManagedPackageResult{
		Source:     value.Summary(),
		Generation: generation,
	}
	contentChanged := generation != beforeGeneration ||
		(requestedGeneration != "" &&
			requestedGeneration != beforeGeneration)
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
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	expectedSourceRevision uint64,
	address source.ManagedPackageAddress,
	expectedGeneration string,
	allowProtected bool,
) (ManagedPackageResult, error) {
	if c == nil {
		return ManagedPackageResult{}, basespec.ErrClosed
	}
	if c.isProtectedRoot(rootID) && !allowProtected {
		return ManagedPackageResult{}, fmt.Errorf(
			"%w: managed package removal for protected Root %q requires the protected installer path",
			basespec.ErrProtected,
			rootID,
		)
	}
	if err := protection.RequireMutableRoot(ctx, c.rootMutationPolicy, rootID); err != nil {
		return ManagedPackageResult{}, err
	}
	if err := address.Validate(); err != nil {
		return ManagedPackageResult{}, err
	}
	if err := basespec.ValidateSourceGeneration(expectedGeneration); err != nil {
		return ManagedPackageResult{}, err
	}

	value, err := c.managedSource(
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
		value,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}
	if beforeGeneration != expectedGeneration {
		exists, err := managedPackageExists(
			ctx,
			c.SourceRuntime,
			value,
			address,
		)
		if err != nil {
			return ManagedPackageResult{}, err
		}
		if exists {
			return ManagedPackageResult{}, fmt.Errorf(
				"%w: managed Source changed before package removal",
				basespec.ErrConflict,
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
		value,
		address,
		expectedGeneration,
	); err != nil {
		return ManagedPackageResult{}, err
	}

	generation, err := sourceSnapshotGeneration(ctx, c.SourceRuntime, value)
	if err != nil {
		return ManagedPackageResult{}, err
	}
	if generation == beforeGeneration {
		return ManagedPackageResult{
			Source:     value.Summary(),
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
	runtime source.Runtime,
	value source.Source,
	address source.ManagedPackageAddress,
) (bool, error) {
	snapshot, err := runtime.Open(ctx, value)
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
	if errors.Is(statErr, basespec.ErrNotFound) {
		return false, nil
	}
	if statErr != nil {
		return false, statErr
	}
	if !entry.IsDirectory {
		return false, fmt.Errorf(
			"%w: managed package %q is not a directory",
			basespec.ErrInvalid,
			address,
		)
	}
	return true, nil
}

func (c *Components) isProtectedRoot(
	rootID basespec.RootID,
) bool {
	return c != nil &&
		c.rootMutationPolicy != nil &&
		c.rootMutationPolicy.IsProtectedRoot(rootID)
}

func (c *Components) managedSource(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	expectedSourceRevision uint64,
) (source.Source, error) {
	if c == nil ||
		c.Sources == nil ||
		c.SourceRuntime == nil ||
		c.managedSources == nil {
		return source.Source{}, basespec.ErrClosed
	}
	if ctx == nil {
		return source.Source{}, fmt.Errorf(
			"%w: managed Source context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return source.Source{}, err
	}
	if expectedSourceRevision == 0 {
		return source.Source{}, fmt.Errorf(
			"%w: expected source revision is required",
			basespec.ErrInvalid,
		)
	}
	value, err := c.SourceRuntime.Get(ctx, rootID, sourceID)
	if err != nil {
		return source.Source{}, err
	}
	if value.Revision != expectedSourceRevision {
		return source.Source{}, basespec.ErrConflict
	}
	if !c.managedSources.SupportsManagedPackages(value.Kind) {
		return source.Source{}, fmt.Errorf(
			"%w: source kind %q is not writable",
			basespec.ErrUnsupported,
			value.Kind,
		)
	}
	return value, nil
}

func sourceSnapshotGeneration(
	ctx context.Context,
	runtime source.Runtime,
	value source.Source,
) (string, error) {
	snapshot, err := runtime.Open(ctx, value)
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
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	expectedRevision uint64,
	address source.ManagedPackageAddress,
	expectedGeneration string,
) (managedartifact.SourceState, error) {
	result, err := c.RemoveManagedPackage(
		ctx,
		rootID,
		sourceID,
		expectedRevision,
		address,
		expectedGeneration,
	)
	if err != nil {
		return managedartifact.SourceState{}, err
	}
	return managedartifact.SourceState{
		Source:     result.Source,
		Generation: result.Generation,
	}, nil
}

func (c *Components) removeProtectedManagedArtifactPackage(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	expectedRevision uint64,
	address source.ManagedPackageAddress,
	expectedGeneration string,
) (managedartifact.SourceState, error) {
	result, err := c.RemoveProtectedManagedPackage(
		ctx,
		rootID,
		sourceID,
		expectedRevision,
		address,
		expectedGeneration,
	)
	if err != nil {
		return managedartifact.SourceState{}, err
	}
	return managedartifact.SourceState{
		Source:     result.Source,
		Generation: result.Generation,
	}, nil
}
