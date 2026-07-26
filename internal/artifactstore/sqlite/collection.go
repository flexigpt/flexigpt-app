package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
)

const collectionColumns = `
	id, root_id, kind, display_name, description, enabled, data_json,
	revision, created_at, modified_at, retired_at`

const attachmentColumns = `
	root_id, collection_id, source_id, role, enabled, data_json,
	revision, created_at, modified_at`

func (s *Store) createCollection(
	ctx context.Context,
	value collection.Collection,
	attachments []collection.Attachment,
) error {
	if err := value.Validate(); err != nil {
		return err
	}

	seenSources := make(map[artifactstore.SourceID]struct{}, len(attachments))
	for index, attachment := range attachments {
		if err := attachment.Validate(); err != nil {
			return fmt.Errorf("attachment %d: %w", index, err)
		}
		if attachment.RootID != value.RootID ||
			attachment.CollectionID != value.ID {
			return fmt.Errorf(
				"%w: attachment %d belongs to another collection",
				artifactstore.ErrInvalid,
				index,
			)
		}
		if attachment.Revision != 1 {
			return fmt.Errorf(
				"%w: initial attachment revision must be one",
				artifactstore.ErrInvalid,
			)
		}
		if _, duplicate := seenSources[attachment.SourceID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate collection attachment source %q",
				artifactstore.ErrInvalid,
				attachment.SourceID,
			)
		}
		seenSources[attachment.SourceID] = struct{}{}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveRootTx(ctx, tx, value.RootID); err != nil {
		return err
	}
	for _, attachment := range attachments {
		if err := requireAttachableSourceTx(
			ctx,
			tx,
			value.RootID,
			attachment.SourceID,
			attachment.Enabled,
		); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO artifact_collections (
			id, root_id, kind, display_name, description, enabled, data_json,
			revision, created_at, modified_at, retired_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(value.ID),
		string(value.RootID),
		string(value.Kind),
		value.DisplayName,
		value.Description,
		boolInt(value.Enabled),
		[]byte(value.Data),
		value.Revision,
		timeValue(value.CreatedAt),
		timeValue(value.ModifiedAt),
		nullableTime(value.RetiredAt),
	); err != nil {
		return sqliteError(err)
	}

	for _, attachment := range attachments {
		if err := insertCollectionAttachmentTx(ctx, tx, attachment); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) getCollection(
	ctx context.Context,
	ref artifactstore.CollectionRef,
) (collection.Collection, error) {
	if err := ref.Validate(); err != nil {
		return collection.Collection{}, err
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return collection.Collection{}, err
	}
	defer func() { _ = tx.Rollback() }()

	value, err := getActiveCollectionTx(ctx, tx, ref)
	if err != nil {
		return collection.Collection{}, err
	}
	if err := tx.Commit(); err != nil {
		return collection.Collection{}, err
	}
	return value.Clone(), nil
}

func (s *Store) getRetiredCollection(
	ctx context.Context,
	ref artifactstore.CollectionRef,
) (collection.Collection, error) {
	if err := ref.Validate(); err != nil {
		return collection.Collection{}, err
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return collection.Collection{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveRootTx(ctx, tx, ref.RootID); err != nil {
		return collection.Collection{}, err
	}
	value, err := scanCollection(tx.QueryRowContext(
		ctx,
		`SELECT `+collectionColumns+`
		 FROM artifact_collections
		 WHERE id = ?
		   AND root_id = ?
		   AND retired_at IS NOT NULL`,
		string(ref.CollectionID),
		string(ref.RootID),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return collection.Collection{}, fmt.Errorf(
			"%w: retired collection %q in root %q",
			artifactstore.ErrCollectionNotFound,
			ref.CollectionID,
			ref.RootID,
		)
	}
	if err != nil {
		return collection.Collection{}, err
	}
	if err := tx.Commit(); err != nil {
		return collection.Collection{}, err
	}
	return value.Clone(), nil
}

func (s *Store) listCollectionsByRoot(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]collection.Collection, error) {
	if err := artifactstore.ValidateRootID(rootID); err != nil {
		return nil, err
	}
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+collectionColumns+`
		 FROM artifact_collections
		 WHERE root_id = ? AND retired_at IS NULL
		 ORDER BY modified_at DESC, id ASC`,
		string(rootID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make([]collection.Collection, 0)
	for rows.Next() {
		value, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value.Clone())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return values, nil
}

func (s *Store) updateCollection(
	ctx context.Context,
	value collection.Collection,
	expectedRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 ||
		value.Revision != expectedRevision+1 ||
		value.RetiredAt != nil {
		return fmt.Errorf("%w: invalid collection update", artifactstore.ErrInvalid)
	}
	if err := s.requireActiveRoot(ctx, value.RootID); err != nil {
		return err
	}

	result, err := s.db.ExecContext(
		ctx,
		`UPDATE artifact_collections
		 SET display_name = ?,
		     description = ?,
		     enabled = ?,
		     data_json = ?,
		     revision = ?,
		     modified_at = ?
		 WHERE id = ?
		   AND root_id = ?
		   AND revision = ?
		   AND retired_at IS NULL`,
		value.DisplayName,
		value.Description,
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
	return requireOneChanged(result, "collection changed during update")
}

func (s *Store) retireCollection(
	ctx context.Context,
	value collection.Collection,
	expectedRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 ||
		value.RetiredAt == nil ||
		value.Enabled ||
		value.Revision != expectedRevision+1 {
		return fmt.Errorf("%w: invalid collection retirement", artifactstore.ErrInvalid)
	}
	if err := s.requireActiveRoot(ctx, value.RootID); err != nil {
		return err
	}

	result, err := s.db.ExecContext(
		ctx,
		`UPDATE artifact_collections
		 SET enabled = 0,
		     revision = ?,
		     modified_at = ?,
		     retired_at = ?
		 WHERE id = ?
		   AND root_id = ?
		   AND revision = ?
		   AND retired_at IS NULL`,
		value.Revision,
		timeValue(value.ModifiedAt),
		timeValue(*value.RetiredAt),
		string(value.ID),
		string(value.RootID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	return requireOneChanged(result, "collection changed during retirement")
}

func (s *Store) purgeCollection(
	ctx context.Context,
	ref artifactstore.CollectionRef,
	expectedRevision uint64,
) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected collection revision is required",
			artifactstore.ErrInvalid,
		)
	}

	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM artifact_collections
		 WHERE id = ?
		   AND root_id = ?
		   AND revision = ?
		   AND retired_at IS NOT NULL`,
		string(ref.CollectionID),
		string(ref.RootID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	return requireOneChanged(
		result,
		"collection changed or was not retired before purge",
	)
}

func (s *Store) attachCollectionSource(
	ctx context.Context,
	value collection.Attachment,
	expectedCollectionRevision uint64,
) (collection.Collection, error) {
	if err := value.Validate(); err != nil {
		return collection.Collection{}, err
	}
	if expectedCollectionRevision == 0 || value.Revision != 1 {
		return collection.Collection{}, fmt.Errorf(
			"%w: invalid collection attachment creation",
			artifactstore.ErrInvalid,
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return collection.Collection{}, err
	}
	defer func() { _ = tx.Rollback() }()

	ref := artifactstore.CollectionRef{
		RootID:       value.RootID,
		CollectionID: value.CollectionID,
	}
	current, err := getActiveCollectionTx(ctx, tx, ref)
	if err != nil {
		return collection.Collection{}, err
	}
	if current.Revision != expectedCollectionRevision {
		return collection.Collection{}, artifactstore.ErrConflict
	}
	if err := requireAttachableSourceTx(
		ctx,
		tx,
		value.RootID,
		value.SourceID,
		value.Enabled,
	); err != nil {
		return collection.Collection{}, err
	}
	if err := insertCollectionAttachmentTx(ctx, tx, value); err != nil {
		return collection.Collection{}, err
	}

	updated, err := advanceCollectionRevisionTx(
		ctx,
		tx,
		current,
		expectedCollectionRevision,
		value.ModifiedAt,
	)
	if err != nil {
		return collection.Collection{}, err
	}
	if err := tx.Commit(); err != nil {
		return collection.Collection{}, err
	}
	return updated.Clone(), nil
}

func (s *Store) getCollectionAttachment(
	ctx context.Context,
	ref artifactstore.CollectionRef,
	sourceID artifactstore.SourceID,
) (collection.Attachment, error) {
	if err := ref.Validate(); err != nil {
		return collection.Attachment{}, err
	}
	if err := artifactstore.ValidateSourceID(sourceID); err != nil {
		return collection.Attachment{}, err
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return collection.Attachment{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveCollectionTx(ctx, tx, ref); err != nil {
		return collection.Attachment{}, err
	}
	value, err := getCollectionAttachmentTx(ctx, tx, ref, sourceID)
	if err != nil {
		return collection.Attachment{}, err
	}
	if err := tx.Commit(); err != nil {
		return collection.Attachment{}, err
	}
	return value.Clone(), nil
}

func (s *Store) listCollectionAttachments(
	ctx context.Context,
	ref artifactstore.CollectionRef,
) ([]collection.Attachment, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := getActiveCollectionTx(ctx, tx, ref); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(
		ctx,
		`SELECT `+attachmentColumns+`
		 FROM artifact_collection_attachments
		 WHERE root_id = ? AND collection_id = ?
		 ORDER BY source_id ASC`,
		string(ref.RootID),
		string(ref.CollectionID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make([]collection.Attachment, 0)
	for rows.Next() {
		value, err := scanCollectionAttachment(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value.Clone())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return values, nil
}

func (s *Store) updateCollectionAttachment(
	ctx context.Context,
	value collection.Attachment,
	expectedCollectionRevision uint64,
	expectedAttachmentRevision uint64,
) (collection.Collection, error) {
	if err := value.Validate(); err != nil {
		return collection.Collection{}, err
	}
	if expectedCollectionRevision == 0 ||
		expectedAttachmentRevision == 0 ||
		value.Revision != expectedAttachmentRevision+1 {
		return collection.Collection{}, fmt.Errorf(
			"%w: invalid collection attachment update",
			artifactstore.ErrInvalid,
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return collection.Collection{}, err
	}
	defer func() { _ = tx.Rollback() }()

	ref := artifactstore.CollectionRef{
		RootID:       value.RootID,
		CollectionID: value.CollectionID,
	}
	currentCollection, err := getActiveCollectionTx(ctx, tx, ref)
	if err != nil {
		return collection.Collection{}, err
	}
	if currentCollection.Revision != expectedCollectionRevision {
		return collection.Collection{}, artifactstore.ErrConflict
	}
	currentAttachment, err := getCollectionAttachmentTx(
		ctx,
		tx,
		ref,
		value.SourceID,
	)
	if err != nil {
		return collection.Collection{}, err
	}
	if currentAttachment.Revision != expectedAttachmentRevision {
		return collection.Collection{}, artifactstore.ErrConflict
	}
	if value.RootID != currentAttachment.RootID ||
		value.CollectionID != currentAttachment.CollectionID ||
		value.SourceID != currentAttachment.SourceID ||
		!value.CreatedAt.Equal(currentAttachment.CreatedAt) ||
		!value.ModifiedAt.After(currentAttachment.ModifiedAt) {
		return collection.Collection{}, fmt.Errorf(
			"%w: collection attachment immutable fields changed",
			artifactstore.ErrInvalid,
		)
	}
	if err := requireAttachableSourceTx(
		ctx,
		tx,
		value.RootID,
		value.SourceID,
		value.Enabled,
	); err != nil {
		return collection.Collection{}, err
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_collection_attachments
		 SET role = ?,
		     enabled = ?,
		     data_json = ?,
		     revision = ?,
		     modified_at = ?
		 WHERE root_id = ?
		   AND collection_id = ?
		   AND source_id = ?
		   AND revision = ?`,
		string(value.Role),
		boolInt(value.Enabled),
		[]byte(value.Data),
		value.Revision,
		timeValue(value.ModifiedAt),
		string(value.RootID),
		string(value.CollectionID),
		string(value.SourceID),
		expectedAttachmentRevision,
	)
	if err != nil {
		return collection.Collection{}, sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"collection attachment changed during update",
	); err != nil {
		return collection.Collection{}, err
	}

	updated, err := advanceCollectionRevisionTx(
		ctx,
		tx,
		currentCollection,
		expectedCollectionRevision,
		value.ModifiedAt,
	)
	if err != nil {
		return collection.Collection{}, err
	}
	if err := tx.Commit(); err != nil {
		return collection.Collection{}, err
	}
	return updated.Clone(), nil
}

func (s *Store) detachCollectionSource(
	ctx context.Context,
	ref artifactstore.CollectionRef,
	sourceID artifactstore.SourceID,
	expectedCollectionRevision uint64,
	expectedAttachmentRevision uint64,
	modifiedAt time.Time,
) (collection.Collection, error) {
	if err := ref.Validate(); err != nil {
		return collection.Collection{}, err
	}
	if err := artifactstore.ValidateSourceID(sourceID); err != nil {
		return collection.Collection{}, err
	}
	if expectedCollectionRevision == 0 ||
		expectedAttachmentRevision == 0 ||
		modifiedAt.IsZero() {
		return collection.Collection{}, fmt.Errorf(
			"%w: invalid collection attachment detach",
			artifactstore.ErrInvalid,
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return collection.Collection{}, err
	}
	defer func() { _ = tx.Rollback() }()

	current, err := getActiveCollectionTx(ctx, tx, ref)
	if err != nil {
		return collection.Collection{}, err
	}
	if current.Revision != expectedCollectionRevision {
		return collection.Collection{}, artifactstore.ErrConflict
	}
	attachment, err := getCollectionAttachmentTx(ctx, tx, ref, sourceID)
	if err != nil {
		return collection.Collection{}, err
	}
	if attachment.Revision != expectedAttachmentRevision {
		return collection.Collection{}, artifactstore.ErrConflict
	}

	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_collection_attachments
		 WHERE root_id = ?
		   AND collection_id = ?
		   AND source_id = ?
		   AND revision = ?`,
		string(ref.RootID),
		string(ref.CollectionID),
		string(sourceID),
		expectedAttachmentRevision,
	)
	if err != nil {
		return collection.Collection{}, sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"collection attachment changed during detach",
	); err != nil {
		return collection.Collection{}, err
	}

	updated, err := advanceCollectionRevisionTx(
		ctx,
		tx,
		current,
		expectedCollectionRevision,
		modifiedAt,
	)
	if err != nil {
		return collection.Collection{}, err
	}
	if err := tx.Commit(); err != nil {
		return collection.Collection{}, err
	}
	return updated.Clone(), nil
}

func (s *Store) replaceCollectionAttachment(
	ctx context.Context,
	ref artifactstore.CollectionRef,
	previousSourceID artifactstore.SourceID,
	expectedPreviousRevision uint64,
	replacement collection.Attachment,
	expectedCollectionRevision uint64,
) (collection.Collection, error) {
	if err := ref.Validate(); err != nil {
		return collection.Collection{}, err
	}
	if err := artifactstore.ValidateSourceID(previousSourceID); err != nil {
		return collection.Collection{}, err
	}
	if err := replacement.Validate(); err != nil {
		return collection.Collection{}, err
	}
	if expectedCollectionRevision == 0 ||
		expectedPreviousRevision == 0 ||
		replacement.Revision != 1 ||
		replacement.RootID != ref.RootID ||
		replacement.CollectionID != ref.CollectionID {
		return collection.Collection{}, fmt.Errorf(
			"%w: invalid collection attachment replacement",
			artifactstore.ErrInvalid,
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return collection.Collection{}, err
	}
	defer func() { _ = tx.Rollback() }()

	current, err := getActiveCollectionTx(ctx, tx, ref)
	if err != nil {
		return collection.Collection{}, err
	}
	if current.Revision != expectedCollectionRevision {
		return collection.Collection{}, artifactstore.ErrConflict
	}
	previous, err := getCollectionAttachmentTx(
		ctx,
		tx,
		ref,
		previousSourceID,
	)
	if err != nil {
		return collection.Collection{}, err
	}
	if previous.Revision != expectedPreviousRevision {
		return collection.Collection{}, artifactstore.ErrConflict
	}
	if err := requireAttachableSourceTx(
		ctx,
		tx,
		replacement.RootID,
		replacement.SourceID,
		replacement.Enabled,
	); err != nil {
		return collection.Collection{}, err
	}

	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_collection_attachments
		 WHERE root_id = ?
		   AND collection_id = ?
		   AND source_id = ?
		   AND revision = ?`,
		string(ref.RootID),
		string(ref.CollectionID),
		string(previousSourceID),
		expectedPreviousRevision,
	)
	if err != nil {
		return collection.Collection{}, sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"previous collection attachment changed during replacement",
	); err != nil {
		return collection.Collection{}, err
	}
	if err := insertCollectionAttachmentTx(ctx, tx, replacement); err != nil {
		return collection.Collection{}, err
	}

	updated, err := advanceCollectionRevisionTx(
		ctx,
		tx,
		current,
		expectedCollectionRevision,
		replacement.ModifiedAt,
	)
	if err != nil {
		return collection.Collection{}, err
	}
	if err := tx.Commit(); err != nil {
		return collection.Collection{}, err
	}
	return updated.Clone(), nil
}

func getActiveCollectionTx(
	ctx context.Context,
	tx *sql.Tx,
	ref artifactstore.CollectionRef,
) (collection.Collection, error) {
	if err := ref.Validate(); err != nil {
		return collection.Collection{}, err
	}
	if _, err := getActiveRootTx(ctx, tx, ref.RootID); err != nil {
		return collection.Collection{}, err
	}

	value, err := scanCollection(tx.QueryRowContext(
		ctx,
		`SELECT `+collectionColumns+`
		 FROM artifact_collections
		 WHERE id = ? AND root_id = ? AND retired_at IS NULL`,
		string(ref.CollectionID),
		string(ref.RootID),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return collection.Collection{}, fmt.Errorf(
			"%w: collection %q in root %q",
			artifactstore.ErrCollectionNotFound,
			ref.CollectionID,
			ref.RootID,
		)
	}
	return value, err
}

// requireAttachedSourceTx verifies only structural membership and active
// Source existence. It intentionally does not require either the Source or
// attachment to be enabled because pinned Artifacts and suppressions can refer
// to disabled attachments.
func requireAttachedSourceTx(
	ctx context.Context,
	tx *sql.Tx,
	ref artifactstore.CollectionRef,
	sourceID artifactstore.SourceID,
) error {
	if err := artifactstore.ValidateSourceID(sourceID); err != nil {
		return err
	}
	if _, err := getActiveCollectionTx(ctx, tx, ref); err != nil {
		return err
	}

	var attached int
	err := tx.QueryRowContext(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM artifact_collection_attachments a
			JOIN artifact_sources s
			  ON s.root_id = a.root_id AND s.id = a.source_id
			WHERE a.root_id = ?
			  AND a.collection_id = ?
			  AND a.source_id = ?
			  AND s.retired_at IS NULL
		)`,
		string(ref.RootID),
		string(ref.CollectionID),
		string(sourceID),
	).Scan(&attached)
	if err != nil {
		return err
	}
	if attached == 0 {
		return fmt.Errorf(
			"%w: source %q is not attached to collection %q",
			artifactstore.ErrAttachmentNotFound,
			sourceID,
			ref.CollectionID,
		)
	}
	return nil
}

func requireAttachableSourceTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
	attachmentEnabled bool,
) error {
	var enabled int
	err := tx.QueryRowContext(
		ctx,
		`SELECT enabled
		 FROM artifact_sources
		 WHERE id = ? AND root_id = ? AND retired_at IS NULL`,
		string(sourceID),
		string(rootID),
	).Scan(&enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf(
			"%w: source %q in root %q",
			artifactstore.ErrSourceNotFound,
			sourceID,
			rootID,
		)
	}
	if err != nil {
		return err
	}
	if attachmentEnabled && enabled == 0 {
		return fmt.Errorf(
			"%w: enabled attachment cannot use disabled source %q",
			artifactstore.ErrConflict,
			sourceID,
		)
	}
	return nil
}

func insertCollectionAttachmentTx(
	ctx context.Context,
	tx *sql.Tx,
	value collection.Attachment,
) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO artifact_collection_attachments (
			root_id, collection_id, source_id, role, enabled, data_json,
			revision, created_at, modified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(value.RootID),
		string(value.CollectionID),
		string(value.SourceID),
		string(value.Role),
		boolInt(value.Enabled),
		[]byte(value.Data),
		value.Revision,
		timeValue(value.CreatedAt),
		timeValue(value.ModifiedAt),
	)
	if err != nil {
		return sqliteError(err)
	}
	return nil
}

func getCollectionAttachmentTx(
	ctx context.Context,
	tx *sql.Tx,
	ref artifactstore.CollectionRef,
	sourceID artifactstore.SourceID,
) (collection.Attachment, error) {
	value, err := scanCollectionAttachment(tx.QueryRowContext(
		ctx,
		`SELECT `+attachmentColumns+`
		 FROM artifact_collection_attachments
		 WHERE root_id = ? AND collection_id = ? AND source_id = ?`,
		string(ref.RootID),
		string(ref.CollectionID),
		string(sourceID),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return collection.Attachment{}, fmt.Errorf(
			"%w: source %q in collection %q",
			artifactstore.ErrAttachmentNotFound,
			sourceID,
			ref.CollectionID,
		)
	}
	return value, err
}

func advanceCollectionRevisionTx(
	ctx context.Context,
	tx *sql.Tx,
	current collection.Collection,
	expectedRevision uint64,
	modifiedAt time.Time,
) (collection.Collection, error) {
	if current.Revision != expectedRevision {
		return collection.Collection{}, artifactstore.ErrConflict
	}
	if current.Revision == ^uint64(0) {
		return collection.Collection{}, fmt.Errorf(
			"%w: collection revision is exhausted",
			artifactstore.ErrInvalid,
		)
	}
	modifiedAt = modifiedAt.UTC()
	if !modifiedAt.After(current.ModifiedAt) {
		return collection.Collection{}, fmt.Errorf(
			"%w: collection membership mutation time must advance",
			artifactstore.ErrInvalid,
		)
	}

	next := current
	next.Revision++
	next.ModifiedAt = modifiedAt
	if err := next.Validate(); err != nil {
		return collection.Collection{}, err
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_collections
		 SET revision = ?, modified_at = ?
		 WHERE id = ?
		   AND root_id = ?
		   AND revision = ?
		   AND retired_at IS NULL`,
		next.Revision,
		timeValue(next.ModifiedAt),
		string(next.ID),
		string(next.RootID),
		expectedRevision,
	)
	if err != nil {
		return collection.Collection{}, sqliteError(err)
	}
	if err := requireOneChanged(
		result,
		"collection changed during membership mutation",
	); err != nil {
		return collection.Collection{}, err
	}
	return next, nil
}

func scanCollection(row scanner) (collection.Collection, error) {
	var (
		id, rootID, kind, displayName, description string
		enabled                                    int
		data                                       []byte
		revision                                   uint64
		createdAt, modifiedAt                      int64
		retiredAt                                  sql.NullInt64
	)
	if err := row.Scan(
		&id,
		&rootID,
		&kind,
		&displayName,
		&description,
		&enabled,
		&data,
		&revision,
		&createdAt,
		&modifiedAt,
		&retiredAt,
	); err != nil {
		return collection.Collection{}, err
	}

	value := collection.Collection{
		ID:          artifactstore.CollectionID(id),
		RootID:      artifactstore.RootID(rootID),
		Kind:        artifactstore.CollectionKind(kind),
		DisplayName: displayName,
		Description: description,
		Enabled:     enabled != 0,
		Data:        append(json.RawMessage(nil), data...),
		Revision:    revision,
		CreatedAt:   parseTime(createdAt),
		ModifiedAt:  parseTime(modifiedAt),
		RetiredAt:   parseNullableTime(retiredAt),
	}
	if err := value.Validate(); err != nil {
		return collection.Collection{}, fmt.Errorf(
			"invalid persisted collection %q: %w",
			id,
			err,
		)
	}
	return value, nil
}

func scanCollectionAttachment(row scanner) (collection.Attachment, error) {
	var (
		rootID, collectionID, sourceID, role string
		enabled                              int
		data                                 []byte
		revision                             uint64
		createdAt, modifiedAt                int64
	)
	if err := row.Scan(
		&rootID,
		&collectionID,
		&sourceID,
		&role,
		&enabled,
		&data,
		&revision,
		&createdAt,
		&modifiedAt,
	); err != nil {
		return collection.Attachment{}, err
	}

	value := collection.Attachment{
		RootID:       artifactstore.RootID(rootID),
		CollectionID: artifactstore.CollectionID(collectionID),
		SourceID:     artifactstore.SourceID(sourceID),
		Role:         artifactstore.AttachmentRole(role),
		Enabled:      enabled != 0,
		Data:         append(json.RawMessage(nil), data...),
		Revision:     revision,
		CreatedAt:    parseTime(createdAt),
		ModifiedAt:   parseTime(modifiedAt),
	}
	if err := value.Validate(); err != nil {
		return collection.Attachment{}, fmt.Errorf(
			"invalid persisted collection attachment %q/%q/%q: %w",
			rootID,
			collectionID,
			sourceID,
			err,
		)
	}
	return value, nil
}
