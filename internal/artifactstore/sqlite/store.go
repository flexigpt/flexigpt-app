package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/mapstoreio"

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
	if strings.HasPrefix(filepath.VolumeName(path), `\\`) {
		return nil, fmt.Errorf(
			"%w: SQLite database paths on UNC or device shares are unsupported",
			basespec.ErrUnsupported,
		)
	}
	if err := prepareDatabaseFile(path); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dataSourceName(path))
	if err != nil {
		return nil, fmt.Errorf("open artifact metadata database: %w", err)
	}
	db.SetMaxOpenConns(4)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping artifact metadata database: %w", err)
	}
	if err := initializeSchemaV1(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := secureDatabaseFiles(path); err != nil {
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

func initializeSchemaV1(
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
			WHERE type = 'table' AND name = 'artifact_store_v1'
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
			"%w: legacy Artifact Store metadata is not supported; use a fresh artifacts_v1 directory",
			basespec.ErrUnsupported,
		)
	}

	if _, err := tx.ExecContext(ctx, schemaV1); err != nil {
		return fmt.Errorf("initialize Artifact Store v1 schema: %w", err)
	}
	return tx.Commit()
}

func prepareDatabaseFile(path string) error {
	info, err := os.Stat(path)
	switch {
	case err == nil:
		if !info.Mode().IsRegular() {
			return fmt.Errorf(
				"%w: SQLite path must identify a regular file",
				basespec.ErrInvalid,
			)
		}
	case errors.Is(err, os.ErrNotExist):
		if _, err := mapstoreio.PreparePrivateDirectory(
			filepath.Dir(path),
		); err != nil {
			return err
		}
	default:
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("prepare artifact metadata database: %w", err)
	}
	if err := file.Close(); err != nil {
		return err
	}
	return mapstoreio.ApplyPrivateFileMode(path)
}

func secureDatabaseFiles(path string) error {
	for _, candidate := range []string{
		path,
		path + "-wal",
		path + "-shm",
		path + "-journal",
	} {
		if err := mapstoreio.ApplyPrivateFileMode(candidate); err != nil &&
			!errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("secure artifact metadata database: %w", err)
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
