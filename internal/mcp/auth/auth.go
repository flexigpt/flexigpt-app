package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strings"

	mcpAuth "github.com/modelcontextprotocol/go-sdk/auth"
	mcpExtAuth "github.com/modelcontextprotocol/go-sdk/auth/extauth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	maxResolvedHTTPHeaderValueLen = 4096
	oauthClientCredentialsSlot    = "clientCredentials"
)

type ResolvedTransportAuth struct {
	Headers         map[string]string
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
	statusSink       AuthStatusSink
	oauthBroker      OAuthAuthorizationBroker
	oauthRedirectURL string
	httpClient       *http.Client
}

type AuthManagerOption func(*AuthManager)

func WithAuthStatusSink(sink AuthStatusSink) AuthManagerOption {
	return func(m *AuthManager) {
		m.statusSink = sink
	}
}

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
	m := &AuthManager{secrets: secrets}
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
	out := ResolvedTransportAuth{
		Headers: map[string]string{},
		Env:     map[string]string{},
		Status: spec.MCPAuthStatus{
			ServerID: cfg.ID,
			AuthMode: spec.MCPHTTPAuthNone,
			State:    spec.MCPAuthStateNotRequired,
		},
	}

	if cfg.Transport == spec.MCPTransportStdio && cfg.Stdio != nil {
		out.Env = cloneStringMapNonNil(cfg.Stdio.Env)
		for key, ref := range cfg.Stdio.SecretEnvRefs {
			v, err := m.secrets.ResolveSecret(ctx, ref)
			if err != nil {
				return out, err
			}
			out.Env[key] = v
			out.SensitiveValues = append(out.SensitiveValues, v)
		}
		return out, nil
	}

	if cfg.Transport != spec.MCPTransportStreamableHTTP || cfg.StreamableHTTP == nil {
		return out, nil
	}

	httpCfg := cfg.StreamableHTTP
	out.Headers = cloneStringMapNonNil(httpCfg.CustomHeaders)
	for key, ref := range httpCfg.SecretHeaderRefs {
		v, err := m.secrets.ResolveSecret(ctx, ref)
		if err != nil {
			return out, err
		}
		out.Headers[key] = v
		out.SensitiveValues = append(out.SensitiveValues, v)
	}

	mode := normalizeHTTPAuthMode(httpCfg.AuthMode)
	out.Status.AuthMode = mode
	if cfg.AuthRef != nil {
		authMode := spec.MCPHTTPAuthMode(strings.TrimSpace(string(cfg.AuthRef.AuthMode)))
		if authMode != "" && authMode != mode {
			out.Status.State = spec.MCPAuthStateError
			out.Status.LastError = fmt.Sprintf(
				"authRef.authMode %q does not match streamableHttp.authMode %q",
				authMode,
				mode,
			)
			return out, fmt.Errorf("%w: %s", spec.ErrMCPInvalidRequest, out.Status.LastError)
		}
	}

	switch mode {
	case spec.MCPHTTPAuthNone:

		out.Status.State = spec.MCPAuthStateNotRequired

	case spec.MCPHTTPAuthCustomBearer:
		if cfg.AuthRef == nil || strings.TrimSpace(cfg.AuthRef.TokenRef) == "" {
			out.Status.State = spec.MCPAuthStateRequired
			return out, spec.ErrMCPAuthRequired
		}
		token, err := m.secrets.ResolveSecret(ctx, cfg.AuthRef.TokenRef)
		if err != nil {
			out.Status.State = spec.MCPAuthStateError
			out.Status.LastError = err.Error()
			return out, err
		}
		out.OAuthHandler = &trackedOAuthHandler{
			inner: &staticBearerOAuthHandler{token: token},
			sink:  m.statusSink,
			status: spec.MCPAuthStatus{
				ServerID: cfg.ID,
				AuthMode: mode,
				State:    spec.MCPAuthStateAuthorized,
			},
		}
		out.SensitiveValues = append(out.SensitiveValues, token)
		out.Status.State = spec.MCPAuthStateAuthorized

	case spec.MCPHTTPAuthCustomHeaders:
		out.Status.State = spec.MCPAuthStateAuthorized

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
	if err := validateResolvedHTTPHeaders(out.Headers); err != nil {
		out.Status.State = spec.MCPAuthStateError
		out.Status.LastError = err.Error()
		return out, err
	}
	return out, nil
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

	var preregistered *oauthex.ClientCredentials
	if cfg.AuthRef != nil && strings.TrimSpace(cfg.AuthRef.ClientCredentialRef) != "" {
		raw, err := m.secrets.ResolveSecret(ctx, cfg.AuthRef.ClientCredentialRef)
		if err != nil {
			out.Status.State = spec.MCPAuthStateError
			out.Status.LastError = err.Error()
			return err
		}
		creds, sensitive, err := parseOAuthClientCredentialsSecret(raw, false)
		if err != nil {
			out.Status.State = spec.MCPAuthStateError
			out.Status.LastError = err.Error()
			return err
		}
		preregistered = creds
		out.SensitiveValues = append(out.SensitiveValues, raw)
		out.SensitiveValues = append(out.SensitiveValues, sensitive...)
	}

	var dcr *mcpAuth.DynamicClientRegistrationConfig
	if preregistered == nil {
		dcr = &mcpAuth.DynamicClientRegistrationConfig{
			Metadata: &oauthex.ClientRegistrationMetadata{
				RedirectURIs:    []string{m.oauthRedirectURL},
				ClientName:      spec.MCPHostName,
				SoftwareID:      "flexigpt",
				SoftwareVersion: spec.MCPHostVersion,
				// Desktop clients are public clients. Requesting "none" avoids
				// receiving/storing a dynamically issued client secret. If a server
				// requires a confidential client, users can configure a pre-registered
				// client credential ref.
				TokenEndpointAuthMethod: "none",
				ResponseTypes:           []string{"code"},
				GrantTypes:              []string{"authorization_code", "refresh_token"},
			},
		}
	}

	handler, err := mcpAuth.NewAuthorizationCodeHandler(&mcpAuth.AuthorizationCodeHandlerConfig{
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
		sink:  m.statusSink,
		status: spec.MCPAuthStatus{
			ServerID: cfg.ID,
			AuthMode: spec.MCPHTTPAuthOAuth,
			State:    spec.MCPAuthStateRequired,
		},
	}
	return nil
}

func (m *AuthManager) configureClientCredentialsOAuth(
	ctx context.Context,
	cfg spec.MCPServerConfig,
	out *ResolvedTransportAuth,
) error {
	if cfg.AuthRef == nil || strings.TrimSpace(cfg.AuthRef.ClientCredentialRef) == "" {
		out.Status.State = spec.MCPAuthStateRequired
		out.Status.LastError = "authRef.clientCredentialRef is required for clientCredentials auth"
		return fmt.Errorf("%w: %s", spec.ErrMCPAuthRequired, out.Status.LastError)
	}

	raw, err := m.secrets.ResolveSecret(ctx, cfg.AuthRef.ClientCredentialRef)
	if err != nil {
		out.Status.State = spec.MCPAuthStateError
		out.Status.LastError = err.Error()
		return err
	}

	creds, sensitive, err := parseOAuthClientCredentialsSecret(raw, true)
	if err != nil {
		out.Status.State = spec.MCPAuthStateError
		out.Status.LastError = err.Error()
		return err
	}

	handler, err := mcpExtAuth.NewClientCredentialsHandler(&mcpExtAuth.ClientCredentialsHandlerConfig{
		Credentials: creds,
		HTTPClient:  m.httpClient,
	})
	if err != nil {
		out.Status.State = spec.MCPAuthStateError
		out.Status.LastError = err.Error()
		return err
	}

	out.OAuthHandler = &trackedOAuthHandler{
		inner: handler,
		sink:  m.statusSink,
		status: spec.MCPAuthStatus{
			ServerID: cfg.ID,
			AuthMode: spec.MCPHTTPAuthClientCredentials,
			State:    spec.MCPAuthStateAuthorized,
		},
	}
	out.SensitiveValues = append(out.SensitiveValues, raw)
	out.SensitiveValues = append(out.SensitiveValues, sensitive...)
	out.Status.State = spec.MCPAuthStateAuthorized
	return nil
}

func parseOAuthClientCredentialsSecret(
	raw string,
	requireSecret bool,
) (*oauthex.ClientCredentials, []string, error) {
	var wire struct {
		ClientID          string `json:"clientID"`
		ClientIDSnake     string `json:"client_id"`
		ClientSecret      string `json:"clientSecret"`
		ClientSecretSnake string `json:"client_secret"`
	}
	if err := json.Unmarshal([]byte(raw), &wire); err != nil {
		return nil, nil, fmt.Errorf(
			"%w: OAuth client credentials secret must be a JSON object",
			spec.ErrMCPInvalidRequest,
		)
	}

	clientID := strings.TrimSpace(firstNonEmpty(wire.ClientID, wire.ClientIDSnake))
	clientSecret := firstNonEmpty(wire.ClientSecret, wire.ClientSecretSnake)

	creds := &oauthex.ClientCredentials{ClientID: clientID}
	if strings.TrimSpace(clientSecret) != "" {
		creds.ClientSecretAuth = &oauthex.ClientSecretAuth{ClientSecret: clientSecret}
	}
	if requireSecret && creds.ClientSecretAuth == nil {
		return nil, nil, fmt.Errorf(
			"%w: OAuth client credentials secret requires clientSecret",
			spec.ErrMCPInvalidRequest,
		)
	}
	if err := creds.Validate(); err != nil {
		return nil, nil, fmt.Errorf("%w: %w", spec.ErrMCPInvalidRequest, err)
	}

	var sensitive []string
	if clientSecret != "" {
		sensitive = append(sensitive, clientSecret)
	}
	return creds, sensitive, nil
}

func validateResolvedHTTPHeaders(headers map[string]string) error {
	for key, value := range headers {
		if strings.TrimSpace(key) == "" {
			return errors.New("resolved HTTP header name cannot be empty")
		}
		if err := validateResolvedHTTPHeaderName(key); err != nil {
			return err
		}

		if strings.EqualFold(key, "authorization") {
			return errors.New("resolved HTTP headers must not include Authorization; use authRef")
		}
		if managedResolvedHTTPHeader(key) {
			return fmt.Errorf("resolved HTTP header %q is managed by the MCP/HTTP transport", key)
		}

		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("resolved HTTP header %q contains newline characters", key)
		}
		if len(value) > maxResolvedHTTPHeaderValueLen {
			return fmt.Errorf(
				"resolved HTTP header %q exceeds maximum length of %d",
				key,
				maxResolvedHTTPHeaderValueLen,
			)
		}
	}
	return nil
}

func validateResolvedHTTPHeaderName(name string) error {
	for _, c := range name {
		if c <= 0x20 || c > 0x7E || c == ':' {
			return fmt.Errorf("invalid resolved HTTP header name %q", name)
		}
	}
	return nil
}

func cloneStringMapNonNil(in map[string]string) map[string]string {
	out := maps.Clone(in)
	if out == nil {
		return map[string]string{}
	}
	return out
}
