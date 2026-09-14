package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	refreshimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/refresh"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type Publisher struct {
	store *Store
}

func (s *Store) getRefreshState(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) (source.RefreshState, error) {
	if err := rootID.Validate(); err != nil {
		return source.RefreshState{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return source.RefreshState{}, err
	}
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return source.RefreshState{}, err
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
		return source.RefreshState{}, fmt.Errorf(
			"%w: Source %q in Root %q",
			basespec.ErrRefreshStateNotFound,
			sourceID,
			rootID,
		)
	}
	if err != nil {
		return source.RefreshState{}, err
	}
	return value.Clone(), nil
}

func (p *Publisher) Publish(
	ctx context.Context,
	publication refreshimpl.Publication,
) (source.RefreshState, error) {
	if p == nil || p.store == nil {
		return source.RefreshState{}, basespec.ErrClosed
	}
	if err := publication.Validate(); err != nil {
		return source.RefreshState{}, err
	}

	tx, err := p.store.db.BeginTx(ctx, nil)
	if err != nil {
		return source.RefreshState{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveRootTx(
		ctx,
		tx,
		publication.RootID,
	); err != nil {
		return source.RefreshState{}, err
	}
	currentSource, err := getActiveSourceTx(
		ctx,
		tx,
		publication.RootID,
		publication.SourceID,
	)
	if err != nil {
		return source.RefreshState{}, err
	}
	if !currentSource.Enabled ||
		currentSource.Revision != publication.ExpectedSourceRevision {
		return source.RefreshState{}, fmt.Errorf(
			"%w: Source changed or was disabled during refresh",
			basespec.ErrConflict,
		)
	}
	if currentSource.RootID != publication.RootID ||
		currentSource.ID != publication.SourceID {
		return source.RefreshState{}, fmt.Errorf(
			"%w: Source refresh publisher loaded another Source",
			basespec.ErrInvalid,
		)
	}
	if currentSource.Discovery.Empty() {
		return source.RefreshState{}, fmt.Errorf(
			"%w: Source has no declaration discovery configuration",
			basespec.ErrRefreshRequired,
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
		return source.RefreshState{}, err
	}
	if currentRefreshRevision != publication.ExpectedRefreshRevision {
		return source.RefreshState{}, fmt.Errorf(
			"%w: Source refresh state changed during refresh",
			basespec.ErrConflict,
		)
	}
	if currentRefreshRevision == ^uint64(0) {
		return source.RefreshState{}, fmt.Errorf(
			"%w: Source refresh revision is exhausted",
			basespec.ErrInvalid,
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
			return source.RefreshState{}, err
		}
	}
	for _, value := range publication.ArtifactCreates {
		if value.ResolvedDefinition == nil {
			return source.RefreshState{}, fmt.Errorf(
				"%w: source-created Artifact has no Definition",
				basespec.ErrInvalid,
			)
		}
		if _, err := getDefinitionTx(ctx, tx, publication.RootID, *value.ResolvedDefinition); err != nil {
			return source.RefreshState{}, err
		}
	}
	for _, value := range publication.ArtifactCreates {
		if err := requireActiveSourceTx(
			ctx,
			tx,
			publication.RootID,
			value.Binding.SourceID,
		); err != nil {
			return source.RefreshState{}, err
		}
		if err := insertArtifactTx(ctx, tx, value); err != nil {
			return source.RefreshState{}, err
		}
	}
	for _, update := range publication.ArtifactUpdates {
		if update.ResolvedDefinition == nil {
			continue
		}
		if _, err := getDefinitionTx(
			ctx,
			tx,
			publication.RootID,
			*update.ResolvedDefinition,
		); err != nil {
			return source.RefreshState{}, err
		}
	}
	for _, update := range publication.ArtifactUpdates {
		if err := updateArtifactSourceStateTx(
			ctx,
			tx,
			update,
		); err != nil {
			return source.RefreshState{}, err
		}
	}

	diagnostics, err := encodeJSON(publication.Diagnostics)
	if err != nil {
		return source.RefreshState{}, err
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
		return source.RefreshState{}, sqliteError(err)
	}
	if err := tx.Commit(); err != nil {
		return source.RefreshState{}, err
	}

	output := source.RefreshState{
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
		return source.RefreshState{}, err
	}
	return output.Clone(), nil
}

func scanRefreshState(
	row scanner,
) (source.RefreshState, error) {
	var (
		rootID, sourceID, generation             string
		discoveryFingerprint, decoderFingerprint string
		sourceRevision, revision                 uint64
		refreshedAt                              int64
		diagnosticsRaw                           []byte
	)
	if row == nil {
		return source.RefreshState{}, fmt.Errorf(
			"%w: Source refresh state row is nil",
			basespec.ErrInvalid,
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
		return source.RefreshState{}, err
	}
	var diagnostics []diagnostic.Diagnostic
	if err := decodeJSON(diagnosticsRaw, &diagnostics); err != nil {
		return source.RefreshState{}, err
	}
	value := source.RefreshState{
		RootID:               root.RootID(rootID),
		SourceID:             source.SourceID(sourceID),
		SourceRevision:       sourceRevision,
		SourceGeneration:     generation,
		DiscoveryFingerprint: cryptoutil.Digest(discoveryFingerprint),
		DecoderFingerprint:   cryptoutil.Digest(decoderFingerprint),
		Revision:             revision,
		RefreshedAt:          parseTime(refreshedAt),
		Diagnostics:          diagnostics,
	}
	if err := value.Validate(); err != nil {
		return source.RefreshState{}, fmt.Errorf(
			"invalid persisted Source refresh state: %w",
			err,
		)
	}
	return value, nil
}
