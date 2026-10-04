package root

import (
	"context"
	"errors"
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

// Service owns Root lifecycle behavior. Persistence remains a narrow provider
// contract; protected and retained policy remains Root policy.
type Service struct {
	repository Repository
	clock      clockutil.Clock
	policy     Policy
}

func NewService(
	repository Repository,
	timeClock clockutil.Clock,
	policy Policy,
) (*Service, error) {
	if repository == nil || timeClock == nil {
		return nil, fmt.Errorf(
			"%w: root service dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	return &Service{
		repository: repository,
		clock:      timeClock,
		policy:     policy,
	}, nil
}

func (s *Service) Create(ctx context.Context, draft rootModel.RootDraft) (rootModel.Root, error) {
	if err := RequireMutableRoot(ctx, s.policy, draft.ID); err != nil {
		return rootModel.Root{}, err
	}
	return s.create(ctx, draft)
}

// EnsureSystem is reserved for application-owned protected topology. Artifact
// Store assigns no feature meaning to the Root.
func (s *Service) EnsureSystem(ctx context.Context, draft rootModel.RootDraft) (rootModel.Root, error) {
	if err := draft.ID.Validate(); err != nil {
		return rootModel.Root{}, err
	}
	if s.policy == nil || !s.policy.IsProtectedRoot(draft.ID) {
		return rootModel.Root{}, fmt.Errorf(
			"%w: Root %q is not declared as protected application topology",
			spec.ErrProtected,
			draft.ID,
		)
	}
	if err := RequireInstallerPrivilege(ctx); err != nil {
		return rootModel.Root{}, err
	}
	return s.create(ctx, draft)
}

func (s *Service) Get(ctx context.Context, id rootModel.RootID) (rootModel.Root, error) {
	if err := id.Validate(); err != nil {
		return rootModel.Root{}, err
	}
	return s.repository.Get(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]rootModel.Root, error) {
	return s.repository.List(ctx)
}

func (s *Service) Update(
	ctx context.Context,
	id rootModel.RootID,
	update rootModel.RootUpdate,
) (rootModel.Root, error) {
	if err := RequireMutableRoot(ctx, s.policy, id); err != nil {
		return rootModel.Root{}, err
	}
	if update.ExpectedRevision == 0 {
		return rootModel.Root{}, fmt.Errorf("%w: expected root revision is required", spec.ErrInvalid)
	}
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return rootModel.Root{}, err
	}
	if current.Revision != update.ExpectedRevision {
		return rootModel.Root{}, fmt.Errorf("%w: root %q changed since it was read", spec.ErrConflict, id)
	}
	next := current
	next.DisplayName = update.DisplayName
	next.Description = update.Description
	if current.DisplayName == next.DisplayName && current.Description == next.Description {
		return current, nil
	}
	next.Revision++
	next.ModifiedAt = clockutil.Next(s.clock, current.ModifiedAt)
	if err := next.Validate(); err != nil {
		return rootModel.Root{}, err
	}
	if err := s.repository.Update(ctx, next, update.ExpectedRevision); err != nil {
		return rootModel.Root{}, err
	}
	return next, nil
}

func (s *Service) Retire(ctx context.Context, id rootModel.RootID, expectedRevision uint64) (rootModel.Root, error) {
	if err := id.Validate(); err != nil {
		return rootModel.Root{}, err
	}
	if err := RequireRootDeletion(ctx, s.policy, id); err != nil {
		return rootModel.Root{}, err
	}
	if expectedRevision == 0 {
		return rootModel.Root{}, fmt.Errorf("%w: expected root revision is required", spec.ErrInvalid)
	}
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return rootModel.Root{}, err
	}
	if current.Revision != expectedRevision {
		return rootModel.Root{}, fmt.Errorf("%w: root %q changed since it was read", spec.ErrConflict, id)
	}
	modifiedAt := clockutil.Next(s.clock, current.ModifiedAt)
	next := current
	next.RetiredAt = &modifiedAt
	next.ModifiedAt = modifiedAt
	next.Revision++
	if err := next.Validate(); err != nil {
		return rootModel.Root{}, err
	}
	if err := s.repository.Retire(ctx, next, expectedRevision); err != nil {
		return rootModel.Root{}, err
	}
	return next, nil
}

func (s *Service) Purge(ctx context.Context, id rootModel.RootID, expectedRevision uint64) error {
	if err := id.Validate(); err != nil {
		return err
	}
	if err := RequireRootDeletion(ctx, s.policy, id); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf("%w: expected root revision is required", spec.ErrInvalid)
	}
	return s.repository.Purge(ctx, id, expectedRevision)
}

func (s *Service) create(ctx context.Context, draft rootModel.RootDraft) (rootModel.Root, error) {
	if err := draft.ID.Validate(); err != nil {
		return rootModel.Root{}, err
	}
	now := clockutil.NowUTC(s.clock)
	value := rootModel.Root{
		ID: draft.ID, StorageKey: draft.StorageKey, DisplayName: draft.DisplayName,
		Description: draft.Description, Revision: 1, CreatedAt: now, ModifiedAt: now,
	}
	if err := value.Validate(); err != nil {
		return rootModel.Root{}, err
	}
	createErr := s.repository.Create(ctx, value)
	if createErr == nil {
		return value, nil
	}
	if !errors.Is(createErr, spec.ErrConflict) {
		return rootModel.Root{}, createErr
	}
	existing, err := s.repository.Get(ctx, draft.ID)
	if err != nil {
		return rootModel.Root{}, createErr
	}
	if existing.DisplayName != draft.DisplayName || existing.StorageKey != draft.StorageKey ||
		existing.Description != draft.Description {
		return rootModel.Root{}, fmt.Errorf("%w: root %q creation intent differs", spec.ErrConflict, draft.ID)
	}
	return existing, nil
}
