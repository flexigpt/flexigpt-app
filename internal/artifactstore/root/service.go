package root

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

type Service struct {
	repository Repository
	clock      clockutil.Clock
	policy     protection.RootPolicy
}

func NewService(
	repository Repository,
	timeClock clockutil.Clock,
	policy protection.RootPolicy,
) (*Service, error) {
	if repository == nil || timeClock == nil {
		return nil, fmt.Errorf(
			"%w: root service dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Service{
		repository: repository,
		clock:      timeClock,
		policy:     policy,
	}, nil
}

func (s *Service) Create(
	ctx context.Context,
	draft RootDraft,
) (Root, error) {
	if err := protection.RequireMutableRoot(ctx, s.policy, draft.ID); err != nil {
		return Root{}, err
	}
	return s.create(ctx, draft)
}

// EnsureSystem is reserved for an application-owned protected-topology
// installer. Artifact Store does not assign any feature meaning to the Root.
func (s *Service) EnsureSystem(
	ctx context.Context,
	draft RootDraft,
) (Root, error) {
	if err := basespec.ValidateRootID(draft.ID); err != nil {
		return Root{}, err
	}
	if s.policy == nil || !s.policy.IsProtectedRoot(draft.ID) {
		return Root{}, fmt.Errorf(
			"%w: Root %q is not declared as protected application topology",
			basespec.ErrProtected,
			draft.ID,
		)
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return Root{}, err
	}
	return s.create(ctx, draft)
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
	if err := protection.RequireMutableRoot(ctx, s.policy, id); err != nil {
		return Root{}, err
	}
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
	if err := basespec.ValidateRootID(id); err != nil {
		return Root{}, err
	}
	if err := protection.RequireRootDeletion(ctx, s.policy, id); err != nil {
		return Root{}, err
	}
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
	if err := basespec.ValidateRootID(id); err != nil {
		return err
	}
	if err := protection.RequireRootDeletion(ctx, s.policy, id); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected root revision is required",
			basespec.ErrInvalid,
		)
	}
	return s.repository.Purge(ctx, id, expectedRevision)
}

func (s *Service) create(
	ctx context.Context,
	draft RootDraft,
) (Root, error) {
	if err := basespec.ValidateRootID(draft.ID); err != nil {
		return Root{}, err
	}
	now := clockutil.NowUTC(s.clock)
	value := Root{
		ID:          draft.ID,
		DisplayName: draft.DisplayName,
		Description: draft.Description,
		Revision:    1,
		CreatedAt:   now,
		ModifiedAt:  now,
	}
	if err := value.Validate(); err != nil {
		return Root{}, err
	}
	createErr := s.repository.Create(ctx, value)
	if createErr == nil {
		return value, nil
	}
	if !errors.Is(createErr, basespec.ErrConflict) {
		return Root{}, createErr
	}

	existing, err := s.repository.Get(ctx, draft.ID)
	if err != nil {
		return Root{}, createErr
	}
	if existing.DisplayName != draft.DisplayName ||
		existing.Description != draft.Description {
		return Root{}, fmt.Errorf(
			"%w: root %q creation intent differs",
			basespec.ErrConflict,
			draft.ID,
		)
	}
	return existing, nil
}
