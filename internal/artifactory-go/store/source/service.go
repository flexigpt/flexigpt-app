package source

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

type rootReader interface {
	Get(ctx context.Context, id rootModel.RootID) (rootModel.Root, error)
}

type Service struct {
	repository Repository
	registry   *Registry
	roots      rootReader
	clock      clockutil.Clock
	policy     root.Policy
	lifecycle  LifecyclePublisher
}

func NewService(
	repository Repository,
	registry *Registry,
	roots rootReader,
	timeClock clockutil.Clock,
	policy root.Policy,
	lifecycle LifecyclePublisher,
) (*Service, error) {
	if repository == nil || registry == nil || roots == nil || timeClock == nil || lifecycle == nil {
		return nil, fmt.Errorf("%w: source service dependencies are incomplete", spec.ErrInvalid)
	}
	return &Service{
		repository: repository,
		registry:   registry,
		roots:      roots,
		clock:      timeClock,
		policy:     policy,
		lifecycle:  lifecycle,
	}, nil
}

func (s *Service) Create(
	ctx context.Context,
	rootID rootModel.RootID,
	draft sourceModel.Draft,
) (sourceModel.Summary, error) {
	value, _, err := s.CreateWithStatus(ctx, rootID, draft)
	return value, err
}

// Ensure reuses a Source with the same Root-local storage key and physical
// Source identity. Discovery and local display state are intentionally not
// compared because the owner may reconcile them after ensure.
func (s *Service) Ensure(
	ctx context.Context,
	rootID rootModel.RootID,
	draft sourceModel.Draft,
) (sourceModel.Summary, bool, error) {
	if ctx == nil {
		return sourceModel.Summary{}, false, fmt.Errorf("%w: Source ensure context is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return sourceModel.Summary{}, false, err
	}
	if err := rootID.Validate(); err != nil {
		return sourceModel.Summary{}, false, err
	}
	if err := draft.ID.Validate(); err != nil {
		return sourceModel.Summary{}, false, err
	}
	if err := draft.StorageKey.Validate(); err != nil {
		return sourceModel.Summary{}, false, err
	}
	if err := draft.Kind.Validate(); err != nil {
		return sourceModel.Summary{}, false, err
	}
	if err := spec.ValidateRequiredText(
		"source display name",
		draft.DisplayName,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return sourceModel.Summary{}, false, err
	}
	if err := root.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return sourceModel.Summary{}, false, err
	}
	rootValue, err := s.roots.Get(ctx, rootID)
	if err != nil {
		return sourceModel.Summary{}, false, err
	}
	normalizedConfig, err := s.registry.NormalizeConfig(ctx, draft.Kind, draft.Config)
	if err != nil {
		return sourceModel.Summary{}, false, err
	}
	if err := draft.Discovery.Normalized().Validate(); err != nil {
		return sourceModel.Summary{}, false, err
	}
	existing, err := s.repository.FindByStorageKey(ctx, rootID, draft.StorageKey)
	if err == nil {
		if existing.RetiredAt != nil {
			return sourceModel.Summary{}, false, fmt.Errorf("%w: Source %q is retired", spec.ErrRetired, existing.ID)
		}
		if !sameEnsuredSource(existing, rootID, rootValue.StorageKey, draft, normalizedConfig) {
			return sourceModel.Summary{}, false, fmt.Errorf(
				"%w: Source storage key %q identifies another physical Source",
				spec.ErrConflict,
				draft.StorageKey,
			)
		}
		return existing.Summary(), false, nil
	}
	if !errors.Is(err, spec.ErrSourceNotFound) && !errors.Is(err, spec.ErrNotFound) {
		return sourceModel.Summary{}, false, err
	}
	return s.CreateWithStatus(ctx, rootID, draft)
}

// CreateWithStatus follows caller-supplied-ID replay and reports whether this
// invocation committed a new Source row for provisioning compensation.
func (s *Service) CreateWithStatus(
	ctx context.Context,
	rootID rootModel.RootID,
	draft sourceModel.Draft,
) (sourceModel.Summary, bool, error) {
	if ctx == nil {
		return sourceModel.Summary{}, false, fmt.Errorf("%w: source creation context is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return sourceModel.Summary{}, false, err
	}
	if err := rootID.Validate(); err != nil {
		return sourceModel.Summary{}, false, err
	}
	rootValue, err := s.roots.Get(ctx, rootID)
	if err != nil {
		return sourceModel.Summary{}, false, err
	}
	if err := root.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return sourceModel.Summary{}, false, err
	}
	if err := draft.ID.Validate(); err != nil {
		return sourceModel.Summary{}, false, err
	}
	if err := draft.StorageKey.Validate(); err != nil {
		return sourceModel.Summary{}, false, err
	}
	if err := draft.Kind.Validate(); err != nil {
		return sourceModel.Summary{}, false, err
	}
	if err := spec.ValidateRequiredText(
		"source display name",
		draft.DisplayName,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return sourceModel.Summary{}, false, err
	}
	config, err := s.registry.NormalizeConfig(ctx, draft.Kind, draft.Config)
	if err != nil {
		return sourceModel.Summary{}, false, err
	}
	discovery := draft.Discovery.Normalized()
	if err := discovery.Validate(); err != nil {
		return sourceModel.Summary{}, false, fmt.Errorf("source discovery: %w", err)
	}
	now := clockutil.NowUTC(s.clock)
	value := sourceModel.Source{
		ID:             draft.ID,
		RootID:         rootID,
		RootStorageKey: rootValue.StorageKey,
		StorageKey:     draft.StorageKey,
		Kind:           draft.Kind,
		DisplayName:    draft.DisplayName,
		Enabled:        draft.Enabled,
		Config:         config,
		Discovery:      discovery,
		Revision:       1,
		CreatedAt:      now,
		ModifiedAt:     now,
	}
	if err := value.ValidateRead(); err != nil {
		return sourceModel.Summary{}, false, err
	}

	existing, lookupErr := s.repository.Get(ctx, rootID, draft.ID)
	switch {
	case lookupErr == nil:
		if sourceCreationIntentMatches(existing, value) {
			return existing.Summary(), false, nil
		}
		return sourceModel.Summary{}, false, fmt.Errorf(
			"%w: source %q creation intent differs",
			spec.ErrConflict,
			draft.ID,
		)
	case !errors.Is(lookupErr, spec.ErrSourceNotFound) && !errors.Is(lookupErr, spec.ErrNotFound):
		return sourceModel.Summary{}, false, lookupErr
	}

	var bootstrapper driver.ManagedSourceBootstrapper
	adapter, found := s.registry.adapter(draft.Kind)
	if !found {
		return sourceModel.Summary{}, false, fmt.Errorf("%w: source adapter %q", spec.ErrSourceUnavailable, draft.Kind)
	}
	if candidate, supported := adapter.(driver.ManagedSourceBootstrapper); supported {
		bootstrapper = candidate
	}
	cleanupBootstrap := func(cause error) error {
		if bootstrapper == nil {
			return cause
		}
		cleanupErr := bootstrapper.DiscardBootstrappedManagedSource(context.WithoutCancel(ctx), value.Clone())
		return errors.Join(cause, cleanupErr)
	}
	if bootstrapper != nil {
		if err := bootstrapper.BootstrapManagedSource(ctx, value.Clone()); err != nil {
			return sourceModel.Summary{}, false, cleanupBootstrap(err)
		}
	}
	createErr := s.repository.Create(ctx, value)
	if createErr == nil {
		return value.Summary(), true, nil
	}
	if !errors.Is(createErr, spec.ErrConflict) {
		return sourceModel.Summary{}, false, createErr
	}
	existing, lookupErr = s.repository.Get(ctx, rootID, draft.ID)
	if lookupErr != nil {
		if errors.Is(lookupErr, spec.ErrSourceNotFound) || errors.Is(lookupErr, spec.ErrNotFound) {
			return sourceModel.Summary{}, false, cleanupBootstrap(createErr)
		}
		return sourceModel.Summary{}, false, createErr
	}
	if !sourceCreationIntentMatches(existing, value) {
		return sourceModel.Summary{}, false, fmt.Errorf(
			"%w: source %q creation intent differs",
			spec.ErrConflict,
			draft.ID,
		)
	}
	return existing.Summary(), false, nil
}

func sourceCreationIntentMatches(existing, requested sourceModel.Source) bool {
	return existing.ID == requested.ID && existing.RootID == requested.RootID &&
		existing.RootStorageKey == requested.RootStorageKey &&
		existing.StorageKey == requested.StorageKey &&
		existing.Kind == requested.Kind &&
		existing.DisplayName == requested.DisplayName &&
		existing.Enabled == requested.Enabled &&
		bytes.Equal(existing.Config, requested.Config) &&
		existing.Discovery.Equal(requested.Discovery)
}

func sameEnsuredSource(
	existing sourceModel.Source,
	rootID rootModel.RootID,
	rootStorageKey spec.StorageKey,
	draft sourceModel.Draft,
	normalizedConfig json.RawMessage,
) bool {
	return existing.RootID == rootID && existing.RootStorageKey == rootStorageKey &&
		existing.StorageKey == draft.StorageKey &&
		existing.Kind == draft.Kind &&
		bytes.Equal(existing.Config, normalizedConfig)
}

func (s *Service) Get(
	ctx context.Context,
	rootID rootModel.RootID,
	id sourceModel.SourceID,
) (sourceModel.Summary, error) {
	if err := rootID.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := id.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	value, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	return value.Summary(), nil
}

func (s *Service) List(ctx context.Context, rootID rootModel.RootID) ([]sourceModel.Summary, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	values, err := s.repository.List(ctx, rootID)
	if err != nil {
		return nil, err
	}
	output := make([]sourceModel.Summary, len(values))
	for index, value := range values {
		output[index] = value.Summary()
	}
	return output, nil
}

func (s *Service) Update(
	ctx context.Context,
	rootID rootModel.RootID,
	id sourceModel.SourceID,
	update sourceModel.Update,
) (sourceModel.Summary, error) {
	if err := root.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := rootID.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := id.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if update.ExpectedRevision == 0 {
		return sourceModel.Summary{}, fmt.Errorf("%w: expected source revision is required", spec.ErrInvalid)
	}
	current, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	if current.Revision != update.ExpectedRevision {
		return sourceModel.Summary{}, fmt.Errorf("%w: source %q changed since it was read", spec.ErrConflict, id)
	}
	config := append(json.RawMessage(nil), current.Config...)
	if update.Config != nil {
		config, err = s.registry.NormalizeConfig(ctx, current.Kind, update.Config)
		if err != nil {
			return sourceModel.Summary{}, err
		}
	}
	discovery := current.Discovery.Clone()
	if update.Discovery != nil {
		discovery = update.Discovery.Normalized()
		if err := discovery.Validate(); err != nil {
			return sourceModel.Summary{}, fmt.Errorf("source discovery: %w", err)
		}
	}
	next := current.Clone()
	next.DisplayName, next.Enabled, next.Config, next.Discovery = update.DisplayName, update.Enabled, config, discovery
	if current.DisplayName == next.DisplayName && current.Enabled == next.Enabled &&
		bytes.Equal(current.Config, next.Config) &&
		current.Discovery.Equal(next.Discovery) {
		return current.Summary(), nil
	}
	if current.Revision == ^uint64(0) {
		return sourceModel.Summary{}, fmt.Errorf("%w: source revision is exhausted", spec.ErrInvalid)
	}
	next.Revision++
	next.ModifiedAt = clockutil.Next(s.clock, current.ModifiedAt)
	if err := next.ValidateRead(); err != nil {
		return sourceModel.Summary{}, err
	}
	if transition, required := sourceLifecycleTransition(current, next); required {
		if err := s.lifecycle.PublishSourceLifecycle(ctx, transition); err != nil {
			return sourceModel.Summary{}, err
		}
		return next.Summary(), nil
	}
	if err := s.repository.Update(ctx, next, update.ExpectedRevision); err != nil {
		return sourceModel.Summary{}, err
	}
	return next.Summary(), nil
}

func sourceLifecycleTransition(current, next sourceModel.Source) (LifecycleTransition, bool) {
	if !current.Enabled {
		return LifecycleTransition{}, false
	}
	if !next.Enabled {
		return LifecycleTransition{
			Source:                 next,
			ExpectedSourceRevision: current.Revision,
			Invalidation:           LifecycleInvalidationDisabled,
		}, true
	}
	if !current.Discovery.Empty() && next.Discovery.Empty() {
		return LifecycleTransition{
			Source:                 next,
			ExpectedSourceRevision: current.Revision,
			Invalidation:           LifecycleInvalidationDiscoveryRemoved,
		}, true
	}
	return LifecycleTransition{}, false
}

func (s *Service) Retire(
	ctx context.Context,
	rootID rootModel.RootID,
	id sourceModel.SourceID,
	expectedRevision uint64,
) (sourceModel.Summary, error) {
	if err := root.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := rootID.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := id.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if expectedRevision == 0 {
		return sourceModel.Summary{}, fmt.Errorf("%w: expected source revision is required", spec.ErrInvalid)
	}
	current, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	if current.Revision != expectedRevision {
		return sourceModel.Summary{}, fmt.Errorf("%w: source %q changed since it was read", spec.ErrConflict, id)
	}
	if current.Revision == ^uint64(0) {
		return sourceModel.Summary{}, fmt.Errorf("%w: source revision is exhausted", spec.ErrInvalid)
	}
	now := clockutil.Next(s.clock, current.ModifiedAt)
	next := current.Clone()
	next.Enabled, next.RetiredAt, next.ModifiedAt, next.Revision = false, &now, now, current.Revision+1
	if err := next.ValidateRead(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := s.lifecycle.PublishSourceLifecycle(
		ctx,
		LifecycleTransition{
			Source:                 next,
			ExpectedSourceRevision: current.Revision,
			Invalidation:           LifecycleInvalidationRetired,
		},
	); err != nil {
		return sourceModel.Summary{}, err
	}
	return next.Summary(), nil
}

func (s *Service) Discard(
	ctx context.Context,
	rootID rootModel.RootID,
	id sourceModel.SourceID,
	expectedRevision uint64,
) error {
	if ctx == nil {
		return fmt.Errorf("%w: source discard context is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := root.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return err
	}
	if err := rootID.Validate(); err != nil {
		return err
	}
	if err := id.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf("%w: expected Source revision is required", spec.ErrInvalid)
	}
	current, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return spec.ErrConflict
	}
	cleanupContext := context.WithoutCancel(ctx)
	if err := s.repository.Discard(cleanupContext, rootID, id, expectedRevision); err != nil {
		return err
	}
	if err := s.discardManagedStorage(cleanupContext, current); err != nil {
		return fmt.Errorf("source metadata was discarded but managed bootstrap cleanup remains pending: %w", err)
	}
	return nil
}

func (s *Service) Purge(
	ctx context.Context,
	rootID rootModel.RootID,
	id sourceModel.SourceID,
	expectedRevision uint64,
) error {
	if err := root.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf("%w: expected source revision is required", spec.ErrInvalid)
	}
	if err := rootID.Validate(); err != nil {
		return err
	}
	if err := id.Validate(); err != nil {
		return err
	}
	return s.repository.Purge(ctx, rootID, id, expectedRevision)
}

func (s *Service) MarkContentChanged(
	ctx context.Context,
	rootID rootModel.RootID,
	id sourceModel.SourceID,
	expectedRevision uint64,
) (sourceModel.Summary, error) {
	if err := root.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return sourceModel.Summary{}, err
	}
	if ctx == nil {
		return sourceModel.Summary{}, fmt.Errorf("%w: source content-change context is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := rootID.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := id.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if expectedRevision == 0 {
		return sourceModel.Summary{}, fmt.Errorf("%w: expected source revision is required", spec.ErrInvalid)
	}
	current, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	if current.Revision != expectedRevision {
		return sourceModel.Summary{}, spec.ErrConflict
	}
	if current.Revision == ^uint64(0) {
		return sourceModel.Summary{}, fmt.Errorf("%w: source revision is exhausted", spec.ErrInvalid)
	}
	next := current.Clone()
	next.Revision++
	next.ModifiedAt = clockutil.Next(s.clock, current.ModifiedAt)
	if err := next.ValidateRead(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := s.repository.Update(ctx, next, expectedRevision); err != nil {
		return sourceModel.Summary{}, err
	}
	return next.Summary(), nil
}

func (s *Service) discardManagedStorage(ctx context.Context, value sourceModel.Source) error {
	adapter, exists := s.registry.adapter(value.Kind)
	if !exists {
		return nil
	}
	bootstrapper, supported := adapter.(driver.ManagedSourceBootstrapper)
	if !supported {
		return nil
	}
	if err := bootstrapper.DiscardBootstrappedManagedSource(ctx, value.Clone()); err != nil {
		return fmt.Errorf("discard managed Source storage: %w", err)
	}
	return nil
}
