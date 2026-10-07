package main

import (
	"context"
	"errors"
	"time"

	mcpAuth "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/auth"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/connection"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/invocation"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/sdkclient"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/mcpcatalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	artifactcleanupFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/artifactcleanup"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	storeSecret "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/llmsupport"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/mcpruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/mcpsecrets"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/mcpsettings"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	mcpAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/overlay"
	"github.com/flexigpt/flexigpt-app/internal/mcppolicy"
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
	managedArtifacts managepackageFlow.API,
	protection root.ProtectionAPI,
	protectedOverlays overlay.API,
	storeOverlays overlay.StoreAPI,
	secretBindings storeSecret.API,
	secretRuntime storeSecret.RuntimeAPI,
	localState artifactcleanupFlow.API,
	hydrator installModel.CompiledHydrationCoordinator,
	resolver *composition.Resolver,
) (installFlow.HydrationInstaller, *mcpruntime.RuntimeAdapter, error) {
	if storeWrapper == nil ||
		runtimeWrapper == nil ||
		aggregateWrapper == nil {
		return nil, nil, errors.New("MCP wrapper receivers are incomplete")
	}

	support, err := llmsupport.MCP()
	if err != nil {
		return nil, nil, err
	}

	// Each constructor validates the dependencies it actually owns.
	settings, err := mcpsettings.New(storeOverlays)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	secrets, err := mcpsecrets.New(
		artifacts,
		secretBindings,
		secretRuntime,
	)
	if err != nil {
		return nil, nil, err
	}

	storeAPI, err := mcpAPI.New(
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
		mcppolicy.Baseline(),
		mcpAPI.WithCompositionResolver(resolver),
		mcpAPI.WithSupport(support),
	)
	if err != nil {
		return nil, nil, err
	}

	listService, err := mcpAPI.NewMCPListService(roots, storeAPI)
	if err != nil {
		return nil, nil, err
	}
	builtinCleanup, err := mcpAPI.NewBuiltinPackageCleanup(storeAPI)
	if err != nil {
		return nil, nil, err
	}
	builtIns, err := mcpcatalog.NewInstaller(
		mcpcatalog.InstallerDependencies{
			Hydrator: hydrator,
			Cleanup:  builtinCleanup,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	adapter, err := mcpruntime.NewRuntimeAdapter(
		storeAPI,
		secrets,
		mcpsettings.EnvironmentResolver{},
	)
	if err != nil {
		return nil, nil, err
	}

	global, _, err := settings.Get(ctx)
	if err != nil {
		return nil, nil, err
	}
	broker, err := mcpAuth.NewOAuthLoopbackBroker(
		ctx,
		&mcpAuth.OAuthLoopbackBrokerOptions{
			ListenAddr: global.OAuthLoopbackListenAddr,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	var runtimeManager *mcpConnection.MCPRuntimeManager
	cleanup := func(
		cause error,
	) (installFlow.HydrationInstaller, *mcpruntime.RuntimeAdapter, error) {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if runtimeManager != nil {
			cause = errors.Join(cause, runtimeManager.Close(cleanupCtx))
		}
		cause = errors.Join(cause, broker.Close())
		return nil, nil, cause
	}

	authManager := mcpAuth.NewAuthManager(
		secrets,
		mcpAuth.WithOAuthAuthorizationBroker(broker),
		mcpAuth.WithOAuthRedirectURL(broker.RedirectURL()),
		mcpAuth.WithOAuthTokenStore(adapter),
		mcpAuth.WithClientInfo(mcpHostName, mcpHostVersion),
	)
	clientFactory, err := sdkclient.NewFactory(mcpServer.ClientInfo{
		Name:    mcpHostName,
		Version: mcpHostVersion,
	})
	if err != nil {
		return cleanup(err)
	}
	runtimeManager, err = mcpConnection.NewMCPRuntimeManager(
		adapter,
		authManager,
		clientFactory,
	)
	if err != nil {
		return cleanup(err)
	}

	// Management and completion share one approval/invocation owner.
	toolBridge := invocation.NewToolBridge(
		runtimeManager,
		invocation.NewApprovalManager(5*time.Minute),
	)
	if err := adapter.BindRuntime(runtimeManager); err != nil {
		return cleanup(err)
	}

	// Publish only after the entire construction and binding phase succeeds.
	storeWrapper.api = storeAPI
	storeWrapper.management = listService
	storeWrapper.roots = roots
	storeWrapper.settings = settings

	runtimeWrapper.runtime = runtimeManager
	runtimeWrapper.toolBridge = toolBridge
	runtimeWrapper.auth = authManager
	runtimeWrapper.oauthBroker = broker

	aggregateWrapper.store = storeAPI
	aggregateWrapper.source = adapter
	aggregateWrapper.runtime = runtimeManager
	aggregateWrapper.auth = authManager
	aggregateWrapper.secrets = secrets

	return builtIns, adapter, nil
}
