package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/bundle"
	"github.com/flexigpt/flexigpt-app/internal/mcp/overlay"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schemaadapter"
	"github.com/flexigpt/flexigpt-app/internal/mcp/sdkclient"
	"github.com/flexigpt/flexigpt-app/internal/mcp/secret"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

const (
	defaultMCPBundleCollectionID basespec.CollectionID = "0198f097-0d5b-7000-8000-000000000020"
	defaultMCPBundleSourceID     basespec.SourceID     = "0198f097-0d5b-7000-8000-000000000021"
)

type MCPWrapper struct {
	bundleAPI  *bundle.API
	artifacts  *artifact.Service
	runtime    *runtime.MCPRuntimeManager
	toolBridge *runtime.ToolBridge
	auth       *auth.AuthManager

	overlays *overlay.SettingsOverlayRepository
	settings *mcpSettingsAdapter
	secrets  *settingMCPSecretResolver

	oauthLoopbackListenAddrAtStart string
	oauthBroker                    *auth.OAuthLoopbackBroker
}

type MCPSecretWriteResult struct {
	SecretRef string `json:"secretRef"`
	SHA256    string `json:"sha256,omitempty"`
	NonEmpty  bool   `json:"nonEmpty"`
}

type MCPGlobalSettingsView struct {
	Settings                auth.MCPAuthSettings `json:"settings"`
	Revision                uint64               `json:"revision"`
	OAuthRedirectURL        string               `json:"oauthRedirectURL,omitempty"`
	OAuthLoopbackListenAddr string               `json:"oauthLoopbackListenAddr,omitempty"`
	OAuthRestartRequired    bool                 `json:"oauthRestartRequired"`
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
	if _, err := components.Roots.Get(ctx, mcpUserRootID); err != nil {
		return fmt.Errorf("ensure retained MCP user Root: %w", err)
	}

	settings, err := newMCPSettingsAdapter(settingsStore)
	if err != nil {
		return err
	}
	overlays, err := overlay.NewSettingsOverlayRepository(settings)
	if err != nil {
		return err
	}
	secrets := newSettingMCPSecretResolver(settingsStore)

	global, _, err := settings.GetMCPGlobalSettings(ctx)
	if err != nil {
		return err
	}

	configuredLoopback := strings.TrimSpace(global.OAuthLoopbackListenAddr)
	brokerOptions := &auth.OAuthLoopbackBrokerOptions{
		ListenAddr: configuredLoopback,
	}
	oauthBroker, err := auth.NewOAuthLoopbackBroker(ctx, brokerOptions)
	if err != nil {
		return err
	}

	authManager := auth.NewAuthManager(
		secrets,
		auth.WithOAuthAuthorizationBroker(oauthBroker),
		auth.WithOAuthRedirectURL(oauthBroker.RedirectURL()),
		auth.WithOAuthTokenStore(secrets),
	)

	invalidator := newMCPRuntimeInvalidator()
	bundleAPI, err := bundle.New(bundle.Dependencies{
		Sources:            components.Sources,
		Collections:        components.Collections,
		Artifacts:          components.Artifacts,
		ManagedArtifacts:   components.ManagedArtifacts,
		Refresh:            components.Refresh,
		Catalogs:           components.Catalogs,
		Definitions:        components.Definitions,
		SourceRuntime:      components.SourceRuntime,
		ShareableDocuments: components.ShareableSchemas,
		HasDecoder:         components.HasDecoder,
		DecoderFingerprint: components.DecoderFingerprint,
		RootPolicy:         components.RootMutationPolicy(),
		UserRootID:         mcpUserRootID,
		Runtime:            invalidator,
		Overlays:           overlays,
		SecretCleaner:      secrets,
		BaselinePolicy:     policy.Baseline(),
	})
	if err != nil {
		_ = oauthBroker.Close()
		return err
	}

	if err := ensureDefaultMCPBundle(ctx, bundleAPI); err != nil {
		_ = oauthBroker.Close()
		return err
	}

	rt, err := runtime.NewMCPRuntimeManager(
		bundleAPI,
		secrets,
		mcpEnvironmentResolver{},
		authManager,
		sdkclient.NewFactory(),
	)
	if err != nil {
		_ = oauthBroker.Close()
		return err
	}
	invalidator.Set(rt)

	toolBridge := runtime.NewToolBridge(
		rt,
		runtime.NewApprovalManager(5*time.Minute),
	)

	if err := installMCPBuiltIns(
		context.WithoutCancel(ctx),
		components,
		bundleAPI,
		overlays,
	); err != nil {
		_ = rt.Close(context.Background())
		_ = oauthBroker.Close()
		return err
	}

	wrapper.bundleAPI = bundleAPI
	wrapper.artifacts = components.Artifacts
	wrapper.runtime = rt
	wrapper.toolBridge = toolBridge
	wrapper.auth = authManager
	wrapper.overlays = overlays
	wrapper.settings = settings
	wrapper.secrets = secrets
	wrapper.oauthLoopbackListenAddrAtStart = configuredLoopback
	wrapper.oauthBroker = oauthBroker
	return nil
}

func installMCPBuiltIns(
	ctx context.Context,
	components *system.Components,
	bundles *bundle.API,
	overlays overlay.OverlayRepository,
) error {
	registry, packages, err := schemaadapter.LoadEmbeddedRegistry()
	if err != nil {
		return err
	}
	if registry.Topology.Root.ID != mcpBuiltInRootID {
		return fmt.Errorf(
			"%w: converted MCP built-in registry uses unexpected Root %q",
			basespec.ErrInvalid,
			registry.Topology.Root.ID,
		)
	}

	installer, err := schemaadapter.NewInstaller(
		schemaadapter.InstallerDependencies{
			Bundles:            bundles,
			Registry:           registry,
			Packages:           packages,
			Overlays:           overlays,
			ShareableDocuments: components.ShareableSchemas,
		},
	)
	if err != nil {
		return err
	}

	topologyRegistry, err := builtin.RegistryFromTopologyDeclaration(
		registry.Topology,
	)
	if err != nil {
		return err
	}
	bootstrap, err := builtin.NewBootstrapRegistry(
		topologyRegistry,
		components,
		components,
	)
	if err != nil {
		return err
	}
	if err := bootstrap.Register(installer); err != nil {
		return err
	}
	return bootstrap.Ensure(ctx)
}

func (w *MCPWrapper) CreateMCPBundle(
	request *bundle.CreateRequest,
) (bundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (bundle.Bundle, error) {
		if err := w.ready(); err != nil {
			return bundle.Bundle{}, err
		}
		if request == nil {
			return bundle.Bundle{}, errors.New("MCP Bundle create request is required")
		}
		return w.bundleAPI.Create(context.Background(), *request)
	})
}

func (w *MCPWrapper) GetMCPBundle(
	ref collection.CollectionRef,
) (bundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (bundle.Bundle, error) {
		if err := w.ready(); err != nil {
			return bundle.Bundle{}, err
		}
		return w.bundleAPI.Get(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPBundles(
	rootID basespec.RootID,
) ([]bundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() ([]bundle.Bundle, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.bundleAPI.List(context.Background(), rootID)
	})
}

func (w *MCPWrapper) GetMCPBundleDocument(
	ref collection.CollectionRef,
) (bundle.BundleDocument, error) {
	return middleware.WithRecoveryResp(func() (bundle.BundleDocument, error) {
		if err := w.ready(); err != nil {
			return bundle.BundleDocument{}, err
		}
		return w.bundleAPI.GetDocument(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPBundleServers(
	ref collection.CollectionRef,
) ([]artifact.Artifact, error) {
	return middleware.WithRecoveryResp(func() ([]artifact.Artifact, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.bundleAPI.ListServers(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPBundlePolicies(
	ref collection.CollectionRef,
) ([]artifact.Artifact, error) {
	return middleware.WithRecoveryResp(func() ([]artifact.Artifact, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.bundleAPI.ListPolicies(context.Background(), ref)
	})
}

func (w *MCPWrapper) GetMCPServerInstallation(
	ref artifact.ArtifactRef,
) (bundle.ServerInstallationView, error) {
	return middleware.WithRecoveryResp(
		func() (bundle.ServerInstallationView, error) {
			if err := w.ready(); err != nil {
				return bundle.ServerInstallationView{}, err
			}
			return w.bundleAPI.GetServerInstallation(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *MCPWrapper) InspectMCPServer(
	ref artifact.ArtifactRef,
) (server.Resolved, error) {
	return middleware.WithRecoveryResp(func() (server.Resolved, error) {
		if err := w.ready(); err != nil {
			return server.Resolved{}, err
		}
		return w.bundleAPI.InspectMCPServer(context.Background(), ref)
	})
}

func (w *MCPWrapper) InspectMCPPolicy(
	ref artifact.ArtifactRef,
) (bundle.PolicyView, error) {
	return middleware.WithRecoveryResp(func() (bundle.PolicyView, error) {
		if err := w.ready(); err != nil {
			return bundle.PolicyView{}, err
		}
		return w.bundleAPI.InspectMCPPolicy(context.Background(), ref)
	})
}

func (w *MCPWrapper) GetMCPBundleInstallation(
	ref collection.CollectionRef,
) (bundle.BundleInstallationView, error) {
	return middleware.WithRecoveryResp(
		func() (bundle.BundleInstallationView, error) {
			if err := w.ready(); err != nil {
				return bundle.BundleInstallationView{}, err
			}
			return w.bundleAPI.GetBundleInstallation(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *MCPWrapper) ReplaceMCPBundleDocument(
	request *bundle.ReplaceDocumentRequest,
) (bundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (bundle.Bundle, error) {
		if err := w.ready(); err != nil {
			return bundle.Bundle{}, err
		}
		if request == nil {
			return bundle.Bundle{}, errors.New("MCP document replacement request is required")
		}

		before, _ := w.artifacts.ListByCollection(
			context.Background(),
			request.Bundle,
		)
		result, err := w.bundleAPI.ReplaceDocument(
			context.Background(),
			*request,
		)
		if err != nil {
			return bundle.Bundle{}, err
		}
		for _, record := range before {
			w.auth.ClearAuthStatus(record.Ref())
		}
		return result, nil
	})
}

func (w *MCPWrapper) RefreshMCPBundle(
	ref collection.CollectionRef,
) (bundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (bundle.Bundle, error) {
		if err := w.ready(); err != nil {
			return bundle.Bundle{}, err
		}
		return w.bundleAPI.Refresh(context.Background(), ref, false)
	})
}

func (w *MCPWrapper) UpdateMCPBundleEnabled(
	ref collection.CollectionRef,
	expectedRevision uint64,
	enabled bool,
) (bundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (bundle.Bundle, error) {
		if err := w.ready(); err != nil {
			return bundle.Bundle{}, err
		}
		return w.bundleAPI.UpdateBundleEnabled(
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
		return w.bundleAPI.Retire(
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
		return w.bundleAPI.Purge(
			context.Background(),
			ref,
			expectedRevision,
		)
	})
}

func (w *MCPWrapper) UpdateMCPServerInstallation(
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	data server.ServerData,
) (artifact.Artifact, error) {
	return middleware.WithRecoveryResp(func() (artifact.Artifact, error) {
		if err := w.ready(); err != nil {
			return artifact.Artifact{}, err
		}
		value, err := w.bundleAPI.UpdateServerInstallation(
			context.Background(),
			ref,
			expectedArtifactRevision,
			data,
		)
		if err == nil {
			w.auth.ClearAuthStatus(ref)
		}
		return value, err
	})
}

func (w *MCPWrapper) UpdateProtectedMCPServerInstallation(
	ref artifact.ArtifactRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
	data server.ServerData,
) error {
	return middleware.WithRecovery(func() error {
		if err := w.ready(); err != nil {
			return err
		}
		if err := w.bundleAPI.UpdateProtectedServerInstallation(
			context.Background(),
			ref,
			expectedOverlayRevision,
			runtimeEnabled,
			data,
		); err != nil {
			return err
		}
		w.auth.ClearAuthStatus(ref)
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
		return w.bundleAPI.UpdateProtectedBundleInstallation(
			context.Background(),
			ref,
			expectedOverlayRevision,
			runtimeEnabled,
		)
	})
}

func (w *MCPWrapper) ConnectMCPServer(
	ref artifact.ArtifactRef,
) (*runtime.MCPServerRuntimeSnapshot, error) {
	return middleware.WithRecoveryResp(func() (*runtime.MCPServerRuntimeSnapshot, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.Connect(context.Background(), ref)
	})
}

func (w *MCPWrapper) DisconnectMCPServer(
	ref artifact.ArtifactRef,
) error {
	return middleware.WithRecovery(func() error {
		if err := w.ready(); err != nil {
			return err
		}
		return w.runtime.Disconnect(context.Background(), ref)
	})
}

func (w *MCPWrapper) RefreshMCPServer(
	ref artifact.ArtifactRef,
) (*runtime.MCPServerRuntimeSnapshot, error) {
	return middleware.WithRecoveryResp(func() (*runtime.MCPServerRuntimeSnapshot, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.Refresh(context.Background(), ref)
	})
}

func (w *MCPWrapper) GetMCPServerStatus(
	ref artifact.ArtifactRef,
) (*runtime.MCPServerRuntimeSnapshot, error) {
	return middleware.WithRecoveryResp(func() (*runtime.MCPServerRuntimeSnapshot, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.Status(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPServerTools(
	ref artifact.ArtifactRef,
) ([]runtime.MCPToolCapability, error) {
	return middleware.WithRecoveryResp(func() ([]runtime.MCPToolCapability, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.ListTools(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPServerResources(
	ref artifact.ArtifactRef,
) ([]runtime.MCPResourceRef, error) {
	return middleware.WithRecoveryResp(func() ([]runtime.MCPResourceRef, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.ListResources(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPServerResourceTemplates(
	ref artifact.ArtifactRef,
) ([]runtime.MCPResourceTemplateRef, error) {
	return middleware.WithRecoveryResp(func() ([]runtime.MCPResourceTemplateRef, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.ListResourceTemplates(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPServerPrompts(
	ref artifact.ArtifactRef,
) ([]runtime.MCPPromptRef, error) {
	return middleware.WithRecoveryResp(func() ([]runtime.MCPPromptRef, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.ListPrompts(context.Background(), ref)
	})
}

func (w *MCPWrapper) ReadMCPResource(
	srv artifact.ArtifactRef,
	uri string,
) (*runtime.MCPReadResourceResponseBody, error) {
	return middleware.WithRecoveryResp(func() (*runtime.MCPReadResourceResponseBody, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.ReadResource(context.Background(), srv, uri)
	})
}

func (w *MCPWrapper) GetMCPPrompt(
	srv artifact.ArtifactRef,
	name string,
	arguments map[string]string,
) (*runtime.MCPGetPromptResponseBody, error) {
	return middleware.WithRecoveryResp(func() (*runtime.MCPGetPromptResponseBody, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.GetPrompt(
			context.Background(),
			srv,
			name,
			arguments,
		)
	})
}

func (w *MCPWrapper) CompleteMCPArgument(
	srv artifact.ArtifactRef,
	request runtime.MCPCompleteArgumentRequestBody,
) (*runtime.MCPCompletionResult, error) {
	return middleware.WithRecoveryResp(func() (*runtime.MCPCompletionResult, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.Complete(context.Background(), srv, request)
	})
}

func (w *MCPWrapper) EvaluateMCPToolCall(
	srv artifact.ArtifactRef,
	request *runtime.InvokeMCPToolRequestBody,
) (*runtime.MCPApprovalEvaluation, error) {
	return middleware.WithRecoveryResp(func() (*runtime.MCPApprovalEvaluation, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		if request == nil {
			return nil, fmt.Errorf("%w: MCP tool request is required", runtime.ErrMCPInvalidRuntimeRequest)
		}
		return w.toolBridge.Evaluate(context.Background(), srv, *request)
	})
}

func (w *MCPWrapper) EvaluateMappedMCPToolCall(
	mapping runtime.MCPProviderToolMapping,
	request *runtime.InvokeMCPToolRequestBody,
) (*runtime.MCPApprovalEvaluation, error) {
	return middleware.WithRecoveryResp(func() (*runtime.MCPApprovalEvaluation, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		if request == nil {
			return nil, fmt.Errorf(
				"%w: mapped MCP tool request is required",
				runtime.ErrMCPInvalidRuntimeRequest,
			)
		}
		return w.toolBridge.EvaluateMapped(
			context.Background(),
			mapping,
			*request,
		)
	})
}

func (w *MCPWrapper) ResolveMCPApproval(
	approvalID string,
	resolution runtime.MCPApprovalResolution,
) (*runtime.MCPApprovalToken, error) {
	return middleware.WithRecoveryResp(func() (*runtime.MCPApprovalToken, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.toolBridge.ResolveApproval(
			context.Background(),
			approvalID,
			resolution,
		)
	})
}

func (w *MCPWrapper) InvokeMCPTool(
	srv artifact.ArtifactRef,
	request *runtime.InvokeMCPToolRequestBody,
) (*runtime.InvokeMCPToolResponseBody, error) {
	return middleware.WithRecoveryResp(func() (*runtime.InvokeMCPToolResponseBody, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		if request == nil {
			return nil, fmt.Errorf("%w: MCP tool request is required", runtime.ErrMCPInvalidRuntimeRequest)
		}
		return w.toolBridge.Invoke(context.Background(), srv, *request)
	})
}

func (w *MCPWrapper) InvokeMappedMCPTool(
	mapping runtime.MCPProviderToolMapping,
	request *runtime.InvokeMCPToolRequestBody,
) (*runtime.InvokeMCPToolResponseBody, error) {
	return middleware.WithRecoveryResp(func() (*runtime.InvokeMCPToolResponseBody, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		if request == nil {
			return nil, fmt.Errorf(
				"%w: mapped MCP tool request is required",
				runtime.ErrMCPInvalidRuntimeRequest,
			)
		}
		return w.toolBridge.InvokeMapped(
			context.Background(),
			mapping,
			*request,
		)
	})
}

func (w *MCPWrapper) PutMCPServerSecret(
	srv artifact.ArtifactRef,
	kind secret.MCPSecretKind,
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
		if kind == secret.MCPSecretKindOAuthToken {
			return MCPSecretWriteResult{}, fmt.Errorf(
				"%w: OAuth token secrets are runtime-managed",
				auth.ErrMCPInvalidAuthRequest,
			)
		}

		installationView, err := w.bundleAPI.GetServerInstallation(
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

		if kind == secret.MCPSecretKindOAuthClientCredentials {
			mode := installationView.Document.Extension.Auth.Mode
			switch mode {
			case server.MCPHTTPAuthOAuth,
				server.MCPHTTPAuthClientCredentials:
			default:
				return MCPSecretWriteResult{}, fmt.Errorf(
					"%w: MCP server does not declare OAuth client credentials",
					auth.ErrMCPInvalidAuthRequest,
				)
			}
			if installationView.Document.Extension.Auth.ClientCredentialsInput == "" {
				return MCPSecretWriteResult{}, fmt.Errorf(
					"%w: MCP server does not declare an OAuth client credentials input",
					auth.ErrMCPInvalidAuthRequest,
				)
			}
			if err := auth.ValidateOAuthClientCredentialsSecret(
				value,
				installationView.Document.OAuthClientSecretRequired(),
			); err != nil {
				return MCPSecretWriteResult{}, err
			}
		}

		if kind == secret.MCPSecretKindHTTPHeader &&
			(strings.TrimSpace(value) == "" ||
				strings.ContainsAny(value, "\r\n\x00")) {
			return MCPSecretWriteResult{}, fmt.Errorf(
				"%w: invalid HTTP header secret value",
				auth.ErrMCPInvalidAuthRequest,
			)
		}
		if err := w.runtime.Invalidate(ctx, srv); err != nil {
			return MCPSecretWriteResult{}, err
		}

		ref, err := secret.NewMCPSecretRefString(srv, kind, slot)
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
		w.auth.ClearAuthStatus(srv)

		return MCPSecretWriteResult{
			SecretRef: ref,
			SHA256:    hash,
			NonEmpty:  nonEmpty,
		}, nil
	})
}

func (w *MCPWrapper) DeleteMCPServerSecret(
	srv artifact.ArtifactRef,
	kind secret.MCPSecretKind,
	slot string,
) error {
	return middleware.WithRecovery(func() error {
		if err := w.ready(); err != nil {
			return err
		}
		if err := w.requireServerArtifact(context.Background(), srv); err != nil {
			return err
		}
		if kind == secret.MCPSecretKindOAuthToken {
			return fmt.Errorf(
				"%w: OAuth token secrets are runtime-managed",
				auth.ErrMCPInvalidAuthRequest,
			)
		}
		if err := w.runtime.Invalidate(context.Background(), srv); err != nil {
			return err
		}
		ref, err := secret.NewMCPSecretRefString(srv, kind, slot)
		if err != nil {
			return err
		}
		if err := w.secrets.DeleteSecret(context.Background(), ref); err != nil {
			return err
		}
		w.auth.ClearAuthStatus(srv)
		return nil
	})
}

func (w *MCPWrapper) GetMCPServerAuthHealth(
	ref artifact.ArtifactRef,
) (auth.MCPAuthHealth, error) {
	return middleware.WithRecoveryResp(func() (auth.MCPAuthHealth, error) {
		if err := w.ready(); err != nil {
			return auth.MCPAuthHealth{}, err
		}

		resolved, err := w.bundleAPI.InspectMCPServer(context.Background(), ref)
		if err != nil {
			return auth.MCPAuthHealth{}, err
		}
		config, err := resolved.MaterializeForInspection(
			context.Background(),
			mcpEnvironmentResolver{},
		)
		if err != nil {
			//nolint:nilerr // Explicit value.
			return auth.MCPAuthHealth{
				Server:     ref,
				AuthMode:   resolved.Document.Extension.Auth.Mode,
				State:      auth.MCPAuthHealthStateNotConfigured,
				Configured: false,
				LastError:  "required MCP installation input is not configured",
			}, nil
		}
		return w.auth.BuildAuthHealth(context.Background(), config), nil
	})
}

func (w *MCPWrapper) ListPendingMCPOAuthAuthorizations() []auth.MCPOAuthAuthorization {
	if w == nil || w.oauthBroker == nil {
		return []auth.MCPOAuthAuthorization{}
	}
	values := w.oauthBroker.Pending()
	if values == nil {
		return []auth.MCPOAuthAuthorization{}
	}
	return values
}

func (w *MCPWrapper) CancelPendingMCPOAuthAuthorization(
	srv artifact.ArtifactRef,
) bool {
	if w == nil || w.oauthBroker == nil {
		return false
	}
	return w.oauthBroker.Cancel(srv)
}

func (w *MCPWrapper) UpdateMCPGlobalSettings(
	expectedRevision uint64,
	settings auth.MCPAuthSettings,
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
	if record.Kind != schema.ServerKind {
		return fmt.Errorf(
			"%w: Artifact %q is not an MCP Server",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	return nil
}

func validateMCPSecretTarget(
	document server.ServerDocument,
	kind secret.MCPSecretKind,
	slot string,
) error {
	switch kind {
	case secret.MCPSecretKindOAuthClientCredentials:
		input := document.Extension.Auth.ClientCredentialsInput
		if input == "" {
			return fmt.Errorf(
				"%w: MCP Server does not declare OAuth client credentials",
				auth.ErrMCPInvalidAuthRequest,
			)
		}
		declaration, found := document.Extension.Install.Inputs[input]
		if !found ||
			declaration.Kind != server.InputOAuthClientCredentials ||
			!strings.EqualFold(strings.TrimSpace(slot), "clientCredentials") {
			return fmt.Errorf(
				"%w: invalid OAuth client credentials secret target",
				auth.ErrMCPInvalidAuthRequest,
			)
		}
		return nil

	case secret.MCPSecretKindStdioEnv,
		secret.MCPSecretKindHTTPHeader:
		targets, err := server.SecretInputTargets(document)
		if err != nil {
			return err
		}
		for _, target := range targets {
			expectedKind := secret.MCPSecretKindStdioEnv
			if target.Kind == server.SecretInputTargetHTTPHeader {
				expectedKind = secret.MCPSecretKindHTTPHeader
			}
			if kind == expectedKind &&
				strings.EqualFold(target.Slot, strings.TrimSpace(slot)) {
				return nil
			}
		}
		return fmt.Errorf(
			"%w: secret target is not declared by the MCP Server",
			auth.ErrMCPInvalidAuthRequest,
		)

	default:
		return fmt.Errorf(
			"%w: unsupported MCP secret kind %q",
			auth.ErrMCPInvalidAuthRequest,
			kind,
		)
	}
}

func (w *MCPWrapper) ready() error {
	if w == nil ||
		w.bundleAPI == nil ||
		w.artifacts == nil ||
		w.runtime == nil ||
		w.toolBridge == nil ||
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

	w.bundleAPI = nil
	w.artifacts = nil
	w.runtime = nil
	w.toolBridge = nil
	w.auth = nil
	w.overlays = nil
	w.settings = nil
	w.secrets = nil
	w.oauthBroker = nil
	w.oauthLoopbackListenAddrAtStart = ""

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
	api *bundle.API,
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
		RootID:       mcpUserRootID,
		CollectionID: defaultMCPBundleCollectionID,
	}
	existing, err := api.Get(ctx, ref)
	switch {
	case err == nil:
		if existing.Data.ManagedSourceID != defaultMCPBundleSourceID ||
			existing.Source.ID != defaultMCPBundleSourceID {
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

	_, err = api.Create(ctx, bundle.CreateRequest{
		RootID:       mcpUserRootID,
		CollectionID: defaultMCPBundleCollectionID,
		SourceID:     defaultMCPBundleSourceID,
		Document:     json.RawMessage(document),
	})
	if err != nil {
		return fmt.Errorf("create default MCP Bundle: %w", err)
	}
	return nil
}

func defaultMCPBundleDocument() bundle.BundleDocument {
	return bundle.BundleDocument{
		Kind:          schema.BundleKind,
		SchemaID:      schema.BundleSchemaID,
		SchemaVersion: schema.MCPSchemaVersion,
		LogicalName:   "base",
		DisplayName:   "Base MCP Servers",
		Description:   "Editable starter bundle for user-managed MCP server definitions.",
		MCPServers:    map[string]server.CoreServer{},
		BundleExtension: bundle.BundleExtension{
			Servers:  map[string]server.ServerExtension{},
			Policies: map[string]policy.PolicyDocument{},
		},
	}
}
