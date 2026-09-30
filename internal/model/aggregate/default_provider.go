package aggregate

import (
	"context"

	"github.com/flexigpt/inference-go/modelpreset"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
)

const baseDefaultProviderName = basespec.LogicalName(
	modelpreset.ProviderOpenAIResponses,
)

// DefaultProviderPreferences stores only the user's optional preference.
// It does not check Provider availability or resolve Models.
type DefaultProviderPreferences interface {
	GetDefaultProvider(ctx context.Context) (*artifact.ArtifactRef, error)
	SetDefaultProvider(ctx context.Context, ref *artifact.ArtifactRef) error
}

// GetDefaultModelProvider returns the effective Provider without resolving
// Models, credentials, capabilities, or runtime configuration.
func (s *Service) GetDefaultModelProvider(
	ctx context.Context,
) (*artifact.ArtifactRef, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}

	preferred, err := s.preferences.GetDefaultProvider(ctx)
	if err != nil {
		return nil, err
	}
	return s.store.SelectDefaultProvider(
		ctx,
		preferred,
		baseDefaultProviderName,
	)
}

// SetDefaultModelProvider stores a loose preference. An explicit non-nil
// selection must currently be enabled and have a configured API key. Nil
// clears the preference. Fallbacks are never written back over it.
func (s *Service) SetDefaultModelProvider(
	ctx context.Context,
	provider *artifact.ArtifactRef,
) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if provider != nil {
		if err := s.store.RequireSettableDefaultProvider(
			ctx,
			*provider,
		); err != nil {
			return err
		}
	}
	return s.preferences.SetDefaultProvider(ctx, provider)
}

// GetModelProviderDefaultModel is deliberately separate from Provider
// selection. It uses the existing per-Provider default-Model resolution rules.
func (s *Service) GetModelProviderDefaultModel(
	ctx context.Context,
	provider artifact.ArtifactRef,
) (artifact.ArtifactRef, error) {
	if err := s.ready(ctx); err != nil {
		return artifact.ArtifactRef{}, err
	}

	value, err := s.store.ResolveProviderDefaultModel(ctx, provider)
	if err != nil {
		return artifact.ArtifactRef{}, err
	}
	return value.Resolved.Model.Artifact.Ref(), nil
}
