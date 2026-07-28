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
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
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
			artifactstore.ErrInvalid,
		)
	}
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("%w: SQLite path is empty", artifactstore.ErrInvalid)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	path = filepath.Clean(absolute)
	if strings.HasPrefix(filepath.VolumeName(path), `\\`) {
		return nil, fmt.Errorf(
			"%w: SQLite database paths on UNC or device shares are unsupported",
			artifactstore.ErrUnsupported,
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
	if err := applySchemaMigrations(ctx, db); err != nil {
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

func applySchemaMigrations(
	ctx context.Context,
	db *sql.DB,
) error {
	if err := validateSchemaMigrations(schemaMigrations); err != nil {
		return err
	}

	if _, err := db.ExecContext(
		ctx,
		`CREATE TABLE IF NOT EXISTS artifact_schema_migrations (
			version INTEGER PRIMARY KEY,
			fingerprint TEXT NOT NULL,
			applied_at INTEGER NOT NULL
		)`,
	); err != nil {
		return fmt.Errorf("initialize artifact schema ledger: %w", err)
	}
	if err := validateAppliedSchemaMigrations(
		ctx,
		db,
		schemaMigrations,
	); err != nil {
		return err
	}
	for _, migration := range schemaMigrations {
		var fingerprint string
		err := db.QueryRowContext(
			ctx,
			`SELECT fingerprint
			 FROM artifact_schema_migrations
			 WHERE version = ?`,
			migration.version,
		).Scan(&fingerprint)
		switch {
		case err == nil:
			if fingerprint != migration.fingerprint {
				return fmt.Errorf(
					"%w: artifact schema migration %d has fingerprint %q",
					artifactstore.ErrUnsupported,
					migration.version,
					fingerprint,
				)
			}
			continue
		case !errors.Is(err, sql.ErrNoRows):
			return fmt.Errorf("read artifact schema ledger: %w", err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migration.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf(
				"apply artifact schema migration %d: %w",
				migration.version,
				err,
			)
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO artifact_schema_migrations(
				version, fingerprint, applied_at
			) VALUES (?, ?, ?)`,
			migration.version,
			migration.fingerprint,
			time.Now().UTC().UnixNano(),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf(
				"record artifact schema migration %d: %w",
				migration.version,
				err,
			)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func validateAppliedSchemaMigrations(
	ctx context.Context,
	db *sql.DB,
	values []migration,
) error {
	rows, err := db.QueryContext(
		ctx,
		`SELECT version, fingerprint
		 FROM artifact_schema_migrations
		 ORDER BY version`,
	)
	if err != nil {
		return fmt.Errorf("read artifact schema ledger: %w", err)
	}
	defer rows.Close()

	expectedIndex := 0
	for rows.Next() {
		var version int
		var fingerprint string
		if err := rows.Scan(&version, &fingerprint); err != nil {
			return fmt.Errorf("read artifact schema ledger entry: %w", err)
		}
		if expectedIndex >= len(values) {
			return fmt.Errorf(
				"%w: artifact metadata database contains unknown migration %d",
				artifactstore.ErrUnsupported,
				version,
			)
		}
		expected := values[expectedIndex]
		if version != expected.version {
			return fmt.Errorf(
				"%w: artifact metadata migration ledger is not an ordered prefix; expected migration %d before %d",
				artifactstore.ErrUnsupported,
				expected.version,
				version,
			)
		}
		if fingerprint != expected.fingerprint {
			return fmt.Errorf(
				"%w: artifact schema migration %d has fingerprint %q",
				artifactstore.ErrUnsupported,
				version,
				fingerprint,
			)
		}
		expectedIndex++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read artifact schema ledger: %w", err)
	}
	return nil
}

func validateSchemaMigrations(values []migration) error {
	previousVersion := 0
	for _, value := range values {
		if value.version <= previousVersion {
			return fmt.Errorf(
				"%w: artifact schema migrations must have strictly increasing versions",
				artifactstore.ErrInvalid,
			)
		}
		if strings.TrimSpace(value.fingerprint) == "" {
			return fmt.Errorf(
				"%w: artifact schema migration %d has no fingerprint",
				artifactstore.ErrInvalid,
				value.version,
			)
		}
		if strings.TrimSpace(value.sql) == "" {
			return fmt.Errorf(
				"%w: artifact schema migration %d has no SQL",
				artifactstore.ErrInvalid,
				value.version,
			)
		}
		previousVersion = value.version
	}
	return nil
}

func prepareDatabaseFile(path string) error {
	info, err := os.Stat(path)
	switch {
	case err == nil:
		if !info.Mode().IsRegular() {
			return fmt.Errorf(
				"%w: SQLite path must identify a regular file",
				artifactstore.ErrInvalid,
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
