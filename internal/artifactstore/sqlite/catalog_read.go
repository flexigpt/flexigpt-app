package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const occurrenceColumns = `
	root_id, collection_id, source_id, locator, subresource_locator,
	kind, logical_name, logical_version,
	definition_digest, source_content_digest, decoder_id,
	state, diagnostics_json, observed_at`

func (s *Store) getCurrentCatalog(
	ctx context.Context,
	ref basespec.CollectionRef,
) (catalog.Snapshot, error) {
	if err := ref.Validate(); err != nil {
		return catalog.Snapshot{}, err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return catalog.Snapshot{}, err
	}
	defer func() { _ = tx.Rollback() }()

	currentCollection, err := getActiveCollectionTx(ctx, tx, ref)
	if err != nil {
		return catalog.Snapshot{}, err
	}

	var (
		revision               uint64
		collectionRevision     uint64
		attachmentRevisionsRaw []byte
		sourceRevisionsRaw     []byte
		sourceGenerationsRaw   []byte
		planFingerprint        string
		decoderFingerprint     string
		publishedAt            int64
		diagnosticsRaw         []byte
	)
	err = tx.QueryRowContext(
		ctx,
		`SELECT revision, collection_revision,
		        attachment_revisions_json, source_revisions_json,
		        source_generations_json, plan_fingerprint,
		        decoder_fingerprint, published_at, diagnostics_json
		 FROM artifact_current_catalogs
		 WHERE root_id = ? AND collection_id = ?`,
		string(ref.RootID),
		string(ref.CollectionID),
	).Scan(
		&revision,
		&collectionRevision,
		&attachmentRevisionsRaw,
		&sourceRevisionsRaw,
		&sourceGenerationsRaw,
		&planFingerprint,
		&decoderFingerprint,
		&publishedAt,
		&diagnosticsRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: collection %q has no current catalog",
			basespec.ErrCatalogUnavailable,
			ref.CollectionID,
		)
	}
	if err != nil {
		return catalog.Snapshot{}, err
	}

	attachmentRevisions := map[basespec.SourceID]uint64{}
	sourceRevisions := map[basespec.SourceID]uint64{}
	sourceGenerations := map[basespec.SourceID]string{}
	diagnostics := []diagnostic.Diagnostic{}
	if err := decodeJSON(attachmentRevisionsRaw, &attachmentRevisions); err != nil {
		return catalog.Snapshot{}, err
	}
	if err := decodeJSON(sourceRevisionsRaw, &sourceRevisions); err != nil {
		return catalog.Snapshot{}, err
	}
	if err := decodeJSON(sourceGenerationsRaw, &sourceGenerations); err != nil {
		return catalog.Snapshot{}, err
	}
	if err := decodeJSON(diagnosticsRaw, &diagnostics); err != nil {
		return catalog.Snapshot{}, err
	}

	rows, err := tx.QueryContext(
		ctx,
		`SELECT `+occurrenceColumns+`
		 FROM artifact_current_occurrences
		 WHERE root_id = ? AND collection_id = ?
		 ORDER BY source_id, locator, subresource_locator`,
		string(ref.RootID),
		string(ref.CollectionID),
	)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	defer rows.Close()

	occurrences := make([]catalog.Occurrence, 0)
	for rows.Next() {
		value, err := scanOccurrence(rows)
		if err != nil {
			return catalog.Snapshot{}, err
		}
		occurrences = append(occurrences, value)
	}
	if err := rows.Err(); err != nil {
		return catalog.Snapshot{}, err
	}
	if err := rows.Close(); err != nil {
		return catalog.Snapshot{}, err
	}

	value := catalog.Snapshot{
		RootID:              ref.RootID,
		CollectionID:        ref.CollectionID,
		Revision:            revision,
		CollectionRevision:  collectionRevision,
		AttachmentRevisions: attachmentRevisions,
		SourceRevisions:     sourceRevisions,
		SourceGenerations:   sourceGenerations,
		PlanFingerprint:     cryptoutil.Digest(planFingerprint),
		DecoderFingerprint:  cryptoutil.Digest(decoderFingerprint),
		PublishedAt:         parseTime(publishedAt),
		Diagnostics:         diagnostics,
		Occurrences:         occurrences,
	}
	if err := value.Validate(); err != nil {
		return catalog.Snapshot{}, fmt.Errorf(
			"invalid persisted catalog: %w",
			err,
		)
	}

	currentAttachments, currentSources, err := currentAttachmentSourceRevisionsTx(
		ctx,
		tx,
		ref,
	)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	stale := value.CollectionRevision != currentCollection.Revision ||
		!maps.Equal(value.AttachmentRevisions, currentAttachments) ||
		!maps.Equal(value.SourceRevisions, currentSources)

	if err := tx.Commit(); err != nil {
		return catalog.Snapshot{}, err
	}
	if stale {
		return catalog.CloneSnapshot(value), fmt.Errorf(
			"%w: catalog for collection %q does not match current metadata",
			basespec.ErrCatalogStale,
			ref.CollectionID,
		)
	}
	return catalog.CloneSnapshot(value), nil
}

func scanOccurrence(row scanner) (catalog.Occurrence, error) {
	var (
		rootID, collectionID, sourceID, locator, subresource string
		kind, logicalName, logicalVersion                    string
		definitionDigest, sourceDigest                       sql.NullString
		decoderID, state                                     string
		diagnosticsRaw                                       []byte
		observedAt                                           int64
	)
	if err := row.Scan(
		&rootID,
		&collectionID,
		&sourceID,
		&locator,
		&subresource,
		&kind,
		&logicalName,
		&logicalVersion,
		&definitionDigest,
		&sourceDigest,
		&decoderID,
		&state,
		&diagnosticsRaw,
		&observedAt,
	); err != nil {
		return catalog.Occurrence{}, err
	}
	diagnostics := []diagnostic.Diagnostic{}
	if err := decodeJSON(diagnosticsRaw, &diagnostics); err != nil {
		return catalog.Occurrence{}, err
	}
	value := catalog.Occurrence{
		RootID:       basespec.RootID(rootID),
		CollectionID: basespec.CollectionID(collectionID),
		Key: catalog.OccurrenceKey{
			CollectionID:       basespec.CollectionID(collectionID),
			SourceID:           basespec.SourceID(sourceID),
			Locator:            basespec.Locator(locator),
			SubresourceLocator: basespec.SubresourceLocator(subresource),
		},
		Kind:                basespec.ArtifactKind(kind),
		LogicalName:         basespec.LogicalName(logicalName),
		LogicalVersion:      basespec.LogicalVersion(logicalVersion),
		DefinitionDigest:    parseDigest(definitionDigest),
		SourceContentDigest: parseDigest(sourceDigest),
		DecoderID:           basespec.DecoderID(decoderID),
		State:               catalog.OccurrenceState(state),
		Diagnostics:         diagnostics,
		ObservedAt:          parseTime(observedAt),
	}
	if err := value.Validate(); err != nil {
		return catalog.Occurrence{}, err
	}
	return value, nil
}

func currentAttachmentSourceRevisionsTx(
	ctx context.Context,
	tx *sql.Tx,
	ref basespec.CollectionRef,
) (
	currentAttachments map[basespec.SourceID]uint64,
	currentSources map[basespec.SourceID]uint64,
	err error,
) {
	rows, err := tx.QueryContext(
		ctx,
		`SELECT a.source_id, a.revision, s.revision
		 FROM artifact_collection_attachments a
		 JOIN artifact_sources s
		   ON s.root_id = a.root_id AND s.id = a.source_id
		 WHERE a.root_id = ? AND a.collection_id = ?
		   AND s.retired_at IS NULL
		 ORDER BY a.source_id`,
		string(ref.RootID),
		string(ref.CollectionID),
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	attachments := make(map[basespec.SourceID]uint64)
	sources := make(map[basespec.SourceID]uint64)
	for rows.Next() {
		var sourceID string
		var attachmentRevision, sourceRevision uint64
		if err := rows.Scan(
			&sourceID,
			&attachmentRevision,
			&sourceRevision,
		); err != nil {
			return nil, nil, err
		}
		id := basespec.SourceID(sourceID)
		attachments[id] = attachmentRevision
		sources[id] = sourceRevision
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return attachments, sources, nil
}
