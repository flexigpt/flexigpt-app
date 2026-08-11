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
	"github.com/flexigpt/flexigpt-app/internal/mcp/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
	mcpBundle "github.com/flexigpt/flexigpt-app/internal/mcp/bundle"
	"github.com/flexigpt/flexigpt-app/internal/mcp/installation"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
	mcpRuntime "github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
	"github.com/flexigpt/flexigpt-app/internal/mcp/sdkclient"
	"github.com/flexigpt/flexigpt-app/internal/mcp/secret"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

const (
	defaultMCPBundleCollectionID basespec.CollectionID = "0198f097-0d5b-7000-8000-000000000020"
	defaultMCPBundleSourceID     basespec.SourceID     = "0198f097-0d5b-7000-8000-000000000021"
)

type MCPWrapper struct {
	bundleAPI  *mcpBundle.API
	artifacts  *artifact.Service
	runtime    *mcpRuntime.MCPRuntimeManager
	toolBridge *mcpRuntime.ToolBridge
	auth       *auth.AuthManager

	overlays *installation.SettingsOverlayRepository
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
	Settings                mcpSpec.MCPSettings `json:"settings"`
	Revision                uint64              `json:"revision"`
	OAuthRedirectURL        string              `json:"oauthRedirectURL,omitempty"`
	OAuthLoopbackListenAddr string              `json:"oauthLoopbackListenAddr,omitempty"`
	OAuthRestartRequired    bool                `json:"oauthRestartRequired"`
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
	overlays, err := installation.NewSettingsOverlayRepository(settings)
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
	bundleAPI, err := mcpBundle.New(mcpBundle.Dependencies{
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

	runtime, err := mcpRuntime.NewMCPRuntimeManager(
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
	invalidator.Set(runtime)

	toolBridge := mcpRuntime.NewToolBridge(
		runtime,
		mcpRuntime.NewApprovalManager(5*time.Minute),
	)

	if err := installMCPBuiltIns(
		context.WithoutCancel(ctx),
		components,
		bundleAPI,
		overlays,
	); err != nil {
		_ = runtime.Close(context.Background())
		_ = oauthBroker.Close()
		return err
	}

	wrapper.bundleAPI = bundleAPI
	wrapper.artifacts = components.Artifacts
	wrapper.runtime = runtime
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
	bundles *mcpBundle.API,
	overlays installation.OverlayRepository,
) error {
	registry, packages, err := artifactbuiltin.LoadEmbeddedRegistry()
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

	installer, err := artifactbuiltin.NewInstaller(
		artifactbuiltin.InstallerDependencies{
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
	request *mcpBundle.CreateRequest,
) (mcpBundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (mcpBundle.Bundle, error) {
		if err := w.ready(); err != nil {
			return mcpBundle.Bundle{}, err
		}
		if request == nil {
			return mcpBundle.Bundle{}, errors.New("MCP Bundle create request is required")
		}
		return w.bundleAPI.Create(context.Background(), *request)
	})
}

func (w *MCPWrapper) GetMCPBundle(
	ref collection.CollectionRef,
) (mcpBundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (mcpBundle.Bundle, error) {
		if err := w.ready(); err != nil {
			return mcpBundle.Bundle{}, err
		}
		return w.bundleAPI.Get(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPBundles(
	rootID basespec.RootID,
) ([]mcpBundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() ([]mcpBundle.Bundle, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.bundleAPI.List(context.Background(), rootID)
	})
}

func (w *MCPWrapper) ReplaceMCPBundleDocument(
	request *mcpBundle.ReplaceDocumentRequest,
) (mcpBundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (mcpBundle.Bundle, error) {
		if err := w.ready(); err != nil {
			return mcpBundle.Bundle{}, err
		}
		if request == nil {
			return mcpBundle.Bundle{}, errors.New("MCP document replacement request is required")
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
			return mcpBundle.Bundle{}, err
		}
		for _, record := range before {
			w.auth.ClearAuthStatus(record.Ref())
		}
		return result, nil
	})
}

func (w *MCPWrapper) RefreshMCPBundle(
	ref collection.CollectionRef,
) (mcpBundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (mcpBundle.Bundle, error) {
		if err := w.ready(); err != nil {
			return mcpBundle.Bundle{}, err
		}
		return w.bundleAPI.Refresh(context.Background(), ref, false)
	})
}

func (w *MCPWrapper) UpdateMCPBundleEnabled(
	ref collection.CollectionRef,
	expectedRevision uint64,
	enabled bool,
) (mcpBundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (mcpBundle.Bundle, error) {
		if err := w.ready(); err != nil {
			return mcpBundle.Bundle{}, err
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
	data installation.ServerData,
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
	data installation.ServerData,
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
) (*mcpSpec.MCPServerRuntimeSnapshot, error) {
	return middleware.WithRecoveryResp(func() (*mcpSpec.MCPServerRuntimeSnapshot, error) {
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
) (*mcpSpec.MCPServerRuntimeSnapshot, error) {
	return middleware.WithRecoveryResp(func() (*mcpSpec.MCPServerRuntimeSnapshot, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.Refresh(context.Background(), ref)
	})
}

func (w *MCPWrapper) GetMCPServerStatus(
	ref artifact.ArtifactRef,
) (*mcpSpec.MCPServerRuntimeSnapshot, error) {
	return middleware.WithRecoveryResp(func() (*mcpSpec.MCPServerRuntimeSnapshot, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.Status(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPServerTools(
	ref artifact.ArtifactRef,
) ([]mcpSpec.MCPToolCapability, error) {
	return middleware.WithRecoveryResp(func() ([]mcpSpec.MCPToolCapability, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.ListTools(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPServerResources(
	ref artifact.ArtifactRef,
) ([]mcpSpec.MCPResourceRef, error) {
	return middleware.WithRecoveryResp(func() ([]mcpSpec.MCPResourceRef, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.ListResources(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPServerResourceTemplates(
	ref artifact.ArtifactRef,
) ([]mcpSpec.MCPResourceTemplateRef, error) {
	return middleware.WithRecoveryResp(func() ([]mcpSpec.MCPResourceTemplateRef, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.ListResourceTemplates(context.Background(), ref)
	})
}

func (w *MCPWrapper) ListMCPServerPrompts(
	ref artifact.ArtifactRef,
) ([]mcpSpec.MCPPromptRef, error) {
	return middleware.WithRecoveryResp(func() ([]mcpSpec.MCPPromptRef, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.ListPrompts(context.Background(), ref)
	})
}

func (w *MCPWrapper) ReadMCPResource(
	server artifact.ArtifactRef,
	uri string,
) (*mcpSpec.MCPReadResourceResponseBody, error) {
	return middleware.WithRecoveryResp(func() (*mcpSpec.MCPReadResourceResponseBody, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.ReadResource(context.Background(), server, uri)
	})
}

func (w *MCPWrapper) GetMCPPrompt(
	server artifact.ArtifactRef,
	name string,
	arguments map[string]string,
) (*mcpSpec.MCPGetPromptResponseBody, error) {
	return middleware.WithRecoveryResp(func() (*mcpSpec.MCPGetPromptResponseBody, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.GetPrompt(
			context.Background(),
			server,
			name,
			arguments,
		)
	})
}

func (w *MCPWrapper) CompleteMCPArgument(
	server artifact.ArtifactRef,
	request mcpSpec.MCPCompleteArgumentRequestBody,
) (*mcpSpec.MCPCompletionResult, error) {
	return middleware.WithRecoveryResp(func() (*mcpSpec.MCPCompletionResult, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		return w.runtime.Complete(context.Background(), server, request)
	})
}

func (w *MCPWrapper) EvaluateMCPToolCall(
	server artifact.ArtifactRef,
	request *mcpSpec.InvokeMCPToolRequestBody,
) (*mcpSpec.MCPApprovalEvaluation, error) {
	return middleware.WithRecoveryResp(func() (*mcpSpec.MCPApprovalEvaluation, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		if request == nil {
			return nil, fmt.Errorf("%w: MCP tool request is required", mcpSpec.ErrMCPInvalidRequest)
		}
		return w.toolBridge.Evaluate(context.Background(), server, *request)
	})
}

func (w *MCPWrapper) EvaluateMappedMCPToolCall(
	mapping mcpSpec.MCPProviderToolMapping,
	request *mcpSpec.InvokeMCPToolRequestBody,
) (*mcpSpec.MCPApprovalEvaluation, error) {
	return middleware.WithRecoveryResp(func() (*mcpSpec.MCPApprovalEvaluation, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		if request == nil {
			return nil, fmt.Errorf(
				"%w: mapped MCP tool request is required",
				mcpSpec.ErrMCPInvalidRequest,
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
	resolution mcpSpec.MCPApprovalResolution,
) (*mcpSpec.MCPApprovalToken, error) {
	return middleware.WithRecoveryResp(func() (*mcpSpec.MCPApprovalToken, error) {
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
	server artifact.ArtifactRef,
	request *mcpSpec.InvokeMCPToolRequestBody,
) (*mcpSpec.InvokeMCPToolResponseBody, error) {
	return middleware.WithRecoveryResp(func() (*mcpSpec.InvokeMCPToolResponseBody, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		if request == nil {
			return nil, fmt.Errorf("%w: MCP tool request is required", mcpSpec.ErrMCPInvalidRequest)
		}
		return w.toolBridge.Invoke(context.Background(), server, *request)
	})
}

func (w *MCPWrapper) InvokeMappedMCPTool(
	mapping mcpSpec.MCPProviderToolMapping,
	request *mcpSpec.InvokeMCPToolRequestBody,
) (*mcpSpec.InvokeMCPToolResponseBody, error) {
	return middleware.WithRecoveryResp(func() (*mcpSpec.InvokeMCPToolResponseBody, error) {
		if err := w.ready(); err != nil {
			return nil, err
		}
		if request == nil {
			return nil, fmt.Errorf(
				"%w: mapped MCP tool request is required",
				mcpSpec.ErrMCPInvalidRequest,
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
	server artifact.ArtifactRef,
	kind mcpSpec.MCPSecretKind,
	slot string,
	value string,
) (MCPSecretWriteResult, error) {
	return middleware.WithRecoveryResp(func() (MCPSecretWriteResult, error) {
		if err := w.ready(); err != nil {
			return MCPSecretWriteResult{}, err
		}
		ctx := context.Background()
		if err := w.requireServerArtifact(ctx, server); err != nil {
			return MCPSecretWriteResult{}, err
		}
		if kind == mcpSpec.MCPSecretKindOAuthToken {
			return MCPSecretWriteResult{}, fmt.Errorf(
				"%w: OAuth token secrets are runtime-managed",
				mcpSpec.ErrMCPInvalidRequest,
			)
		}

		if kind == mcpSpec.MCPSecretKindOAuthClientCredentials {
			resolved, err := w.bundleAPI.ResolveMCPServer(ctx, server)
			if err != nil {
				return MCPSecretWriteResult{}, err
			}
			mode := resolved.Document.Extension.Auth.Mode
			switch mode {
			case mcpSpec.MCPHTTPAuthOAuth,
				mcpSpec.MCPHTTPAuthClientCredentials:
			default:
				return MCPSecretWriteResult{}, fmt.Errorf(
					"%w: MCP server does not declare OAuth client credentials",
					mcpSpec.ErrMCPInvalidRequest,
				)
			}
			if resolved.Document.Extension.Auth.ClientCredentialsInput == "" {
				return MCPSecretWriteResult{}, fmt.Errorf(
					"%w: MCP server does not declare an OAuth client credentials input",
					mcpSpec.ErrMCPInvalidRequest,
				)
			}
			if err := auth.ValidateOAuthClientCredentialsSecret(
				value,
				resolved.Document.OAuthClientSecretRequired(),
			); err != nil {
				return MCPSecretWriteResult{}, err
			}
		}

		if kind == mcpSpec.MCPSecretKindHTTPHeader &&
			(strings.TrimSpace(value) == "" ||
				strings.ContainsAny(value, "\r\n\x00")) {
			return MCPSecretWriteResult{}, fmt.Errorf(
				"%w: invalid HTTP header secret value",
				mcpSpec.ErrMCPInvalidRequest,
			)
		}
		if err := w.runtime.Invalidate(ctx, server); err != nil {
			return MCPSecretWriteResult{}, err
		}

		ref, err := secret.NewMCPSecretRefString(server, kind, slot)
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
		w.auth.ClearAuthStatus(server)

		return MCPSecretWriteResult{
			SecretRef: ref,
			SHA256:    hash,
			NonEmpty:  nonEmpty,
		}, nil
	})
}

func (w *MCPWrapper) DeleteMCPServerSecret(
	server artifact.ArtifactRef,
	kind mcpSpec.MCPSecretKind,
	slot string,
) error {
	return middleware.WithRecovery(func() error {
		if err := w.ready(); err != nil {
			return err
		}
		if err := w.requireServerArtifact(context.Background(), server); err != nil {
			return err
		}
		if kind == mcpSpec.MCPSecretKindOAuthToken {
			return fmt.Errorf(
				"%w: OAuth token secrets are runtime-managed",
				mcpSpec.ErrMCPInvalidRequest,
			)
		}
		if err := w.runtime.Invalidate(context.Background(), server); err != nil {
			return err
		}
		ref, err := secret.NewMCPSecretRefString(server, kind, slot)
		if err != nil {
			return err
		}
		if err := w.secrets.DeleteSecret(context.Background(), ref); err != nil {
			return err
		}
		w.auth.ClearAuthStatus(server)
		return nil
	})
}

func (w *MCPWrapper) GetMCPServerAuthHealth(
	ref artifact.ArtifactRef,
) (mcpSpec.MCPAuthHealth, error) {
	return middleware.WithRecoveryResp(func() (mcpSpec.MCPAuthHealth, error) {
		if err := w.ready(); err != nil {
			return mcpSpec.MCPAuthHealth{}, err
		}

		resolved, err := w.bundleAPI.InspectMCPServer(context.Background(), ref)
		if err != nil {
			return mcpSpec.MCPAuthHealth{}, err
		}
		config, err := resolved.MaterializeForInspection(
			context.Background(),
			mcpEnvironmentResolver{},
		)
		if err != nil {
			//nolint:nilerr // Explicit value.
			return mcpSpec.MCPAuthHealth{
				Server:     ref,
				AuthMode:   resolved.Document.Extension.Auth.Mode,
				State:      mcpSpec.MCPAuthHealthStateNotConfigured,
				Configured: false,
				LastError:  "required MCP installation input is not configured",
			}, nil
		}
		return w.auth.BuildAuthHealth(context.Background(), config), nil
	})
}

func (w *MCPWrapper) ListPendingMCPOAuthAuthorizations() []mcpSpec.MCPOAuthAuthorization {
	if w == nil || w.oauthBroker == nil {
		return []mcpSpec.MCPOAuthAuthorization{}
	}
	values := w.oauthBroker.Pending()
	if values == nil {
		return []mcpSpec.MCPOAuthAuthorization{}
	}
	return values
}

func (w *MCPWrapper) CancelPendingMCPOAuthAuthorization(
	server artifact.ArtifactRef,
) bool {
	if w == nil || w.oauthBroker == nil {
		return false
	}
	return w.oauthBroker.Cancel(server)
}

func (w *MCPWrapper) UpdateMCPGlobalSettings(
	expectedRevision uint64,
	settings mcpSpec.MCPSettings,
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

	runtime := w.runtime
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

	if runtime != nil {
		if err := runtime.Close(ctx); err != nil {
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
	api *mcpBundle.API,
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

	_, err = api.Create(ctx, mcpBundle.CreateRequest{
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

func defaultMCPBundleDocument() schema.BundleDocument {
	return schema.BundleDocument{
		Kind:          schema.BundleKind,
		SchemaID:      schema.BundleSchemaID,
		SchemaVersion: schema.SchemaVersion,
		LogicalName:   "base",
		DisplayName:   "Base MCP Servers",
		Description:   "Editable starter bundle for user-managed MCP server definitions.",
		MCPServers:    map[string]schema.CoreServer{},
		BundleExtension: schema.BundleExtension{
			Servers:  map[string]schema.ServerExtension{},
			Policies: map[string]schema.PolicyDocument{},
		},
	}
}
