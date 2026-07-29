package collection

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
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
	ids        uuidutil.Generator
	clock      clockutil.Clock
}

func NewService(
	repository Repository,
	sources sourceReader,
	ids uuidutil.Generator,
	timeClock clockutil.Clock,
) (*Service, error) {
	if repository == nil || sources == nil || ids == nil || timeClock == nil {
		return nil, fmt.Errorf(
			"%w: collection service dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Service{
		repository: repository,
		sources:    sources,
		ids:        ids,
		clock:      timeClock,
	}, nil
}

func (s *Service) Create(
	ctx context.Context,
	rootID basespec.RootID,
	draft Draft,
	attachmentDrafts []AttachmentDraft,
) (Collection, []Attachment, error) {
	if err := basespec.ValidateRootID(rootID); err != nil {
		return Collection{}, nil, err
	}
	if err := basespec.ValidateCollectionKind(draft.Kind); err != nil {
		return Collection{}, nil, err
	}
	data, err := canonicalData(draft.Data)
	if err != nil {
		return Collection{}, nil, err
	}
	id, err := s.ids.NewID(ctx)
	if err != nil {
		return Collection{}, nil, err
	}
	now := clockutil.NowUTC(s.clock)
	value := Collection{
		ID:          basespec.CollectionID(id),
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
		return Collection{}, nil, err
	}

	seenSources := make(
		map[basespec.SourceID]struct{},
		len(attachmentDrafts),
	)
	attachments := make([]Attachment, 0, len(attachmentDrafts))
	for _, draft := range attachmentDrafts {
		if _, duplicate := seenSources[draft.SourceID]; duplicate {
			return Collection{}, nil, fmt.Errorf(
				"%w: duplicate collection attachment for source %q",
				basespec.ErrInvalid,
				draft.SourceID,
			)
		}
		seenSources[draft.SourceID] = struct{}{}
		sourceValue, err := s.sources.Get(ctx, rootID, draft.SourceID)
		if err != nil {
			return Collection{}, nil, err
		}
		if draft.Enabled && !sourceValue.Enabled {
			return Collection{}, nil, fmt.Errorf(
				"%w: enabled attachment cannot use disabled source %q",
				basespec.ErrInvalid,
				draft.SourceID,
			)
		}
		attachmentData, err := canonicalData(draft.Data)
		if err != nil {
			return Collection{}, nil, err
		}
		attachment := Attachment{
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
			return Collection{}, nil, err
		}
		attachments = append(attachments, attachment)
	}
	if err := s.repository.Create(ctx, value, attachments); err != nil {
		return Collection{}, nil, err
	}
	return value.Clone(), cloneAttachments(attachments), nil
}

func (s *Service) Get(
	ctx context.Context,
	ref CollectionRef,
) (Collection, error) {
	if err := ref.Validate(); err != nil {
		return Collection{}, err
	}
	return s.repository.Get(ctx, ref)
}

// GetRetired is a privileged lifecycle read. It is intentionally not exposed
// by public consumer APIs because a retired Collection cannot be used as an
// active aggregate.
func (s *Service) GetRetired(
	ctx context.Context,
	ref CollectionRef,
) (Collection, error) {
	if err := ref.Validate(); err != nil {
		return Collection{}, err
	}
	return s.repository.GetRetired(ctx, ref)
}

func (s *Service) ListByRoot(
	ctx context.Context,
	rootID basespec.RootID,
) ([]Collection, error) {
	if err := basespec.ValidateRootID(rootID); err != nil {
		return nil, err
	}
	return s.repository.ListByRoot(ctx, rootID)
}

func (s *Service) Update(
	ctx context.Context,
	ref CollectionRef,
	update Update,
) (Collection, error) {
	if update.ExpectedRevision == 0 {
		return Collection{}, fmt.Errorf(
			"%w: expected collection revision is required",
			basespec.ErrInvalid,
		)
	}
	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return Collection{}, err
	}
	if current.Revision != update.ExpectedRevision {
		return Collection{}, fmt.Errorf(
			"%w: collection changed since it was read",
			basespec.ErrConflict,
		)
	}
	data, err := canonicalData(update.Data)
	if err != nil {
		return Collection{}, err
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
		return Collection{}, err
	}
	if err := s.repository.Update(ctx, next, update.ExpectedRevision); err != nil {
		return Collection{}, err
	}
	return next.Clone(), nil
}

func (s *Service) Retire(
	ctx context.Context,
	ref CollectionRef,
	expectedRevision uint64,
) (Collection, error) {
	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return Collection{}, err
	}
	if expectedRevision == 0 || current.Revision != expectedRevision {
		return Collection{}, fmt.Errorf(
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
		return Collection{}, err
	}
	if err := s.repository.Retire(ctx, next, expectedRevision); err != nil {
		return Collection{}, err
	}
	return next.Clone(), nil
}

func (s *Service) Purge(
	ctx context.Context,
	ref CollectionRef,
	expectedRevision uint64,
) error {
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
	ref CollectionRef,
) ([]Attachment, error) {
	if _, err := s.repository.Get(ctx, ref); err != nil {
		return nil, err
	}
	return s.repository.ListAttachments(ctx, ref)
}

func (s *Service) GetAttachment(
	ctx context.Context,
	ref CollectionRef,
	sourceID basespec.SourceID,
) (Attachment, error) {
	if _, err := s.repository.Get(ctx, ref); err != nil {
		return Attachment{}, err
	}
	return s.repository.GetAttachment(ctx, ref, sourceID)
}

func (s *Service) Attach(
	ctx context.Context,
	ref CollectionRef,
	expectedCollectionRevision uint64,
	draft AttachmentDraft,
) (Collection, Attachment, error) {
	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return Collection{}, Attachment{}, err
	}
	if expectedCollectionRevision == 0 ||
		current.Revision != expectedCollectionRevision {
		return Collection{}, Attachment{}, basespec.ErrConflict
	}
	sourceValue, err := s.sources.Get(ctx, ref.RootID, draft.SourceID)
	if err != nil {
		return Collection{}, Attachment{}, err
	}
	if draft.Enabled && !sourceValue.Enabled {
		return Collection{}, Attachment{}, fmt.Errorf(
			"%w: enabled attachment cannot use a disabled source",
			basespec.ErrInvalid,
		)
	}
	data, err := canonicalData(draft.Data)
	if err != nil {
		return Collection{}, Attachment{}, err
	}
	now := clockutil.Next(s.clock, current.ModifiedAt)
	value := Attachment{
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
		return Collection{}, Attachment{}, err
	}
	updated, err := s.repository.Attach(
		ctx,
		value,
		expectedCollectionRevision,
	)
	if err != nil {
		return Collection{}, Attachment{}, err
	}
	return updated.Clone(), value.Clone(), nil
}

func (s *Service) UpdateAttachment(
	ctx context.Context,
	ref CollectionRef,
	sourceID basespec.SourceID,
	update AttachmentUpdate,
) (Collection, Attachment, error) {
	currentCollection, err := s.repository.Get(ctx, ref)
	if err != nil {
		return Collection{}, Attachment{}, err
	}
	current, err := s.repository.GetAttachment(ctx, ref, sourceID)
	if err != nil {
		return Collection{}, Attachment{}, err
	}
	if currentCollection.Revision != update.ExpectedCollectionRevision ||
		current.Revision != update.ExpectedAttachmentRevision {
		return Collection{}, Attachment{}, basespec.ErrConflict
	}
	sourceValue, err := s.sources.Get(ctx, ref.RootID, sourceID)
	if err != nil {
		return Collection{}, Attachment{}, err
	}
	if update.Enabled && !sourceValue.Enabled {
		return Collection{}, Attachment{}, fmt.Errorf(
			"%w: enabled attachment cannot use disabled source",
			basespec.ErrInvalid,
		)
	}
	data, err := canonicalData(update.Data)
	if err != nil {
		return Collection{}, Attachment{}, err
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
		return Collection{}, Attachment{}, err
	}
	updated, err := s.repository.UpdateAttachment(
		ctx,
		next,
		update.ExpectedCollectionRevision,
		update.ExpectedAttachmentRevision,
	)
	if err != nil {
		return Collection{}, Attachment{}, err
	}
	return updated.Clone(), next.Clone(), nil
}

func (s *Service) Detach(
	ctx context.Context,
	ref CollectionRef,
	sourceID basespec.SourceID,
	expectedCollectionRevision uint64,
	expectedAttachmentRevision uint64,
) (Collection, error) {
	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return Collection{}, err
	}
	if current.Revision != expectedCollectionRevision {
		return Collection{}, basespec.ErrConflict
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
	ref CollectionRef,
	replacement AttachmentReplacement,
) (Collection, Attachment, error) {
	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return Collection{}, Attachment{}, err
	}
	if replacement.ExpectedCollectionRevision == 0 ||
		current.Revision != replacement.ExpectedCollectionRevision {
		return Collection{}, Attachment{}, basespec.ErrConflict
	}
	sourceValue, err := s.sources.Get(
		ctx,
		ref.RootID,
		replacement.Replacement.SourceID,
	)
	if err != nil {
		return Collection{}, Attachment{}, err
	}
	if replacement.Replacement.Enabled && !sourceValue.Enabled {
		return Collection{}, Attachment{}, fmt.Errorf(
			"%w: enabled replacement cannot use disabled source",
			basespec.ErrInvalid,
		)
	}
	data, err := canonicalData(replacement.Replacement.Data)
	if err != nil {
		return Collection{}, Attachment{}, err
	}
	now := clockutil.Next(s.clock, current.ModifiedAt)
	value := Attachment{
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
		return Collection{}, Attachment{}, err
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
		return Collection{}, Attachment{}, err
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

func cloneAttachments(values []Attachment) []Attachment {
	output := make([]Attachment, len(values))
	for index, value := range values {
		output[index] = value.Clone()
	}
	return output
}
