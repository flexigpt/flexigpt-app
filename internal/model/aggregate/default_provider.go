package aggregate

import (
	"context"

	"github.com/flexigpt/inference-go/modelpreset"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

const baseDefaultProviderName = spec.LogicalName(
	modelpreset.ProviderOpenAIResponses,
)

// DefaultProviderPreferences stores only the user's optional preference.
// It does not check Provider availability or resolve Models.
type DefaultProviderPreferences interface {
	GetDefaultProvider(ctx context.Context) (*artifactModel.ArtifactRef, error)
	SetDefaultProvider(ctx context.Context, ref *artifactModel.ArtifactRef) error
}

// GetDefaultProvider returns the effective Provider without resolving Models,
// credentials, capabilities, or runtime configuration.
func (s *Service) GetDefaultProvider(
	ctx context.Context,
) (*artifactModel.ArtifactRef, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}

	value, err := s.preferences.GetDefaultProvider(ctx)
	if err != nil {
		return nil, err
	}
	return s.store.SelectDefaultProvider(
		ctx,
		value,
		baseDefaultProviderName,
	)
}

// SetDefaultProvider stores a loose preference. The selected Provider must
// currently be enabled and have a configured API key. Fallbacks are never
// written back over the saved preference.
func (s *Service) SetDefaultProvider(
	ctx context.Context,
	provider artifactModel.ArtifactRef,
) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if err := s.store.RequireSettableDefaultProvider(
		ctx,
		provider,
	); err != nil {
		return err
	}
	value := provider
	return s.preferences.SetDefaultProvider(ctx, &value)
}

// ClearDefaultProvider removes the saved preference. The next
// GetDefaultProvider call returns the normal fallback Provider.
func (s *Service) ClearDefaultProvider(
	ctx context.Context,
) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	return s.preferences.SetDefaultProvider(ctx, nil)
}
