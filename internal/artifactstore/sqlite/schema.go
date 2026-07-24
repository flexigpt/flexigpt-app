package sqlite

const schemaVersion = 1

const initializeSchemaSQL = `
CREATE TABLE IF NOT EXISTS artifact_schema (
	version INTEGER PRIMARY KEY,
	fingerprint TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS artifact_sources (
	id TEXT PRIMARY KEY,
	kind TEXT NOT NULL,
	display_name TEXT NOT NULL,
	enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
	config_json BLOB NOT NULL,
	revision INTEGER NOT NULL CHECK (revision > 0),
	created_at INTEGER NOT NULL,
	modified_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS artifact_roots (
	id TEXT PRIMARY KEY,
	kind TEXT NOT NULL,
	display_name TEXT NOT NULL,
	description TEXT NOT NULL,
	enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
	data_json BLOB NOT NULL,
	revision INTEGER NOT NULL CHECK (revision > 0),
	created_at INTEGER NOT NULL,
	modified_at INTEGER NOT NULL,
	deleted_at INTEGER
);

CREATE TABLE IF NOT EXISTS artifact_attachments (
	root_id TEXT NOT NULL REFERENCES artifact_roots(id) ON DELETE CASCADE,
	source_id TEXT NOT NULL REFERENCES artifact_sources(id) ON DELETE RESTRICT,
	role TEXT NOT NULL,
	enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
	data_json BLOB NOT NULL,
	revision INTEGER NOT NULL CHECK (revision > 0),
	created_at INTEGER NOT NULL,
	modified_at INTEGER NOT NULL,
	PRIMARY KEY (root_id, source_id)
);

CREATE TABLE IF NOT EXISTS artifact_current_catalogs (
	root_id TEXT PRIMARY KEY REFERENCES artifact_roots(id) ON DELETE CASCADE,
	revision INTEGER NOT NULL CHECK (revision > 0),
	root_revision INTEGER NOT NULL CHECK (root_revision > 0),
	source_revisions_json BLOB NOT NULL,
	source_generations_json BLOB NOT NULL,
	published_at INTEGER NOT NULL,
	diagnostics_json BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS artifact_current_occurrences (
	root_id TEXT NOT NULL REFERENCES artifact_roots(id) ON DELETE CASCADE,
	source_id TEXT NOT NULL REFERENCES artifact_sources(id) ON DELETE RESTRICT,
	locator TEXT NOT NULL,
	subresource_locator TEXT NOT NULL,
	kind TEXT NOT NULL,
	logical_name TEXT NOT NULL,
	logical_version TEXT NOT NULL,
	definition_digest TEXT,
	source_content_digest TEXT,
	decoder_id TEXT NOT NULL,
	state TEXT NOT NULL CHECK (state IN ('valid', 'invalid', 'missing')),
	diagnostics_json BLOB NOT NULL,
	observed_at INTEGER NOT NULL,
	PRIMARY KEY (root_id, source_id, locator, subresource_locator)
);

CREATE TABLE IF NOT EXISTS artifact_records (
	id TEXT PRIMARY KEY,
	root_id TEXT NOT NULL REFERENCES artifact_roots(id) ON DELETE CASCADE,
	source_id TEXT NOT NULL REFERENCES artifact_sources(id) ON DELETE RESTRICT,
	locator TEXT NOT NULL,
	subresource_locator TEXT NOT NULL,
	kind TEXT NOT NULL,
	name TEXT NOT NULL,
	enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
	resolved_definition_digest TEXT,
	data_json BLOB NOT NULL,
	state TEXT NOT NULL CHECK (
		state IN ('available', 'missing', 'invalid', 'incompatible')
	),
	diagnostics_json BLOB NOT NULL,
	revision INTEGER NOT NULL CHECK (revision > 0),
	created_at INTEGER NOT NULL,
	modified_at INTEGER NOT NULL,
	UNIQUE (root_id, source_id, locator, subresource_locator, kind)
);

CREATE INDEX IF NOT EXISTS idx_artifact_attachments_source
	ON artifact_attachments(source_id);

CREATE INDEX IF NOT EXISTS idx_artifact_occurrences_root_kind
	ON artifact_current_occurrences(root_id, kind, logical_name);

CREATE INDEX IF NOT EXISTS idx_artifact_records_root
	ON artifact_records(root_id, modified_at DESC);
`
