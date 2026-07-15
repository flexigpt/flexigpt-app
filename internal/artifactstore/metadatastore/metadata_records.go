package metadatastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

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

func (s *MetadataStore) UpdateRecord(ctx context.Context, record spec.ArtifactRecord) error {
	if err := spec.ValidateArtifactRecord(record); err != nil {
		return err
	}
	diagnostics, err := encodeDiagnostics(record.Diagnostics)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE artifact_records SET collection_id = ?, record_mode = ?, tracking_mode = ?, pinned_definition_digest = ?, last_resolved_definition_digest = ?, enabled = ?, data_schema_id = ?, data_json = ?, state = ?, diagnostics_json = ?, modified_at = ? WHERE record_id = ?`,
		nullableID(record.CollectionID),
		string(record.RecordMode),
		string(record.TrackingMode),
		nullableDigest(record.PinnedDefinitionDigest),
		nullableDigest(record.LastResolvedDefinitionDigest),
		boolToInt(record.Enabled),
		string(record.DataSchemaID),
		[]byte(record.Data),
		string(record.State),
		diagnostics,
		formatTime(record.ModifiedAt),
		string(record.RecordID),
	)
	if err != nil {
		return sqliteError(err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return fmt.Errorf("%w: record %q", spec.ErrNotFound, record.RecordID)
	}
	return nil
}

func (s *MetadataStore) DeleteRecord(ctx context.Context, recordID spec.RecordID) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM artifact_records WHERE record_id = ?`, string(recordID))
	if err != nil {
		return sqliteError(err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return fmt.Errorf("%w: record %q", spec.ErrNotFound, recordID)
	}
	return nil
}

func (s *MetadataStore) CreateTransferProvenance(ctx context.Context, provenance spec.TransferProvenance) error {
	return insertTransferProvenance(ctx, s.db, provenance)
}

func (s *MetadataStore) PublishRecordSynchronization(
	ctx context.Context,
	publication spec.RecordSynchronizationPublication,
) error {
	if publication.RootID == "" {
		return fmt.Errorf("%w: synchronization root ID is empty", spec.ErrInvalidRequest)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin record synchronization: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

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
		if err := spec.ValidateArtifactRecord(record); err != nil {
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
	if err := spec.ValidateCatalogResource(publication.Resource); err != nil {
		return err
	}
	if err := spec.ValidateCatalogResourceRevision(publication.Revision); err != nil {
		return err
	}
	if err := spec.ValidateArtifactRecord(publication.Record); err != nil {
		return err
	}
	if err := spec.ValidateTransferProvenance(publication.Provenance); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin record transfer publication: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
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
		if err := spec.ValidateTransferProvenance(value); err != nil {
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
	if err := spec.ValidateTransferProvenance(provenance); err != nil {
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
	if err := spec.ValidateArtifactRecord(record); err != nil {
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
	if err := spec.ValidateArtifactRecord(record); err != nil {
		return spec.ArtifactRecord{}, fmt.Errorf("invalid persisted record %q: %w", row.RecordID, err)
	}
	return record, nil
}
