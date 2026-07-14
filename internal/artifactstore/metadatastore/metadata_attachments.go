package metadatastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

const rootSourceAttachmentColumns = `
	root_id,
	source_id,
	role,
	priority,
	enabled,
	data_schema_id,
	data_json,
	created_at,
	modified_at`

func (s *MetadataStore) CreateRootSourceAttachment(ctx context.Context, attachment spec.RootSourceAttachment) error {
	if err := spec.ValidateRootSourceAttachment(attachment); err != nil {
		return fmt.Errorf("validate root source attachment for persistence: %w", err)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO root_source_attachments (
			root_id, source_id, role, priority, enabled, data_schema_id, data_json, created_at, modified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(attachment.RootID),
		string(attachment.SourceID),
		string(attachment.Role),
		attachment.Priority,
		boolToInt(attachment.Enabled),
		string(attachment.DataSchemaID),
		[]byte(attachment.Data),
		formatTime(attachment.CreatedAt),
		formatTime(attachment.ModifiedAt),
	)
	if err != nil {
		return sqliteError(fmt.Errorf("insert root source attachment: %w", err))
	}
	return nil
}

func (s *MetadataStore) GetRootSourceAttachment(
	ctx context.Context,
	rootID spec.RootID,
	sourceID spec.SourceID,
) (spec.RootSourceAttachment, error) {
	attachment, err := scanRootSourceAttachment(s.db.QueryRowContext(
		ctx,
		`SELECT `+rootSourceAttachmentColumns+` FROM root_source_attachments WHERE root_id = ? AND source_id = ?`,
		string(rootID),
		string(sourceID),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return spec.RootSourceAttachment{}, fmt.Errorf(
			"%w: root/source attachment %q/%q",
			spec.ErrNotFound,
			rootID,
			sourceID,
		)
	}
	if err != nil {
		return spec.RootSourceAttachment{}, err
	}
	return attachment, nil
}

func (s *MetadataStore) ListRootSourceAttachments(
	ctx context.Context,
	rootID spec.RootID,
) ([]spec.RootSourceAttachment, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+rootSourceAttachmentColumns+` FROM root_source_attachments WHERE root_id = ? ORDER BY priority DESC, source_id ASC`,
		string(rootID),
	)
	if err != nil {
		return nil, fmt.Errorf("list root source attachments: %w", err)
	}
	defer rows.Close()

	attachments := make([]spec.RootSourceAttachment, 0)
	for rows.Next() {
		attachment, err := scanRootSourceAttachment(rows)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, attachment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate root source attachments: %w", err)
	}
	return attachments, nil
}

func (s *MetadataStore) UpdateRootSourceAttachment(ctx context.Context, attachment spec.RootSourceAttachment) error {
	if err := spec.ValidateRootSourceAttachment(attachment); err != nil {
		return fmt.Errorf("validate root source attachment for persistence: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE root_source_attachments
		   SET role = ?,
		       priority = ?,
		       enabled = ?,
		       data_schema_id = ?,
		       data_json = ?,
		       modified_at = ?
		 WHERE root_id = ? AND source_id = ?`,
		string(attachment.Role),
		attachment.Priority,
		boolToInt(attachment.Enabled),
		string(attachment.DataSchemaID),
		[]byte(attachment.Data),
		formatTime(attachment.ModifiedAt),
		string(attachment.RootID),
		string(attachment.SourceID),
	)
	if err != nil {
		return sqliteError(fmt.Errorf("update root source attachment: %w", err))
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect root source attachment update: %w", err)
	}
	if changed == 0 {
		return fmt.Errorf("%w: root/source attachment %q/%q", spec.ErrNotFound, attachment.RootID, attachment.SourceID)
	}
	return nil
}

func (s *MetadataStore) DeleteRootSourceAttachment(
	ctx context.Context,
	rootID spec.RootID,
	sourceID spec.SourceID,
) error {
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM root_source_attachments WHERE root_id = ? AND source_id = ?`,
		string(rootID),
		string(sourceID),
	)
	if err != nil {
		return sqliteError(fmt.Errorf("delete root source attachment: %w", err))
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect root source attachment deletion: %w", err)
	}
	if changed == 0 {
		return fmt.Errorf("%w: root/source attachment %q/%q", spec.ErrNotFound, rootID, sourceID)
	}
	return nil
}

func scanRootSourceAttachment(scanner sqlScanner) (spec.RootSourceAttachment, error) {
	var (
		rootID       string
		sourceID     string
		role         string
		priority     int
		enabled      int
		dataSchemaID string
		data         []byte
		createdAt    string
		modifiedAt   string
	)
	if err := scanner.Scan(
		&rootID,
		&sourceID,
		&role,
		&priority,
		&enabled,
		&dataSchemaID,
		&data,
		&createdAt,
		&modifiedAt,
	); err != nil {
		return spec.RootSourceAttachment{}, err
	}
	created, err := parseRequiredTime("attachment.createdAt", createdAt)
	if err != nil {
		return spec.RootSourceAttachment{}, err
	}
	modified, err := parseRequiredTime("attachment.modifiedAt", modifiedAt)
	if err != nil {
		return spec.RootSourceAttachment{}, err
	}
	attachment := spec.RootSourceAttachment{
		RootID:       spec.RootID(rootID),
		SourceID:     spec.SourceID(sourceID),
		Role:         spec.AttachmentRole(role),
		Priority:     priority,
		Enabled:      enabled != 0,
		DataSchemaID: spec.SchemaID(dataSchemaID),
		Data:         append([]byte(nil), data...),
		CreatedAt:    created,
		ModifiedAt:   modified,
	}
	if err := spec.ValidateRootSourceAttachment(attachment); err != nil {
		return spec.RootSourceAttachment{}, fmt.Errorf(
			"invalid persisted root source attachment %q/%q: %w",
			rootID,
			sourceID,
			err,
		)
	}
	return attachment, nil
}
