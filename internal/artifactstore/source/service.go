package source

import (
	"context"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type Service struct {
	repository Repository
	registry   *Registry
	ids        artifactstore.IDGenerator
	clock      artifactstore.Clock
}

func NewService(
	repository Repository,
	registry *Registry,
	ids artifactstore.IDGenerator,
	clock artifactstore.Clock,
) (*Service, error) {
	if repository == nil || registry == nil || ids == nil || clock == nil {
		return nil, fmt.Errorf(
			"%w: source service dependencies are incomplete",
			artifactstore.ErrInvalid,
		)
	}
	return &Service{
		repository: repository,
		registry:   registry,
		ids:        ids,
		clock:      clock,
	}, nil
}

func (s *Service) Create(
	ctx context.Context,
	rootID artifactstore.RootID,
	draft Draft,
) (Summary, error) {
	if err := artifactstore.ValidateRootID(rootID); err != nil {
		return Summary{}, err
	}
	if err := artifactstore.ValidateSourceKind(draft.Kind); err != nil {
		return Summary{}, err
	}
	if err := artifactstore.ValidateRequiredText(
		"source display name",
		draft.DisplayName,
		artifactstore.MaxDisplayNameBytes,
	); err != nil {
		return Summary{}, err
	}
	adapter, exists := s.registry.adapter(draft.Kind)
	if !exists {
		return Summary{}, fmt.Errorf(
			"%w: source adapter %q",
			artifactstore.ErrSourceUnavailable,
			draft.Kind,
		)
	}
	config, err := adapter.NormalizeConfig(ctx, draft.Config)
	if err != nil {
		return Summary{}, err
	}
	config, err = jsoncanon.CanonicalizeObject(
		config,
		artifactstore.MaxConfigBytes,
	)
	if err != nil {
		return Summary{}, fmt.Errorf("%w: source config: %w", artifactstore.ErrInvalid, err)
	}

	id, err := s.ids.NewID(ctx)
	if err != nil {
		return Summary{}, err
	}
	now := s.clock.Now().UTC()
	value := Source{
		ID:          artifactstore.SourceID(id),
		RootID:      rootID,
		Kind:        draft.Kind,
		DisplayName: draft.DisplayName,
		Enabled:     draft.Enabled,
		Config:      config,
		Revision:    1,
		CreatedAt:   now,
		ModifiedAt:  now,
	}
	if err := value.Validate(); err != nil {
		return Summary{}, err
	}
	if err := s.repository.Create(ctx, value); err != nil {
		return Summary{}, err
	}
	return value.Summary(), nil
}

func (s *Service) Get(
	ctx context.Context,
	rootID artifactstore.RootID,
	id artifactstore.SourceID,
) (Summary, error) {
	if err := artifactstore.ValidateRootID(rootID); err != nil {
		return Summary{}, err
	}
	if err := artifactstore.ValidateSourceID(id); err != nil {
		return Summary{}, err
	}
	value, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return Summary{}, err
	}
	return value.Summary(), nil
}

func (s *Service) List(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]Summary, error) {
	values, err := s.repository.List(ctx, rootID)
	if err != nil {
		return nil, err
	}
	output := make([]Summary, len(values))
	for index, value := range values {
		output[index] = value.Summary()
	}
	return output, nil
}

func (s *Service) Update(
	ctx context.Context,
	rootID artifactstore.RootID,
	id artifactstore.SourceID,
	update Update,
) (Summary, error) {
	if update.ExpectedRevision == 0 {
		return Summary{}, fmt.Errorf(
			"%w: expected source revision is required",
			artifactstore.ErrInvalid,
		)
	}
	current, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return Summary{}, err
	}
	if current.Revision != update.ExpectedRevision {
		return Summary{}, fmt.Errorf(
			"%w: source %q changed since it was read",
			artifactstore.ErrConflict,
			id,
		)
	}

	adapter, exists := s.registry.adapter(current.Kind)
	if !exists {
		return Summary{}, fmt.Errorf(
			"%w: source adapter %q",
			artifactstore.ErrSourceUnavailable,
			current.Kind,
		)
	}
	config, err := adapter.NormalizeConfig(ctx, update.Config)
	if err != nil {
		return Summary{}, err
	}
	config, err = jsoncanon.CanonicalizeObject(
		config,
		artifactstore.MaxConfigBytes,
	)
	if err != nil {
		return Summary{}, err
	}

	next := current
	next.DisplayName = update.DisplayName
	next.Enabled = update.Enabled
	next.Config = config

	unchanged := current.DisplayName == next.DisplayName &&
		current.Enabled == next.Enabled &&
		jsoncanon.Equal(current.Config, next.Config)
	if unchanged {
		return current.Summary(), nil
	}

	if current.Revision == ^uint64(0) {
		return Summary{}, fmt.Errorf("%w: source revision is exhausted", artifactstore.ErrInvalid)
	}
	next.Revision++
	next.ModifiedAt = s.nextModifiedAt(current.ModifiedAt)
	if err := next.Validate(); err != nil {
		return Summary{}, err
	}
	if err := s.repository.Update(ctx, next, update.ExpectedRevision); err != nil {
		return Summary{}, err
	}
	return next.Summary(), nil
}

func (s *Service) Retire(
	ctx context.Context,
	rootID artifactstore.RootID,
	id artifactstore.SourceID,
	expectedRevision uint64,
) (Summary, error) {
	if expectedRevision == 0 {
		return Summary{}, fmt.Errorf(
			"%w: expected source revision is required",
			artifactstore.ErrInvalid,
		)
	}
	current, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return Summary{}, err
	}
	if current.Revision != expectedRevision {
		return Summary{}, fmt.Errorf(
			"%w: source %q changed since it was read",
			artifactstore.ErrConflict,
			id,
		)
	}
	if current.Revision == ^uint64(0) {
		return Summary{}, fmt.Errorf("%w: source revision is exhausted", artifactstore.ErrInvalid)
	}
	now := s.nextModifiedAt(current.ModifiedAt)
	next := current
	next.Enabled = false
	next.RetiredAt = &now
	next.ModifiedAt = now
	next.Revision++
	if err := next.Validate(); err != nil {
		return Summary{}, err
	}
	if err := s.repository.Retire(ctx, next, expectedRevision); err != nil {
		return Summary{}, err
	}
	return next.Summary(), nil
}

// Discard removes a newly created, unattached Source after a higher-level
// workflow failed before it could publish a Collection attachment. Unlike
// Purge, it is intentionally limited to active Sources with no attachments.
func (s *Service) Discard(
	ctx context.Context,
	rootID artifactstore.RootID,
	id artifactstore.SourceID,
	expectedRevision uint64,
) error {
	if err := artifactstore.ValidateRootID(rootID); err != nil {
		return err
	}
	if err := artifactstore.ValidateSourceID(id); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected source revision is required",
			artifactstore.ErrInvalid,
		)
	}
	return s.repository.Discard(ctx, rootID, id, expectedRevision)
}

func (s *Service) Purge(
	ctx context.Context,
	rootID artifactstore.RootID,
	id artifactstore.SourceID,
	expectedRevision uint64,
) error {
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected source revision is required",
			artifactstore.ErrInvalid,
		)
	}
	if err := artifactstore.ValidateRootID(rootID); err != nil {
		return err
	}
	if err := artifactstore.ValidateSourceID(id); err != nil {
		return err
	}
	return s.repository.Purge(ctx, rootID, id, expectedRevision)
}

// MarkContentChanged advances Source metadata after an application-managed
// Source publication changes snapshot-visible content.
//
// It is intentionally a trusted internal operation. It does not expose Source
// configuration and is used by system composition after a successful managed
// package publication or removal. Advancing the Source revision invalidates
// catalogs that were published against the prior snapshot generation.
func (s *Service) MarkContentChanged(
	ctx context.Context,
	rootID artifactstore.RootID,
	id artifactstore.SourceID,
	expectedRevision uint64,
) (Summary, error) {
	if ctx == nil {
		return Summary{}, fmt.Errorf(
			"%w: source content-change context is nil",
			artifactstore.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return Summary{}, err
	}
	if err := artifactstore.ValidateRootID(rootID); err != nil {
		return Summary{}, err
	}
	if err := artifactstore.ValidateSourceID(id); err != nil {
		return Summary{}, err
	}
	if expectedRevision == 0 {
		return Summary{}, fmt.Errorf(
			"%w: expected source revision is required",
			artifactstore.ErrInvalid,
		)
	}

	current, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return Summary{}, err
	}
	if current.Revision != expectedRevision {
		return Summary{}, artifactstore.ErrConflict
	}
	if current.Revision == ^uint64(0) {
		return Summary{}, fmt.Errorf("%w: source revision is exhausted", artifactstore.ErrInvalid)
	}

	next := current.Clone()
	next.Revision++
	next.ModifiedAt = s.nextModifiedAt(current.ModifiedAt)
	if err := next.Validate(); err != nil {
		return Summary{}, err
	}
	if err := s.repository.Update(ctx, next, expectedRevision); err != nil {
		return Summary{}, err
	}
	return next.Summary(), nil
}

func (s *Service) Kinds() []artifactstore.SourceKind {
	return s.registry.Kinds()
}

func (s *Service) nextModifiedAt(previous time.Time) time.Time {
	next := s.clock.Now().UTC()
	if !next.After(previous) {
		return previous.Add(time.Nanosecond)
	}
	return next
}
