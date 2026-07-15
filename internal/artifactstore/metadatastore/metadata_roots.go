package metadatastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

const rootColumns = `
	root_id,
	kind,
	display_name,
	description,
	enabled,
	data_schema_id,
	data_json,
	created_at,
	modified_at,
	soft_deleted_at`

func (s *MetadataStore) CreateRoot(ctx context.Context, root spec.ArtifactRoot) error {
	if err := spec.ValidateArtifactRoot(root); err != nil {
		return fmt.Errorf("validate root for persistence: %w", err)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO artifact_roots (
			root_id, kind, display_name, description, enabled,
			data_schema_id, data_json, created_at, modified_at, soft_deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(root.RootID),
		string(root.Kind),
		root.DisplayName,
		root.Description,
		boolToInt(root.Enabled),
		string(root.DataSchemaID),
		[]byte(root.Data),
		formatTime(root.CreatedAt),
		formatTime(root.ModifiedAt),
		nullableTime(root.SoftDeletedAt),
	)
	if err != nil {
		return sqliteError(fmt.Errorf("insert root: %w", err))
	}
	return nil
}

func (s *MetadataStore) GetRoot(
	ctx context.Context,
	rootID spec.RootID,
	includeSoftDeleted bool,
) (spec.ArtifactRoot, error) {
	query := `SELECT ` + rootColumns + ` FROM artifact_roots WHERE root_id = ?`
	arguments := []any{string(rootID)}
	if !includeSoftDeleted {
		query += noSoftDeletedSuffix
	}
	root, err := scanRoot(s.db.QueryRowContext(ctx, query, arguments...))
	if errors.Is(err, sql.ErrNoRows) {
		return spec.ArtifactRoot{}, fmt.Errorf("%w: root %q", spec.ErrNotFound, rootID)
	}
	if err != nil {
		return spec.ArtifactRoot{}, err
	}
	return root, nil
}

func (s *MetadataStore) ListRoots(ctx context.Context, includeSoftDeleted bool) ([]spec.ArtifactRoot, error) {
	query := `SELECT ` + rootColumns + ` FROM artifact_roots`
	if !includeSoftDeleted {
		query += ` WHERE soft_deleted_at IS NULL`
	}
	query += ` ORDER BY modified_at DESC, root_id ASC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list roots: %w", err)
	}
	defer rows.Close()

	roots := make([]spec.ArtifactRoot, 0)
	for rows.Next() {
		root, err := scanRoot(rows)
		if err != nil {
			return nil, err
		}
		roots = append(roots, root)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roots: %w", err)
	}
	return roots, nil
}

func (s *MetadataStore) UpdateRoot(ctx context.Context, root spec.ArtifactRoot) error {
	if err := spec.ValidateArtifactRoot(root); err != nil {
		return fmt.Errorf("validate root for persistence: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE artifact_roots
		   SET display_name = ?,
		       description = ?,
		       enabled = ?,
		       data_schema_id = ?,
		       data_json = ?,
		       modified_at = ?,
		       soft_deleted_at = ?
		 WHERE root_id = ?`,
		root.DisplayName,
		root.Description,
		boolToInt(root.Enabled),
		string(root.DataSchemaID),
		[]byte(root.Data),
		formatTime(root.ModifiedAt),
		nullableTime(root.SoftDeletedAt),
		string(root.RootID),
	)
	if err != nil {
		return sqliteError(fmt.Errorf("update root: %w", err))
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect root update: %w", err)
	}
	if changed == 0 {
		return fmt.Errorf("%w: root %q", spec.ErrNotFound, root.RootID)
	}
	return nil
}

func scanRoot(scanner sqlScanner) (spec.ArtifactRoot, error) {
	row := artifactRootRow{}
	if err := scanner.Scan(row.destinations()...); err != nil {
		return spec.ArtifactRoot{}, err
	}
	created, err := parseRequiredTime("root.createdAt", row.CreatedAt)
	if err != nil {
		return spec.ArtifactRoot{}, err
	}
	modified, err := parseRequiredTime("root.modifiedAt", row.ModifiedAt)
	if err != nil {
		return spec.ArtifactRoot{}, err
	}
	deleted, err := parseNullableTime("root.softDeletedAt", row.SoftDeletedAt)
	if err != nil {
		return spec.ArtifactRoot{}, err
	}
	root := spec.ArtifactRoot{
		RootID:        spec.RootID(row.RootID),
		Kind:          spec.RootKind(row.Kind),
		DisplayName:   row.DisplayName,
		Description:   row.Description,
		Enabled:       row.Enabled != 0,
		DataSchemaID:  spec.SchemaID(row.DataSchemaID),
		Data:          append([]byte(nil), row.Data...),
		CreatedAt:     created,
		ModifiedAt:    modified,
		SoftDeletedAt: deleted,
	}
	if err := spec.ValidateArtifactRoot(root); err != nil {
		return spec.ArtifactRoot{}, fmt.Errorf("invalid persisted root %q: %w", row.RootID, err)
	}
	return root, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
