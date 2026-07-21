package system

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition/fsrepo"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/embedded"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/sqlite"
)

type Config struct {
	BaseDirectory     string
	EmbeddedProviders map[string]fs.FS
	AdditionalSources []source.Adapter
	Decoders          []discovery.Decoder
	Clock             artifactstore.Clock
	IDGenerator       artifactstore.IDGenerator
}

type Components struct {
	Sources *source.Service
	Catalog *catalog.Service
	Records *record.Service
	Refresh *refresh.Service

	Definitions       definition.Repository
	SourceRepository  source.Repository
	CatalogRepository catalog.Repository
	RecordRepository  record.Repository

	SourceRegistry  *source.Registry
	DecoderRegistry *discovery.DecoderRegistry

	metadata *sqlite.Store
	content  *fsrepo.Repository
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

	base := filepath.Clean(config.BaseDirectory)
	if err := os.MkdirAll(base, 0o700); err != nil {
		return nil, err
	}

	metadata, err := sqlite.Open(
		ctx,
		filepath.Join(base, "artifact-metadata.sqlite"),
	)
	if err != nil {
		return nil, err
	}

	content, err := fsrepo.Open(
		filepath.Join(base, "definitions"),
	)
	if err != nil {
		_ = metadata.Close()
		return nil, err
	}

	embeddedAdapter, err := embedded.New(config.EmbeddedProviders)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}
	sourceAdapters := make([]source.Adapter, 0, 2)
	sourceAdapters = append(sourceAdapters, fsdir.New(), embeddedAdapter)
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
	catalogRepository := metadata.Catalog()
	recordRepository := metadata.Records()

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
	catalogService, err := catalog.NewService(
		catalogRepository,
		config.IDGenerator,
		config.Clock,
	)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}
	recordService, err := record.NewService(
		recordRepository,
		content,
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
	reconciler, err := record.NewReconciler(
		config.IDGenerator,
		config.Clock,
	)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}
	refreshService, err := refresh.NewService(
		catalogRepository,
		sourceRepository,
		recordRepository,
		sourceRegistry,
		discoveryEngine,
		content,
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
		Sources: sourceService,
		Catalog: catalogService,
		Records: recordService,
		Refresh: refreshService,

		Definitions:       content,
		SourceRepository:  sourceRepository,
		CatalogRepository: catalogRepository,
		RecordRepository:  recordRepository,

		SourceRegistry:  sourceRegistry,
		DecoderRegistry: decoderRegistry,
		metadata:        metadata,
		content:         content,
	}, nil
}

func (c *Components) Close() error {
	if c == nil {
		return nil
	}
	var first error
	if c.content != nil {
		if err := c.content.Close(); err != nil {
			first = err
		}
	}
	if c.metadata != nil {
		if err := c.metadata.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
