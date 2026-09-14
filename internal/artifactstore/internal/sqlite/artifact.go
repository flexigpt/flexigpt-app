package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	artifactimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/artifact"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const artifactColumns = `
	id, root_id, source_id, locator, subresource_locator,
	kind, logical_name, logical_version,
	resolved_definition_digest, source_content_digest,
	state, diagnostics_json, display_name, enabled, data_json,
	revision, created_at, modified_at`

func (s *Store) getArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	if err := ref.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if err := s.requireActiveRoot(ctx, ref.RootID); err != nil {
		return artifact.Artifact{}, err
	}
	value, err := getArtifactTx(ctx, s.db, ref)
	if errors.Is(err, sql.ErrNoRows) {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: Artifact %q in Root %q",
			basespec.ErrArtifactNotFound,
			ref.ArtifactID,
			ref.RootID,
		)
	}
	if err != nil {
		return artifact.Artifact{}, err
	}
	return value.Clone(), nil
}

func (s *Store) listArtifactsByRoot(
	ctx context.Context,
	rootID root.RootID,
) ([]artifact.Artifact, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+artifactColumns+`
		 FROM artifact_artifacts
		 WHERE root_id = ?
		 ORDER BY modified_at DESC, id ASC`,
		string(rootID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArtifacts(rows)
}

func (s *Store) listArtifactsBySource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) ([]artifact.Artifact, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := sourceID.Validate(); err != nil {
		return nil, err
	}
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+artifactColumns+`
		 FROM artifact_artifacts
		 WHERE root_id = ? AND source_id = ?
		 ORDER BY locator, subresource_locator, kind, id`,
		string(rootID),
		string(sourceID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArtifacts(rows)
}

func (s *Store) findArtifactsByIdentity(
	ctx context.Context,
	rootID root.RootID,
	kind artifact.ArtifactKind,
	logicalName basespec.LogicalName,
) ([]artifact.Artifact, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := kind.Validate(); err != nil {
		return nil, err
	}
	if err := logicalName.Validate(); err != nil {
		return nil, err
	}
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+artifactColumns+`
		 FROM artifact_artifacts
		 WHERE root_id = ?
		   AND kind = ?
		   AND logical_name = ?
		 ORDER BY source_id, locator, subresource_locator, id`,
		string(rootID),
		string(kind),
		string(logicalName),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArtifacts(rows)
}

func (s *Store) findArtifactByOrigin(
	ctx context.Context,
	rootID root.RootID,
	binding artifact.SourceBinding,
	kind artifact.ArtifactKind,
) (artifact.Artifact, error) {
	if err := rootID.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if err := binding.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if err := kind.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return artifact.Artifact{}, err
	}
	value, err := scanArtifact(s.db.QueryRowContext(
		ctx,
		`SELECT `+artifactColumns+`
		 FROM artifact_artifacts
		 WHERE root_id = ?
		   AND source_id = ?
		   AND locator = ?
		   AND subresource_locator = ?
		   AND kind = ?`,
		string(rootID),
		string(binding.SourceID),
		string(binding.Locator),
		string(binding.SubresourceLocator),
		string(kind),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: Artifact origin %q/%q/%q",
			basespec.ErrArtifactNotFound,
			binding.SourceID,
			binding.Locator,
			binding.SubresourceLocator,
		)
	}
	if err != nil {
		return artifact.Artifact{}, err
	}
	return value.Clone(), nil
}

func (s *Store) createArtifact(
	ctx context.Context,
	value artifact.Artifact,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if value.Revision != 1 {
		return fmt.Errorf(
			"%w: initial Artifact revision must be one",
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
	if err := requireActiveSourceTx(
		ctx,
		tx,
		value.RootID,
		value.Binding.SourceID,
	); err != nil {
		return err
	}
	if err := insertArtifactTx(ctx, tx, value); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) updateArtifactLocal(
	ctx context.Context,
	value artifact.Artifact,
	expectedRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 ||
		value.Revision != expectedRevision+1 {
		return fmt.Errorf(
			"%w: invalid Artifact local update",
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
	current, err := getArtifactTx(ctx, tx, value.Ref())
	if err != nil {
		return artifactNotFound(err, value.Ref())
	}
	if current.Revision != expectedRevision {
		return basespec.ErrConflict
	}
	if current.RootID != value.RootID ||
		current.ID != value.ID {
		return fmt.Errorf(
			"%w: Artifact local update changed Artifact identity",
			basespec.ErrInvalid,
		)
	}
	if !value.ModifiedAt.After(current.ModifiedAt) {
		return fmt.Errorf(
			"%w: Artifact update time must advance current state",
			basespec.ErrInvalid,
		)
	}
	if !sameArtifactSourceFields(current, value) {
		return fmt.Errorf(
			"%w: local Artifact update attempted to change source-owned fields",
			basespec.ErrInvalid,
		)
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_artifacts
		 SET display_name = ?,
		     enabled = ?,
		     data_json = ?,
		     revision = ?,
		     modified_at = ?
		 WHERE id = ?
		   AND root_id = ?
		   AND revision = ?`,
		value.DisplayName,
		boolInt(value.Enabled),
		[]byte(value.Data),
		value.Revision,
		timeValue(value.ModifiedAt),
		string(value.ID),
		string(value.RootID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"Artifact changed during local update",
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) updateArtifactSourceState(
	ctx context.Context,
	update artifactimpl.SourceStateUpdate,
) error {
	if err := update.Validate(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := updateArtifactSourceStateTx(
		ctx,
		tx,
		update,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func updateArtifactSourceStateTx(
	ctx context.Context,
	tx *sql.Tx,
	update artifactimpl.SourceStateUpdate,
) error {
	current, err := getArtifactTx(ctx, tx, artifact.ArtifactRef{
		RootID:     update.RootID,
		ArtifactID: update.ArtifactID,
	})
	if err != nil {
		return artifactNotFound(err, artifact.ArtifactRef{
			RootID:     update.RootID,
			ArtifactID: update.ArtifactID,
		})
	}
	if current.Revision != update.ExpectedRevision {
		return basespec.ErrConflict
	}
	if current.Binding != update.Binding {
		return fmt.Errorf(
			"%w: source-derived Artifact update changed binding",
			basespec.ErrInvalid,
		)
	}
	if current.RootID != update.RootID ||
		current.Binding.SourceID != update.Binding.SourceID {
		return fmt.Errorf(
			"%w: source-derived Artifact update changed Source identity",
			basespec.ErrInvalid,
		)
	}
	if !update.ModifiedAt.After(current.ModifiedAt) {
		return fmt.Errorf(
			"%w: source-derived Artifact update time must advance",
			basespec.ErrInvalid,
		)
	}

	next := current.Clone()
	next.LogicalName = update.LogicalName
	next.LogicalVersion = update.LogicalVersion
	next.ResolvedDefinition = cryptoutil.CloneDigest(
		update.ResolvedDefinition,
	)
	next.SourceContentDigest = cryptoutil.CloneDigest(
		update.SourceContentDigest,
	)
	next.State = update.State
	next.Diagnostics = diagnostic.Clone(update.Diagnostics)
	next.Revision = update.Revision
	next.ModifiedAt = update.ModifiedAt
	if err := next.Validate(); err != nil {
		return err
	}

	diagnostics, err := encodeJSON(next.Diagnostics)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_artifacts
		 SET logical_name = ?,
		     logical_version = ?,
		     resolved_definition_digest = ?,
		     source_content_digest = ?,
		     state = ?,
		     diagnostics_json = ?,
		     revision = ?,
		     modified_at = ?
		 WHERE id = ?
		   AND root_id = ?
		   AND revision = ?`,
		string(next.LogicalName),
		string(next.LogicalVersion),
		nullableDigest(next.ResolvedDefinition),
		nullableDigest(next.SourceContentDigest),
		string(next.State),
		diagnostics,
		next.Revision,
		timeValue(next.ModifiedAt),
		string(next.ID),
		string(next.RootID),
		update.ExpectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	return requireOneChanged(
		result,
		"Artifact changed during source refresh",
	)
}

func (s *Store) purgeArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Artifact revision is required",
			basespec.ErrInvalid,
		)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := getActiveRootTx(ctx, tx, ref.RootID); err != nil {
		return err
	}
	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_artifacts
		 WHERE id = ?
		   AND root_id = ?
		   AND revision = ?`,
		string(ref.ArtifactID),
		string(ref.RootID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"Artifact changed during purge",
	); err != nil {
		return err
	}
	return tx.Commit()
}

func insertArtifactTx(
	ctx context.Context,
	tx *sql.Tx,
	value artifact.Artifact,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	diagnostics, err := encodeJSON(value.Diagnostics)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO artifact_artifacts (
			id, root_id, source_id, locator, subresource_locator,
			kind, logical_name, logical_version,
			resolved_definition_digest, source_content_digest,
			state, diagnostics_json, display_name, enabled, data_json,
			revision, created_at, modified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(value.ID),
		string(value.RootID),
		string(value.Binding.SourceID),
		string(value.Binding.Locator),
		string(value.Binding.SubresourceLocator),
		string(value.Kind),
		string(value.LogicalName),
		string(value.LogicalVersion),
		nullableDigest(value.ResolvedDefinition),
		nullableDigest(value.SourceContentDigest),
		string(value.State),
		diagnostics,
		value.DisplayName,
		boolInt(value.Enabled),
		[]byte(value.Data),
		value.Revision,
		timeValue(value.CreatedAt),
		timeValue(value.ModifiedAt),
	)
	return sqliteError(err)
}

type artifactQueryer interface {
	QueryRowContext(
		ctx context.Context,
		query string,
		args ...any,
	) *sql.Row
}

func getArtifactTx(
	ctx context.Context,
	queryer artifactQueryer,
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	return scanArtifact(queryer.QueryRowContext(
		ctx,
		`SELECT `+artifactColumns+`
		 FROM artifact_artifacts
		 WHERE id = ? AND root_id = ?`,
		string(ref.ArtifactID),
		string(ref.RootID),
	))
}

func scanArtifacts(
	rows *sql.Rows,
) ([]artifact.Artifact, error) {
	output := make([]artifact.Artifact, 0)
	for rows.Next() {
		value, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		output = append(output, value.Clone())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return output, nil
}

func scanArtifact(
	row scanner,
) (artifact.Artifact, error) {
	var (
		id, rootID, sourceID, locator, subresource string
		kind, logicalName, logicalVersion          string
		state, displayName                         string
		resolvedDefinition, sourceContent          sql.NullString
		diagnosticsRaw, dataRaw                    []byte
		enabled                                    int
		revision                                   uint64
		createdAt, modifiedAt                      int64
	)
	if row == nil {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: Artifact row is nil",
			basespec.ErrInvalid,
		)
	}
	if err := row.Scan(
		&id,
		&rootID,
		&sourceID,
		&locator,
		&subresource,
		&kind,
		&logicalName,
		&logicalVersion,
		&resolvedDefinition,
		&sourceContent,
		&state,
		&diagnosticsRaw,
		&displayName,
		&enabled,
		&dataRaw,
		&revision,
		&createdAt,
		&modifiedAt,
	); err != nil {
		return artifact.Artifact{}, err
	}
	var diagnostics []diagnostic.Diagnostic
	if err := decodeJSON(diagnosticsRaw, &diagnostics); err != nil {
		return artifact.Artifact{}, err
	}
	value := artifact.Artifact{
		ID:     artifact.ArtifactID(id),
		RootID: root.RootID(rootID),
		Binding: artifact.SourceBinding{
			SourceID:           source.SourceID(sourceID),
			Locator:            basespec.Locator(locator),
			SubresourceLocator: basespec.SubresourceLocator(subresource),
		},
		Kind:                artifact.ArtifactKind(kind),
		LogicalName:         basespec.LogicalName(logicalName),
		LogicalVersion:      basespec.LogicalVersion(logicalVersion),
		ResolvedDefinition:  parseDigest(resolvedDefinition),
		SourceContentDigest: parseDigest(sourceContent),
		State:               artifact.State(state),
		Diagnostics:         diagnostics,
		DisplayName:         displayName,
		Enabled:             enabled != 0,
		Data:                append(json.RawMessage(nil), dataRaw...),
		Revision:            revision,
		CreatedAt:           parseTime(createdAt),
		ModifiedAt:          parseTime(modifiedAt),
	}
	if err := value.Validate(); err != nil {
		return artifact.Artifact{}, fmt.Errorf(
			"invalid persisted Artifact %q: %w",
			id,
			err,
		)
	}
	return value, nil
}

func sameArtifactSourceFields(
	left artifact.Artifact,
	right artifact.Artifact,
) bool {
	return left.ID == right.ID &&
		left.RootID == right.RootID &&
		left.Binding == right.Binding &&
		left.Kind == right.Kind &&
		left.LogicalName == right.LogicalName &&
		left.LogicalVersion == right.LogicalVersion &&
		cryptoutil.IsDigestEqual(
			left.ResolvedDefinition,
			right.ResolvedDefinition,
		) &&
		cryptoutil.IsDigestEqual(
			left.SourceContentDigest,
			right.SourceContentDigest,
		) &&
		left.State == right.State &&
		diagnostic.Equal(left.Diagnostics, right.Diagnostics) &&
		left.CreatedAt.Equal(right.CreatedAt)
}

func artifactNotFound(
	err error,
	ref artifact.ArtifactRef,
) error {
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return fmt.Errorf(
		"%w: Artifact %q in Root %q",
		basespec.ErrArtifactNotFound,
		ref.ArtifactID,
		ref.RootID,
	)
}
