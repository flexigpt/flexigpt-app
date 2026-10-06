package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

const rootColumns = `
	id, storage_key, display_name, description, revision,
	created_at, modified_at, retired_at`

func (s *Store) createRoot(
	ctx context.Context,
	value rootModel.Root,
) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO artifact_roots (
			id, storage_key, display_name, description, revision,
			created_at, modified_at, retired_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		string(value.ID),
		string(value.StorageKey),
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
	id rootModel.RootID,
) (rootModel.Root, error) {
	value, err := scanRoot(s.db.QueryRowContext(
		ctx,
		`SELECT `+rootColumns+`
		 FROM artifact_roots
		 WHERE id = ? AND retired_at IS NULL`,
		string(id),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return rootModel.Root{}, fmt.Errorf(
			"%w: root %q",
			spec.ErrRootNotFound,
			id,
		)
	}
	return value, err
}

func (s *Store) listRoots(
	ctx context.Context,
) ([]rootModel.Root, error) {
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

	output := make([]rootModel.Root, 0)
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
	value rootModel.Root,
	expectedRevision uint64,
) error {
	if expectedRevision == 0 ||
		value.Revision != expectedRevision+1 ||
		value.RetiredAt != nil {
		return fmt.Errorf("%w: invalid root update", spec.ErrInvalid)
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
	value rootModel.Root,
	expectedRevision uint64,
) error {
	if value.RetiredAt == nil ||
		value.Revision != expectedRevision+1 {
		return fmt.Errorf("%w: invalid root retirement", spec.ErrInvalid)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

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
	id rootModel.RootID,
	expectedRevision uint64,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

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
	if err := tx.Commit(); err != nil {
		return err
	}
	s.forgetDefinitionsForRoot(id)
	return nil
}

func (s *Store) requireActiveRoot(
	ctx context.Context,
	id rootModel.RootID,
) error {
	var marker int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM artifact_roots
		 WHERE id = ? AND retired_at IS NULL`,
		string(id),
	).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: root %q", spec.ErrRootNotFound, id)
	}
	return err
}

func getActiveRootTx(
	ctx context.Context,
	tx *sql.Tx,
	id rootModel.RootID,
) (rootModel.Root, error) {
	value, err := scanRoot(tx.QueryRowContext(
		ctx,
		`SELECT `+rootColumns+`
		 FROM artifact_roots
		 WHERE id = ? AND retired_at IS NULL`,
		string(id),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return rootModel.Root{}, fmt.Errorf(
			"%w: root %q",
			spec.ErrRootNotFound,
			id,
		)
	}
	return value, err
}

func scanRoot(row scanner) (rootModel.Root, error) {
	var (
		id, storageKey, displayName, description string
		revision                                 uint64
		createdAt, modifiedAt                    int64
		retiredAt                                sql.NullInt64
	)
	if err := row.Scan(
		&id,
		&storageKey,
		&displayName,
		&description,
		&revision,
		&createdAt,
		&modifiedAt,
		&retiredAt,
	); err != nil {
		return rootModel.Root{}, err
	}
	value := rootModel.Root{
		ID:          rootModel.RootID(id),
		StorageKey:  spec.StorageKey(storageKey),
		DisplayName: displayName,
		Description: description,
		Revision:    revision,
		CreatedAt:   parseTime(createdAt),
		ModifiedAt:  parseTime(modifiedAt),
		RetiredAt:   parseNullableTime(retiredAt),
	}
	if err := value.Validate(); err != nil {
		return rootModel.Root{}, fmt.Errorf("invalid persisted root %q: %w", id, err)
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
		return fmt.Errorf("%w: %s", spec.ErrConflict, message)
	}
	return nil
}
