package sqlite

import (
	"context"
	"database/sql"
)

func migrateSchemaV2(
	ctx context.Context,
	tx *sql.Tx,
) error {
	_, err := tx.ExecContext(
		ctx,
		`CREATE TABLE artifact_topology_hydrations (
			installer_name TEXT PRIMARY KEY,
			root_id TEXT NOT NULL,
			source_id TEXT NOT NULL,
			fingerprint TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
	)
	return err
}
