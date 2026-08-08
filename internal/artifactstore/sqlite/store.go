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

const schemaMarkerTable = "artifact_store_v1"

var schemaV1RequiredTables = []string{
	"artifact_topology_hydrations",
	"artifact_collection_shareable_documents",
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
		return nil, fmt.Errorf("%w: SQLite path is empty", basespec.ErrInvalid)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	path = filepath.Clean(absolute)

	db, err := sql.Open("sqlite", dataSourceName(path))
	if err != nil {
		return nil, fmt.Errorf("open artifact metadata database: %w", err)
	}
	db.SetMaxOpenConns(4)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping artifact metadata database: %w", err)
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

	markerExists, err := tableExistsTx(ctx, tx, schemaMarkerTable)
	if err != nil {
		return err
	}

	if markerExists {
		if err := verifySchemaV1Tx(ctx, tx); err != nil {
			return err
		}
		return tx.Commit()
	}

	if _, err := tx.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("initialize Artifact Store schema: %w", err)
	}
	return tx.Commit()
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

func verifySchemaV1Tx(
	ctx context.Context,
	tx *sql.Tx,
) error {
	for _, table := range schemaV1RequiredTables {
		exists, err := tableExistsTx(ctx, tx, table)
		if err != nil {
			return fmt.Errorf(
				"inspect Artifact Store schema table %q: %w",
				table,
				err,
			)
		}
		if !exists {
			return fmt.Errorf(
				"%w: Artifact Store database does not match schema v1; delete the development Artifact Store directory",
				basespec.ErrUnsupported,
			)
		}
	}
	return nil
}

func dataSourceName(path string) string {
	normalized := filepath.ToSlash(filepath.Clean(path))
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
