package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

const sourceColumns = `
	id, root_id, kind, display_name, enabled, config_json,
	revision, created_at, modified_at, retired_at`

func (s *Store) createSource(
	ctx context.Context,
	value source.Source,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveRootTx(ctx, tx, value.RootID); err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO artifact_sources (
			id, root_id, kind, display_name, enabled, config_json,
			revision, created_at, modified_at, retired_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(value.ID),
		string(value.RootID),
		string(value.Kind),
		value.DisplayName,
		boolInt(value.Enabled),
		[]byte(value.Config),
		value.Revision,
		timeValue(value.CreatedAt),
		timeValue(value.ModifiedAt),
		nullableTime(value.RetiredAt),
	)
	if err != nil {
		return sqliteError(err)
	}
	return sqliteError(tx.Commit())
}

func (s *Store) getSource(
	ctx context.Context,
	rootID artifactstore.RootID,
	id artifactstore.SourceID,
) (source.Source, error) {
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return source.Source{}, err
	}

	value, err := scanSource(s.db.QueryRowContext(
		ctx,
		`SELECT `+sourceColumns+`
		 FROM artifact_sources
		 WHERE id = ? AND root_id = ? AND retired_at IS NULL`,
		string(id),
		string(rootID),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return source.Source{}, fmt.Errorf(
			"%w: source %q in root %q",
			artifactstore.ErrSourceNotFound,
			id,
			rootID,
		)
	}
	return value, err
}

func (s *Store) listSources(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]source.Source, error) {
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+sourceColumns+`
		 FROM artifact_sources
		 WHERE root_id = ? AND retired_at IS NULL
		 ORDER BY modified_at DESC, id ASC`,
		string(rootID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	output := make([]source.Source, 0)
	for rows.Next() {
		value, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	return output, rows.Err()
}

func (s *Store) updateSource(
	ctx context.Context,
	value source.Source,
	expectedRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if err := s.requireActiveRoot(ctx, value.RootID); err != nil {
		return err
	}

	if expectedRevision == 0 ||
		value.Revision != expectedRevision+1 ||
		value.RetiredAt != nil {
		return fmt.Errorf("%w: invalid source update", artifactstore.ErrInvalid)
	}
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE artifact_sources
		 SET display_name = ?,
		     enabled = ?,
		     config_json = ?,
		     revision = ?,
		     modified_at = ?
		 WHERE id = ?
		   AND root_id = ?
		   AND revision = ?
		   AND retired_at IS NULL
		   AND (
			? = 1 OR NOT EXISTS (
				SELECT 1
				FROM artifact_collection_attachments a
				JOIN artifact_collections c
				  ON c.root_id = a.root_id
				 AND c.id = a.collection_id
				WHERE a.root_id = artifact_sources.root_id
				  AND a.source_id = artifact_sources.id
				  AND a.enabled = 1
				  AND c.retired_at IS NULL
			)
		   )`,
		value.DisplayName,
		boolInt(value.Enabled),
		[]byte(value.Config),
		value.Revision,
		timeValue(value.ModifiedAt),
		string(value.ID),
		string(value.RootID),
		expectedRevision,
		boolInt(value.Enabled),
	)
	if err != nil {
		return sqliteError(err)
	}
	return requireOneChanged(
		result,
		"source changed, no longer exists, or still has enabled attachments",
	)
}

func (s *Store) retireSource(
	ctx context.Context,
	value source.Source,
	expectedRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if err := s.requireActiveRoot(ctx, value.RootID); err != nil {
		return err
	}

	if value.RetiredAt == nil ||
		value.Enabled ||
		value.Revision != expectedRevision+1 {
		return fmt.Errorf("%w: invalid source retirement", artifactstore.ErrInvalid)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var attached int
	if err := tx.QueryRowContext(
		ctx,
		`SELECT EXISTS(
			SELECT 1 FROM artifact_collection_attachments
			WHERE root_id = ? AND source_id = ?
		)`,
		string(value.RootID),
		string(value.ID),
	).Scan(&attached); err != nil {
		return err
	}
	if attached != 0 {
		return fmt.Errorf("%w: source is still attached to a collection", artifactstore.ErrConflict)
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_sources
		 SET enabled = 0,
		     revision = ?,
		     modified_at = ?,
		     retired_at = ?
		 WHERE id = ? AND root_id = ? AND revision = ? AND retired_at IS NULL`,
		value.Revision,
		timeValue(value.ModifiedAt),
		timeValue(*value.RetiredAt),
		string(value.ID),
		string(value.RootID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(result, "source changed during retirement"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) discardSource(
	ctx context.Context,
	rootID artifactstore.RootID,
	id artifactstore.SourceID,
	expectedRevision uint64,
) error {
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM artifact_sources
		 WHERE id = ?
		   AND root_id = ?
		   AND revision = ?
		   AND retired_at IS NULL
		   AND NOT EXISTS (
			SELECT 1
			FROM artifact_collection_attachments
			WHERE root_id = ? AND source_id = ?
		   )`,
		string(id),
		string(rootID),
		expectedRevision,
		string(rootID),
		string(id),
	)
	if err != nil {
		return sqliteError(err)
	}
	return requireOneChanged(
		result,
		"source changed or was attached before discard",
	)
}

func (s *Store) purgeSource(
	ctx context.Context,
	rootID artifactstore.RootID,
	id artifactstore.SourceID,
	expectedRevision uint64,
) error {
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM artifact_sources
		 WHERE id = ? AND root_id = ? AND revision = ? AND retired_at IS NOT NULL`,
		string(id),
		string(rootID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	return requireOneChanged(result, "source changed or was not retired before purge")
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSource(row scanner) (source.Source, error) {
	var (
		id, rootID, kind, displayName string
		enabled                       int
		config                        []byte
		revision                      uint64
		createdAt, modifiedAt         int64
		retiredAt                     sql.NullInt64
	)
	if err := row.Scan(
		&id,
		&rootID,
		&kind,
		&displayName,
		&enabled,
		&config,
		&revision,
		&createdAt,
		&modifiedAt,
		&retiredAt,
	); err != nil {
		return source.Source{}, err
	}
	value := source.Source{
		ID:          artifactstore.SourceID(id),
		RootID:      artifactstore.RootID(rootID),
		Kind:        artifactstore.SourceKind(kind),
		DisplayName: displayName,
		Enabled:     enabled != 0,
		Config:      append([]byte(nil), config...),
		Revision:    revision,
		CreatedAt:   parseTime(createdAt),
		ModifiedAt:  parseTime(modifiedAt),
		RetiredAt:   parseNullableTime(retiredAt),
	}
	if err := value.Validate(); err != nil {
		return source.Source{}, fmt.Errorf(
			"invalid persisted source %q: %w",
			id,
			err,
		)
	}
	return value, nil
}
