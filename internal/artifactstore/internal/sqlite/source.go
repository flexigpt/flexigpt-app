package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

const sourceColumns = `
	id, root_id, root_storage_key, storage_key,
	kind, display_name, enabled, config_json, discovery_json,
	revision, created_at, modified_at, retired_at`

func (s *Store) createSource(
	ctx context.Context,
	value source.Source,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	discoveryRaw, err := encodeJSON(value.Discovery.Normalized())
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	rootValue, err := getActiveRootTx(ctx, tx, value.RootID)
	if err != nil {
		return err
	}
	if rootValue.StorageKey != value.RootStorageKey {
		return fmt.Errorf(
			"%w: Source Root storage key does not match Root metadata",
			basespec.ErrInvalid,
		)
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO artifact_sources (
			id, root_id, root_storage_key, storage_key,
			kind, display_name, enabled, config_json, discovery_json,
			revision, created_at, modified_at, retired_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(value.ID),
		string(value.RootID),
		string(value.RootStorageKey),
		string(value.StorageKey),
		string(value.Kind),
		value.DisplayName,
		boolInt(value.Enabled),
		[]byte(value.Config),
		discoveryRaw,
		value.Revision,
		timeValue(value.CreatedAt),
		timeValue(value.ModifiedAt),
		nullableTime(value.RetiredAt),
	)
	if err != nil {
		return sqliteError(err)
	}
	return tx.Commit()
}

func (s *Store) getSource(
	ctx context.Context,
	rootID root.RootID,
	id source.SourceID,
) (source.Source, error) {
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return source.Source{}, err
	}
	value, err := scanSource(s.db.QueryRowContext(
		ctx,
		`SELECT `+sourceColumns+`
		 FROM artifact_sources
		 WHERE root_id = ?
		   AND id = ?
		   AND retired_at IS NULL`,
		string(rootID),
		string(id),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return source.Source{}, fmt.Errorf(
			"%w: Source %q in Root %q",
			basespec.ErrSourceNotFound,
			id,
			rootID,
		)
	}
	return value, err
}

func (s *Store) listSources(
	ctx context.Context,
	rootID root.RootID,
) ([]source.Source, error) {
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+sourceColumns+`
		 FROM artifact_sources
		 WHERE root_id = ?
		   AND retired_at IS NULL
		 ORDER BY modified_at DESC, id ASC`,
		string(rootID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	output := make([]source.Source, 0)
	for rows.Next() {
		value, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		output = append(output, value.Clone())
	}
	return output, rows.Err()
}

func (s *Store) updateSource(
	ctx context.Context,
	value source.Source,
	expectedRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 ||
		value.Revision != expectedRevision+1 ||
		value.RetiredAt != nil {
		return fmt.Errorf(
			"%w: invalid Source update",
			basespec.ErrInvalid,
		)
	}
	discoveryRaw, err := encodeJSON(value.Discovery.Normalized())
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveRootTx(ctx, tx, value.RootID); err != nil {
		return err
	}
	current, err := getActiveSourceTx(
		ctx,
		tx,
		value.RootID,
		value.ID,
	)
	if err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return basespec.ErrConflict
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_sources
		 SET display_name = ?,
		     enabled = ?,
		     config_json = ?,
		     discovery_json = ?,
		     revision = ?,
		     modified_at = ?
		 WHERE root_id = ?
		   AND id = ?
		   AND revision = ?
		   AND retired_at IS NULL`,
		value.DisplayName,
		boolInt(value.Enabled),
		[]byte(value.Config),
		discoveryRaw,
		value.Revision,
		timeValue(value.ModifiedAt),
		string(value.RootID),
		string(value.ID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"Source changed during update",
	); err != nil {
		return err
	}
	if current.Enabled && !value.Enabled {
		_, err := tx.ExecContext(
			ctx,
			`DELETE FROM artifact_source_refresh_state
			 WHERE root_id = ? AND source_id = ?`,
			string(value.RootID),
			string(value.ID),
		)
		if err != nil {
			return sqliteError(err)
		}
		if err := markSourceArtifactsMissingTx(
			ctx,
			tx,
			value.RootID,
			value.ID,
			value.ModifiedAt,
			"artifact.source-disabled",
			"the Artifact Source was disabled",
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) retireSource(
	ctx context.Context,
	value source.Source,
	expectedRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if value.RetiredAt == nil ||
		value.Enabled ||
		expectedRevision == 0 ||
		value.Revision != expectedRevision+1 {
		return fmt.Errorf(
			"%w: invalid Source retirement",
			basespec.ErrInvalid,
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveRootTx(ctx, tx, value.RootID); err != nil {
		return err
	}
	current, err := getActiveSourceTx(
		ctx,
		tx,
		value.RootID,
		value.ID,
	)
	if err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return basespec.ErrConflict
	}
	if current.RootStorageKey != value.RootStorageKey ||
		current.StorageKey != value.StorageKey ||
		current.Kind != value.Kind {
		return fmt.Errorf(
			"%w: Source retirement changed Source identity",
			basespec.ErrInvalid,
		)
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_sources
		 SET enabled = 0,
		     revision = ?,
		     modified_at = ?,
		     retired_at = ?
		 WHERE root_id = ?
		   AND id = ?
		   AND revision = ?
		   AND retired_at IS NULL`,
		value.Revision,
		timeValue(value.ModifiedAt),
		timeValue(*value.RetiredAt),
		string(value.RootID),
		string(value.ID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"Source changed during retirement",
	); err != nil {
		return err
	}
	if err := markSourceArtifactsMissingTx(
		ctx,
		tx,
		value.RootID,
		value.ID,
		value.ModifiedAt,
		"artifact.source-retired",
		"the Artifact Source was retired",
	); err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		`DELETE FROM artifact_source_refresh_state
		 WHERE root_id = ? AND source_id = ?`,
		string(value.RootID),
		string(value.ID),
	)
	if err != nil {
		return sqliteError(err)
	}
	return tx.Commit()
}

func (s *Store) discardSource(
	ctx context.Context,
	rootID root.RootID,
	id source.SourceID,
	expectedRevision uint64,
) error {
	if err := rootID.Validate(); err != nil {
		return err
	}
	if err := id.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Source revision is required",
			basespec.ErrInvalid,
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := getActiveRootTx(ctx, tx, rootID); err != nil {
		return err
	}
	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_sources
		 WHERE root_id = ?
		   AND id = ?
		   AND revision = ?
		   AND retired_at IS NULL
		   AND NOT EXISTS (
			SELECT 1
			FROM artifact_artifacts
			WHERE root_id = ? AND source_id = ?
		   )
		   AND NOT EXISTS (
			SELECT 1
			FROM artifact_source_refresh_state
			WHERE root_id = ? AND source_id = ?
		   )`,
		string(rootID),
		string(id),
		expectedRevision,
		string(rootID),
		string(id),
		string(rootID),
		string(id),
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"Source changed or has synchronized Artifacts before discard",
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) purgeSource(
	ctx context.Context,
	rootID root.RootID,
	id source.SourceID,
	expectedRevision uint64,
) error {
	if err := rootID.Validate(); err != nil {
		return err
	}
	if err := id.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Source revision is required",
			basespec.ErrInvalid,
		)
	}
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM artifact_sources
		 WHERE root_id = ?
		   AND id = ?
		   AND revision = ?
		   AND retired_at IS NOT NULL`,
		string(rootID),
		string(id),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	return requireOneChanged(
		result,
		"Source changed, is not retired, or still owns Artifacts",
	)
}

func requireActiveSourceTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID root.RootID,
	sourceID source.SourceID,
) error {
	_, err := getActiveSourceTx(ctx, tx, rootID, sourceID)
	return err
}

func getActiveSourceTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID root.RootID,
	sourceID source.SourceID,
) (source.Source, error) {
	value, err := scanSource(tx.QueryRowContext(
		ctx,
		`SELECT `+sourceColumns+`
		 FROM artifact_sources
		 WHERE root_id = ?
		   AND id = ?
		   AND retired_at IS NULL`,
		string(rootID),
		string(sourceID),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return source.Source{}, fmt.Errorf(
			"%w: Source %q in Root %q",
			basespec.ErrSourceNotFound,
			sourceID,
			rootID,
		)
	}
	return value, err
}

func markSourceArtifactsMissingTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID root.RootID,
	sourceID source.SourceID,
	modifiedAt time.Time,
	code string,
	message string,
) error {
	if err := rootID.Validate(); err != nil {
		return err
	}
	diagnostics, err := encodeJSON([]diagnostic.Diagnostic{{
		Severity: diagnostic.SeverityWarning,
		Code:     code,
		Message:  message,
	}})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		`UPDATE artifact_artifacts
		 SET resolved_definition_digest = NULL,
		     source_content_digest = NULL,
		     state = ?,
		     diagnostics_json = ?,
		     revision = revision + 1,
		     modified_at = CASE
			WHEN modified_at >= ? THEN modified_at + 1
			ELSE ?
		     END
		 WHERE root_id = ?
		   AND source_id = ?
		   AND (
			state != ?
			OR resolved_definition_digest IS NOT NULL
			OR source_content_digest IS NOT NULL
		   )`,
		string(artifact.StateMissing),
		diagnostics,
		timeValue(modifiedAt),
		timeValue(modifiedAt),
		string(rootID),
		string(sourceID),
		string(artifact.StateMissing),
	)
	return sqliteError(err)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSource(
	row scanner,
) (source.Source, error) {
	var (
		id, rootID, rootStorageKey, storageKey, kind, displayName string
		enabled                                                   int
		config, discoveryRaw                                      []byte
		revision                                                  uint64
		createdAt, modifiedAt                                     int64
		retiredAt                                                 sql.NullInt64
	)
	if row == nil {
		return source.Source{}, fmt.Errorf(
			"%w: Source row is nil",
			basespec.ErrInvalid,
		)
	}
	if err := row.Scan(
		&id,
		&rootID,
		&rootStorageKey,
		&storageKey,
		&kind,
		&displayName,
		&enabled,
		&config,
		&discoveryRaw,
		&revision,
		&createdAt,
		&modifiedAt,
		&retiredAt,
	); err != nil {
		return source.Source{}, err
	}
	var discovery source.DiscoverySpec
	if err := decodeJSON(discoveryRaw, &discovery); err != nil {
		return source.Source{}, err
	}
	value := source.Source{
		ID:             source.SourceID(id),
		RootID:         root.RootID(rootID),
		RootStorageKey: basespec.StorageKey(rootStorageKey),
		StorageKey:     basespec.StorageKey(storageKey),
		Kind:           source.SourceKind(kind),
		DisplayName:    displayName,
		Enabled:        enabled != 0,
		Config:         append([]byte(nil), config...),
		Discovery:      discovery,
		Revision:       revision,
		CreatedAt:      parseTime(createdAt),
		ModifiedAt:     parseTime(modifiedAt),
		RetiredAt:      parseNullableTime(retiredAt),
	}
	if err := value.Validate(); err != nil {
		return source.Source{}, fmt.Errorf(
			"invalid persisted Source %q: %w",
			id,
			err,
		)
	}
	return value, nil
}
