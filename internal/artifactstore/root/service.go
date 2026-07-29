package root

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

type Service struct {
	repository Repository
	ids        uuidutil.Generator
	clock      clockutil.Clock
}

func NewService(
	repository Repository,
	ids uuidutil.Generator,
	timeClock clockutil.Clock,
) (*Service, error) {
	if repository == nil || ids == nil || timeClock == nil {
		return nil, fmt.Errorf(
			"%w: root service dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Service{
		repository: repository,
		ids:        ids,
		clock:      timeClock,
	}, nil
}

func (s *Service) Create(
	ctx context.Context,
	draft RootDraft,
) (Root, error) {
	id, err := s.ids.NewID(ctx)
	if err != nil {
		return Root{}, err
	}
	now := clockutil.NowUTC(s.clock)
	value := Root{
		ID:          basespec.RootID(id),
		DisplayName: draft.DisplayName,
		Description: draft.Description,
		Revision:    1,
		CreatedAt:   now,
		ModifiedAt:  now,
	}
	if err := value.Validate(); err != nil {
		return Root{}, err
	}
	if err := s.repository.Create(ctx, value); err != nil {
		return Root{}, err
	}
	return value, nil
}

func (s *Service) Get(
	ctx context.Context,
	id basespec.RootID,
) (Root, error) {
	if err := basespec.ValidateRootID(id); err != nil {
		return Root{}, err
	}
	return s.repository.Get(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Root, error) {
	return s.repository.List(ctx)
}

func (s *Service) Update(
	ctx context.Context,
	id basespec.RootID,
	update RootUpdate,
) (Root, error) {
	if update.ExpectedRevision == 0 {
		return Root{}, fmt.Errorf(
			"%w: expected root revision is required",
			basespec.ErrInvalid,
		)
	}
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return Root{}, err
	}
	if current.Revision != update.ExpectedRevision {
		return Root{}, fmt.Errorf(
			"%w: root %q changed since it was read",
			basespec.ErrConflict,
			id,
		)
	}
	next := current
	next.DisplayName = update.DisplayName
	next.Description = update.Description
	unchanged := current.DisplayName == next.DisplayName &&
		current.Description == next.Description
	if unchanged {
		return current, nil
	}
	next.Revision++
	next.ModifiedAt = clockutil.Next(s.clock, current.ModifiedAt)
	if err := next.Validate(); err != nil {
		return Root{}, err
	}
	if err := s.repository.Update(ctx, next, update.ExpectedRevision); err != nil {
		return Root{}, err
	}
	return next, nil
}

func (s *Service) Retire(
	ctx context.Context,
	id basespec.RootID,
	expectedRevision uint64,
) (Root, error) {
	if expectedRevision == 0 {
		return Root{}, fmt.Errorf(
			"%w: expected root revision is required",
			basespec.ErrInvalid,
		)
	}
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return Root{}, err
	}
	if current.Revision != expectedRevision {
		return Root{}, fmt.Errorf(
			"%w: root %q changed since it was read",
			basespec.ErrConflict,
			id,
		)
	}
	modifiedAt := clockutil.Next(s.clock, current.ModifiedAt)
	next := current
	next.RetiredAt = &modifiedAt
	next.ModifiedAt = modifiedAt
	next.Revision++
	if err := next.Validate(); err != nil {
		return Root{}, err
	}
	if err := s.repository.Retire(ctx, next, expectedRevision); err != nil {
		return Root{}, err
	}
	return next, nil
}

func (s *Service) Purge(
	ctx context.Context,
	id basespec.RootID,
	expectedRevision uint64,
) error {
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected root revision is required",
			basespec.ErrInvalid,
		)
	}
	return s.repository.Purge(ctx, id, expectedRevision)
}
