package main

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/artifactcleanup"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managedpackage"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	modelAggregate "github.com/flexigpt/flexigpt-app/internal/model/aggregate"
	"github.com/flexigpt/flexigpt-app/internal/model/inferenceadapter"
	modelBuiltin "github.com/flexigpt/flexigpt-app/internal/model/store/builtin"
	modelConsumerAPI "github.com/flexigpt/flexigpt-app/internal/model/store/consumerapi"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/model/store/overlay"
)

type ModelStoreWrapper struct {
	api        *modelConsumerAPI.API
	management *modelConsumerAPI.CatalogStore
	roots      root.API
	protection root.ProtectionAPI
}

func initModelWrappers(
	ctx context.Context,
	storeWrapper *ModelStoreWrapper,
	aggregateWrapper *ModelAggregateWrapper,
	sources source.API,
	discovery refresh.API,
	artifacts artifact.API,
	roots root.API,
	managedArtifacts managedpackage.API,
	protection root.ProtectionAPI,
	protectedOverlays overlay.API,
	secretBindings secret.API,
	secretRuntime secret.RuntimeAPI,
	localState artifactcleanup.API,
	storeOverlays overlay.StoreAPI,
	hydrator installModel.CompiledHydrationCoordinator,
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

func (w *ModelStoreWrapper) ListProviders(
	rootID rootModel.RootID,
) ([]modelConsumerAPI.ProviderListItem, error) {
	if w == nil || w.management == nil {
		return nil, spec.ErrClosed
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
	rootID rootModel.RootID,
) ([]modelConsumerAPI.ModelListItem, error) {
	if w == nil || w.management == nil {
		return nil, spec.ErrClosed
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

func (w *ModelStoreWrapper) GetProvider(
	ref artifactModel.ArtifactRef,
) (modelConsumerAPI.ProviderView, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ProviderView{}, spec.ErrClosed
	}
	return w.api.GetProvider(context.Background(), ref)
}

func (w *ModelStoreWrapper) GetModel(
	ref artifactModel.ArtifactRef,
) (modelConsumerAPI.ModelView, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ModelView{}, spec.ErrClosed
	}
	return w.api.GetModel(context.Background(), ref)
}

func (w *ModelStoreWrapper) SaveModelSettings(
	request modelConsumerAPI.SaveModelSettingsRequest,
) (modelConsumerAPI.ModelView, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ModelView{}, spec.ErrClosed
	}
	return w.api.SaveModelSettings(context.Background(), request)
}

func (w *ModelStoreWrapper) ResetModelSettings(
	ref artifactModel.ArtifactRef,
	expectedModelRevision uint64,
	expectedSettingsRevision uint64,
) (modelConsumerAPI.ModelView, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ModelView{}, spec.ErrClosed
	}
	return w.api.ResetModelSettings(
		context.Background(),
		ref,
		expectedModelRevision,
		expectedSettingsRevision,
	)
}

func (w *ModelStoreWrapper) GetProviderAPIKeyStatus(
	ref artifactModel.ArtifactRef,
) (modelConsumerAPI.ProviderAPIKeyStatus, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ProviderAPIKeyStatus{}, spec.ErrClosed
	}
	return w.api.GetProviderAPIKeyStatus(context.Background(), ref)
}

func (w *ModelStoreWrapper) CreateModel(
	request modelConsumerAPI.ManagedModelCreateRequest,
) (modelConsumerAPI.ManagedModelCreateResult, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ManagedModelCreateResult{}, spec.ErrClosed
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

func (w *ModelStoreWrapper) UpdateModel(
	request modelConsumerAPI.ManagedModelReplaceRequest,
) (modelConsumerAPI.ManagedModelReplaceResult, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ManagedModelReplaceResult{}, spec.ErrClosed
	}
	return w.api.ReplaceModel(context.Background(), request)
}

func (w *ModelStoreWrapper) DeleteModel(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	if w == nil || w.api == nil {
		return spec.ErrClosed
	}
	return w.api.DeleteModel(
		context.Background(),
		ref,
		expectedRevision,
	)
}

func (w *ModelStoreWrapper) SetModelEnabled(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifactModel.Artifact, error) {
	if w == nil || w.api == nil {
		return artifactModel.Artifact{}, spec.ErrClosed
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
	requested rootModel.RootID,
) ([]rootModel.RootID, error) {
	if w == nil || w.roots == nil || w.protection == nil {
		return nil, spec.ErrClosed
	}
	if requested != "" {
		if err := requested.Validate(); err != nil {
			return nil, err
		}
		return []rootModel.RootID{requested}, nil
	}

	values, err := w.roots.List(ctx)
	if err != nil {
		return nil, err
	}

	output := make([]rootModel.RootID, 0, len(values))
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
	requested rootModel.RootID,
) (rootModel.RootID, error) {
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
		spec.ErrReferenceUnresolved,
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
