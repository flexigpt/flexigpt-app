package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
)

const rootColumns = `
	id, kind, display_name, description, enabled, data_json,
	revision, created_at, modified_at, deleted_at`

const attachmentColumns = `
	root_id, source_id, role, enabled, data_json,
	revision, created_at, modified_at`

func (s *Store) requireActiveRoot(
	ctx context.Context,
	id artifactstore.RootID,
) error {
	var marker int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM artifact_roots
		 WHERE id = ? AND deleted_at IS NULL`,
		string(id),
	).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: root %q", artifactstore.ErrNotFound, id)
	}
	return err
}

func (s *Store) createRoot(
	ctx context.Context,
	value root.Root,
	attachments []root.Attachment,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	seenSources := make(map[artifactstore.SourceID]struct{}, len(attachments))
	for index, attachment := range attachments {
		if err := attachment.Validate(); err != nil {
			return err
		}
		if attachment.RootID != value.ID {
			return fmt.Errorf(
				"%w: attachment %d belongs to root %q, not root %q",
				artifactstore.ErrInvalid,
				index,
				attachment.RootID,
				value.ID,
			)
		}
		if _, duplicate := seenSources[attachment.SourceID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate attachment source %q",
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

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO artifact_roots (
			id, kind, display_name, description, enabled, data_json,
			revision, created_at, modified_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(value.ID),
		string(value.Kind),
		value.DisplayName,
		value.Description,
		boolInt(value.Enabled),
		[]byte(value.Data),
		value.Revision,
		timeValue(value.CreatedAt),
		timeValue(value.ModifiedAt),
		nullableTime(value.DeletedAt),
	)
	if err != nil {
		return sqliteError(err)
	}
	for _, attachment := range attachments {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO artifact_attachments (
				root_id, source_id, role, enabled, data_json,
				revision, created_at, modified_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			string(attachment.RootID),
			string(attachment.SourceID),
			string(attachment.Role),
			boolInt(attachment.Enabled),
			[]byte(attachment.Data),
			attachment.Revision,
			timeValue(attachment.CreatedAt),
			timeValue(attachment.ModifiedAt),
		); err != nil {
			return sqliteError(err)
		}
	}
	return tx.Commit()
}

func (s *Store) getRoot(
	ctx context.Context,
	id artifactstore.RootID,
) (root.Root, error) {
	query := `SELECT ` + rootColumns + `
		FROM artifact_roots WHERE id = ? AND deleted_at IS NULL`
	value, err := scanRoot(s.db.QueryRowContext(ctx, query, string(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return root.Root{}, fmt.Errorf(
			"%w: root %q",
			artifactstore.ErrNotFound,
			id,
		)
	}
	return value, err
}

func (s *Store) listRoots(
	ctx context.Context,
) ([]root.Root, error) {
	query := `SELECT ` + rootColumns + `
		FROM artifact_roots WHERE deleted_at IS NULL`
	query += ` ORDER BY modified_at DESC, id ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	output := make([]root.Root, 0)
	for rows.Next() {
		value, err := scanRoot(rows)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	return output, rows.Err()
}

func (s *Store) updateRoot(
	ctx context.Context,
	value root.Root,
	expectedRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE artifact_roots
		 SET display_name = ?,
		     description = ?,
		     enabled = ?,
		     data_json = ?,
		     revision = ?,
		     modified_at = ?,
		     deleted_at = ?
		 WHERE id = ? AND revision = ?`,
		value.DisplayName,
		value.Description,
		boolInt(value.Enabled),
		[]byte(value.Data),
		value.Revision,
		timeValue(value.ModifiedAt),
		nullableTime(value.DeletedAt),
		string(value.ID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf(
			"%w: root %q changed or no longer exists",
			artifactstore.ErrConflict,
			value.ID,
		)
	}
	return nil
}

func (s *Store) retireRoot(
	ctx context.Context,
	value root.Root,
	expectedRevision uint64,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 ||
		value.DeletedAt == nil ||
		value.Enabled {
		return fmt.Errorf(
			"%w: invalid root retirement state",
			artifactstore.ErrInvalid,
		)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_roots
		 SET enabled = 0,
		     revision = ?,
		     modified_at = ?,
		     deleted_at = ?
		 WHERE id = ? AND revision = ? AND deleted_at IS NULL`,
		value.Revision,
		timeValue(value.ModifiedAt),
		timeValue(*value.DeletedAt),
		string(value.ID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf(
			"%w: root %q changed or no longer exists",
			artifactstore.ErrConflict,
			value.ID,
		)
	}

	cleanupStatements := []string{
		`DELETE FROM artifact_current_occurrences WHERE root_id = ?`,
		`DELETE FROM artifact_current_catalogs WHERE root_id = ?`,
		`DELETE FROM artifact_records WHERE root_id = ?`,
		`DELETE FROM artifact_attachments WHERE root_id = ?`,
	}
	for _, statement := range cleanupStatements {
		if _, err := tx.ExecContext(ctx, statement, string(value.ID)); err != nil {
			return sqliteError(err)
		}
	}

	return tx.Commit()
}

func (s *Store) attach(
	ctx context.Context,
	attachment root.Attachment,
	expectedRootRevision uint64,
) (root.Root, error) {
	if err := attachment.Validate(); err != nil {
		return root.Root{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return root.Root{}, err
	}
	defer func() { _ = tx.Rollback() }()

	r, err := getRootTx(ctx, tx, attachment.RootID)
	if err != nil {
		return root.Root{}, err
	}
	if r.Revision != expectedRootRevision {
		return root.Root{}, artifactstore.ErrConflict
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO artifact_attachments (
			root_id, source_id, role, enabled, data_json,
			revision, created_at, modified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		string(attachment.RootID),
		string(attachment.SourceID),
		string(attachment.Role),
		boolInt(attachment.Enabled),
		[]byte(attachment.Data),
		attachment.Revision,
		timeValue(attachment.CreatedAt),
		timeValue(attachment.ModifiedAt),
	); err != nil {
		return root.Root{}, sqliteError(err)
	}
	r.Revision++
	r.ModifiedAt = attachment.ModifiedAt
	if err := updateRootRevisionTx(
		ctx,
		tx,
		r,
		expectedRootRevision,
	); err != nil {
		return root.Root{}, err
	}
	if err := tx.Commit(); err != nil {
		return root.Root{}, err
	}
	return r, nil
}

func (s *Store) getAttachment(
	ctx context.Context,
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
) (root.Attachment, error) {
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return root.Attachment{}, err
	}

	value, err := scanAttachment(s.db.QueryRowContext(
		ctx,
		`SELECT `+attachmentColumns+`
		 FROM artifact_attachments
		 WHERE root_id = ? AND source_id = ?`,
		string(rootID),
		string(sourceID),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return root.Attachment{}, fmt.Errorf(
			"%w: root/source attachment",
			artifactstore.ErrNotFound,
		)
	}
	return value, err
}

func (s *Store) listAttachments(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]root.Attachment, error) {
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+attachmentColumns+`
		 FROM artifact_attachments
		 WHERE root_id = ?
		 ORDER BY source_id ASC`,
		string(rootID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	output := make([]root.Attachment, 0)
	for rows.Next() {
		value, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	return output, rows.Err()
}

func (s *Store) updateAttachment(
	ctx context.Context,
	value root.Attachment,
	expectedRootRevision uint64,
	expectedAttachmentRevision uint64,
) (root.Root, error) {
	if err := value.Validate(); err != nil {
		return root.Root{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return root.Root{}, err
	}
	defer func() { _ = tx.Rollback() }()

	r, err := getRootTx(ctx, tx, value.RootID)
	if err != nil {
		return root.Root{}, err
	}
	if r.Revision != expectedRootRevision {
		return root.Root{}, artifactstore.ErrConflict
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_attachments
		 SET role = ?, enabled = ?, data_json = ?, revision = ?, modified_at = ?
		 WHERE root_id = ? AND source_id = ? AND revision = ?`,
		string(value.Role),
		boolInt(value.Enabled),
		[]byte(value.Data),
		value.Revision,
		timeValue(value.ModifiedAt),
		string(value.RootID),
		string(value.SourceID),
		expectedAttachmentRevision,
	)
	if err != nil {
		return root.Root{}, sqliteError(err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return root.Root{}, err
	}
	if changed != 1 {
		return root.Root{}, artifactstore.ErrConflict
	}

	r.Revision++
	r.ModifiedAt = value.ModifiedAt
	if err := updateRootRevisionTx(
		ctx,
		tx,
		r,
		expectedRootRevision,
	); err != nil {
		return root.Root{}, err
	}
	if err := tx.Commit(); err != nil {
		return root.Root{}, err
	}
	return r, nil
}

func (s *Store) detach(
	ctx context.Context,
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
	expectedRootRevision uint64,
	expectedAttachmentRevision uint64,
	modifiedAt time.Time,
) (root.Root, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return root.Root{}, err
	}
	defer func() { _ = tx.Rollback() }()

	r, err := getRootTx(ctx, tx, rootID)
	if err != nil {
		return root.Root{}, err
	}
	if r.Revision != expectedRootRevision {
		return root.Root{}, artifactstore.ErrConflict
	}

	result, err := tx.ExecContext(
		ctx,
		`DELETE FROM artifact_attachments
		 WHERE root_id = ? AND source_id = ? AND revision = ?`,
		string(rootID),
		string(sourceID),
		expectedAttachmentRevision,
	)
	if err != nil {
		return root.Root{}, sqliteError(err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return root.Root{}, err
	}
	if changed != 1 {
		return root.Root{}, artifactstore.ErrConflict
	}

	r.Revision++
	r.ModifiedAt = modifiedAt
	if err := updateRootRevisionTx(
		ctx,
		tx,
		r,
		expectedRootRevision,
	); err != nil {
		return root.Root{}, err
	}
	if err := tx.Commit(); err != nil {
		return root.Root{}, err
	}
	return r, nil
}

func scanAttachment(row scanner) (root.Attachment, error) {
	var (
		rootID, sourceID, role string
		enabled                int
		data                   []byte
		revision               uint64
		createdAt, modifiedAt  int64
	)
	if err := row.Scan(
		&rootID,
		&sourceID,
		&role,
		&enabled,
		&data,
		&revision,
		&createdAt,
		&modifiedAt,
	); err != nil {
		return root.Attachment{}, err
	}
	value := root.Attachment{
		RootID:     artifactstore.RootID(rootID),
		SourceID:   artifactstore.SourceID(sourceID),
		Role:       artifactstore.AttachmentRole(role),
		Enabled:    enabled != 0,
		Data:       append([]byte(nil), data...),
		Revision:   revision,
		CreatedAt:  parseTime(createdAt),
		ModifiedAt: parseTime(modifiedAt),
	}
	if err := value.Validate(); err != nil {
		return root.Attachment{}, err
	}
	return value, nil
}

func getRootTx(
	ctx context.Context,
	tx *sql.Tx,
	id artifactstore.RootID,
) (root.Root, error) {
	value, err := scanRoot(tx.QueryRowContext(
		ctx,
		`SELECT `+rootColumns+` FROM artifact_roots
		 WHERE id = ? AND deleted_at IS NULL`,
		string(id),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return root.Root{}, fmt.Errorf(
			"%w: root %q",
			artifactstore.ErrNotFound,
			id,
		)
	}
	return value, err
}

func scanRoot(row scanner) (root.Root, error) {
	var (
		id, kind, displayName, description string
		enabled                            int
		data                               []byte
		revision                           uint64
		createdAt, modifiedAt              int64
		deletedAt                          sql.NullInt64
	)
	if err := row.Scan(
		&id,
		&kind,
		&displayName,
		&description,
		&enabled,
		&data,
		&revision,
		&createdAt,
		&modifiedAt,
		&deletedAt,
	); err != nil {
		return root.Root{}, err
	}
	value := root.Root{
		ID:          artifactstore.RootID(id),
		Kind:        artifactstore.RootKind(kind),
		DisplayName: displayName,
		Description: description,
		Enabled:     enabled != 0,
		Data:        append([]byte(nil), data...),
		Revision:    revision,
		CreatedAt:   parseTime(createdAt),
		ModifiedAt:  parseTime(modifiedAt),
		DeletedAt:   parseNullableTime(deletedAt),
	}
	if err := value.Validate(); err != nil {
		return root.Root{}, err
	}
	return value, nil
}

func updateRootRevisionTx(
	ctx context.Context,
	tx *sql.Tx,
	value root.Root,
	expectedRevision uint64,
) error {
	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_roots
		 SET revision = ?, modified_at = ?
		 WHERE id = ? AND revision = ?`,
		value.Revision,
		timeValue(value.ModifiedAt),
		string(value.ID),
		expectedRevision,
	)
	if err != nil {
		return sqliteError(err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return artifactstore.ErrConflict
	}
	return nil
}
