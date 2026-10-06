package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/modelcatalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/modelcatalog/inferenceadapter"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	artifactcleanupFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/artifactcleanup"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	storeSecret "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	modelAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/overlay"
)

func initModelWrappers(
	ctx context.Context,
	storeWrapper *ModelStoreWrapper,
	aggregateWrapper *ModelAggregateWrapper,
	sources source.API,
	discovery refreshFlow.API,
	artifacts artifact.API,
	roots root.API,
	managedArtifacts managepackageFlow.API,
	protection root.ProtectionAPI,
	protectedOverlays overlay.API,
	secretBindings storeSecret.API,
	secretRuntime storeSecret.RuntimeAPI,
	localState artifactcleanupFlow.API,
	storeOverlays overlay.StoreAPI,
	cat catalog.API,
	definitions definition.API,
	hydrator installModel.CompiledHydrationCoordinator,
) (installFlow.HydrationInstaller, *inferenceadapter.RuntimeAdapter, error) {
	if storeWrapper == nil || aggregateWrapper == nil || roots == nil {
		return nil, nil, fmt.Errorf(
			"%w: Model wrapper dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	preferences, err := newArtifactModelDefaultProviderPreferences(storeOverlays)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	credentials, err := newArtifactModelCredentialResolver(secretRuntime)
	if err != nil {
		return nil, nil, err
	}
	runtimeAdapter, err := inferenceadapter.NewRuntimeAdapter(credentials)
	if err != nil {
		return nil, nil, err
	}

	api, err := modelAPI.New(modelAPI.Dependencies{
		Artifacts:   artifacts,
		Cat:         cat,
		Definitions: definitions,
		Sources:     sources,
		Discovery:   discovery,

		ManagedArtifacts: managedArtifacts,
		Protection:       protection,
		Overlays:         overlays,
		Adapters:         runtimeAdapter,
		BuiltinRoot:      topology.BuiltinRootID(),
	})
	if err != nil {
		return nil, nil, err
	}
	if err := runtimeAdapter.BindModelStore(api); err != nil {
		return nil, nil, err
	}

	cleanup, err := modelAPI.NewBuiltinPackageCleanup(api)
	if err != nil {
		return nil, nil, err
	}
	installer, err := modelcatalog.NewInstaller(
		modelcatalog.InstallerDependencies{
			Hydrator: hydrator,
			Cleanup:  cleanup,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	storeWrapper.api = api
	storeWrapper.roots = roots
	storeWrapper.protection = protection

	aggregateWrapper.store = api
	aggregateWrapper.runtime = runtimeAdapter
	aggregateWrapper.preferences = preferences
	aggregateWrapper.writableRoot = storeWrapper.writableManagementRoot
	aggregateWrapper.fallbackProvider = spec.LogicalName(
		inferenceadapter.ProviderNameOpenAIResponses,
	)

	return installer, runtimeAdapter, nil
}

// initModelProviderRuntime selects active management Roots after hydration.
// Provider filtering, collision handling, and conversion belong to the adapter.
func initModelProviderRuntime(
	ctx context.Context,
	store *ModelStoreWrapper,
	runtimeAdapter *inferenceadapter.RuntimeAdapter,
	publisher providerRuntimePublisher,
) error {
	if store == nil || store.api == nil ||
		runtimeAdapter == nil || publisher == nil {
		return spec.ErrClosed
	}

	roots, err := store.managementRootIDs(ctx, "")
	if err != nil {
		return err
	}

	var (
		providers []modelAPI.ProviderListItem
		result    error
	)
	for _, rootID := range roots {
		values, err := store.api.Providers.List(ctx, modelAPI.ListProvidersRequest{
			RootID: rootID,
		})
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
		runtimeAdapter.InitializeProviderRuntime(
			ctx,
			providers,
			publisher.PublishProvider,
		),
	)
}
