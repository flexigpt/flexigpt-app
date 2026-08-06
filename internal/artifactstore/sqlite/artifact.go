package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const artifactColumns = `
	id, root_id, collection_id, source_id, locator, subresource_locator,
	kind, name, enabled, adoption, resolved_definition_digest, data_json,
	state, diagnostics_json, revision, created_at, modified_at`

const activeArtifactColumns = `
	a.id, a.root_id, a.collection_id, a.source_id, a.locator,
	a.subresource_locator, a.kind, a.name, a.enabled, a.adoption,
	a.resolved_definition_digest, a.data_json, a.state,
	a.diagnostics_json, a.revision, a.created_at, a.modified_at`

const suppressionColumns = `
	root_id, collection_id, source_id, locator, subresource_locator,
	expected_kind, revision, created_at, modified_at`

func (s *Store) getArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	if err := ref.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if err := s.requireActiveRoot(ctx, ref.RootID); err != nil {
		return artifact.Artifact{}, err
	}

	value, err := getArtifactTx(ctx, s.db, ref)
	if err != nil {
		return artifact.Artifact{}, err
	}
	return value.Clone(), nil
}

func (s *Store) listArtifactsByCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]artifact.Artifact, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveCollectionTx(ctx, tx, ref); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(
		ctx,
		`SELECT `+artifactColumns+`
		 FROM artifact_artifacts
		 WHERE root_id = ? AND collection_id = ?
		 ORDER BY modified_at DESC, id ASC`,
		string(ref.RootID),
		string(ref.CollectionID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make([]artifact.Artifact, 0)
	for rows.Next() {
		value, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value.Clone())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return values, nil
}

func (s *Store) listSuppressions(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]artifact.Suppression, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveCollectionTx(ctx, tx, ref); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(
		ctx,
		`SELECT `+suppressionColumns+`
		 FROM artifact_suppressions
		 WHERE root_id = ? AND collection_id = ?
		 ORDER BY source_id, locator, subresource_locator, expected_kind`,
		string(ref.RootID),
		string(ref.CollectionID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make([]artifact.Suppression, 0)
	for rows.Next() {
		value, err := scanSuppression(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return values, nil
}

func (s *Store) updateArtifact(
	ctx context.Context,
	value artifact.Artifact,
	expectedRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 ||
		value.Revision != expectedRevision+1 {
		return fmt.Errorf("%w: invalid artifact update", basespec.ErrInvalid)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	ref := collection.CollectionRef{
		RootID:       value.RootID,
		CollectionID: value.CollectionID,
	}
	if _, err := getActiveCollectionTx(ctx, tx, ref); err != nil {
		return err
	}
	current, err := getArtifactTx(ctx, tx, value.Ref())
	if err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return basespec.ErrConflict
	}
	if !value.ModifiedAt.After(current.ModifiedAt) {
		return fmt.Errorf(
			"%w: artifact update time must advance current state",
			basespec.ErrInvalid,
		)
	}
	if !sameArtifactImmutableFields(current, value) {
		return fmt.Errorf(
			"%w: artifact update attempted to change source-derived fields",
			basespec.ErrInvalid,
		)
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_artifacts
		 SET name = ?,
		     enabled = ?,
		     data_json = ?,
		     revision = ?,
		     modified_at = ?
		 WHERE id = ?
		   AND root_id = ?
		   AND collection_id = ?
		   AND revision = ?`,
		value.Name,
		boolInt(value.Enabled),
		[]byte(value.Data),
		value.Revision,
		timeValue(value.ModifiedAt),
		string(value.ID),
		string(value.RootID),
		string(value.CollectionID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(result, "artifact changed during update"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) createAdoptedArtifact(
	ctx context.Context,
	value artifact.Artifact,
	expectedCollectionRevision uint64,
	expectedCatalogRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if value.Revision != 1 ||
		expectedCollectionRevision == 0 ||
		expectedCatalogRevision == 0 ||
		value.Adoption != artifact.AdoptionObserved ||
		value.State != artifact.StateAvailable ||
		value.ResolvedDefinition == nil {
		return fmt.Errorf("%w: invalid observed artifact creation", basespec.ErrInvalid)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	ref := collection.CollectionRef{
		RootID:       value.RootID,
		CollectionID: value.CollectionID,
	}
	currentCollection, err := getActiveCollectionTx(ctx, tx, ref)
	if err != nil {
		return err
	}
	if !currentCollection.Enabled ||
		currentCollection.Revision != expectedCollectionRevision {
		return basespec.ErrConflict
	}
	if err := requireCurrentCatalogTx(
		ctx,
		tx,
		ref,
		expectedCatalogRevision,
		currentCollection.Revision,
	); err != nil {
		return err
	}
	if err := requireAttachedSourceTx(ctx, tx, ref, value.Binding.SourceID); err != nil {
		return err
	}
	if err := requireAdoptableOccurrenceTx(ctx, tx, value); err != nil {
		return err
	}
	if err := insertArtifactTx(ctx, tx, value); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) createPinnedArtifact(
	ctx context.Context,
	value artifact.Artifact,
	expectedCollectionRevision uint64,
	expectedCatalogRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if value.Revision != 1 ||
		expectedCollectionRevision == 0 ||
		value.Adoption != artifact.AdoptionPinned ||
		(expectedCatalogRevision == 0 &&
			(value.State != artifact.StateMissing ||
				value.ResolvedDefinition != nil)) {
		return fmt.Errorf("%w: invalid pinned artifact creation", basespec.ErrInvalid)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	ref := collection.CollectionRef{
		RootID:       value.RootID,
		CollectionID: value.CollectionID,
	}
	currentCollection, err := getActiveCollectionTx(ctx, tx, ref)
	if err != nil {
		return err
	}
	if !currentCollection.Enabled ||
		currentCollection.Revision != expectedCollectionRevision {
		return basespec.ErrConflict
	}
	if err := requireAttachedSourceTx(ctx, tx, ref, value.Binding.SourceID); err != nil {
		return err
	}
	if expectedCatalogRevision != 0 {
		if err := requireCurrentCatalogTx(
			ctx,
			tx,
			ref,
			expectedCatalogRevision,
			currentCollection.Revision,
		); err != nil {
			return err
		}
		if err := requirePinnedSourceStateTx(ctx, tx, value); err != nil {
			return err
		}
	} else if err := requireCatalogUnavailableOrStaleTx(
		ctx,
		tx,
		ref,
		currentCollection.Revision,
	); err != nil {
		return err
	}
	if err := insertArtifactTx(ctx, tx, value); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) unadoptArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	suppression *artifact.Suppression,
) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected artifact revision is required",
			basespec.ErrInvalid,
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveRootTx(ctx, tx, ref.RootID); err != nil {
		return err
	}
	current, err := getArtifactTx(ctx, tx, ref)
	if err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return basespec.ErrConflict
	}
	if current.Adoption != artifact.AdoptionObserved {
		return fmt.Errorf(
			"%w: only observed artifacts can be unadopted",
			basespec.ErrConflict,
		)
	}

	collectionRef := collection.CollectionRef{
		RootID:       current.RootID,
		CollectionID: current.CollectionID,
	}
	if _, err := getActiveCollectionTx(ctx, tx, collectionRef); err != nil {
		return err
	}
	if suppression != nil {
		if err := suppression.Validate(); err != nil {
			return err
		}
		if suppression.RootID != current.RootID ||
			suppression.CollectionID != current.CollectionID ||
			suppression.Binding != current.Binding {
			return fmt.Errorf(
				"%w: suppression does not match artifact binding",
				basespec.ErrInvalid,
			)
		}
		if err := requireAttachedSourceTx(
			ctx,
			tx,
			collectionRef,
			suppression.Binding.SourceID,
		); err != nil {
			return err
		}
		if err := insertSuppressionTx(ctx, tx, *suppression, true); err != nil {
			return err
		}
	}

	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_artifacts
		 WHERE id = ? AND root_id = ? AND revision = ?`,
		string(ref.ArtifactID),
		string(ref.RootID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(result, "artifact changed during unadopt"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) createSuppression(
	ctx context.Context,
	value artifact.Suppression,
	expectedCollectionRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if expectedCollectionRevision == 0 || value.Revision != 1 {
		return fmt.Errorf("%w: invalid suppression creation", basespec.ErrInvalid)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	ref := collection.CollectionRef{
		RootID:       value.RootID,
		CollectionID: value.CollectionID,
	}
	currentCollection, err := getActiveCollectionTx(ctx, tx, ref)
	if err != nil {
		return err
	}
	if !currentCollection.Enabled ||
		currentCollection.Revision != expectedCollectionRevision {
		return basespec.ErrConflict
	}
	if err := requireAttachedSourceTx(ctx, tx, ref, value.Binding.SourceID); err != nil {
		return err
	}
	if err := insertSuppressionTx(ctx, tx, value, false); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) deleteSuppression(
	ctx context.Context,
	ref collection.CollectionRef,
	binding artifact.SourceBinding,
	expectedRevision uint64,
) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if err := binding.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected suppression revision is required",
			basespec.ErrInvalid,
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveCollectionTx(ctx, tx, ref); err != nil {
		return err
	}
	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_suppressions
		 WHERE root_id = ?
		   AND collection_id = ?
		   AND source_id = ?
		   AND locator = ?
		   AND subresource_locator = ?
		   AND expected_kind = ?
		   AND revision = ?`,
		string(ref.RootID),
		string(ref.CollectionID),
		string(binding.SourceID),
		string(binding.Locator),
		string(binding.SubresourceLocator),
		string(binding.ExpectedKind),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(result, "suppression changed during delete"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) purgeArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected artifact revision is required",
			basespec.ErrInvalid,
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveRootTx(ctx, tx, ref.RootID); err != nil {
		return err
	}
	current, err := getArtifactTx(ctx, tx, ref)
	if err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return basespec.ErrConflict
	}
	collectionRef := collection.CollectionRef{
		RootID:       current.RootID,
		CollectionID: current.CollectionID,
	}
	if _, err := getActiveCollectionTx(
		ctx,
		tx,
		collectionRef,
	); err != nil {
		return err
	}

	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_artifacts
		 WHERE id = ? AND root_id = ? AND revision = ?`,
		string(ref.ArtifactID),
		string(ref.RootID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"artifact changed during purge",
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) purgeArtifactAndSuppress(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	suppression artifact.Suppression,
) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if err := suppression.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 || suppression.Revision != 1 {
		return fmt.Errorf(
			"%w: invalid artifact purge with suppression",
			basespec.ErrInvalid,
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveRootTx(ctx, tx, ref.RootID); err != nil {
		return err
	}
	current, err := getArtifactTx(ctx, tx, ref)
	if err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return basespec.ErrConflict
	}

	collectionRef := collection.CollectionRef{
		RootID:       current.RootID,
		CollectionID: current.CollectionID,
	}
	if _, err := getActiveCollectionTx(ctx, tx, collectionRef); err != nil {
		return err
	}
	if suppression.RootID != current.RootID ||
		suppression.CollectionID != current.CollectionID ||
		suppression.Binding != current.Binding {
		return fmt.Errorf(
			"%w: suppression does not match artifact binding",
			basespec.ErrInvalid,
		)
	}
	if err := requireAttachedSourceTx(
		ctx,
		tx,
		collectionRef,
		suppression.Binding.SourceID,
	); err != nil {
		return err
	}

	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_artifacts
		 WHERE id = ? AND root_id = ? AND revision = ?`,
		string(ref.ArtifactID),
		string(ref.RootID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"artifact changed during purge with suppression",
	); err != nil {
		return err
	}

	// This runs after the delete but inside the same transaction. A refresh
	// cannot observe a gap in which the binding is neither represented nor
	// suppressed.
	if err := insertSuppressionTx(ctx, tx, suppression, false); err != nil {
		return err
	}
	return tx.Commit()
}

// insertArtifactTx is intentionally a pure insert. Callers must establish
// Collection membership, current catalog, and source-binding preconditions
// before invoking it.
func insertArtifactTx(
	ctx context.Context,
	tx *sql.Tx,
	value artifact.Artifact,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if err := requireNoSuppressionTx(
		ctx,
		tx,
		value.RootID,
		value.CollectionID,
		value.Binding,
	); err != nil {
		return err
	}
	diagnostics, err := encodeJSON(value.Diagnostics)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO artifact_artifacts (
			id, root_id, collection_id, source_id, locator, subresource_locator,
			kind, name, enabled, adoption, resolved_definition_digest, data_json,
			state, diagnostics_json, revision, created_at, modified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(value.ID),
		string(value.RootID),
		string(value.CollectionID),
		string(value.Binding.SourceID),
		string(value.Binding.Locator),
		string(value.Binding.SubresourceLocator),
		string(value.Kind),
		value.Name,
		boolInt(value.Enabled),
		string(value.Adoption),
		nullableDigest(value.ResolvedDefinition),
		[]byte(value.Data),
		string(value.State),
		diagnostics,
		value.Revision,
		timeValue(value.CreatedAt),
		timeValue(value.ModifiedAt),
	)
	if err != nil {
		return sqliteError(err)
	}
	return nil
}

func insertSuppressionTx(
	ctx context.Context,
	tx *sql.Tx,
	value artifact.Suppression,
	ignoreExisting bool,
) error {
	if !ignoreExisting {
		if err := requireNoArtifactForBindingTx(
			ctx,
			tx,
			value.RootID,
			value.CollectionID,
			value.Binding,
		); err != nil {
			return err
		}
	}

	query := `INSERT INTO artifact_suppressions (
		root_id, collection_id, source_id, locator, subresource_locator,
		expected_kind, revision, created_at, modified_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if ignoreExisting {
		query += ` ON CONFLICT(
			root_id, collection_id, source_id,
			locator, subresource_locator, expected_kind
		) DO NOTHING`
	}

	_, err := tx.ExecContext(
		ctx,
		query,
		string(value.RootID),
		string(value.CollectionID),
		string(value.Binding.SourceID),
		string(value.Binding.Locator),
		string(value.Binding.SubresourceLocator),
		string(value.Binding.ExpectedKind),
		value.Revision,
		timeValue(value.CreatedAt),
		timeValue(value.ModifiedAt),
	)
	if err != nil {
		return sqliteError(err)
	}
	return nil
}

func requireNoSuppressionTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID basespec.RootID,
	collectionID basespec.CollectionID,
	binding artifact.SourceBinding,
) error {
	var exists int
	err := tx.QueryRowContext(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM artifact_suppressions
			WHERE root_id = ?
			  AND collection_id = ?
			  AND source_id = ?
			  AND locator = ?
			  AND subresource_locator = ?
			  AND expected_kind = ?
		)`,
		string(rootID),
		string(collectionID),
		string(binding.SourceID),
		string(binding.Locator),
		string(binding.SubresourceLocator),
		string(binding.ExpectedKind),
	).Scan(&exists)
	if err != nil {
		return err
	}
	if exists != 0 {
		return fmt.Errorf(
			"%w: source binding is explicitly suppressed",
			basespec.ErrSuppressed,
		)
	}
	return nil
}

func requireNoArtifactForBindingTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID basespec.RootID,
	collectionID basespec.CollectionID,
	binding artifact.SourceBinding,
) error {
	var exists int
	err := tx.QueryRowContext(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM artifact_artifacts
			WHERE root_id = ?
			  AND collection_id = ?
			  AND source_id = ?
			  AND locator = ?
			  AND subresource_locator = ?
			  AND kind = ?
		)`,
		string(rootID),
		string(collectionID),
		string(binding.SourceID),
		string(binding.Locator),
		string(binding.SubresourceLocator),
		string(binding.ExpectedKind),
	).Scan(&exists)
	if err != nil {
		return err
	}
	if exists != 0 {
		return fmt.Errorf(
			"%w: source binding is already represented by an Artifact",
			basespec.ErrConflict,
		)
	}
	return nil
}

func requireCurrentCatalogTx(
	ctx context.Context,
	tx *sql.Tx,
	ref collection.CollectionRef,
	expectedCatalogRevision uint64,
	expectedCollectionRevision uint64,
) error {
	var (
		revision               uint64
		collectionRevision     uint64
		attachmentRevisionsRaw []byte
		sourceRevisionsRaw     []byte
	)
	err := tx.QueryRowContext(
		ctx,
		`SELECT revision, collection_revision,
		        attachment_revisions_json, source_revisions_json
		 FROM artifact_current_catalogs
		 WHERE root_id = ? AND collection_id = ?`,
		string(ref.RootID),
		string(ref.CollectionID),
	).Scan(
		&revision,
		&collectionRevision,
		&attachmentRevisionsRaw,
		&sourceRevisionsRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf(
			"%w: collection %q has no current catalog",
			basespec.ErrCatalogUnavailable,
			ref.CollectionID,
		)
	}
	if err != nil {
		return err
	}
	if revision != expectedCatalogRevision {
		return fmt.Errorf(
			"%w: catalog changed during artifact adoption",
			basespec.ErrConflict,
		)
	}
	if collectionRevision != expectedCollectionRevision {
		return fmt.Errorf(
			"%w: catalog collection revision changed",
			basespec.ErrConflict,
		)
	}

	catalogAttachmentRevisions := map[basespec.SourceID]uint64{}
	catalogSourceRevisions := map[basespec.SourceID]uint64{}
	if err := decodeJSON(
		attachmentRevisionsRaw,
		&catalogAttachmentRevisions,
	); err != nil {
		return err
	}
	if err := decodeJSON(sourceRevisionsRaw, &catalogSourceRevisions); err != nil {
		return err
	}

	currentAttachments, currentSources, err := currentAttachmentSourceRevisionsTx(
		ctx,
		tx,
		ref,
	)
	if err != nil {
		return err
	}
	if !maps.Equal(catalogAttachmentRevisions, currentAttachments) ||
		!maps.Equal(catalogSourceRevisions, currentSources) {
		return fmt.Errorf(
			"%w: catalog metadata changed during artifact adoption",
			basespec.ErrConflict,
		)
	}
	return nil
}

// requireCatalogUnavailableOrStaleTx validates the zero catalog-revision pin
// path. A pinned Artifact may use that path only when no catalog exists or
// when the existing catalog is stale against current Collection metadata.
func requireCatalogUnavailableOrStaleTx(
	ctx context.Context,
	tx *sql.Tx,
	ref collection.CollectionRef,
	expectedCollectionRevision uint64,
) error {
	var (
		catalogCollectionRevision uint64
		attachmentRevisionsRaw    []byte
		sourceRevisionsRaw        []byte
	)
	err := tx.QueryRowContext(
		ctx,
		`SELECT collection_revision,
		        attachment_revisions_json, source_revisions_json
		 FROM artifact_current_catalogs
		 WHERE root_id = ? AND collection_id = ?`,
		string(ref.RootID),
		string(ref.CollectionID),
	).Scan(
		&catalogCollectionRevision,
		&attachmentRevisionsRaw,
		&sourceRevisionsRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	catalogAttachmentRevisions := map[basespec.SourceID]uint64{}
	catalogSourceRevisions := map[basespec.SourceID]uint64{}
	if err := decodeJSON(
		attachmentRevisionsRaw,
		&catalogAttachmentRevisions,
	); err != nil {
		return err
	}
	if err := decodeJSON(sourceRevisionsRaw, &catalogSourceRevisions); err != nil {
		return err
	}

	currentAttachments, currentSources, err := currentAttachmentSourceRevisionsTx(
		ctx,
		tx,
		ref,
	)
	if err != nil {
		return err
	}
	if catalogCollectionRevision == expectedCollectionRevision &&
		maps.Equal(catalogAttachmentRevisions, currentAttachments) &&
		maps.Equal(catalogSourceRevisions, currentSources) {
		return fmt.Errorf(
			"%w: current catalog requires an expected revision",
			basespec.ErrConflict,
		)
	}
	return nil
}

func requireAdoptableOccurrenceTx(
	ctx context.Context,
	tx *sql.Tx,
	value artifact.Artifact,
) error {
	if value.ResolvedDefinition == nil {
		return fmt.Errorf(
			"%w: adopted artifact has no resolved definition",
			basespec.ErrInvalid,
		)
	}

	var exists int
	err := tx.QueryRowContext(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM artifact_current_occurrences
			WHERE root_id = ?
			  AND collection_id = ?
			  AND source_id = ?
			  AND locator = ?
			  AND subresource_locator = ?
			  AND kind = ?
			  AND state = ?
			  AND definition_digest = ?
		)`,
		string(value.RootID),
		string(value.CollectionID),
		string(value.Binding.SourceID),
		string(value.Binding.Locator),
		string(value.Binding.SubresourceLocator),
		string(value.Kind),
		string(catalog.OccurrenceValid),
		string(*value.ResolvedDefinition),
	).Scan(&exists)
	if err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf(
			"%w: source occurrence is not currently adoptable",
			basespec.ErrReferenceUnresolved,
		)
	}
	return nil
}

// requirePinnedSourceStateTx prevents a current-catalog pin from storing a
// state which does not match the occurrence observed by the caller.
func requirePinnedSourceStateTx(
	ctx context.Context,
	tx *sql.Tx,
	value artifact.Artifact,
) error {
	observed, err := currentOccurrenceForBindingTx(ctx, tx, value)
	if err != nil {
		return err
	}
	expectedDigest, expectedState, expectedDiagnostics, err := artifact.DeriveSourceState(value, observed)
	if err != nil {
		return err
	}
	if value.State != expectedState ||
		!cryptoutil.IsDigestEqual(value.ResolvedDefinition, expectedDigest) ||
		!diagnostic.EqualDiagnostics(value.Diagnostics, expectedDiagnostics) {
		return fmt.Errorf(
			"%w: pinned artifact does not match the current source occurrence",
			basespec.ErrConflict,
		)
	}
	return nil
}

func currentOccurrenceForBindingTx(
	ctx context.Context,
	tx *sql.Tx,
	value artifact.Artifact,
) (*catalog.Occurrence, error) {
	occurrence, err := scanOccurrence(tx.QueryRowContext(
		ctx,
		`SELECT `+occurrenceColumns+`
		 FROM artifact_current_occurrences
		 WHERE root_id = ?
		   AND collection_id = ?
		   AND source_id = ?
		   AND locator = ?
		   AND subresource_locator = ?`,
		string(value.RootID),
		string(value.CollectionID),
		string(value.Binding.SourceID),
		string(value.Binding.Locator),
		string(value.Binding.SubresourceLocator),
	))
	if errors.Is(err, sql.ErrNoRows) {
		//nolint:nilnil // No rows.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &occurrence, nil
}

type rowQueryer interface {
	QueryRowContext(
		ctx context.Context,
		query string,
		args ...any,
	) *sql.Row
}

func getArtifactTx(
	ctx context.Context,
	queryer rowQueryer,
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	value, err := scanArtifact(queryer.QueryRowContext(
		ctx,
		`SELECT `+activeArtifactColumns+`
		 FROM artifact_artifacts a
		 JOIN artifact_collections c
		   ON c.root_id = a.root_id
		  AND c.id = a.collection_id
		  AND c.retired_at IS NULL
		 JOIN artifact_roots r
		   ON r.id = a.root_id
		  AND r.retired_at IS NULL
		 WHERE a.id = ? AND a.root_id = ?`,
		string(ref.ArtifactID),
		string(ref.RootID),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: artifact %q in root %q",
			basespec.ErrArtifactNotFound,
			ref.ArtifactID,
			ref.RootID,
		)
	}
	return value, err
}

func sameArtifactImmutableFields(
	current artifact.Artifact,
	next artifact.Artifact,
) bool {
	return current.ID == next.ID &&
		current.RootID == next.RootID &&
		current.CollectionID == next.CollectionID &&
		current.Binding == next.Binding &&
		current.Kind == next.Kind &&
		current.Adoption == next.Adoption &&
		cryptoutil.IsDigestEqual(
			current.ResolvedDefinition,
			next.ResolvedDefinition,
		) &&
		current.State == next.State &&
		diagnostic.EqualDiagnostics(current.Diagnostics, next.Diagnostics) &&
		current.CreatedAt.Equal(next.CreatedAt)
}

func scanArtifact(row scanner) (artifact.Artifact, error) {
	var (
		id, rootID, collectionID, sourceID string
		locator, subresource, kind         string
		name, adoption, state              string
		enabled                            int
		resolvedDefinition                 sql.NullString
		data, diagnosticsRaw               []byte
		revision                           uint64
		createdAt, modifiedAt              int64
	)
	if err := row.Scan(
		&id,
		&rootID,
		&collectionID,
		&sourceID,
		&locator,
		&subresource,
		&kind,
		&name,
		&enabled,
		&adoption,
		&resolvedDefinition,
		&data,
		&state,
		&diagnosticsRaw,
		&revision,
		&createdAt,
		&modifiedAt,
	); err != nil {
		return artifact.Artifact{}, err
	}

	diagnostics := []diagnostic.Diagnostic{}
	if err := decodeJSON(diagnosticsRaw, &diagnostics); err != nil {
		return artifact.Artifact{}, err
	}
	value := artifact.Artifact{
		ID:           basespec.ArtifactID(id),
		RootID:       basespec.RootID(rootID),
		CollectionID: basespec.CollectionID(collectionID),
		Binding: artifact.SourceBinding{
			SourceID:           basespec.SourceID(sourceID),
			Locator:            basespec.Locator(locator),
			SubresourceLocator: basespec.SubresourceLocator(subresource),
			ExpectedKind:       basespec.ArtifactKind(kind),
		},
		Kind:               basespec.ArtifactKind(kind),
		Name:               name,
		Enabled:            enabled != 0,
		Adoption:           artifact.AdoptionMode(adoption),
		ResolvedDefinition: parseDigest(resolvedDefinition),
		Data:               append(json.RawMessage(nil), data...),
		State:              artifact.State(state),
		Diagnostics:        diagnostics,
		Revision:           revision,
		CreatedAt:          parseTime(createdAt),
		ModifiedAt:         parseTime(modifiedAt),
	}
	if err := value.Validate(); err != nil {
		return artifact.Artifact{}, fmt.Errorf(
			"invalid persisted artifact %q: %w",
			id,
			err,
		)
	}
	return value, nil
}

func scanSuppression(row scanner) (artifact.Suppression, error) {
	var (
		rootID, collectionID, sourceID string
		locator, subresource, kind     string
		revision                       uint64
		createdAt, modifiedAt          int64
	)
	if err := row.Scan(
		&rootID,
		&collectionID,
		&sourceID,
		&locator,
		&subresource,
		&kind,
		&revision,
		&createdAt,
		&modifiedAt,
	); err != nil {
		return artifact.Suppression{}, err
	}

	value := artifact.Suppression{
		RootID:       basespec.RootID(rootID),
		CollectionID: basespec.CollectionID(collectionID),
		Binding: artifact.SourceBinding{
			SourceID:           basespec.SourceID(sourceID),
			Locator:            basespec.Locator(locator),
			SubresourceLocator: basespec.SubresourceLocator(subresource),
			ExpectedKind:       basespec.ArtifactKind(kind),
		},
		Revision:   revision,
		CreatedAt:  parseTime(createdAt),
		ModifiedAt: parseTime(modifiedAt),
	}
	if err := value.Validate(); err != nil {
		return artifact.Suppression{}, fmt.Errorf(
			"invalid persisted suppression %q/%q/%q: %w",
			rootID,
			collectionID,
			sourceID,
			err,
		)
	}
	return value, nil
}
