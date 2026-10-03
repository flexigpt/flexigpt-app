package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	secretimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/impl"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

const protectedOverlayColumns = `
	root_id, artifact_id, namespace, schema_version, payload_json,
	revision, created_at, modified_at`

type LocalStateRepository struct {
	store *Store
}

func (r *LocalStateRepository) GetOverlay(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	namespace overlayModel.Namespace,
) (overlayModel.Record, bool, error) {
	if r == nil || r.store == nil {
		return overlayModel.Record{}, false, spec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return overlayModel.Record{}, false, err
	}
	if err := namespace.Validate(); err != nil {
		return overlayModel.Record{}, false, err
	}
	if err := r.store.requireActiveRoot(ctx, ref.RootID); err != nil {
		return overlayModel.Record{}, false, err
	}

	value, err := getProtectedOverlayTx(
		ctx,
		r.store.db,
		ref,
		namespace,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return overlayModel.Record{}, false, nil
	}
	if err != nil {
		return overlayModel.Record{}, false, err
	}
	return value.Clone(), true, nil
}

func (r *LocalStateRepository) PutOverlay(
	ctx context.Context,
	request overlayModel.PutRequest,
	now time.Time,
) (overlayModel.Record, error) {
	if r == nil || r.store == nil {
		return overlayModel.Record{}, spec.ErrClosed
	}
	if err := request.Validate(); err != nil {
		return overlayModel.Record{}, err
	}
	if now.IsZero() {
		return overlayModel.Record{}, fmt.Errorf(
			"%w: protected overlay time is required",
			spec.ErrInvalid,
		)
	}

	payload, err := overlayModel.CanonicalPayload(request.Payload)
	if err != nil {
		return overlayModel.Record{}, err
	}
	request.Payload = payload

	tx, err := r.store.db.BeginTx(ctx, nil)
	if err != nil {
		return overlayModel.Record{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := ensureLocalStateArtifactTx(
		ctx,
		tx,
		request.Artifact,
		request.ExpectedArtifactRevision,
		true,
	); err != nil {
		return overlayModel.Record{}, err
	}

	current, err := getProtectedOverlayTx(
		ctx,
		tx,
		request.Artifact,
		request.Namespace,
	)

	var output overlayModel.Record
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if request.ExpectedOverlayRevision != 0 {
			return overlayModel.Record{}, spec.ErrConflict
		}
		output = overlayModel.Record{
			Artifact:      request.Artifact,
			Namespace:     request.Namespace,
			SchemaVersion: request.SchemaVersion,
			Payload:       append([]byte(nil), request.Payload...),
			Revision:      1,
			CreatedAt:     now.UTC(),
			ModifiedAt:    now.UTC(),
		}
		if err := output.Validate(); err != nil {
			return overlayModel.Record{}, err
		}

		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO artifact_protected_overlays (
				root_id, artifact_id, namespace, schema_version, payload_json,
				revision, created_at, modified_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			string(output.Artifact.RootID),
			string(output.Artifact.ArtifactID),
			string(output.Namespace),
			output.SchemaVersion,
			[]byte(output.Payload),
			output.Revision,
			timeValue(output.CreatedAt),
			timeValue(output.ModifiedAt),
		)
		if err != nil {
			return overlayModel.Record{}, sqliteError(err)
		}

	case err != nil:
		return overlayModel.Record{}, err

	default:
		if current.Revision != request.ExpectedOverlayRevision {
			return overlayModel.Record{}, spec.ErrConflict
		}
		if current.Revision == ^uint64(0) {
			return overlayModel.Record{}, fmt.Errorf(
				"%w: protected overlay revision is exhausted",
				spec.ErrInvalid,
			)
		}

		output = current.Clone()
		output.SchemaVersion = request.SchemaVersion
		output.Payload = append([]byte(nil), request.Payload...)
		output.Revision++
		output.ModifiedAt = now.UTC()

		if err := output.Validate(); err != nil {
			return overlayModel.Record{}, err
		}

		result, err := tx.ExecContext(
			ctx,
			`UPDATE artifact_protected_overlays
			 SET schema_version = ?,
			     payload_json = ?,
			     revision = ?,
			     modified_at = ?
			 WHERE root_id = ?
			   AND artifact_id = ?
			   AND namespace = ?
			   AND revision = ?`,
			output.SchemaVersion,
			[]byte(output.Payload),
			output.Revision,
			timeValue(output.ModifiedAt),
			string(output.Artifact.RootID),
			string(output.Artifact.ArtifactID),
			string(output.Namespace),
			current.Revision,
		)
		if err != nil {
			return overlayModel.Record{}, sqliteError(err)
		}
		if err := requireOneChanged(
			result,
			"protected overlay changed during update",
		); err != nil {
			return overlayModel.Record{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return overlayModel.Record{}, err
	}
	return output.Clone(), nil
}

func (r *LocalStateRepository) DeleteOverlay(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	namespace overlayModel.Namespace,
	expectedArtifactRevision uint64,
	expectedOverlayRevision uint64,
	now time.Time,
) error {
	if r == nil || r.store == nil {
		return spec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if err := namespace.Validate(); err != nil {
		return err
	}
	if expectedArtifactRevision == 0 ||
		expectedOverlayRevision == 0 ||
		now.IsZero() {
		return fmt.Errorf(
			"%w: invalid protected overlay deletion request",
			spec.ErrInvalid,
		)
	}

	tx, err := r.store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := ensureLocalStateArtifactTx(
		ctx,
		tx,
		ref,
		expectedArtifactRevision,
		true,
	); err != nil {
		return err
	}

	current, err := getProtectedOverlayTx(
		ctx,
		tx,
		ref,
		namespace,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return spec.ErrConflict
	}
	if err != nil {
		return err
	}
	if current.Revision != expectedOverlayRevision {
		return spec.ErrConflict
	}

	if err := queueNamespaceBindingSecretsTx(
		ctx,
		tx,
		ref,
		namespace,
		now.UTC(),
	); err != nil {
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

	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_protected_overlays
		 WHERE root_id = ?
		   AND artifact_id = ?
		   AND namespace = ?
		   AND revision = ?`,
		string(ref.RootID),
		string(ref.ArtifactID),
		string(namespace),
		expectedOverlayRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"protected overlay changed during deletion",
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *LocalStateRepository) GetBinding(
	ctx context.Context,
	key secretModel.BindingKey,
) (secretModel.Binding, bool, error) {
	if r == nil || r.store == nil {
		return secretModel.Binding{}, false, spec.ErrClosed
	}
	if err := key.Validate(); err != nil {
		return secretModel.Binding{}, false, err
	}
	if err := r.store.requireActiveRoot(
		ctx,
		key.Artifact.RootID,
	); err != nil {
		return secretModel.Binding{}, false, err
	}

	value, err := getSecretBindingTx(ctx, r.store.db, key)
	if errors.Is(err, sql.ErrNoRows) {
		return secretModel.Binding{}, false, nil
	}
	if err != nil {
		return secretModel.Binding{}, false, err
	}
	return value.Clone(), true, nil
}

func (r *LocalStateRepository) CreatePendingSecret(
	ctx context.Context,
	record secretModel.Record,
) error {
	if r == nil || r.store == nil {
		return spec.ErrClosed
	}
	if err := record.Validate(); err != nil {
		return err
	}
	if record.State != secretModel.RecordStatePending {
		return fmt.Errorf(
			"%w: new secret record must begin pending",
			spec.ErrInvalid,
		)
	}

	_, err := r.store.db.ExecContext(
		ctx,
		`INSERT INTO artifact_secret_records (
			ref, store_name, sha256, state, created_at, modified_at
		) VALUES (?, ?, ?, ?, ?, ?)`,
		string(record.Ref),
		record.StoreName,
		record.SHA256,
		string(record.State),
		timeValue(record.CreatedAt),
		timeValue(record.ModifiedAt),
	)
	return sqliteError(err)
}

func (r *LocalStateRepository) AttachSecretBinding(
	ctx context.Context,
	request secretimpl.AttachBindingRequest,
	now time.Time,
) (secretModel.Binding, error) {
	if r == nil || r.store == nil {
		return secretModel.Binding{}, spec.ErrClosed
	}
	if err := request.Key.Validate(); err != nil {
		return secretModel.Binding{}, err
	}
	if request.ExpectedArtifactRevision == 0 {
		return secretModel.Binding{}, fmt.Errorf(
			"%w: expected Artifact revision is required",
			spec.ErrInvalid,
		)
	}
	if err := request.Record.Validate(); err != nil {
		return secretModel.Binding{}, err
	}
	if request.Record.State != secretModel.RecordStatePending {
		return secretModel.Binding{}, fmt.Errorf(
			"%w: attached secret record must be pending",
			spec.ErrInvalid,
		)
	}
	if now.IsZero() {
		return secretModel.Binding{}, fmt.Errorf(
			"%w: secret binding time is required",
			spec.ErrInvalid,
		)
	}

	tx, err := r.store.db.BeginTx(ctx, nil)
	if err != nil {
		return secretModel.Binding{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := ensureLocalStateArtifactTx(
		ctx,
		tx,
		request.Key.Artifact,
		request.ExpectedArtifactRevision,
		true,
	); err != nil {
		return secretModel.Binding{}, err
	}

	record, err := getSecretRecordTx(
		ctx,
		tx,
		request.Record.Ref,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return secretModel.Binding{}, fmt.Errorf(
			"%w: staged secret record is unavailable",
			spec.ErrSecretNotFound,
		)
	}
	if err != nil {
		return secretModel.Binding{}, err
	}
	if record.State != secretModel.RecordStatePending ||
		record.StoreName != request.Record.StoreName ||
		record.SHA256 != request.Record.SHA256 {
		return secretModel.Binding{}, fmt.Errorf(
			"%w: staged secret record changed before binding publication",
			spec.ErrConflict,
		)
	}

	current, err := getSecretBindingTx(
		ctx,
		tx,
		request.Key,
	)
	found := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return secretModel.Binding{}, err
	}
	if found {
		if current.Revision != request.ExpectedBindingRevision {
			return secretModel.Binding{}, spec.ErrConflict
		}
	} else if request.ExpectedBindingRevision != 0 {
		return secretModel.Binding{}, spec.ErrConflict
	}

	if found && current.Ref != nil {
		if err := enqueueSecretRecordTx(
			ctx,
			tx,
			*current.Ref,
			now.UTC(),
		); err != nil {
			return secretModel.Binding{}, err
		}
	}

	ref := request.Record.Ref
	var output secretModel.Binding
	if found {
		if current.Revision == ^uint64(0) {
			return secretModel.Binding{}, fmt.Errorf(
				"%w: secret binding revision is exhausted",
				spec.ErrInvalid,
			)
		}
		output = current.Clone()
		output.Ref = &ref
		output.SHA256 = request.Record.SHA256
		output.Revision++
		output.ModifiedAt = now.UTC()

		result, err := tx.ExecContext(
			ctx,
			`UPDATE artifact_secret_bindings
			 SET secret_ref = ?,
			     revision = ?,
			     modified_at = ?
			 WHERE root_id = ?
			   AND artifact_id = ?
			   AND namespace = ?
			   AND slot = ?
			   AND revision = ?`,
			string(ref),
			output.Revision,
			timeValue(output.ModifiedAt),
			string(output.Key.Artifact.RootID),
			string(output.Key.Artifact.ArtifactID),
			string(output.Key.Namespace),
			string(output.Key.Slot),
			current.Revision,
		)
		if err != nil {
			return secretModel.Binding{}, sqliteError(err)
		}
		if err := requireOneChanged(
			result,
			"secret binding changed during replacement",
		); err != nil {
			return secretModel.Binding{}, err
		}
	} else {
		output = secretModel.Binding{
			Key:        request.Key,
			Ref:        &ref,
			SHA256:     request.Record.SHA256,
			Revision:   1,
			CreatedAt:  now.UTC(),
			ModifiedAt: now.UTC(),
		}
		_, err := tx.ExecContext(
			ctx,
			`INSERT INTO artifact_secret_bindings (
				root_id, artifact_id, namespace, slot, secret_ref,
				revision, created_at, modified_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			string(output.Key.Artifact.RootID),
			string(output.Key.Artifact.ArtifactID),
			string(output.Key.Namespace),
			string(output.Key.Slot),
			string(ref),
			output.Revision,
			timeValue(output.CreatedAt),
			timeValue(output.ModifiedAt),
		)
		if err != nil {
			return secretModel.Binding{}, sqliteError(err)
		}
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_secret_records
		 SET state = ?,
		     modified_at = ?
		 WHERE ref = ?
		   AND state = ?`,
		string(secretModel.RecordStateActive),
		timeValue(now.UTC()),
		string(ref),
		string(secretModel.RecordStatePending),
	)
	if err != nil {
		return secretModel.Binding{}, sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"staged secret record changed during binding publication",
	); err != nil {
		return secretModel.Binding{}, err
	}

	if err := output.Validate(); err != nil {
		return secretModel.Binding{}, err
	}
	if err := tx.Commit(); err != nil {
		return secretModel.Binding{}, err
	}
	return output.Clone(), nil
}

func (r *LocalStateRepository) ClearSecretBinding(
	ctx context.Context,
	request secretModel.ClearBindingRequest,
	now time.Time,
) error {
	if r == nil || r.store == nil {
		return spec.ErrClosed
	}
	if err := request.Validate(); err != nil {
		return err
	}
	if now.IsZero() {
		return fmt.Errorf(
			"%w: secret binding time is required",
			spec.ErrInvalid,
		)
	}

	tx, err := r.store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := ensureLocalStateArtifactTx(
		ctx,
		tx,
		request.Key.Artifact,
		request.ExpectedArtifactRevision,
		true,
	); err != nil {
		return err
	}

	current, err := getSecretBindingTx(ctx, tx, request.Key)
	if errors.Is(err, sql.ErrNoRows) {
		if request.ExpectedBindingRevision == 0 {
			return tx.Commit()
		}
		return spec.ErrConflict
	}
	if err != nil {
		return err
	}
	if current.Revision != request.ExpectedBindingRevision {
		return spec.ErrConflict
	}
	if current.Ref == nil {
		return tx.Commit()
	}
	if current.Revision == ^uint64(0) {
		return fmt.Errorf(
			"%w: secret binding revision is exhausted",
			spec.ErrInvalid,
		)
	}

	if err := enqueueSecretRecordTx(
		ctx,
		tx,
		*current.Ref,
		now.UTC(),
	); err != nil {
		return err
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_secret_bindings
		 SET secret_ref = NULL,
		     revision = ?,
		     modified_at = ?
		 WHERE root_id = ?
		   AND artifact_id = ?
		   AND namespace = ?
		   AND slot = ?
		   AND revision = ?`,
		current.Revision+1,
		timeValue(now.UTC()),
		string(current.Key.Artifact.RootID),
		string(current.Key.Artifact.ArtifactID),
		string(current.Key.Namespace),
		string(current.Key.Slot),
		current.Revision,
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"secret binding changed during clear",
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *LocalStateRepository) QueueSecretForCleanup(
	ctx context.Context,
	ref secretModel.Ref,
	now time.Time,
) error {
	if r == nil || r.store == nil {
		return spec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if now.IsZero() {
		return fmt.Errorf(
			"%w: secret cleanup time is required",
			spec.ErrInvalid,
		)
	}

	tx, err := r.store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := enqueueSecretRecordTx(ctx, tx, ref, now.UTC()); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *LocalStateRepository) PurgeArtifactLocalState(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	now time.Time,
) error {
	if r == nil || r.store == nil {
		return spec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if now.IsZero() {
		return fmt.Errorf(
			"%w: local-state purge time is required",
			spec.ErrInvalid,
		)
	}

	tx, err := r.store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := purgeArtifactLocalStateTx(
		ctx,
		tx,
		ref,
		now.UTC(),
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *LocalStateRepository) RecoverPendingSecrets(
	ctx context.Context,
	now time.Time,
) error {
	if r == nil || r.store == nil {
		return spec.ErrClosed
	}
	if now.IsZero() {
		return fmt.Errorf(
			"%w: secret recovery time is required",
			spec.ErrInvalid,
		)
	}

	tx, err := r.store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(
		ctx,
		`SELECT ref
		 FROM artifact_secret_records
		 WHERE state = ?
		 ORDER BY ref`,
		string(secretModel.RecordStatePending),
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	refs := make([]secretModel.Ref, 0)
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err != nil {
			return err
		}
		refs = append(refs, secretModel.Ref(ref))
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, ref := range refs {
		if err := enqueueSecretRecordTx(
			ctx,
			tx,
			ref,
			now.UTC(),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *LocalStateRepository) ListSecretCleanup(
	ctx context.Context,
	maximum int,
) ([]secretModel.Cleanup, error) {
	if r == nil || r.store == nil {
		return nil, spec.ErrClosed
	}
	if maximum <= 0 || maximum > spec.MaxDiscoveryEntries {
		return nil, fmt.Errorf(
			"%w: secret cleanup limit is invalid",
			spec.ErrInvalid,
		)
	}

	rows, err := r.store.db.QueryContext(
		ctx,
		`SELECT secret_ref, store_name, attempts, last_error,
		        created_at, modified_at
		 FROM artifact_secret_cleanup
		 ORDER BY modified_at, secret_ref
		 LIMIT ?`,
		maximum,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	output := make([]secretModel.Cleanup, 0)
	for rows.Next() {
		var (
			ref, storeName, lastError string
			attempts                  int
			createdAt, modifiedAt     int64
		)
		if err := rows.Scan(
			&ref,
			&storeName,
			&attempts,
			&lastError,
			&createdAt,
			&modifiedAt,
		); err != nil {
			return nil, err
		}

		value := secretModel.Cleanup{
			Ref:        secretModel.Ref(ref),
			StoreName:  storeName,
			Attempts:   attempts,
			LastError:  lastError,
			CreatedAt:  parseTime(createdAt),
			ModifiedAt: parseTime(modifiedAt),
		}
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf(
				"invalid persisted secret cleanup record: %w",
				err,
			)
		}
		output = append(output, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return output, nil
}

func (r *LocalStateRepository) CompleteSecretCleanup(
	ctx context.Context,
	ref secretModel.Ref,
) error {
	if r == nil || r.store == nil {
		return spec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return err
	}

	tx, err := r.store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	record, err := getSecretRecordTx(ctx, tx, ref)
	if errors.Is(err, sql.ErrNoRows) {
		return tx.Commit()
	}
	if err != nil {
		return err
	}
	if record.State != secretModel.RecordStateCleanup {
		return spec.ErrConflict
	}

	var bound int
	if err := tx.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM artifact_secret_bindings
		 WHERE secret_ref = ?`,
		string(ref),
	).Scan(&bound); err != nil {
		return err
	}
	if bound != 0 {
		return spec.ErrConflict
	}

	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_secret_records
		 WHERE ref = ?
		   AND state = ?`,
		string(ref),
		string(secretModel.RecordStateCleanup),
	)
	if err != nil {
		return sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"secret cleanup record changed during completion",
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *LocalStateRepository) RecordSecretCleanupFailure(
	ctx context.Context,
	ref secretModel.Ref,
	reason string,
	now time.Time,
) error {
	if r == nil || r.store == nil {
		return spec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if err := spec.ValidateRequiredText(
		"secret cleanup failure reason",
		reason,
		spec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if now.IsZero() {
		return fmt.Errorf(
			"%w: secret cleanup time is required",
			spec.ErrInvalid,
		)
	}

	_, err := r.store.db.ExecContext(
		ctx,
		`UPDATE artifact_secret_cleanup
		 SET attempts = attempts + 1,
		     last_error = ?,
		     modified_at = ?
		 WHERE secret_ref = ?`,
		reason,
		timeValue(now.UTC()),
		string(ref),
	)
	return sqliteError(err)
}

type localStateQueryer interface {
	QueryRowContext(
		ctx context.Context,
		query string,
		args ...any,
	) *sql.Row
}

func getProtectedOverlayTx(
	ctx context.Context,
	queryer localStateQueryer,
	ref artifactModel.ArtifactRef,
	namespace overlayModel.Namespace,
) (overlayModel.Record, error) {
	return scanProtectedOverlay(queryer.QueryRowContext(
		ctx,
		`SELECT `+protectedOverlayColumns+`
		 FROM artifact_protected_overlays
		 WHERE root_id = ?
		   AND artifact_id = ?
		   AND namespace = ?`,
		string(ref.RootID),
		string(ref.ArtifactID),
		string(namespace),
	))
}

func scanProtectedOverlay(
	row scanner,
) (overlayModel.Record, error) {
	if row == nil {
		return overlayModel.Record{}, fmt.Errorf(
			"%w: protected overlay row is nil",
			spec.ErrInvalid,
		)
	}

	var (
		rootID, artifactID, namespace, schemaVersion string
		payload                                      []byte
		revision                                     uint64
		createdAt, modifiedAt                        int64
	)
	if err := row.Scan(
		&rootID,
		&artifactID,
		&namespace,
		&schemaVersion,
		&payload,
		&revision,
		&createdAt,
		&modifiedAt,
	); err != nil {
		return overlayModel.Record{}, err
	}

	value := overlayModel.Record{
		Artifact: artifactModel.ArtifactRef{
			RootID:     rootModel.RootID(rootID),
			ArtifactID: artifactModel.ArtifactID(artifactID),
		},
		Namespace:     overlayModel.Namespace(namespace),
		SchemaVersion: schemaVersion,
		Payload:       append([]byte(nil), payload...),
		Revision:      revision,
		CreatedAt:     parseTime(createdAt),
		ModifiedAt:    parseTime(modifiedAt),
	}
	if err := value.Validate(); err != nil {
		return overlayModel.Record{}, fmt.Errorf(
			"invalid persisted protected overlay: %w",
			err,
		)
	}
	return value, nil
}

func getSecretBindingTx(
	ctx context.Context,
	queryer localStateQueryer,
	key secretModel.BindingKey,
) (secretModel.Binding, error) {
	return scanSecretBinding(queryer.QueryRowContext(
		ctx,
		`SELECT b.root_id,
		        b.artifact_id,
		        b.namespace,
		        b.slot,
		        b.secret_ref,
		        b.revision,
		        b.created_at,
		        b.modified_at,
		        r.sha256
		 FROM artifact_secret_bindings b
		 LEFT JOIN artifact_secret_records r
		   ON r.ref = b.secret_ref
		 WHERE b.root_id = ?
		   AND b.artifact_id = ?
		   AND b.namespace = ?
		   AND b.slot = ?`,
		string(key.Artifact.RootID),
		string(key.Artifact.ArtifactID),
		string(key.Namespace),
		string(key.Slot),
	))
}

func scanSecretBinding(
	row scanner,
) (secretModel.Binding, error) {
	if row == nil {
		return secretModel.Binding{}, fmt.Errorf(
			"%w: secret binding row is nil",
			spec.ErrInvalid,
		)
	}

	var (
		rootID, artifactID, namespace, slot string
		secretRef, sha256Value              sql.NullString
		revision                            uint64
		createdAt, modifiedAt               int64
	)
	if err := row.Scan(
		&rootID,
		&artifactID,
		&namespace,
		&slot,
		&secretRef,
		&revision,
		&createdAt,
		&modifiedAt,
		&sha256Value,
	); err != nil {
		return secretModel.Binding{}, err
	}

	value := secretModel.Binding{
		Key: secretModel.BindingKey{
			Artifact: artifactModel.ArtifactRef{
				RootID:     rootModel.RootID(rootID),
				ArtifactID: artifactModel.ArtifactID(artifactID),
			},
			Namespace: overlayModel.Namespace(namespace),
			Slot:      secretModel.Slot(slot),
		},
		Revision:   revision,
		CreatedAt:  parseTime(createdAt),
		ModifiedAt: parseTime(modifiedAt),
	}
	if secretRef.Valid {
		if !sha256Value.Valid {
			return secretModel.Binding{}, fmt.Errorf(
				"%w: active secret binding has no secret record",
				spec.ErrInvalid,
			)
		}
		ref := secretModel.Ref(secretRef.String)
		value.Ref = &ref
		value.SHA256 = sha256Value.String
	}
	if err := value.Validate(); err != nil {
		return secretModel.Binding{}, fmt.Errorf(
			"invalid persisted secret binding: %w",
			err,
		)
	}
	return value, nil
}

func getSecretRecordTx(
	ctx context.Context,
	queryer localStateQueryer,
	ref secretModel.Ref,
) (secretModel.Record, error) {
	return scanSecretRecord(queryer.QueryRowContext(
		ctx,
		`SELECT ref, store_name, sha256, state, created_at, modified_at
		 FROM artifact_secret_records
		 WHERE ref = ?`,
		string(ref),
	))
}

func scanSecretRecord(
	row scanner,
) (secretModel.Record, error) {
	if row == nil {
		return secretModel.Record{}, fmt.Errorf(
			"%w: secret record row is nil",
			spec.ErrInvalid,
		)
	}

	var (
		ref, storeName, sha256Value, state string
		createdAt, modifiedAt              int64
	)
	if err := row.Scan(
		&ref,
		&storeName,
		&sha256Value,
		&state,
		&createdAt,
		&modifiedAt,
	); err != nil {
		return secretModel.Record{}, err
	}

	value := secretModel.Record{
		Ref:        secretModel.Ref(ref),
		StoreName:  storeName,
		SHA256:     sha256Value,
		State:      secretModel.RecordState(state),
		CreatedAt:  parseTime(createdAt),
		ModifiedAt: parseTime(modifiedAt),
	}
	if err := value.Validate(); err != nil {
		return secretModel.Record{}, fmt.Errorf(
			"invalid persisted secret record: %w",
			err,
		)
	}
	return value, nil
}

func ensureLocalStateArtifactTx(
	ctx context.Context,
	tx *sql.Tx,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	requireAvailable bool,
) (artifactModel.Artifact, error) {
	if _, err := getActiveRootTx(ctx, tx, ref.RootID); err != nil {
		return artifactModel.Artifact{}, err
	}

	value, err := getArtifactTx(ctx, tx, ref)
	if err != nil {
		return artifactModel.Artifact{}, artifactNotFound(err, ref)
	}
	if expectedRevision != 0 &&
		value.Revision != expectedRevision {
		return artifactModel.Artifact{}, spec.ErrConflict
	}
	if requireAvailable && value.State != artifactModel.StateAvailable {
		return artifactModel.Artifact{}, fmt.Errorf(
			"%w: Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
			value.ID,
		)
	}
	return value, nil
}

func enqueueSecretRecordTx(
	ctx context.Context,
	tx *sql.Tx,
	ref secretModel.Ref,
	now time.Time,
) error {
	record, err := getSecretRecordTx(ctx, tx, ref)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf(
			"%w: secret record %q",
			spec.ErrSecretNotFound,
			ref,
		)
	}
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE artifact_secret_records
		 SET state = ?,
		     modified_at = ?
		 WHERE ref = ?`,
		string(secretModel.RecordStateCleanup),
		timeValue(now),
		string(record.Ref),
	)
	if err != nil {
		return sqliteError(err)
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO artifact_secret_cleanup (
			secret_ref, store_name, attempts, last_error,
			created_at, modified_at
		) VALUES (?, ?, 0, '', ?, ?)
		ON CONFLICT(secret_ref) DO NOTHING`,
		string(record.Ref),
		record.StoreName,
		timeValue(now),
		timeValue(now),
	)
	return sqliteError(err)
}

func queueNamespaceBindingSecretsTx(
	ctx context.Context,
	tx *sql.Tx,
	ref artifactModel.ArtifactRef,
	namespace overlayModel.Namespace,
	now time.Time,
) error {
	return queueBindingSecretsTx(
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
		now,
	)
}

func queueArtifactBindingSecretsTx(
	ctx context.Context,
	tx *sql.Tx,
	ref artifactModel.ArtifactRef,
	now time.Time,
) error {
	return queueBindingSecretsTx(
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
		now,
	)
}

func queueRootBindingSecretsTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID rootModel.RootID,
	now time.Time,
) error {
	return queueBindingSecretsTx(
		ctx,
		tx,
		`SELECT DISTINCT secret_ref
		 FROM artifact_secret_bindings
		 WHERE root_id = ?
		   AND secret_ref IS NOT NULL`,
		[]any{string(rootID)},
		now,
	)
}

func queueBindingSecretsTx(
	ctx context.Context,
	tx *sql.Tx,
	query string,
	args []any,
	now time.Time,
) error {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	refs := make([]secretModel.Ref, 0)
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err != nil {
			return err
		}
		refs = append(refs, secretModel.Ref(ref))
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, ref := range refs {
		if err := enqueueSecretRecordTx(ctx, tx, ref, now); err != nil {
			return err
		}
	}
	return nil
}

func purgeArtifactLocalStateTx(
	ctx context.Context,
	tx *sql.Tx,
	ref artifactModel.ArtifactRef,
	now time.Time,
) error {
	if _, err := ensureLocalStateArtifactTx(
		ctx,
		tx,
		ref,
		0,
		false,
	); err != nil {
		return err
	}

	if err := queueArtifactBindingSecretsTx(
		ctx,
		tx,
		ref,
		now,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_protected_overlays
		 WHERE root_id = ? AND artifact_id = ?`,
		string(ref.RootID),
		string(ref.ArtifactID),
	); err != nil {
		return sqliteError(err)
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
	return nil
}

// purgeRootLocalStateTx is called from protected-topology reset before the
// Root's Artifact rows are deleted. Cleanup rows intentionally outlive the
// Root and Artifact metadata they originated from.
func purgeRootLocalStateTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID rootModel.RootID,
	now time.Time,
) error {
	if err := queueRootBindingSecretsTx(
		ctx,
		tx,
		rootID,
		now,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_protected_overlays
		 WHERE root_id = ?`,
		string(rootID),
	); err != nil {
		return sqliteError(err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_secret_bindings
		 WHERE root_id = ?`,
		string(rootID),
	); err != nil {
		return sqliteError(err)
	}
	return nil
}
