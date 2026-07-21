package source

import (
	"context"
	"fmt"

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
	draft Draft,
) (Source, error) {
	if err := artifactstore.ValidateSourceKind(draft.Kind); err != nil {
		return Source{}, err
	}
	if err := artifactstore.ValidateRequiredText(
		"source display name",
		draft.DisplayName,
		artifactstore.MaxDisplayNameBytes,
	); err != nil {
		return Source{}, err
	}
	adapter, exists := s.registry.Adapter(draft.Kind)
	if !exists {
		return Source{}, fmt.Errorf(
			"%w: source adapter %q",
			artifactstore.ErrSourceUnavailable,
			draft.Kind,
		)
	}
	config, err := adapter.NormalizeConfig(ctx, draft.Config)
	if err != nil {
		return Source{}, err
	}
	config, err = jsoncanon.CanonicalizeObject(
		config,
		artifactstore.MaxConfigBytes,
	)
	if err != nil {
		return Source{}, fmt.Errorf("%w: source config: %w", artifactstore.ErrInvalid, err)
	}

	id, err := s.ids.NewID(ctx)
	if err != nil {
		return Source{}, err
	}
	now := s.clock.Now().UTC()
	value := Source{
		ID:          artifactstore.SourceID(id),
		Kind:        draft.Kind,
		DisplayName: draft.DisplayName,
		Enabled:     draft.Enabled,
		Config:      config,
		Revision:    1,
		CreatedAt:   now,
		ModifiedAt:  now,
	}
	if err := value.Validate(); err != nil {
		return Source{}, err
	}
	if err := s.repository.Create(ctx, value); err != nil {
		return Source{}, err
	}
	return value, nil
}

func (s *Service) Get(
	ctx context.Context,
	id artifactstore.SourceID,
) (Source, error) {
	if err := artifactstore.ValidateSourceID(id); err != nil {
		return Source{}, err
	}
	return s.repository.Get(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Source, error) {
	return s.repository.List(ctx)
}

func (s *Service) Update(
	ctx context.Context,
	id artifactstore.SourceID,
	update Update,
) (Source, error) {
	if update.ExpectedRevision == 0 {
		return Source{}, fmt.Errorf(
			"%w: expected source revision is required",
			artifactstore.ErrInvalid,
		)
	}
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return Source{}, err
	}
	if current.Revision != update.ExpectedRevision {
		return Source{}, fmt.Errorf(
			"%w: source %q changed since it was read",
			artifactstore.ErrConflict,
			id,
		)
	}

	adapter, exists := s.registry.Adapter(current.Kind)
	if !exists {
		return Source{}, fmt.Errorf(
			"%w: source adapter %q",
			artifactstore.ErrSourceUnavailable,
			current.Kind,
		)
	}
	config, err := adapter.NormalizeConfig(ctx, update.Config)
	if err != nil {
		return Source{}, err
	}
	config, err = jsoncanon.CanonicalizeObject(
		config,
		artifactstore.MaxConfigBytes,
	)
	if err != nil {
		return Source{}, err
	}

	next := current
	next.DisplayName = update.DisplayName
	next.Enabled = update.Enabled
	next.Config = config

	unchanged := current.DisplayName == next.DisplayName &&
		current.Enabled == next.Enabled &&
		jsoncanon.Equal(current.Config, next.Config)
	if unchanged {
		return current, nil
	}

	next.Revision++
	next.ModifiedAt = s.clock.Now().UTC()
	if !next.ModifiedAt.After(current.ModifiedAt) {
		next.ModifiedAt = current.ModifiedAt.Add(1)
	}
	if err := next.Validate(); err != nil {
		return Source{}, err
	}
	if err := s.repository.Update(ctx, next, update.ExpectedRevision); err != nil {
		return Source{}, err
	}
	return next, nil
}

func (s *Service) Delete(
	ctx context.Context,
	id artifactstore.SourceID,
	expectedRevision uint64,
) error {
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected source revision is required",
			artifactstore.ErrInvalid,
		)
	}
	return s.repository.Delete(ctx, id, expectedRevision)
}

func (s *Service) Registry() *Registry {
	return s.registry
}
