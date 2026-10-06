package refresh

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// PublishSourceLifecycle is Source's injected aggregate publication port.
// Refresh reads the committed preconditions, asks Artifact to derive explicit
// source-owned updates, and delegates one atomic command to persistence.
func (s *Service) PublishSourceLifecycle(
	ctx context.Context,
	transition source.LifecycleTransition,
) error {
	if s == nil || s.sources == nil || s.artifacts == nil ||
		s.states == nil || s.synchronizer == nil || s.publisher == nil {
		return spec.ErrClosed
	}

	if err := transition.Validate(); err != nil {
		return err
	}
	if err := root.RequireMutableRoot(ctx, s.policy, transition.Source.RootID); err != nil {
		return err
	}

	current, err := s.sources.Get(ctx, transition.Source.RootID, transition.Source.ID)
	if err != nil {
		return err
	}
	if current.Revision != transition.ExpectedSourceRevision {
		return spec.ErrConflict
	}

	expectedRefreshRevision := uint64(0)
	previous, err := s.states.GetRefreshState(ctx, transition.Source.RootID, transition.Source.ID)
	switch {
	case err == nil:
		expectedRefreshRevision = previous.Revision
	case errors.Is(err, spec.ErrRefreshStateNotFound):
	default:
		return err
	}

	existing, err := s.artifacts.ListBySource(ctx, transition.Source.RootID, transition.Source.ID)
	if err != nil {
		return err
	}
	updates, err := s.synchronizer.DeriveLifecycleInvalidation(ctx, transition, existing)
	if err != nil {
		return err
	}
	return s.publisher.PublishLifecycle(ctx, LifecyclePublication{
		Transition:              transition,
		ExpectedRefreshRevision: expectedRefreshRevision,
		ArtifactUpdates:         updates,
	})
}
