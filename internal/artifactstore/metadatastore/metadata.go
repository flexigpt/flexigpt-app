package metadatastore

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/glebarez/go-sqlite"
)

const metadataSchemaVersion = 3

var metadataSchemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS artifact_roots (
		root_id TEXT PRIMARY KEY,
		kind TEXT NOT NULL,
		display_name TEXT NOT NULL,
		description TEXT NOT NULL,
		enabled INTEGER NOT NULL,
		data_schema_id TEXT NOT NULL,
		data_json BLOB NOT NULL,
		created_at TEXT NOT NULL,
		modified_at TEXT NOT NULL,
		soft_deleted_at TEXT
	);`,
	`CREATE TABLE IF NOT EXISTS artifact_sources (
		source_id TEXT PRIMARY KEY,
		kind TEXT NOT NULL,
		display_name TEXT NOT NULL,
		enabled INTEGER NOT NULL,
		config_schema_id TEXT NOT NULL,
		config_json BLOB NOT NULL,
		last_observed_generation TEXT,
		last_scanned_at TEXT,
		diagnostics_json BLOB NOT NULL,
		created_at TEXT NOT NULL,
		modified_at TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS root_source_attachments (
		root_id TEXT NOT NULL REFERENCES artifact_roots(root_id) ON DELETE RESTRICT,
		source_id TEXT NOT NULL REFERENCES artifact_sources(source_id) ON DELETE RESTRICT,
		role TEXT NOT NULL,
		priority INTEGER NOT NULL,
		enabled INTEGER NOT NULL,
		data_schema_id TEXT NOT NULL,
		data_json BLOB NOT NULL,
		created_at TEXT NOT NULL,
		modified_at TEXT NOT NULL,
		PRIMARY KEY (root_id, source_id)
	);`,
	`CREATE TABLE IF NOT EXISTS artifact_packages (
		source_id TEXT NOT NULL REFERENCES artifact_sources(source_id) ON DELETE RESTRICT,
		manifest_locator TEXT NOT NULL,
		name TEXT NOT NULL,
		version TEXT NOT NULL,
		display_name TEXT NOT NULL,
		description TEXT NOT NULL,
		current_manifest_digest TEXT,
		state TEXT NOT NULL,
		diagnostics_json BLOB NOT NULL,
		first_seen_at TEXT NOT NULL,
		last_seen_at TEXT NOT NULL,
		PRIMARY KEY (source_id, manifest_locator)
	);`,
	`CREATE TABLE IF NOT EXISTS catalog_resources (
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
		state TEXT NOT NULL,
		first_seen_at TEXT NOT NULL,
		last_seen_at TEXT NOT NULL,
		diagnostics_json BLOB NOT NULL,
		PRIMARY KEY (source_id, locator, subresource_locator)
	);`,
	`CREATE TABLE IF NOT EXISTS catalog_resource_revisions (
		source_id TEXT NOT NULL REFERENCES artifact_sources(source_id) ON DELETE RESTRICT,
		locator TEXT NOT NULL,
		subresource_locator TEXT NOT NULL,
		definition_digest TEXT NOT NULL,
		source_content_digest TEXT NOT NULL,
		kind TEXT NOT NULL,
		frontend_id TEXT NOT NULL,
		first_seen_at TEXT NOT NULL,
		last_seen_at TEXT NOT NULL,
		PRIMARY KEY (source_id, locator, subresource_locator, definition_digest)
	);`,
	`CREATE TABLE IF NOT EXISTS artifact_collections (
		collection_id TEXT PRIMARY KEY,
		root_id TEXT NOT NULL REFERENCES artifact_roots(root_id) ON DELETE RESTRICT,
		kind TEXT NOT NULL,
		slug TEXT NOT NULL,
		display_name TEXT NOT NULL,
		description TEXT NOT NULL,
		enabled INTEGER NOT NULL,
		data_schema_id TEXT NOT NULL,
		data_json BLOB NOT NULL,
		created_at TEXT NOT NULL,
		modified_at TEXT NOT NULL,
		soft_deleted_at TEXT,
		UNIQUE (root_id, slug)
	);`,
	`CREATE TABLE IF NOT EXISTS artifact_records (
		record_id TEXT PRIMARY KEY,
		root_id TEXT NOT NULL REFERENCES artifact_roots(root_id) ON DELETE RESTRICT,
		collection_id TEXT REFERENCES artifact_collections(collection_id) ON DELETE RESTRICT,
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
		enabled INTEGER NOT NULL,
		data_schema_id TEXT NOT NULL,
		data_json BLOB NOT NULL,
		state TEXT NOT NULL,
		diagnostics_json BLOB NOT NULL,
		created_at TEXT NOT NULL,
		modified_at TEXT NOT NULL,
		UNIQUE (root_id, source_id, locator, subresource_locator, kind)
	);`,
	`CREATE TABLE IF NOT EXISTS root_catalog_generations (
		root_id TEXT NOT NULL REFERENCES artifact_roots(root_id) ON DELETE RESTRICT,
		generation INTEGER NOT NULL,
		source_generations_json BLOB NOT NULL,
		scan_plan_digest TEXT NOT NULL,
		catalog_digest TEXT NOT NULL,
		created_at TEXT NOT NULL,
		diagnostics_json BLOB NOT NULL,
		PRIMARY KEY (root_id, generation)
	);`,
	`CREATE TABLE IF NOT EXISTS artifact_transfer_provenance (
		provenance_id TEXT PRIMARY KEY,
		target_record_id TEXT NOT NULL REFERENCES artifact_records(record_id) ON DELETE CASCADE,
		operation TEXT NOT NULL,
		origin_record_id TEXT,
		origin_resource_json BLOB,
		origin_definition_digest TEXT NOT NULL,
		created_at TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS artifact_dependencies (
		root_id TEXT NOT NULL REFERENCES artifact_roots(root_id) ON DELETE RESTRICT,
		record_id TEXT NOT NULL REFERENCES artifact_records(record_id) ON DELETE CASCADE,
		selector_index INTEGER NOT NULL,
		state TEXT NOT NULL,
		candidates_json BLOB NOT NULL,
		diagnostics_json BLOB NOT NULL,
		modified_at TEXT NOT NULL,
		PRIMARY KEY (root_id, record_id, selector_index)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_root_source_attachments_source ON root_source_attachments (source_id);`,
	`CREATE INDEX IF NOT EXISTS idx_artifact_packages_source ON artifact_packages (source_id, state);`,
	`CREATE INDEX IF NOT EXISTS idx_catalog_resources_source_state ON catalog_resources (source_id, state);`,
	`CREATE INDEX IF NOT EXISTS idx_catalog_resources_kind_name ON catalog_resources (kind, logical_name);`,
	`CREATE INDEX IF NOT EXISTS idx_catalog_resource_revisions_resource ON catalog_resource_revisions (source_id, locator, subresource_locator, last_seen_at DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_artifact_collections_root ON artifact_collections (root_id, modified_at DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_artifact_records_root ON artifact_records (root_id, modified_at DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_artifact_records_collection ON artifact_records (collection_id);`,
	`CREATE INDEX IF NOT EXISTS idx_root_catalog_generations_root ON root_catalog_generations (root_id, generation DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_artifact_dependencies_record ON artifact_dependencies (record_id, selector_index);`,
}

var metadataSchemaV3Statements = []string{
	`CREATE TABLE IF NOT EXISTS root_catalog_generation_counters (
		root_id TEXT PRIMARY KEY REFERENCES artifact_roots(root_id) ON DELETE RESTRICT,
		generation INTEGER NOT NULL CHECK (generation >= 0)
	);`,
	`INSERT INTO root_catalog_generation_counters (root_id, generation)
		SELECT root_id, MAX(generation)
		  FROM root_catalog_generations
		 GROUP BY root_id
		ON CONFLICT (root_id) DO UPDATE SET
			generation = MAX(root_catalog_generation_counters.generation, excluded.generation);`,
	`CREATE TRIGGER IF NOT EXISTS trg_artifact_records_collection_insert
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
			SELECT RAISE(ABORT, 'foreign key constraint failed: invalid active record collection');
		END;`,
	`CREATE TRIGGER IF NOT EXISTS trg_artifact_records_collection_update
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
			SELECT RAISE(ABORT, 'foreign key constraint failed: invalid active record collection');
		END;`,
	`CREATE TRIGGER IF NOT EXISTS trg_artifact_collections_nonempty_delete
		BEFORE UPDATE OF soft_deleted_at ON artifact_collections
		WHEN OLD.soft_deleted_at IS NULL
		 AND NEW.soft_deleted_at IS NOT NULL
		 AND EXISTS (
			SELECT 1
			  FROM artifact_records
			 WHERE collection_id = NEW.collection_id
		 )
		BEGIN
			SELECT RAISE(ABORT, 'foreign key constraint failed: collection still contains records');
		END;`,
	`CREATE TRIGGER IF NOT EXISTS trg_artifact_records_attachment_insert
		BEFORE INSERT ON artifact_records
		WHEN NOT EXISTS (
			SELECT 1
			  FROM root_source_attachments
			 WHERE root_id = NEW.root_id
			   AND source_id = NEW.source_id
		 )
		BEGIN
			SELECT RAISE(ABORT, 'foreign key constraint failed: source is not attached to record root');
		END;`,
}

type MetadataStore struct {
	db *sql.DB
}

type metadataMigration struct {
	Version    int
	Statements []string
}

var metadataMigrations = []metadataMigration{
	{
		Version:    2,
		Statements: metadataSchemaStatements,
	},
	{
		Version:    3,
		Statements: metadataSchemaV3Statements,
	},
}

func OpenMetadataStore(ctx context.Context, path string) (*MetadataStore, error) {
	db, err := sql.Open(
		"sqlite",
		path+"?busy_timeout=5000&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)",
	)
	if err != nil {
		return nil, fmt.Errorf("open artifact metadata database: %w", err)
	}
	db.SetMaxOpenConns(4)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping artifact metadata database: %w", err)
	}
	if err := migrateMetadata(ctx, db); err != nil {
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

func migrateMetadata(ctx context.Context, db *sql.DB) error {
	var currentVersion int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version;").Scan(&currentVersion); err != nil {
		return fmt.Errorf("read artifact metadata schema version: %w", err)
	}
	if currentVersion > metadataSchemaVersion {
		return fmt.Errorf(
			"artifact metadata schema version %d is newer than supported version %d",
			currentVersion,
			metadataSchemaVersion,
		)
	}
	if currentVersion == metadataSchemaVersion {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin artifact metadata migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, migration := range metadataMigrations {
		if migration.Version <= currentVersion {
			continue
		}
		for _, statement := range migration.Statements {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf(
					"apply artifact metadata migration %d: %w",
					migration.Version,
					err,
				)
			}
		}
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d;", metadataSchemaVersion)); err != nil {
		return fmt.Errorf("set artifact metadata schema version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit artifact metadata migration: %w", err)
	}
	return nil
}
