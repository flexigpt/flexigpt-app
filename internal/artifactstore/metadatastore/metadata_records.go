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

const selectLatestRootGenerationForSynchronizationSQL = `SELECT generation
	FROM root_catalog_generations
	WHERE root_id = ?
	ORDER BY generation DESC
	LIMIT 1`

const selectTransferAttachmentSQL = `SELECT
		a.enabled, r.enabled, r.soft_deleted_at, r.mount_revision
	FROM root_source_attachments a
	JOIN artifact_roots r ON r.root_id = a.root_id
	WHERE a.root_id = ? AND a.source_id = ?`

const invalidateTransferredSourceSQL = `UPDATE artifact_sources
	SET last_observed_generation = NULL,
	    last_scanned_at = NULL,
	    observation_revision = observation_revision + 1
	WHERE source_id = ?
	  AND observation_revision = ?
	  AND observation_revision < ?
	  AND enabled = 1`

const recordColumns = `record_id, root_id, collection_id, kind, name, version, source_id, locator, subresource_locator, record_mode, tracking_mode, pinned_definition_digest, last_resolved_definition_digest, enabled, data_schema_id, data_json, state, diagnostics_json, created_at, modified_at`

func (s *MetadataStore) CreateRecord(ctx context.Context, record spec.ArtifactRecord) error {
	return insertRecord(ctx, s.db, record)
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (s *MetadataStore) GetRecord(ctx context.Context, recordID spec.RecordID) (spec.ArtifactRecord, error) {
	record, err := scanRecord(
		s.db.QueryRowContext(
			ctx,
			`SELECT `+recordColumns+` FROM artifact_records WHERE record_id = ?`,
			string(recordID),
		),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return spec.ArtifactRecord{}, fmt.Errorf("%w: record %q", spec.ErrNotFound, recordID)
	}
	return record, err
}

func (s *MetadataStore) ListRecordsForRoot(ctx context.Context, rootID spec.RootID) ([]spec.ArtifactRecord, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+recordColumns+` FROM artifact_records WHERE root_id = ? ORDER BY modified_at DESC, record_id ASC`,
		string(rootID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []spec.ArtifactRecord{}
	for rows.Next() {
		value, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func (s *MetadataStore) FindRecordBySource(
	ctx context.Context,
	rootID spec.RootID,
	key spec.CatalogResourceKey,
	kind spec.ArtifactKind,
) (spec.ArtifactRecord, error) {
	record, err := scanRecord(
		s.db.QueryRowContext(
			ctx,
			`SELECT `+recordColumns+` FROM artifact_records WHERE root_id = ? AND source_id = ? AND locator = ? AND subresource_locator = ? AND kind = ?`,
			string(rootID),
			string(key.SourceID),
			string(key.Locator),
			string(key.SubresourceLocator),
			string(kind),
		),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return spec.ArtifactRecord{}, fmt.Errorf("%w: record for source occurrence", spec.ErrNotFound)
	}
	return record, err
}

func (s *MetadataStore) UpdateRecord(
	ctx context.Context,
	record spec.ArtifactRecord,
	expectedModifiedAt time.Time,
) error {
	if err := validate.ValidateArtifactRecord(record); err != nil {
		return err
	}
	if err := validateExpectedModifiedAt("record", expectedModifiedAt); err != nil {
		return err
	}
	diagnostics, err := encodeDiagnostics(record.Diagnostics)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, updateRecordSQL,
		nullableID(record.CollectionID),
		string(record.RecordMode),
		string(record.TrackingMode),
		nullableDigest(record.PinnedDefinitionDigest),
		nullableDigest(record.LastResolvedDefinitionDigest),
		boolToInt(record.Enabled),
		string(record.DataSchemaID),
		string(record.Data),
		string(record.State),
		string(diagnostics),
		formatTime(record.ModifiedAt),
		string(record.RecordID),
		formatTime(expectedModifiedAt),
	)
	if err != nil {
		return sqliteError(err)
	}
	return optimisticMutationResult(result, "record "+string(record.RecordID))
}

func (s *MetadataStore) DeleteRecord(
	ctx context.Context,
	recordID spec.RecordID,
	expectedModifiedAt time.Time,
) error {
	if err := validateExpectedModifiedAt("record", expectedModifiedAt); err != nil {
		return err
	}
	result, err := s.db.ExecContext(
		ctx,
		deleteRecordSQL,
		string(recordID),
		formatTime(expectedModifiedAt),
	)
	if err != nil {
		return sqliteError(err)
	}
	return optimisticMutationResult(result, "record "+string(recordID))
}

func (s *MetadataStore) PublishRecordSynchronization(
	ctx context.Context,
	publication spec.RecordSynchronizationPublication,
) error {
	if publication.RootID == "" {
		return fmt.Errorf("%w: synchronization root ID is empty", spec.ErrInvalidRequest)
	}
	if publication.ExpectedCatalogGeneration == 0 {
		return fmt.Errorf(
			"%w: synchronization catalog generation is required",
			spec.ErrInvalidRequest,
		)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin record synchronization: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var currentGeneration uint64
	err = tx.QueryRowContext(
		ctx,
		selectLatestRootGenerationForSynchronizationSQL,
		string(publication.RootID),
	).Scan(&currentGeneration)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf(
			"%w: root %q has no published catalog",
			spec.ErrConflict,
			publication.RootID,
		)
	}
	if err != nil {
		return fmt.Errorf("read synchronization catalog generation: %w", err)
	}
	if currentGeneration != publication.ExpectedCatalogGeneration {
		return fmt.Errorf(
			"%w: root catalog changed during record synchronization",
			spec.ErrConflict,
		)
	}
	if err := ensureRootCatalogCurrentTx(
		ctx,
		tx,
		publication.RootID,
		publication.ExpectedCatalogGeneration,
	); err != nil {
		return err
	}

	for _, record := range publication.Creates {
		if record.RootID != publication.RootID {
			return fmt.Errorf("%w: synchronized record root mismatch", spec.ErrInvalidRequest)
		}
		if err := insertRecord(ctx, tx, record); err != nil {
			return err
		}
	}
	for _, update := range publication.Updates {
		record := update.Record
		if record.RootID != publication.RootID ||
			update.ExpectedModifiedAt.IsZero() ||
			update.ExpectedRecordMode == "" ||
			update.ExpectedTrackingMode == "" {
			return fmt.Errorf("%w: invalid synchronized record update", spec.ErrInvalidRequest)
		}
		if err := validate.ValidateArtifactRecord(record); err != nil {
			return err
		}
		diagnostics, err := encodeDiagnostics(record.Diagnostics)
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(
			ctx,
			`UPDATE artifact_records
			    SET last_resolved_definition_digest = ?,
			        state = ?,
			        diagnostics_json = ?,
			        modified_at = ?
			  WHERE record_id = ?
			    AND root_id = ?
			    AND modified_at = ?
			    AND record_mode = ?
			    AND tracking_mode = ?`,
			nullableDigest(record.LastResolvedDefinitionDigest),
			string(record.State),
			diagnostics,
			formatTime(record.ModifiedAt),
			string(record.RecordID),
			string(record.RootID),
			formatTime(update.ExpectedModifiedAt),
			string(update.ExpectedRecordMode),
			string(update.ExpectedTrackingMode),
		)
		if err != nil {
			return sqliteError(fmt.Errorf("synchronize record %q: %w", record.RecordID, err))
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect synchronized record %q: %w", record.RecordID, err)
		}
		if changed == 0 {
			return fmt.Errorf(
				"%w: record %q changed during synchronization",
				spec.ErrConflict,
				record.RecordID,
			)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit record synchronization: %w", err)
	}
	return nil
}

func (s *MetadataStore) PublishRecordTransfer(
	ctx context.Context,
	publication spec.RecordTransferPublication,
) error {
	resourceKey := spec.CatalogResourceKey{
		SourceID:           publication.Resource.SourceID,
		Locator:            publication.Resource.Locator,
		SubresourceLocator: publication.Resource.SubresourceLocator,
	}
	if publication.Record.SourceID != resourceKey.SourceID ||
		publication.Record.Locator != resourceKey.Locator ||
		publication.Record.SubresourceLocator != resourceKey.SubresourceLocator ||
		publication.Revision.SourceID != resourceKey.SourceID ||
		publication.Revision.Locator != resourceKey.Locator ||
		publication.Revision.SubresourceLocator != resourceKey.SubresourceLocator ||
		publication.Provenance.TargetRecordID != publication.Record.RecordID {
		return fmt.Errorf("%w: inconsistent record transfer publication", spec.ErrInvalidRequest)
	}
	if publication.ExpectedRootRevision == 0 {
		return fmt.Errorf(
			"%w: record transfer optimistic expectations are incomplete",
			spec.ErrInvalidRequest,
		)
	}
	if err := validate.ValidateCatalogResource(publication.Resource); err != nil {
		return err
	}
	if err := validate.ValidateCatalogResourceRevision(publication.Revision); err != nil {
		return err
	}
	if err := validate.ValidateArtifactRecord(publication.Record); err != nil {
		return err
	}
	if err := validate.ValidateTransferProvenance(publication.Provenance); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin record transfer publication: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var softDeletedAt sql.NullString
	var attachmentEnabled, rootEnabled int
	var rootRevision uint64
	err = tx.QueryRowContext(
		ctx,
		selectTransferAttachmentSQL,
		string(publication.Record.RootID),
		string(publication.Record.SourceID),
	).Scan(
		&attachmentEnabled,
		&rootEnabled,
		&softDeletedAt,
		&rootRevision,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf(
			"%w: destination source is no longer attached to the root",
			spec.ErrConflict,
		)
	}
	if err != nil {
		return fmt.Errorf("read transfer attachment: %w", err)
	}
	if attachmentEnabled == 0 ||
		rootEnabled == 0 ||
		softDeletedAt.Valid ||
		rootRevision != publication.ExpectedRootRevision {
		return fmt.Errorf(
			"%w: destination root or source attachment changed during transfer",
			spec.ErrConflict,
		)
	}

	sourceResult, err := tx.ExecContext(
		ctx,
		invalidateTransferredSourceSQL,
		string(publication.Record.SourceID),
		publication.ExpectedSourceObservationRevision,
		spec.MaxObservationRevision,
	)
	if err != nil {
		return sqliteError(fmt.Errorf("invalidate transferred source observation: %w", err))
	}
	changed, err := sourceResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect transferred source invalidation: %w", err)
	}
	if changed != 1 {
		return fmt.Errorf(
			"%w: destination source changed during transfer",
			spec.ErrConflict,
		)
	}
	if err := upsertCatalogResourceTx(ctx, tx, publication.Resource); err != nil {
		return err
	}
	if err := upsertCatalogRevisionTx(ctx, tx, publication.Revision); err != nil {
		return err
	}
	if err := insertRecord(ctx, tx, publication.Record); err != nil {
		return err
	}
	if err := insertTransferProvenance(ctx, tx, publication.Provenance); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit record transfer publication: %w", err)
	}
	return nil
}

func (s *MetadataStore) ListTransferProvenance(
	ctx context.Context,
	recordID spec.RecordID,
) ([]spec.TransferProvenance, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT provenance_id, target_record_id, operation, origin_record_id, origin_resource_json, origin_definition_digest, created_at FROM artifact_transfer_provenance WHERE target_record_id = ? ORDER BY created_at ASC, provenance_id ASC`,
		string(recordID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []spec.TransferProvenance{}
	for rows.Next() {
		row := transferProvenanceRow{}
		if err := rows.Scan(row.destinations()...); err != nil {
			return nil, err
		}
		originResource, err := decodeCatalogResourceKey(row.OriginResource)
		if err != nil {
			return nil, err
		}
		createdAt, err := parseRequiredTime("provenance.createdAt", row.CreatedAt)
		if err != nil {
			return nil, err
		}
		value := spec.TransferProvenance{
			ProvenanceID:           spec.ProvenanceID(row.ProvenanceID),
			TargetRecordID:         spec.RecordID(row.TargetRecordID),
			Operation:              spec.TransferOperation(row.Operation),
			OriginRecordID:         optionalRecordID(row.OriginRecordID),
			OriginResource:         originResource,
			OriginDefinitionDigest: spec.Digest(row.OriginDefinitionDigest),
			CreatedAt:              createdAt,
		}
		if err := validate.ValidateTransferProvenance(value); err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func insertTransferProvenance(
	ctx context.Context,
	executor sqlExecutor,
	provenance spec.TransferProvenance,
) error {
	if err := validate.ValidateTransferProvenance(provenance); err != nil {
		return err
	}
	origin, err := encodeCatalogResourceKey(provenance.OriginResource)
	if err != nil {
		return err
	}
	_, err = executor.ExecContext(
		ctx,
		`INSERT INTO artifact_transfer_provenance (provenance_id, target_record_id, operation, origin_record_id, origin_resource_json, origin_definition_digest, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		string(provenance.ProvenanceID),
		string(provenance.TargetRecordID),
		string(provenance.Operation),
		nullableID(provenance.OriginRecordID),
		origin,
		string(provenance.OriginDefinitionDigest),
		formatTime(provenance.CreatedAt),
	)
	return sqliteError(err)
}

func insertRecord(ctx context.Context, executor sqlExecutor, record spec.ArtifactRecord) error {
	if err := validate.ValidateArtifactRecord(record); err != nil {
		return err
	}
	diagnostics, err := encodeDiagnostics(record.Diagnostics)
	if err != nil {
		return err
	}
	_, err = executor.ExecContext(
		ctx,
		`INSERT INTO artifact_records (record_id, root_id, collection_id, kind, name, version, source_id, locator, subresource_locator, record_mode, tracking_mode, pinned_definition_digest, last_resolved_definition_digest, enabled, data_schema_id, data_json, state, diagnostics_json, created_at, modified_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(record.RecordID),
		string(record.RootID),
		nullableID(record.CollectionID),
		string(record.Kind),
		string(record.Name),
		string(record.Version),
		string(record.SourceID),
		string(record.Locator),
		string(record.SubresourceLocator),
		string(record.RecordMode),
		string(record.TrackingMode),
		nullableDigest(record.PinnedDefinitionDigest),
		nullableDigest(record.LastResolvedDefinitionDigest),
		boolToInt(record.Enabled),
		string(record.DataSchemaID),
		[]byte(record.Data),
		string(record.State),
		diagnostics,
		formatTime(record.CreatedAt),
		formatTime(record.ModifiedAt),
	)
	return sqliteError(err)
}

func scanRecord(scanner sqlScanner) (spec.ArtifactRecord, error) {
	row := artifactRecordRow{}
	if err := scanner.Scan(row.destinations()...); err != nil {
		return spec.ArtifactRecord{}, err
	}
	created, err := parseRequiredTime("record.createdAt", row.CreatedAt)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	modified, err := parseRequiredTime("record.modifiedAt", row.ModifiedAt)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	decodedDiagnostics, err := decodeDiagnostics(row.Diagnostics)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	record := spec.ArtifactRecord{
		RecordID:                     spec.RecordID(row.RecordID),
		RootID:                       spec.RootID(row.RootID),
		CollectionID:                 optionalCollectionID(row.CollectionID),
		Kind:                         spec.ArtifactKind(row.Kind),
		Name:                         spec.RecordName(row.Name),
		Version:                      spec.RecordVersion(row.Version),
		SourceID:                     spec.SourceID(row.SourceID),
		Locator:                      spec.SourceLocator(row.Locator),
		SubresourceLocator:           spec.SubresourceLocator(row.SubresourceLocator),
		RecordMode:                   spec.RecordMode(row.RecordMode),
		TrackingMode:                 spec.TrackingMode(row.TrackingMode),
		PinnedDefinitionDigest:       optionalDigest(row.PinnedDefinitionDigest),
		LastResolvedDefinitionDigest: optionalDigest(row.LastResolvedDefinitionDigest),
		Enabled:                      row.Enabled != 0,
		DataSchemaID:                 spec.SchemaID(row.DataSchemaID),
		Data:                         append([]byte(nil), row.Data...),
		State:                        spec.RecordState(row.State),
		Diagnostics:                  decodedDiagnostics,
		CreatedAt:                    created,
		ModifiedAt:                   modified,
	}
	if err := validate.ValidateArtifactRecord(record); err != nil {
		return spec.ArtifactRecord{}, fmt.Errorf("invalid persisted record %q: %w", row.RecordID, err)
	}
	return record, nil
}
