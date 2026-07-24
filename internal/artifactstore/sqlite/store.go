package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"

	_ "github.com/glebarez/go-sqlite"
)

const schemaFingerprint = "artifactstore.clean.v1"

type Store struct {
	db *sql.DB
}

func Open(
	ctx context.Context,
	path string,
) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("%w: SQLite path is empty", artifactstore.ErrInvalid)
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
	if _, err := db.ExecContext(ctx, initializeSchemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize artifact metadata schema: %w", err)
	}
	if err := verifySchema(ctx, db); err != nil {
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

func verifySchema(
	ctx context.Context,
	db *sql.DB,
) error {
	var fingerprint string
	err := db.QueryRowContext(
		ctx,
		`SELECT fingerprint FROM artifact_schema WHERE version = ?`,
		schemaVersion,
	).Scan(&fingerprint)

	if err == sql.ErrNoRows {
		if _, err := db.ExecContext(
			ctx,
			`INSERT OR IGNORE INTO artifact_schema(version, fingerprint) VALUES (?, ?)`,
			schemaVersion,
			schemaFingerprint,
		); err != nil {
			return fmt.Errorf("record artifact schema identity: %w", err)
		}
		err = db.QueryRowContext(
			ctx,
			`SELECT fingerprint FROM artifact_schema WHERE version = ?`,
			schemaVersion,
		).Scan(&fingerprint)
	}
	if err != nil {
		return fmt.Errorf("read artifact schema identity: %w", err)
	}
	if fingerprint != schemaFingerprint {
		return fmt.Errorf(
			"%w: artifact metadata schema fingerprint %q is unsupported",
			artifactstore.ErrUnsupported,
			fingerprint,
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
