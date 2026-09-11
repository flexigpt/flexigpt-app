package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const occurrenceColumns = `
	root_id, collection_id, source_id, locator, subresource_locator,
	kind, logical_name, logical_version,
	definition_digest, definition_json, source_content_digest, decoder_id,
	state, diagnostics_json, observed_at`

func (s *Store) getCurrentCatalog(
	ctx context.Context,
	ref collection.CollectionRef,
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

	attachmentRevisions := map[source.SourceID]uint64{}
	sourceRevisions := map[source.SourceID]uint64{}
	sourceGenerations := map[source.SourceID]string{}
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
		return value.Clone(), fmt.Errorf(
			"%w: catalog for collection %q does not match current metadata",
			basespec.ErrCatalogStale,
			ref.CollectionID,
		)
	}
	return value.Clone(), nil
}

func scanOccurrence(row scanner) (catalog.Occurrence, error) {
	var (
		rootID, collectionID, sourceID, locator, subresource string
		kind, logicalName, logicalVersion                    string
		definitionDigest, sourceDigest                       sql.NullString
		definitionRaw                                        []byte
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
		&definitionRaw,
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
	var cachedDefinition *definition.Definition
	if len(definitionRaw) != 0 {
		var value definition.Definition
		if err := decodeJSON(definitionRaw, &value); err != nil {
			return catalog.Occurrence{}, err
		}
		cachedDefinition = &value
	}
	value := catalog.Occurrence{
		RootID:       root.RootID(rootID),
		CollectionID: collection.CollectionID(collectionID),
		Key: catalog.OccurrenceKey{
			CollectionID:       collection.CollectionID(collectionID),
			SourceID:           source.SourceID(sourceID),
			Locator:            basespec.Locator(locator),
			SubresourceLocator: basespec.SubresourceLocator(subresource),
		},
		Kind:                artifact.ArtifactKind(kind),
		LogicalName:         basespec.LogicalName(logicalName),
		LogicalVersion:      basespec.LogicalVersion(logicalVersion),
		DefinitionDigest:    parseDigest(definitionDigest),
		Definition:          cachedDefinition,
		SourceContentDigest: parseDigest(sourceDigest),
		DecoderID:           basespec.DecoderID(decoderID),
		State:               catalog.OccurrenceState(state),
		Diagnostics:         diagnostics,
		ObservedAt:          parseTime(observedAt),
	}
	return value, nil
}

func currentAttachmentSourceRevisionsTx(
	ctx context.Context,
	tx *sql.Tx,
	ref collection.CollectionRef,
) (
	currentAttachments map[source.SourceID]uint64,

	currentSources map[source.SourceID]uint64,
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

	attachments := make(map[source.SourceID]uint64)
	sources := make(map[source.SourceID]uint64)
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
		id := source.SourceID(sourceID)
		attachments[id] = attachmentRevision
		sources[id] = sourceRevision
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return attachments, sources, nil
}
