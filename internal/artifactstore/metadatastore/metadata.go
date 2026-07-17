package metadatastore

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	_ "github.com/glebarez/go-sqlite"
)

const (
	metadataSchemaVersion     = 1
	metadataSchemaFingerprint = "artifactstore.metadata.schema-1.2026-05-workspace-root-catalog"
)

const (
	readMetadataSchemaVersionSQL = `PRAGMA user_version;`
	setMetadataSchemaVersionSQL  = `PRAGMA user_version = 1;`

	createMetadataSchemaIdentitySQL = `CREATE TABLE artifact_store_schema (
		schema_version INTEGER PRIMARY KEY CHECK (schema_version = 1),
		schema_fingerprint TEXT NOT NULL
	);`
	insertMetadataSchemaIdentitySQL = `INSERT INTO artifact_store_schema (
		schema_version, schema_fingerprint
	) VALUES (1, 'artifactstore.metadata.schema-1.2026-05-workspace-root-catalog');`
	readMetadataSchemaIdentitySQL = `SELECT schema_fingerprint
		FROM artifact_store_schema
		WHERE schema_version = 1;`

	createArtifactRootsSQL = `CREATE TABLE artifact_roots (
		root_id TEXT PRIMARY KEY,
		kind TEXT NOT NULL,
		display_name TEXT NOT NULL,
		description TEXT NOT NULL,
		enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
		mount_revision INTEGER NOT NULL CHECK (mount_revision BETWEEN 1 AND 9223372036854775807),
		data_schema_id TEXT NOT NULL,
		data_json TEXT NOT NULL CHECK (json_valid(data_json) AND json_type(data_json) = 'object'),
		created_at TEXT NOT NULL,
		modified_at TEXT NOT NULL,
		soft_deleted_at TEXT,
		CHECK (modified_at >= created_at),
		CHECK (
			soft_deleted_at IS NULL OR
			(enabled = 0 AND soft_deleted_at >= created_at)
		)
	);`
	createArtifactSourcesSQL = `CREATE TABLE artifact_sources (
		source_id TEXT PRIMARY KEY,
		kind TEXT NOT NULL,
		display_name TEXT NOT NULL,
		enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
		config_schema_id TEXT NOT NULL,
		config_json TEXT NOT NULL CHECK (json_valid(config_json) AND json_type(config_json) = 'object'),
		last_observed_generation TEXT,
		last_scanned_at TEXT,
		observation_revision INTEGER NOT NULL CHECK (
			observation_revision BETWEEN 0 AND 9223372036854775807
		),
		diagnostics_json TEXT NOT NULL CHECK (
			json_valid(diagnostics_json) AND json_type(diagnostics_json) = 'array'
		),
		created_at TEXT NOT NULL,
		modified_at TEXT NOT NULL,
		CHECK (modified_at >= created_at),
		CHECK (last_scanned_at IS NULL OR last_scanned_at >= created_at)
	);`
	createRootSourceAttachmentsSQL = `CREATE TABLE root_source_attachments (
		root_id TEXT NOT NULL REFERENCES artifact_roots(root_id) ON DELETE RESTRICT,
		source_id TEXT NOT NULL REFERENCES artifact_sources(source_id) ON DELETE RESTRICT,
		role TEXT NOT NULL,
		priority INTEGER NOT NULL,
		enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
		data_schema_id TEXT NOT NULL,
		data_json TEXT NOT NULL CHECK (json_valid(data_json) AND json_type(data_json) = 'object'),
		created_at TEXT NOT NULL,
		modified_at TEXT NOT NULL,
		PRIMARY KEY (root_id, source_id),
		CHECK (modified_at >= created_at)
	);`
	createArtifactPackagesSQL = `CREATE TABLE artifact_packages (
		source_id TEXT NOT NULL REFERENCES artifact_sources(source_id) ON DELETE RESTRICT,
		manifest_locator TEXT NOT NULL,
		name TEXT NOT NULL,
		version TEXT NOT NULL,
		display_name TEXT NOT NULL,
		description TEXT NOT NULL,
		current_manifest_digest TEXT,
		state TEXT NOT NULL CHECK (state IN ('valid', 'invalid', 'missing')),
		diagnostics_json TEXT NOT NULL CHECK (
			json_valid(diagnostics_json) AND json_type(diagnostics_json) = 'array'
		),
		first_seen_at TEXT NOT NULL,
		last_seen_at TEXT NOT NULL,
		PRIMARY KEY (source_id, manifest_locator),
		CHECK (last_seen_at >= first_seen_at),
		CHECK (
			state <> 'valid' OR
			(name <> '' AND version <> '' AND current_manifest_digest IS NOT NULL)
		)
	);`
	createCatalogResourcesSQL = `CREATE TABLE catalog_resources (
		source_id TEXT NOT NULL REFERENCES artifact_sources(source_id) ON DELETE RESTRICT,
		locator TEXT NOT NULL,
		subresource_locator TEXT NOT NULL,
		package_manifest_locator TEXT NOT NULL,
		kind TEXT NOT NULL,
		logical_name TEXT NOT NULL,
		logical_version TEXT NOT NULL,
		current_definition_digest TEXT,
		source_content_digest TEXT,
		frontend_id TEXT NOT NULL,
		state TEXT NOT NULL CHECK (state IN ('valid', 'invalid', 'missing')),
		first_seen_at TEXT NOT NULL,
		last_seen_at TEXT NOT NULL,
		diagnostics_json TEXT NOT NULL CHECK (
			json_valid(diagnostics_json) AND json_type(diagnostics_json) = 'array'
		),
		PRIMARY KEY (source_id, locator, subresource_locator),
		CHECK (last_seen_at >= first_seen_at),
		CHECK (
			state <> 'valid' OR
			(kind <> '' AND logical_name <> '' AND frontend_id <> '' AND
			 current_definition_digest IS NOT NULL AND source_content_digest IS NOT NULL)
		)
	);`
	createCatalogResourceRevisionsSQL = `CREATE TABLE catalog_resource_revisions (
		source_id TEXT NOT NULL REFERENCES artifact_sources(source_id) ON DELETE RESTRICT,
		locator TEXT NOT NULL,
		subresource_locator TEXT NOT NULL,
		definition_digest TEXT NOT NULL,
		source_content_digest TEXT NOT NULL,
		kind TEXT NOT NULL,
		frontend_id TEXT NOT NULL,
		first_seen_at TEXT NOT NULL,
		last_seen_at TEXT NOT NULL,
		PRIMARY KEY (source_id, locator, subresource_locator, definition_digest),
		CHECK (last_seen_at >= first_seen_at)
	);`
	createArtifactCollectionsSQL = `CREATE TABLE artifact_collections (
		collection_id TEXT PRIMARY KEY,
		root_id TEXT NOT NULL REFERENCES artifact_roots(root_id) ON DELETE RESTRICT,
		kind TEXT NOT NULL,
		slug TEXT NOT NULL,
		display_name TEXT NOT NULL,
		description TEXT NOT NULL,
		enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
		data_schema_id TEXT NOT NULL,
		data_json TEXT NOT NULL CHECK (json_valid(data_json) AND json_type(data_json) = 'object'),
		created_at TEXT NOT NULL,
		modified_at TEXT NOT NULL,
		soft_deleted_at TEXT,
		UNIQUE (root_id, slug),
		UNIQUE (collection_id, root_id),
		CHECK (modified_at >= created_at),
		CHECK (
			soft_deleted_at IS NULL OR
			(enabled = 0 AND soft_deleted_at >= created_at)
		)
	);`
	createArtifactRecordsSQL = `CREATE TABLE artifact_records (
		record_id TEXT PRIMARY KEY,
		root_id TEXT NOT NULL REFERENCES artifact_roots(root_id) ON DELETE RESTRICT,
		collection_id TEXT,
		kind TEXT NOT NULL,
		name TEXT NOT NULL,
		version TEXT NOT NULL,
		source_id TEXT NOT NULL REFERENCES artifact_sources(source_id) ON DELETE RESTRICT,
		locator TEXT NOT NULL,
		subresource_locator TEXT NOT NULL,
		record_mode TEXT NOT NULL,
		tracking_mode TEXT NOT NULL,
		pinned_definition_digest TEXT,
		last_resolved_definition_digest TEXT,
		enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
		data_schema_id TEXT NOT NULL,
		data_json TEXT NOT NULL CHECK (json_valid(data_json) AND json_type(data_json) = 'object'),
		state TEXT NOT NULL CHECK (
			state IN ('available', 'stale', 'missing', 'invalid', 'incompatible')
		),
		diagnostics_json TEXT NOT NULL CHECK (
			json_valid(diagnostics_json) AND json_type(diagnostics_json) = 'array'
		),
		created_at TEXT NOT NULL,
		modified_at TEXT NOT NULL,
		UNIQUE (root_id, source_id, locator, subresource_locator, kind),
		UNIQUE (record_id, root_id),
		FOREIGN KEY (collection_id, root_id)
			REFERENCES artifact_collections(collection_id, root_id)
			ON DELETE RESTRICT,
		CHECK (record_mode IN ('linked', 'captured', 'forked', 'app-local', 'embedded-overlay')),
		CHECK (tracking_mode IN ('follow-source', 'pin-digest', 'manual-refresh')),
		CHECK (
			(tracking_mode = 'pin-digest' AND pinned_definition_digest IS NOT NULL) OR
			(tracking_mode <> 'pin-digest' AND pinned_definition_digest IS NULL)
		),
		CHECK (modified_at >= created_at)
	);`
	createRootCatalogGenerationsSQL = `CREATE TABLE root_catalog_generations (
		root_id TEXT NOT NULL REFERENCES artifact_roots(root_id) ON DELETE RESTRICT,
		generation INTEGER NOT NULL CHECK (generation > 0),
		root_revision INTEGER NOT NULL CHECK (root_revision BETWEEN 1 AND 9223372036854775807),
		source_versions_json TEXT NOT NULL CHECK (
			json_valid(source_versions_json) AND json_type(source_versions_json) = 'object'
		),
		scan_plan_digest TEXT NOT NULL,
		catalog_digest TEXT NOT NULL,
		created_at TEXT NOT NULL,
		diagnostics_json TEXT NOT NULL CHECK (
			json_valid(diagnostics_json) AND json_type(diagnostics_json) = 'array'
		),
		PRIMARY KEY (root_id, generation)
	);`
	createRootCatalogResourceSnapshotsSQL = `CREATE TABLE root_catalog_resource_snapshots (
		root_id TEXT NOT NULL,
		generation INTEGER NOT NULL,
		source_id TEXT NOT NULL REFERENCES artifact_sources(source_id) ON DELETE RESTRICT,
		locator TEXT NOT NULL,
		subresource_locator TEXT NOT NULL,
		package_manifest_locator TEXT NOT NULL,
		kind TEXT NOT NULL,
		logical_name TEXT NOT NULL,
		logical_version TEXT NOT NULL,
		current_definition_digest TEXT,
		source_content_digest TEXT,
		frontend_id TEXT NOT NULL,
		state TEXT NOT NULL CHECK (state IN ('valid', 'invalid', 'missing')),
		first_seen_at TEXT NOT NULL,
		last_seen_at TEXT NOT NULL,
		diagnostics_json TEXT NOT NULL CHECK (
			json_valid(diagnostics_json) AND json_type(diagnostics_json) = 'array'
		),
		PRIMARY KEY (
			root_id,
			generation,
			source_id,
			locator,
			subresource_locator
		),
		FOREIGN KEY (root_id, generation)
			REFERENCES root_catalog_generations(root_id, generation)
			ON DELETE CASCADE,
		CHECK (last_seen_at >= first_seen_at)
	);`
	createRootCatalogGenerationCountersSQL = `CREATE TABLE root_catalog_generation_counters (
		root_id TEXT PRIMARY KEY REFERENCES artifact_roots(root_id) ON DELETE RESTRICT,
		generation INTEGER NOT NULL CHECK (generation >= 0)
	);`
	createArtifactTransferProvenanceSQL = `CREATE TABLE artifact_transfer_provenance (
		provenance_id TEXT PRIMARY KEY,
		target_record_id TEXT NOT NULL REFERENCES artifact_records(record_id) ON DELETE CASCADE,
		operation TEXT NOT NULL,
		origin_record_id TEXT,
		origin_resource_json TEXT CHECK (
			origin_resource_json IS NULL OR
			(json_valid(origin_resource_json) AND json_type(origin_resource_json) = 'object')
		),
		origin_definition_digest TEXT NOT NULL,
		created_at TEXT NOT NULL,
		CHECK (operation IN ('import', 'capture', 'fork'))
	);`
	createArtifactDependenciesSQL = `CREATE TABLE artifact_dependencies (
		root_id TEXT NOT NULL REFERENCES artifact_roots(root_id) ON DELETE RESTRICT,
		record_id TEXT NOT NULL,
		catalog_generation INTEGER NOT NULL CHECK (catalog_generation > 0),
		root_definition_digest TEXT NOT NULL,
		definition_digest TEXT NOT NULL,
		selector_index INTEGER NOT NULL,
		selector_json TEXT NOT NULL CHECK (
			json_valid(selector_json) AND json_type(selector_json) = 'object'
		),
		state TEXT NOT NULL CHECK (state IN ('resolved', 'missing', 'ambiguous')),
		candidates_json TEXT NOT NULL CHECK (
			json_valid(candidates_json) AND json_type(candidates_json) = 'array'
		),
		diagnostics_json TEXT NOT NULL CHECK (
			json_valid(diagnostics_json) AND json_type(diagnostics_json) = 'array'
		),
		modified_at TEXT NOT NULL,
		PRIMARY KEY (
			root_id,
			record_id,
			catalog_generation,
			root_definition_digest,
			definition_digest,
			selector_index
		),
		FOREIGN KEY (record_id, root_id)
			REFERENCES artifact_records(record_id, root_id)
			ON DELETE CASCADE,
		FOREIGN KEY (root_id, catalog_generation)
			REFERENCES root_catalog_generations(root_id, generation)
			ON DELETE CASCADE
	);`
	createActiveAttachmentRootTriggerSQL = `CREATE TRIGGER trg_root_source_attachments_active_root
		BEFORE INSERT ON root_source_attachments
		WHEN NOT EXISTS (
			SELECT 1 FROM artifact_roots
			 WHERE root_id = NEW.root_id AND soft_deleted_at IS NULL
		)
		BEGIN
			SELECT RAISE(ABORT, 'artifactstore conflict: attachment root is not active');
		END;`
	createAttachmentRootRevisionInsertTriggerSQL = `CREATE TRIGGER trg_root_source_attachments_revision_insert
		AFTER INSERT ON root_source_attachments
		BEGIN
			UPDATE artifact_roots SET mount_revision = mount_revision + 1 WHERE root_id = NEW.root_id;
		END;`
	createAttachmentRootRevisionUpdateTriggerSQL = `CREATE TRIGGER trg_root_source_attachments_revision_update
		AFTER UPDATE ON root_source_attachments
		BEGIN
			UPDATE artifact_roots SET mount_revision = mount_revision + 1 WHERE root_id = NEW.root_id;
		END;`
	createAttachmentRootRevisionDeleteTriggerSQL = `CREATE TRIGGER trg_root_source_attachments_revision_delete
		AFTER DELETE ON root_source_attachments
		BEGIN
			UPDATE artifact_roots SET mount_revision = mount_revision + 1 WHERE root_id = OLD.root_id;
		END;`
	createRecordCollectionInsertTriggerSQL = `CREATE TRIGGER trg_artifact_records_collection_insert
		BEFORE INSERT ON artifact_records
		WHEN NEW.collection_id IS NOT NULL
		 AND NOT EXISTS (
			SELECT 1
			  FROM artifact_collections
			 WHERE collection_id = NEW.collection_id
			   AND root_id = NEW.root_id
			   AND soft_deleted_at IS NULL
		 )
		BEGIN
			SELECT RAISE(ABORT, 'artifactstore conflict: invalid active record collection');
		END;`
	createRecordCollectionUpdateTriggerSQL = `CREATE TRIGGER trg_artifact_records_collection_update
		BEFORE UPDATE OF collection_id ON artifact_records
		WHEN NEW.collection_id IS NOT NULL
		 AND NOT EXISTS (
			SELECT 1
			  FROM artifact_collections
			 WHERE collection_id = NEW.collection_id
			   AND root_id = NEW.root_id
			   AND soft_deleted_at IS NULL
		 )
		BEGIN
			SELECT RAISE(ABORT, 'artifactstore conflict: invalid active record collection');
		END;`
	createNonemptyCollectionDeleteTriggerSQL = `CREATE TRIGGER trg_artifact_collections_nonempty_delete
		BEFORE UPDATE OF soft_deleted_at ON artifact_collections
		WHEN OLD.soft_deleted_at IS NULL
		 AND NEW.soft_deleted_at IS NOT NULL
		 AND EXISTS (
			SELECT 1
			  FROM artifact_records
			 WHERE collection_id = NEW.collection_id
		 )
		BEGIN
			SELECT RAISE(ABORT, 'artifactstore conflict: collection still contains records');
		END;`
	createRecordAttachmentInsertTriggerSQL = `CREATE TRIGGER trg_artifact_records_attachment_insert
		BEFORE INSERT ON artifact_records
		WHEN NOT EXISTS (
			SELECT 1
			  FROM root_source_attachments a
			  JOIN artifact_sources s ON s.source_id = a.source_id
			  JOIN artifact_roots r ON r.root_id = a.root_id
			 WHERE a.root_id = NEW.root_id
			   AND a.source_id = NEW.source_id
			   AND a.enabled = 1
			   AND s.enabled = 1
			   AND r.enabled = 1
			   AND r.soft_deleted_at IS NULL
		 )
		BEGIN
			SELECT RAISE(ABORT, 'artifactstore conflict: source is not actively attached to record root');
		END;`
	createActiveCollectionRootTriggerSQL = `CREATE TRIGGER trg_artifact_collections_active_root
		BEFORE INSERT ON artifact_collections
		WHEN NOT EXISTS (
			SELECT 1 FROM artifact_roots
			 WHERE root_id = NEW.root_id AND soft_deleted_at IS NULL
		)
		BEGIN
			SELECT RAISE(ABORT, 'artifactstore conflict: collection root is not active');
		END;`
	createRootSourceAttachmentSourceIndexSQL = `CREATE INDEX idx_root_source_attachments_source
		ON root_source_attachments (source_id);`
	createArtifactPackagesSourceIndexSQL = `CREATE INDEX idx_artifact_packages_source
		ON artifact_packages (source_id, state);`
	createCatalogResourcesSourceStateIndexSQL = `CREATE INDEX idx_catalog_resources_source_state
		ON catalog_resources (source_id, state);`
	createCatalogResourcesKindNameIndexSQL = `CREATE INDEX idx_catalog_resources_kind_name
		ON catalog_resources (kind, logical_name);`
	createCatalogRevisionsResourceIndexSQL = `CREATE INDEX idx_catalog_resource_revisions_resource
		ON catalog_resource_revisions (source_id, locator, subresource_locator, last_seen_at DESC);`
	createCollectionsRootIndexSQL = `CREATE INDEX idx_artifact_collections_root
		ON artifact_collections (root_id, modified_at DESC);`
	createRecordsRootIndexSQL = `CREATE INDEX idx_artifact_records_root
		ON artifact_records (root_id, modified_at DESC);`
	createRecordsCollectionIndexSQL = `CREATE INDEX idx_artifact_records_collection
		ON artifact_records (collection_id);`
	createRootGenerationsRootIndexSQL = `CREATE INDEX idx_root_catalog_generations_root
		ON root_catalog_generations (root_id, generation DESC);`
	createDependenciesRecordIndexSQL = `CREATE INDEX idx_artifact_dependencies_record
		ON artifact_dependencies (record_id, catalog_generation, root_definition_digest, definition_digest, selector_index);`
)

type MetadataStore struct {
	db *sql.DB
}

var metadataSchemaStatements = []string{
	createMetadataSchemaIdentitySQL,
	insertMetadataSchemaIdentitySQL,
	createArtifactRootsSQL,
	createArtifactSourcesSQL,
	createRootSourceAttachmentsSQL,
	createArtifactPackagesSQL,
	createCatalogResourcesSQL,
	createCatalogResourceRevisionsSQL,
	createArtifactCollectionsSQL,
	createArtifactRecordsSQL,
	createRootCatalogGenerationsSQL,
	createRootCatalogResourceSnapshotsSQL,
	createRootCatalogGenerationCountersSQL,
	createArtifactTransferProvenanceSQL,
	createArtifactDependenciesSQL,
	createActiveAttachmentRootTriggerSQL,
	createAttachmentRootRevisionInsertTriggerSQL,
	createAttachmentRootRevisionUpdateTriggerSQL,
	createAttachmentRootRevisionDeleteTriggerSQL,
	createRecordCollectionInsertTriggerSQL,
	createRecordCollectionUpdateTriggerSQL,
	createNonemptyCollectionDeleteTriggerSQL,
	createRecordAttachmentInsertTriggerSQL,
	createActiveCollectionRootTriggerSQL,
	createRootSourceAttachmentSourceIndexSQL,
	createArtifactPackagesSourceIndexSQL,
	createCatalogResourcesSourceStateIndexSQL,
	createCatalogResourcesKindNameIndexSQL,
	createCatalogRevisionsResourceIndexSQL,
	createCollectionsRootIndexSQL,
	createRecordsRootIndexSQL,
	createRecordsCollectionIndexSQL,
	createRootGenerationsRootIndexSQL,
	createDependenciesRecordIndexSQL,
}

func OpenMetadataStore(ctx context.Context, path string) (*MetadataStore, error) {
	db, err := sql.Open(
		"sqlite",
		sqliteDataSourceName(path),
	)
	if err != nil {
		return nil, fmt.Errorf("open artifact metadata database: %w", err)
	}
	db.SetMaxOpenConns(4)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping artifact metadata database: %w", err)
	}
	if err := initializeMetadataSchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &MetadataStore{db: db}, nil
}

func (s *MetadataStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func sqliteDataSourceName(path string) string {
	normalized := filepath.ToSlash(path)
	if filepath.VolumeName(path) != "" && !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	value := &url.URL{Scheme: "file", Path: normalized}
	query := value.Query()
	query.Set("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "busy_timeout(5000)")
	value.RawQuery = query.Encode()
	return value.String()
}

func initializeMetadataSchema(ctx context.Context, db *sql.DB) error {
	var currentVersion int
	if err := db.QueryRowContext(ctx, readMetadataSchemaVersionSQL).Scan(&currentVersion); err != nil {
		return fmt.Errorf("read artifact metadata schema version: %w", err)
	}
	if currentVersion == metadataSchemaVersion {
		return verifyMetadataSchemaIdentity(ctx, db)
	}
	if currentVersion != 0 {
		return fmt.Errorf(
			"artifact metadata schema version %d is unsupported; this new store requires version %d",
			currentVersion,
			metadataSchemaVersion,
		)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin artifact metadata schema initialization: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, statement := range metadataSchemaStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("initialize artifact metadata schema: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, setMetadataSchemaVersionSQL); err != nil {
		return fmt.Errorf("set artifact metadata schema version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit artifact metadata schema initialization: %w", err)
	}
	return verifyMetadataSchemaIdentity(ctx, db)
}

func verifyMetadataSchemaIdentity(ctx context.Context, db *sql.DB) error {
	var fingerprint string
	if err := db.QueryRowContext(ctx, readMetadataSchemaIdentitySQL).Scan(&fingerprint); err != nil {
		return fmt.Errorf(
			"artifact metadata schema 1 is not the current schema 1; recreate the new store: %w",
			err,
		)
	}
	if fingerprint != metadataSchemaFingerprint {
		return fmt.Errorf(
			"artifact metadata schema 1 fingerprint %q is unsupported; recreate the new store",
			fingerprint,
		)
	}
	return nil
}
