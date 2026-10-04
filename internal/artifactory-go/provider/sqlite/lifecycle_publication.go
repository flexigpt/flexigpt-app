package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// PublishLifecycle applies Refresh's already-decided lifecycle command. It
// checks current applicability and commits Source, refresh-state, and Artifact
// changes as one transaction; it never chooses lifecycle policy itself.
func (p *Publisher) PublishLifecycle(ctx context.Context, publication refreshFlow.LifecyclePublication) error {
	if p == nil || p.store == nil {
		return spec.ErrClosed
	}
	if err := publication.Validate(); err != nil {
		return err
	}
	tx, err := p.store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := getActiveRootTx(ctx, tx, publication.Transition.Source.RootID); err != nil {
		return err
	}
	current, err := getActiveSourceTx(ctx, tx, publication.Transition.Source.RootID, publication.Transition.Source.ID)
	if err != nil {
		return err
	}
	if current.Revision != publication.Transition.ExpectedSourceRevision {
		return spec.ErrConflict
	}
	if err := validateLifecycleApplicability(current, publication.Transition); err != nil {
		return err
	}
	refreshRevision, err := refreshRevisionTx(
		ctx,
		tx,
		publication.Transition.Source.RootID,
		publication.Transition.Source.ID,
	)
	if err != nil {
		return err
	}
	if refreshRevision != publication.ExpectedRefreshRevision {
		return fmt.Errorf("%w: Source refresh state changed during lifecycle publication", spec.ErrConflict)
	}
	if err := applyLifecycleSourceTransitionTx(ctx, tx, publication.Transition); err != nil {
		return err
	}
	for _, update := range publication.ArtifactUpdates {
		if err := updateArtifactSourceStateTx(ctx, tx, update); err != nil {
			return err
		}
	}
	if publication.ExpectedRefreshRevision != 0 {
		result, err := tx.ExecContext(
			ctx,
			`DELETE FROM artifact_source_refresh_state WHERE root_id = ? AND source_id = ? AND revision = ?`,
			string(publication.Transition.Source.RootID),
			string(publication.Transition.Source.ID),
			publication.ExpectedRefreshRevision,
		)
		if err != nil {
			return sqliteError(err)
		}
		if err := requireOneChanged(result, "Source refresh state changed during lifecycle invalidation"); err != nil {
			return err
		}
	} else {
		var exists int
		if err := tx.QueryRowContext(
			ctx,
			`SELECT EXISTS(
				SELECT 1 FROM artifact_source_refresh_state
				WHERE root_id = ? AND source_id = ?
			)`,
			string(publication.Transition.Source.RootID),
			string(publication.Transition.Source.ID),
		).Scan(&exists); err != nil {
			return err
		}
		if exists != 0 {
			return fmt.Errorf("%w: Source refresh state appeared during lifecycle publication", spec.ErrConflict)
		}
	}
	return tx.Commit()
}

func refreshRevisionTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) (uint64, error) {
	var revision uint64
	err := tx.QueryRowContext(ctx, `SELECT revision FROM artifact_source_refresh_state WHERE root_id = ? AND source_id = ?`, string(rootID), string(sourceID)).
		Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return revision, err
}

func validateLifecycleApplicability(current sourceModel.Source, transition source.LifecycleTransition) error {
	if current.RootID != transition.Source.RootID || current.ID != transition.Source.ID ||
		current.RootStorageKey != transition.Source.RootStorageKey ||
		current.StorageKey != transition.Source.StorageKey ||
		current.Kind != transition.Source.Kind {
		return fmt.Errorf("%w: Source lifecycle transition changed Source identity", spec.ErrInvalid)
	}
	switch transition.Invalidation {
	case source.LifecycleInvalidationDisabled:
		if !current.Enabled || transition.Source.Enabled || transition.Source.RetiredAt != nil {
			return fmt.Errorf("%w: Source disable lifecycle transition is no longer applicable", spec.ErrConflict)
		}
	case source.LifecycleInvalidationDiscoveryRemoved:
		if !current.Enabled || current.Discovery.Empty() || !transition.Source.Enabled ||
			!transition.Source.Discovery.Empty() ||
			transition.Source.RetiredAt != nil {
			return fmt.Errorf(
				"%w: Source discovery-removal lifecycle transition is no longer applicable",
				spec.ErrConflict,
			)
		}
	case source.LifecycleInvalidationRetired:
		if transition.Source.Enabled || transition.Source.RetiredAt == nil {
			return fmt.Errorf("%w: Source retirement lifecycle transition is invalid", spec.ErrInvalid)
		}
	default:
		return fmt.Errorf("%w: unsupported Source lifecycle transition", spec.ErrInvalid)
	}
	return nil
}

func applyLifecycleSourceTransitionTx(ctx context.Context, tx *sql.Tx, transition source.LifecycleTransition) error {
	value := transition.Source
	discoveryRaw, err := encodeJSON(value.Discovery.Normalized())
	if err != nil {
		return err
	}
	var result sql.Result
	switch transition.Invalidation {
	case source.LifecycleInvalidationRetired:
		result, err = tx.ExecContext(
			ctx,
			`UPDATE artifact_sources SET enabled = 0, revision = ?, modified_at = ?, retired_at = ? WHERE root_id = ? AND id = ? AND revision = ? AND retired_at IS NULL`,
			value.Revision,
			timeValue(value.ModifiedAt),
			nullableTime(value.RetiredAt),
			string(value.RootID),
			string(value.ID),
			transition.ExpectedSourceRevision,
		)
	default:
		result, err = tx.ExecContext(
			ctx,
			`UPDATE artifact_sources SET display_name = ?, enabled = ?, config_json = ?, discovery_json = ?, revision = ?, modified_at = ? WHERE root_id = ? AND id = ? AND revision = ? AND retired_at IS NULL`,
			value.DisplayName,
			boolInt(value.Enabled),
			[]byte(value.Config),
			discoveryRaw,
			value.Revision,
			timeValue(value.ModifiedAt),
			string(value.RootID),
			string(value.ID),
			transition.ExpectedSourceRevision,
		)
	}
	if err != nil {
		return sqliteError(err)
	}
	return requireOneChanged(result, "Source changed during lifecycle publication")
}
