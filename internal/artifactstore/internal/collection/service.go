package collectionimpl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type sourceReader interface {
	Get(
		ctx context.Context,
		rootID basespec.RootID,
		id basespec.SourceID,
	) (source.Summary, error)
}

type Service struct {
	repository Repository
	sources    sourceReader
	clock      clockutil.Clock
	policy     protection.RootPolicy
}

func NewService(
	repository Repository,
	sources sourceReader,
	timeClock clockutil.Clock,
	policy protection.RootPolicy,
) (*Service, error) {
	if repository == nil || sources == nil || timeClock == nil {
		return nil, fmt.Errorf(
			"%w: collection service dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Service{
		repository: repository,
		sources:    sources,
		clock:      timeClock,
		policy:     policy,
	}, nil
}

func (s *Service) Create(
	ctx context.Context,
	rootID basespec.RootID,
	draft collection.Draft,
	attachmentDrafts []collection.AttachmentDraft,
) (collection.Collection, []collection.Attachment, error) {
	if err := basespec.ValidateRootID(rootID); err != nil {
		return collection.Collection{}, nil, err
	}
	if err := protection.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return collection.Collection{}, nil, err
	}
	if err := basespec.ValidateCollectionID(draft.ID); err != nil {
		return collection.Collection{}, nil, err
	}
	if err := basespec.ValidateCollectionKind(draft.Kind); err != nil {
		return collection.Collection{}, nil, err
	}
	data, err := canonicalData(draft.Data)
	if err != nil {
		return collection.Collection{}, nil, err
	}
	now := clockutil.NowUTC(s.clock)
	value := collection.Collection{
		ID:          draft.ID,
		RootID:      rootID,
		Kind:        draft.Kind,
		DisplayName: draft.DisplayName,
		Description: draft.Description,
		Enabled:     draft.Enabled,
		Data:        data,
		Revision:    1,
		CreatedAt:   now,
		ModifiedAt:  now,
	}
	if err := value.Validate(); err != nil {
		return collection.Collection{}, nil, err
	}

	seenSources := make(
		map[basespec.SourceID]struct{},
		len(attachmentDrafts),
	)
	attachments := make([]collection.Attachment, 0, len(attachmentDrafts))
	for _, draft := range attachmentDrafts {
		if _, duplicate := seenSources[draft.SourceID]; duplicate {
			return collection.Collection{}, nil, fmt.Errorf(
				"%w: duplicate collection attachment for source %q",
				basespec.ErrInvalid,
				draft.SourceID,
			)
		}
		seenSources[draft.SourceID] = struct{}{}
		sourceValue, err := s.sources.Get(ctx, rootID, draft.SourceID)
		if err != nil {
			return collection.Collection{}, nil, err
		}
		if draft.Enabled && !sourceValue.Enabled {
			return collection.Collection{}, nil, fmt.Errorf(
				"%w: enabled attachment cannot use disabled source %q",
				basespec.ErrInvalid,
				draft.SourceID,
			)
		}
		attachmentData, err := canonicalData(draft.Data)
		if err != nil {
			return collection.Collection{}, nil, err
		}
		attachment := collection.Attachment{
			RootID:       rootID,
			CollectionID: value.ID,
			SourceID:     draft.SourceID,
			Role:         draft.Role,
			Enabled:      draft.Enabled,
			Data:         attachmentData,
			Revision:     1,
			CreatedAt:    now,
			ModifiedAt:   now,
		}
		if err := attachment.Validate(); err != nil {
			return collection.Collection{}, nil, err
		}
		attachments = append(attachments, attachment)
	}
	createErr := s.repository.Create(ctx, value, attachments)
	if createErr == nil {
		return value.Clone(), cloneAttachments(attachments), nil
	}
	if !errors.Is(createErr, basespec.ErrConflict) {
		return collection.Collection{}, nil, createErr
	}

	existing, err := s.repository.Get(ctx, value.Ref())
	if err != nil {
		// Collection IDs are globally unique in the namespace. A matching
		// ID in another Root remains an ID conflict, not a not-found result.
		return collection.Collection{}, nil, createErr
	}
	if existing.RootID != value.RootID ||
		existing.ID != value.ID ||
		existing.Kind != value.Kind {
		return collection.Collection{}, nil, fmt.Errorf(
			"%w: collection %q creation intent differs",
			basespec.ErrConflict,
			value.ID,
		)
	}

	existingAttachments, err := s.repository.ListAttachments(ctx, value.Ref())
	if err != nil {
		return collection.Collection{}, nil, err
	}

	return existing.Clone(), cloneAttachments(existingAttachments), nil
}

func (s *Service) Get(
	ctx context.Context,
	ref collection.CollectionRef,
) (collection.Collection, error) {
	if err := ref.Validate(); err != nil {
		return collection.Collection{}, err
	}
	return s.repository.Get(ctx, ref)
}

// GetRetired is a privileged lifecycle read. It is intentionally not exposed
// by public consumer APIs because a retired Collection cannot be used as an
// active aggregate.
func (s *Service) GetRetired(
	ctx context.Context,
	ref collection.CollectionRef,
) (collection.Collection, error) {
	if err := ref.Validate(); err != nil {
		return collection.Collection{}, err
	}
	return s.repository.GetRetired(ctx, ref)
}

func (s *Service) ListByRoot(
	ctx context.Context,
	rootID basespec.RootID,
) ([]collection.Collection, error) {
	if err := basespec.ValidateRootID(rootID); err != nil {
		return nil, err
	}
	return s.repository.ListByRoot(ctx, rootID)
}

func (s *Service) Update(
	ctx context.Context,
	ref collection.CollectionRef,
	update collection.Update,
) (collection.Collection, error) {
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return collection.Collection{}, err
	}
	if update.ExpectedRevision == 0 {
		return collection.Collection{}, fmt.Errorf(
			"%w: expected collection revision is required",
			basespec.ErrInvalid,
		)
	}
	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return collection.Collection{}, err
	}
	if current.Revision != update.ExpectedRevision {
		return collection.Collection{}, fmt.Errorf(
			"%w: collection changed since it was read",
			basespec.ErrConflict,
		)
	}
	data, err := canonicalData(update.Data)
	if err != nil {
		return collection.Collection{}, err
	}
	next := current
	next.DisplayName = update.DisplayName
	next.Description = update.Description
	next.Enabled = update.Enabled
	next.Data = data
	if current.DisplayName == next.DisplayName &&
		current.Description == next.Description &&
		current.Enabled == next.Enabled &&
		jsonutil.Equal(current.Data, next.Data) {
		return current, nil
	}
	next.Revision++
	next.ModifiedAt = clockutil.Next(s.clock, current.ModifiedAt)
	if err := next.Validate(); err != nil {
		return collection.Collection{}, err
	}
	if err := s.repository.Update(ctx, next, update.ExpectedRevision); err != nil {
		return collection.Collection{}, err
	}
	return next.Clone(), nil
}

func (s *Service) Retire(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) (collection.Collection, error) {
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return collection.Collection{}, err
	}
	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return collection.Collection{}, err
	}
	if expectedRevision == 0 || current.Revision != expectedRevision {
		return collection.Collection{}, fmt.Errorf(
			"%w: collection changed since it was read",
			basespec.ErrConflict,
		)
	}
	now := clockutil.Next(s.clock, current.ModifiedAt)
	next := current
	next.Enabled = false
	next.RetiredAt = &now
	next.ModifiedAt = now
	next.Revision++
	if err := next.Validate(); err != nil {
		return collection.Collection{}, err
	}
	if err := s.repository.Retire(ctx, next, expectedRevision); err != nil {
		return collection.Collection{}, err
	}
	return next.Clone(), nil
}

func (s *Service) Purge(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) error {
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected collection revision is required",
			basespec.ErrInvalid,
		)
	}
	return s.repository.Purge(ctx, ref, expectedRevision)
}

func (s *Service) ListAttachments(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]collection.Attachment, error) {
	if _, err := s.repository.Get(ctx, ref); err != nil {
		return nil, err
	}
	return s.repository.ListAttachments(ctx, ref)
}

func (s *Service) GetAttachment(
	ctx context.Context,
	ref collection.CollectionRef,
	sourceID basespec.SourceID,
) (collection.Attachment, error) {
	if _, err := s.repository.Get(ctx, ref); err != nil {
		return collection.Attachment{}, err
	}
	return s.repository.GetAttachment(ctx, ref, sourceID)
}

func (s *Service) Attach(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedCollectionRevision uint64,
	draft collection.AttachmentDraft,
) (collection.Collection, collection.Attachment, error) {
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	if expectedCollectionRevision == 0 ||
		current.Revision != expectedCollectionRevision {
		return collection.Collection{}, collection.Attachment{}, basespec.ErrConflict
	}
	sourceValue, err := s.sources.Get(ctx, ref.RootID, draft.SourceID)
	if err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	if draft.Enabled && !sourceValue.Enabled {
		return collection.Collection{}, collection.Attachment{}, fmt.Errorf(
			"%w: enabled attachment cannot use a disabled source",
			basespec.ErrInvalid,
		)
	}
	data, err := canonicalData(draft.Data)
	if err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	now := clockutil.Next(s.clock, current.ModifiedAt)
	value := collection.Attachment{
		RootID:       ref.RootID,
		CollectionID: ref.CollectionID,
		SourceID:     draft.SourceID,
		Role:         draft.Role,
		Enabled:      draft.Enabled,
		Data:         data,
		Revision:     1,
		CreatedAt:    now,
		ModifiedAt:   now,
	}
	if err := value.Validate(); err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	updated, err := s.repository.Attach(
		ctx,
		value,
		expectedCollectionRevision,
	)
	if err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	return updated.Clone(), value.Clone(), nil
}

func (s *Service) UpdateAttachment(
	ctx context.Context,
	ref collection.CollectionRef,
	sourceID basespec.SourceID,
	update collection.AttachmentUpdate,
) (collection.Collection, collection.Attachment, error) {
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	currentCollection, err := s.repository.Get(ctx, ref)
	if err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	current, err := s.repository.GetAttachment(ctx, ref, sourceID)
	if err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	if currentCollection.Revision != update.ExpectedCollectionRevision ||
		current.Revision != update.ExpectedAttachmentRevision {
		return collection.Collection{}, collection.Attachment{}, basespec.ErrConflict
	}
	sourceValue, err := s.sources.Get(ctx, ref.RootID, sourceID)
	if err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	if update.Enabled && !sourceValue.Enabled {
		return collection.Collection{}, collection.Attachment{}, fmt.Errorf(
			"%w: enabled attachment cannot use disabled source",
			basespec.ErrInvalid,
		)
	}
	data, err := canonicalData(update.Data)
	if err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	next := current
	next.Role = update.Role
	next.Enabled = update.Enabled
	next.Data = data
	if current.Role == next.Role &&
		current.Enabled == next.Enabled &&
		jsonutil.Equal(current.Data, next.Data) {
		return currentCollection, current, nil
	}
	next.Revision++
	previous := current.ModifiedAt
	if currentCollection.ModifiedAt.After(previous) {
		previous = currentCollection.ModifiedAt
	}
	next.ModifiedAt = clockutil.Next(s.clock, previous)
	if err := next.Validate(); err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	updated, err := s.repository.UpdateAttachment(
		ctx,
		next,
		update.ExpectedCollectionRevision,
		update.ExpectedAttachmentRevision,
	)
	if err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	return updated.Clone(), next.Clone(), nil
}

func (s *Service) Detach(
	ctx context.Context,
	ref collection.CollectionRef,
	sourceID basespec.SourceID,
	expectedCollectionRevision uint64,
	expectedAttachmentRevision uint64,
) (collection.Collection, error) {
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return collection.Collection{}, err
	}
	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return collection.Collection{}, err
	}
	if current.Revision != expectedCollectionRevision {
		return collection.Collection{}, basespec.ErrConflict
	}
	return s.repository.Detach(
		ctx,
		ref,
		sourceID,
		expectedCollectionRevision,
		expectedAttachmentRevision,
		clockutil.Next(s.clock, current.ModifiedAt),
	)
}

func (s *Service) ReplaceAttachment(
	ctx context.Context,
	ref collection.CollectionRef,
	replacement collection.AttachmentReplacement,
) (collection.Collection, collection.Attachment, error) {
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	if replacement.ExpectedCollectionRevision == 0 ||
		current.Revision != replacement.ExpectedCollectionRevision {
		return collection.Collection{}, collection.Attachment{}, basespec.ErrConflict
	}
	sourceValue, err := s.sources.Get(
		ctx,
		ref.RootID,
		replacement.Replacement.SourceID,
	)
	if err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	if replacement.Replacement.Enabled && !sourceValue.Enabled {
		return collection.Collection{}, collection.Attachment{}, fmt.Errorf(
			"%w: enabled replacement cannot use disabled source",
			basespec.ErrInvalid,
		)
	}
	data, err := canonicalData(replacement.Replacement.Data)
	if err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	now := clockutil.Next(s.clock, current.ModifiedAt)
	value := collection.Attachment{
		RootID:       ref.RootID,
		CollectionID: ref.CollectionID,
		SourceID:     replacement.Replacement.SourceID,
		Role:         replacement.Replacement.Role,
		Enabled:      replacement.Replacement.Enabled,
		Data:         data,
		Revision:     1,
		CreatedAt:    now,
		ModifiedAt:   now,
	}
	if err := value.Validate(); err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	updated, err := s.repository.ReplaceAttachment(
		ctx,
		ref,
		replacement.PreviousSourceID,
		replacement.PreviousAttachmentRevision,
		value,
		replacement.ExpectedCollectionRevision,
	)
	if err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	return updated.Clone(), value.Clone(), nil
}

func canonicalData(raw json.RawMessage) (json.RawMessage, error) {
	value, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(value), nil
}

func cloneAttachments(values []collection.Attachment) []collection.Attachment {
	output := make([]collection.Attachment, len(values))
	for index, value := range values {
		output[index] = value.Clone()
	}
	return output
}
