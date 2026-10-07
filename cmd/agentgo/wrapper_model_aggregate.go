package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	modelAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model"
)

const providerRuntimeCleanupTimeout = 5 * time.Second

type modelDefaultProviderPreferences interface {
	GetDefaultProvider(ctx context.Context) (*artifactModel.ArtifactRef, error)
	SetDefaultProvider(ctx context.Context, ref *artifactModel.ArtifactRef) error
}

type modelProviderRuntime interface {
	ResolveProvider(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (inferenceSpec.ProviderParam, error)
}

type providerRuntimePublisher interface {
	ClearProvider(ctx context.Context, pName inferenceSpec.ProviderName) error
	PublishProvider(ctx context.Context, pName inferenceSpec.ProviderParam) error
}

// ModelAggregateWrapper owns application API side effects, not completion
// runtime resolution. All dependencies are wired before serving requests.
type ModelAggregateWrapper struct {
	mu sync.Mutex

	store            *modelAPI.Service
	runtime          modelProviderRuntime
	preferences      modelDefaultProviderPreferences
	fallbackProvider spec.LogicalName
	providers        providerRuntimePublisher

	writableRoot func(
		context.Context,
		rootModel.RootID,
	) (rootModel.RootID, error)
}

func (w *ModelAggregateWrapper) GetDefaultProvider() (
	*artifactModel.ArtifactRef,
	error,
) {
	if err := w.lock(); err != nil {
		return nil, err
	}
	defer w.mu.Unlock()

	ctx := context.Background()
	preference, err := w.preferences.GetDefaultProvider(ctx)
	if err != nil {
		return nil, err
	}
	return w.store.SelectDefaultProvider(
		ctx,
		preference,
		w.fallbackProvider,
	)
}

func (w *ModelAggregateWrapper) SetDefaultProvider(
	provider artifactModel.ArtifactRef,
) error {
	if err := w.lock(); err != nil {
		return err
	}
	defer w.mu.Unlock()

	ctx := context.Background()
	if err := w.store.RequireSettableDefaultProvider(
		ctx,
		provider,
	); err != nil {
		return err
	}
	return w.preferences.SetDefaultProvider(ctx, &provider)
}

func (w *ModelAggregateWrapper) ClearDefaultProvider() error {
	if err := w.lock(); err != nil {
		return err
	}
	defer w.mu.Unlock()

	return w.preferences.SetDefaultProvider(context.Background(), nil)
}

func (w *ModelAggregateWrapper) SaveProviderSettings(
	request modelAPI.SaveProviderSettingsRequest,
) (modelAPI.ProviderView, error) {
	return mutateModelProvider(
		w,
		&request.Provider,
		"provider settings were saved",
		func(ctx context.Context, _ modelAPI.ProviderView) (
			modelAPI.ProviderView,
			artifactModel.Artifact,
			error,
		) {
			value, err := w.store.SaveProviderSettings(ctx, request)
			return value, value.Artifact, err
		},
	)
}

func (w *ModelAggregateWrapper) ResetProviderSettings(
	ref artifactModel.ArtifactRef,
	expectedProviderRevision uint64,
	expectedSettingsRevision uint64,
) (modelAPI.ProviderView, error) {
	return mutateModelProvider(
		w,
		&ref,
		"provider settings were reset",
		func(ctx context.Context, _ modelAPI.ProviderView) (
			modelAPI.ProviderView,
			artifactModel.Artifact,
			error,
		) {
			value, err := w.store.ResetProviderSettings(
				ctx,
				ref,
				expectedProviderRevision,
				expectedSettingsRevision,
			)
			return value, value.Artifact, err
		},
	)
}

func (w *ModelAggregateWrapper) SetProviderAPIKey(
	request modelAPI.SetProviderAPIKeyRequest,
) (modelAPI.ProviderAPIKeyStatus, error) {
	return mutateModelProvider(
		w,
		&request.Provider,
		"provider API key was saved",
		func(ctx context.Context, current modelAPI.ProviderView) (
			modelAPI.ProviderAPIKeyStatus,
			artifactModel.Artifact,
			error,
		) {
			value, err := w.store.SetProviderAPIKey(ctx, request)
			return value, current.Artifact, err
		},
	)
}

func (w *ModelAggregateWrapper) ClearProviderAPIKey(
	ref artifactModel.ArtifactRef,
	expectedProviderRevision uint64,
	expectedAPIKeyRevision uint64,
) (modelAPI.ProviderAPIKeyStatus, error) {
	return mutateModelProvider(
		w,
		&ref,
		"provider API key was cleared",
		func(ctx context.Context, current modelAPI.ProviderView) (
			modelAPI.ProviderAPIKeyStatus,
			artifactModel.Artifact,
			error,
		) {
			value, err := w.store.ClearProviderAPIKey(
				ctx,
				ref,
				expectedProviderRevision,
				expectedAPIKeyRevision,
			)
			return value, current.Artifact, err
		},
	)
}

func (w *ModelAggregateWrapper) CreateProvider(
	request modelAPI.ManagedProviderCreateRequest,
) (modelAPI.ManagedProviderCreateResult, error) {
	return mutateModelProvider(
		w,
		nil,
		"provider was created",
		func(ctx context.Context, _ modelAPI.ProviderView) (
			modelAPI.ManagedProviderCreateResult,
			artifactModel.Artifact,
			error,
		) {
			rootID, err := w.writableRoot(ctx, request.RootID)
			if err != nil {
				return modelAPI.ManagedProviderCreateResult{}, artifactModel.Artifact{}, err
			}
			request.RootID = rootID
			value, err := w.store.CreateProvider(ctx, request)
			return value, value.Artifact, err
		},
	)
}

func (w *ModelAggregateWrapper) UpdateProvider(
	request modelAPI.ManagedProviderReplaceRequest,
) (modelAPI.ManagedProviderReplaceResult, error) {
	return mutateModelProvider(
		w,
		&request.Provider,
		"provider was updated",
		func(ctx context.Context, _ modelAPI.ProviderView) (
			modelAPI.ManagedProviderReplaceResult,
			artifactModel.Artifact,
			error,
		) {
			value, err := w.store.ReplaceProvider(ctx, request)
			return value, value.Artifact, err
		},
	)
}

func (w *ModelAggregateWrapper) DeleteProvider(
	ref artifactModel.ArtifactRef,
	expectedProviderRevision uint64,
) error {
	_, err := mutateModelProvider(
		w,
		&ref,
		"provider was deleted",
		func(ctx context.Context, _ modelAPI.ProviderView) (
			struct{},
			artifactModel.Artifact,
			error,
		) {
			err := w.store.DeleteProvider(
				ctx,
				ref,
				expectedProviderRevision,
			)
			// A deleted Provider has no enabled runtime target.
			return struct{}{}, artifactModel.Artifact{}, err
		},
	)
	return err
}

func (w *ModelAggregateWrapper) SetProviderEnabled(
	ref artifactModel.ArtifactRef,
	expectedProviderRevision uint64,
	enabled bool,
) (artifactModel.Artifact, error) {
	return mutateModelProvider(
		w,
		&ref,
		"provider enablement was saved",
		func(ctx context.Context, _ modelAPI.ProviderView) (
			artifactModel.Artifact,
			artifactModel.Artifact,
			error,
		) {
			value, err := w.store.SetProviderEnabled(
				ctx,
				ref,
				expectedProviderRevision,
				enabled,
			)
			return value, value, err
		},
	)
}

// mutateModelProvider serializes API-owned provider lifecycle changes:
//
//   - snapshot and clear an existing enabled Provider;
//   - perform the persistent mutation;
//   - restore the snapshot if persistence fails;
//   - publish the new configuration if the resulting Artifact is enabled.
//
// A nil ref denotes creation. The callback returns the resulting Artifact;
// a zero Artifact denotes deletion and therefore requires no publication.
func mutateModelProvider[T any](
	w *ModelAggregateWrapper,
	ref *artifactModel.ArtifactRef,
	committedMessage string,
	write func(context.Context, modelAPI.ProviderView) (T, artifactModel.Artifact, error),
) (T, error) {
	var zero T
	if err := w.lock(); err != nil {
		return zero, err
	}
	defer w.mu.Unlock()

	if w.providers == nil {
		return zero, spec.ErrClosed
	}

	ctx := context.Background()
	var (
		current  modelAPI.ProviderView
		previous *inferenceSpec.ProviderParam
	)
	if ref != nil {
		var err error
		current, err = w.store.GetProvider(ctx, *ref)
		if err != nil {
			return zero, err
		}
		if current.Artifact.Enabled {
			// An unresolved existing configuration must not prevent a user
			// from repairing it. Restore only when a snapshot was available.
			if value, err := w.runtime.ResolveProvider(
				ctx,
				current.Artifact.Ref(),
			); err == nil {
				previous = &value
			}
			if err := w.providers.ClearProvider(
				ctx,
				inferenceSpec.ProviderName(current.Artifact.LogicalName),
			); err != nil {
				return zero, err
			}
		}
	}

	value, artifact, err := write(ctx, current)
	if err != nil {
		if previous != nil {
			restoreCtx, cancel := context.WithTimeout(
				context.Background(),
				providerRuntimeCleanupTimeout,
			)
			defer cancel()

			if restoreErr := w.providers.PublishProvider(
				restoreCtx,
				*previous,
			); restoreErr != nil {
				err = errors.Join(
					err,
					fmt.Errorf("restore runtime Provider: %w", restoreErr),
				)
			}
		}
		return zero, err
	}
	if !artifact.Enabled {
		return value, nil
	}

	provider, err := w.runtime.ResolveProvider(ctx, artifact.Ref())
	if err == nil {
		err = w.providers.PublishProvider(ctx, provider)
	}
	if err != nil {
		return value, fmt.Errorf(
			"%s but runtime Provider refresh failed: %w",
			committedMessage,
			err,
		)
	}
	return value, nil
}

func (w *ModelAggregateWrapper) setProviderRuntimePublisher(
	publisher providerRuntimePublisher,
) error {
	if err := w.lock(); err != nil {
		return err
	}
	defer w.mu.Unlock()

	if publisher == nil {
		return fmt.Errorf(
			"%w: Model Provider runtime publisher is required",
			spec.ErrInvalid,
		)
	}
	w.providers = publisher
	return nil
}

// lock acquires the wrapper lifecycle lock on success.
func (w *ModelAggregateWrapper) lock() error {
	if w == nil {
		return spec.ErrClosed
	}
	w.mu.Lock()
	if w.store == nil || w.runtime == nil || w.preferences == nil {
		w.mu.Unlock()
		return spec.ErrClosed
	}
	return nil
}

func (w *ModelAggregateWrapper) close() {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	w.store = nil
	w.runtime = nil
	w.preferences = nil
	w.providers = nil
	w.fallbackProvider = ""
	w.writableRoot = nil
}
