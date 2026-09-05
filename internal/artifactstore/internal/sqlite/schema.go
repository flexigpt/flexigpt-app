package sqlite

// schema is the complete Artifact Store metadata schema for the unreleased
// v1 format. There are intentionally no historical migrations.
const schema = `
CREATE TABLE artifact_store_v2 (
	singleton INTEGER PRIMARY KEY CHECK (singleton = 1)
);

INSERT INTO artifact_store_v2(singleton) VALUES (1);

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

CREATE TABLE artifact_sources (
	id TEXT PRIMARY KEY,
	root_id TEXT NOT NULL REFERENCES artifact_roots(id) ON DELETE CASCADE,
	root_storage_key TEXT NOT NULL,
	storage_key TEXT NOT NULL,
	kind TEXT NOT NULL,
	display_name TEXT NOT NULL,
	enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
	config_json BLOB NOT NULL,
	revision INTEGER NOT NULL CHECK (revision > 0),
	created_at INTEGER NOT NULL,
	modified_at INTEGER NOT NULL,
	retired_at INTEGER,
	UNIQUE (root_id, id),
	UNIQUE (root_id, storage_key)
);

CREATE TABLE artifact_collections (
	id TEXT PRIMARY KEY,
	root_id TEXT NOT NULL REFERENCES artifact_roots(id) ON DELETE CASCADE,
	kind TEXT NOT NULL,
	display_name TEXT NOT NULL,
	description TEXT NOT NULL,
	enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
	data_json BLOB NOT NULL,
	revision INTEGER NOT NULL CHECK (revision > 0),
	created_at INTEGER NOT NULL,
	modified_at INTEGER NOT NULL,
	retired_at INTEGER,
	UNIQUE (root_id, id)
);

CREATE TABLE artifact_collection_attachments (
	root_id TEXT NOT NULL,
	collection_id TEXT NOT NULL,
	source_id TEXT NOT NULL,
	role TEXT NOT NULL,
	enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
	data_json BLOB NOT NULL,
	revision INTEGER NOT NULL CHECK (revision > 0),
	created_at INTEGER NOT NULL,
	modified_at INTEGER NOT NULL,
	PRIMARY KEY (root_id, collection_id, source_id),
	FOREIGN KEY (root_id, collection_id)
		REFERENCES artifact_collections(root_id, id) ON DELETE CASCADE,
	FOREIGN KEY (root_id, source_id)
		REFERENCES artifact_sources(root_id, id) ON DELETE RESTRICT
);

CREATE TABLE artifact_current_catalogs (
	root_id TEXT NOT NULL,
	collection_id TEXT NOT NULL,
	revision INTEGER NOT NULL CHECK (revision > 0),
	collection_revision INTEGER NOT NULL CHECK (collection_revision > 0),
	attachment_revisions_json BLOB NOT NULL,
	source_revisions_json BLOB NOT NULL,
	source_generations_json BLOB NOT NULL,
	plan_fingerprint TEXT NOT NULL,
	decoder_fingerprint TEXT NOT NULL,
	published_at INTEGER NOT NULL,
	diagnostics_json BLOB NOT NULL,
	PRIMARY KEY (root_id, collection_id),
	FOREIGN KEY (root_id, collection_id)
		REFERENCES artifact_collections(root_id, id) ON DELETE CASCADE
);

CREATE TABLE artifact_current_occurrences (
	root_id TEXT NOT NULL,
	collection_id TEXT NOT NULL,
	source_id TEXT NOT NULL,
	locator TEXT NOT NULL,
	subresource_locator TEXT NOT NULL,
	kind TEXT NOT NULL,
	logical_name TEXT NOT NULL,
	logical_version TEXT NOT NULL,
	definition_digest TEXT,
	definition_json BLOB,
	source_content_digest TEXT,
	decoder_id TEXT NOT NULL,
	state TEXT NOT NULL CHECK (state IN ('valid', 'invalid', 'missing')),
	diagnostics_json BLOB NOT NULL,
	observed_at INTEGER NOT NULL,
	PRIMARY KEY (
		root_id, collection_id, source_id, locator, subresource_locator
	),
	FOREIGN KEY (root_id, collection_id)
		REFERENCES artifact_collections(root_id, id) ON DELETE CASCADE,
	FOREIGN KEY (root_id, source_id)
		REFERENCES artifact_sources(root_id, id) ON DELETE RESTRICT
);

CREATE TABLE artifact_artifacts (
	id TEXT PRIMARY KEY,
	root_id TEXT NOT NULL,
	collection_id TEXT NOT NULL,
	source_id TEXT NOT NULL,
	locator TEXT NOT NULL,
	subresource_locator TEXT NOT NULL,
	kind TEXT NOT NULL,
	name TEXT NOT NULL,
	enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
	adoption TEXT NOT NULL CHECK (adoption IN ('observed', 'pinned')),
	resolved_definition_digest TEXT,
	data_json BLOB NOT NULL,
	state TEXT NOT NULL CHECK (
		state IN ('available', 'missing', 'invalid', 'incompatible')
	),
	diagnostics_json BLOB NOT NULL,
	revision INTEGER NOT NULL CHECK (revision > 0),
	created_at INTEGER NOT NULL,
	modified_at INTEGER NOT NULL,
	UNIQUE (
		root_id, collection_id, source_id,
		locator, subresource_locator, kind
	),
	FOREIGN KEY (root_id, collection_id)
		REFERENCES artifact_collections(root_id, id) ON DELETE CASCADE,
	FOREIGN KEY (root_id, source_id)
		REFERENCES artifact_sources(root_id, id) ON DELETE RESTRICT
);

CREATE TABLE artifact_suppressions (
	root_id TEXT NOT NULL,
	collection_id TEXT NOT NULL,
	source_id TEXT NOT NULL,
	locator TEXT NOT NULL,
	subresource_locator TEXT NOT NULL,
	expected_kind TEXT NOT NULL,
	revision INTEGER NOT NULL CHECK (revision > 0),
	created_at INTEGER NOT NULL,
	modified_at INTEGER NOT NULL,
	PRIMARY KEY (
		root_id, collection_id, source_id,
		locator, subresource_locator, expected_kind
	),
	FOREIGN KEY (root_id, collection_id)
		REFERENCES artifact_collections(root_id, id) ON DELETE CASCADE,
	FOREIGN KEY (root_id, source_id)
		REFERENCES artifact_sources(root_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_artifact_sources_root
	ON artifact_sources(root_id, modified_at DESC);

CREATE INDEX idx_artifact_attachments_source
	ON artifact_collection_attachments(root_id, source_id);

CREATE INDEX idx_artifact_occurrences_collection_kind
	ON artifact_current_occurrences(
		root_id, collection_id, kind, logical_name
	);

CREATE INDEX idx_artifact_artifacts_collection
	ON artifact_artifacts(
		root_id, collection_id, modified_at DESC
	);


CREATE TRIGGER artifact_attachment_requires_active_source_insert
BEFORE INSERT ON artifact_collection_attachments
FOR EACH ROW
WHEN NOT EXISTS (
	SELECT 1
	FROM artifact_sources s
	JOIN artifact_roots r ON r.id = s.root_id
	WHERE s.root_id = NEW.root_id
	  AND s.id = NEW.source_id
	  AND s.retired_at IS NULL
	  AND r.retired_at IS NULL
) OR NOT EXISTS (
	SELECT 1
	FROM artifact_collections c
	WHERE c.root_id = NEW.root_id
	  AND c.id = NEW.collection_id
	  AND c.retired_at IS NULL
)
BEGIN
	SELECT RAISE(ABORT, 'artifact attachment requires active source and collection');
END;

CREATE TRIGGER artifact_enabled_attachment_requires_enabled_source_insert
BEFORE INSERT ON artifact_collection_attachments
FOR EACH ROW
WHEN NEW.enabled = 1
 AND NOT EXISTS (
	SELECT 1
	FROM artifact_sources s
	JOIN artifact_roots r ON r.id = s.root_id
	WHERE s.root_id = NEW.root_id
	  AND s.id = NEW.source_id
	  AND s.enabled = 1
	  AND s.retired_at IS NULL
	  AND r.retired_at IS NULL
)
BEGIN
	SELECT RAISE(ABORT, 'artifact enabled attachment requires enabled source');
END;

CREATE TRIGGER artifact_enabled_attachment_requires_enabled_source_update
BEFORE UPDATE OF root_id, collection_id, source_id, enabled
ON artifact_collection_attachments
FOR EACH ROW
WHEN NEW.enabled = 1
 AND NOT EXISTS (
	SELECT 1
	FROM artifact_sources s
	JOIN artifact_roots r ON r.id = s.root_id
	WHERE s.root_id = NEW.root_id
	  AND s.id = NEW.source_id
	  AND s.enabled = 1
	  AND s.retired_at IS NULL
	  AND r.retired_at IS NULL
)
BEGIN
	SELECT RAISE(ABORT, 'artifact enabled attachment requires enabled source');
END;

CREATE TRIGGER artifact_source_disable_requires_disabled_attachments
BEFORE UPDATE OF enabled ON artifact_sources
FOR EACH ROW
WHEN OLD.enabled = 1
 AND NEW.enabled = 0
 AND EXISTS (
	SELECT 1
	FROM artifact_collection_attachments a
	JOIN artifact_collections c
	  ON c.root_id = a.root_id
	 AND c.id = a.collection_id
	WHERE a.root_id = OLD.root_id
	  AND a.source_id = OLD.id
	  AND a.enabled = 1
	  AND c.retired_at IS NULL
)
BEGIN
	SELECT RAISE(ABORT, 'artifact source disable requires disabled attachments');
END;

CREATE TRIGGER artifact_record_requires_attached_source_insert
BEFORE INSERT ON artifact_artifacts
FOR EACH ROW
WHEN NOT EXISTS (
	SELECT 1
	FROM artifact_collection_attachments a
	JOIN artifact_sources s
	  ON s.root_id = a.root_id AND s.id = a.source_id
	JOIN artifact_collections c
	  ON c.root_id = a.root_id AND c.id = a.collection_id
	JOIN artifact_roots r ON r.id = a.root_id
	WHERE a.root_id = NEW.root_id
	  AND a.collection_id = NEW.collection_id
	  AND a.source_id = NEW.source_id
	  AND s.retired_at IS NULL
	  AND c.retired_at IS NULL
	  AND r.retired_at IS NULL
)
BEGIN
	SELECT RAISE(ABORT, 'artifact record requires attached source');
END;

CREATE TRIGGER artifact_suppression_requires_attached_source_insert
BEFORE INSERT ON artifact_suppressions
FOR EACH ROW
WHEN NOT EXISTS (
	SELECT 1
	FROM artifact_collection_attachments a
	JOIN artifact_sources s
	  ON s.root_id = a.root_id AND s.id = a.source_id
	JOIN artifact_collections c
	  ON c.root_id = a.root_id AND c.id = a.collection_id
	JOIN artifact_roots r ON r.id = a.root_id
	WHERE a.root_id = NEW.root_id
	  AND a.collection_id = NEW.collection_id
	  AND a.source_id = NEW.source_id
	  AND s.retired_at IS NULL
	  AND c.retired_at IS NULL
	  AND r.retired_at IS NULL
)
BEGIN
	SELECT RAISE(ABORT, 'artifact suppression requires attached source');
END;

CREATE TRIGGER artifact_occurrence_requires_attached_source_insert
BEFORE INSERT ON artifact_current_occurrences
FOR EACH ROW
WHEN NOT EXISTS (
	SELECT 1
	FROM artifact_collection_attachments a
	JOIN artifact_sources s
	  ON s.root_id = a.root_id AND s.id = a.source_id
	JOIN artifact_collections c
	  ON c.root_id = a.root_id AND c.id = a.collection_id
	JOIN artifact_roots r ON r.id = a.root_id
	WHERE a.root_id = NEW.root_id
	  AND a.collection_id = NEW.collection_id
	  AND a.source_id = NEW.source_id
	  AND s.retired_at IS NULL
	  AND c.retired_at IS NULL
	  AND r.retired_at IS NULL
)
BEGIN
	SELECT RAISE(ABORT, 'artifact occurrence requires attached source');
END;

CREATE TRIGGER artifact_source_retirement_requires_no_active_attachments
BEFORE UPDATE OF retired_at ON artifact_sources
FOR EACH ROW
WHEN OLD.retired_at IS NULL
 AND NEW.retired_at IS NOT NULL
 AND EXISTS (
	SELECT 1
	FROM artifact_collection_attachments a
	JOIN artifact_collections c
	  ON c.root_id = a.root_id
	 AND c.id = a.collection_id
	WHERE a.root_id = OLD.root_id
	  AND a.source_id = OLD.id
	  AND c.retired_at IS NULL
)
BEGIN
	SELECT RAISE(
		ABORT,
		'artifact source retirement requires no active attachments'
	);
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
	FROM artifact_collections
	WHERE root_id = OLD.id
	  AND retired_at IS NULL
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
	FROM artifact_collections
	WHERE root_id = OLD.id
	  AND retired_at IS NULL
)
BEGIN
	SELECT RAISE(
		ABORT,
		'artifact root purge requires no active children'
	);
END;
`
