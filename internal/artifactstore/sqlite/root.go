package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
)

const rootColumns = `
	id, display_name, description, revision,
	created_at, modified_at, retired_at`

func (s *Store) createRoot(
	ctx context.Context,
	value root.Root,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO artifact_roots (
			id, display_name, description, revision,
			created_at, modified_at, retired_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		string(value.ID),
		value.DisplayName,
		value.Description,
		value.Revision,
		timeValue(value.CreatedAt),
		timeValue(value.ModifiedAt),
		nullableTime(value.RetiredAt),
	)
	return sqliteError(err)
}

func (s *Store) getRoot(
	ctx context.Context,
	id basespec.RootID,
) (root.Root, error) {
	value, err := scanRoot(s.db.QueryRowContext(
		ctx,
		`SELECT `+rootColumns+`
		 FROM artifact_roots
		 WHERE id = ? AND retired_at IS NULL`,
		string(id),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return root.Root{}, fmt.Errorf(
			"%w: root %q",
			basespec.ErrRootNotFound,
			id,
		)
	}
	return value, err
}

func (s *Store) listRoots(
	ctx context.Context,
) ([]root.Root, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+rootColumns+`
		 FROM artifact_roots
		 WHERE retired_at IS NULL
		 ORDER BY modified_at DESC, id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	output := make([]root.Root, 0)
	for rows.Next() {
		value, err := scanRoot(rows)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	return output, rows.Err()
}

func (s *Store) updateRoot(
	ctx context.Context,
	value root.Root,
	expectedRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 ||
		value.Revision != expectedRevision+1 ||
		value.RetiredAt != nil {
		return fmt.Errorf("%w: invalid root update", basespec.ErrInvalid)
	}
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE artifact_roots
		 SET display_name = ?,
		     description = ?,
		     revision = ?,
		     modified_at = ?
		 WHERE id = ? AND revision = ? AND retired_at IS NULL`,
		value.DisplayName,
		value.Description,
		value.Revision,
		timeValue(value.ModifiedAt),
		string(value.ID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	return requireOneChanged(result, "root changed during update")
}

func (s *Store) retireRoot(
	ctx context.Context,
	value root.Root,
	expectedRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if value.RetiredAt == nil ||
		value.Revision != expectedRevision+1 {
		return fmt.Errorf("%w: invalid root retirement", basespec.ErrInvalid)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	activeChildren, err := rootHasActiveChildrenTx(ctx, tx, value.ID)
	if err != nil {
		return err
	}
	if activeChildren {
		return fmt.Errorf(
			"%w: root %q still owns active sources or collections",
			basespec.ErrConflict,
			value.ID,
		)
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_roots
		 SET revision = ?, modified_at = ?, retired_at = ?
		 WHERE id = ? AND revision = ? AND retired_at IS NULL`,
		value.Revision,
		timeValue(value.ModifiedAt),
		timeValue(*value.RetiredAt),
		string(value.ID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(result, "root changed during retirement"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) purgeRoot(
	ctx context.Context,
	id basespec.RootID,
	expectedRevision uint64,
) error {
	if expectedRevision == 0 {
		return fmt.Errorf("%w: expected root revision is required", basespec.ErrInvalid)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	activeChildren, err := rootHasActiveChildrenTx(ctx, tx, id)
	if err != nil {
		return err
	}
	if activeChildren {
		return fmt.Errorf(
			"%w: root %q still owns active sources or collections",
			basespec.ErrConflict,
			id,
		)
	}

	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_roots
		 WHERE id = ? AND revision = ? AND retired_at IS NOT NULL`,
		string(id),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(result, "root changed or was not retired before purge"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) requireActiveRoot(
	ctx context.Context,
	id basespec.RootID,
) error {
	var marker int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM artifact_roots
		 WHERE id = ? AND retired_at IS NULL`,
		string(id),
	).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: root %q", basespec.ErrRootNotFound, id)
	}
	return err
}

func getActiveRootTx(
	ctx context.Context,
	tx *sql.Tx,
	id basespec.RootID,
) (root.Root, error) {
	value, err := scanRoot(tx.QueryRowContext(
		ctx,
		`SELECT `+rootColumns+`
		 FROM artifact_roots
		 WHERE id = ? AND retired_at IS NULL`,
		string(id),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return root.Root{}, fmt.Errorf(
			"%w: root %q",
			basespec.ErrRootNotFound,
			id,
		)
	}
	return value, err
}

func rootHasActiveChildrenTx(
	ctx context.Context,
	tx *sql.Tx,
	id basespec.RootID,
) (bool, error) {
	var exists int
	err := tx.QueryRowContext(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM artifact_sources
			WHERE root_id = ? AND retired_at IS NULL
			UNION ALL
			SELECT 1
			FROM artifact_collections
			WHERE root_id = ? AND retired_at IS NULL
		)`,
		string(id),
		string(id),
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists != 0, nil
}

func scanRoot(row scanner) (root.Root, error) {
	var (
		id, displayName, description string
		revision                     uint64
		createdAt, modifiedAt        int64
		retiredAt                    sql.NullInt64
	)
	if err := row.Scan(
		&id,
		&displayName,
		&description,
		&revision,
		&createdAt,
		&modifiedAt,
		&retiredAt,
	); err != nil {
		return root.Root{}, err
	}
	value := root.Root{
		ID:          basespec.RootID(id),
		DisplayName: displayName,
		Description: description,
		Revision:    revision,
		CreatedAt:   parseTime(createdAt),
		ModifiedAt:  parseTime(modifiedAt),
		RetiredAt:   parseNullableTime(retiredAt),
	}
	if err := value.Validate(); err != nil {
		return root.Root{}, fmt.Errorf("invalid persisted root %q: %w", id, err)
	}
	return value, nil
}

func requireOneChanged(
	result sql.Result,
	message string,
) error {
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("%w: %s", basespec.ErrConflict, message)
	}
	return nil
}
