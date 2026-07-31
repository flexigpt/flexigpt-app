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
	if err := initializeSchemaV2(ctx, db); err != nil {
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

func initializeSchemaV2(
	ctx context.Context,
	db *sql.DB,
) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var markerExists int
	if err := tx.QueryRowContext(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM sqlite_master
			WHERE type = 'table' AND name = 'artifact_store_v2'
		)`,
	).Scan(&markerExists); err != nil {
		return err
	}
	if markerExists != 0 {
		return tx.Commit()
	}

	var legacyExists int
	if err := tx.QueryRowContext(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM sqlite_master
			WHERE type = 'table'
			  AND name IN (
				'artifact_schema_migrations',
				'artifact_roots',
				'artifact_sources',
				'artifact_collections'
			  )
		)`,
	).Scan(&legacyExists); err != nil {
		return err
	}
	if legacyExists != 0 {
		return fmt.Errorf(
			"%w: legacy Artifact Store metadata is not supported; use a fresh artifacts_v2 directory",
			basespec.ErrUnsupported,
		)
	}

	if _, err := tx.ExecContext(ctx, schemaV2); err != nil {
		return fmt.Errorf("initialize Artifact Store v2 schema: %w", err)
	}
	return tx.Commit()
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
