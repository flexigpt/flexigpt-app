package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
)

const occurrenceColumns = `
	root_id, source_id, locator, subresource_locator,
	kind, logical_name, logical_version,
	definition_digest, source_content_digest, decoder_id,
	state, diagnostics_json, observed_at`

func (s *Store) getCurrentCatalog(
	ctx context.Context,
	rootID artifactstore.RootID,
) (catalog.Snapshot, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return catalog.Snapshot{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var currentRootRevision uint64
	err = tx.QueryRowContext(
		ctx,
		`SELECT revision FROM artifact_roots
		 WHERE id = ? AND deleted_at IS NULL`,
		string(rootID),
	).Scan(&currentRootRevision)
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: root %q",
			artifactstore.ErrNotFound,
			rootID,
		)
	}
	if err != nil {
		return catalog.Snapshot{}, err
	}

	var (
		revision, rootRevision uint64
		sourceRevisionsRaw     []byte
		sourceGenerationsRaw   []byte
		publishedAt            int64
		diagnosticsRaw         []byte
	)
	err = tx.QueryRowContext(
		ctx,
		`SELECT revision, root_revision, source_revisions_json,
		        source_generations_json, published_at, diagnostics_json
		 FROM artifact_current_catalogs
		 WHERE root_id = ?`,
		string(rootID),
	).Scan(
		&revision,
		&rootRevision,
		&sourceRevisionsRaw,
		&sourceGenerationsRaw,
		&publishedAt,
		&diagnosticsRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: root %q has no current catalog",
			artifactstore.ErrCatalogUnavailable,
			rootID,
		)
	}
	if err != nil {
		return catalog.Snapshot{}, err
	}

	sourceRevisions := map[artifactstore.SourceID]uint64{}
	sourceGenerations := map[artifactstore.SourceID]string{}
	diagnostics := []artifactstore.Diagnostic{}
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
		 WHERE root_id = ?
		 ORDER BY source_id, locator, subresource_locator`,
		string(rootID),
	)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	defer rows.Close()
	defer func() { _ = rows.Close() }()

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
		RootID:            rootID,
		Revision:          revision,
		RootRevision:      rootRevision,
		SourceRevisions:   sourceRevisions,
		SourceGenerations: sourceGenerations,
		PublishedAt:       parseTime(publishedAt),
		Diagnostics:       diagnostics,
		Occurrences:       occurrences,
	}
	if err := value.Validate(); err != nil {
		return catalog.Snapshot{}, fmt.Errorf(
			"invalid persisted catalog: %w",
			err,
		)
	}
	currentSourceRevisions, err := currentAttachedSourceRevisions(
		ctx,
		tx,
		rootID,
	)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	stale := value.RootRevision != currentRootRevision ||
		!maps.Equal(value.SourceRevisions, currentSourceRevisions)

	if err := tx.Commit(); err != nil {
		return catalog.Snapshot{}, err
	}
	if stale {
		return catalog.CloneSnapshot(value), fmt.Errorf(
			"%w: catalog for root %q does not match current metadata",
			artifactstore.ErrCatalogStale,
			rootID,
		)
	}
	return catalog.CloneSnapshot(value), nil
}

func scanOccurrence(row scanner) (catalog.Occurrence, error) {
	var (
		rootID, sourceID, locator, subresource string
		kind, logicalName, logicalVersion      string
		definitionDigest, sourceDigest         sql.NullString
		decoderID, state                       string
		diagnosticsRaw                         []byte
		observedAt                             int64
	)
	if err := row.Scan(
		&rootID,
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
	diagnostics := []artifactstore.Diagnostic{}
	if err := decodeJSON(diagnosticsRaw, &diagnostics); err != nil {
		return catalog.Occurrence{}, err
	}
	value := catalog.Occurrence{
		RootID: artifactstore.RootID(rootID),
		Key: catalog.OccurrenceKey{
			SourceID:           artifactstore.SourceID(sourceID),
			Locator:            artifactstore.Locator(locator),
			SubresourceLocator: artifactstore.SubresourceLocator(subresource),
		},
		Kind:                artifactstore.ArtifactKind(kind),
		LogicalName:         artifactstore.LogicalName(logicalName),
		LogicalVersion:      artifactstore.LogicalVersion(logicalVersion),
		DefinitionDigest:    parseDigest(definitionDigest),
		SourceContentDigest: parseDigest(sourceDigest),
		DecoderID:           artifactstore.DecoderID(decoderID),
		State:               catalog.OccurrenceState(state),
		Diagnostics:         diagnostics,
		ObservedAt:          parseTime(observedAt),
	}
	if err := value.Validate(); err != nil {
		return catalog.Occurrence{}, err
	}
	return value, nil
}
