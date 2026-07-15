package metadatastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
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
	if err := validate.ValidateRootSourceAttachment(attachment); err != nil {
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

func (s *MetadataStore) UpdateRootSourceAttachment(
	ctx context.Context,
	attachment spec.RootSourceAttachment,
	expectedModifiedAt time.Time,
) error {
	if err := validate.ValidateRootSourceAttachment(attachment); err != nil {
		return fmt.Errorf("validate root source attachment for persistence: %w", err)
	}
	if err := validateExpectedModifiedAt("root/source attachment", expectedModifiedAt); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, updateRootSourceAttachmentSQL,
		string(attachment.Role),
		attachment.Priority,
		boolToInt(attachment.Enabled),
		string(attachment.DataSchemaID),
		string(attachment.Data),
		formatTime(attachment.ModifiedAt),
		string(attachment.RootID),
		string(attachment.SourceID),
		formatTime(expectedModifiedAt),
	)
	if err != nil {
		return sqliteError(fmt.Errorf("update root source attachment: %w", err))
	}
	return optimisticMutationResult(
		result,
		"root/source attachment "+string(attachment.RootID)+"/"+string(attachment.SourceID),
	)
}

func (s *MetadataStore) DeleteRootSourceAttachment(
	ctx context.Context,
	rootID spec.RootID,
	sourceID spec.SourceID,
	expectedModifiedAt time.Time,
) error {
	if err := validateExpectedModifiedAt("root/source attachment", expectedModifiedAt); err != nil {
		return err
	}
	result, err := s.db.ExecContext(
		ctx,
		deleteRootSourceAttachmentSQL,
		string(rootID),
		string(sourceID),
		formatTime(expectedModifiedAt),
	)
	if err != nil {
		return sqliteError(fmt.Errorf("delete root source attachment: %w", err))
	}
	return optimisticMutationResult(
		result,
		"root/source attachment "+string(rootID)+"/"+string(sourceID),
	)
}

func scanRootSourceAttachment(scanner sqlScanner) (spec.RootSourceAttachment, error) {
	row := rootSourceAttachmentRow{}
	if err := scanner.Scan(row.destinations()...); err != nil {
		return spec.RootSourceAttachment{}, err
	}
	created, err := parseRequiredTime("attachment.createdAt", row.CreatedAt)
	if err != nil {
		return spec.RootSourceAttachment{}, err
	}
	modified, err := parseRequiredTime("attachment.modifiedAt", row.ModifiedAt)
	if err != nil {
		return spec.RootSourceAttachment{}, err
	}
	attachment := spec.RootSourceAttachment{
		RootID:       spec.RootID(row.RootID),
		SourceID:     spec.SourceID(row.SourceID),
		Role:         spec.AttachmentRole(row.Role),
		Priority:     row.Priority,
		Enabled:      row.Enabled != 0,
		DataSchemaID: spec.SchemaID(row.DataSchemaID),
		Data:         append([]byte(nil), row.Data...),
		CreatedAt:    created,
		ModifiedAt:   modified,
	}
	if err := validate.ValidateRootSourceAttachment(attachment); err != nil {
		return spec.RootSourceAttachment{}, fmt.Errorf(
			"invalid persisted root source attachment %q/%q: %w",
			row.RootID,
			row.SourceID,
			err,
		)
	}
	return attachment, nil
}
