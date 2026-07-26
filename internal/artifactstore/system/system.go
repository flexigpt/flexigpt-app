package system

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition/maprepo"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/mapstoreio"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/embedded"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/managed"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/sqlite"
)

type Config struct {
	BaseDirectory             string
	EmbeddedProviders         map[string]fs.FS
	AdditionalSources         []source.Adapter
	Decoders                  []discovery.Decoder
	Clock                     artifactstore.Clock
	IDGenerator               artifactstore.IDGenerator
	FilesystemTraversalPolicy *fsdir.TraversalPolicy
}

type ManagedPackageResult struct {
	Source     source.Summary
	Generation string
}

type Components struct {
	Roots       *root.Service
	Sources     *source.Service
	Collections *collection.Service
	Artifacts   *artifact.Service
	Refresh     *refresh.Service

	CollectionReader collection.Reader
	ArtifactReader   artifact.Reader
	Catalogs         catalog.Reader
	Definitions      definition.Reader
	SourceRuntime    source.Runtime
	discovery        *discovery.Engine

	metadata       *sqlite.Store
	content        *maprepo.Repository
	managedSources *source.Registry
	decoderIDs     map[artifactstore.DecoderID]struct{}
}

func Open(
	ctx context.Context,
	config Config,
) (*Components, error) {
	if config.BaseDirectory == "" {
		return nil, fmt.Errorf(
			"%w: artifact system base directory is empty",
			artifactstore.ErrInvalid,
		)
	}
	if config.Clock == nil {
		config.Clock = artifactstore.SystemClock{}
	}
	if config.IDGenerator == nil {
		config.IDGenerator = artifactstore.UUIDv7Generator{}
	}

	base, err := mapstoreio.PreparePrivateDirectory(
		config.BaseDirectory,
	)
	if err != nil {
		return nil, err
	}

	metadata, err := sqlite.Open(
		ctx,
		filepath.Join(base, "artifact-metadata.sqlite"),
	)
	if err != nil {
		return nil, err
	}

	content, err := maprepo.Open(
		filepath.Join(base, "definitions"),
	)
	if err != nil {
		_ = metadata.Close()
		return nil, err
	}

	filesystemAdapter, err := fsdir.NewWithTraversalPolicy(
		config.FilesystemTraversalPolicy,
	)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}

	managedAdapter, err := managed.New(
		filepath.Join(base, "managed-sources"),
	)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}

	embeddedAdapter, err := embedded.New(config.EmbeddedProviders)
	if err != nil {
		_ = content.Close()
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
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}
	decoderRegistry, err := discovery.NewDecoderRegistry(config.Decoders...)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}

	sourceRepository := metadata.Sources()
	rootRepository := metadata.Roots()
	collectionRepository := metadata.Collections()
	catalogRepository := metadata.Catalogs()
	artifactRepository := metadata.Artifacts()
	definitions, err := definition.NewRootScopedRepository(
		content,
		func(ctx context.Context, rootID artifactstore.RootID) error {
			_, err := rootRepository.Get(ctx, rootID)
			return err
		},
	)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}

	sourceRuntime, err := source.NewRuntime(
		sourceRepository,
		sourceRegistry,
	)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}

	sourceService, err := source.NewService(
		sourceRepository,
		sourceRegistry,
		config.IDGenerator,
		config.Clock,
	)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}
	rootService, err := root.NewService(
		rootRepository,
		config.IDGenerator,
		config.Clock,
	)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}
	collectionService, err := collection.NewService(
		collectionRepository,
		sourceService,
		config.IDGenerator,
		config.Clock,
	)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}
	artifactService, err := artifact.NewService(
		artifactRepository,
		collectionRepository,
		catalogRepository,
		config.IDGenerator,
		config.Clock,
	)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}
	discoveryEngine, err := discovery.NewEngine(
		decoderRegistry,
		config.Clock,
	)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}
	reconciler, err := artifact.NewReconciler(
		config.IDGenerator,
		config.Clock,
	)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}
	decoderIDs := make(map[artifactstore.DecoderID]struct{}, len(config.Decoders))
	for _, decoder := range config.Decoders {
		decoderIDs[decoder.ID()] = struct{}{}
	}

	refreshService, err := refresh.NewService(
		collectionRepository,
		catalogRepository,
		sourceRuntime,
		artifactRepository,
		discoveryEngine,
		definitions,
		reconciler,
		metadata.Publisher(),
		config.Clock,
	)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}

	return &Components{
		Roots:            rootService,
		Sources:          sourceService,
		Collections:      collectionService,
		Artifacts:        artifactService,
		Refresh:          refreshService,
		CollectionReader: collectionRepository,
		ArtifactReader:   artifactRepository,
		Catalogs:         catalogRepository,
		Definitions:      definitions,
		SourceRuntime:    sourceRuntime,
		discovery:        discoveryEngine,
		metadata:         metadata,
		content:          content,
		managedSources:   sourceRegistry,
		decoderIDs:       decoderIDs,
	}, nil
}

func (c *Components) HasDecoder(id artifactstore.DecoderID) bool {
	if c == nil {
		return false
	}
	_, exists := c.decoderIDs[id]
	return exists
}

// DecoderFingerprint returns the exact decoder capability fingerprint used by
// newly published catalogs. Consumers use it to mark old catalogs stale after
// a decoder implementation or revision changes.
func (c *Components) DecoderFingerprint() (artifactstore.Digest, error) {
	if c == nil || c.discovery == nil {
		return "", fmt.Errorf(
			"%w: artifact decoder registry is unavailable",
			artifactstore.ErrClosed,
		)
	}
	return c.discovery.DecoderFingerprint()
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
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
	expectedSourceRevision uint64,
	publication source.ManagedPackagePublication,
) (ManagedPackageResult, error) {
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
	if generation == beforeGeneration {
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

// RemoveManagedPackage removes one complete package and advances the Source
// revision after successful source-side removal.
func (c *Components) RemoveManagedPackage(
	ctx context.Context,
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
	expectedSourceRevision uint64,
	directory artifactstore.Locator,
	expectedGeneration string,
) (ManagedPackageResult, error) {
	value, err := c.managedSource(
		ctx,
		rootID,
		sourceID,
		expectedSourceRevision,
	)
	if err != nil {
		return ManagedPackageResult{}, err
	}
	if err := c.managedSources.RemovePackage(
		ctx,
		value,
		directory,
		expectedGeneration,
	); err != nil {
		return ManagedPackageResult{}, err
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
	generation, err := sourceSnapshotGeneration(ctx, c.SourceRuntime, value)
	if err != nil {
		return ManagedPackageResult{}, err
	}
	return ManagedPackageResult{
		Source:     updated,
		Generation: generation,
	}, nil
}

func (c *Components) Close() error {
	if c == nil {
		return nil
	}
	var closeErrors []error
	if c.content != nil {
		if err := c.content.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	if c.metadata != nil {
		if err := c.metadata.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	return errors.Join(closeErrors...)
}

func (c *Components) managedSource(
	ctx context.Context,
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
	expectedSourceRevision uint64,
) (source.Source, error) {
	if c == nil ||
		c.Sources == nil ||
		c.SourceRuntime == nil ||
		c.managedSources == nil {
		return source.Source{}, artifactstore.ErrClosed
	}
	if ctx == nil {
		return source.Source{}, fmt.Errorf(
			"%w: managed Source context is nil",
			artifactstore.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return source.Source{}, err
	}
	if expectedSourceRevision == 0 {
		return source.Source{}, fmt.Errorf(
			"%w: expected source revision is required",
			artifactstore.ErrInvalid,
		)
	}
	value, err := c.SourceRuntime.Get(ctx, rootID, sourceID)
	if err != nil {
		return source.Source{}, err
	}
	if value.Revision != expectedSourceRevision {
		return source.Source{}, artifactstore.ErrConflict
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
