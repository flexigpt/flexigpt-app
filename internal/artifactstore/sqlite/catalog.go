package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
)

const rootColumns = `
	id, kind, display_name, description, enabled, data_json,
	revision, created_at, modified_at, deleted_at`

const attachmentColumns = `
	root_id, source_id, role, priority, enabled, data_json,
	revision, created_at, modified_at`

const occurrenceColumns = `
	root_id, source_id, locator, subresource_locator,
	kind, logical_name, logical_version,
	definition_digest, source_content_digest, decoder_id,
	state, diagnostics_json, observed_at`

func (s *Store) createRoot(
	ctx context.Context,
	root catalog.Root,
	attachments []catalog.Attachment,
) error {
	if err := root.Validate(); err != nil {
		return err
	}
	for _, attachment := range attachments {
		if err := attachment.Validate(); err != nil {
			return err
		}
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
		string(root.ID),
		string(root.Kind),
		root.DisplayName,
		root.Description,
		boolInt(root.Enabled),
		[]byte(root.Data),
		root.Revision,
		timeValue(root.CreatedAt),
		timeValue(root.ModifiedAt),
		nullableTime(root.DeletedAt),
	)
	if err != nil {
		return sqliteError(err)
	}
	for _, attachment := range attachments {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO artifact_attachments (
				root_id, source_id, role, priority, enabled, data_json,
				revision, created_at, modified_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			string(attachment.RootID),
			string(attachment.SourceID),
			string(attachment.Role),
			attachment.Priority,
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
	includeDeleted bool,
) (catalog.Root, error) {
	query := `SELECT ` + rootColumns + ` FROM artifact_roots WHERE id = ?`
	if !includeDeleted {
		query += ` AND deleted_at IS NULL`
	}
	value, err := scanRoot(s.db.QueryRowContext(ctx, query, string(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.Root{}, fmt.Errorf(
			"%w: root %q",
			artifactstore.ErrNotFound,
			id,
		)
	}
	return value, err
}

func (s *Store) listRoots(
	ctx context.Context,
	includeDeleted bool,
) ([]catalog.Root, error) {
	query := `SELECT ` + rootColumns + ` FROM artifact_roots`
	if !includeDeleted {
		query += ` WHERE deleted_at IS NULL`
	}
	query += ` ORDER BY modified_at DESC, id ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	output := make([]catalog.Root, 0)
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
	value catalog.Root,
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

func (s *Store) attach(
	ctx context.Context,
	attachment catalog.Attachment,
	expectedRootRevision uint64,
) (catalog.Root, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return catalog.Root{}, err
	}
	defer func() { _ = tx.Rollback() }()

	root, err := getRootTx(ctx, tx, attachment.RootID)
	if err != nil {
		return catalog.Root{}, err
	}
	if root.Revision != expectedRootRevision {
		return catalog.Root{}, artifactstore.ErrConflict
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO artifact_attachments (
			root_id, source_id, role, priority, enabled, data_json,
			revision, created_at, modified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(attachment.RootID),
		string(attachment.SourceID),
		string(attachment.Role),
		attachment.Priority,
		boolInt(attachment.Enabled),
		[]byte(attachment.Data),
		attachment.Revision,
		timeValue(attachment.CreatedAt),
		timeValue(attachment.ModifiedAt),
	); err != nil {
		return catalog.Root{}, sqliteError(err)
	}
	root.Revision++
	root.ModifiedAt = attachment.ModifiedAt
	if err := updateRootRevisionTx(
		ctx,
		tx,
		root,
		expectedRootRevision,
	); err != nil {
		return catalog.Root{}, err
	}
	if err := tx.Commit(); err != nil {
		return catalog.Root{}, err
	}
	return root, nil
}

func (s *Store) getAttachment(
	ctx context.Context,
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
) (catalog.Attachment, error) {
	value, err := scanAttachment(s.db.QueryRowContext(
		ctx,
		`SELECT `+attachmentColumns+`
		 FROM artifact_attachments
		 WHERE root_id = ? AND source_id = ?`,
		string(rootID),
		string(sourceID),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.Attachment{}, fmt.Errorf(
			"%w: root/source attachment",
			artifactstore.ErrNotFound,
		)
	}
	return value, err
}

func (s *Store) listAttachments(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]catalog.Attachment, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+attachmentColumns+`
		 FROM artifact_attachments
		 WHERE root_id = ?
		 ORDER BY priority DESC, source_id ASC`,
		string(rootID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	output := make([]catalog.Attachment, 0)
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
	value catalog.Attachment,
	expectedRootRevision uint64,
	expectedAttachmentRevision uint64,
) (catalog.Root, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return catalog.Root{}, err
	}
	defer func() { _ = tx.Rollback() }()

	root, err := getRootTx(ctx, tx, value.RootID)
	if err != nil {
		return catalog.Root{}, err
	}
	if root.Revision != expectedRootRevision {
		return catalog.Root{}, artifactstore.ErrConflict
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE artifact_attachments
		 SET role = ?, priority = ?, enabled = ?, data_json = ?,
		     revision = ?, modified_at = ?
		 WHERE root_id = ? AND source_id = ? AND revision = ?`,
		string(value.Role),
		value.Priority,
		boolInt(value.Enabled),
		[]byte(value.Data),
		value.Revision,
		timeValue(value.ModifiedAt),
		string(value.RootID),
		string(value.SourceID),
		expectedAttachmentRevision,
	)
	if err != nil {
		return catalog.Root{}, sqliteError(err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return catalog.Root{}, err
	}
	if changed != 1 {
		return catalog.Root{}, artifactstore.ErrConflict
	}

	root.Revision++
	root.ModifiedAt = value.ModifiedAt
	if err := updateRootRevisionTx(
		ctx,
		tx,
		root,
		expectedRootRevision,
	); err != nil {
		return catalog.Root{}, err
	}
	if err := tx.Commit(); err != nil {
		return catalog.Root{}, err
	}
	return root, nil
}

func (s *Store) detach(
	ctx context.Context,
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
	expectedRootRevision uint64,
	expectedAttachmentRevision uint64,
	modifiedAt time.Time,
) (catalog.Root, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return catalog.Root{}, err
	}
	defer func() { _ = tx.Rollback() }()

	root, err := getRootTx(ctx, tx, rootID)
	if err != nil {
		return catalog.Root{}, err
	}
	if root.Revision != expectedRootRevision {
		return catalog.Root{}, artifactstore.ErrConflict
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
		return catalog.Root{}, sqliteError(err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return catalog.Root{}, err
	}
	if changed != 1 {
		return catalog.Root{}, artifactstore.ErrConflict
	}

	root.Revision++
	root.ModifiedAt = modifiedAt
	if err := updateRootRevisionTx(
		ctx,
		tx,
		root,
		expectedRootRevision,
	); err != nil {
		return catalog.Root{}, err
	}
	if err := tx.Commit(); err != nil {
		return catalog.Root{}, err
	}
	return root, nil
}

func (s *Store) getCurrentCatalog(
	ctx context.Context,
	rootID artifactstore.RootID,
) (catalog.Snapshot, error) {
	var (
		revision, rootRevision uint64
		sourceRevisionsRaw     []byte
		sourceGenerationsRaw   []byte
		publishedAt            int64
		diagnosticsRaw         []byte
	)
	err := s.db.QueryRowContext(
		ctx,
		`SELECT revision, root_revision, source_revisions_json,
		        source_generations_json, published_at, diagnostics_json
		 FROM artifact_current_catalogs
		 WHERE root_id = ?`,
		string(rootID),
	).Scan(
		&revision,
		&rootRevision,
		&sourceRevisionsRaw,
		&sourceGenerationsRaw,
		&publishedAt,
		&diagnosticsRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: root %q has no current catalog",
			artifactstore.ErrCatalogUnavailable,
			rootID,
		)
	}
	if err != nil {
		return catalog.Snapshot{}, err
	}

	sourceRevisions := map[artifactstore.SourceID]uint64{}
	sourceGenerations := map[artifactstore.SourceID]string{}
	diagnostics := []artifactstore.Diagnostic{}
	if err := decodeJSON(sourceRevisionsRaw, &sourceRevisions); err != nil {
		return catalog.Snapshot{}, err
	}
	if err := decodeJSON(sourceGenerationsRaw, &sourceGenerations); err != nil {
		return catalog.Snapshot{}, err
	}
	if err := decodeJSON(diagnosticsRaw, &diagnostics); err != nil {
		return catalog.Snapshot{}, err
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+occurrenceColumns+`
		 FROM artifact_current_occurrences
		 WHERE root_id = ?
		 ORDER BY source_id, locator, subresource_locator`,
		string(rootID),
	)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	defer rows.Close()

	occurrences := make([]catalog.Occurrence, 0)
	for rows.Next() {
		value, err := scanOccurrence(rows)
		if err != nil {
			return catalog.Snapshot{}, err
		}
		occurrences = append(occurrences, value)
	}
	if err := rows.Err(); err != nil {
		return catalog.Snapshot{}, err
	}

	value := catalog.Snapshot{
		RootID:            rootID,
		Revision:          revision,
		RootRevision:      rootRevision,
		SourceRevisions:   sourceRevisions,
		SourceGenerations: sourceGenerations,
		PublishedAt:       parseTime(publishedAt),
		Diagnostics:       diagnostics,
		Occurrences:       occurrences,
	}
	if err := value.Validate(); err != nil {
		return catalog.Snapshot{}, fmt.Errorf(
			"invalid persisted catalog: %w",
			err,
		)
	}
	return value, nil
}

func scanRoot(row scanner) (catalog.Root, error) {
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
		return catalog.Root{}, err
	}
	value := catalog.Root{
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
		return catalog.Root{}, err
	}
	return value, nil
}

func scanAttachment(row scanner) (catalog.Attachment, error) {
	var (
		rootID, sourceID, role string
		priority, enabled      int
		data                   []byte
		revision               uint64
		createdAt, modifiedAt  int64
	)
	if err := row.Scan(
		&rootID,
		&sourceID,
		&role,
		&priority,
		&enabled,
		&data,
		&revision,
		&createdAt,
		&modifiedAt,
	); err != nil {
		return catalog.Attachment{}, err
	}
	value := catalog.Attachment{
		RootID:     artifactstore.RootID(rootID),
		SourceID:   artifactstore.SourceID(sourceID),
		Role:       artifactstore.AttachmentRole(role),
		Priority:   priority,
		Enabled:    enabled != 0,
		Data:       append([]byte(nil), data...),
		Revision:   revision,
		CreatedAt:  parseTime(createdAt),
		ModifiedAt: parseTime(modifiedAt),
	}
	if err := value.Validate(); err != nil {
		return catalog.Attachment{}, err
	}
	return value, nil
}

func scanOccurrence(row scanner) (catalog.Occurrence, error) {
	var (
		rootID, sourceID, locator, subresource string
		kind, logicalName, logicalVersion      string
		definitionDigest, sourceDigest         sql.NullString
		decoderID, state                       string
		diagnosticsRaw                         []byte
		observedAt                             int64
	)
	if err := row.Scan(
		&rootID,
		&sourceID,
		&locator,
		&subresource,
		&kind,
		&logicalName,
		&logicalVersion,
		&definitionDigest,
		&sourceDigest,
		&decoderID,
		&state,
		&diagnosticsRaw,
		&observedAt,
	); err != nil {
		return catalog.Occurrence{}, err
	}
	diagnostics := []artifactstore.Diagnostic{}
	if err := decodeJSON(diagnosticsRaw, &diagnostics); err != nil {
		return catalog.Occurrence{}, err
	}
	value := catalog.Occurrence{
		RootID: artifactstore.RootID(rootID),
		Key: catalog.OccurrenceKey{
			SourceID:           artifactstore.SourceID(sourceID),
			Locator:            artifactstore.Locator(locator),
			SubresourceLocator: artifactstore.SubresourceLocator(subresource),
		},
		Kind:                artifactstore.ArtifactKind(kind),
		LogicalName:         artifactstore.LogicalName(logicalName),
		LogicalVersion:      artifactstore.LogicalVersion(logicalVersion),
		DefinitionDigest:    parseDigest(definitionDigest),
		SourceContentDigest: parseDigest(sourceDigest),
		DecoderID:           artifactstore.DecoderID(decoderID),
		State:               catalog.OccurrenceState(state),
		Diagnostics:         diagnostics,
		ObservedAt:          parseTime(observedAt),
	}
	if err := value.Validate(); err != nil {
		return catalog.Occurrence{}, err
	}
	return value, nil
}

func getRootTx(
	ctx context.Context,
	tx *sql.Tx,
	id artifactstore.RootID,
) (catalog.Root, error) {
	return scanRoot(tx.QueryRowContext(
		ctx,
		`SELECT `+rootColumns+` FROM artifact_roots
		 WHERE id = ? AND deleted_at IS NULL`,
		string(id),
	))
}

func updateRootRevisionTx(
	ctx context.Context,
	tx *sql.Tx,
	value catalog.Root,
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
