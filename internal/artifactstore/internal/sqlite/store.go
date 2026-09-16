package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"

	_ "github.com/glebarez/go-sqlite"
)

type Store struct {
	db *sql.DB
}

const (
	schemaMarkerTable   = "artifact_store_v3"
	legacyMarkerTableV2 = "artifact_store_v2"
)

var schemaV3RequiredTables = []string{
	"artifact_roots",
	"artifact_topology_hydrations",
	"artifact_sources",
	topologyPackageHydrationTable,
	"artifact_source_refresh_state",
	"artifact_definitions",
	"artifact_artifacts",
}

func Open(
	ctx context.Context,
	path string,
) (*Store, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: SQLite context is nil",
			basespec.ErrInvalid,
		)
	}
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf(
			"%w: SQLite path is empty",
			basespec.ErrInvalid,
		)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(
		"sqlite",
		dataSourceName(filepath.Clean(absolute)),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"open Artifact Store metadata database: %w",
			err,
		)
	}
	db.SetMaxOpenConns(4)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf(
			"ping Artifact Store metadata database: %w",
			err,
		)
	}
	if err := initializeSchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func initializeSchema(
	ctx context.Context,
	db *sql.DB,
) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	v3Exists, err := tableExistsTx(ctx, tx, schemaMarkerTable)
	if err != nil {
		return err
	}
	if v3Exists {
		if _, err := tx.ExecContext(
			ctx,
			`CREATE TABLE IF NOT EXISTS artifact_topology_package_hydrations (
				installer_name TEXT NOT NULL,
				package_scope TEXT NOT NULL,
				root_id TEXT NOT NULL,
				source_id TEXT NOT NULL,
				fingerprint TEXT NOT NULL,
				updated_at INTEGER NOT NULL,
				PRIMARY KEY (installer_name, package_scope)
			)`,
		); err != nil {
			return err
		}
		if err := verifySchemaV3Tx(ctx, tx); err != nil {
			return err
		}
		return tx.Commit()
	}

	v2Exists, err := tableExistsTx(
		ctx,
		tx,
		legacyMarkerTableV2,
	)
	if err != nil {
		return err
	}
	if v2Exists {
		return fmt.Errorf(
			"%w: Artifact Store metadata is v2; use the v3 Artifact Store directory",
			basespec.ErrUnsupported,
		)
	}

	if _, err := tx.ExecContext(ctx, sqliteSchema); err != nil {
		return fmt.Errorf(
			"initialize Artifact Store v3 schema: %w",
			err,
		)
	}
	return tx.Commit()
}

func verifySchemaV3Tx(
	ctx context.Context,
	tx *sql.Tx,
) error {
	for _, table := range schemaV3RequiredTables {
		exists, err := tableExistsTx(ctx, tx, table)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf(
				"%w: Artifact Store database does not match v3 schema",
				basespec.ErrUnsupported,
			)
		}
	}
	return nil
}

func tableExistsTx(
	ctx context.Context,
	tx *sql.Tx,
	name string,
) (bool, error) {
	var exists int
	err := tx.QueryRowContext(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM sqlite_master
			WHERE type = 'table' AND name = ?
		)`,
		name,
	).Scan(&exists)
	return exists != 0, err
}

func dataSourceName(path string) string {
	normalized := filepath.ToSlash(path)
	if filepath.VolumeName(path) != "" &&
		!strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	value := &url.URL{
		Scheme: "file",
		Path:   normalized,
	}
	query := value.Query()
	query.Set("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "busy_timeout(5000)")
	value.RawQuery = query.Encode()
	return value.String()
}
