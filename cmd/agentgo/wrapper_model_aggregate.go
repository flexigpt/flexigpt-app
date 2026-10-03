package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	modelAggregate "github.com/flexigpt/flexigpt-app/internal/model/aggregate"
	modelConsumerAPI "github.com/flexigpt/flexigpt-app/internal/model/store/consumerapi"
)

type ModelAggregateWrapper struct {
	service *modelAggregate.Service
}

func (w *ModelAggregateWrapper) GetDefaultProvider() (
	*artifactModel.ArtifactRef,
	error,
) {
	if w == nil || w.service == nil {
		return nil, spec.ErrClosed
	}
	return w.service.GetDefaultProvider(context.Background())
}

func (w *ModelAggregateWrapper) SetDefaultProvider(
	provider artifactModel.ArtifactRef,
) error {
	if w == nil || w.service == nil {
		return spec.ErrClosed
	}
	return w.service.SetDefaultProvider(context.Background(), provider)
}

func (w *ModelAggregateWrapper) ClearDefaultProvider() error {
	if w == nil || w.service == nil {
		return spec.ErrClosed
	}
	return w.service.ClearDefaultProvider(context.Background())
}

func (w *ModelAggregateWrapper) SaveProviderSettings(
	request modelConsumerAPI.SaveProviderSettingsRequest,
) (modelConsumerAPI.ProviderView, error) {
	if w == nil || w.service == nil {
		return modelConsumerAPI.ProviderView{}, spec.ErrClosed
	}
	return w.service.SaveProviderSettings(context.Background(), request)
}

func (w *ModelAggregateWrapper) ResetProviderSettings(
	ref artifactModel.ArtifactRef,
	expectedProviderRevision uint64,
	expectedSettingsRevision uint64,
) (modelConsumerAPI.ProviderView, error) {
	if w == nil || w.service == nil {
		return modelConsumerAPI.ProviderView{}, spec.ErrClosed
	}
	return w.service.ResetProviderSettings(
		context.Background(),
		ref,
		expectedProviderRevision,
		expectedSettingsRevision,
	)
}

func (w *ModelAggregateWrapper) SetProviderAPIKey(
	request modelConsumerAPI.SetProviderAPIKeyRequest,
) (modelConsumerAPI.ProviderAPIKeyStatus, error) {
	if w == nil || w.service == nil {
		return modelConsumerAPI.ProviderAPIKeyStatus{}, spec.ErrClosed
	}
	return w.service.SetProviderAPIKey(context.Background(), request)
}

func (w *ModelAggregateWrapper) ClearProviderAPIKey(
	ref artifactModel.ArtifactRef,
	expectedProviderRevision uint64,
	expectedAPIKeyRevision uint64,
) (modelConsumerAPI.ProviderAPIKeyStatus, error) {
	if w == nil || w.service == nil {
		return modelConsumerAPI.ProviderAPIKeyStatus{}, spec.ErrClosed
	}
	return w.service.ClearProviderAPIKey(
		context.Background(),
		ref,
		expectedProviderRevision,
		expectedAPIKeyRevision,
	)
}

func (w *ModelAggregateWrapper) CreateProvider(
	request modelConsumerAPI.ManagedProviderCreateRequest,
) (modelConsumerAPI.ManagedProviderCreateResult, error) {
	if w == nil || w.service == nil {
		return modelConsumerAPI.ManagedProviderCreateResult{}, spec.ErrClosed
	}
	return w.service.CreateProvider(context.Background(), request)
}

func (w *ModelAggregateWrapper) UpdateProvider(
	request modelConsumerAPI.ManagedProviderReplaceRequest,
) (modelConsumerAPI.ManagedProviderReplaceResult, error) {
	if w == nil || w.service == nil {
		return modelConsumerAPI.ManagedProviderReplaceResult{}, spec.ErrClosed
	}
	return w.service.UpdateProvider(context.Background(), request)
}

func (w *ModelAggregateWrapper) DeleteProvider(
	ref artifactModel.ArtifactRef,
	expectedProviderRevision uint64,
) error {
	if w == nil || w.service == nil {
		return spec.ErrClosed
	}
	return w.service.DeleteProvider(
		context.Background(),
		ref,
		expectedProviderRevision,
	)
}

func (w *ModelAggregateWrapper) SetProviderEnabled(
	ref artifactModel.ArtifactRef,
	expectedProviderRevision uint64,
	enabled bool,
) (artifactModel.Artifact, error) {
	if w == nil || w.service == nil {
		return artifactModel.Artifact{}, spec.ErrClosed
	}
	return w.service.SetProviderEnabled(
		context.Background(),
		ref,
		expectedProviderRevision,
		enabled,
	)
}

func (w *ModelAggregateWrapper) setProviderRuntimePublisher(
	publisher modelAggregate.ProviderRuntimePublisher,
) error {
	if w == nil || w.service == nil {
		return spec.ErrClosed
	}
	return w.service.SetProviderRuntimePublisher(publisher)
}

func (w *ModelAggregateWrapper) targetMappers() (
	map[declaration.Type]resolve.ArtifactTargetMapper,
	error,
) {
	if w == nil || w.service == nil {
		return nil, spec.ErrClosed
	}
	return map[declaration.Type]resolve.ArtifactTargetMapper{
		declaration.TypeModel: w.service,
	}, nil
}

// initModelProviderRuntime restores the shared inference registry at startup.
// Management Root selection includes built-ins and skips retired Roots.
func initModelProviderRuntime(
	ctx context.Context,
	store *ModelStoreWrapper,
	aggregate *ModelAggregateWrapper,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: Model Provider startup context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if store == nil || store.management == nil ||
		aggregate == nil || aggregate.service == nil {
		return spec.ErrClosed
	}

	roots, err := store.managementRootIDs(ctx, "")
	if err != nil {
		return err
	}

	var (
		providers []modelConsumerAPI.ProviderListItem
		result    error
	)
	for _, rootID := range roots {
		if err := ctx.Err(); err != nil {
			return errors.Join(result, err)
		}
		values, err := store.management.ListProviders(ctx, rootID)
		if err != nil {
			result = errors.Join(
				result,
				fmt.Errorf("list Model Providers in Root %q: %w", rootID, err),
			)
			continue
		}
		providers = append(providers, values...)
	}
	return errors.Join(
		result,
		aggregate.service.InitializeProviderRuntime(ctx, providers),
	)
}

func (w *ModelAggregateWrapper) close() {
	if w == nil {
		return
	}
	w.service = nil
}
