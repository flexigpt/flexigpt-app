package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
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

	occurrencesByKey := make(
		map[catalog.OccurrenceKey]catalog.Occurrence,
		len(publication.Occurrences),
	)
	for _, occurrence := range publication.Occurrences {
		occurrencesByKey[occurrence.Key] = occurrence
	}

	tx, err := p.store.db.BeginTx(ctx, nil)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	defer func() { _ = tx.Rollback() }()

	currentCollection, err := getActiveCollectionTx(ctx, tx, publication.Ref)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	if !currentCollection.Enabled ||
		currentCollection.Revision != publication.ExpectedCollectionRevision {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: collection changed or was disabled during refresh",
			basespec.ErrConflict,
		)
	}

	currentAttachments, currentSources, err := currentAttachmentSourceRevisionsTx(
		ctx,
		tx,
		publication.Ref,
	)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	if !maps.Equal(
		currentAttachments,
		publication.ExpectedAttachmentRevisions,
	) || !maps.Equal(
		currentSources,
		publication.ExpectedSourceRevisions,
	) {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: collection attachments or sources changed during refresh",
			basespec.ErrConflict,
		)
	}

	attachmentRevisionsRaw, err := encodeJSON(
		publication.ExpectedAttachmentRevisions,
	)
	if err != nil {
		return catalog.Snapshot{}, err
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
		`SELECT revision
		 FROM artifact_current_catalogs
		 WHERE root_id = ? AND collection_id = ?`,
		string(publication.Ref.RootID),
		string(publication.Ref.CollectionID),
	).Scan(&currentCatalogRevision)
	if errors.Is(err, sql.ErrNoRows) {
		currentCatalogRevision = 0
	} else if err != nil {
		return catalog.Snapshot{}, err
	}
	if currentCatalogRevision != publication.ExpectedCatalogRevision {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: catalog changed during refresh",
			basespec.ErrConflict,
		)
	}
	if currentCatalogRevision == ^uint64(0) {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: catalog revision is exhausted",
			basespec.ErrInvalid,
		)
	}
	nextCatalogRevision := currentCatalogRevision + 1

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO artifact_current_catalogs (
			root_id, collection_id, revision, collection_revision,
			attachment_revisions_json, source_revisions_json,
			source_generations_json, plan_fingerprint, decoder_fingerprint,
			published_at, diagnostics_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(root_id, collection_id) DO UPDATE SET
			revision = excluded.revision,
			collection_revision = excluded.collection_revision,
			attachment_revisions_json = excluded.attachment_revisions_json,
			source_revisions_json = excluded.source_revisions_json,
			source_generations_json = excluded.source_generations_json,
			plan_fingerprint = excluded.plan_fingerprint,
			decoder_fingerprint = excluded.decoder_fingerprint,
			published_at = excluded.published_at,
			diagnostics_json = excluded.diagnostics_json`,
		string(publication.Ref.RootID),
		string(publication.Ref.CollectionID),
		nextCatalogRevision,
		publication.ExpectedCollectionRevision,
		attachmentRevisionsRaw,
		sourceRevisionsRaw,
		sourceGenerationsRaw,
		string(publication.PlanFingerprint),
		string(publication.DecoderFingerprint),
		timeValue(publication.PublishedAt),
		diagnosticsRaw,
	)
	if err != nil {
		return catalog.Snapshot{}, sqliteError(err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_current_occurrences
		 WHERE root_id = ? AND collection_id = ?`,
		string(publication.Ref.RootID),
		string(publication.Ref.CollectionID),
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
				root_id, collection_id, source_id, locator, subresource_locator,
				kind, logical_name, logical_version,
				definition_digest, source_content_digest, decoder_id,
				state, diagnostics_json, observed_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			string(publication.Ref.RootID),
			string(publication.Ref.CollectionID),
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

	for _, value := range publication.ArtifactCreates {
		if err := requireAttachedSourceTx(
			ctx,
			tx,
			publication.Ref,
			value.Binding.SourceID,
		); err != nil {
			return catalog.Snapshot{}, err
		}
		if err := insertArtifactTx(ctx, tx, value); err != nil {
			return catalog.Snapshot{}, err
		}
	}
	// Artifact updates are source-derived state transitions. Validate them
	// against both the persisted Artifact binding and this publication's
	// occurrences before mutating metadata.
	for _, update := range publication.ArtifactUpdates {
		if err := updateArtifactSourceStateTx(
			ctx,
			tx,
			update,
			occurrencesByKey,
		); err != nil {
			return catalog.Snapshot{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return catalog.Snapshot{}, err
	}

	occurrences := make([]catalog.Occurrence, len(publication.Occurrences))
	for index, occurrence := range publication.Occurrences {
		occurrences[index] = catalog.CloneOccurrence(occurrence)
	}

	snapshot := catalog.Snapshot{
		RootID:              publication.Ref.RootID,
		CollectionID:        publication.Ref.CollectionID,
		Revision:            nextCatalogRevision,
		CollectionRevision:  publication.ExpectedCollectionRevision,
		AttachmentRevisions: maps.Clone(publication.ExpectedAttachmentRevisions),
		SourceRevisions:     maps.Clone(publication.ExpectedSourceRevisions),
		SourceGenerations:   maps.Clone(publication.SourceGenerations),
		PlanFingerprint:     publication.PlanFingerprint,
		DecoderFingerprint:  publication.DecoderFingerprint,
		PublishedAt:         publication.PublishedAt,
		Diagnostics:         diagnostic.CloneDiagnostics(publication.Diagnostics),
		Occurrences:         occurrences,
	}
	if err := snapshot.Validate(); err != nil {
		return catalog.Snapshot{}, err
	}
	return catalog.CloneSnapshot(snapshot), nil
}

func updateArtifactSourceStateTx(
	ctx context.Context,
	tx *sql.Tx,
	value artifact.SourceStateUpdate,
	occurrences map[catalog.OccurrenceKey]catalog.Occurrence,
) error {
	current, err := getArtifactTx(ctx, tx, artifact.ArtifactRef{
		RootID:     value.RootID,
		ArtifactID: value.ArtifactID,
	})
	if err != nil {
		return err
	}
	if current.CollectionID != value.CollectionID {
		return fmt.Errorf(
			"%w: source-derived artifact update belongs to another collection",
			basespec.ErrInvalid,
		)
	}
	if current.Revision != value.ExpectedRevision {
		return fmt.Errorf(
			"%w: artifact %q changed during refresh",
			basespec.ErrConflict,
			value.ArtifactID,
		)
	}
	if value.Revision != current.Revision+1 {
		return fmt.Errorf(
			"%w: source-derived artifact update revision does not advance current state",
			basespec.ErrInvalid,
		)
	}
	if !value.ModifiedAt.After(current.ModifiedAt) {
		return fmt.Errorf(
			"%w: source-derived artifact update time must advance current state",
			basespec.ErrInvalid,
		)
	}

	key := catalog.OccurrenceKey{
		CollectionID:       current.CollectionID,
		SourceID:           current.Binding.SourceID,
		Locator:            current.Binding.Locator,
		SubresourceLocator: current.Binding.SubresourceLocator,
	}
	observed, found := occurrences[key]
	var occurrence *catalog.Occurrence
	if found {
		occurrence = &observed
	}

	expectedDigest, expectedState, expectedDiagnostics, err := artifact.DeriveSourceState(current, occurrence)
	if err != nil {
		return err
	}
	if value.State != expectedState ||
		!cryptoutil.IsDigestEqual(value.ResolvedDefinition, expectedDigest) ||
		!diagnostic.EqualDiagnostics(value.Diagnostics, expectedDiagnostics) {
		return fmt.Errorf(
			"%w: source-derived artifact update does not match current occurrence",
			basespec.ErrInvalid,
		)
	}

	diagnostics, err := encodeJSON(value.Diagnostics)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_artifacts
		 SET resolved_definition_digest = ?,
		     state = ?,
		     diagnostics_json = ?,
		     revision = ?,
		     modified_at = ?
		 WHERE id = ? AND root_id = ? AND collection_id = ? AND revision = ?`,
		nullableDigest(value.ResolvedDefinition),
		string(value.State),
		diagnostics,
		value.Revision,
		timeValue(value.ModifiedAt),
		string(value.ArtifactID),
		string(value.RootID),
		string(value.CollectionID),
		value.ExpectedRevision,
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
			"%w: artifact %q changed during refresh",
			basespec.ErrConflict,
			value.ArtifactID,
		)
	}
	return nil
}
