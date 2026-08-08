package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func (s *Store) putCollectionDocumentBinding(
	ctx context.Context,
	value shareable.CollectionDocumentBinding,
) error {
	if err := value.Validate(); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	current, err := getActiveCollectionTx(ctx, tx, value.Collection)
	if err != nil {
		return err
	}
	if current.Kind != value.Key.Kind {
		return fmt.Errorf(
			"%w: shareable collection kind %q does not match local collection kind %q",
			basespec.ErrInvalid,
			value.Key.Kind,
			current.Kind,
		)
	}

	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO artifact_collection_shareable_documents (
			root_id, collection_id, kind, schema_id, schema_version, digest,
			bound_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(root_id, collection_id) DO NOTHING`,
		string(value.Collection.RootID),
		string(value.Collection.CollectionID),
		string(value.Key.Kind),
		string(value.Key.SchemaID),
		value.Key.SchemaVersion,
		string(value.Digest),
		time.Now().UTC().UnixNano(),
	)
	if err != nil {
		return sqliteError(err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		var (
			kind          string
			schemaID      string
			schemaVersion string
			digest        string
		)
		err := tx.QueryRowContext(
			ctx,
			`SELECT kind, schema_id, schema_version, digest
			 FROM artifact_collection_shareable_documents
			 WHERE root_id = ? AND collection_id = ?`,
			string(value.Collection.RootID),
			string(value.Collection.CollectionID),
		).Scan(&kind, &schemaID, &schemaVersion, &digest)
		if err != nil {
			return err
		}
		if kind != string(value.Key.Kind) ||
			schemaID != string(value.Key.SchemaID) ||
			schemaVersion != value.Key.SchemaVersion ||
			digest != string(value.Digest) {
			return fmt.Errorf(
				"%w: collection %q already has a different shareable document",
				basespec.ErrConflict,
				value.Collection.CollectionID,
			)
		}
	}
	if changed != 0 && changed != 1 {
		return fmt.Errorf(
			"%w: unexpected shareable document binding write count",
			basespec.ErrConflict,
		)
	}
	return sqliteError(tx.Commit())
}

func (s *Store) getCollectionDocumentBinding(
	ctx context.Context,
	ref collection.CollectionRef,
) (shareable.CollectionDocumentBinding, error) {
	if err := ref.Validate(); err != nil {
		return shareable.CollectionDocumentBinding{}, err
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return shareable.CollectionDocumentBinding{}, err
	}
	defer func() { _ = tx.Rollback() }()

	current, err := getActiveCollectionTx(ctx, tx, ref)
	if err != nil {
		return shareable.CollectionDocumentBinding{}, err
	}

	var (
		kind          string
		schemaID      string
		schemaVersion string
		digest        string
	)
	err = tx.QueryRowContext(
		ctx,
		`SELECT kind, schema_id, schema_version, digest
		 FROM artifact_collection_shareable_documents
		 WHERE root_id = ? AND collection_id = ?`,
		string(ref.RootID),
		string(ref.CollectionID),
	).Scan(&kind, &schemaID, &schemaVersion, &digest)
	if errors.Is(err, sql.ErrNoRows) {
		return shareable.CollectionDocumentBinding{}, fmt.Errorf(
			"%w: collection %q",
			basespec.ErrShareableDocumentNotFound,
			ref.CollectionID,
		)
	}
	if err != nil {
		return shareable.CollectionDocumentBinding{}, err
	}

	value := shareable.CollectionDocumentBinding{
		Collection: ref,
		Key: shareable.SchemaKey{
			Entity:        shareable.EntityCollection,
			Kind:          basespec.CollectionKind(kind),
			SchemaID:      basespec.SchemaID(schemaID),
			SchemaVersion: schemaVersion,
		},
		Digest: cryptoutil.Digest(digest),
	}
	if err := value.Validate(); err != nil {
		return shareable.CollectionDocumentBinding{}, fmt.Errorf(
			"%w: invalid persisted shareable collection binding: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if current.Kind != value.Key.Kind {
		return shareable.CollectionDocumentBinding{}, fmt.Errorf(
			"%w: persisted shareable collection kind does not match local collection",
			basespec.ErrInvalid,
		)
	}
	if err := tx.Commit(); err != nil {
		return shareable.CollectionDocumentBinding{}, err
	}
	return value, nil
}
