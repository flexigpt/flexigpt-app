package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/middleware"

	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	"github.com/flexigpt/flexigpt-app/internal/mcp/sdkclient"
	"github.com/flexigpt/flexigpt-app/internal/mcp/secret"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/store"

	settingSpec "github.com/flexigpt/flexigpt-app/internal/setting/spec"
)

type mcpAuthKeyStore interface {
	mcpAuthKeyReader
	SetAuthKey(ctx context.Context, req *settingSpec.SetAuthKeyRequest) (*settingSpec.SetAuthKeyResponse, error)
	DeleteAuthKey(
		ctx context.Context,
		req *settingSpec.DeleteAuthKeyRequest,
	) (*settingSpec.DeleteAuthKeyResponse, error)
}

type mcpSecretWriter interface {
	SetMCPSecret(ctx context.Context, secretRef, value string) (sha256 string, nonEmpty bool, err error)
	DeleteMCPSecret(ctx context.Context, secretRef string) error
}

type mcpAuthKeyReader interface {
	GetAuthKey(ctx context.Context, req *settingSpec.GetAuthKeyRequest) (*settingSpec.GetAuthKeyResponse, error)
}

type settingSecretResolver struct {
	store mcpAuthKeyReader
}

func newSettingSecretResolver(s mcpAuthKeyReader) auth.SecretResolver {
	if s == nil {
		return auth.StaticSecretResolver{}
	}
	return &settingSecretResolver{store: s}
}

func (r *settingSecretResolver) ResolveSecret(
	ctx context.Context,
	keyName string,
) (string, error) {
	if r == nil || r.store == nil {
		return "", errors.New("secret resolver is not configured")
	}
	keyName = strings.TrimSpace(keyName)
	if keyName == "" {
		return "", errors.New("invalid secret ref")
	}

	ref, err := secret.ParseMCPSecretRef(keyName)
	if err != nil {
		return "", err
	}

	resp, err := r.store.GetAuthKey(ctx, &settingSpec.GetAuthKeyRequest{
		Type:    settingSpec.AuthKeyTypeMCP,
		KeyName: settingSpec.AuthKeyName(secret.GetMCPSecretRefStorageKey(ref)),
	})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.Body == nil {
		return "", fmt.Errorf("secret ref %s/%s returned empty response", settingSpec.AuthKeyTypeMCP, keyName)
	}
	if !resp.Body.NonEmpty {
		return "", fmt.Errorf("secret ref %s/%s is empty", settingSpec.AuthKeyTypeMCP, keyName)
	}
	return resp.Body.Secret, nil
}

func (r *settingSecretResolver) SetMCPSecret(
	ctx context.Context,
	secretRef string,
	value string,
) (sha256 string, nonEmpty bool, err error) {
	if r == nil || r.store == nil {
		return "", false, errors.New("secret resolver is not configured")
	}
	st, ok := r.store.(mcpAuthKeyStore)
	if !ok {
		return "", false, errors.New("secret writer is not configured")
	}

	ref, err := secret.ParseMCPSecretRef(secretRef)
	if err != nil {
		return "", false, err
	}

	keyName := settingSpec.AuthKeyName(secret.GetMCPSecretRefStorageKey(ref))
	if keyName == "" {
		return "", false, errors.New("invalid secret ref storage key")
	}

	if _, err := st.SetAuthKey(ctx, &settingSpec.SetAuthKeyRequest{
		Type:    settingSpec.AuthKeyTypeMCP,
		KeyName: keyName,
		Body: &settingSpec.SetAuthKeyRequestBody{
			Secret: value,
		},
	}); err != nil {
		return "", false, err
	}

	resp, err := st.GetAuthKey(ctx, &settingSpec.GetAuthKeyRequest{
		Type:    settingSpec.AuthKeyTypeMCP,
		KeyName: keyName,
	})
	if err != nil {
		return "", false, err
	}
	if resp == nil || resp.Body == nil {
		return "", false, errors.New("secret set returned empty response")
	}
	return resp.Body.SHA256, resp.Body.NonEmpty, nil
}

func (r *settingSecretResolver) DeleteMCPSecret(ctx context.Context, secretRef string) error {
	if r == nil || r.store == nil {
		return errors.New("secret resolver is not configured")
	}
	st, ok := r.store.(mcpAuthKeyStore)
	if !ok {
		return errors.New("secret writer is not configured")
	}

	ref, err := secret.ParseMCPSecretRef(secretRef)
	if err != nil {
		return err
	}

	keyName := settingSpec.AuthKeyName(secret.GetMCPSecretRefStorageKey(ref))
	if keyName == "" {
		return errors.New("invalid secret ref storage key")
	}

	_, err = st.DeleteAuthKey(ctx, &settingSpec.DeleteAuthKeyRequest{
		Type:    settingSpec.AuthKeyTypeMCP,
		KeyName: keyName,
	})
	return err
}

type MCPWrapper struct {
	store *store.Store

	auth           *auth.AuthManager
	secretResolver auth.SecretResolver
	secretWriter   mcpSecretWriter
	oauthBroker    *auth.OAuthLoopbackBroker

	runtime    *runtime.MCPRuntimeManager
	approvals  *runtime.ApprovalManager
	toolBridge *runtime.ToolBridge
}

func InitMCPWrapper(ctx context.Context, w *MCPWrapper, baseDir string, secrets auth.SecretResolver) error {
	if secrets == nil {
		secrets = auth.StaticSecretResolver{}
	}
	st, err := store.NewMCPStore(ctx, baseDir)
	if err != nil {
		return err
	}

	oauthBroker, err := auth.NewOAuthLoopbackBroker(ctx, nil)
	if err != nil {
		_ = st.Close()
		return err
	}

	authMgr := auth.NewAuthManager(
		secrets,
		auth.WithOAuthAuthorizationBroker(oauthBroker),
		auth.WithOAuthRedirectURL(oauthBroker.RedirectURL()),
	)
	rt := runtime.NewMCPRuntimeManager(st, authMgr, sdkclient.NewFactory())
	appr := runtime.NewApprovalManager(5 * time.Minute)
	tb := runtime.NewToolBridge(rt, appr)

	w.store = st
	w.auth = authMgr
	w.secretResolver = secrets
	if writer, ok := secrets.(mcpSecretWriter); ok {
		w.secretWriter = writer
	}
	w.oauthBroker = oauthBroker
	w.runtime = rt
	w.approvals = appr
	w.toolBridge = tb

	return nil
}

func (w *MCPWrapper) PutMCPBundle(req *spec.PutMCPBundleRequest) (*spec.PutMCPBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PutMCPBundleResponse, error) {
		return w.store.PutMCPBundle(context.Background(), req)
	})
}

func (w *MCPWrapper) PatchMCPBundle(req *spec.PatchMCPBundleRequest) (*spec.PatchMCPBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchMCPBundleResponse, error) {
		resp, err := w.store.PatchMCPBundle(context.Background(), req)
		if err != nil {
			return nil, err
		}
		if req != nil && req.Body != nil && !req.Body.IsEnabled {
			w.disconnectBundleServers(context.Background(), req.BundleID)
		}
		return resp, nil
	})
}

func (w *MCPWrapper) DeleteMCPBundle(req *spec.DeleteMCPBundleRequest) (*spec.DeleteMCPBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DeleteMCPBundleResponse, error) {
		resp, err := w.store.DeleteMCPBundle(context.Background(), req)
		if err != nil {
			return nil, err
		}
		if req != nil {
			w.disconnectBundleServers(context.Background(), req.BundleID)
		}
		return resp, nil
	})
}

func (w *MCPWrapper) ListMCPBundles(req *spec.ListMCPBundlesRequest) (*spec.ListMCPBundlesResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListMCPBundlesResponse, error) {
		return w.store.ListMCPBundles(context.Background(), req)
	})
}

func (w *MCPWrapper) GetMCPServerAuthStatus(
	req *spec.GetMCPServerAuthStatusRequest,
) (*spec.GetMCPServerAuthStatusResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.GetMCPServerAuthStatusResponse, error) {
		if req == nil || req.BundleID == "" || req.ServerID == "" {
			return nil, fmt.Errorf("%w: bundleID and serverID required", spec.ErrMCPInvalidRequest)
		}

		cfgResp, err := w.store.GetMCPServer(context.Background(), &spec.GetMCPServerRequest{
			BundleID: req.BundleID,
			ServerID: req.ServerID,
		})
		if err != nil {
			return nil, err
		}
		if cfgResp == nil || cfgResp.Body == nil {
			return nil, fmt.Errorf("%w: empty server config response", spec.ErrMCPRuntimeNotReady)
		}

		st := auth.DefaultMCPAuthStatusFromConfig(*cfgResp.Body)
		if w != nil && w.auth != nil {
			if cur, ok := w.auth.GetAuthStatus(req.BundleID, req.ServerID); ok {
				st = auth.MergeMCPAuthStatus(cur, *cfgResp.Body)
			}
		}
		return &spec.GetMCPServerAuthStatusResponse{Body: &st}, nil
	})
}

func (w *MCPWrapper) PutMCPServer(req *spec.PutMCPServerRequest) (*spec.PutMCPServerResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PutMCPServerResponse, error) {
		resp, err := w.store.PutMCPServer(context.Background(), req)
		if err != nil {
			return nil, err
		}
		if req != nil && req.ServerID != "" {
			_, _ = w.runtime.Disconnect(context.Background(), &spec.DisconnectMCPServerRequest{
				BundleID: req.BundleID,
				ServerID: req.ServerID,
			})
		}
		return resp, nil
	})
}

func (w *MCPWrapper) GetMCPServer(req *spec.GetMCPServerRequest) (*spec.GetMCPServerResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.GetMCPServerResponse, error) {
		return w.store.GetMCPServer(context.Background(), req)
	})
}

func (w *MCPWrapper) ListMCPServers(req *spec.ListMCPServersRequest) (*spec.ListMCPServersResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListMCPServersResponse, error) {
		return w.store.ListMCPServers(context.Background(), req)
	})
}

func (w *MCPWrapper) PatchMCPServerEnabled(
	req *spec.PatchMCPServerEnabledRequest,
) (*spec.PatchMCPServerEnabledResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchMCPServerEnabledResponse, error) {
		resp, err := w.store.PatchMCPServerEnabled(context.Background(), req)
		if err != nil {
			return nil, err
		}
		if req != nil && req.Body != nil && !req.Body.Enabled {
			_, _ = w.runtime.Disconnect(context.Background(), &spec.DisconnectMCPServerRequest{
				BundleID: req.BundleID,
				ServerID: req.ServerID,
			})
		}
		return resp, nil
	})
}

func (w *MCPWrapper) PatchMCPServerPolicy(
	req *spec.PatchMCPServerPolicyRequest,
) (*spec.PatchMCPServerPolicyResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchMCPServerPolicyResponse, error) {
		return w.store.PatchMCPServerPolicy(context.Background(), req)
	})
}

func (w *MCPWrapper) DeleteMCPServer(req *spec.DeleteMCPServerRequest) (*spec.DeleteMCPServerResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DeleteMCPServerResponse, error) {
		resp, err := w.store.DeleteMCPServer(context.Background(), req)
		if err != nil {
			return nil, err
		}
		if req != nil {
			_, _ = w.runtime.Disconnect(context.Background(), &spec.DisconnectMCPServerRequest{
				BundleID: req.BundleID,
				ServerID: req.ServerID,
			})
		}
		return resp, nil
	})
}

func (w *MCPWrapper) ConnectMCPServer(req *spec.ConnectMCPServerRequest) (*spec.ConnectMCPServerResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ConnectMCPServerResponse, error) {
		return w.runtime.Connect(context.Background(), req)
	})
}

func (w *MCPWrapper) DisconnectMCPServer(
	req *spec.DisconnectMCPServerRequest,
) (*spec.DisconnectMCPServerResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DisconnectMCPServerResponse, error) {
		return w.runtime.Disconnect(context.Background(), req)
	})
}

func (w *MCPWrapper) RefreshMCPServer(req *spec.RefreshMCPServerRequest) (*spec.RefreshMCPServerResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.RefreshMCPServerResponse, error) {
		return w.runtime.Refresh(context.Background(), req)
	})
}

func (w *MCPWrapper) GetMCPServerStatus(
	req *spec.GetMCPServerStatusRequest,
) (*spec.GetMCPServerStatusResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.GetMCPServerStatusResponse, error) {
		return w.runtime.Status(context.Background(), req)
	})
}

func (w *MCPWrapper) ListMCPServerTools(
	req *spec.ListMCPServerToolsRequest,
) (*spec.ListMCPServerToolsResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListMCPServerToolsResponse, error) {
		return w.runtime.ListTools(context.Background(), req)
	})
}

func (w *MCPWrapper) ListMCPServerResources(
	req *spec.ListMCPServerResourcesRequest,
) (*spec.ListMCPServerResourcesResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListMCPServerResourcesResponse, error) {
		return w.runtime.ListResources(context.Background(), req)
	})
}

func (w *MCPWrapper) ListMCPServerResourceTemplates(
	req *spec.ListMCPServerResourceTemplatesRequest,
) (*spec.ListMCPServerResourceTemplatesResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListMCPServerResourceTemplatesResponse, error) {
		return w.runtime.ListResourceTemplates(context.Background(), req)
	})
}

func (w *MCPWrapper) ListMCPServerPrompts(
	req *spec.ListMCPServerPromptsRequest,
) (*spec.ListMCPServerPromptsResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListMCPServerPromptsResponse, error) {
		return w.runtime.ListPrompts(context.Background(), req)
	})
}

func (w *MCPWrapper) ReadMCPResource(
	req *spec.MCPReadResourceRequest,
) (*spec.MCPReadResourceResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.MCPReadResourceResponse, error) {
		return w.runtime.ReadResource(context.Background(), req)
	})
}

func (w *MCPWrapper) GetMCPPrompt(req *spec.MCPGetPromptRequest) (*spec.MCPGetPromptResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.MCPGetPromptResponse, error) {
		return w.runtime.GetPrompt(context.Background(), req)
	})
}

func (w *MCPWrapper) CompleteMCPArgument(
	req *spec.MCPCompleteArgumentRequest,
) (*spec.MCPCompletionResult, error) {
	return middleware.WithRecoveryResp(func() (*spec.MCPCompletionResult, error) {
		return w.runtime.Complete(context.Background(), req)
	})
}

func (w *MCPWrapper) EvaluateMCPToolCall(
	req *spec.EvaluateMCPToolCallRequest,
) (*spec.EvaluateMCPToolCallResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.EvaluateMCPToolCallResponse, error) {
		return w.toolBridge.Evaluate(context.Background(), req)
	})
}

func (w *MCPWrapper) ResolveMCPApproval(
	req *spec.ResolveMCPApprovalRequest,
) (*spec.ResolveMCPApprovalResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ResolveMCPApprovalResponse, error) {
		if req == nil || req.Body == nil {
			return nil, spec.ErrMCPInvalidRequest
		}
		token, err := w.approvals.Resolve(context.Background(), req.Body.ApprovalID, req.Body.Resolution)
		if err != nil {
			return nil, err
		}
		return &spec.ResolveMCPApprovalResponse{Body: token}, nil
	})
}

func (w *MCPWrapper) InvokeMCPTool(req *spec.InvokeMCPToolRequest) (*spec.InvokeMCPToolResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.InvokeMCPToolResponse, error) {
		return w.toolBridge.Invoke(context.Background(), req)
	})
}

func (w *MCPWrapper) ListPendingMCPOAuthAuthorizations(
	req *spec.ListPendingMCPOAuthAuthorizationsRequest,
) (*spec.ListPendingMCPOAuthAuthorizationsResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListPendingMCPOAuthAuthorizationsResponse, error) {
		body := &spec.ListPendingMCPOAuthAuthorizationsResponseBody{
			Authorizations: []spec.MCPOAuthAuthorization{},
		}
		if w != nil && w.oauthBroker != nil {
			body.Authorizations = w.oauthBroker.Pending()
			if body.Authorizations == nil {
				body.Authorizations = []spec.MCPOAuthAuthorization{}
			}
		}
		return &spec.ListPendingMCPOAuthAuthorizationsResponse{Body: body}, nil
	})
}

func (w *MCPWrapper) CancelPendingMCPOAuthAuthorization(
	req *spec.CancelPendingMCPOAuthAuthorizationRequest,
) (*spec.CancelPendingMCPOAuthAuthorizationResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.CancelPendingMCPOAuthAuthorizationResponse, error) {
		if req == nil || req.BundleID == "" || req.ServerID == "" {
			return nil, fmt.Errorf("%w: bundleID and serverID required", spec.ErrMCPInvalidRequest)
		}
		if w != nil && w.oauthBroker != nil {
			_ = w.oauthBroker.Cancel(req.BundleID, req.ServerID)
		}
		return &spec.CancelPendingMCPOAuthAuthorizationResponse{}, nil
	})
}

func (w *MCPWrapper) GetMCPServerAuthHealth(
	req *spec.GetMCPServerAuthHealthRequest,
) (*spec.GetMCPServerAuthHealthResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.GetMCPServerAuthHealthResponse, error) {
		if req == nil || req.BundleID == "" || req.ServerID == "" {
			return nil, fmt.Errorf("%w: bundleID and serverID required", spec.ErrMCPInvalidRequest)
		}

		cfgResp, err := w.store.GetMCPServer(context.Background(), &spec.GetMCPServerRequest{
			BundleID: req.BundleID,
			ServerID: req.ServerID,
		})
		if err != nil {
			return nil, err
		}
		if cfgResp == nil || cfgResp.Body == nil {
			return nil, fmt.Errorf("%w: empty server config response", spec.ErrMCPRuntimeNotReady)
		}

		health := w.buildMCPAuthHealth(context.Background(), *cfgResp.Body)
		return &spec.GetMCPServerAuthHealthResponse{Body: health}, nil
	})
}

func (w *MCPWrapper) PutMCPServerSecret(
	req *spec.PutMCPServerSecretRequest,
) (*spec.PutMCPServerSecretResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PutMCPServerSecretResponse, error) {
		if req == nil || req.Body == nil || req.BundleID == "" || req.ServerID == "" {
			return nil, fmt.Errorf("%w: bundleID, serverID, and body required", spec.ErrMCPInvalidRequest)
		}
		if w == nil || w.secretWriter == nil {
			return nil, fmt.Errorf("%w: secret writer is not configured", spec.ErrMCPRuntimeNotReady)
		}

		cfgResp, err := w.store.GetMCPServer(context.Background(), &spec.GetMCPServerRequest{
			BundleID: req.BundleID,
			ServerID: req.ServerID,
		})
		if err != nil {
			return nil, err
		}
		if cfgResp == nil || cfgResp.Body == nil {
			return nil, fmt.Errorf("%w: empty server config response", spec.ErrMCPRuntimeNotReady)
		}
		secretRef, err := secret.NewMCPSecretRefString(req.BundleID, req.ServerID, req.Body.Kind, req.Body.Slot)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", spec.ErrMCPInvalidRequest, err)
		}

		if req.Body.Kind == spec.MCPSecretKindOAuthClientCredentials {
			requireClientSecret := false
			if cfgResp.Body.Transport == spec.MCPTransportStreamableHTTP &&
				cfgResp.Body.StreamableHTTP != nil &&
				cfgResp.Body.StreamableHTTP.AuthMode == spec.MCPHTTPAuthClientCredentials {
				requireClientSecret = true
			}

			if err := auth.ValidateOAuthClientCredentialsSecret(req.Body.Secret, requireClientSecret); err != nil {
				return nil, err
			}
		}

		sha, nonEmpty, err := w.secretWriter.SetMCPSecret(context.Background(), secretRef, req.Body.Secret)
		if err != nil {
			return nil, err
		}

		return &spec.PutMCPServerSecretResponse{
			Body: &spec.PutMCPServerSecretResponseBody{
				SecretRef: secretRef,
				SHA256:    sha,
				NonEmpty:  nonEmpty,
			},
		}, nil
	})
}

func (w *MCPWrapper) DeleteMCPServerSecret(
	req *spec.DeleteMCPServerSecretRequest,
) (*spec.DeleteMCPServerSecretResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DeleteMCPServerSecretResponse, error) {
		if req == nil || req.BundleID == "" || req.ServerID == "" || req.Kind == "" ||
			strings.TrimSpace(req.Slot) == "" {
			return nil, fmt.Errorf("%w: bundleID, serverID, kind, and slot required", spec.ErrMCPInvalidRequest)
		}
		if w == nil || w.secretWriter == nil {
			return nil, fmt.Errorf("%w: secret writer is not configured", spec.ErrMCPRuntimeNotReady)
		}

		if _, err := w.store.GetMCPServer(context.Background(), &spec.GetMCPServerRequest{
			BundleID: req.BundleID,
			ServerID: req.ServerID,
		}); err != nil {
			return nil, err
		}

		secretRef, err := secret.NewMCPSecretRefString(req.BundleID, req.ServerID, req.Kind, req.Slot)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", spec.ErrMCPInvalidRequest, err)
		}

		if err := w.secretWriter.DeleteMCPSecret(context.Background(), secretRef); err != nil {
			return nil, err
		}
		return &spec.DeleteMCPServerSecretResponse{}, nil
	})
}

func (w *MCPWrapper) disconnectBundleServers(ctx context.Context, bundleID bundleitemutils.BundleID) {
	if w == nil || w.store == nil || w.runtime == nil || bundleID == "" {
		return
	}
	resp, err := w.store.ListMCPServers(ctx, &spec.ListMCPServersRequest{
		BundleID:        bundleID,
		IncludeDisabled: true,
		PageSize:        spec.MaxMCPServerPageSize,
	})
	if err != nil || resp == nil || resp.Body == nil {
		return
	}
	for _, cfg := range resp.Body.Servers {
		_, _ = w.runtime.Disconnect(ctx, &spec.DisconnectMCPServerRequest{
			BundleID: cfg.BundleID,
			ServerID: cfg.ID,
		})
	}
}

func (w *MCPWrapper) buildMCPAuthHealth(
	ctx context.Context,
	cfg spec.MCPServerConfig,
) *spec.MCPAuthHealth {
	st := auth.DefaultMCPAuthStatusFromConfig(cfg)
	if w != nil && w.auth != nil {
		if cur, ok := w.auth.GetAuthStatus(cfg.BundleID, cfg.ID); ok {
			st = auth.MergeMCPAuthStatus(cur, cfg)
		}
	}

	health := &spec.MCPAuthHealth{
		BundleID:   cfg.BundleID,
		ServerID:   cfg.ID,
		AuthMode:   normalizeWrapperHTTPAuthMode(st.AuthMode),
		State:      spec.MCPAuthHealthStateAuthorizationNeeded,
		Configured: true,
		Resource:   st.Resource,
		Scopes:     st.Scopes,
		ExpiresAt:  st.ExpiresAt,
		LastError:  st.LastError,
	}

	switch cfg.Transport {
	case spec.MCPTransportStdio:
		health.AuthMode = spec.MCPHTTPAuthNone
		if missing, msg := w.firstMissingStdioSecret(ctx, cfg); missing {
			health.State = spec.MCPAuthHealthStateNotConfigured
			health.Configured = false
			health.LastError = msg
			return health
		}
		health.State = spec.MCPAuthHealthStateNotRequired
		return health

	case spec.MCPTransportStreamableHTTP:
		if cfg.StreamableHTTP == nil {
			health.State = spec.MCPAuthHealthStateNotConfigured
			health.Configured = false
			health.LastError = "missing streamableHttp config"
			return health
		}

	default:
		health.State = spec.MCPAuthHealthStateNotConfigured
		health.Configured = false
		health.LastError = "unsupported MCP transport"
		return health
	}

	httpCfg := cfg.StreamableHTTP
	mode := normalizeWrapperHTTPAuthMode(httpCfg.AuthMode)
	health.AuthMode = mode
	health.Resource = strings.TrimSpace(httpCfg.URL)

	switch mode {
	case spec.MCPHTTPAuthNone:
		health.State = spec.MCPAuthHealthStateNotRequired
		health.Configured = true
		health.LastError = ""
		return health

	case spec.MCPHTTPAuthOAuth:
		if w == nil || w.oauthBroker == nil || strings.TrimSpace(w.oauthBroker.RedirectURL()) == "" {
			health.State = spec.MCPAuthHealthStateNotConfigured
			health.Configured = false
			health.LastError = "OAuth authorization broker is not configured"
			return health
		}

		if ref := strings.TrimSpace(httpCfg.ClientCredentialRef); ref != "" {
			if ok, msg := w.oauthClientSecretConfigured(ctx, ref, false); !ok {
				health.State = spec.MCPAuthHealthStateNotConfigured
				health.Configured = false
				health.LastError = msg
				return health
			}
		}

		if pending, ok := w.pendingOAuthAuthorization(cfg.BundleID, cfg.ID); ok {
			health.State = spec.MCPAuthHealthStateAuthorizationPending
			health.Configured = true
			health.AuthorizationPending = true
			health.AuthorizationURL = pending.AuthorizationURL
			health.AuthorizationExpiresAt = pending.ExpiresAt
			return health
		}

	case spec.MCPHTTPAuthClientCredentials:
		ref := strings.TrimSpace(httpCfg.ClientCredentialRef)
		if ref == "" {
			health.State = spec.MCPAuthHealthStateNotConfigured
			health.Configured = false
			health.LastError = "streamableHttp.clientCredentialRef is required for clientCredentials auth"
			return health
		}
		if ok, msg := w.oauthClientSecretConfigured(ctx, ref, true); !ok {
			health.State = spec.MCPAuthHealthStateNotConfigured
			health.Configured = false
			health.LastError = msg
			return health
		}

	default:
		health.State = spec.MCPAuthHealthStateNotConfigured
		health.Configured = false
		health.LastError = "unsupported HTTP auth mode"
		return health
	}

	health.State = authHealthStateFromStatus(st)
	return health
}

func (w *MCPWrapper) firstMissingStdioSecret(
	ctx context.Context,
	cfg spec.MCPServerConfig,
) (missing bool, msg string) {
	if cfg.Stdio == nil || len(cfg.Stdio.SecretEnvRefs) == 0 {
		return false, ""
	}
	if w == nil || w.secretResolver == nil {
		return true, "secret resolver is not configured"
	}
	for key, ref := range cfg.Stdio.SecretEnvRefs {
		value, err := w.secretResolver.ResolveSecret(ctx, ref)
		if err != nil {
			return true, fmt.Sprintf("stdio secret %s is not configured: %v", key, err)
		}
		if strings.TrimSpace(value) == "" {
			return true, fmt.Sprintf("stdio secret %s is empty", key)
		}
	}
	return false, ""
}

func (w *MCPWrapper) oauthClientSecretConfigured(
	ctx context.Context,
	ref string,
	requireClientSecret bool,
) (ok bool, msg string) {
	if w == nil || w.secretResolver == nil {
		return false, "secret resolver is not configured"
	}
	raw, err := w.secretResolver.ResolveSecret(ctx, ref)
	if err != nil {
		return false, err.Error()
	}
	if strings.TrimSpace(raw) == "" {
		return false, "OAuth client credentials secret is empty"
	}
	if err := auth.ValidateOAuthClientCredentialsSecret(raw, requireClientSecret); err != nil {
		return false, err.Error()
	}
	return true, ""
}

func (w *MCPWrapper) pendingOAuthAuthorization(
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
) (spec.MCPOAuthAuthorization, bool) {
	if w == nil || w.oauthBroker == nil {
		return spec.MCPOAuthAuthorization{}, false
	}
	for _, pending := range w.oauthBroker.Pending() {
		if pending.BundleID == bundleID && pending.ServerID == serverID {
			return pending, true
		}
	}
	return spec.MCPOAuthAuthorization{}, false
}

func authHealthStateFromStatus(st spec.MCPAuthStatus) spec.MCPAuthHealthState {
	if st.ExpiresAt != nil && time.Now().UTC().After(st.ExpiresAt.UTC()) {
		return spec.MCPAuthHealthStateExpired
	}

	switch st.State {
	case spec.MCPAuthStateNotRequired:
		return spec.MCPAuthHealthStateNotRequired
	case spec.MCPAuthStateRequired:
		return spec.MCPAuthHealthStateAuthorizationNeeded
	case spec.MCPAuthStateAuthorized:
		return spec.MCPAuthHealthStateAuthorized
	case spec.MCPAuthStateExpired:
		return spec.MCPAuthHealthStateExpired
	case spec.MCPAuthStateInsufficientScope:
		return spec.MCPAuthHealthStateInsufficientScope
	case spec.MCPAuthStateError:
		return spec.MCPAuthHealthStateError
	default:
		return spec.MCPAuthHealthStateAuthorizationNeeded
	}
}

func normalizeWrapperHTTPAuthMode(mode spec.MCPHTTPAuthMode) spec.MCPHTTPAuthMode {
	mode = spec.MCPHTTPAuthMode(strings.TrimSpace(string(mode)))
	if mode == "" {
		return spec.MCPHTTPAuthNone
	}
	return mode
}

func (w *MCPWrapper) close() {
	if w == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if w.runtime != nil {
		_ = w.runtime.Close(ctx)
	}
	if w.oauthBroker != nil {
		_ = w.oauthBroker.Close()
	}
	if w.store != nil {
		_ = w.store.Close()
	}
}
