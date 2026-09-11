package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpbundlev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	mcpAggregate "github.com/flexigpt/flexigpt-app/internal/mcp/aggregate"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/connection"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime/invocation"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime/sdkclient"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpBuiltin "github.com/flexigpt/flexigpt-app/internal/mcp/store/builtin"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	mcpDomainBundle "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/bundle"
	mcpDomainPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/policy"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/mcp/store/overlay"
)

func InitMCPWrappers(
	ctx context.Context,
	storeWrapper *MCPStoreWrapper,
	runtimeWrapper *MCPRuntimeWrapper,
	aggregateWrapper *MCPAggregateWrapper,
	roots compositionapi.RootAPI,
	sources compositionapi.SourceAPI,
	collections compositionapi.CollectionAPI,
	artifacts compositionapi.ArtifactAPI,
	catalogs compositionapi.CatalogAPI,
	resources compositionapi.ResourceAPI,
	schemas compositionapi.SchemaAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	protection compositionapi.ProtectionAPI,
	userRootID root.RootID,
	settingsStore mcpAuthKeyStore,
) (artifactbuiltin.HydrationInstaller, error) {
	if storeWrapper == nil ||
		runtimeWrapper == nil ||
		aggregateWrapper == nil {
		return nil, errors.New("MCP wrapper receivers are incomplete")
	}
	if roots == nil {
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
		collections,
		artifacts,
		catalogs,
		resources,
		schemas,
		managedArtifacts,
		protection,
		userRootID,
		overlays,
		secrets,
		mcpPolicy.Baseline(),
	)
	if err != nil {
		return nil, err
	}

	if err := ensureDefaultMCPBundle(ctx, storeAPI); err != nil {
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
	configuredLoopback := strings.TrimSpace(global.OAuthLoopbackListenAddr)
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
	cleanup := func(cause error) (artifactbuiltin.HydrationInstaller, error) {
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
			artifactbuiltin.MCPHostName,
			artifactbuiltin.MCPHostVersion,
		),
	)

	clientFactory, err := sdkclient.NewFactory(mcpServer.ClientInfo{
		Name:    artifactbuiltin.MCPHostName,
		Version: artifactbuiltin.MCPHostVersion,
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
		Bundles:   storeAPI,
		Auth:      authManager,
		Secrets:   secrets,
	})
	if err != nil {
		return cleanup(err)
	}

	builtIns, err := NewMCPBuiltInInstaller(
		schemas,
		storeAPI,
		overlays,
	)
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
	documents providerapi.ExpectedCanonicalizer,
	store mcpConsumerAPI.BuiltinStore,
	overlays mcpOverlay.OverlayRepository,
) (artifactbuiltin.HydrationInstaller, error) {
	registry, packages, err := mcpBuiltin.LoadEmbeddedRegistry()
	if err != nil {
		return nil, err
	}

	return mcpBuiltin.NewInstaller(
		mcpBuiltin.InstallerDependencies{
			Bundles:            store,
			Registry:           registry,
			Packages:           packages,
			Overlays:           overlays,
			ShareableDocuments: documents,
		},
	)
}

func ensureDefaultMCPBundle(
	ctx context.Context,
	api *mcpConsumerAPI.API,
) error {
	if ctx == nil {
		return fmt.Errorf("%w: default MCP Bundle context is nil", basespec.ErrInvalid)
	}
	if api == nil {
		return fmt.Errorf("%w: default MCP Bundle API is unavailable", basespec.ErrClosed)
	}

	ref := collection.CollectionRef{
		RootID:       artifactbuiltin.MCPUserRootID,
		CollectionID: artifactbuiltin.DefaultMCPBundleCollectionID,
	}
	existing, err := api.Get(ctx, ref)
	switch {
	case err == nil:
		if existing.Data.ManagedSourceID != artifactbuiltin.DefaultMCPBundleSourceID ||
			existing.Source.ID != artifactbuiltin.DefaultMCPBundleSourceID {
			return fmt.Errorf(
				"%w: default MCP Bundle identity conflicts with existing state",
				basespec.ErrConflict,
			)
		}
		return nil

	case !errors.Is(err, basespec.ErrCollectionNotFound):
		return fmt.Errorf("read default MCP Bundle: %w", err)
	}

	raw, err := json.Marshal(defaultMCPBundleDocument())
	if err != nil {
		return fmt.Errorf("encode default MCP Bundle document: %w", err)
	}
	_, err = api.Create(ctx, mcpConsumerAPI.CreateMCPBundleBody{
		RootID:           artifactbuiltin.MCPUserRootID,
		CollectionID:     artifactbuiltin.DefaultMCPBundleCollectionID,
		SourceID:         artifactbuiltin.DefaultMCPBundleSourceID,
		SourceStorageKey: artifactbuiltin.DefaultMCPBundleSourceKey,
		Document:         raw,
	})
	if err != nil {
		return fmt.Errorf("create default MCP Bundle: %w", err)
	}
	return nil
}

func defaultMCPBundleDocument() mcpDomainBundle.BundleDocument {
	return mcpDomainBundle.BundleDocument{
		Kind:          mcpbundlev1.MCPBundleKind,
		SchemaID:      mcpbundlev1.MCPBundleSchemaID,
		SchemaVersion: artifactbuiltin.MCPSchemaVersion,
		LogicalName:   artifactbuiltin.DefaultMCPBundleLogicalName,
		DisplayName:   artifactbuiltin.DefaultMCPBundleDisplayName,
		Description:   artifactbuiltin.DefaultMCPBundleDescription,
		MCPServers:    map[string]mcpDomainServer.CoreServer{},
		BundleExtension: mcpDomainBundle.BundleExtension{
			Servers:  map[string]mcpDomainServer.ServerExtension{},
			Policies: map[string]mcpDomainPolicy.PolicyDocument{},
		},
	}
}
