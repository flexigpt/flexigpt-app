package artifactstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

// RootSourceAttachmentDraft contains local fields for a new root/source link.
type RootSourceAttachmentDraft struct {
	RootID       spec.RootID
	SourceID     spec.SourceID
	Role         spec.AttachmentRole
	Priority     int
	Enabled      bool
	DataSchemaID spec.SchemaID
	Data         json.RawMessage
}

// RootSourceAttachmentUpdate replaces mutable fields of an existing link.
type RootSourceAttachmentUpdate struct {
	ExpectedModifiedAt time.Time
	Role               spec.AttachmentRole
	Priority           int
	Enabled            bool
	DataSchemaID       spec.SchemaID
	Data               json.RawMessage
}

// AttachSource creates an app-local root/source attachment after root-hook
// validation. It does not access the attached source content.
func (s *Store) AttachSource(ctx context.Context, draft RootSourceAttachmentDraft) (spec.RootSourceAttachment, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.RootSourceAttachment{}, err
	}
	defer finish()
	root, err := s.repository.GetRoot(ctx, draft.RootID, false)
	if err != nil {
		return spec.RootSourceAttachment{}, err
	}
	now := s.nowUTC()
	attachment := spec.RootSourceAttachment{
		RootID:       draft.RootID,
		SourceID:     draft.SourceID,
		Role:         draft.Role,
		Priority:     draft.Priority,
		Enabled:      draft.Enabled,
		DataSchemaID: draft.DataSchemaID,
		Data:         normalizedJSONObject(draft.Data),
		CreatedAt:    now,
		ModifiedAt:   now,
	}
	attachments, err := s.repository.ListRootSourceAttachments(ctx, draft.RootID)
	if err != nil {
		return spec.RootSourceAttachment{}, err
	}
	attachments = append(attachments, attachment)
	if err := s.validateAttachmentSet(ctx, root, attachments); err != nil {
		return spec.RootSourceAttachment{}, err
	}
	if err := s.repository.CreateRootSourceAttachment(
		ctx,
		attachment,
		root.MountRevision,
	); err != nil {
		return spec.RootSourceAttachment{}, err
	}
	return attachment, nil
}

// GetRootSourceAttachment returns one app-local attachment by its natural key.
func (s *Store) GetRootSourceAttachment(
	ctx context.Context,
	rootID spec.RootID,
	sourceID spec.SourceID,
) (spec.RootSourceAttachment, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.RootSourceAttachment{}, err
	}
	defer finish()
	return s.repository.GetRootSourceAttachment(ctx, rootID, sourceID)
}

// ListRootSources lists source attachments ordered by priority then source ID.
func (s *Store) ListRootSources(ctx context.Context, rootID spec.RootID) ([]spec.RootSourceAttachment, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	return s.repository.ListRootSourceAttachments(ctx, rootID)
}

// UpdateRootSourceAttachment replaces mutable attachment fields.
func (s *Store) UpdateRootSourceAttachment(
	ctx context.Context,
	rootID spec.RootID,
	sourceID spec.SourceID,
	update RootSourceAttachmentUpdate,
) (spec.RootSourceAttachment, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.RootSourceAttachment{}, err
	}
	defer finish()

	root, err := s.repository.GetRoot(ctx, rootID, false)
	if err != nil {
		return spec.RootSourceAttachment{}, err
	}
	attachment, err := s.repository.GetRootSourceAttachment(ctx, rootID, sourceID)
	if err != nil {
		return spec.RootSourceAttachment{}, err
	}
	if err := requireExpectedModifiedAt(
		"root/source attachment "+string(rootID)+"/"+string(sourceID),
		attachment.ModifiedAt,
		update.ExpectedModifiedAt,
	); err != nil {
		return spec.RootSourceAttachment{}, err
	}
	nextData := normalizedJSONObject(update.Data)
	if attachment.Role == update.Role &&
		attachment.Priority == update.Priority &&
		attachment.Enabled == update.Enabled &&
		attachment.DataSchemaID == update.DataSchemaID &&
		bytes.Equal(attachment.Data, nextData) {
		return attachment, nil
	}
	attachment.Role = update.Role
	attachment.Priority = update.Priority
	attachment.Enabled = update.Enabled
	attachment.DataSchemaID = update.DataSchemaID
	attachment.Data = nextData
	attachment.ModifiedAt = s.nextModifiedAt(attachment.ModifiedAt)
	attachments, err := s.repository.ListRootSourceAttachments(ctx, rootID)
	if err != nil {
		return spec.RootSourceAttachment{}, err
	}
	for index := range attachments {
		if attachments[index].SourceID == sourceID {
			attachments[index] = attachment
			break
		}
	}
	if err := s.validateAttachmentSet(ctx, root, attachments); err != nil {
		return spec.RootSourceAttachment{}, err
	}
	if err := s.repository.UpdateRootSourceAttachment(
		ctx,
		attachment,
		update.ExpectedModifiedAt,
		root.MountRevision,
	); err != nil {
		return spec.RootSourceAttachment{}, err
	}
	return attachment, nil
}

// DetachSource removes only the local root/source relationship. It does not
// delete source registration, source content, catalog history, or records.
func (s *Store) DetachSource(
	ctx context.Context,
	rootID spec.RootID,
	sourceID spec.SourceID,
	expectedModifiedAt time.Time,
) error {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return err
	}
	defer finish()
	root, err := s.repository.GetRoot(ctx, rootID, false)
	if err != nil {
		return err
	}
	attachment, err := s.repository.GetRootSourceAttachment(ctx, rootID, sourceID)
	if err != nil {
		return err
	}
	if err := requireExpectedModifiedAt(
		"root/source attachment "+string(rootID)+"/"+string(sourceID),
		attachment.ModifiedAt,
		expectedModifiedAt,
	); err != nil {
		return err
	}
	attachments, err := s.repository.ListRootSourceAttachments(ctx, rootID)
	if err != nil {
		return err
	}
	remaining := make([]spec.RootSourceAttachment, 0, len(attachments)-1)
	for _, value := range attachments {
		if value.SourceID != sourceID {
			remaining = append(remaining, value)
		}
	}
	if err := s.validateAttachmentSet(ctx, root, remaining); err != nil {
		return err
	}
	return s.repository.DeleteRootSourceAttachment(
		ctx,
		rootID,
		sourceID,
		expectedModifiedAt,
		root.MountRevision,
	)
}

func (s *Store) validateAttachment(
	ctx context.Context,
	root spec.ArtifactRoot,
	attachment spec.RootSourceAttachment,
	source spec.ArtifactSource,
) error {
	if err := validate.ValidateRootSourceAttachment(attachment); err != nil {
		return fmt.Errorf("%w: source attachment: %w", spec.ErrInvalidRequest, err)
	}
	if hook, ok := s.rootHookFor(root.Kind); ok {
		if err := errorDiagnostics(
			"root attachment "+string(root.Kind),
			hook.ValidateSourceAttachment(ctx, root, attachment),
		); err != nil {
			return err
		}
		if sourceHook, ok := hook.(spec.RootAttachmentSourceHook); ok {
			if err := errorDiagnostics(
				"root attachment source "+string(root.Kind),
				sourceHook.ValidateSourceAttachmentSource(ctx, root, attachment, source),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) validateAttachmentSet(
	ctx context.Context,
	root spec.ArtifactRoot,
	attachments []spec.RootSourceAttachment,
) error {
	for _, attachment := range attachments {
		source, err := s.repository.GetSource(ctx, attachment.SourceID)
		if err != nil {
			return err
		}
		if err := s.validateAttachment(ctx, root, attachment, source); err != nil {
			return err
		}
	}

	hook, ok := s.rootHookFor(root.Kind)
	if !ok {
		return nil
	}
	setHook, ok := hook.(spec.RootAttachmentSetHook)
	if !ok {
		return nil
	}
	if err := errorDiagnostics(
		"root attachment set "+string(root.Kind),
		setHook.ValidateSourceAttachments(ctx, root, append([]spec.RootSourceAttachment(nil), attachments...)),
	); err != nil {
		return err
	}
	return nil
}
