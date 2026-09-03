package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	mcpAggregate "github.com/flexigpt/flexigpt-app/internal/mcp/aggregate"
	mcpRuntime "github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime/sdkclient"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/spec"
	mcpStore "github.com/flexigpt/flexigpt-app/internal/mcp/store"
	mcpArtifact "github.com/flexigpt/flexigpt-app/internal/mcp/store/artifact"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/mcp/store/overlay"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/policy"
	mcpSchemaadapter "github.com/flexigpt/flexigpt-app/internal/mcp/store/schemaadapter"
	mcpSecret "github.com/flexigpt/flexigpt-app/internal/mcp/store/secret"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

type MCPWrapper struct {
	storeAPI  *mcpStore.API
	aggregate *mcpAggregate.Service
	artifacts *artifact.Service
	runtime   *mcpRuntime.MCPRuntimeManager
	auth      *mcpAuth.AuthManager

	overlays *mcpOverlay.SettingsOverlayRepository
	settings *mcpSettingsAdapter
	secrets  *settingMCPSecretResolver

	oauthLoopbackListenAddrAtStart string
	oauthBroker                    *mcpAuth.OAuthLoopbackBroker
	builtInInstaller               artifactbuiltin.HydrationInstaller
}

type MCPSecretWriteResult struct {
	SecretRef string `json:"secretRef"`
	SHA256    string `json:"sha256,omitempty"`
	NonEmpty  bool   `json:"nonEmpty"`
}

type MCPGlobalSettingsView struct {
	Settings                mcpAuth.MCPAuthSettings `json:"settings"`
	Revision                uint64                  `json:"revision"`
	OAuthRedirectURL        string                  `json:"oauthRedirectURL,omitempty"`
	OAuthLoopbackListenAddr string                  `json:"oauthLoopbackListenAddr,omitempty"`
	OAuthRestartRequired    bool                    `json:"oauthRestartRequired"`
	OAuthLoopbackReady      bool                    `json:"oauthLoopbackReady"`
	OAuthLoopbackError      string                  `json:"oauthLoopbackError,omitempty"`
}

func InitMCPWrapper(
	ctx context.Context,
	wrapper *MCPWrapper,
	components *system.Components,
	settingsStore mcpAuthKeyStore,
) error {
	if wrapper == nil || components == nil {
		return errors.New("MCP wrapper dependencies are incomplete")
	}
	if _, err := components.Roots.Get(ctx, artifactbuiltin.MCPUserRootID); err != nil {
		return fmt.Errorf("ensure retained MCP user Root: %w", err)
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

	global, _, err := settings.GetMCPGlobalSettings(ctx)
	if err != nil {
		return err
	}

	configuredLoopback := strings.TrimSpace(global.OAuthLoopbackListenAddr)
	brokerOptions := &mcpAuth.OAuthLoopbackBrokerOptions{
		ListenAddr: configuredLoopback,
	}
	oauthBroker, err := mcpAuth.NewOAuthLoopbackBroker(ctx, brokerOptions)
	if err != nil {
		return err
	}

	authManager := mcpAuth.NewAuthManager(
		secrets,
		mcpAuth.WithOAuthAuthorizationBroker(oauthBroker),
		mcpAuth.WithOAuthRedirectURL(oauthBroker.RedirectURL()),
		mcpAuth.WithOAuthTokenStore(secrets),
		mcpAuth.WithClientInfo(
			artifactbuiltin.MCPHostName,
			artifactbuiltin.MCPHostVersion,
		),
	)

	storeAPI, err := mcpStore.New(mcpStore.Dependencies{
		Sources:            components.Sources,
		Collections:        components.Collections,
		Artifacts:          components.Artifacts,
		ManagedArtifacts:   components.ManagedArtifacts,
		Refresh:            components.Refresh,
		Catalogs:           components.Catalogs,
		SourceRuntime:      components.SourceRuntime,
		ShareableDocuments: components.ShareableSchemas,
		HasDecoder:         components.HasDecoder,
		DecoderFingerprint: components.DecoderFingerprint,
		RootPolicy:         components.RootMutationPolicy(),
		UserRootID:         artifactbuiltin.MCPUserRootID,
		Overlays:           overlays,
		SecretCleaner:      secrets,
		BaselinePolicy:     mcpPolicy.Baseline(),
	})
	if err != nil {
		_ = oauthBroker.Close()
		return err
	}

	if err := ensureDefaultMCPBundle(ctx, storeAPI); err != nil {
		_ = oauthBroker.Close()
		return err
	}

	source, err := mcpAggregate.NewServerSource(
		storeAPI,
		secrets,
		mcpEnvironmentResolver{},
	)
	if err != nil {
		_ = oauthBroker.Close()
		return err
	}
	clientFactory, err := sdkclient.NewFactory(mcpSpec.ClientInfo{
		Name:    artifactbuiltin.MCPHostName,
		Version: artifactbuiltin.MCPHostVersion,
	})
	if err != nil {
		_ = oauthBroker.Close()
		return err
	}
	rt, err := mcpRuntime.NewMCPRuntimeManager(
		source,
		authManager,
		clientFactory,
	)
	if err != nil {
		_ = oauthBroker.Close()
		return err
	}

	toolBridge := mcpRuntime.NewToolBridge(
		rt,
		mcpRuntime.NewApprovalManager(5*time.Minute),
	)
	aggregateService, err := mcpAggregate.New(
		storeAPI,
		source,
		rt,
		toolBridge,
	)
	if err != nil {
		_ = rt.Close(context.Background())
		_ = oauthBroker.Close()
		return err
	}

	builtIns, err := newMCPBuiltInInstaller(
		components,
		storeAPI,
		overlays,
	)
	if err != nil {
		_ = rt.Close(context.Background())
		_ = oauthBroker.Close()
		return err
	}

	wrapper.storeAPI = storeAPI
	wrapper.aggregate = aggregateService
	wrapper.artifacts = components.Artifacts
	wrapper.runtime = rt
	wrapper.auth = authManager
	wrapper.overlays = overlays
	wrapper.settings = settings
	wrapper.secrets = secrets
	wrapper.oauthLoopbackListenAddrAtStart = configuredLoopback
	wrapper.oauthBroker = oauthBroker
	wrapper.builtInInstaller = builtIns
	return nil
}

func newMCPBuiltInInstaller(
	components *system.Components,
	bundles *mcpStore.API,
	overlays mcpOverlay.OverlayRepository,
) (artifactbuiltin.HydrationInstaller, error) {
	registry, packages, err := mcpSchemaadapter.LoadEmbeddedRegistry()
	if err != nil {
		return nil, err
	}

	installer, err := mcpSchemaadapter.NewInstaller(
		mcpSchemaadapter.InstallerDependencies{
			Bundles:            bundles,
			Registry:           registry,
			Packages:           packages,
			Overlays:           overlays,
			ShareableDocuments: components.ShareableSchemas,
		},
	)
	if err != nil {
		return nil, err
	}
	return installer, nil
}

func (w *MCPWrapper) CreateMCPBundle(
	request *mcpStore.CreateRequest,
) (mcpStore.Bundle, error) {
	return middleware.WithRecoveryResp(func() (mcpStore.Bundle, error) {
		if err := w.ready(); err != nil {
			return mcpStore.Bundle{}, err
		}
		if request == nil {
			return mcpStore.Bundle{}, errors.New("MCP Bundle create request is required")
		}
		return w.storeAPI.Create(context.Background(), *request)
	})
}

func (w *MCPWrapper) GetMCPBundle(
	ref collection.CollectionRef,
) (mcpStore.Bundle, error) {
	return middleware.WithRecoveryResp(func() (mcpStore.Bundle, error) {
		if err := w.ready(); err != nil {
			return mcpStore.Bundle{}, err
		}
		return w.storeAPI.Get(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPBundles(
	rootID basespec.RootID,
) ([]mcpStore.Bundle, error) {
	return middleware.WithRecoveryResp(func() ([]mcpStore.Bundle, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.storeAPI.List(context.Background(), rootID)
	})
}

func (w *MCPWrapper) GetMCPBundleDocument(
	ref collection.CollectionRef,
) (mcpStore.BundleDocument, error) {
	return middleware.WithRecoveryResp(func() (mcpStore.BundleDocument, error) {
		if err := w.ready(); err != nil {
			return mcpStore.BundleDocument{}, err
		}
		return w.storeAPI.GetDocument(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPBundleServers(
	ref collection.CollectionRef,
) ([]artifact.Artifact, error) {
	return middleware.WithRecoveryResp(func() ([]artifact.Artifact, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.storeAPI.ListServers(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPBundlePolicies(
	ref collection.CollectionRef,
) ([]artifact.Artifact, error) {
	return middleware.WithRecoveryResp(func() ([]artifact.Artifact, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.storeAPI.ListPolicies(context.Background(), ref)
	})
}

func (w *MCPWrapper) GetMCPServerInstallation(
	ref artifact.ArtifactRef,
) (mcpStore.ServerInstallationView, error) {
	return middleware.WithRecoveryResp(
		func() (mcpStore.ServerInstallationView, error) {
			if err := w.ready(); err != nil {
				return mcpStore.ServerInstallationView{}, err
			}
			return w.storeAPI.GetServerInstallation(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *MCPWrapper) InspectMCPServer(
	ref artifact.ArtifactRef,
) (mcpArtifact.Resolved, error) {
	return middleware.WithRecoveryResp(func() (mcpArtifact.Resolved, error) {
		if err := w.ready(); err != nil {
			return mcpArtifact.Resolved{}, err
		}
		return w.storeAPI.InspectMCPServer(context.Background(), ref)
	})
}

func (w *MCPWrapper) InspectMCPPolicy(
	ref artifact.ArtifactRef,
) (mcpStore.PolicyView, error) {
	return middleware.WithRecoveryResp(func() (mcpStore.PolicyView, error) {
		if err := w.ready(); err != nil {
			return mcpStore.PolicyView{}, err
		}
		return w.storeAPI.InspectMCPPolicy(context.Background(), ref)
	})
}

func (w *MCPWrapper) GetMCPBundleInstallation(
	ref collection.CollectionRef,
) (mcpStore.BundleInstallationView, error) {
	return middleware.WithRecoveryResp(
		func() (mcpStore.BundleInstallationView, error) {
			if err := w.ready(); err != nil {
				return mcpStore.BundleInstallationView{}, err
			}
			return w.storeAPI.GetBundleInstallation(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *MCPWrapper) ReplaceMCPBundleDocument(
	request *mcpStore.ReplaceDocumentRequest,
) (mcpStore.Bundle, error) {
	return middleware.WithRecoveryResp(func() (mcpStore.Bundle, error) {
		if err := w.ready(); err != nil {
			return mcpStore.Bundle{}, err
		}
		if request == nil {
			return mcpStore.Bundle{}, errors.New("MCP document replacement request is required")
		}

		before, _ := w.artifacts.ListByCollection(
			context.Background(),
			request.Bundle,
		)
		result, err := w.aggregate.ReplaceDocument(
			context.Background(),
			*request,
		)
		if err != nil {
			return mcpStore.Bundle{}, err
		}
		for _, record := range before {
			serverID, idErr := mcpAggregate.ServerIDForArtifact(record.Ref())
			if idErr != nil {
				return mcpStore.Bundle{}, idErr
			}
			w.auth.ClearAuthStatus(serverID)
		}
		return result, nil
	})
}

func (w *MCPWrapper) RefreshMCPBundle(
	ref collection.CollectionRef,
) (mcpStore.Bundle, error) {
	return middleware.WithRecoveryResp(func() (mcpStore.Bundle, error) {
		if err := w.ready(); err != nil {
			return mcpStore.Bundle{}, err
		}
		return w.aggregate.RefreshBundle(context.Background(), ref, false)
	})
}

func (w *MCPWrapper) UpdateMCPBundleEnabled(
	ref collection.CollectionRef,
	expectedRevision uint64,
	enabled bool,
) (mcpStore.Bundle, error) {
	return middleware.WithRecoveryResp(func() (mcpStore.Bundle, error) {
		if err := w.ready(); err != nil {
			return mcpStore.Bundle{}, err
		}
		return w.aggregate.UpdateBundleEnabled(
			context.Background(),
			ref,
			expectedRevision,
			enabled,
		)
	})
}

func (w *MCPWrapper) RetireMCPBundle(
	ref collection.CollectionRef,
	expectedRevision uint64,
) (collection.Collection, error) {
	return middleware.WithRecoveryResp(func() (collection.Collection, error) {
		if err := w.ready(); err != nil {
			return collection.Collection{}, err
		}
		return w.aggregate.RetireBundle(
			context.Background(),
			ref,
			expectedRevision,
		)
	})
}

func (w *MCPWrapper) PurgeMCPBundle(
	ref collection.CollectionRef,
	expectedRevision uint64,
) error {
	return middleware.WithRecovery(func() error {
		if err := w.ready(); err != nil {
			return err
		}
		return w.aggregate.PurgeBundle(
			context.Background(),
			ref,
			expectedRevision,
		)
	})
}

func (w *MCPWrapper) UpdateMCPServerInstallation(
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	data mcpArtifact.ServerData,
) (artifact.Artifact, error) {
	return middleware.WithRecoveryResp(func() (artifact.Artifact, error) {
		if err := w.ready(); err != nil {
			return artifact.Artifact{}, err
		}
		value, err := w.aggregate.UpdateServerInstallation(
			context.Background(),
			ref,
			expectedArtifactRevision,
			data,
		)
		if err == nil {
			serverID, idErr := mcpAggregate.ServerIDForArtifact(ref)
			if idErr != nil {
				return artifact.Artifact{}, idErr
			}

			w.auth.ClearAuthStatus(serverID)
		}
		return value, err
	})
}

func (w *MCPWrapper) UpdateProtectedMCPServerInstallation(
	ref artifact.ArtifactRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
	data mcpArtifact.ServerData,
) error {
	return middleware.WithRecovery(func() error {
		if err := w.ready(); err != nil {
			return err
		}
		if err := w.aggregate.UpdateProtectedServerInstallation(
			context.Background(),
			ref,
			expectedOverlayRevision,
			runtimeEnabled,
			data,
		); err != nil {
			return err
		}
		serverID, idErr := mcpAggregate.ServerIDForArtifact(ref)
		if idErr != nil {
			return idErr
		}

		w.auth.ClearAuthStatus(serverID)

		return nil
	})
}

func (w *MCPWrapper) UpdateProtectedMCPBundleInstallation(
	ref collection.CollectionRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
) error {
	return middleware.WithRecovery(func() error {
		if err := w.ready(); err != nil {
			return err
		}
		return w.aggregate.UpdateProtectedBundleInstallation(
			context.Background(),
			ref,
			expectedOverlayRevision,
			runtimeEnabled,
		)
	})
}

func (w *MCPWrapper) ConnectMCPServer(
	ref artifact.ArtifactRef,
) (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
	return middleware.WithRecoveryResp(func() (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		serverID, idErr := mcpAggregate.ServerIDForArtifact(ref)
		if idErr != nil {
			return nil, idErr
		}

		return w.runtime.StartConnect(context.Background(), serverID)
	})
}

func (w *MCPWrapper) DisconnectMCPServer(
	ref artifact.ArtifactRef,
) error {
	return middleware.WithRecovery(func() error {
		if err := w.ready(); err != nil {
			return err
		}
		serverID, idErr := mcpAggregate.ServerIDForArtifact(ref)
		if idErr != nil {
			return idErr
		}

		return w.runtime.Disconnect(context.Background(), serverID)
	})
}

func (w *MCPWrapper) RefreshMCPServer(
	ref artifact.ArtifactRef,
) (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
	return middleware.WithRecoveryResp(func() (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		serverID, idErr := mcpAggregate.ServerIDForArtifact(ref)
		if idErr != nil {
			return nil, idErr
		}

		return w.runtime.Refresh(context.Background(), serverID)
	})
}

func (w *MCPWrapper) GetMCPServerStatus(
	ref artifact.ArtifactRef,
) (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
	return middleware.WithRecoveryResp(func() (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		serverID, idErr := mcpAggregate.ServerIDForArtifact(ref)
		if idErr != nil {
			return nil, idErr
		}
		return w.runtime.Status(context.Background(), serverID)
	})
}

func (w *MCPWrapper) ListMCPServerTools(
	ref artifact.ArtifactRef,
) ([]mcpRuntime.MCPToolCapability, error) {
	return middleware.WithRecoveryResp(func() ([]mcpRuntime.MCPToolCapability, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		serverID, idErr := mcpAggregate.ServerIDForArtifact(ref)
		if idErr != nil {
			return nil, idErr
		}
		return w.runtime.ListTools(context.Background(), serverID)
	})
}

func (w *MCPWrapper) ListMCPServerResources(
	ref artifact.ArtifactRef,
) ([]mcpRuntime.MCPResourceRef, error) {
	return middleware.WithRecoveryResp(func() ([]mcpRuntime.MCPResourceRef, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		serverID, idErr := mcpAggregate.ServerIDForArtifact(ref)
		if idErr != nil {
			return nil, idErr
		}
		return w.runtime.ListResources(context.Background(), serverID)
	})
}

func (w *MCPWrapper) ListMCPServerResourceTemplates(
	ref artifact.ArtifactRef,
) ([]mcpRuntime.MCPResourceTemplateRef, error) {
	return middleware.WithRecoveryResp(func() ([]mcpRuntime.MCPResourceTemplateRef, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		serverID, idErr := mcpAggregate.ServerIDForArtifact(ref)
		if idErr != nil {
			return nil, idErr
		}
		return w.runtime.ListResourceTemplates(context.Background(), serverID)
	})
}

func (w *MCPWrapper) ListMCPServerPrompts(
	ref artifact.ArtifactRef,
) ([]mcpRuntime.MCPPromptRef, error) {
	return middleware.WithRecoveryResp(func() ([]mcpRuntime.MCPPromptRef, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		serverID, idErr := mcpAggregate.ServerIDForArtifact(ref)
		if idErr != nil {
			return nil, idErr
		}
		return w.runtime.ListPrompts(context.Background(), serverID)
	})
}

func (w *MCPWrapper) ReadMCPResource(
	srv artifact.ArtifactRef,
	uri string,
) (*mcpRuntime.MCPReadResourceResponseBody, error) {
	return middleware.WithRecoveryResp(func() (*mcpRuntime.MCPReadResourceResponseBody, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		serverID, idErr := mcpAggregate.ServerIDForArtifact(srv)
		if idErr != nil {
			return nil, idErr
		}
		return w.runtime.ReadResource(context.Background(), serverID, uri)
	})
}

func (w *MCPWrapper) GetMCPPrompt(
	srv artifact.ArtifactRef,
	name string,
	arguments map[string]string,
) (*mcpRuntime.MCPGetPromptResponseBody, error) {
	return middleware.WithRecoveryResp(func() (*mcpRuntime.MCPGetPromptResponseBody, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		serverID, idErr := mcpAggregate.ServerIDForArtifact(srv)
		if idErr != nil {
			return nil, idErr
		}
		return w.runtime.GetPrompt(
			context.Background(),
			serverID,
			name,
			arguments,
		)
	})
}

func (w *MCPWrapper) CompleteMCPArgument(
	srv artifact.ArtifactRef,
	request mcpRuntime.MCPCompleteArgumentRequestBody,
) (*mcpRuntime.MCPCompletionResult, error) {
	return middleware.WithRecoveryResp(func() (*mcpRuntime.MCPCompletionResult, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		serverID, idErr := mcpAggregate.ServerIDForArtifact(srv)
		if idErr != nil {
			return nil, idErr
		}
		return w.runtime.Complete(context.Background(), serverID, request)
	})
}

func (w *MCPWrapper) EvaluateMCPToolCall(
	srv artifact.ArtifactRef,
	request *mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.MCPApprovalEvaluation, error) {
	return middleware.WithRecoveryResp(func() (*mcpRuntime.MCPApprovalEvaluation, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		if request == nil {
			return nil, fmt.Errorf("%w: MCP tool request is required", mcpRuntime.ErrMCPInvalidRuntimeRequest)
		}
		return w.aggregate.Evaluate(context.Background(), srv, *request)
	})
}

func (w *MCPWrapper) EvaluateMappedMCPToolCall(
	mapping mcpRuntime.MCPProviderToolMapping,
	request *mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.MCPApprovalEvaluation, error) {
	return middleware.WithRecoveryResp(func() (*mcpRuntime.MCPApprovalEvaluation, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		if request == nil {
			return nil, fmt.Errorf(
				"%w: mapped MCP tool request is required",
				mcpRuntime.ErrMCPInvalidRuntimeRequest,
			)
		}
		return w.aggregate.EvaluateMapped(
			context.Background(),
			mapping,
			*request,
		)
	})
}

func (w *MCPWrapper) ResolveMCPApproval(
	approvalID string,
	resolution mcpRuntime.MCPApprovalResolution,
) (mcpRuntime.MCPApprovalResolutionResult, error) {
	return middleware.WithRecoveryResp(func() (mcpRuntime.MCPApprovalResolutionResult, error) {
		if err := w.ready(); err != nil {
			return mcpRuntime.MCPApprovalResolutionResult{}, err
		}
		return w.aggregate.ResolveApproval(
			context.Background(),
			approvalID,
			resolution,
		)
	})
}

func (w *MCPWrapper) InvokeMCPTool(
	srv artifact.ArtifactRef,
	request *mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.InvokeMCPToolResponseBody, error) {
	return middleware.WithRecoveryResp(func() (*mcpRuntime.InvokeMCPToolResponseBody, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		if request == nil {
			return nil, fmt.Errorf("%w: MCP tool request is required", mcpRuntime.ErrMCPInvalidRuntimeRequest)
		}
		return w.aggregate.Invoke(context.Background(), srv, *request)
	})
}

func (w *MCPWrapper) InvokeMappedMCPTool(
	mapping mcpRuntime.MCPProviderToolMapping,
	request *mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.InvokeMCPToolResponseBody, error) {
	return middleware.WithRecoveryResp(func() (*mcpRuntime.InvokeMCPToolResponseBody, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		if request == nil {
			return nil, fmt.Errorf(
				"%w: mapped MCP tool request is required",
				mcpRuntime.ErrMCPInvalidRuntimeRequest,
			)
		}
		return w.aggregate.InvokeMapped(
			context.Background(),
			mapping,
			*request,
		)
	})
}

func (w *MCPWrapper) PutMCPServerSecret(
	srv artifact.ArtifactRef,
	kind mcpSecret.MCPSecretKind,
	slot string,
	value string,
) (MCPSecretWriteResult, error) {
	return middleware.WithRecoveryResp(func() (MCPSecretWriteResult, error) {
		if err := w.ready(); err != nil {
			return MCPSecretWriteResult{}, err
		}
		ctx := context.Background()
		if err := w.requireServerArtifact(ctx, srv); err != nil {
			return MCPSecretWriteResult{}, err
		}
		if kind == mcpSecret.MCPSecretKindOAuthToken {
			return MCPSecretWriteResult{}, fmt.Errorf(
				"%w: OAuth token secrets are runtime-managed",
				mcpAuth.ErrMCPInvalidAuthRequest,
			)
		}

		installationView, err := w.storeAPI.GetServerInstallation(
			ctx,
			srv,
		)
		if err != nil {
			return MCPSecretWriteResult{}, err
		}
		if err := validateMCPSecretTarget(
			installationView.Document,
			kind,
			slot,
		); err != nil {
			return MCPSecretWriteResult{}, err
		}

		if kind == mcpSecret.MCPSecretKindOAuthClientCredentials {
			mode := installationView.Document.Extension.Auth.Mode
			switch mode {
			case mcpSpec.MCPHTTPAuthOAuth,
				mcpSpec.MCPHTTPAuthClientCredentials:
			default:
				return MCPSecretWriteResult{}, fmt.Errorf(
					"%w: MCP server does not declare OAuth client credentials",
					mcpAuth.ErrMCPInvalidAuthRequest,
				)
			}
			if installationView.Document.Extension.Auth.ClientCredentialsInput == "" {
				return MCPSecretWriteResult{}, fmt.Errorf(
					"%w: MCP server does not declare an OAuth client credentials input",
					mcpAuth.ErrMCPInvalidAuthRequest,
				)
			}
			if err := mcpAuth.ValidateOAuthClientCredentialsSecret(
				value,
				installationView.Document.OAuthClientSecretRequired(),
			); err != nil {
				return MCPSecretWriteResult{}, err
			}
		}
		serverID, idErr := mcpAggregate.ServerIDForArtifact(srv)
		if idErr != nil {
			return MCPSecretWriteResult{}, idErr
		}
		if kind == mcpSecret.MCPSecretKindHTTPHeader &&
			(strings.TrimSpace(value) == "" ||
				strings.ContainsAny(value, "\r\n\x00")) {
			return MCPSecretWriteResult{}, fmt.Errorf(
				"%w: invalid HTTP header secret value",
				mcpAuth.ErrMCPInvalidAuthRequest,
			)
		}
		if err := w.runtime.Invalidate(ctx, serverID); err != nil {
			return MCPSecretWriteResult{}, err
		}

		ref, err := mcpSecret.NewMCPSecretRefString(srv, kind, slot)
		if err != nil {
			return MCPSecretWriteResult{}, err
		}
		hash, nonEmpty, err := w.secrets.SetMCPSecret(
			ctx,
			ref,
			value,
		)
		if err != nil {
			return MCPSecretWriteResult{}, err
		}
		w.auth.ClearAuthStatus(serverID)

		return MCPSecretWriteResult{
			SecretRef: ref,
			SHA256:    hash,
			NonEmpty:  nonEmpty,
		}, nil
	})
}

func (w *MCPWrapper) DeleteMCPServerSecret(
	srv artifact.ArtifactRef,
	kind mcpSecret.MCPSecretKind,
	slot string,
) error {
	return middleware.WithRecovery(func() error {
		if err := w.ready(); err != nil {
			return err
		}
		if err := w.requireServerArtifact(context.Background(), srv); err != nil {
			return err
		}
		if kind == mcpSecret.MCPSecretKindOAuthToken {
			return fmt.Errorf(
				"%w: OAuth token secrets are runtime-managed",
				mcpAuth.ErrMCPInvalidAuthRequest,
			)
		}
		serverID, idErr := mcpAggregate.ServerIDForArtifact(srv)
		if idErr != nil {
			return idErr
		}
		if err := w.runtime.Invalidate(context.Background(), serverID); err != nil {
			return err
		}
		ref, err := mcpSecret.NewMCPSecretRefString(srv, kind, slot)
		if err != nil {
			return err
		}
		if err := w.secrets.DeleteSecret(context.Background(), ref); err != nil {
			return err
		}
		w.auth.ClearAuthStatus(serverID)
		return nil
	})
}

func (w *MCPWrapper) GetMCPServerAuthHealth(
	ref artifact.ArtifactRef,
) (mcpAuth.MCPAuthHealth, error) {
	return middleware.WithRecoveryResp(func() (mcpAuth.MCPAuthHealth, error) {
		if err := w.ready(); err != nil {
			return mcpAuth.MCPAuthHealth{}, err
		}

		config, resolved, err := w.aggregate.InspectRuntimeConfig(
			context.Background(),
			ref,
		)
		if err != nil {
			serverID, idErr := mcpAggregate.ServerIDForArtifact(ref)
			if idErr != nil {
				return mcpAuth.MCPAuthHealth{}, idErr
			}
			return mcpAuth.MCPAuthHealth{
				Server:     serverID,
				AuthMode:   resolved.Document.Extension.Auth.Mode,
				State:      mcpAuth.MCPAuthHealthStateNotConfigured,
				Configured: false,
				LastError:  "required MCP installation input is not configured",
			}, nil
		}
		return w.auth.BuildAuthHealth(context.Background(), config), nil
	})
}

func (w *MCPWrapper) ListPendingMCPOAuthAuthorizations() []mcpAuth.MCPOAuthAuthorization {
	if w == nil || w.oauthBroker == nil {
		return []mcpAuth.MCPOAuthAuthorization{}
	}
	values := w.oauthBroker.Pending()
	if values == nil {
		return []mcpAuth.MCPOAuthAuthorization{}
	}
	return values
}

func (w *MCPWrapper) CancelPendingMCPOAuthAuthorization(
	srv artifact.ArtifactRef,
) bool {
	if w == nil {
		return false
	}
	serverID, err := mcpAggregate.ServerIDForArtifact(srv)
	if err != nil {
		return false
	}

	cancelled := false
	if w.oauthBroker != nil {
		cancelled = w.oauthBroker.Cancel(serverID)
	}
	if w.runtime != nil {
		if err := w.runtime.Disconnect(context.Background(), serverID); err != nil {
			slog.Warn("cancel MCP OAuth runtime connection", "server", srv, "error", err)
		}
	}
	if w.auth != nil {
		w.auth.ClearAuthStatus(serverID)
	}
	return cancelled
}

func (w *MCPWrapper) UpdateMCPGlobalSettings(
	expectedRevision uint64,
	settings mcpAuth.MCPAuthSettings,
) (uint64, error) {
	return middleware.WithRecoveryResp(func() (uint64, error) {
		if err := w.ready(); err != nil {
			return 0, err
		}
		return w.settings.PutMCPGlobalSettings(
			context.Background(),
			expectedRevision,
			settings,
		)
	})
}

func (w *MCPWrapper) GetMCPGlobalSettings() (
	MCPGlobalSettingsView,
	error,
) {
	return middleware.WithRecoveryResp(func() (MCPGlobalSettingsView, error) {
		if err := w.ready(); err != nil {
			return MCPGlobalSettingsView{}, err
		}
		settings, revision, err := w.settings.GetMCPGlobalSettings(
			context.Background(),
		)
		if err != nil {
			return MCPGlobalSettingsView{}, err
		}
		view := MCPGlobalSettingsView{
			Settings: settings,
			Revision: revision,
		}
		if w.oauthBroker != nil {
			view.OAuthRedirectURL = w.oauthBroker.RedirectURL()
			view.OAuthLoopbackListenAddr = w.oauthBroker.ListenAddr()
			view.OAuthLoopbackReady, view.OAuthLoopbackError = w.oauthBroker.Readiness()
			view.OAuthRestartRequired = strings.TrimSpace(settings.OAuthLoopbackListenAddr) !=
				w.oauthLoopbackListenAddrAtStart
		}
		return view, nil
	})
}

func (w *MCPWrapper) requireServerArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	record, err := w.artifacts.Get(ctx, ref)
	if err != nil {
		return err
	}
	if record.Kind != artifactbuiltin.ServerKind {
		return fmt.Errorf(
			"%w: Artifact %q is not an MCP Server",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	return nil
}

func validateMCPSecretTarget(
	document mcpArtifact.ServerDocument,
	kind mcpSecret.MCPSecretKind,
	slot string,
) error {
	switch kind {
	case mcpSecret.MCPSecretKindOAuthClientCredentials:
		input := document.Extension.Auth.ClientCredentialsInput
		if input == "" {
			return fmt.Errorf(
				"%w: MCP Server does not declare OAuth client credentials",
				mcpAuth.ErrMCPInvalidAuthRequest,
			)
		}
		declaration, found := document.Extension.Install.Inputs[input]
		if !found ||
			declaration.Kind != mcpArtifact.InputOAuthClientCredentials ||
			!strings.EqualFold(strings.TrimSpace(slot), "clientCredentials") {
			return fmt.Errorf(
				"%w: invalid OAuth client credentials secret target",
				mcpAuth.ErrMCPInvalidAuthRequest,
			)
		}
		return nil

	case mcpSecret.MCPSecretKindStdioEnv,
		mcpSecret.MCPSecretKindHTTPHeader:
		targets, err := mcpArtifact.SecretInputTargets(document)
		if err != nil {
			return err
		}
		for _, target := range targets {
			expectedKind := mcpSecret.MCPSecretKindStdioEnv
			if target.Kind == mcpArtifact.SecretInputTargetHTTPHeader {
				expectedKind = mcpSecret.MCPSecretKindHTTPHeader
			}
			if kind == expectedKind &&
				strings.EqualFold(target.Slot, strings.TrimSpace(slot)) {
				return nil
			}
		}
		return fmt.Errorf(
			"%w: secret target is not declared by the MCP Server",
			mcpAuth.ErrMCPInvalidAuthRequest,
		)

	default:
		return fmt.Errorf(
			"%w: unsupported MCP secret kind %q",
			mcpAuth.ErrMCPInvalidAuthRequest,
			kind,
		)
	}
}

func (w *MCPWrapper) ready() error {
	if w == nil ||
		w.storeAPI == nil ||
		w.artifacts == nil ||
		w.runtime == nil ||
		w.auth == nil ||
		w.settings == nil ||
		w.secrets == nil {
		return basespec.ErrClosed
	}
	return nil
}

func (w *MCPWrapper) close() {
	if w == nil {
		return
	}

	rt := w.runtime
	broker := w.oauthBroker

	w.storeAPI = nil
	w.artifacts = nil
	w.runtime = nil
	w.auth = nil
	w.overlays = nil
	w.settings = nil
	w.secrets = nil
	w.oauthBroker = nil
	w.oauthLoopbackListenAddrAtStart = ""
	w.builtInInstaller = nil

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if rt != nil {
		if err := rt.Close(ctx); err != nil {
			slog.Error("close artifact-backed MCP runtime", "error", err)
		}
	}
	if broker != nil {
		if err := broker.Close(); err != nil {
			slog.Error("close MCP OAuth broker", "error", err)
		}
	}
}

func ensureDefaultMCPBundle(
	ctx context.Context,
	api *mcpStore.API,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: default MCP Bundle context is nil",
			basespec.ErrInvalid,
		)
	}
	if api == nil {
		return fmt.Errorf(
			"%w: default MCP Bundle API is unavailable",
			basespec.ErrClosed,
		)
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

	document, err := json.Marshal(defaultMCPBundleDocument())
	if err != nil {
		return fmt.Errorf("encode default MCP Bundle document: %w", err)
	}

	_, err = api.Create(ctx, mcpStore.CreateRequest{
		RootID:           artifactbuiltin.MCPUserRootID,
		CollectionID:     artifactbuiltin.DefaultMCPBundleCollectionID,
		SourceID:         artifactbuiltin.DefaultMCPBundleSourceID,
		SourceStorageKey: artifactbuiltin.DefaultMCPBundleSourceKey,
		Document:         json.RawMessage(document),
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
		MCPServers:    map[string]mcpArtifact.CoreServer{},
		BundleExtension: mcpStore.BundleExtension{
			Servers:  map[string]mcpArtifact.ServerExtension{},
			Policies: map[string]mcpPolicy.PolicyDocument{},
		},
	}
}
