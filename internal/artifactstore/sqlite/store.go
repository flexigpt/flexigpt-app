package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"

	_ "github.com/glebarez/go-sqlite"
)

type Store struct {
	db *sql.DB
}

const (
	initialSchemaVersion = 1
	currentSchemaVersion = 2
)

type schemaMigration func(context.Context, *sql.Tx) error

var schemaMigrations = map[int]schemaMigration{
	2: migrateSchemaV2,
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
		if err := bootstrapSchemaMigrationLedgerTx(ctx, tx); err != nil {
			return err
		}
	} else {
		if _, err := tx.ExecContext(ctx, schema); err != nil {
			return fmt.Errorf("initialize Artifact Store schema: %w", err)
		}
		if err := recordSchemaMigrationTx(
			ctx,
			tx,
			initialSchemaVersion,
		); err != nil {
			return err
		}
	}

	installedVersion, err := installedSchemaVersionTx(ctx, tx)
	if err != nil {
		return err
	}
	if installedVersion > currentSchemaVersion {
		return fmt.Errorf(
			"%w: artifact metadata schema version %d is newer than supported version %d",
			basespec.ErrUnsupported,
			installedVersion,
			currentSchemaVersion,
		)
	}
	if err := applySchemaMigrations(ctx, tx, installedVersion); err != nil {
		return err
	}
	return tx.Commit()
}

// bootstrapSchemaMigrationLedgerTx supports Artifact Store v1 databases
// created before the ledger existed. The artifact_store_v1 marker is exactly
// the schema-v1 baseline and does not require data transformation.
func bootstrapSchemaMigrationLedgerTx(
	ctx context.Context,
	tx *sql.Tx,
) error {
	var exists int
	if err := tx.QueryRowContext(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM sqlite_master
			WHERE type = 'table' AND name = 'artifact_schema_migrations'
		)`,
	).Scan(&exists); err != nil {
		return err
	}
	if exists != 0 {
		return nil
	}

	if _, err := tx.ExecContext(
		ctx,
		`CREATE TABLE artifact_schema_migrations (
			version INTEGER PRIMARY KEY CHECK (version > 0),
			applied_at INTEGER NOT NULL
		)`,
	); err != nil {
		return fmt.Errorf(
			"initialize artifact metadata migration ledger: %w",
			err,
		)
	}
	return recordSchemaMigrationTx(ctx, tx, initialSchemaVersion)
}

func installedSchemaVersionTx(
	ctx context.Context,
	tx *sql.Tx,
) (int, error) {
	rows, err := tx.QueryContext(
		ctx,
		`SELECT version
		 FROM artifact_schema_migrations
		 ORDER BY version`,
	)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()

	expectedVersion := initialSchemaVersion
	installedVersion := 0
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return 0, err
		}
		if version != expectedVersion {
			return 0, fmt.Errorf(
				"%w: artifact metadata migration ledger is not contiguous",
				basespec.ErrInvalid,
			)
		}
		installedVersion = version
		expectedVersion++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if installedVersion == 0 {
		return 0, fmt.Errorf(
			"%w: artifact metadata migration ledger is empty",
			basespec.ErrInvalid,
		)
	}
	return installedVersion, nil
}

func applySchemaMigrations(
	ctx context.Context,
	tx *sql.Tx,
	installedVersion int,
) error {
	for version := installedVersion + 1; version <= currentSchemaVersion; version++ {
		migration, exists := schemaMigrations[version]
		if !exists {
			return fmt.Errorf(
				"%w: artifact metadata migration %d is unavailable",
				basespec.ErrUnsupported,
				version,
			)
		}
		if err := migration(ctx, tx); err != nil {
			return fmt.Errorf(
				"apply artifact metadata migration %d: %w",
				version,
				err,
			)
		}
		if err := recordSchemaMigrationTx(ctx, tx, version); err != nil {
			return err
		}
	}
	return nil
}

func recordSchemaMigrationTx(
	ctx context.Context,
	tx *sql.Tx,
	version int,
) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO artifact_schema_migrations (version, applied_at)
		 VALUES (?, ?)`,
		version,
		time.Now().UTC().UnixNano(),
	)
	if err != nil {
		return fmt.Errorf(
			"record artifact metadata migration %d: %w",
			version,
			err,
		)
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
