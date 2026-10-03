package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	refreshimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh/impl"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type Publisher struct {
	store *Store
}

func (s *Store) getRefreshState(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) (sourceModel.RefreshState, error) {
	if err := rootID.Validate(); err != nil {
		return sourceModel.RefreshState{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return sourceModel.RefreshState{}, err
	}
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return sourceModel.RefreshState{}, err
	}
	value, err := scanRefreshState(s.db.QueryRowContext(
		ctx,
		`SELECT root_id, source_id, source_revision, source_generation,
		        discovery_fingerprint, decoder_fingerprint, revision,
		        refreshed_at, diagnostics_json
		 FROM artifact_source_refresh_state
		 WHERE root_id = ? AND source_id = ?`,
		string(rootID),
		string(sourceID),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return sourceModel.RefreshState{}, fmt.Errorf(
			"%w: Source %q in Root %q",
			spec.ErrRefreshStateNotFound,
			sourceID,
			rootID,
		)
	}
	if err != nil {
		return sourceModel.RefreshState{}, err
	}
	return value.Clone(), nil
}

func (p *Publisher) Publish(
	ctx context.Context,
	publication refreshimpl.Publication,
) (sourceModel.RefreshState, error) {
	if p == nil || p.store == nil {
		return sourceModel.RefreshState{}, spec.ErrClosed
	}
	if err := publication.Validate(); err != nil {
		return sourceModel.RefreshState{}, err
	}

	tx, err := p.store.db.BeginTx(ctx, nil)
	if err != nil {
		return sourceModel.RefreshState{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveRootTx(
		ctx,
		tx,
		publication.RootID,
	); err != nil {
		return sourceModel.RefreshState{}, err
	}
	currentSource, err := getActiveSourceTx(
		ctx,
		tx,
		publication.RootID,
		publication.SourceID,
	)
	if err != nil {
		return sourceModel.RefreshState{}, err
	}
	if !currentSource.Enabled ||
		currentSource.Revision != publication.ExpectedSourceRevision {
		return sourceModel.RefreshState{}, fmt.Errorf(
			"%w: Source changed or was disabled during refresh",
			spec.ErrConflict,
		)
	}
	if currentSource.RootID != publication.RootID ||
		currentSource.ID != publication.SourceID {
		return sourceModel.RefreshState{}, fmt.Errorf(
			"%w: Source refresh publisher loaded another Source",
			spec.ErrInvalid,
		)
	}
	if currentSource.Discovery.Empty() {
		return sourceModel.RefreshState{}, fmt.Errorf(
			"%w: Source has no declaration discovery configuration",
			spec.ErrRefreshRequired,
		)
	}
	var currentRefreshRevision uint64
	err = tx.QueryRowContext(
		ctx,
		`SELECT revision
		 FROM artifact_source_refresh_state
		 WHERE root_id = ? AND source_id = ?`,
		string(publication.RootID),
		string(publication.SourceID),
	).Scan(&currentRefreshRevision)
	if errors.Is(err, sql.ErrNoRows) {
		currentRefreshRevision = 0
	} else if err != nil {
		return sourceModel.RefreshState{}, err
	}
	if currentRefreshRevision != publication.ExpectedRefreshRevision {
		return sourceModel.RefreshState{}, fmt.Errorf(
			"%w: Source refresh state changed during refresh",
			spec.ErrConflict,
		)
	}
	if currentRefreshRevision == ^uint64(0) {
		return sourceModel.RefreshState{}, fmt.Errorf(
			"%w: Source refresh revision is exhausted",
			spec.ErrInvalid,
		)
	}

	for _, value := range publication.Definitions {
		if err := putDefinitionTx(
			ctx,
			tx,
			publication.RootID,
			value,
			publication.RefreshedAt,
		); err != nil {
			return sourceModel.RefreshState{}, err
		}
	}
	// Publication validation checks Artifact state. SQLite foreign keys
	// enforce Definition existence and the insert trigger checks Source
	// liveness. Neither requires decoding the same entities once per row.
	for _, value := range publication.ArtifactCreates {
		if err := insertArtifactTx(ctx, tx, value); err != nil {
			return sourceModel.RefreshState{}, err
		}
	}
	for _, update := range publication.ArtifactUpdates {
		if err := updateArtifactSourceStateTx(
			ctx,
			tx,
			update,
		); err != nil {
			return sourceModel.RefreshState{}, err
		}
	}

	diagnostics, err := encodeJSON(publication.Diagnostics)
	if err != nil {
		return sourceModel.RefreshState{}, err
	}
	nextRevision := currentRefreshRevision + 1
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO artifact_source_refresh_state (
			root_id, source_id, source_revision, source_generation,
			discovery_fingerprint, decoder_fingerprint, revision,
			refreshed_at, diagnostics_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(root_id, source_id) DO UPDATE SET
			source_revision = excluded.source_revision,
			source_generation = excluded.source_generation,
			discovery_fingerprint = excluded.discovery_fingerprint,
			decoder_fingerprint = excluded.decoder_fingerprint,
			revision = excluded.revision,
			refreshed_at = excluded.refreshed_at,
			diagnostics_json = excluded.diagnostics_json`,
		string(publication.RootID),
		string(publication.SourceID),
		publication.ExpectedSourceRevision,
		publication.SourceGeneration,
		string(publication.DiscoveryFingerprint),
		string(publication.DecoderFingerprint),
		nextRevision,
		timeValue(publication.RefreshedAt),
		diagnostics,
	)
	if err != nil {
		return sourceModel.RefreshState{}, sqliteError(err)
	}
	if err := tx.Commit(); err != nil {
		return sourceModel.RefreshState{}, err
	}

	output := sourceModel.RefreshState{
		RootID:               publication.RootID,
		SourceID:             publication.SourceID,
		SourceRevision:       publication.ExpectedSourceRevision,
		SourceGeneration:     publication.SourceGeneration,
		DiscoveryFingerprint: publication.DiscoveryFingerprint,
		DecoderFingerprint:   publication.DecoderFingerprint,
		Revision:             nextRevision,
		RefreshedAt:          publication.RefreshedAt,
		Diagnostics:          publication.Diagnostics,
	}
	if err := output.Validate(); err != nil {
		return sourceModel.RefreshState{}, err
	}
	return output.Clone(), nil
}

func scanRefreshState(
	row scanner,
) (sourceModel.RefreshState, error) {
	var (
		rootID, sourceID, generation             string
		discoveryFingerprint, decoderFingerprint string
		sourceRevision, revision                 uint64
		refreshedAt                              int64
		diagnosticsRaw                           []byte
	)
	if row == nil {
		return sourceModel.RefreshState{}, fmt.Errorf(
			"%w: Source refresh state row is nil",
			spec.ErrInvalid,
		)
	}
	if err := row.Scan(
		&rootID,
		&sourceID,
		&sourceRevision,
		&generation,
		&discoveryFingerprint,
		&decoderFingerprint,
		&revision,
		&refreshedAt,
		&diagnosticsRaw,
	); err != nil {
		return sourceModel.RefreshState{}, err
	}
	var diagnostics []diagnostic.Diagnostic
	if err := decodeJSON(diagnosticsRaw, &diagnostics); err != nil {
		return sourceModel.RefreshState{}, err
	}
	value := sourceModel.RefreshState{
		RootID:               rootModel.RootID(rootID),
		SourceID:             sourceModel.SourceID(sourceID),
		SourceRevision:       sourceRevision,
		SourceGeneration:     generation,
		DiscoveryFingerprint: cryptoutil.Digest(discoveryFingerprint),
		DecoderFingerprint:   cryptoutil.Digest(decoderFingerprint),
		Revision:             revision,
		RefreshedAt:          parseTime(refreshedAt),
		Diagnostics:          diagnostics,
	}
	if err := value.Validate(); err != nil {
		return sourceModel.RefreshState{}, fmt.Errorf(
			"invalid persisted Source refresh state: %w",
			err,
		)
	}
	return value, nil
}
