package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/overlay"
)

const storeOverlayColumns = `
	namespace, schema_version, payload_json,
	revision, created_at, modified_at`

func (r *LocalStateRepository) GetStoreOverlay(
	ctx context.Context,
	namespace overlay.Namespace,
) (overlay.StoreRecord, bool, error) {
	if r == nil || r.store == nil {
		return overlay.StoreRecord{}, false, basespec.ErrClosed
	}
	if err := namespace.Validate(); err != nil {
		return overlay.StoreRecord{}, false, err
	}

	value, err := getStoreOverlayTx(
		ctx,
		r.store.db,
		namespace,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return overlay.StoreRecord{}, false, nil
	}
	if err != nil {
		return overlay.StoreRecord{}, false, err
	}
	return value.Clone(), true, nil
}

func (r *LocalStateRepository) PutStoreOverlay(
	ctx context.Context,
	request overlay.StorePutRequest,
	now time.Time,
) (overlay.StoreRecord, error) {
	if r == nil || r.store == nil {
		return overlay.StoreRecord{}, basespec.ErrClosed
	}
	if err := request.Validate(); err != nil {
		return overlay.StoreRecord{}, err
	}
	if now.IsZero() {
		return overlay.StoreRecord{}, fmt.Errorf(
			"%w: store overlay time is required",
			basespec.ErrInvalid,
		)
	}

	payload, err := overlay.CanonicalPayload(request.Payload)
	if err != nil {
		return overlay.StoreRecord{}, err
	}
	request.Payload = payload

	tx, err := r.store.db.BeginTx(ctx, nil)
	if err != nil {
		return overlay.StoreRecord{}, err
	}
	defer func() { _ = tx.Rollback() }()

	current, err := getStoreOverlayTx(
		ctx,
		tx,
		request.Namespace,
	)

	var output overlay.StoreRecord
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if request.ExpectedRevision != 0 {
			return overlay.StoreRecord{}, basespec.ErrConflict
		}

		output = overlay.StoreRecord{
			Namespace:     request.Namespace,
			SchemaVersion: request.SchemaVersion,
			Payload:       append([]byte(nil), request.Payload...),
			Revision:      1,
			CreatedAt:     now.UTC(),
			ModifiedAt:    now.UTC(),
		}
		if err := output.Validate(); err != nil {
			return overlay.StoreRecord{}, err
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
			return overlay.StoreRecord{}, sqliteError(err)
		}

	case err != nil:
		return overlay.StoreRecord{}, err

	default:
		if current.Revision != request.ExpectedRevision {
			return overlay.StoreRecord{}, basespec.ErrConflict
		}
		if current.Revision == ^uint64(0) {
			return overlay.StoreRecord{}, fmt.Errorf(
				"%w: store overlay revision is exhausted",
				basespec.ErrInvalid,
			)
		}

		output = current.Clone()
		output.SchemaVersion = request.SchemaVersion
		output.Payload = append([]byte(nil), request.Payload...)
		output.Revision++
		output.ModifiedAt = now.UTC()

		if err := output.Validate(); err != nil {
			return overlay.StoreRecord{}, err
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
			return overlay.StoreRecord{}, sqliteError(err)
		}
		if err := requireOneChanged(
			result,
			"store overlay changed during update",
		); err != nil {
			return overlay.StoreRecord{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return overlay.StoreRecord{}, err
	}
	return output.Clone(), nil
}

func (r *LocalStateRepository) DeleteStoreOverlay(
	ctx context.Context,
	namespace overlay.Namespace,
	expectedRevision uint64,
) error {
	if r == nil || r.store == nil {
		return basespec.ErrClosed
	}
	if err := namespace.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected store overlay revision is required",
			basespec.ErrInvalid,
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
	namespace overlay.Namespace,
) (overlay.StoreRecord, error) {
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
) (overlay.StoreRecord, error) {
	if row == nil {
		return overlay.StoreRecord{}, fmt.Errorf(
			"%w: store overlay row is nil",
			basespec.ErrInvalid,
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
		return overlay.StoreRecord{}, err
	}

	value := overlay.StoreRecord{
		Namespace:     overlay.Namespace(namespace),
		SchemaVersion: schemaVersion,
		Payload:       append([]byte(nil), payload...),
		Revision:      revision,
		CreatedAt:     parseTime(createdAt),
		ModifiedAt:    parseTime(modifiedAt),
	}
	if err := value.Validate(); err != nil {
		return overlay.StoreRecord{}, fmt.Errorf(
			"invalid persisted store overlay: %w",
			err,
		)
	}
	return value, nil
}
