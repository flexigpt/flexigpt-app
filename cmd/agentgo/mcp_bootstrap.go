package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	artifactAPI "github.com/flexigpt/flexigpt-app/internal/artifactstore/api"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	mcpAggregate "github.com/flexigpt/flexigpt-app/internal/mcp/aggregate"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/connection"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime/invocation"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime/sdkclient"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpStore "github.com/flexigpt/flexigpt-app/internal/mcp/store"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/mcp/store/overlay"
	mcpStorePolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/policy"
	mcpSchemaadapter "github.com/flexigpt/flexigpt-app/internal/mcp/store/schemaadapter"
	mcpStoreServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/server"
)

func InitMCPWrappers(
	ctx context.Context,
	storeWrapper *MCPStoreWrapper,
	runtimeWrapper *MCPRuntimeWrapper,
	aggregateWrapper *MCPAggregateWrapper,
	store artifactAPI.ConsumerAPI,
	settingsStore mcpAuthKeyStore,
) error {
	if storeWrapper == nil ||
		runtimeWrapper == nil ||
		aggregateWrapper == nil ||
		store == nil {
		return errors.New("MCP wrapper dependencies are incomplete")
	}

	settings, err := newMCPSettingsAdapter(settingsStore)
	if err != nil {
		return err
	}
	overlays, err := mcpOverlay.NewSettingsOverlayRepository(settings)
	if err != nil {
		return err
	}
	secrets := newSettingMCPSecretResolver(settingsStore)
	storeAPI, err := mcpStore.New(mcpStore.Dependencies{
		Store:          store,
		UserRootID:     artifactbuiltin.MCPUserRootID,
		Overlays:       overlays,
		SecretCleaner:  secrets,
		BaselinePolicy: mcpPolicy.Baseline(),
	})
	if err != nil {
		return err
	}
	if err := ensureDefaultMCPBundle(ctx, storeAPI); err != nil {
		return err
	}

	serverResolver, err := mcpAggregate.NewArtifactServerResolver(storeAPI)
	if err != nil {
		return err
	}
	source, err := mcpAggregate.NewRuntimeServerSource(
		serverResolver,
		secrets,
		mcpEnvironmentResolver{},
	)
	if err != nil {
		return err
	}

	global, _, err := settings.GetMCPGlobalSettings(ctx)
	if err != nil {
		return err
	}
	configuredLoopback := strings.TrimSpace(global.OAuthLoopbackListenAddr)
	broker, err := mcpAuth.NewOAuthLoopbackBroker(
		ctx,
		&mcpAuth.OAuthLoopbackBrokerOptions{
			ListenAddr: configuredLoopback,
		},
	)
	if err != nil {
		return err
	}

	var runtimeManager *mcpConnection.MCPRuntimeManager
	cleanup := func(cause error) error {
		if runtimeManager != nil {
			_ = runtimeManager.Close(context.Background())
		}
		_ = broker.Close()
		return cause
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

	builtIns, err := newMCPBuiltInInstaller(store, storeAPI, overlays)
	if err != nil {
		return cleanup(err)
	}

	storeWrapper.api = storeAPI
	storeWrapper.builtInInstaller = builtIns

	runtimeWrapper.runtime = runtimeManager
	runtimeWrapper.toolBridge = toolBridge
	runtimeWrapper.auth = authManager
	runtimeWrapper.settings = settings
	runtimeWrapper.oauthBroker = broker
	runtimeWrapper.oauthLoopbackListenAddrAtStart = configuredLoopback

	aggregateWrapper.service = service
	aggregateWrapper.serverResolver = serverResolver

	return nil
}

func newMCPBuiltInInstaller(
	documents providerapi.ExpectedCanonicalizer,
	bundles *mcpStore.API,
	overlays mcpOverlay.OverlayRepository,
) (artifactbuiltin.HydrationInstaller, error) {
	registry, packages, err := mcpSchemaadapter.LoadEmbeddedRegistry()
	if err != nil {
		return nil, err
	}

	return mcpSchemaadapter.NewInstaller(
		mcpSchemaadapter.InstallerDependencies{
			Bundles:            bundles,
			Registry:           registry,
			Packages:           packages,
			Overlays:           overlays,
			ShareableDocuments: documents,
		},
	)
}

func ensureDefaultMCPBundle(
	ctx context.Context,
	api *mcpStore.API,
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
	_, err = api.Create(ctx, mcpStore.CreateRequest{
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

func defaultMCPBundleDocument() mcpStore.BundleDocument {
	return mcpStore.BundleDocument{
		Kind:          artifactbuiltin.BundleKind,
		SchemaID:      artifactbuiltin.BundleSchemaID,
		SchemaVersion: artifactbuiltin.MCPSchemaVersion,
		LogicalName:   artifactbuiltin.DefaultMCPBundleLogicalName,
		DisplayName:   artifactbuiltin.DefaultMCPBundleDisplayName,
		Description:   artifactbuiltin.DefaultMCPBundleDescription,
		MCPServers:    map[string]mcpStoreServer.CoreServer{},
		BundleExtension: mcpStore.BundleExtension{
			Servers:  map[string]mcpStoreServer.ServerExtension{},
			Policies: map[string]mcpStorePolicy.PolicyDocument{},
		},
	}
}
