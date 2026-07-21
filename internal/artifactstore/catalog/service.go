package catalog

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type Service struct {
	repository Repository
	ids        artifactstore.IDGenerator
	clock      artifactstore.Clock
}

func NewService(
	repository Repository,
	ids artifactstore.IDGenerator,
	clock artifactstore.Clock,
) (*Service, error) {
	if repository == nil || ids == nil || clock == nil {
		return nil, fmt.Errorf(
			"%w: catalog service dependencies are incomplete",
			artifactstore.ErrInvalid,
		)
	}
	return &Service{
		repository: repository,
		ids:        ids,
		clock:      clock,
	}, nil
}

func (s *Service) CreateRoot(
	ctx context.Context,
	draft RootDraft,
	attachmentDrafts []AttachmentDraft,
) (Root, []Attachment, error) {
	if err := artifactstore.ValidateRootKind(draft.Kind); err != nil {
		return Root{}, nil, err
	}
	data, err := jsoncanon.CanonicalizeObject(
		draft.Data,
		artifactstore.MaxLocalDataBytes,
	)
	if err != nil {
		return Root{}, nil, err
	}
	id, err := s.ids.NewID(ctx)
	if err != nil {
		return Root{}, nil, err
	}
	now := s.clock.Now().UTC()

	root := Root{
		ID:          artifactstore.RootID(id),
		Kind:        draft.Kind,
		DisplayName: draft.DisplayName,
		Description: draft.Description,
		Enabled:     draft.Enabled,
		Data:        json.RawMessage(data),
		Revision:    1,
		CreatedAt:   now,
		ModifiedAt:  now,
	}
	if err := root.Validate(); err != nil {
		return Root{}, nil, err
	}

	attachments := make([]Attachment, 0, len(attachmentDrafts))
	seenSources := make(map[artifactstore.SourceID]struct{}, len(attachmentDrafts))
	for _, attachmentDraft := range attachmentDrafts {
		if _, duplicate := seenSources[attachmentDraft.SourceID]; duplicate {
			return Root{}, nil, fmt.Errorf(
				"%w: duplicate root attachment for source %q",
				artifactstore.ErrInvalid,
				attachmentDraft.SourceID,
			)
		}
		seenSources[attachmentDraft.SourceID] = struct{}{}

		attachmentData, err := jsoncanon.CanonicalizeObject(
			attachmentDraft.Data,
			artifactstore.MaxLocalDataBytes,
		)
		if err != nil {
			return Root{}, nil, err
		}
		attachment := Attachment{
			RootID:     root.ID,
			SourceID:   attachmentDraft.SourceID,
			Role:       attachmentDraft.Role,
			Priority:   attachmentDraft.Priority,
			Enabled:    attachmentDraft.Enabled,
			Data:       json.RawMessage(attachmentData),
			Revision:   1,
			CreatedAt:  now,
			ModifiedAt: now,
		}
		if err := attachment.Validate(); err != nil {
			return Root{}, nil, err
		}
		attachments = append(attachments, attachment)
	}

	if err := s.repository.CreateRoot(ctx, root, attachments); err != nil {
		return Root{}, nil, err
	}
	return root, attachments, nil
}

func (s *Service) GetRoot(
	ctx context.Context,
	id artifactstore.RootID,
) (Root, error) {
	return s.repository.GetRoot(ctx, id, false)
}

func (s *Service) GetRootIncludingDeleted(
	ctx context.Context,
	id artifactstore.RootID,
) (Root, error) {
	return s.repository.GetRoot(ctx, id, true)
}

func (s *Service) ListRoots(
	ctx context.Context,
	includeDeleted bool,
) ([]Root, error) {
	return s.repository.ListRoots(ctx, includeDeleted)
}

func (s *Service) UpdateRoot(
	ctx context.Context,
	id artifactstore.RootID,
	update RootUpdate,
) (Root, error) {
	if update.ExpectedRevision == 0 {
		return Root{}, fmt.Errorf(
			"%w: expected root revision is required",
			artifactstore.ErrInvalid,
		)
	}
	current, err := s.repository.GetRoot(ctx, id, false)
	if err != nil {
		return Root{}, err
	}
	if current.Revision != update.ExpectedRevision {
		return Root{}, fmt.Errorf(
			"%w: root %q changed since it was read",
			artifactstore.ErrConflict,
			id,
		)
	}
	data, err := jsoncanon.CanonicalizeObject(
		update.Data,
		artifactstore.MaxLocalDataBytes,
	)
	if err != nil {
		return Root{}, err
	}

	next := current
	next.DisplayName = update.DisplayName
	next.Description = update.Description
	next.Enabled = update.Enabled
	next.Data = json.RawMessage(data)

	unchanged := current.DisplayName == next.DisplayName &&
		current.Description == next.Description &&
		current.Enabled == next.Enabled &&
		jsoncanon.Equal(current.Data, next.Data)
	if unchanged {
		return current, nil
	}

	next.Revision++
	next.ModifiedAt = s.clock.Now().UTC()
	if !next.ModifiedAt.After(current.ModifiedAt) {
		next.ModifiedAt = current.ModifiedAt.Add(1)
	}
	if err := next.Validate(); err != nil {
		return Root{}, err
	}
	if err := s.repository.UpdateRoot(ctx, next, update.ExpectedRevision); err != nil {
		return Root{}, err
	}
	return next, nil
}

func (s *Service) DeleteRoot(
	ctx context.Context,
	id artifactstore.RootID,
	expectedRevision uint64,
) (Root, error) {
	current, err := s.repository.GetRoot(ctx, id, false)
	if err != nil {
		return Root{}, err
	}
	if current.Revision != expectedRevision {
		return Root{}, fmt.Errorf(
			"%w: root %q changed since it was read",
			artifactstore.ErrConflict,
			id,
		)
	}
	now := s.clock.Now().UTC()
	next := current
	next.Enabled = false
	next.DeletedAt = &now
	next.ModifiedAt = now
	next.Revision++
	if err := next.Validate(); err != nil {
		return Root{}, err
	}
	if err := s.repository.UpdateRoot(ctx, next, expectedRevision); err != nil {
		return Root{}, err
	}
	return next, nil
}

func (s *Service) Attach(
	ctx context.Context,
	rootID artifactstore.RootID,
	expectedRootRevision uint64,
	draft AttachmentDraft,
) (Root, Attachment, error) {
	data, err := jsoncanon.CanonicalizeObject(
		draft.Data,
		artifactstore.MaxLocalDataBytes,
	)
	if err != nil {
		return Root{}, Attachment{}, err
	}
	now := s.clock.Now().UTC()
	value := Attachment{
		RootID:     rootID,
		SourceID:   draft.SourceID,
		Role:       draft.Role,
		Priority:   draft.Priority,
		Enabled:    draft.Enabled,
		Data:       json.RawMessage(data),
		Revision:   1,
		CreatedAt:  now,
		ModifiedAt: now,
	}
	if err := value.Validate(); err != nil {
		return Root{}, Attachment{}, err
	}
	root, err := s.repository.Attach(ctx, value, expectedRootRevision)
	if err != nil {
		return Root{}, Attachment{}, err
	}
	return root, value, nil
}

func (s *Service) GetAttachment(
	ctx context.Context,
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
) (Attachment, error) {
	return s.repository.GetAttachment(ctx, rootID, sourceID)
}

func (s *Service) ListAttachments(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]Attachment, error) {
	return s.repository.ListAttachments(ctx, rootID)
}

func (s *Service) UpdateAttachment(
	ctx context.Context,
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
	update AttachmentUpdate,
) (Root, Attachment, error) {
	current, err := s.repository.GetAttachment(ctx, rootID, sourceID)
	if err != nil {
		return Root{}, Attachment{}, err
	}
	if current.Revision != update.ExpectedAttachmentRevision {
		return Root{}, Attachment{}, fmt.Errorf(
			"%w: attachment changed since it was read",
			artifactstore.ErrConflict,
		)
	}
	data, err := jsoncanon.CanonicalizeObject(
		update.Data,
		artifactstore.MaxLocalDataBytes,
	)
	if err != nil {
		return Root{}, Attachment{}, err
	}
	next := current
	next.Role = update.Role
	next.Priority = update.Priority
	next.Enabled = update.Enabled
	next.Data = json.RawMessage(data)

	unchanged := current.Role == next.Role &&
		current.Priority == next.Priority &&
		current.Enabled == next.Enabled &&
		jsoncanon.Equal(current.Data, next.Data)
	if unchanged {
		root, err := s.repository.GetRoot(ctx, rootID, false)
		return root, current, err
	}

	next.Revision++
	next.ModifiedAt = s.clock.Now().UTC()
	if !next.ModifiedAt.After(current.ModifiedAt) {
		next.ModifiedAt = current.ModifiedAt.Add(1)
	}
	if err := next.Validate(); err != nil {
		return Root{}, Attachment{}, err
	}
	root, err := s.repository.UpdateAttachment(
		ctx,
		next,
		update.ExpectedRootRevision,
		update.ExpectedAttachmentRevision,
	)
	if err != nil {
		return Root{}, Attachment{}, err
	}
	return root, next, nil
}

func (s *Service) Detach(
	ctx context.Context,
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
	expectedRootRevision uint64,
	expectedAttachmentRevision uint64,
) (Root, error) {
	return s.repository.Detach(
		ctx,
		rootID,
		sourceID,
		expectedRootRevision,
		expectedAttachmentRevision,
	)
}

func (s *Service) Current(
	ctx context.Context,
	rootID artifactstore.RootID,
) (Snapshot, error) {
	return s.repository.GetCurrentCatalog(ctx, rootID)
}
