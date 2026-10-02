package aggregate

import (
	"context"
	"errors"
	"fmt"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/artifact"
	modelConsumerAPI "github.com/flexigpt/flexigpt-app/internal/model/store/consumerapi"
)

// ProviderRuntimePublisher owns the long-lived in-memory inference Provider
// registration used by applications other than the completion-specific runtime.
type ProviderRuntimePublisher interface {
	ClearProvider(
		ctx context.Context,
		provider inferenceSpec.ProviderName,
	) error

	PublishProvider(
		ctx context.Context,
		provider inferenceSpec.ProviderParam,
	) error
}

func (s *Service) SetProviderRuntimePublisher(
	publisher ProviderRuntimePublisher,
) error {
	if s == nil {
		return basespec.ErrClosed
	}
	if publisher == nil {
		return fmt.Errorf(
			"%w: Model Provider runtime publisher is required",
			basespec.ErrInvalid,
		)
	}
	s.providers = publisher
	return nil
}

func (s *Service) SaveProviderSettings(
	ctx context.Context,
	request modelConsumerAPI.SaveProviderSettingsRequest,
) (modelConsumerAPI.ProviderView, error) {
	if err := s.ready(ctx); err != nil {
		return modelConsumerAPI.ProviderView{}, err
	}

	current, err := s.store.GetProvider(ctx, request.Provider)
	if err != nil {
		return modelConsumerAPI.ProviderView{}, err
	}
	previous, err := s.clearEnabledProvider(ctx, current)
	if err != nil {
		return modelConsumerAPI.ProviderView{}, err
	}

	updated, err := s.store.SaveProviderSettings(ctx, request)
	if err != nil {
		return modelConsumerAPI.ProviderView{}, s.restoreProviderRuntime(
			ctx,
			previous,
			err,
		)
	}
	if !updated.Artifact.Enabled {
		return updated, nil
	}
	if err := s.publishProviderRuntime(ctx, updated.Artifact.Ref()); err != nil {
		return updated, fmt.Errorf(
			"provider settings were saved but runtime Provider refresh failed: %w",
			err,
		)
	}
	return updated, nil
}

func (s *Service) ResetProviderSettings(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedProviderRevision uint64,
	expectedSettingsRevision uint64,
) (modelConsumerAPI.ProviderView, error) {
	if err := s.ready(ctx); err != nil {
		return modelConsumerAPI.ProviderView{}, err
	}

	current, err := s.store.GetProvider(ctx, ref)
	if err != nil {
		return modelConsumerAPI.ProviderView{}, err
	}
	previous, err := s.clearEnabledProvider(ctx, current)
	if err != nil {
		return modelConsumerAPI.ProviderView{}, err
	}

	updated, err := s.store.ResetProviderSettings(
		ctx,
		ref,
		expectedProviderRevision,
		expectedSettingsRevision,
	)
	if err != nil {
		return modelConsumerAPI.ProviderView{}, s.restoreProviderRuntime(
			ctx,
			previous,
			err,
		)
	}
	if !updated.Artifact.Enabled {
		return updated, nil
	}
	if err := s.publishProviderRuntime(ctx, updated.Artifact.Ref()); err != nil {
		return updated, fmt.Errorf(
			"provider settings were reset but runtime Provider refresh failed: %w",
			err,
		)
	}
	return updated, nil
}

func (s *Service) SetProviderAPIKey(
	ctx context.Context,
	request modelConsumerAPI.SetProviderAPIKeyRequest,
) (modelConsumerAPI.ProviderAPIKeyStatus, error) {
	if err := s.ready(ctx); err != nil {
		return modelConsumerAPI.ProviderAPIKeyStatus{}, err
	}

	current, err := s.store.GetProvider(ctx, request.Provider)
	if err != nil {
		return modelConsumerAPI.ProviderAPIKeyStatus{}, err
	}
	previous, err := s.clearEnabledProvider(ctx, current)
	if err != nil {
		return modelConsumerAPI.ProviderAPIKeyStatus{}, err
	}

	updated, err := s.store.SetProviderAPIKey(ctx, request)
	if err != nil {
		return modelConsumerAPI.ProviderAPIKeyStatus{}, s.restoreProviderRuntime(
			ctx,
			previous,
			err,
		)
	}
	if !current.Artifact.Enabled {
		return updated, nil
	}
	if err := s.publishProviderRuntime(ctx, request.Provider); err != nil {
		return updated, fmt.Errorf(
			"provider API key was saved but runtime Provider refresh failed: %w",
			err,
		)
	}
	return updated, nil
}

func (s *Service) ClearProviderAPIKey(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedProviderRevision uint64,
	expectedAPIKeyRevision uint64,
) (modelConsumerAPI.ProviderAPIKeyStatus, error) {
	if err := s.ready(ctx); err != nil {
		return modelConsumerAPI.ProviderAPIKeyStatus{}, err
	}

	current, err := s.store.GetProvider(ctx, ref)
	if err != nil {
		return modelConsumerAPI.ProviderAPIKeyStatus{}, err
	}
	previous, err := s.clearEnabledProvider(ctx, current)
	if err != nil {
		return modelConsumerAPI.ProviderAPIKeyStatus{}, err
	}

	updated, err := s.store.ClearProviderAPIKey(
		ctx,
		ref,
		expectedProviderRevision,
		expectedAPIKeyRevision,
	)
	if err != nil {
		return modelConsumerAPI.ProviderAPIKeyStatus{}, s.restoreProviderRuntime(
			ctx,
			previous,
			err,
		)
	}
	if !current.Artifact.Enabled {
		return updated, nil
	}
	if err := s.publishProviderRuntime(ctx, ref); err != nil {
		return updated, fmt.Errorf(
			"provider API key was cleared but runtime Provider refresh failed: %w",
			err,
		)
	}
	return updated, nil
}

func (s *Service) CreateProvider(
	ctx context.Context,
	request modelConsumerAPI.ManagedProviderCreateRequest,
) (modelConsumerAPI.ManagedProviderCreateResult, error) {
	if err := s.ready(ctx); err != nil {
		return modelConsumerAPI.ManagedProviderCreateResult{}, err
	}
	if request.Enabled {
		if _, err := s.providerRuntimePublisher(); err != nil {
			return modelConsumerAPI.ManagedProviderCreateResult{}, err
		}
	}

	created, err := s.store.CreateProvider(ctx, request)
	if err != nil {
		return modelConsumerAPI.ManagedProviderCreateResult{}, err
	}
	if !created.Artifact.Enabled {
		return created, nil
	}
	if err := s.publishProviderRuntime(ctx, created.Artifact.Ref()); err != nil {
		return created, fmt.Errorf(
			"provider was created but runtime Provider refresh failed: %w",
			err,
		)
	}
	return created, nil
}

func (s *Service) UpdateProvider(
	ctx context.Context,
	request modelConsumerAPI.ManagedProviderReplaceRequest,
) (modelConsumerAPI.ManagedProviderReplaceResult, error) {
	if err := s.ready(ctx); err != nil {
		return modelConsumerAPI.ManagedProviderReplaceResult{}, err
	}

	current, err := s.store.GetProvider(ctx, request.Provider)
	if err != nil {
		return modelConsumerAPI.ManagedProviderReplaceResult{}, err
	}
	previous, err := s.clearEnabledProvider(ctx, current)
	if err != nil {
		return modelConsumerAPI.ManagedProviderReplaceResult{}, err
	}

	updated, err := s.store.ReplaceProvider(ctx, request)
	if err != nil {
		return modelConsumerAPI.ManagedProviderReplaceResult{}, s.restoreProviderRuntime(
			ctx,
			previous,
			err,
		)
	}
	if !updated.Artifact.Enabled {
		return updated, nil
	}
	if err := s.publishProviderRuntime(ctx, updated.Artifact.Ref()); err != nil {
		return updated, fmt.Errorf(
			"provider was updated but runtime Provider refresh failed: %w",
			err,
		)
	}
	return updated, nil
}

func (s *Service) DeleteProvider(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedProviderRevision uint64,
) error {
	if err := s.ready(ctx); err != nil {
		return err
	}

	current, err := s.store.GetProvider(ctx, ref)
	if err != nil {
		return err
	}
	previous, err := s.clearEnabledProvider(ctx, current)
	if err != nil {
		return err
	}

	if err := s.store.DeleteProvider(
		ctx,
		ref,
		expectedProviderRevision,
	); err != nil {
		return s.restoreProviderRuntime(ctx, previous, err)
	}
	return nil
}

func (s *Service) SetProviderEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedProviderRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if err := s.ready(ctx); err != nil {
		return artifact.Artifact{}, err
	}
	if enabled {
		if _, err := s.providerRuntimePublisher(); err != nil {
			return artifact.Artifact{}, err
		}
	}

	current, err := s.store.GetProvider(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, err
	}
	previous, err := s.clearEnabledProvider(ctx, current)
	if err != nil {
		return artifact.Artifact{}, err
	}

	updated, err := s.store.SetProviderEnabled(
		ctx,
		ref,
		expectedProviderRevision,
		enabled,
	)
	if err != nil {
		return artifact.Artifact{}, s.restoreProviderRuntime(
			ctx,
			previous,
			err,
		)
	}
	if !enabled {
		return updated, nil
	}
	if err := s.publishProviderRuntime(ctx, updated.Ref()); err != nil {
		return updated, fmt.Errorf(
			"provider was enabled but runtime Provider refresh failed: %w",
			err,
		)
	}
	return updated, nil
}

func (s *Service) clearEnabledProvider(
	ctx context.Context,
	current modelConsumerAPI.ProviderView,
) (*inferenceSpec.ProviderParam, error) {
	if !current.Artifact.Enabled {
		//nolint:nilnil // Ok.
		return nil, nil
	}

	publisher, err := s.providerRuntimePublisher()
	if err != nil {
		return nil, err
	}

	var previous *inferenceSpec.ProviderParam
	if value, err := s.resolveProviderRuntime(
		ctx,
		current.Artifact.Ref(),
	); err == nil {
		previous = &value
	}

	if err := publisher.ClearProvider(
		ctx,
		inferenceSpec.ProviderName(current.Artifact.LogicalName),
	); err != nil {
		return nil, err
	}
	return previous, nil
}

func (s *Service) publishProviderRuntime(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	publisher, err := s.providerRuntimePublisher()
	if err != nil {
		return err
	}
	value, err := s.resolveProviderRuntime(ctx, ref)
	if err != nil {
		return err
	}
	return publisher.PublishProvider(ctx, value)
}

func (s *Service) resolveProviderRuntime(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (inferenceSpec.ProviderParam, error) {
	resolved, err := s.store.ResolveProvider(ctx, ref)
	if err != nil {
		return inferenceSpec.ProviderParam{}, err
	}
	return s.runtime.ResolveProviderRuntime(ctx, resolved)
}

func (s *Service) restoreProviderRuntime(
	ctx context.Context,
	previous *inferenceSpec.ProviderParam,
	cause error,
) error {
	if previous == nil {
		return cause
	}

	publisher, err := s.providerRuntimePublisher()
	if err != nil {
		return errors.Join(cause, err)
	}
	if err := publisher.PublishProvider(ctx, *previous); err != nil {
		return errors.Join(
			cause,
			fmt.Errorf("restore runtime Provider: %w", err),
		)
	}
	return cause
}

func (s *Service) providerRuntimePublisher() (
	ProviderRuntimePublisher,
	error,
) {
	if s == nil || s.providers == nil {
		return nil, basespec.ErrClosed
	}
	return s.providers, nil
}
