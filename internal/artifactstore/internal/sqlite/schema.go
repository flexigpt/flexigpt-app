package sqlite

const topologyPackageHydrationTable = "artifact_topology_package_hydrations"

// schema is the complete Artifact Store metadata schema for the direct v3
// Root-scoped model.
const sqliteSchema = `
CREATE TABLE artifact_store_v3 (
	singleton INTEGER PRIMARY KEY CHECK (singleton = 1)
);

INSERT INTO artifact_store_v3(singleton) VALUES (1);

CREATE TABLE artifact_roots (
	id TEXT PRIMARY KEY,
	storage_key TEXT NOT NULL UNIQUE,
	display_name TEXT NOT NULL,
	description TEXT NOT NULL,
	revision INTEGER NOT NULL CHECK (revision > 0),
	created_at INTEGER NOT NULL,
	modified_at INTEGER NOT NULL,
	retired_at INTEGER
);

CREATE TABLE artifact_topology_hydrations (
	installer_name TEXT PRIMARY KEY,
	root_id TEXT NOT NULL,
	source_id TEXT NOT NULL,
	fingerprint TEXT NOT NULL,
	updated_at INTEGER NOT NULL
);

CREATE TABLE artifact_topology_package_hydrations (
	installer_name TEXT NOT NULL,
	package_scope TEXT NOT NULL,
	root_id TEXT NOT NULL,
	source_id TEXT NOT NULL,
	fingerprint TEXT NOT NULL,
	updated_at INTEGER NOT NULL,
	PRIMARY KEY (installer_name, package_scope)
);

CREATE TABLE artifact_sources (
	id TEXT PRIMARY KEY,
	root_id TEXT NOT NULL REFERENCES artifact_roots(id) ON DELETE CASCADE,
	root_storage_key TEXT NOT NULL,
	storage_key TEXT NOT NULL,
	kind TEXT NOT NULL,
	display_name TEXT NOT NULL,
	enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
	config_json BLOB NOT NULL,
	discovery_json BLOB NOT NULL,
	revision INTEGER NOT NULL CHECK (revision > 0),
	created_at INTEGER NOT NULL,
	modified_at INTEGER NOT NULL,
	retired_at INTEGER,
	UNIQUE (root_id, id),
	UNIQUE (root_id, storage_key)
);

CREATE TABLE artifact_source_refresh_state (
	root_id TEXT NOT NULL,
	source_id TEXT NOT NULL,
	source_revision INTEGER NOT NULL CHECK (source_revision > 0),
	source_generation TEXT NOT NULL,
	discovery_fingerprint TEXT NOT NULL,
	decoder_fingerprint TEXT NOT NULL,
	revision INTEGER NOT NULL CHECK (revision > 0),
	refreshed_at INTEGER NOT NULL,
	diagnostics_json BLOB NOT NULL,
	PRIMARY KEY (root_id, source_id),
	FOREIGN KEY (root_id, source_id)
		REFERENCES artifact_sources(root_id, id) ON DELETE CASCADE
);

CREATE TABLE artifact_definitions (
	root_id TEXT NOT NULL REFERENCES artifact_roots(id) ON DELETE CASCADE,
	digest TEXT NOT NULL,
	kind TEXT NOT NULL,
	schema_id TEXT NOT NULL,
	schema_version TEXT NOT NULL,
	logical_name TEXT NOT NULL,
	logical_version TEXT NOT NULL,
	display_name TEXT NOT NULL,
	description TEXT NOT NULL,
	labels_json BLOB NOT NULL,
	body_json BLOB NOT NULL,
	dependencies_json BLOB NOT NULL,
	created_at INTEGER NOT NULL,
	PRIMARY KEY (root_id, digest)
);

CREATE TABLE artifact_artifacts (
	id TEXT PRIMARY KEY,
	root_id TEXT NOT NULL,
	source_id TEXT NOT NULL,
	locator TEXT NOT NULL,
	subresource_locator TEXT NOT NULL,
	kind TEXT NOT NULL,
	logical_name TEXT NOT NULL,
	logical_version TEXT NOT NULL,
	resolved_definition_digest TEXT,
	source_content_digest TEXT,
	state TEXT NOT NULL CHECK (
		state IN ('available', 'missing', 'invalid', 'incompatible')
	),
	diagnostics_json BLOB NOT NULL,
	display_name TEXT NOT NULL,
	enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
	data_json BLOB NOT NULL,
	revision INTEGER NOT NULL CHECK (revision > 0),
	created_at INTEGER NOT NULL,
	modified_at INTEGER NOT NULL,
	UNIQUE (
		root_id, source_id, locator, subresource_locator, kind
	),
	FOREIGN KEY (root_id, source_id)
		REFERENCES artifact_sources(root_id, id) ON DELETE RESTRICT,
	FOREIGN KEY (root_id, resolved_definition_digest)
		REFERENCES artifact_definitions(root_id, digest) ON DELETE RESTRICT
);

CREATE INDEX idx_artifact_sources_root
	ON artifact_sources(root_id, modified_at DESC);

CREATE INDEX idx_artifact_refresh_state_source
	ON artifact_source_refresh_state(root_id, source_id);

CREATE INDEX idx_artifact_definitions_identity
	ON artifact_definitions(root_id, kind, logical_name);

CREATE INDEX idx_artifact_artifacts_root_identity
	ON artifact_artifacts(root_id, kind, logical_name);

CREATE INDEX idx_artifact_artifacts_source
	ON artifact_artifacts(root_id, source_id, modified_at DESC);

CREATE INDEX idx_artifact_artifacts_origin
	ON artifact_artifacts(
		root_id, source_id, locator, subresource_locator
	);

CREATE TRIGGER artifact_record_requires_active_source_insert
BEFORE INSERT ON artifact_artifacts
FOR EACH ROW
WHEN NOT EXISTS (
	SELECT 1
	FROM artifact_sources s
	JOIN artifact_roots r ON r.id = s.root_id
	WHERE s.root_id = NEW.root_id
	  AND s.id = NEW.source_id
	  AND s.retired_at IS NULL
	  AND r.retired_at IS NULL
)
BEGIN
	SELECT RAISE(ABORT, 'artifact record requires active source');
END;

CREATE TRIGGER artifact_root_retirement_requires_no_active_children
BEFORE UPDATE OF retired_at ON artifact_roots
FOR EACH ROW
WHEN OLD.retired_at IS NULL
 AND NEW.retired_at IS NOT NULL
 AND EXISTS (
	SELECT 1
	FROM artifact_sources
	WHERE root_id = OLD.id
	  AND retired_at IS NULL
	UNION ALL
	SELECT 1
	FROM artifact_artifacts
	WHERE root_id = OLD.id
)
BEGIN
	SELECT RAISE(
		ABORT,
		'artifact root retirement requires no active children'
	);
END;

CREATE TRIGGER artifact_root_purge_requires_no_active_children
BEFORE DELETE ON artifact_roots
FOR EACH ROW
WHEN EXISTS (
	SELECT 1
	FROM artifact_sources
	WHERE root_id = OLD.id
	  AND retired_at IS NULL
	UNION ALL
	SELECT 1
	FROM artifact_artifacts
	WHERE root_id = OLD.id
)
BEGIN
	SELECT RAISE(
		ABORT,
		'artifact root purge requires no active children'
	);
END;
`
