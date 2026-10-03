package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
)

const sourceColumns = `
	id, root_id, root_storage_key, storage_key,
	kind, display_name, enabled, config_json, discovery_json,
	revision, created_at, modified_at, retired_at`

func (s *Store) createSource(
	ctx context.Context,
	value sourceModel.Source,
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
			spec.ErrInvalid,
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
	rootID rootModel.RootID,
	id sourceModel.SourceID,
) (sourceModel.Source, error) {
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return sourceModel.Source{}, err
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
		return sourceModel.Source{}, fmt.Errorf(
			"%w: Source %q in Root %q",
			spec.ErrSourceNotFound,
			id,
			rootID,
		)
	}
	return value, err
}

func (s *Store) findSourceByStorageKey(
	ctx context.Context,
	rootID rootModel.RootID,
	storageKey spec.StorageKey,
) (sourceModel.Source, error) {
	if err := rootID.Validate(); err != nil {
		return sourceModel.Source{}, err
	}
	if err := storageKey.Validate(); err != nil {
		return sourceModel.Source{}, err
	}
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return sourceModel.Source{}, err
	}

	value, err := scanSource(s.db.QueryRowContext(
		ctx,
		`SELECT `+sourceColumns+`
		 FROM artifact_sources
		 WHERE root_id = ? AND storage_key = ?`,
		string(rootID),
		string(storageKey),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return sourceModel.Source{}, fmt.Errorf(
			"%w: Source storage key %q in Root %q",
			spec.ErrSourceNotFound,
			storageKey,
			rootID,
		)
	}
	return value, err
}

func (s *Store) listSources(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]sourceModel.Source, error) {
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

	output := make([]sourceModel.Source, 0)
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
	value sourceModel.Source,
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
			spec.ErrInvalid,
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
		return spec.ErrConflict
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
	if current.Enabled &&
		(!value.Enabled || value.Discovery.Empty()) {
		code := "artifact.source-disabled"
		message := "the Artifact Source was disabled"
		if value.Enabled {
			code = "artifact.discovery-disabled"
			message = "the Artifact Source no longer has declaration discovery"
		}

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
			code,
			message,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) retireSource(
	ctx context.Context,
	value sourceModel.Source,
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
			spec.ErrInvalid,
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
		return spec.ErrConflict
	}
	if current.RootStorageKey != value.RootStorageKey ||
		current.StorageKey != value.StorageKey ||
		current.Kind != value.Kind {
		return fmt.Errorf(
			"%w: Source retirement changed Source identity",
			spec.ErrInvalid,
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
	rootID rootModel.RootID,
	id sourceModel.SourceID,
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
			spec.ErrInvalid,
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
	rootID rootModel.RootID,
	id sourceModel.SourceID,
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
			spec.ErrInvalid,
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
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) error {
	_, err := getActiveSourceTx(ctx, tx, rootID, sourceID)
	return err
}

func getActiveSourceTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) (sourceModel.Source, error) {
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
		return sourceModel.Source{}, fmt.Errorf(
			"%w: Source %q in Root %q",
			spec.ErrSourceNotFound,
			sourceID,
			rootID,
		)
	}
	return value, err
}

func markSourceArtifactsMissingTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
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
		string(artifactModel.StateMissing),
		diagnostics,
		timeValue(modifiedAt),
		timeValue(modifiedAt),
		string(rootID),
		string(sourceID),
		string(artifactModel.StateMissing),
	)
	return sqliteError(err)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSource(
	row scanner,
) (sourceModel.Source, error) {
	var (
		id, rootID, rootStorageKey, storageKey, kind, displayName string
		enabled                                                   int
		config, discoveryRaw                                      []byte
		revision                                                  uint64
		createdAt, modifiedAt                                     int64
		retiredAt                                                 sql.NullInt64
	)
	if row == nil {
		return sourceModel.Source{}, fmt.Errorf(
			"%w: Source row is nil",
			spec.ErrInvalid,
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
		return sourceModel.Source{}, err
	}
	var discovery sourceModel.DiscoverySpec
	if err := decodeJSON(discoveryRaw, &discovery); err != nil {
		return sourceModel.Source{}, err
	}
	value := sourceModel.Source{
		ID:             sourceModel.SourceID(id),
		RootID:         rootModel.RootID(rootID),
		RootStorageKey: spec.StorageKey(rootStorageKey),
		StorageKey:     spec.StorageKey(storageKey),
		Kind:           sourceModel.SourceKind(kind),
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
		return sourceModel.Source{}, fmt.Errorf(
			"invalid persisted Source %q: %w",
			id,
			err,
		)
	}
	return value, nil
}
