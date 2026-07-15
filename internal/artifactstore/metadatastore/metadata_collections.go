package metadatastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

const (
	noSoftDeletedSuffix = " AND soft_deleted_at IS NULL"

	collectionColumns = `
	collection_id,
	root_id,
	kind,
	slug,
	display_name,
	description,
	enabled,
	data_schema_id,
	data_json,
	created_at,
	modified_at,
	soft_deleted_at`
)

func (s *MetadataStore) CreateCollection(ctx context.Context, collection spec.ArtifactCollection) error {
	if err := validate.ValidateArtifactCollection(collection); err != nil {
		return fmt.Errorf("validate collection for persistence: %w", err)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO artifact_collections (
			collection_id, root_id, kind, slug, display_name, description, enabled,
			data_schema_id, data_json, created_at, modified_at, soft_deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(collection.CollectionID),
		string(collection.RootID),
		string(collection.Kind),
		string(collection.Slug),
		collection.DisplayName,
		collection.Description,
		boolToInt(collection.Enabled),
		string(collection.DataSchemaID),
		[]byte(collection.Data),
		formatTime(collection.CreatedAt),
		formatTime(collection.ModifiedAt),
		nullableTime(collection.SoftDeletedAt),
	)
	if err != nil {
		return sqliteError(fmt.Errorf("insert collection: %w", err))
	}
	return nil
}

func (s *MetadataStore) GetCollection(
	ctx context.Context,
	collectionID spec.CollectionID,
	includeSoftDeleted bool,
) (spec.ArtifactCollection, error) {
	query := `SELECT ` + collectionColumns + ` FROM artifact_collections WHERE collection_id = ?`
	arguments := []any{string(collectionID)}
	if !includeSoftDeleted {
		query += noSoftDeletedSuffix
	}
	collection, err := scanCollection(s.db.QueryRowContext(ctx, query, arguments...))
	if errors.Is(err, sql.ErrNoRows) {
		return spec.ArtifactCollection{}, fmt.Errorf("%w: collection %q", spec.ErrNotFound, collectionID)
	}
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	return collection, nil
}

func (s *MetadataStore) GetCollectionByRootSlug(
	ctx context.Context,
	rootID spec.RootID,
	slug spec.CollectionSlug,
	includeSoftDeleted bool,
) (spec.ArtifactCollection, error) {
	query := `SELECT ` + collectionColumns + ` FROM artifact_collections WHERE root_id = ? AND slug = ?`
	arguments := []any{string(rootID), string(slug)}
	if !includeSoftDeleted {
		query += noSoftDeletedSuffix
	}
	collection, err := scanCollection(s.db.QueryRowContext(ctx, query, arguments...))
	if errors.Is(err, sql.ErrNoRows) {
		return spec.ArtifactCollection{}, fmt.Errorf("%w: collection root %q slug %q", spec.ErrNotFound, rootID, slug)
	}
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	return collection, nil
}

func (s *MetadataStore) ListCollections(
	ctx context.Context,
	rootID spec.RootID,
	includeSoftDeleted bool,
) ([]spec.ArtifactCollection, error) {
	query := `SELECT ` + collectionColumns + ` FROM artifact_collections WHERE root_id = ?`
	arguments := []any{string(rootID)}
	if !includeSoftDeleted {
		query += noSoftDeletedSuffix
	}
	query += ` ORDER BY modified_at DESC, collection_id ASC`
	rows, err := s.db.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	defer rows.Close()

	collections := make([]spec.ArtifactCollection, 0)
	for rows.Next() {
		collection, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		collections = append(collections, collection)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate collections: %w", err)
	}
	return collections, nil
}

func (s *MetadataStore) UpdateCollection(
	ctx context.Context,
	collection spec.ArtifactCollection,
	expectedModifiedAt time.Time,
) error {
	if err := validate.ValidateArtifactCollection(collection); err != nil {
		return fmt.Errorf("validate collection for persistence: %w", err)
	}
	if err := validateExpectedModifiedAt("collection", expectedModifiedAt); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, updateCollectionSQL,
		collection.DisplayName,
		collection.Description,
		boolToInt(collection.Enabled),
		string(collection.DataSchemaID),
		string(collection.Data),
		formatTime(collection.ModifiedAt),
		nullableTime(collection.SoftDeletedAt),
		string(collection.CollectionID),
		formatTime(expectedModifiedAt),
	)
	if err != nil {
		return sqliteError(fmt.Errorf("update collection: %w", err))
	}
	return optimisticMutationResult(result, "collection "+string(collection.CollectionID))
}

func (s *MetadataStore) CountRecordsInCollection(ctx context.Context, collectionID spec.CollectionID) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM artifact_records WHERE collection_id = ?`,
		string(collectionID),
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count collection records: %w", err)
	}
	return count, nil
}

func scanCollection(scanner sqlScanner) (spec.ArtifactCollection, error) {
	row := artifactCollectionRow{}
	if err := scanner.Scan(row.destinations()...); err != nil {
		return spec.ArtifactCollection{}, err
	}
	created, err := parseRequiredTime("collection.createdAt", row.CreatedAt)
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	modified, err := parseRequiredTime("collection.modifiedAt", row.ModifiedAt)
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	deleted, err := parseNullableTime("collection.softDeletedAt", row.SoftDeletedAt)
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	collection := spec.ArtifactCollection{
		CollectionID:  spec.CollectionID(row.CollectionID),
		RootID:        spec.RootID(row.RootID),
		Kind:          spec.CollectionKind(row.Kind),
		Slug:          spec.CollectionSlug(row.Slug),
		DisplayName:   row.DisplayName,
		Description:   row.Description,
		Enabled:       row.Enabled != 0,
		DataSchemaID:  spec.SchemaID(row.DataSchemaID),
		Data:          append([]byte(nil), row.Data...),
		CreatedAt:     created,
		ModifiedAt:    modified,
		SoftDeletedAt: deleted,
	}
	if err := validate.ValidateArtifactCollection(collection); err != nil {
		return spec.ArtifactCollection{}, fmt.Errorf(
			"invalid persisted collection %q: %w",
			row.CollectionID,
			err,
		)
	}
	return collection, nil
}
