package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
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

func InitMCPWrappers(
	ctx context.Context,
	storeWrapper *MCPStoreWrapper,
	runtimeWrapper *MCPRuntimeWrapper,
	aggregateWrapper *MCPAggregateWrapper,
	roots compositionapi.RootAPI,
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	protection compositionapi.ProtectionAPI,
	settingsStore mcpAuthKeyStore,
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
		resources == nil ||
		managedArtifacts == nil ||
		protection == nil ||
		settingsStore == nil {
		return nil, errors.New("MCP wrapper dependencies are incomplete")
	}

	settings, err := newMCPSettingsAdapter(settingsStore)
	if err != nil {
		return nil, err
	}
	overlays, err := mcpOverlay.NewSettingsOverlayRepository(settings)
	if err != nil {
		return nil, err
	}
	secrets := newSettingMCPSecretResolver(settingsStore)

	storeAPI, err := mcpConsumerAPI.New(
		sources,
		discovery,
		artifacts,
		resources,
		managedArtifacts,
		protection,
		overlays,
		secrets,
		mcpPolicy.Baseline(),
	)
	if err != nil {
		return nil, err
	}

	serverResolver, err := mcpAggregate.NewArtifactServerResolver(storeAPI)
	if err != nil {
		return nil, err
	}
	source, err := mcpAggregate.NewRuntimeServerSource(
		serverResolver,
		secrets,
		mcpEnvironmentResolver{},
	)
	if err != nil {
		return nil, err
	}

	global, _, err := settings.GetMCPGlobalSettings(ctx)
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
			builtin.MCPHostName,
			builtin.MCPHostVersion,
		),
	)
	clientFactory, err := sdkclient.NewFactory(mcpServer.ClientInfo{
		Name:    builtin.MCPHostName,
		Version: builtin.MCPHostVersion,
	})
	if err != nil {
		return cleanup(err)
	}
	runtimeManager, err = mcpConnection.NewMCPRuntimeManager(
		source,
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
	lifecycle, err := mcpAggregate.NewLifecycle(storeAPI, runtimeManager)
	if err != nil {
		return cleanup(err)
	}
	service, err := mcpAggregate.NewService(mcpAggregate.Dependencies{
		Lifecycle: lifecycle,
		Servers:   serverResolver,
		Source:    source,
		Store:     storeAPI,
		Auth:      authManager,
		Secrets:   secrets,
	})
	if err != nil {
		return cleanup(err)
	}

	builtIns, err := NewMCPBuiltInInstaller(storeAPI, overlays)
	if err != nil {
		return cleanup(err)
	}

	storeWrapper.api = storeAPI
	storeWrapper.roots = roots

	runtimeWrapper.runtime = runtimeManager
	runtimeWrapper.toolBridge = toolBridge
	runtimeWrapper.auth = authManager
	runtimeWrapper.settings = settings
	runtimeWrapper.oauthBroker = broker
	runtimeWrapper.oauthLoopbackListenAddrAtStart = configuredLoopback

	aggregateWrapper.service = service
	aggregateWrapper.serverResolver = serverResolver
	return builtIns, nil
}

func NewMCPBuiltInInstaller(
	store mcpConsumerAPI.BuiltinStore,
	overlays mcpOverlay.RootPurger,
) (builtin.HydrationInstaller, error) {
	if store == nil {
		return nil, errors.New("MCP built-in Store is required")
	}
	packages, err := builtin.EmbeddedMCPPackages()
	if err != nil {
		return nil, err
	}
	return mcpBuiltin.NewInstaller(
		mcpBuiltin.InstallerDependencies{
			MCP:      store,
			Packages: packages,
			Overlays: overlays,
		},
	)
}
