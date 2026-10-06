package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	artifactcleanupFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/artifactcleanup"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

func (r *ArtifactCleanupRepository) CleanupArtifactLocalState(
	ctx context.Context,
	request artifactcleanupFlow.PurgeRequest,
	now time.Time,
) (artifactModel.Artifact, error) {
	if now.IsZero() {
		return artifactModel.Artifact{}, fmt.Errorf(
			"%w: Artifact cleanup time is required",
			spec.ErrInvalid,
		)
	}

	tx, err := r.store.db.BeginTx(ctx, nil)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	defer func() { _ = tx.Rollback() }()

	current, err := ensureLocalStateArtifactTx(
		ctx,
		tx,
		request.Artifact,
		request.ExpectedArtifactRevision,
		false,
	)
	if err != nil {
		return artifactModel.Artifact{}, err
	}

	switch {
	case request.AllNamespaces:
		if err := detachArtifactBindingsTx(
			ctx,
			tx,
			request.Artifact,
			now.UTC(),
		); err != nil {
			return artifactModel.Artifact{}, err
		}
		if _, err := tx.ExecContext(
			ctx,
			`DELETE FROM artifact_protected_overlays
			 WHERE root_id = ? AND artifact_id = ?`,
			string(request.Artifact.RootID),
			string(request.Artifact.ArtifactID),
		); err != nil {
			return artifactModel.Artifact{}, sqliteError(err)
		}

	default:
		for _, namespace := range request.Namespaces {
			if err := detachNamespaceBindingsTx(
				ctx,
				tx,
				request.Artifact,
				namespace,
				now.UTC(),
			); err != nil {
				return artifactModel.Artifact{}, err
			}
			if _, err := tx.ExecContext(
				ctx,
				`DELETE FROM artifact_protected_overlays
				 WHERE root_id = ?
				   AND artifact_id = ?
				   AND namespace = ?`,
				string(request.Artifact.RootID),
				string(request.Artifact.ArtifactID),
				string(namespace),
			); err != nil {
				return artifactModel.Artifact{}, sqliteError(err)
			}
		}
	}

	next := current.Clone()
	nextData, changed, err := artifactModel.RemoveDataNamespaces(
		current.Data,
		request.DataNamespaces,
	)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if changed {
		if current.Revision == ^uint64(0) {
			return artifactModel.Artifact{}, fmt.Errorf(
				"%w: Artifact revision is exhausted",
				spec.ErrInvalid,
			)
		}
		next.Data = nextData
		next.Revision++
		next.ModifiedAt = clockutil.Advance(now.UTC(), current.ModifiedAt)
		if err := next.Validate(); err != nil {
			return artifactModel.Artifact{}, err
		}

		result, err := tx.ExecContext(
			ctx,
			`UPDATE artifact_artifacts
			 SET data_json = ?,
			     revision = ?,
			     modified_at = ?
			 WHERE root_id = ?
			   AND id = ?
			   AND revision = ?`,
			[]byte(next.Data),
			next.Revision,
			timeValue(next.ModifiedAt),
			string(next.RootID),
			string(next.ID),
			current.Revision,
		)
		if err != nil {
			return artifactModel.Artifact{}, sqliteError(err)
		}
		if err := requireOneChanged(
			result,
			"Artifact changed during local-state cleanup",
		); err != nil {
			return artifactModel.Artifact{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return artifactModel.Artifact{}, err
	}
	return next.Clone(), nil
}

func detachNamespaceBindingsTx(
	ctx context.Context,
	tx *sql.Tx,
	ref artifactModel.ArtifactRef,
	namespace overlayModel.Namespace,
	now time.Time,
) error {
	refs, err := bindingRefsTx(
		ctx,
		tx,
		`SELECT DISTINCT secret_ref
		 FROM artifact_secret_bindings
		 WHERE root_id = ?
		   AND artifact_id = ?
		   AND namespace = ?
		   AND secret_ref IS NOT NULL`,
		[]any{
			string(ref.RootID),
			string(ref.ArtifactID),
			string(namespace),
		},
	)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_secret_bindings
		 WHERE root_id = ?
		   AND artifact_id = ?
		   AND namespace = ?`,
		string(ref.RootID),
		string(ref.ArtifactID),
		string(namespace),
	); err != nil {
		return sqliteError(err)
	}

	for _, ref := range refs {
		if err := enqueueSecretRecordTx(ctx, tx, ref, now); err != nil {
			return err
		}
	}
	return nil
}

func detachRootBindingsTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID rootModel.RootID,
	now time.Time,
) error {
	refs, err := bindingRefsTx(
		ctx,
		tx,
		`SELECT DISTINCT secret_ref
		 FROM artifact_secret_bindings
		 WHERE root_id = ?
		   AND secret_ref IS NOT NULL`,
		[]any{string(rootID)},
	)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_secret_bindings
		 WHERE root_id = ?`,
		string(rootID),
	); err != nil {
		return sqliteError(err)
	}

	for _, ref := range refs {
		if err := enqueueSecretRecordTx(ctx, tx, ref, now); err != nil {
			return err
		}
	}
	return nil
}

func detachArtifactBindingsTx(
	ctx context.Context,
	tx *sql.Tx,
	ref artifactModel.ArtifactRef,
	now time.Time,
) error {
	refs, err := bindingRefsTx(
		ctx,
		tx,
		`SELECT DISTINCT secret_ref
		 FROM artifact_secret_bindings
		 WHERE root_id = ?
		   AND artifact_id = ?
		   AND secret_ref IS NOT NULL`,
		[]any{
			string(ref.RootID),
			string(ref.ArtifactID),
		},
	)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_secret_bindings
		 WHERE root_id = ? AND artifact_id = ?`,
		string(ref.RootID),
		string(ref.ArtifactID),
	); err != nil {
		return sqliteError(err)
	}

	for _, ref := range refs {
		if err := enqueueSecretRecordTx(ctx, tx, ref, now); err != nil {
			return err
		}
	}
	return nil
}

func bindingRefsTx(
	ctx context.Context,
	tx *sql.Tx,
	query string,
	args []any,
) ([]secretModel.Ref, error) {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	refs := make([]secretModel.Ref, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		refs = append(refs, secretModel.Ref(value))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return refs, nil
}
