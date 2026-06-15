package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
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

const secretResolverNotConfiguredReason = "secret resolver is not configured"

type mcpAuthKeyReader interface {
	GetAuthKey(ctx context.Context, req *settingSpec.GetAuthKeyRequest) (*settingSpec.GetAuthKeyResponse, error)
}
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

	settings, err := st.GetMCPSettings(ctx)
	if err != nil {
		_ = st.Close()
		return err
	}

	var opts auth.OAuthLoopbackBrokerOptions
	if settings != nil && strings.TrimSpace(settings.OAuthLoopbackListenAddr) != "" {
		opts = auth.OAuthLoopbackBrokerOptions{
			ListenAddr: settings.OAuthLoopbackListenAddr,
		}
	}

	oauthBroker, err := auth.NewOAuthLoopbackBroker(ctx, &opts)
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
		var previous *spec.MCPServerConfig
		ctx := context.Background()
		if req != nil && req.BundleID != "" && req.ServerID != "" && w != nil && w.store != nil {
			oldResp, oldErr := w.store.GetMCPServer(ctx, &spec.GetMCPServerRequest{
				BundleID: req.BundleID,
				ServerID: req.ServerID,
			})
			if oldErr == nil && oldResp != nil && oldResp.Body != nil {
				old := *oldResp.Body
				previous = &old
			}
		}

		if req != nil &&
			req.Body != nil &&
			req.Body.StreamableHTTP != nil &&
			req.Body.StreamableHTTP.AuthMode == spec.MCPHTTPAuthClientCredentials {
			ref := strings.TrimSpace(req.Body.StreamableHTTP.ClientCredentialRef)
			if ref == "" {
				return nil, fmt.Errorf(
					"%w: streamableHttp.clientCredentialRef is required for clientCredentials auth",
					spec.ErrMCPInvalidRequest,
				)
			}
			if ok, msg := w.oauthClientSecretConfigured(ctx, ref, true); !ok {
				return nil, fmt.Errorf("%w: %s", spec.ErrMCPInvalidRequest, msg)
			}
		}

		resp, err := w.store.PutMCPServer(ctx, req)
		if err != nil {
			return nil, err
		}
		if shouldForgetMCPServerSnapshotAfterPut(previous, req) && w != nil && w.runtime != nil {
			w.runtime.ForgetLastKnownSnapshot(ctx, req.BundleID, req.ServerID)
		}

		if shouldDisconnectMCPServerAfterPut(previous, req) {
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
		ctx := context.Background()
		resp, err := w.store.PatchMCPServerEnabled(ctx, req)
		if err != nil {
			return nil, err
		}
		if req != nil && req.Body != nil && !req.Body.Enabled {
			if w != nil && w.runtime != nil {
				w.runtime.ForgetLastKnownSnapshot(ctx, req.BundleID, req.ServerID)
			}
			_, _ = w.runtime.Disconnect(ctx, &spec.DisconnectMCPServerRequest{
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
		ctx := context.Background()
		resp, err := w.store.DeleteMCPServer(ctx, req)
		if err != nil {
			return nil, err
		}
		if req != nil && w != nil && w.runtime != nil {
			w.runtime.ForgetLastKnownSnapshot(ctx, req.BundleID, req.ServerID)
		}
		if req != nil {
			_, _ = w.runtime.Disconnect(ctx, &spec.DisconnectMCPServerRequest{
				BundleID: req.BundleID,
				ServerID: req.ServerID,
			})
		}
		return resp, nil
	})
}

func (w *MCPWrapper) PatchMCPServerSetup(
	req *spec.PatchMCPServerSetupRequest,
) (*spec.PatchMCPServerSetupResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchMCPServerSetupResponse, error) {
		ctx := context.Background()
		if req == nil || req.Body == nil || req.BundleID == "" || req.ServerID == "" {
			return nil, fmt.Errorf("%w: bundleID, serverID, and body required", spec.ErrMCPInvalidRequest)
		}

		cfgResp, err := w.store.GetMCPServer(ctx, &spec.GetMCPServerRequest{
			BundleID: req.BundleID,
			ServerID: req.ServerID,
		})
		if err != nil {
			return nil, err
		}
		if cfgResp == nil || cfgResp.Body == nil {
			return nil, fmt.Errorf("%w: empty server config response", spec.ErrMCPRuntimeNotReady)
		}
		cfg := *cfgResp.Body

		if len(req.Body.InputValues) > 0 && (w == nil || w.secretWriter == nil) {
			// Only fails if a secret-bearing input is present; checked again below.
			if setupNeedsSecretWriter(cfg, req.Body.InputValues) {
				return nil, fmt.Errorf("%w: secret writer is not configured", spec.ErrMCPRuntimeNotReady)
			}
		}

		overlay, _, err := w.buildSetupOverlay(ctx, cfg, req.Body.InputValues)
		if err != nil {
			return nil, err
		}

		var updated *spec.MCPServerConfig
		if cfg.IsBuiltIn {
			updated, err = w.store.ApplyBuiltInServerSetupOverlay(
				ctx,
				req.BundleID,
				req.ServerID,
				overlay,
				req.Body.Reset,
			)
			if err != nil {
				return nil, err
			}
		} else {
			if req.Body.Reset {
				return nil, fmt.Errorf("%w: reset is only supported for built-in servers", spec.ErrMCPInvalidRequest)
			}
			updated, err = w.store.ApplyUserServerSetupOverlay(ctx, req.BundleID, req.ServerID, overlay)
			if err != nil {
				return nil, err
			}
		}

		if w.runtime != nil {
			w.runtime.ForgetLastKnownSnapshot(ctx, req.BundleID, req.ServerID)
			_, _ = w.runtime.Disconnect(ctx, &spec.DisconnectMCPServerRequest{
				BundleID: req.BundleID,
				ServerID: req.ServerID,
			})
		}
		return &spec.PatchMCPServerSetupResponse{Body: updated}, nil
	})
}

func (w *MCPWrapper) PatchMCPSettings(
	req *spec.PatchMCPSettingsRequest,
) (*spec.PatchMCPSettingsResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchMCPSettingsResponse, error) {
		ctx := context.Background()
		settings, err := w.store.PatchMCPSettings(ctx, req)
		if err != nil {
			return nil, err
		}
		return &spec.PatchMCPSettingsResponse{Body: w.buildMCPSettingsView(settings)}, nil
	})
}

func (w *MCPWrapper) GetMCPSettings(
	req *spec.GetMCPSettingsRequest,
) (*spec.GetMCPSettingsResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.GetMCPSettingsResponse, error) {
		settings, err := w.store.GetMCPSettings(context.Background())
		if err != nil {
			return nil, err
		}
		return &spec.GetMCPSettingsResponse{Body: w.buildMCPSettingsView(settings)}, nil
	})
}

func (w *MCPWrapper) ConnectMCPServer(req *spec.ConnectMCPServerRequest) (*spec.ConnectMCPServerResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ConnectMCPServerResponse, error) {
		ctx := context.Background()
		if req == nil || req.BundleID == "" || req.ServerID == "" {
			return nil, fmt.Errorf("%w: bundleID and serverID required", spec.ErrMCPInvalidRequest)
		}

		cfgResp, err := w.store.GetMCPServer(ctx, &spec.GetMCPServerRequest{
			BundleID: req.BundleID,
			ServerID: req.ServerID,
		})
		if err != nil {
			return nil, err
		}
		if cfgResp == nil || cfgResp.Body == nil {
			return nil, fmt.Errorf("%w: empty server config response", spec.ErrMCPRuntimeNotReady)
		}
		if input, ok := firstUnconfiguredRequiredSetupInput(*cfgResp.Body); ok {
			label := input.Label
			if strings.TrimSpace(label) == "" {
				label = input.ID
			}
			return nil, fmt.Errorf(
				"%w: setup input %q (%s) must be configured before connecting",
				spec.ErrMCPInvalidRequest,
				input.ID,
				label,
			)
		}

		return w.runtime.Connect(ctx, req)
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
		if w.runtime != nil {
			w.runtime.ForgetLastKnownSnapshot(ctx, cfg.BundleID, cfg.ID)
		}
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
		if input, ok := firstUnconfiguredRequiredSetupInput(cfg); ok {
			health.State = spec.MCPAuthHealthStateNotConfigured
			health.Configured = false
			health.LastError = fmt.Sprintf(
				"required setup input %q is not configured",
				input.ID,
			)
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

	if input, ok := firstUnconfiguredRequiredSetupInput(cfg); ok {
		health.State = spec.MCPAuthHealthStateNotConfigured
		health.Configured = false
		health.LastError = fmt.Sprintf(
			"required setup input %q is not configured",
			input.ID,
		)
		return health
	}

	if w != nil && w.oauthBroker != nil {
		health.OAuthRedirectURL = w.oauthBroker.RedirectURL()
		health.OAuthLoopbackListenAddr = w.oauthBroker.ListenAddr()
	}
	if missing, msg := w.firstMissingHTTPHeaderSecret(ctx, cfg); missing {
		health.State = spec.MCPAuthHealthStateNotConfigured
		health.Configured = false
		health.LastError = msg
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
	case spec.MCPHTTPAuthAPIKey:
		if len(httpCfg.SecretHeaderRefs) == 0 {
			health.State = spec.MCPAuthHealthStateNotConfigured
			health.Configured = false
			health.LastError = "API key is not configured"
			return health
		}
		health.State = spec.MCPAuthHealthStateAuthorized
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

func (w *MCPWrapper) firstMissingHTTPHeaderSecret(
	ctx context.Context,
	cfg spec.MCPServerConfig,
) (missing bool, reason string) {
	if cfg.StreamableHTTP == nil || len(cfg.StreamableHTTP.SecretHeaderRefs) == 0 {
		return false, ""
	}
	if w == nil || w.secretResolver == nil {
		return true, secretResolverNotConfiguredReason
	}
	for header, ref := range cfg.StreamableHTTP.SecretHeaderRefs {
		value, err := w.secretResolver.ResolveSecret(ctx, ref)
		if err != nil {
			return true, fmt.Sprintf("HTTP header secret %s is not configured: %v", header, err)
		}
		if strings.TrimSpace(value) == "" {
			return true, fmt.Sprintf("HTTP header secret %s is empty", header)
		}
	}
	return false, ""
}

func (w *MCPWrapper) firstMissingStdioSecret(
	ctx context.Context,
	cfg spec.MCPServerConfig,
) (missing bool, msg string) {
	if cfg.Stdio == nil || len(cfg.Stdio.SecretEnvRefs) == 0 {
		return false, ""
	}
	if w == nil || w.secretResolver == nil {
		return true, secretResolverNotConfiguredReason
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
		return false, secretResolverNotConfiguredReason
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

// buildSetupOverlay converts declared input values into an overlay fragment with secret refs. "directHeaders" is the
// set of plain (non-secret) header values, used only for the user-server path where we update the config directly.
func (w *MCPWrapper) buildSetupOverlay(
	ctx context.Context,
	cfg spec.MCPServerConfig,
	values map[string]spec.MCPServerSetupInputValue,
) (spec.MCPBuiltInServerOverlay, map[string]string, error) {
	var overlay spec.MCPBuiltInServerOverlay
	directHeaders := map[string]string{}

	if len(values) == 0 {
		return overlay, directHeaders, nil
	}
	if cfg.Setup == nil {
		return overlay, directHeaders, fmt.Errorf("%w: server declares no setup inputs", spec.ErrMCPInvalidRequest)
	}

	declared := map[string]spec.MCPServerSetupInput{}
	for _, input := range cfg.Setup.Inputs {
		declared[input.ID] = input
	}
	for id := range values {
		if _, ok := declared[id]; !ok {
			return overlay, directHeaders, fmt.Errorf("%w: unknown setup input %q", spec.ErrMCPInvalidRequest, id)
		}
	}

	ensureHTTP := func() *spec.MCPStreamableHTTPConfigOverlay {
		if overlay.StreamableHTTP == nil {
			overlay.StreamableHTTP = &spec.MCPStreamableHTTPConfigOverlay{}
		}
		return overlay.StreamableHTTP
	}
	ensureStdio := func() *spec.MCPStdioConfigOverlay {
		if overlay.Stdio == nil {
			overlay.Stdio = &spec.MCPStdioConfigOverlay{}
		}
		return overlay.Stdio
	}

	for _, input := range cfg.Setup.Inputs {
		v, ok := values[input.ID]
		if !ok {
			if input.Required && !setupInputConfigured(cfg, input) {
				return overlay, directHeaders, fmt.Errorf(
					"%w: setup input %q is required",
					spec.ErrMCPInvalidRequest,
					input.ID,
				)
			}
			continue
		}

		switch input.Kind {
		case spec.MCPSetupKindOAuthClientCredentials:
			if strings.TrimSpace(v.ClientID) == "" {
				return overlay, directHeaders, fmt.Errorf(
					"%w: input %q requires clientID",
					spec.ErrMCPInvalidRequest,
					input.ID,
				)
			}
			if input.OAuthClientCredentials != nil && input.OAuthClientCredentials.ClientSecretRequired &&
				strings.TrimSpace(v.ClientSecret) == "" {
				return overlay, directHeaders, fmt.Errorf(
					"%w: input %q requires clientSecret",
					spec.ErrMCPInvalidRequest,
					input.ID,
				)
			}
			ref, err := w.storeOAuthClientCredentials(ctx, cfg, v.ClientID, v.ClientSecret)
			if err != nil {
				return overlay, directHeaders, err
			}
			ensureHTTP().ClientCredentialRef = &ref

		case spec.MCPSetupKindHTTPHeader:
			if input.HTTPHeader == nil {
				return overlay, directHeaders, fmt.Errorf(
					"%w: input %q missing httpHeader block",
					spec.ErrMCPInvalidRequest,
					input.ID,
				)
			}
			full := input.HTTPHeader.ValuePrefix + v.Value + input.HTTPHeader.ValueSuffix
			if input.HTTPHeader.Secret {
				if strings.TrimSpace(v.Value) == "" {
					return overlay, directHeaders, fmt.Errorf(
						"%w: input %q requires value",
						spec.ErrMCPInvalidRequest,
						input.ID,
					)
				}
				ref, err := w.storeHeaderSecret(ctx, cfg, input.HTTPHeader.HeaderName, full)
				if err != nil {
					return overlay, directHeaders, err
				}
				h := ensureHTTP()
				if h.SecretHeaderRefs == nil {
					h.SecretHeaderRefs = map[string]string{}
				}
				h.SecretHeaderRefs[input.HTTPHeader.HeaderName] = ref
			} else {
				h := ensureHTTP()
				if h.Headers == nil {
					h.Headers = map[string]string{}
				}
				h.Headers[input.HTTPHeader.HeaderName] = full
				directHeaders[input.HTTPHeader.HeaderName] = full
			}

		case spec.MCPSetupKindStdioEnv:
			if input.StdioEnv == nil {
				return overlay, directHeaders, fmt.Errorf(
					"%w: input %q missing stdioEnv block",
					spec.ErrMCPInvalidRequest,
					input.ID,
				)
			}
			full := input.StdioEnv.ValuePrefix + v.Value + input.StdioEnv.ValueSuffix
			if input.StdioEnv.Secret {
				if strings.TrimSpace(v.Value) == "" {
					return overlay, directHeaders, fmt.Errorf(
						"%w: input %q requires value",
						spec.ErrMCPInvalidRequest,
						input.ID,
					)
				}
				ref, err := w.storeStdioEnvSecret(ctx, cfg, input.StdioEnv.EnvName, full)
				if err != nil {
					return overlay, directHeaders, err
				}
				st := ensureStdio()
				if st.SecretEnvRefs == nil {
					st.SecretEnvRefs = map[string]string{}
				}
				st.SecretEnvRefs[input.StdioEnv.EnvName] = ref
			} else {
				st := ensureStdio()
				if st.Env == nil {
					st.Env = map[string]string{}
				}
				st.Env[input.StdioEnv.EnvName] = full
			}

		case spec.MCPSetupKindStreamableHTTPURL:
			if strings.TrimSpace(v.Value) == "" {
				return overlay, directHeaders, fmt.Errorf(
					"%w: input %q requires value",
					spec.ErrMCPInvalidRequest,
					input.ID,
				)
			}
			value := v.Value
			ensureHTTP().URL = &value

		case spec.MCPSetupKindClientIDMetadataDocURL:
			if strings.TrimSpace(v.Value) == "" {
				return overlay, directHeaders, fmt.Errorf(
					"%w: input %q requires value",
					spec.ErrMCPInvalidRequest,
					input.ID,
				)
			}
			value := v.Value
			ensureHTTP().ClientIDMetadataDocumentURL = &value

		default:
			return overlay, directHeaders, fmt.Errorf(
				"%w: unsupported setup input kind %q",
				spec.ErrMCPInvalidRequest,
				input.Kind,
			)
		}
	}

	return overlay, directHeaders, nil
}

func firstUnconfiguredRequiredSetupInput(cfg spec.MCPServerConfig) (spec.MCPServerSetupInput, bool) {
	if cfg.Setup == nil {
		return spec.MCPServerSetupInput{}, false
	}
	for _, input := range cfg.Setup.Inputs {
		if !input.Required {
			continue
		}
		if !setupInputConfigured(cfg, input) {
			return input, true
		}
	}
	return spec.MCPServerSetupInput{}, false
}

func setupInputConfigured(cfg spec.MCPServerConfig, input spec.MCPServerSetupInput) bool {
	switch input.Kind {
	case spec.MCPSetupKindOAuthClientCredentials:
		return cfg.StreamableHTTP != nil &&
			strings.TrimSpace(cfg.StreamableHTTP.ClientCredentialRef) != ""

	case spec.MCPSetupKindHTTPHeader:
		if cfg.StreamableHTTP == nil || input.HTTPHeader == nil {
			return false
		}
		header := input.HTTPHeader.HeaderName
		if input.HTTPHeader.Secret {
			return hasStringMapKeyFold(cfg.StreamableHTTP.SecretHeaderRefs, header)
		}
		return hasStringMapKeyFold(cfg.StreamableHTTP.Headers, header)

	case spec.MCPSetupKindStdioEnv:
		if cfg.Stdio == nil || input.StdioEnv == nil {
			return false
		}
		env := input.StdioEnv.EnvName
		if input.StdioEnv.Secret {
			_, ok := cfg.Stdio.SecretEnvRefs[env]
			return ok
		}
		_, ok := cfg.Stdio.Env[env]
		return ok

	case spec.MCPSetupKindStreamableHTTPURL:
		return cfg.StreamableHTTP != nil &&
			strings.TrimSpace(cfg.StreamableHTTP.URL) != ""

	case spec.MCPSetupKindClientIDMetadataDocURL:
		return cfg.StreamableHTTP != nil &&
			strings.TrimSpace(cfg.StreamableHTTP.ClientIDMetadataDocumentURL) != ""

	default:
		return false
	}
}

func hasStringMapKeyFold(m map[string]string, key string) bool {
	for k := range m {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			return true
		}
	}
	return false
}

func setupNeedsSecretWriter(cfg spec.MCPServerConfig, values map[string]spec.MCPServerSetupInputValue) bool {
	if cfg.Setup == nil {
		return false
	}
	for _, input := range cfg.Setup.Inputs {
		if _, ok := values[input.ID]; !ok {
			continue
		}
		switch input.Kind {
		case spec.MCPSetupKindOAuthClientCredentials:
			return true
		case spec.MCPSetupKindHTTPHeader:
			if input.HTTPHeader != nil && input.HTTPHeader.Secret {
				return true
			}
		case spec.MCPSetupKindStdioEnv:
			if input.StdioEnv != nil && input.StdioEnv.Secret {
				return true
			}
		default:
		}
	}
	return false
}

func (w *MCPWrapper) storeOAuthClientCredentials(
	ctx context.Context,
	cfg spec.MCPServerConfig,
	clientID, clientSecret string,
) (string, error) {
	//nolint:gosec // Type struct.
	raw, err := json.Marshal(struct {
		ClientID     string `json:"clientID"`
		ClientSecret string `json:"clientSecret,omitempty"`
	}{ClientID: clientID, ClientSecret: clientSecret})
	if err != nil {
		return "", err
	}
	requireSecret := cfg.StreamableHTTP != nil &&
		cfg.StreamableHTTP.AuthMode == spec.MCPHTTPAuthClientCredentials
	if err := auth.ValidateOAuthClientCredentialsSecret(string(raw), requireSecret); err != nil {
		return "", err
	}
	ref, err := secret.NewMCPSecretRefString(
		cfg.BundleID,
		cfg.ID,
		spec.MCPSecretKindOAuthClientCredentials,
		"clientCredentials",
	)
	if err != nil {
		return "", err
	}
	if _, _, err := w.secretWriter.SetMCPSecret(ctx, ref, string(raw)); err != nil {
		return "", err
	}
	return ref, nil
}

func (w *MCPWrapper) storeHeaderSecret(
	ctx context.Context,
	cfg spec.MCPServerConfig,
	header, value string,
) (string, error) {
	ref, err := secret.NewMCPSecretRefString(cfg.BundleID, cfg.ID, spec.MCPSecretKindHTTPHeader, header)
	if err != nil {
		return "", err
	}
	if _, _, err := w.secretWriter.SetMCPSecret(ctx, ref, value); err != nil {
		return "", err
	}
	return ref, nil
}

func (w *MCPWrapper) storeStdioEnvSecret(
	ctx context.Context,
	cfg spec.MCPServerConfig,
	env, value string,
) (string, error) {
	ref, err := secret.NewMCPSecretRefString(cfg.BundleID, cfg.ID, spec.MCPSecretKindStdioEnv, env)
	if err != nil {
		return "", err
	}
	if _, _, err := w.secretWriter.SetMCPSecret(ctx, ref, value); err != nil {
		return "", err
	}
	return ref, nil
}

func (w *MCPWrapper) buildMCPSettingsView(settings *spec.MCPSettings) *spec.MCPSettingsView {
	if settings == nil {
		settings = &spec.MCPSettings{}
	}
	view := &spec.MCPSettingsView{Settings: *settings}
	if w != nil && w.oauthBroker != nil {
		view.OAuthRedirectURL = w.oauthBroker.RedirectURL()
		requested := strings.TrimSpace(settings.OAuthLoopbackListenAddr)
		current := strings.TrimSpace(w.oauthBroker.ListenAddr())
		view.OAuthRestartRequired = requested != "" && requested != current
	}
	return view
}

func shouldForgetMCPServerSnapshotAfterPut(previous *spec.MCPServerConfig, req *spec.PutMCPServerRequest) bool {
	if req == nil || req.Body == nil || req.BundleID == "" || req.ServerID == "" {
		return false
	}
	if !req.Body.Enabled {
		return true
	}
	return mcpServerConnectionMaterialChanged(previous, req)
}

func shouldDisconnectMCPServerAfterPut(previous *spec.MCPServerConfig, req *spec.PutMCPServerRequest) bool {
	if req == nil || req.Body == nil || req.BundleID == "" || req.ServerID == "" {
		return false
	}

	if !req.Body.Enabled {
		return true
	}

	if previous == nil {
		return false
	}

	return mcpServerConnectionMaterialChanged(previous, req)
}

func mcpServerConnectionMaterialChanged(previous *spec.MCPServerConfig, req *spec.PutMCPServerRequest) bool {
	if previous == nil || req == nil || req.Body == nil || req.BundleID == "" || req.ServerID == "" {
		return false
	}
	return previous.Transport != req.Body.Transport ||
		!reflect.DeepEqual(previous.Stdio, req.Body.Stdio) ||
		!reflect.DeepEqual(previous.StreamableHTTP, req.Body.StreamableHTTP) ||
		!reflect.DeepEqual(previous.AppsPolicy, req.Body.AppsPolicy)
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
