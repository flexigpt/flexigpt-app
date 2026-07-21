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
	id, kind, display_name, enabled, config_json,
	revision, created_at, modified_at`

func (s *Store) createSource(
	ctx context.Context,
	value source.Source,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO artifact_sources (
			id, kind, display_name, enabled, config_json,
			revision, created_at, modified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		string(value.ID),
		string(value.Kind),
		value.DisplayName,
		boolInt(value.Enabled),
		[]byte(value.Config),
		value.Revision,
		timeValue(value.CreatedAt),
		timeValue(value.ModifiedAt),
	)
	return sqliteError(err)
}

func (s *Store) getSource(
	ctx context.Context,
	id artifactstore.SourceID,
) (source.Source, error) {
	value, err := scanSource(s.db.QueryRowContext(
		ctx,
		`SELECT `+sourceColumns+` FROM artifact_sources WHERE id = ?`,
		string(id),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return source.Source{}, fmt.Errorf(
			"%w: source %q",
			artifactstore.ErrNotFound,
			id,
		)
	}
	return value, err
}

func (s *Store) listSources(
	ctx context.Context,
) ([]source.Source, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+sourceColumns+`
		 FROM artifact_sources
		 ORDER BY modified_at DESC, id ASC`,
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
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE artifact_sources
		 SET display_name = ?,
		     enabled = ?,
		     config_json = ?,
		     revision = ?,
		     modified_at = ?
		 WHERE id = ? AND revision = ?`,
		value.DisplayName,
		boolInt(value.Enabled),
		[]byte(value.Config),
		value.Revision,
		timeValue(value.ModifiedAt),
		string(value.ID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf(
			"%w: source %q changed or no longer exists",
			artifactstore.ErrConflict,
			value.ID,
		)
	}
	return nil
}

func (s *Store) deleteSource(
	ctx context.Context,
	id artifactstore.SourceID,
	expectedRevision uint64,
) error {
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM artifact_sources WHERE id = ? AND revision = ?`,
		string(id),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf(
			"%w: source %q changed or no longer exists",
			artifactstore.ErrConflict,
			id,
		)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSource(row scanner) (source.Source, error) {
	var (
		id, kind, displayName string
		enabled               int
		config                []byte
		revision              uint64
		createdAt, modifiedAt int64
	)
	if err := row.Scan(
		&id,
		&kind,
		&displayName,
		&enabled,
		&config,
		&revision,
		&createdAt,
		&modifiedAt,
	); err != nil {
		return source.Source{}, err
	}
	value := source.Source{
		ID:          artifactstore.SourceID(id),
		Kind:        artifactstore.SourceKind(kind),
		DisplayName: displayName,
		Enabled:     enabled != 0,
		Config:      append([]byte(nil), config...),
		Revision:    revision,
		CreatedAt:   parseTime(createdAt),
		ModifiedAt:  parseTime(modifiedAt),
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
