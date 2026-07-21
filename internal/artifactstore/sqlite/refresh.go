package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
)

type Publisher struct {
	store *Store
}

func (s *Store) Publisher() *Publisher {
	return &Publisher{store: s}
}

func (p *Publisher) Publish(
	ctx context.Context,
	publication refresh.Publication,
) (catalog.Snapshot, error) {
	if err := publication.Validate(); err != nil {
		return catalog.Snapshot{}, err
	}

	tx, err := p.store.db.BeginTx(ctx, nil)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var currentRootRevision uint64
	var enabled int
	var deletedAt sql.NullInt64
	err = tx.QueryRowContext(
		ctx,
		`SELECT revision, enabled, deleted_at
		 FROM artifact_roots WHERE id = ?`,
		string(publication.RootID),
	).Scan(&currentRootRevision, &enabled, &deletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: root %q",
			artifactstore.ErrNotFound,
			publication.RootID,
		)
	}
	if err != nil {
		return catalog.Snapshot{}, err
	}
	if currentRootRevision != publication.ExpectedRootRevision ||
		enabled == 0 ||
		deletedAt.Valid {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: root changed during refresh",
			artifactstore.ErrConflict,
		)
	}

	currentSourceRevisions, err := currentAttachedSourceRevisions(
		ctx,
		tx,
		publication.RootID,
	)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	if !maps.Equal(
		currentSourceRevisions,
		publication.ExpectedSourceRevisions,
	) {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: attached sources changed during refresh",
			artifactstore.ErrConflict,
		)
	}

	sourceRevisionsRaw, err := encodeJSON(publication.ExpectedSourceRevisions)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	sourceGenerationsRaw, err := encodeJSON(publication.SourceGenerations)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	diagnosticsRaw, err := encodeJSON(publication.Diagnostics)
	if err != nil {
		return catalog.Snapshot{}, err
	}

	var currentCatalogRevision uint64
	err = tx.QueryRowContext(
		ctx,
		`SELECT revision FROM artifact_current_catalogs WHERE root_id = ?`,
		string(publication.RootID),
	).Scan(&currentCatalogRevision)
	if errors.Is(err, sql.ErrNoRows) {
		currentCatalogRevision = 0
	} else if err != nil {
		return catalog.Snapshot{}, err
	}
	nextCatalogRevision := currentCatalogRevision + 1

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO artifact_current_catalogs (
			root_id, revision, root_revision, source_revisions_json,
			source_generations_json, published_at, diagnostics_json
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(root_id) DO UPDATE SET
			revision = excluded.revision,
			root_revision = excluded.root_revision,
			source_revisions_json = excluded.source_revisions_json,
			source_generations_json = excluded.source_generations_json,
			published_at = excluded.published_at,
			diagnostics_json = excluded.diagnostics_json`,
		string(publication.RootID),
		nextCatalogRevision,
		publication.ExpectedRootRevision,
		sourceRevisionsRaw,
		sourceGenerationsRaw,
		timeValue(publication.PublishedAt),
		diagnosticsRaw,
	)
	if err != nil {
		return catalog.Snapshot{}, sqliteError(err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_current_occurrences WHERE root_id = ?`,
		string(publication.RootID),
	); err != nil {
		return catalog.Snapshot{}, err
	}

	for _, occurrence := range publication.Occurrences {
		diagnostics, err := encodeJSON(occurrence.Diagnostics)
		if err != nil {
			return catalog.Snapshot{}, err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO artifact_current_occurrences (
				root_id, source_id, locator, subresource_locator,
				kind, logical_name, logical_version,
				definition_digest, source_content_digest, decoder_id,
				state, diagnostics_json, observed_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			string(publication.RootID),
			string(occurrence.Key.SourceID),
			string(occurrence.Key.Locator),
			string(occurrence.Key.SubresourceLocator),
			string(occurrence.Kind),
			string(occurrence.LogicalName),
			string(occurrence.LogicalVersion),
			nullableDigest(occurrence.DefinitionDigest),
			nullableDigest(occurrence.SourceContentDigest),
			string(occurrence.DecoderID),
			string(occurrence.State),
			diagnostics,
			timeValue(occurrence.ObservedAt),
		); err != nil {
			return catalog.Snapshot{}, sqliteError(err)
		}
	}

	for _, value := range publication.RecordCreates {
		if err := insertRecordTx(ctx, tx, value); err != nil {
			return catalog.Snapshot{}, err
		}
	}
	for _, update := range publication.RecordUpdates {
		if err := updateRecordTx(
			ctx,
			tx,
			update.Record,
			update.ExpectedRevision,
		); err != nil {
			return catalog.Snapshot{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return catalog.Snapshot{}, err
	}

	snapshot := catalog.Snapshot{
		RootID:            publication.RootID,
		Revision:          nextCatalogRevision,
		RootRevision:      publication.ExpectedRootRevision,
		SourceRevisions:   publication.ExpectedSourceRevisions,
		SourceGenerations: publication.SourceGenerations,
		PublishedAt:       publication.PublishedAt,
		Diagnostics:       publication.Diagnostics,
		Occurrences:       publication.Occurrences,
	}
	if err := snapshot.Validate(); err != nil {
		return catalog.Snapshot{}, err
	}
	return snapshot, nil
}

func currentAttachedSourceRevisions(
	ctx context.Context,
	tx *sql.Tx,
	rootID artifactstore.RootID,
) (map[artifactstore.SourceID]uint64, error) {
	rows, err := tx.QueryContext(
		ctx,
		`SELECT s.id, s.revision
		 FROM artifact_attachments a
		 JOIN artifact_sources s ON s.id = a.source_id
		 WHERE a.root_id = ?
		 ORDER BY s.id`,
		string(rootID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	output := make(map[artifactstore.SourceID]uint64)
	for rows.Next() {
		var id string
		var revision uint64
		if err := rows.Scan(&id, &revision); err != nil {
			return nil, err
		}
		output[artifactstore.SourceID(id)] = revision
	}
	return output, rows.Err()
}

func insertRecordTx(
	ctx context.Context,
	tx *sql.Tx,
	value record.Record,
) error {
	diagnostics, err := encodeJSON(value.Diagnostics)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO artifact_records (
			id, root_id, source_id, locator, subresource_locator,
			kind, name, enabled, mode, pinned_definition_digest,
			resolved_definition_digest, data_json, state,
			diagnostics_json, revision, created_at, modified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(value.ID),
		string(value.RootID),
		string(value.Occurrence.SourceID),
		string(value.Occurrence.Locator),
		string(value.Occurrence.SubresourceLocator),
		string(value.Kind),
		value.Name,
		boolInt(value.Enabled),
		string(value.Mode),
		nullableDigest(value.PinnedDefinition),
		nullableDigest(value.ResolvedDefinition),
		[]byte(value.Data),
		string(value.State),
		diagnostics,
		value.Revision,
		timeValue(value.CreatedAt),
		timeValue(value.ModifiedAt),
	)
	return sqliteError(err)
}

func updateRecordTx(
	ctx context.Context,
	tx *sql.Tx,
	value record.Record,
	expectedRevision uint64,
) error {
	diagnostics, err := encodeJSON(value.Diagnostics)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_records
		 SET resolved_definition_digest = ?,
		     state = ?,
		     diagnostics_json = ?,
		     revision = ?,
		     modified_at = ?
		 WHERE id = ? AND root_id = ? AND revision = ?`,
		nullableDigest(value.ResolvedDefinition),
		string(value.State),
		diagnostics,
		value.Revision,
		timeValue(value.ModifiedAt),
		string(value.ID),
		string(value.RootID),
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
			"%w: record %q changed during refresh",
			artifactstore.ErrConflict,
			value.ID,
		)
	}
	return nil
}

var _ refresh.Publisher = (*Publisher)(nil)
