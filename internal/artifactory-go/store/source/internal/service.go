package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type rootReader interface {
	Get(
		ctx context.Context,
		id rootModel.RootID,
	) (rootModel.Root, error)
}

type Service struct {
	repository Repository
	registry   *Registry
	roots      rootReader
	clock      clockutil.Clock
	policy     rootModel.RootPolicy
}

func NewService(
	repository Repository,
	registry *Registry,
	roots rootReader,
	timeClock clockutil.Clock,
	policy rootModel.RootPolicy,
) (*Service, error) {
	if repository == nil || registry == nil || roots == nil || timeClock == nil {
		return nil, fmt.Errorf(
			"%w: source service dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	return &Service{
		repository: repository,
		registry:   registry,
		roots:      roots,
		clock:      timeClock,
		policy:     policy,
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
		return sourceModel.Summary{}, false, fmt.Errorf(
			"%w: Source ensure context is nil",
			spec.ErrInvalid,
		)
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
	adapter, found := s.registry.adapter(draft.Kind)
	if !found {
		return sourceModel.Summary{}, false, fmt.Errorf(
			"%w: source adapter %q",
			spec.ErrSourceUnavailable,
			draft.Kind,
		)
	}
	normalizedConfig, err := adapter.NormalizeConfig(ctx, draft.Config)
	if err != nil {
		return sourceModel.Summary{}, false, err
	}
	normalizedConfig, err = jsonutil.CanonicalizeObject(
		normalizedConfig,
		spec.MaxConfigBytes,
	)
	if err != nil {
		return sourceModel.Summary{}, false, err
	}
	if err := draft.Discovery.Normalized().Validate(); err != nil {
		return sourceModel.Summary{}, false, err
	}

	existing, err := s.repository.FindByStorageKey(
		ctx,
		rootID,
		draft.StorageKey,
	)
	if err == nil {
		if existing.RetiredAt != nil {
			return sourceModel.Summary{}, false, fmt.Errorf(
				"%w: Source %q is retired",
				spec.ErrRetired,
				existing.ID,
			)
		}
		if !sameEnsuredSource(
			existing,
			rootID,
			rootValue.StorageKey,
			draft,
			normalizedConfig,
		) {
			return sourceModel.Summary{}, false, fmt.Errorf(
				"%w: Source storage key %q identifies another physical Source",
				spec.ErrConflict,
				draft.StorageKey,
			)
		}
		return existing.Summary(), false, nil
	}
	if !errors.Is(err, spec.ErrSourceNotFound) &&
		!errors.Is(err, spec.ErrNotFound) {
		return sourceModel.Summary{}, false, err
	}

	return s.CreateWithStatus(ctx, rootID, draft)
}

// CreateWithStatus follows the normal caller-supplied-ID replay contract and
// additionally reports whether this invocation committed a new Source row.
//
// The status is intentionally not persisted and is not part of the public
// Artifact Store API. Higher-level provisioning workflows use it only to avoid
// discarding a Source that existed before the current request.
func (s *Service) CreateWithStatus(
	ctx context.Context,
	rootID rootModel.RootID,
	draft sourceModel.Draft,
) (sourceModel.Summary, bool, error) {
	if ctx == nil {
		return sourceModel.Summary{}, false, fmt.Errorf(
			"%w: source creation context is nil",
			spec.ErrInvalid,
		)
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
	adapter, exists := s.registry.adapter(draft.Kind)
	if !exists {
		return sourceModel.Summary{}, false, fmt.Errorf(
			"%w: source adapter %q",
			spec.ErrSourceUnavailable,
			draft.Kind,
		)
	}
	config, err := adapter.NormalizeConfig(ctx, draft.Config)
	if err != nil {
		return sourceModel.Summary{}, false, err
	}
	config, err = jsonutil.CanonicalizeObject(
		config,
		spec.MaxConfigBytes,
	)
	if err != nil {
		return sourceModel.Summary{}, false, fmt.Errorf("%w: source config: %w", spec.ErrInvalid, err)
	}

	discovery := draft.Discovery.Normalized()
	if err := discovery.Validate(); err != nil {
		return sourceModel.Summary{}, false, fmt.Errorf(
			"source discovery: %w",
			err,
		)
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
	if err := value.Validate(); err != nil {
		return sourceModel.Summary{}, false, err
	}

	// A caller-supplied Source ID is the create replay identity. Check for a
	// completed prior creation before bootstrapping a managed directory or any
	// other adapter-owned physical state.
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

	case !errors.Is(lookupErr, spec.ErrSourceNotFound) &&
		!errors.Is(lookupErr, spec.ErrNotFound):
		return sourceModel.Summary{}, false, lookupErr
	}

	var bootstrapper ManagedSourceBootstrapper
	if candidate, supported := adapter.(ManagedSourceBootstrapper); supported {
		bootstrapper = candidate
	}
	cleanupBootstrap := func(cause error) error {
		if bootstrapper == nil {
			return cause
		}
		cleanupErr := bootstrapper.DiscardBootstrappedManagedSource(
			context.WithoutCancel(ctx),
			value.Clone(),
		)
		return errors.Join(cause, cleanupErr)
	}

	if bootstrapper != nil {
		if err := bootstrapper.BootstrapManagedSource(
			ctx,
			value.Clone(),
		); err != nil {
			return sourceModel.Summary{}, false, cleanupBootstrap(err)
		}
	}

	createErr := s.repository.Create(ctx, value)
	if createErr == nil {
		return value.Summary(), true, nil
	}
	if !errors.Is(createErr, spec.ErrConflict) {
		// A repository commit error can be ambiguous. Do not remove the
		// bootstrapped directory after attempting metadata publication:
		// the Source row may already be durable and must never point to
		// deleted managed content.
		return sourceModel.Summary{}, false, createErr
	}

	existing, lookupErr = s.repository.Get(ctx, rootID, draft.ID)
	if lookupErr != nil {
		// A global Source ID may already belong to another Root. Do not turn
		// that ID collision into an unrelated Source-not-found response. This
		// is a known non-commit outcome, so an empty managed Source directory
		// created for this failed attempt can be safely compensated.
		if errors.Is(lookupErr, spec.ErrSourceNotFound) ||
			errors.Is(lookupErr, spec.ErrNotFound) {
			return sourceModel.Summary{}, false, cleanupBootstrap(createErr)
		}

		// A non-not-found lookup failure may follow an ambiguous repository
		// result. Preserve the bootstrapped directory rather than risking
		// deletion of content referenced by a durable Source row.
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

func sourceCreationIntentMatches(
	existing sourceModel.Source,
	requested sourceModel.Source,
) bool {
	return existing.ID == requested.ID &&
		existing.RootID == requested.RootID &&
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
	return existing.RootID == rootID &&
		existing.RootStorageKey == rootStorageKey &&
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

func (s *Service) List(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]sourceModel.Summary, error) {
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
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: expected source revision is required",
			spec.ErrInvalid,
		)
	}
	current, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	if current.Revision != update.ExpectedRevision {
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: source %q changed since it was read",
			spec.ErrConflict,
			id,
		)
	}

	adapter, exists := s.registry.adapter(current.Kind)
	if !exists {
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: source adapter %q",
			spec.ErrSourceUnavailable,
			current.Kind,
		)
	}

	config := append(json.RawMessage(nil), current.Config...)
	if update.Config != nil {
		normalized, err := adapter.NormalizeConfig(
			ctx,
			append(json.RawMessage(nil), update.Config...),
		)
		if err != nil {
			return sourceModel.Summary{}, err
		}
		normalized, err = jsonutil.CanonicalizeObject(
			normalized,
			spec.MaxConfigBytes,
		)
		if err != nil {
			return sourceModel.Summary{}, err
		}
		config = normalized
	}

	discovery := current.Discovery.Clone()
	if update.Discovery != nil {
		discovery = update.Discovery.Normalized()
		if err := discovery.Validate(); err != nil {
			return sourceModel.Summary{}, fmt.Errorf(
				"source discovery: %w",
				err,
			)
		}
	}

	next := current
	next.DisplayName = update.DisplayName
	next.Enabled = update.Enabled
	next.Config = config
	next.Discovery = discovery

	unchanged := current.DisplayName == next.DisplayName &&
		current.Enabled == next.Enabled &&
		bytes.Equal(current.Config, next.Config) &&
		current.Discovery.Equal(next.Discovery)
	if unchanged {
		return current.Summary(), nil
	}

	if current.Revision == ^uint64(0) {
		return sourceModel.Summary{}, fmt.Errorf("%w: source revision is exhausted", spec.ErrInvalid)
	}
	next.Revision++
	next.ModifiedAt = clockutil.Next(s.clock, current.ModifiedAt)
	if err := next.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := s.repository.Update(ctx, next, update.ExpectedRevision); err != nil {
		return sourceModel.Summary{}, err
	}
	return next.Summary(), nil
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
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: expected source revision is required",
			spec.ErrInvalid,
		)
	}
	current, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	if current.Revision != expectedRevision {
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: source %q changed since it was read",
			spec.ErrConflict,
			id,
		)
	}
	if current.Revision == ^uint64(0) {
		return sourceModel.Summary{}, fmt.Errorf("%w: source revision is exhausted", spec.ErrInvalid)
	}
	now := clockutil.Next(s.clock, current.ModifiedAt)
	next := current
	next.Enabled = false
	next.RetiredAt = &now
	next.ModifiedAt = now
	next.Revision++
	if err := next.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := s.repository.Retire(ctx, next, expectedRevision); err != nil {
		return sourceModel.Summary{}, err
	}
	return next.Summary(), nil
}

// Discard removes a newly created Source after a higher-level workflow failed
// before it could synchronize source-backed Artifacts. Unlike Purge, it is
// limited to active Sources with no Artifact or refresh-state records.
func (s *Service) Discard(
	ctx context.Context,
	rootID rootModel.RootID,
	id sourceModel.SourceID,
	expectedRevision uint64,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: source discard context is nil",
			spec.ErrInvalid,
		)
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
		return fmt.Errorf(
			"%w: expected source revision is required",
			spec.ErrInvalid,
		)
	}

	current, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return spec.ErrConflict
	}

	cleanupContext := context.WithoutCancel(ctx)
	if err := s.repository.Discard(
		cleanupContext,
		rootID,
		id,
		expectedRevision,
	); err != nil {
		return err
	}
	if err := s.discardManagedStorage(cleanupContext, current); err != nil {
		return fmt.Errorf(
			"source metadata was discarded but managed bootstrap cleanup remains pending: %w",
			err,
		)
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
		return fmt.Errorf(
			"%w: expected source revision is required",
			spec.ErrInvalid,
		)
	}
	if err := rootID.Validate(); err != nil {
		return err
	}
	if err := id.Validate(); err != nil {
		return err
	}
	return s.repository.Purge(ctx, rootID, id, expectedRevision)
}

// MarkContentChanged advances Source metadata after a successful managed
// source-side mutation. The actual generation remains source-owned and is
// read from a confirmed snapshot when needed.
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
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: source content-change context is nil",
			spec.ErrInvalid,
		)
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
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: expected source revision is required",
			spec.ErrInvalid,
		)
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
	if err := next.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := s.repository.Update(ctx, next, expectedRevision); err != nil {
		return sourceModel.Summary{}, err
	}
	return next.Summary(), nil
}

func (s *Service) Kinds() []sourceModel.SourceKind {
	return s.registry.Kinds()
}

func (s *Service) discardManagedStorage(
	ctx context.Context,
	value sourceModel.Source,
) error {
	adapter, exists := s.registry.adapter(value.Kind)
	if !exists {
		return nil
	}
	bootstrapper, supported := adapter.(ManagedSourceBootstrapper)
	if !supported {
		return nil
	}
	if err := bootstrapper.DiscardBootstrappedManagedSource(
		ctx,
		value.Clone(),
	); err != nil {
		return fmt.Errorf("discard managed Source storage: %w", err)
	}
	return nil
}
