package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

const storeOverlayColumns = `
	namespace, schema_version, payload_json,
	revision, created_at, modified_at`

func (r *OverlayRepository) GetStoreOverlay(
	ctx context.Context,
	namespace overlayModel.Namespace,
) (overlayModel.StoreRecord, bool, error) {
	if r == nil || r.store == nil {
		return overlayModel.StoreRecord{}, false, spec.ErrClosed
	}
	if err := namespace.Validate(); err != nil {
		return overlayModel.StoreRecord{}, false, err
	}

	value, err := getStoreOverlayTx(
		ctx,
		r.store.db,
		namespace,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return overlayModel.StoreRecord{}, false, nil
	}
	if err != nil {
		return overlayModel.StoreRecord{}, false, err
	}
	return value.Clone(), true, nil
}

func (r *OverlayRepository) PutStoreOverlay(
	ctx context.Context,
	request overlayModel.StorePutRequest,
	now time.Time,
) (overlayModel.StoreRecord, error) {
	if r == nil || r.store == nil {
		return overlayModel.StoreRecord{}, spec.ErrClosed
	}
	if err := request.Validate(); err != nil {
		return overlayModel.StoreRecord{}, err
	}
	if now.IsZero() {
		return overlayModel.StoreRecord{}, fmt.Errorf(
			"%w: store overlay time is required",
			spec.ErrInvalid,
		)
	}

	payload, err := overlayModel.CanonicalPayload(request.Payload)
	if err != nil {
		return overlayModel.StoreRecord{}, err
	}
	request.Payload = payload

	tx, err := r.store.db.BeginTx(ctx, nil)
	if err != nil {
		return overlayModel.StoreRecord{}, err
	}
	defer func() { _ = tx.Rollback() }()

	current, err := getStoreOverlayTx(
		ctx,
		tx,
		request.Namespace,
	)

	var output overlayModel.StoreRecord
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if request.ExpectedRevision != 0 {
			return overlayModel.StoreRecord{}, spec.ErrConflict
		}

		output = overlayModel.StoreRecord{
			Namespace:     request.Namespace,
			SchemaVersion: request.SchemaVersion,
			Payload:       append([]byte(nil), request.Payload...),
			Revision:      1,
			CreatedAt:     now.UTC(),
			ModifiedAt:    now.UTC(),
		}
		if err := output.Validate(); err != nil {
			return overlayModel.StoreRecord{}, err
		}

		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO artifact_store_overlays (
				namespace, schema_version, payload_json,
				revision, created_at, modified_at
			) VALUES (?, ?, ?, ?, ?, ?)`,
			string(output.Namespace),
			output.SchemaVersion,
			[]byte(output.Payload),
			output.Revision,
			timeValue(output.CreatedAt),
			timeValue(output.ModifiedAt),
		)
		if err != nil {
			return overlayModel.StoreRecord{}, sqliteError(err)
		}

	case err != nil:
		return overlayModel.StoreRecord{}, err

	default:
		if current.Revision != request.ExpectedRevision {
			return overlayModel.StoreRecord{}, spec.ErrConflict
		}
		if current.Revision == ^uint64(0) {
			return overlayModel.StoreRecord{}, fmt.Errorf(
				"%w: store overlay revision is exhausted",
				spec.ErrInvalid,
			)
		}

		output = current.Clone()
		output.SchemaVersion = request.SchemaVersion
		output.Payload = append([]byte(nil), request.Payload...)
		output.Revision++
		output.ModifiedAt = now.UTC()

		if err := output.Validate(); err != nil {
			return overlayModel.StoreRecord{}, err
		}

		result, err := tx.ExecContext(
			ctx,
			`UPDATE artifact_store_overlays
			 SET schema_version = ?,
			     payload_json = ?,
			     revision = ?,
			     modified_at = ?
			 WHERE namespace = ?
			   AND revision = ?`,
			output.SchemaVersion,
			[]byte(output.Payload),
			output.Revision,
			timeValue(output.ModifiedAt),
			string(output.Namespace),
			current.Revision,
		)
		if err != nil {
			return overlayModel.StoreRecord{}, sqliteError(err)
		}
		if err := requireOneChanged(
			result,
			"store overlay changed during update",
		); err != nil {
			return overlayModel.StoreRecord{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return overlayModel.StoreRecord{}, err
	}
	return output.Clone(), nil
}

func (r *OverlayRepository) DeleteStoreOverlay(
	ctx context.Context,
	namespace overlayModel.Namespace,
	expectedRevision uint64,
) error {
	if r == nil || r.store == nil {
		return spec.ErrClosed
	}
	if err := namespace.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected store overlay revision is required",
			spec.ErrInvalid,
		)
	}

	result, err := r.store.db.ExecContext(
		ctx,
		`DELETE FROM artifact_store_overlays
		 WHERE namespace = ?
		   AND revision = ?`,
		string(namespace),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	return requireOneChanged(
		result,
		"store overlay changed during deletion",
	)
}

type storeOverlayQueryer interface {
	QueryRowContext(
		ctx context.Context,
		query string,
		args ...any,
	) *sql.Row
}

func getStoreOverlayTx(
	ctx context.Context,
	queryer storeOverlayQueryer,
	namespace overlayModel.Namespace,
) (overlayModel.StoreRecord, error) {
	return scanStoreOverlay(queryer.QueryRowContext(
		ctx,
		`SELECT `+storeOverlayColumns+`
		 FROM artifact_store_overlays
		 WHERE namespace = ?`,
		string(namespace),
	))
}

func scanStoreOverlay(
	row scanner,
) (overlayModel.StoreRecord, error) {
	if row == nil {
		return overlayModel.StoreRecord{}, fmt.Errorf(
			"%w: store overlay row is nil",
			spec.ErrInvalid,
		)
	}

	var (
		namespace, schemaVersion string
		payload                  []byte
		revision                 uint64
		createdAt, modifiedAt    int64
	)
	if err := row.Scan(
		&namespace,
		&schemaVersion,
		&payload,
		&revision,
		&createdAt,
		&modifiedAt,
	); err != nil {
		return overlayModel.StoreRecord{}, err
	}

	value := overlayModel.StoreRecord{
		Namespace:     overlayModel.Namespace(namespace),
		SchemaVersion: schemaVersion,
		Payload:       append([]byte(nil), payload...),
		Revision:      revision,
		CreatedAt:     parseTime(createdAt),
		ModifiedAt:    parseTime(modifiedAt),
	}
	if err := value.Validate(); err != nil {
		return overlayModel.StoreRecord{}, fmt.Errorf(
			"invalid persisted store overlay: %w",
			err,
		)
	}
	return value, nil
}
