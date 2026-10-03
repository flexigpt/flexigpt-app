package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/locator"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	artifactcleanupFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/artifactcleanup"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	managedpackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managedpackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	mcpAggregate "github.com/flexigpt/flexigpt-app/internal/mcp/aggregate"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/connection"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime/invocation"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime/sdkclient"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpBuiltin "github.com/flexigpt/flexigpt-app/internal/mcp/store/builtin"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/mcp/store/overlay"
)

const (
	mcpHostName    = "FlexiGPT"
	mcpHostVersion = "dev"
)

func initMCPWrappers(
	ctx context.Context,
	storeWrapper *MCPStoreWrapper,
	runtimeWrapper *MCPRuntimeWrapper,
	aggregateWrapper *MCPAggregateWrapper,
	roots root.API,
	sources source.API,
	discovery refreshFlow.API,
	artifacts artifact.API,
	cat catalog.API,
	definitions definition.API,
	resources resourceFlow.API,
	managedArtifacts managedpackageFlow.API,
	protection root.ProtectionAPI,
	protectedOverlays overlay.API,
	storeOverlays overlay.StoreAPI,
	secretBindings secret.API,
	secretRuntime secret.RuntimeAPI,
	localState artifactcleanupFlow.API,
	hydrator installModel.CompiledHydrationCoordinator,
	locatorResolvers []locator.Factory,
	fallbackProviders map[declaration.Type]resolve.FallbackProvider,
	targetMappers map[declaration.Type]resolve.ArtifactTargetMapper,
) (builtin.HydrationInstaller, error) {
	if storeWrapper == nil ||
		runtimeWrapper == nil ||
		aggregateWrapper == nil {
		return nil, errors.New("MCP wrapper receivers are incomplete")
	}
	if roots == nil ||
		sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		cat == nil || definitions == nil ||
		resources == nil ||
		managedArtifacts == nil ||
		protection == nil ||
		protectedOverlays == nil ||
		storeOverlays == nil ||
		secretBindings == nil ||
		secretRuntime == nil ||
		localState == nil ||
		hydrator == nil {
		return nil, errors.New("MCP wrapper dependencies are incomplete")
	}

	settings, err := newMCPSettingsAdapter(storeOverlays)
	if err != nil {
		return nil, err
	}
	overlays, err := mcpOverlay.NewArtifactOverlayRepository(
		mcpOverlay.ArtifactOverlayDependencies{
			Artifacts:        artifacts,
			Protection:       protection,
			ProtectedOverlay: protectedOverlays,
			LocalState:       localState,
		},
	)
	if err != nil {
		return nil, err
	}
	secrets, err := newArtifactMCPSecretResolver(
		artifacts,
		secretBindings,
		secretRuntime,
	)
	if err != nil {
		return nil, err
	}

	storeAPI, err := mcpConsumerAPI.New(
		sources,
		discovery,
		artifacts,
		resources,
		managedArtifacts,
		protection,
		cat,
		definitions,
		overlays,
		secrets,
		mcpPolicy.Baseline(),
		mcpConsumerAPI.WithLocatorResolvers(locatorResolvers),
		mcpConsumerAPI.WithFallbackProviders(fallbackProviders),
		mcpConsumerAPI.WithTargetMappers(targetMappers),
	)
	if err != nil {
		return nil, err
	}

	catalogStore, err := mcpConsumerAPI.NewCatalogStore(storeAPI)
	if err != nil {
		return nil, err
	}
	managementStore, err := mcpConsumerAPI.NewManagementStore(storeAPI)
	if err != nil {
		return nil, err
	}
	builtinCleanup, err := mcpConsumerAPI.NewBuiltinPackageCleanup(storeAPI)
	if err != nil {
		return nil, err
	}

	listService, err := mcpConsumerAPI.NewMCPListService(
		roots,
		catalogStore,
	)
	if err != nil {
		return nil, err
	}

	serverResolver, err := mcpAggregate.NewArtifactServerResolver(
		managementStore,
	)
	if err != nil {
		return nil, err
	}
	s, err := mcpAggregate.NewRuntimeServerSource(
		serverResolver,
		secrets,
		mcpEnvironmentResolver{},
	)
	if err != nil {
		return nil, err
	}

	global, _, err := settings.getMCPSettings(ctx)
	if err != nil {
		return nil, err
	}
	configuredLoopback := strings.TrimSpace(
		global.OAuthLoopbackListenAddr,
	)
	broker, err := mcpAuth.NewOAuthLoopbackBroker(
		ctx,
		&mcpAuth.OAuthLoopbackBrokerOptions{
			ListenAddr: configuredLoopback,
		},
	)
	if err != nil {
		return nil, err
	}

	var runtimeManager *mcpConnection.MCPRuntimeManager
	cleanup := func(
		cause error,
	) (builtin.HydrationInstaller, error) {
		if runtimeManager != nil {
			_ = runtimeManager.Close(context.Background())
		}
		_ = broker.Close()
		return nil, cause
	}

	tokenStore, err := mcpAggregate.NewOAuthTokenStore(secrets)
	if err != nil {
		return cleanup(err)
	}
	authManager := mcpAuth.NewAuthManager(
		secrets,
		mcpAuth.WithOAuthAuthorizationBroker(broker),
		mcpAuth.WithOAuthRedirectURL(broker.RedirectURL()),
		mcpAuth.WithOAuthTokenStore(tokenStore),
		mcpAuth.WithClientInfo(
			mcpHostName,
			mcpHostVersion,
		),
	)
	clientFactory, err := sdkclient.NewFactory(mcpServer.ClientInfo{
		Name:    mcpHostName,
		Version: mcpHostVersion,
	})
	if err != nil {
		return cleanup(err)
	}
	runtimeManager, err = mcpConnection.NewMCPRuntimeManager(
		s,
		authManager,
		clientFactory,
	)
	if err != nil {
		return cleanup(err)
	}

	toolBridge := invocation.NewToolBridge(
		runtimeManager,
		invocation.NewApprovalManager(5*time.Minute),
	)
	lifecycle, err := mcpAggregate.NewLifecycle(
		storeAPI,
		runtimeManager,
	)
	if err != nil {
		return cleanup(err)
	}
	service, err := mcpAggregate.NewService(mcpAggregate.Dependencies{
		Lifecycle: lifecycle,
		Servers:   serverResolver,
		Source:    s,
		Store:     managementStore,
		Runtime:   runtimeManager,
		Auth:      authManager,
		Secrets:   secrets,
	})
	if err != nil {
		return cleanup(err)
	}

	builtIns, err := newMCPBuiltInInstaller(
		hydrator,
		builtinCleanup,
	)
	if err != nil {
		return cleanup(err)
	}

	storeWrapper.api = storeAPI
	storeWrapper.management = listService
	storeWrapper.roots = roots
	storeWrapper.settings = settings

	runtimeWrapper.runtime = runtimeManager
	runtimeWrapper.toolBridge = toolBridge
	runtimeWrapper.auth = authManager
	runtimeWrapper.oauthBroker = broker

	aggregateWrapper.service = service
	return builtIns, nil
}

func newMCPBuiltInInstaller(
	hydrator installModel.CompiledHydrationCoordinator,
	cleanup mcpConsumerAPI.BuiltinPackageCleanup,
) (builtin.HydrationInstaller, error) {
	if hydrator == nil || cleanup == nil {
		return nil, errors.New(
			"MCP generated built-in installer dependencies are incomplete",
		)
	}
	return mcpBuiltin.NewInstaller(
		mcpBuiltin.InstallerDependencies{
			Hydrator: hydrator,
			Cleanup:  cleanup,
		},
	)
}
