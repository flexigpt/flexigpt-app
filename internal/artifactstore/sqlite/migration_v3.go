package sqlite

import (
	"context"
	"database/sql"
)

func migrateSchemaV3(
	ctx context.Context,
	tx *sql.Tx,
) error {
	_, err := tx.ExecContext(
		ctx,
		`CREATE TABLE artifact_collection_shareable_documents (
			root_id TEXT NOT NULL,
			collection_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			schema_id TEXT NOT NULL,
			schema_version TEXT NOT NULL,
			digest TEXT NOT NULL,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY (root_id, collection_id),
			FOREIGN KEY (root_id, collection_id)
				REFERENCES artifact_collections(root_id, id) ON DELETE CASCADE
		)`,
	)
	return err
}
