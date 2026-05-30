package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	mcpAuth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/auth/extauth"

	"github.com/modelcontextprotocol/go-sdk/oauthex"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

type ResolvedTransportAuth struct {
	Env             map[string]string
	SensitiveValues []string
	Status          spec.MCPAuthStatus
	OAuthHandler    mcpAuth.OAuthHandler
}

type AuthStatusSink interface {
	SaveAuthStatus(ctx context.Context, st spec.MCPAuthStatus) error
}

type OAuthAuthorizationRequest struct {
	ServerID         spec.MCPServerID
	AuthorizationURL string
}

type OAuthAuthorizationResult struct {
	Code  string
	State string
	Iss   string
}

type OAuthAuthorizationBroker interface {
	FetchAuthorizationCode(
		ctx context.Context,
		req OAuthAuthorizationRequest,
	) (*OAuthAuthorizationResult, error)
}

type AuthManager struct {
	secrets          SecretResolver
	oauthBroker      OAuthAuthorizationBroker
	oauthRedirectURL string
	httpClient       *http.Client

	mu       sync.RWMutex
	statuses map[spec.MCPServerID]spec.MCPAuthStatus
}

type AuthManagerOption func(*AuthManager)

func WithOAuthAuthorizationBroker(broker OAuthAuthorizationBroker) AuthManagerOption {
	return func(m *AuthManager) {
		m.oauthBroker = broker
	}
}

func WithOAuthRedirectURL(redirectURL string) AuthManagerOption {
	return func(m *AuthManager) {
		m.oauthRedirectURL = strings.TrimSpace(redirectURL)
	}
}

func WithAuthHTTPClient(client *http.Client) AuthManagerOption {
	return func(m *AuthManager) {
		m.httpClient = client
	}
}

func NewAuthManager(secrets SecretResolver, opts ...AuthManagerOption) *AuthManager {
	if secrets == nil {
		secrets = StaticSecretResolver{}
	}
	m := &AuthManager{
		secrets:  secrets,
		statuses: map[spec.MCPServerID]spec.MCPAuthStatus{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}
	return m
}

func (m *AuthManager) PrepareTransportAuth(
	ctx context.Context,
	cfg spec.MCPServerConfig,
) (ResolvedTransportAuth, error) {
	if m == nil {
		return ResolvedTransportAuth{
			Env: map[string]string{},
			Status: spec.MCPAuthStatus{
				ServerID: cfg.ID,
				AuthMode: spec.MCPHTTPAuthNone,
				State:    spec.MCPAuthStateNotRequired,
			},
		}, nil
	}
	out := ResolvedTransportAuth{
		Env: map[string]string{},
		Status: spec.MCPAuthStatus{
			ServerID: cfg.ID,
			AuthMode: spec.MCPHTTPAuthNone,
			State:    spec.MCPAuthStateNotRequired,
		},
	}

	saveCtx := ctx
	defer func() {
		_ = m.SaveAuthStatus(context.WithoutCancel(saveCtx), out.Status)
	}()

	switch cfg.Transport {
	case spec.MCPTransportStdio:
		if cfg.Stdio == nil {
			out.Status.State = spec.MCPAuthStateError
			out.Status.LastError = errStrMissingStdIOConfig
			return out, fmt.Errorf("%w: %s", spec.ErrMCPInvalidRequest, errStrMissingStdIOConfig)
		}
		for key, ref := range cfg.Stdio.SecretEnvRefs {
			v, err := m.secrets.ResolveSecret(ctx, ref)
			if err != nil {
				out.Status.State = spec.MCPAuthStateError
				out.Status.LastError = err.Error()
				return out, err
			}
			out.Env[key] = v
			out.SensitiveValues = append(out.SensitiveValues, v)
		}
		return out, nil
	case spec.MCPTransportStreamableHTTP:
		if cfg.StreamableHTTP == nil {
			out.Status.State = spec.MCPAuthStateError
			out.Status.LastError = "missing streamableHttp config"
			return out, fmt.Errorf("%w: missing streamableHttp config", spec.ErrMCPInvalidRequest)
		}
		httpCfg := cfg.StreamableHTTP

		out.Status.Resource = strings.TrimSpace(httpCfg.URL)
		mode := normalizeHTTPAuthMode(httpCfg.AuthMode)
		out.Status.AuthMode = mode

		switch mode {
		case spec.MCPHTTPAuthNone:
			out.Status.State = spec.MCPAuthStateNotRequired

		case spec.MCPHTTPAuthOAuth:
			if err := m.configureAuthorizationCodeOAuth(ctx, cfg, &out); err != nil {
				return out, err
			}
		case spec.MCPHTTPAuthClientCredentials:
			if err := m.configureClientCredentialsOAuth(ctx, cfg, &out); err != nil {
				return out, err
			}

		default:
			out.Status.State = spec.MCPAuthStateError
			out.Status.LastError = "unsupported auth mode"
			return out, fmt.Errorf("%w: unsupported auth mode %s", spec.ErrMCPInvalidRequest, mode)
		}

		return out, nil

	default:
		out.Status.State = spec.MCPAuthStateError
		out.Status.LastError = errStrUnsupportedTransport
		return out, fmt.Errorf("%w: unsupported transport %s", spec.ErrMCPInvalidRequest, cfg.Transport)
	}
}

func (m *AuthManager) configureAuthorizationCodeOAuth(
	ctx context.Context,
	cfg spec.MCPServerConfig,
	out *ResolvedTransportAuth,
) error {
	out.Status.State = spec.MCPAuthStateRequired

	if m.oauthBroker == nil || strings.TrimSpace(m.oauthRedirectURL) == "" {
		out.Status.LastError = "OAuth authorization code flow is not configured"
		return fmt.Errorf("%w: %s", spec.ErrMCPAuthRequired, out.Status.LastError)
	}

	httpCfg := cfg.StreamableHTTP

	var (
		clientIDMetadata *mcpAuth.ClientIDMetadataDocumentConfig
		preregistered    *oauthex.ClientCredentials
		dcr              *mcpAuth.DynamicClientRegistrationConfig
	)
	if rawURL := strings.TrimSpace(httpCfg.ClientIDMetadataDocumentURL); rawURL != "" {
		clientIDMetadata = &mcpAuth.ClientIDMetadataDocumentConfig{
			URL: rawURL,
		}
	}
	if ref := strings.TrimSpace(httpCfg.ClientCredentialRef); ref != "" {
		creds, sensitive, err := resolveOAuthClientCredentials(ctx, m.secrets, ref, false)
		if err != nil {
			out.Status.State = spec.MCPAuthStateError
			out.Status.LastError = err.Error()
			return err
		}
		preregistered = creds
		out.SensitiveValues = append(out.SensitiveValues, sensitive...)
	}
	if preregistered == nil {
		// Dynamic registration is only used when no preregistered client credentials were configured.
		// If a Client ID Metadata Document URL is also configured, the SDK will try it before falling back to DCR.
		dcr = &mcpAuth.DynamicClientRegistrationConfig{
			Metadata: &oauthex.ClientRegistrationMetadata{
				RedirectURIs:    []string{m.oauthRedirectURL},
				ClientName:      spec.MCPHostName,
				SoftwareID:      "flexigpt",
				SoftwareVersion: spec.MCPHostVersion,
				// Desktop clients are public clients. Requesting "none" avoids
				// receiving/storing a dynamically issued client secret.
				TokenEndpointAuthMethod: string(spec.MCPHTTPAuthNone),
				ResponseTypes:           []string{"code"},
				GrantTypes: []string{
					string(spec.GrantTypeAuthorizationCode),
					string(spec.GrantTypeRefreshToken),
				},
			},
		}
	}

	handler, err := mcpAuth.NewAuthorizationCodeHandler(&mcpAuth.AuthorizationCodeHandlerConfig{
		ClientIDMetadataDocumentConfig:  clientIDMetadata,
		PreregisteredClient:             preregistered,
		DynamicClientRegistrationConfig: dcr,
		RedirectURL:                     m.oauthRedirectURL,
		AuthorizationCodeFetcher: func(ctx context.Context, args *mcpAuth.AuthorizationArgs) (*mcpAuth.AuthorizationResult, error) {
			if args == nil || strings.TrimSpace(args.URL) == "" {
				return nil, fmt.Errorf("%w: missing OAuth authorization URL", spec.ErrMCPAuthRequired)
			}
			res, err := m.oauthBroker.FetchAuthorizationCode(ctx, OAuthAuthorizationRequest{
				ServerID:         cfg.ID,
				AuthorizationURL: args.URL,
			})
			if err != nil {
				return nil, err
			}
			if res == nil || strings.TrimSpace(res.Code) == "" {
				return nil, fmt.Errorf("%w: OAuth authorization code was not returned", spec.ErrMCPAuthRequired)
			}
			return &mcpAuth.AuthorizationResult{
				Code:  res.Code,
				State: res.State,
				Iss:   res.Iss,
			}, nil
		},
		// Keep OAuth tokens volatile and process-local, but still request refresh
		// token capability so token refresh can work correctly during this app
		// process. Nothing returned by the authorization server is persisted by
		// FlexiGPT. App restart forces reauthorization.
		//
		// The SDK stores the resulting oauth2.TokenSource in memory only.
		RequestRefreshToken: true,
		Client:              m.httpClient,
	})
	if err != nil {
		out.Status.State = spec.MCPAuthStateError
		out.Status.LastError = err.Error()
		return err
	}

	out.OAuthHandler = &trackedOAuthHandler{
		inner: handler,
		sink:  m,
		status: spec.MCPAuthStatus{
			ServerID: cfg.ID,
			AuthMode: spec.MCPHTTPAuthOAuth,
			State:    spec.MCPAuthStateRequired,
			Resource: out.Status.Resource,
		},
		sensitiveValues: append([]string(nil), out.SensitiveValues...),
	}
	return nil
}

func (m *AuthManager) configureClientCredentialsOAuth(
	ctx context.Context,
	cfg spec.MCPServerConfig,
	out *ResolvedTransportAuth,
) error {
	if cfg.StreamableHTTP == nil || strings.TrimSpace(cfg.StreamableHTTP.ClientCredentialRef) == "" {

		out.Status.State = spec.MCPAuthStateRequired
		out.Status.LastError = "streamableHttp.clientCredentialRef is required for clientCredentials auth"

		return fmt.Errorf("%w: %s", spec.ErrMCPAuthRequired, out.Status.LastError)
	}

	creds, sensitive, err := resolveOAuthClientCredentials(ctx, m.secrets, cfg.StreamableHTTP.ClientCredentialRef, true)
	if err != nil {
		out.Status.State = spec.MCPAuthStateError
		out.Status.LastError = err.Error()
		return err
	}

	handler, err := extauth.NewClientCredentialsHandler(&extauth.ClientCredentialsHandlerConfig{
		Credentials: creds,
		HTTPClient:  m.httpClient,
	})
	if err != nil {
		out.Status.State = spec.MCPAuthStateError
		out.Status.LastError = err.Error()
		return err
	}
	out.SensitiveValues = append(out.SensitiveValues, sensitive...)

	out.OAuthHandler = &trackedOAuthHandler{
		inner: handler,
		sink:  m,
		status: spec.MCPAuthStatus{
			ServerID: cfg.ID,
			AuthMode: spec.MCPHTTPAuthClientCredentials,
			State:    spec.MCPAuthStateRequired,
			Resource: out.Status.Resource,
		},
		sensitiveValues: append([]string(nil), out.SensitiveValues...),
	}

	out.Status.State = spec.MCPAuthStateRequired
	return nil
}

func parseOAuthClientCredentialsSecret(
	raw string,
	requireClientSecret bool,
) (*oauthex.ClientCredentials, []string, error) {
	var wire struct {
		ClientID     string `json:"clientID"`
		ClientSecret string `json:"clientSecret"`
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&wire); err != nil {
		return nil, nil, fmt.Errorf(
			"%w: OAuth client credentials secret must be a JSON object",
			spec.ErrMCPInvalidRequest,
		)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, nil, fmt.Errorf(
			"%w: OAuth client credentials secret must contain a single JSON object",
			spec.ErrMCPInvalidRequest,
		)
	}
	if strings.TrimSpace(wire.ClientID) == "" {
		return nil, nil, fmt.Errorf("%w: OAuth client credentials secret requires clientID", spec.ErrMCPInvalidRequest)
	}
	if strings.TrimSpace(wire.ClientID) != wire.ClientID {
		return nil, nil, fmt.Errorf(
			"%w: OAuth client credentials clientID must not have leading/trailing whitespace",
			spec.ErrMCPInvalidRequest,
		)
	}

	if wire.ClientSecret != "" && strings.TrimSpace(wire.ClientSecret) == "" {
		return nil, nil, fmt.Errorf(
			"%w: OAuth client credentials clientSecret must not be only whitespace",
			spec.ErrMCPInvalidRequest,
		)
	}
	if requireClientSecret && strings.TrimSpace(wire.ClientSecret) == "" {
		return nil, nil, fmt.Errorf(
			"%w: OAuth client credentials secret requires clientSecret",
			spec.ErrMCPInvalidRequest,
		)
	}
	creds := &oauthex.ClientCredentials{
		ClientID: wire.ClientID,
	}
	if wire.ClientSecret != "" {
		creds.ClientSecretAuth = &oauthex.ClientSecretAuth{
			ClientSecret: wire.ClientSecret,
		}
	}
	if err := creds.Validate(); err != nil {
		return nil, nil, fmt.Errorf("%w: %w", spec.ErrMCPInvalidRequest, err)
	}

	sensitive := make([]string, 0, 1)
	if wire.ClientSecret != "" {
		sensitive = append(sensitive, wire.ClientSecret)
	}
	return creds, sensitive, nil
}

func normalizeHTTPAuthMode(mode spec.MCPHTTPAuthMode) spec.MCPHTTPAuthMode {
	mode = spec.MCPHTTPAuthMode(strings.TrimSpace(string(mode)))
	if mode == "" {
		return spec.MCPHTTPAuthNone
	}
	return mode
}
