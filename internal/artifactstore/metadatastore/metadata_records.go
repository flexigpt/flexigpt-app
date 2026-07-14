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
	if err := spec.ValidateArtifactRecord(record); err != nil {
		return err
	}
	diagnostics, err := encodeDiagnostics(record.Diagnostics)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
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
	if err := spec.ValidateTransferProvenance(provenance); err != nil {
		return err
	}
	origin, err := encodeCatalogResourceKey(provenance.OriginResource)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
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
		var id, target, operation, digest, created string
		var originID sql.NullString
		var originRaw []byte
		if err := rows.Scan(&id, &target, &operation, &originID, &originRaw, &digest, &created); err != nil {
			return nil, err
		}
		originResource, err := decodeCatalogResourceKey(originRaw)
		if err != nil {
			return nil, err
		}
		createdAt, err := parseRequiredTime("provenance.createdAt", created)
		if err != nil {
			return nil, err
		}
		value := spec.TransferProvenance{
			ProvenanceID:           spec.ProvenanceID(id),
			TargetRecordID:         spec.RecordID(target),
			Operation:              spec.TransferOperation(operation),
			OriginRecordID:         optionalRecordID(originID),
			OriginResource:         originResource,
			OriginDefinitionDigest: spec.Digest(digest),
			CreatedAt:              createdAt,
		}
		if err := spec.ValidateTransferProvenance(value); err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func scanRecord(scanner sqlScanner) (spec.ArtifactRecord, error) {
	var id, root, kind, name, version, source, locator, subresource, mode, tracking, dataSchema, state, createdRaw, modifiedRaw string
	var collection, pinned, resolved sql.NullString
	var enabled int
	var data, diagnostics []byte
	if err := scanner.Scan(
		&id,
		&root,
		&collection,
		&kind,
		&name,
		&version,
		&source,
		&locator,
		&subresource,
		&mode,
		&tracking,
		&pinned,
		&resolved,
		&enabled,
		&dataSchema,
		&data,
		&state,
		&diagnostics,
		&createdRaw,
		&modifiedRaw,
	); err != nil {
		return spec.ArtifactRecord{}, err
	}
	created, err := parseRequiredTime("record.createdAt", createdRaw)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	modified, err := parseRequiredTime("record.modifiedAt", modifiedRaw)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	decodedDiagnostics, err := decodeDiagnostics(diagnostics)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	record := spec.ArtifactRecord{
		RecordID:                     spec.RecordID(id),
		RootID:                       spec.RootID(root),
		CollectionID:                 optionalCollectionID(collection),
		Kind:                         spec.ArtifactKind(kind),
		Name:                         spec.RecordName(name),
		Version:                      spec.RecordVersion(version),
		SourceID:                     spec.SourceID(source),
		Locator:                      spec.SourceLocator(locator),
		SubresourceLocator:           spec.SubresourceLocator(subresource),
		RecordMode:                   spec.RecordMode(mode),
		TrackingMode:                 spec.TrackingMode(tracking),
		PinnedDefinitionDigest:       optionalDigest(pinned),
		LastResolvedDefinitionDigest: optionalDigest(resolved),
		Enabled:                      enabled != 0,
		DataSchemaID:                 spec.SchemaID(dataSchema),
		Data:                         append([]byte(nil), data...),
		State:                        spec.RecordState(state),
		Diagnostics:                  decodedDiagnostics,
		CreatedAt:                    created,
		ModifiedAt:                   modified,
	}
	if err := spec.ValidateArtifactRecord(record); err != nil {
		return spec.ArtifactRecord{}, fmt.Errorf("invalid persisted record %q: %w", id, err)
	}
	return record, nil
}
