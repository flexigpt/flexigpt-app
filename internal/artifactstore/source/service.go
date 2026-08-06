package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Service struct {
	repository Repository
	registry   *Registry
	clock      clockutil.Clock
	policy     protection.RootPolicy
}

func NewService(
	repository Repository,
	registry *Registry,
	timeClock clockutil.Clock,
	policy protection.RootPolicy,
) (*Service, error) {
	if repository == nil || registry == nil || timeClock == nil {
		return nil, fmt.Errorf(
			"%w: source service dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Service{
		repository: repository,
		registry:   registry,
		clock:      timeClock,
		policy:     policy,
	}, nil
}

func (s *Service) Create(
	ctx context.Context,
	rootID basespec.RootID,
	draft Draft,
) (Summary, error) {
	value, _, err := s.CreateWithStatus(ctx, rootID, draft)
	return value, err
}

// CreateWithStatus follows the normal caller-supplied-ID replay contract and
// additionally reports whether this invocation committed a new Source row.
//
// The status is intentionally not persisted and is not part of the public
// Artifact Store API. Higher-level provisioning workflows use it only to avoid
// discarding a Source that existed before the current request.
func (s *Service) CreateWithStatus(
	ctx context.Context,
	rootID basespec.RootID,
	draft Draft,
) (Summary, bool, error) {
	if ctx == nil {
		return Summary{}, false, fmt.Errorf(
			"%w: source creation context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return Summary{}, false, err
	}
	if err := basespec.ValidateRootID(rootID); err != nil {
		return Summary{}, false, err
	}
	if err := protection.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return Summary{}, false, err
	}
	if err := basespec.ValidateSourceID(draft.ID); err != nil {
		return Summary{}, false, err
	}
	if err := basespec.ValidateSourceKind(draft.Kind); err != nil {
		return Summary{}, false, err
	}
	if err := basespec.ValidateRequiredText(
		"source display name",
		draft.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return Summary{}, false, err
	}
	adapter, exists := s.registry.adapter(draft.Kind)
	if !exists {
		return Summary{}, false, fmt.Errorf(
			"%w: source adapter %q",
			basespec.ErrSourceUnavailable,
			draft.Kind,
		)
	}
	config, err := adapter.NormalizeConfig(ctx, draft.Config)
	if err != nil {
		return Summary{}, false, err
	}
	config, err = jsonutil.CanonicalizeObject(
		config,
		basespec.MaxConfigBytes,
	)
	if err != nil {
		return Summary{}, false, fmt.Errorf("%w: source config: %w", basespec.ErrInvalid, err)
	}

	now := clockutil.NowUTC(s.clock)
	value := Source{
		ID:          draft.ID,
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
		return Summary{}, false, err
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
		return Summary{}, false, fmt.Errorf(
			"%w: source %q creation intent differs",
			basespec.ErrConflict,
			draft.ID,
		)

	case !errors.Is(lookupErr, basespec.ErrSourceNotFound) &&
		!errors.Is(lookupErr, basespec.ErrNotFound):
		return Summary{}, false, lookupErr
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
			return Summary{}, false, cleanupBootstrap(err)
		}
	}

	createErr := s.repository.Create(ctx, value)
	if createErr == nil {
		return value.Summary(), true, nil
	}
	if !errors.Is(createErr, basespec.ErrConflict) {
		// A repository commit error can be ambiguous. Do not remove the
		// bootstrapped directory after attempting metadata publication:
		// the Source row may already be durable and must never point to
		// deleted managed content.
		return Summary{}, false, createErr
	}

	existing, lookupErr = s.repository.Get(ctx, rootID, draft.ID)
	if lookupErr != nil {
		// A global Source ID may already belong to another Root. Do not turn
		// that ID collision into an unrelated Source-not-found response. This
		// is a known non-commit outcome, so an empty managed Source directory
		// created for this failed attempt can be safely compensated.
		if errors.Is(lookupErr, basespec.ErrSourceNotFound) ||
			errors.Is(lookupErr, basespec.ErrNotFound) {
			return Summary{}, false, cleanupBootstrap(createErr)
		}

		// A non-not-found lookup failure may follow an ambiguous repository
		// result. Preserve the bootstrapped directory rather than risking
		// deletion of content referenced by a durable Source row.
		return Summary{}, false, createErr
	}
	if !sourceCreationIntentMatches(existing, value) {
		return Summary{}, false, fmt.Errorf(
			"%w: source %q creation intent differs",
			basespec.ErrConflict,
			draft.ID,
		)
	}
	return existing.Summary(), false, nil
}

func sourceCreationIntentMatches(
	existing Source,
	requested Source,
) bool {
	return existing.ID == requested.ID &&
		existing.RootID == requested.RootID &&
		existing.Kind == requested.Kind &&
		existing.DisplayName == requested.DisplayName &&
		existing.Enabled == requested.Enabled &&
		jsonutil.Equal(existing.Config, requested.Config)
}

func (s *Service) Get(
	ctx context.Context,
	rootID basespec.RootID,
	id basespec.SourceID,
) (Summary, error) {
	if err := basespec.ValidateRootID(rootID); err != nil {
		return Summary{}, err
	}
	if err := basespec.ValidateSourceID(id); err != nil {
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
	rootID basespec.RootID,
) ([]Summary, error) {
	if err := basespec.ValidateRootID(rootID); err != nil {
		return nil, err
	}
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
	rootID basespec.RootID,
	id basespec.SourceID,
	update Update,
) (Summary, error) {
	if err := protection.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return Summary{}, err
	}
	if err := basespec.ValidateRootID(rootID); err != nil {
		return Summary{}, err
	}
	if err := basespec.ValidateSourceID(id); err != nil {
		return Summary{}, err
	}
	if update.ExpectedRevision == 0 {
		return Summary{}, fmt.Errorf(
			"%w: expected source revision is required",
			basespec.ErrInvalid,
		)
	}
	current, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return Summary{}, err
	}
	if current.Revision != update.ExpectedRevision {
		return Summary{}, fmt.Errorf(
			"%w: source %q changed since it was read",
			basespec.ErrConflict,
			id,
		)
	}

	adapter, exists := s.registry.adapter(current.Kind)
	if !exists {
		return Summary{}, fmt.Errorf(
			"%w: source adapter %q",
			basespec.ErrSourceUnavailable,
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
			return Summary{}, err
		}
		normalized, err = jsonutil.CanonicalizeObject(
			normalized,
			basespec.MaxConfigBytes,
		)
		if err != nil {
			return Summary{}, err
		}
		config = normalized
	}

	next := current
	next.DisplayName = update.DisplayName
	next.Enabled = update.Enabled
	next.Config = config

	unchanged := current.DisplayName == next.DisplayName &&
		current.Enabled == next.Enabled &&
		jsonutil.Equal(current.Config, next.Config)
	if unchanged {
		return current.Summary(), nil
	}

	if current.Revision == ^uint64(0) {
		return Summary{}, fmt.Errorf("%w: source revision is exhausted", basespec.ErrInvalid)
	}
	next.Revision++
	next.ModifiedAt = clockutil.Next(s.clock, current.ModifiedAt)
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
	rootID basespec.RootID,
	id basespec.SourceID,
	expectedRevision uint64,
) (Summary, error) {
	if err := protection.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return Summary{}, err
	}
	if err := basespec.ValidateRootID(rootID); err != nil {
		return Summary{}, err
	}
	if err := basespec.ValidateSourceID(id); err != nil {
		return Summary{}, err
	}
	if expectedRevision == 0 {
		return Summary{}, fmt.Errorf(
			"%w: expected source revision is required",
			basespec.ErrInvalid,
		)
	}
	current, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return Summary{}, err
	}
	if current.Revision != expectedRevision {
		return Summary{}, fmt.Errorf(
			"%w: source %q changed since it was read",
			basespec.ErrConflict,
			id,
		)
	}
	if current.Revision == ^uint64(0) {
		return Summary{}, fmt.Errorf("%w: source revision is exhausted", basespec.ErrInvalid)
	}
	now := clockutil.Next(s.clock, current.ModifiedAt)
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
	rootID basespec.RootID,
	id basespec.SourceID,
	expectedRevision uint64,
) error {
	if err := protection.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return err
	}
	if err := basespec.ValidateRootID(rootID); err != nil {
		return err
	}
	if err := basespec.ValidateSourceID(id); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected source revision is required",
			basespec.ErrInvalid,
		)
	}
	return s.repository.Discard(ctx, rootID, id, expectedRevision)
}

func (s *Service) Purge(
	ctx context.Context,
	rootID basespec.RootID,
	id basespec.SourceID,
	expectedRevision uint64,
) error {
	if err := protection.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected source revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidateRootID(rootID); err != nil {
		return err
	}
	if err := basespec.ValidateSourceID(id); err != nil {
		return err
	}
	return s.repository.Purge(ctx, rootID, id, expectedRevision)
}

// MarkContentChanged advances Source metadata after a successful managed
// source-side mutation. The actual generation remains source-owned and is
// read from a confirmed snapshot when needed.
func (s *Service) MarkContentChanged(
	ctx context.Context,
	rootID basespec.RootID,
	id basespec.SourceID,
	expectedRevision uint64,
) (Summary, error) {
	if err := protection.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return Summary{}, err
	}
	if ctx == nil {
		return Summary{}, fmt.Errorf(
			"%w: source content-change context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return Summary{}, err
	}
	if err := basespec.ValidateRootID(rootID); err != nil {
		return Summary{}, err
	}
	if err := basespec.ValidateSourceID(id); err != nil {
		return Summary{}, err
	}
	if expectedRevision == 0 {
		return Summary{}, fmt.Errorf(
			"%w: expected source revision is required",
			basespec.ErrInvalid,
		)
	}

	current, err := s.repository.Get(ctx, rootID, id)
	if err != nil {
		return Summary{}, err
	}
	if current.Revision != expectedRevision {
		return Summary{}, basespec.ErrConflict
	}
	if current.Revision == ^uint64(0) {
		return Summary{}, fmt.Errorf("%w: source revision is exhausted", basespec.ErrInvalid)
	}

	next := current.Clone()
	next.Revision++
	next.ModifiedAt = clockutil.Next(s.clock, current.ModifiedAt)
	if err := next.Validate(); err != nil {
		return Summary{}, err
	}
	if err := s.repository.Update(ctx, next, expectedRevision); err != nil {
		return Summary{}, err
	}
	return next.Summary(), nil
}

func (s *Service) Kinds() []basespec.SourceKind {
	return s.registry.Kinds()
}
