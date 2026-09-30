package main

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	modelAggregate "github.com/flexigpt/flexigpt-app/internal/model/aggregate"
	"github.com/flexigpt/flexigpt-app/internal/model/inferenceadapter"
	modelBuiltin "github.com/flexigpt/flexigpt-app/internal/model/store/builtin"
	modelConsumerAPI "github.com/flexigpt/flexigpt-app/internal/model/store/consumerapi"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/model/store/overlay"
)

type ModelStoreWrapper struct {
	api        *modelConsumerAPI.API
	management *modelConsumerAPI.CatalogStore
	roots      compositionapi.RootAPI
	protection compositionapi.ProtectionAPI
}

func InitModelWrappers(
	ctx context.Context,
	storeWrapper *ModelStoreWrapper,
	aggregateWrapper *ModelAggregateWrapper,
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	roots compositionapi.RootAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	protection compositionapi.ProtectionAPI,
	protectedOverlays compositionapi.ProtectedOverlayAPI,
	secretBindings compositionapi.SecretBindingAPI,
	secretRuntime compositionapi.SecretRuntimeAPI,
	localState compositionapi.LocalStateMaintenanceAPI,
	storeOverlays compositionapi.StoreOverlayAPI,
	hydrator topology.CompiledHydrationCoordinator,
) (builtin.HydrationInstaller, error) {
	if storeWrapper == nil || aggregateWrapper == nil {
		return nil, errors.New("model wrapper receivers are incomplete")
	}
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		managedArtifacts == nil ||
		roots == nil ||
		protection == nil ||
		protectedOverlays == nil ||
		secretBindings == nil ||
		secretRuntime == nil ||
		localState == nil ||
		storeOverlays == nil ||
		hydrator == nil {
		return nil, errors.New("model wrapper dependencies are incomplete")
	}

	preferences, err := newArtifactModelDefaultProviderPreferences(storeOverlays)
	if err != nil {
		return nil, err
	}
	overlays, err := modelOverlay.NewArtifactOverlayRepository(
		modelOverlay.ArtifactOverlayDependencies{
			Artifacts:        artifacts,
			Protection:       protection,
			ProtectedOverlay: protectedOverlays,
			Secrets:          secretBindings,
			LocalState:       localState,
		},
	)
	if err != nil {
		return nil, err
	}
	credentials, err := newArtifactModelCredentialResolver(secretRuntime)
	if err != nil {
		return nil, err
	}
	runtimeAdapter, err := inferenceadapter.NewRuntimeAdapter(credentials)
	if err != nil {
		return nil, err
	}

	api, err := modelConsumerAPI.New(modelConsumerAPI.Dependencies{
		Sources:          sources,
		Discovery:        discovery,
		Artifacts:        artifacts,
		ManagedArtifacts: managedArtifacts,
		Protection:       protection,
		Overlays:         overlays,
		Adapters:         runtimeAdapter,
		BuiltinRoot:      documentTopology.BuiltinRootID(),
	})
	if err != nil {
		return nil, err
	}
	management, err := modelConsumerAPI.NewManagementStore(api)
	if err != nil {
		return nil, err
	}
	catalog, err := modelConsumerAPI.NewCatalogStore(api)
	if err != nil {
		return nil, err
	}
	aggregateService, err := modelAggregate.New(
		management,
		runtimeAdapter,
		preferences,
	)
	if err != nil {
		return nil, err
	}
	cleanup, err := modelConsumerAPI.NewBuiltinPackageCleanup(api)
	if err != nil {
		return nil, err
	}
	installer, err := modelBuiltin.NewInstaller(
		modelBuiltin.InstallerDependencies{
			Hydrator: hydrator,
			Cleanup:  cleanup,
		},
	)
	if err != nil {
		return nil, err
	}

	storeWrapper.api = api
	storeWrapper.management = catalog
	storeWrapper.roots = roots
	storeWrapper.protection = protection
	aggregateWrapper.service = aggregateService

	_ = ctx
	return installer, nil
}

func (w *ModelStoreWrapper) ListModelProviders(
	rootID root.RootID,
) ([]modelConsumerAPI.ProviderListItem, error) {
	if w == nil || w.management == nil {
		return nil, basespec.ErrClosed
	}

	ctx := context.Background()
	roots, err := w.managementRootIDs(ctx, rootID)
	if err != nil {
		return nil, err
	}

	output := make([]modelConsumerAPI.ProviderListItem, 0)
	for _, currentRoot := range roots {
		values, err := w.management.ListProviders(ctx, currentRoot)
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}
	return output, nil
}

func (w *ModelStoreWrapper) ListModels(
	rootID root.RootID,
) ([]modelConsumerAPI.ModelListItem, error) {
	if w == nil || w.management == nil {
		return nil, basespec.ErrClosed
	}

	ctx := context.Background()
	roots, err := w.managementRootIDs(ctx, rootID)
	if err != nil {
		return nil, err
	}

	output := make([]modelConsumerAPI.ModelListItem, 0)
	for _, currentRoot := range roots {
		values, err := w.management.ListModels(ctx, currentRoot)
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}
	return output, nil
}

func (w *ModelStoreWrapper) GetModelProvider(
	ref artifact.ArtifactRef,
) (modelConsumerAPI.ProviderView, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ProviderView{}, basespec.ErrClosed
	}
	return w.api.GetProvider(context.Background(), ref)
}

func (w *ModelStoreWrapper) GetModel(
	ref artifact.ArtifactRef,
) (modelConsumerAPI.ModelView, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ModelView{}, basespec.ErrClosed
	}
	return w.api.GetModel(context.Background(), ref)
}

func (w *ModelStoreWrapper) GetModelProviderRuntimeOverlay(
	ref artifact.ArtifactRef,
) (modelConsumerAPI.ProviderRuntimeOverlayView, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ProviderRuntimeOverlayView{}, basespec.ErrClosed
	}
	return w.api.GetProviderRuntimeOverlay(context.Background(), ref)
}

func (w *ModelStoreWrapper) UpdateModelProviderRuntimeOverlay(
	request modelConsumerAPI.ProviderRuntimeOverlayUpdateRequest,
) (modelConsumerAPI.ProviderRuntimeOverlayView, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ProviderRuntimeOverlayView{}, basespec.ErrClosed
	}
	return w.api.UpdateProviderRuntimeOverlay(
		context.Background(),
		request,
	)
}

func (w *ModelStoreWrapper) DeleteModelProviderRuntimeOverlay(
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	expectedOverlayRevision uint64,
) error {
	if w == nil || w.api == nil {
		return basespec.ErrClosed
	}
	return w.api.DeleteProviderRuntimeOverlay(
		context.Background(),
		ref,
		expectedArtifactRevision,
		expectedOverlayRevision,
	)
}

func (w *ModelStoreWrapper) GetManagedModelRuntimeOverlay(
	ref artifact.ArtifactRef,
) (modelConsumerAPI.ModelRuntimeOverlayView, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ModelRuntimeOverlayView{}, basespec.ErrClosed
	}
	return w.api.GetModelRuntimeOverlay(context.Background(), ref)
}

func (w *ModelStoreWrapper) UpdateManagedModelRuntimeOverlay(
	request modelConsumerAPI.ModelRuntimeOverlayUpdateRequest,
) (modelConsumerAPI.ModelRuntimeOverlayView, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ModelRuntimeOverlayView{}, basespec.ErrClosed
	}
	return w.api.UpdateModelRuntimeOverlay(
		context.Background(),
		request,
	)
}

func (w *ModelStoreWrapper) DeleteManagedModelRuntimeOverlay(
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	expectedOverlayRevision uint64,
) error {
	if w == nil || w.api == nil {
		return basespec.ErrClosed
	}
	return w.api.DeleteModelRuntimeOverlay(
		context.Background(),
		ref,
		expectedArtifactRevision,
		expectedOverlayRevision,
	)
}

func (w *ModelStoreWrapper) SetModelProviderCredential(
	request modelConsumerAPI.ProviderCredentialUpdateRequest,
) (modelConsumerAPI.ProviderRuntimeOverlayView, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ProviderRuntimeOverlayView{}, basespec.ErrClosed
	}
	return w.api.UpdateProviderCredential(
		context.Background(),
		request,
	)
}

func (w *ModelStoreWrapper) CreateModelProvider(
	request modelConsumerAPI.ManagedProviderCreateRequest,
) (modelConsumerAPI.ManagedProviderCreateResult, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ManagedProviderCreateResult{}, basespec.ErrClosed
	}

	rootID, err := w.writableManagementRoot(
		context.Background(),
		request.RootID,
	)
	if err != nil {
		return modelConsumerAPI.ManagedProviderCreateResult{}, err
	}
	request.RootID = rootID
	return w.api.CreateProvider(context.Background(), request)
}

func (w *ModelStoreWrapper) ReplaceModelProvider(
	request modelConsumerAPI.ManagedProviderReplaceRequest,
) (modelConsumerAPI.ManagedProviderReplaceResult, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ManagedProviderReplaceResult{}, basespec.ErrClosed
	}
	return w.api.ReplaceProvider(context.Background(), request)
}

func (w *ModelStoreWrapper) DeleteModelProvider(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if w == nil || w.api == nil {
		return basespec.ErrClosed
	}
	return w.api.DeleteProvider(
		context.Background(),
		ref,
		expectedRevision,
	)
}

func (w *ModelStoreWrapper) CreateManagedModel(
	request modelConsumerAPI.ManagedModelCreateRequest,
) (modelConsumerAPI.ManagedModelCreateResult, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ManagedModelCreateResult{}, basespec.ErrClosed
	}

	rootID, err := w.writableManagementRoot(
		context.Background(),
		request.RootID,
	)
	if err != nil {
		return modelConsumerAPI.ManagedModelCreateResult{}, err
	}
	request.RootID = rootID
	return w.api.CreateModel(context.Background(), request)
}

func (w *ModelStoreWrapper) ReplaceManagedModel(
	request modelConsumerAPI.ManagedModelReplaceRequest,
) (modelConsumerAPI.ManagedModelReplaceResult, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ManagedModelReplaceResult{}, basespec.ErrClosed
	}
	return w.api.ReplaceModel(context.Background(), request)
}

func (w *ModelStoreWrapper) DeleteManagedModel(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if w == nil || w.api == nil {
		return basespec.ErrClosed
	}
	return w.api.DeleteModel(
		context.Background(),
		ref,
		expectedRevision,
	)
}

func (w *ModelStoreWrapper) SetModelProviderEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if w == nil || w.api == nil {
		return artifact.Artifact{}, basespec.ErrClosed
	}
	return w.api.SetProviderEnabled(
		context.Background(),
		ref,
		expectedRevision,
		enabled,
	)
}

func (w *ModelStoreWrapper) SetModelEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if w == nil || w.api == nil {
		return artifact.Artifact{}, basespec.ErrClosed
	}
	return w.api.SetModelEnabled(
		context.Background(),
		ref,
		expectedRevision,
		enabled,
	)
}

func (w *ModelStoreWrapper) managementRootIDs(
	ctx context.Context,
	requested root.RootID,
) ([]root.RootID, error) {
	if w == nil || w.roots == nil || w.protection == nil {
		return nil, basespec.ErrClosed
	}
	if requested != "" {
		if err := requested.Validate(); err != nil {
			return nil, err
		}
		return []root.RootID{requested}, nil
	}

	values, err := w.roots.List(ctx)
	if err != nil {
		return nil, err
	}

	output := make([]root.RootID, 0, len(values))
	for _, value := range values {
		if value.RetiredAt != nil {
			continue
		}
		output = append(output, value.ID)
	}
	slices.Sort(output)
	return output, nil
}

func (w *ModelStoreWrapper) writableManagementRoot(
	ctx context.Context,
	requested root.RootID,
) (root.RootID, error) {
	roots, err := w.managementRootIDs(ctx, requested)
	if err != nil {
		return "", err
	}
	for _, rootID := range roots {
		if !w.protection.IsProtectedRoot(rootID) {
			return rootID, nil
		}
	}
	return "", fmt.Errorf(
		"%w: no writable Artifact Root is available for Model authoring",
		basespec.ErrReferenceUnresolved,
	)
}

func (w *ModelStoreWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
	w.management = nil
	w.roots = nil
	w.protection = nil
}
