package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
)

const recordColumns = `
	id, root_id, source_id, locator, subresource_locator,
	kind, name, enabled, mode, pinned_definition_digest,
	resolved_definition_digest, data_json, state,
	diagnostics_json, revision, created_at, modified_at`

func (s *Store) getRecord(
	ctx context.Context,
	id artifactstore.RecordID,
) (record.Record, error) {
	value, err := scanRecord(s.db.QueryRowContext(
		ctx,
		`SELECT `+recordColumns+`
		 FROM artifact_records WHERE id = ?`,
		string(id),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return record.Record{}, fmt.Errorf(
			"%w: record %q",
			artifactstore.ErrNotFound,
			id,
		)
	}
	return value, err
}

func (s *Store) listRecordsByRoot(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]record.Record, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+recordColumns+`
		 FROM artifact_records
		 WHERE root_id = ?
		 ORDER BY modified_at DESC, id ASC`,
		string(rootID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	output := make([]record.Record, 0)
	for rows.Next() {
		value, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	return output, rows.Err()
}

func (s *Store) updateRecord(
	ctx context.Context,
	value record.Record,
	expectedRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	diagnostics, err := encodeJSON(value.Diagnostics)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE artifact_records
		 SET name = ?,
		     enabled = ?,
		     mode = ?,
		     pinned_definition_digest = ?,
		     resolved_definition_digest = ?,
		     data_json = ?,
		     state = ?,
		     diagnostics_json = ?,
		     revision = ?,
		     modified_at = ?
		 WHERE id = ? AND revision = ?`,
		value.Name,
		boolInt(value.Enabled),
		string(value.Mode),
		nullableDigest(value.PinnedDefinition),
		nullableDigest(value.ResolvedDefinition),
		[]byte(value.Data),
		string(value.State),
		diagnostics,
		value.Revision,
		timeValue(value.ModifiedAt),
		string(value.ID),
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
			"%w: record %q changed or no longer exists",
			artifactstore.ErrConflict,
			value.ID,
		)
	}
	return nil
}

func (s *Store) deleteRecord(
	ctx context.Context,
	id artifactstore.RecordID,
	expectedRevision uint64,
) error {
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM artifact_records WHERE id = ? AND revision = ?`,
		string(id),
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
		return artifactstore.ErrConflict
	}
	return nil
}

func scanRecord(row scanner) (record.Record, error) {
	var (
		id, rootID, sourceID, locator, subresource string
		kind, name, mode, state                    string
		enabled                                    int
		pinned, resolved                           sql.NullString
		data, diagnosticsRaw                       []byte
		revision                                   uint64
		createdAt, modifiedAt                      int64
	)
	if err := row.Scan(
		&id,
		&rootID,
		&sourceID,
		&locator,
		&subresource,
		&kind,
		&name,
		&enabled,
		&mode,
		&pinned,
		&resolved,
		&data,
		&state,
		&diagnosticsRaw,
		&revision,
		&createdAt,
		&modifiedAt,
	); err != nil {
		return record.Record{}, err
	}
	diagnostics := []artifactstore.Diagnostic{}
	if err := decodeJSON(diagnosticsRaw, &diagnostics); err != nil {
		return record.Record{}, err
	}
	value := record.Record{
		ID:     artifactstore.RecordID(id),
		RootID: artifactstore.RootID(rootID),
		Occurrence: catalog.OccurrenceKey{
			SourceID:           artifactstore.SourceID(sourceID),
			Locator:            artifactstore.Locator(locator),
			SubresourceLocator: artifactstore.SubresourceLocator(subresource),
		},
		Kind:               artifactstore.ArtifactKind(kind),
		Name:               name,
		Enabled:            enabled != 0,
		Mode:               record.Mode(mode),
		PinnedDefinition:   parseDigest(pinned),
		ResolvedDefinition: parseDigest(resolved),
		Data:               append([]byte(nil), data...),
		State:              record.State(state),
		Diagnostics:        diagnostics,
		Revision:           revision,
		CreatedAt:          parseTime(createdAt),
		ModifiedAt:         parseTime(modifiedAt),
	}
	if err := value.Validate(); err != nil {
		return record.Record{}, err
	}
	return value, nil
}
