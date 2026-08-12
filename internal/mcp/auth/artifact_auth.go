package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	mcpAuth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/auth/extauth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	errStrInvalidGrant       = "invalid_grant"
	errStrOAuthNotConfigured = "OAuth authorization code flow is not configured"
)

type ResolvedTransportAuth struct {
	Env     map[string]string
	Headers map[string]string

	SensitiveValues []string
	Status          MCPAuthStatus
	OAuthHandler    mcpAuth.OAuthHandler
}

type ConnectionAuthorizer interface {
	PrepareConnection(
		ctx context.Context,
		config server.RuntimeConfig,
	) (ResolvedTransportAuth, error)

	ConnectionSucceeded(
		ctx context.Context,
		config server.RuntimeConfig,
	)

	ConnectionFailed(
		ctx context.Context,
		config server.RuntimeConfig,
		err error,
	)
}

type OAuthAuthorizationRequest struct {
	Server           artifact.ArtifactRef
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
		request OAuthAuthorizationRequest,
	) (*OAuthAuthorizationResult, error)
}

type AuthStatusSink interface {
	SaveAuthStatus(
		ctx context.Context,
		status MCPAuthStatus,
	) error
}

type authStatusKey struct {
	Server artifact.ArtifactRef
}

type AuthManager struct {
	secrets          SecretResolver
	oauthBroker      OAuthAuthorizationBroker
	oauthRedirectURL string
	httpClient       *http.Client
	oauthTokenStore  OAuthTokenStore

	mu       sync.RWMutex
	statuses map[authStatusKey]MCPAuthStatus
}

type AuthManagerOption func(*AuthManager)

func WithOAuthTokenStore(
	store OAuthTokenStore,
) AuthManagerOption {
	return func(manager *AuthManager) {
		manager.oauthTokenStore = store
	}
}

func WithOAuthAuthorizationBroker(
	broker OAuthAuthorizationBroker,
) AuthManagerOption {
	return func(manager *AuthManager) {
		manager.oauthBroker = broker
	}
}

func WithOAuthRedirectURL(
	redirectURL string,
) AuthManagerOption {
	return func(manager *AuthManager) {
		manager.oauthRedirectURL = strings.TrimSpace(redirectURL)
	}
}

func WithAuthHTTPClient(
	client *http.Client,
) AuthManagerOption {
	return func(manager *AuthManager) {
		manager.httpClient = client
	}
}

func NewAuthManager(
	secrets SecretResolver,
	options ...AuthManagerOption,
) *AuthManager {
	if secrets == nil {
		secrets = StaticSecretResolver{}
	}

	manager := &AuthManager{
		secrets:  secrets,
		statuses: make(map[authStatusKey]MCPAuthStatus),
	}
	for _, option := range options {
		if option != nil {
			option(manager)
		}
	}
	return manager
}

func (m *AuthManager) PrepareConnection(
	ctx context.Context,
	config server.RuntimeConfig,
) (ResolvedTransportAuth, error) {
	if err := validateArtifactAuthInput(ctx, config); err != nil {
		return ResolvedTransportAuth{}, err
	}

	output := ResolvedTransportAuth{
		Env:     map[string]string{},
		Headers: map[string]string{},
		Status:  defaultAuthStatus(config),
	}
	if m == nil {
		return output, nil
	}

	defer func() {
		_ = m.SaveAuthStatus(
			context.WithoutCancel(ctx),
			output.Status,
		)
	}()

	switch config.Transport {
	case spec.MCPTransportStdio:
		output.Status.AuthMode = server.MCPHTTPAuthNone
		output.Status.State = MCPAuthStateNotRequired
		return output, nil

	case spec.MCPTransportStreamableHTTP:
		if config.StreamableHTTP == nil {
			output.Status.State = MCPAuthStateError
			output.Status.LastError = "missing streamable HTTP runtime config"
			return output, fmt.Errorf(
				"%w: %s",
				spec.ErrMCPInvalidRequest,
				output.Status.LastError,
			)
		}

		mode := normalizeHTTPAuthMode(config.StreamableHTTP.AuthMode)
		output.Status.AuthMode = mode
		output.Status.Resource = config.StreamableHTTP.URL

		switch mode {
		case server.MCPHTTPAuthNone:
			output.Status.State = MCPAuthStateNotRequired
			return output, nil

		case server.MCPHTTPAuthAPIKey:
			if len(config.StreamableHTTP.Headers) == 0 {
				output.Status.State = MCPAuthStateRequired
				output.Status.LastError = "API-key MCP server has no materialized secret header"
				return output, fmt.Errorf(
					"%w: %s",
					spec.ErrMCPAuthRequired,
					output.Status.LastError,
				)
			}
			output.Status.State = MCPAuthStateRequired
			return output, nil

		case server.MCPHTTPAuthOAuth:
			o := m.configureAuthorizationCodeOAuth(
				ctx,
				config,
				&output,
			)
			return output, o

		case server.MCPHTTPAuthClientCredentials:
			o := m.configureClientCredentialsOAuth(
				ctx,
				config,
				&output,
			)
			return output, o

		default:
			output.Status.State = MCPAuthStateError
			output.Status.LastError = "unsupported MCP HTTP authentication mode"
			return output, fmt.Errorf(
				"%w: %s",
				spec.ErrMCPInvalidRequest,
				output.Status.LastError,
			)
		}

	default:
		return output, fmt.Errorf(
			"%w: unsupported MCP runtime transport %q",
			spec.ErrMCPInvalidRequest,
			config.Transport,
		)
	}
}

func (m *AuthManager) ConnectionSucceeded(
	ctx context.Context,
	config server.RuntimeConfig,
) {
	if m == nil {
		return
	}

	status := defaultAuthStatus(config)
	switch status.AuthMode {
	case server.MCPHTTPAuthNone:
		status.State = MCPAuthStateNotRequired
	case server.MCPHTTPAuthAPIKey:
		status.State = MCPAuthStateAuthorized
	default:
		return
	}
	status.LastError = ""
	_ = m.SaveAuthStatus(context.WithoutCancel(ctx), status)
}

func (m *AuthManager) ConnectionFailed(
	ctx context.Context,
	config server.RuntimeConfig,
	err error,
) {
	if m == nil || err == nil {
		return
	}

	status := defaultAuthStatus(config)
	switch status.AuthMode {
	case server.MCPHTTPAuthOAuth:
		// The tracked OAuth handler owns precise authorization state. Do not
		// overwrite a pending loopback flow with a generic connect error.
		return
	case server.MCPHTTPAuthAPIKey, server.MCPHTTPAuthClientCredentials:
		status.State = MCPAuthStateError
		status.LastError = redactSensitive(
			err.Error(),
			config.SensitiveValues,
		)
	default:
		return
	}
	_ = m.SaveAuthStatus(context.WithoutCancel(ctx), status)
}

func (m *AuthManager) SaveAuthStatus(
	ctx context.Context,
	status MCPAuthStatus,
) error {
	if err := status.Server.Validate(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.statuses == nil {
		m.statuses = make(map[authStatusKey]MCPAuthStatus)
	}
	m.statuses[authStatusKey{Server: status.Server}] = cloneArtifactAuthStatus(status)
	return nil
}

func (m *AuthManager) ClearAuthStatus(
	serverRef artifact.ArtifactRef,
) {
	if m == nil {
		return
	}
	m.mu.Lock()
	delete(m.statuses, authStatusKey{Server: serverRef})
	m.mu.Unlock()
}

func (m *AuthManager) ClearAuthStatuses() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.statuses = make(map[authStatusKey]MCPAuthStatus)
	m.mu.Unlock()
}

func (m *AuthManager) BuildAuthHealth(
	ctx context.Context,
	config server.RuntimeConfig,
) MCPAuthHealth {
	defaultStatus := defaultAuthStatus(config)
	status := defaultStatus
	if saved, found := m.GetAuthStatus(config.Server); found {
		status = mergeArtifactAuthStatus(saved, defaultStatus)
	}

	if status.AuthMode == server.MCPHTTPAuthOAuth &&
		status.State != MCPAuthStateAuthorized &&
		m != nil &&
		m.oauthTokenStore != nil {
		if token, err := m.oauthTokenStore.LoadOAuthToken(
			ctx,
			defaultStatus,
		); err == nil && token != nil {
			status = authStatusFromToken(defaultStatus, token)
		}
	}

	health := MCPAuthHealth{
		Server:     config.Server,
		AuthMode:   status.AuthMode,
		State:      authHealthState(status.State),
		Configured: authConfigured(config, status, m),
		Resource:   status.Resource,
		Scopes:     append([]string(nil), status.Scopes...),
		ExpiresAt:  cloneTime(status.ExpiresAt),
		LastError:  status.LastError,
	}
	if m != nil {
		health.OAuthRedirectURL = m.oauthRedirectURL
		if broker, ok := m.oauthBroker.(interface {
			RedirectURL() string
		}); ok && health.OAuthRedirectURL == "" {
			health.OAuthRedirectURL = broker.RedirectURL()
		}
		if broker, ok := m.oauthBroker.(interface {
			Pending() []spec.MCPOAuthAuthorization
		}); ok {
			for _, pending := range broker.Pending() {
				if pending.Server != config.Server {
					continue
				}
				health.AuthorizationPending = true
				health.AuthorizationURL = pending.AuthorizationURL
				health.AuthorizationExpiresAt = pending.ExpiresAt
				health.State = MCPAuthHealthStateAuthorizationPending
				health.LastError = ""
				break
			}
		}
	}
	return health
}

func (m *AuthManager) GetAuthStatus(
	serverRef artifact.ArtifactRef,
) (MCPAuthStatus, bool) {
	if m == nil {
		return MCPAuthStatus{}, false
	}

	m.mu.RLock()
	status, found := m.statuses[authStatusKey{Server: serverRef}]
	m.mu.RUnlock()

	if !found {
		return MCPAuthStatus{}, false
	}
	return cloneArtifactAuthStatus(status), true
}

func (m *AuthManager) configureAuthorizationCodeOAuth(
	ctx context.Context,
	config server.RuntimeConfig,
	output *ResolvedTransportAuth,
) error {
	output.Status.State = MCPAuthStateRequired
	if m.oauthBroker == nil || m.oauthRedirectURL == "" {
		output.Status.LastError = errStrOAuthNotConfigured
		return fmt.Errorf(
			"%w: %s",
			spec.ErrMCPAuthRequired,
			errStrOAuthNotConfigured,
		)
	}

	httpConfig := config.StreamableHTTP
	var (
		metadataDocument *mcpAuth.ClientIDMetadataDocumentConfig
		preregistered    *oauthex.ClientCredentials
		dynamic          *mcpAuth.DynamicClientRegistrationConfig
	)

	if httpConfig.ClientIDMetadataDocumentURL != "" {
		metadataDocument = &mcpAuth.ClientIDMetadataDocumentConfig{
			URL: httpConfig.ClientIDMetadataDocumentURL,
		}
	}
	if httpConfig.ClientCredentialRef != "" {
		credentials, sensitive, err := resolveOAuthClientCredentials(
			ctx,
			m.secrets,
			httpConfig.ClientCredentialRef,
			config.OAuthClientSecretRequired,
		)
		if err != nil {
			output.Status.State = MCPAuthStateError
			output.Status.LastError = err.Error()
			return err
		}
		preregistered = credentials
		output.SensitiveValues = append(
			output.SensitiveValues,
			sensitive...,
		)
	}
	if preregistered == nil {
		dynamic = &mcpAuth.DynamicClientRegistrationConfig{
			Metadata: &oauthex.ClientRegistrationMetadata{
				RedirectURIs:    []string{m.oauthRedirectURL},
				ClientName:      spec.MCPHostName,
				SoftwareID:      "flexigpt",
				SoftwareVersion: spec.MCPHostVersion,
				TokenEndpointAuthMethod: string(
					server.MCPHTTPAuthNone,
				),
				ResponseTypes: []string{"code"},
				GrantTypes: []string{
					string(spec.GrantTypeAuthorizationCode),
					string(spec.GrantTypeRefreshToken),
				},
			},
		}
	}

	handler, err := mcpAuth.NewAuthorizationCodeHandler(
		&mcpAuth.AuthorizationCodeHandlerConfig{
			ClientIDMetadataDocumentConfig:  metadataDocument,
			PreregisteredClient:             preregistered,
			DynamicClientRegistrationConfig: dynamic,
			RedirectURL:                     m.oauthRedirectURL,
			RequestRefreshToken:             true,
			Client:                          m.httpClient,
			AuthorizationCodeFetcher: func(
				ctx context.Context,
				args *mcpAuth.AuthorizationArgs,
			) (*mcpAuth.AuthorizationResult, error) {
				if args == nil || args.URL == "" {
					return nil, fmt.Errorf(
						"%w: OAuth authorization URL is unavailable",
						spec.ErrMCPAuthRequired,
					)
				}
				result, err := m.oauthBroker.FetchAuthorizationCode(
					ctx,
					OAuthAuthorizationRequest{
						Server:           config.Server,
						AuthorizationURL: args.URL,
					},
				)
				if err != nil {
					return nil, err
				}
				if result == nil || result.Code == "" {
					return nil, fmt.Errorf(
						"%w: OAuth authorization code was not returned",
						spec.ErrMCPAuthRequired,
					)
				}
				return &mcpAuth.AuthorizationResult{
					Code:  result.Code,
					State: result.State,
					Iss:   result.Iss,
				}, nil
			},
		},
	)
	if err != nil {
		output.Status.State = MCPAuthStateError
		output.Status.LastError = redactSensitive(
			err.Error(),
			output.SensitiveValues,
		)
		return redactAuthError(err, output.SensitiveValues)
	}

	output.OAuthHandler = &trackedOAuthHandler{
		inner: handler,
		sink:  m,
		status: MCPAuthStatus{
			Server:   config.Server,
			AuthMode: server.MCPHTTPAuthOAuth,
			State:    MCPAuthStateRequired,
			Resource: httpConfig.URL,
		},
		sensitiveValues: append(
			[]string(nil),
			output.SensitiveValues...,
		),
		tokenStore: m.oauthTokenStore,
	}
	return nil
}

func (m *AuthManager) configureClientCredentialsOAuth(
	ctx context.Context,
	config server.RuntimeConfig,
	output *ResolvedTransportAuth,
) error {
	httpConfig := config.StreamableHTTP
	if httpConfig.ClientCredentialRef == "" {
		output.Status.State = MCPAuthStateRequired
		output.Status.LastError = "clientCredentials requires clientCredentialRef"
		return fmt.Errorf(
			"%w: %s",
			spec.ErrMCPAuthRequired,
			output.Status.LastError,
		)
	}

	credentials, sensitive, err := resolveOAuthClientCredentials(
		ctx,
		m.secrets,
		httpConfig.ClientCredentialRef,
		true,
	)
	if err != nil {
		output.Status.State = MCPAuthStateError
		output.Status.LastError = err.Error()
		return err
	}
	output.SensitiveValues = append(output.SensitiveValues, sensitive...)

	handler, err := extauth.NewClientCredentialsHandler(
		&extauth.ClientCredentialsHandlerConfig{
			Credentials: credentials,
			HTTPClient:  m.httpClient,
		},
	)
	if err != nil {
		output.Status.State = MCPAuthStateError
		output.Status.LastError = redactSensitive(
			err.Error(),
			output.SensitiveValues,
		)
		return redactAuthError(err, output.SensitiveValues)
	}

	output.Status.State = MCPAuthStateRequired
	output.OAuthHandler = &trackedOAuthHandler{
		inner: handler,
		sink:  m,
		status: MCPAuthStatus{
			Server:   config.Server,
			AuthMode: server.MCPHTTPAuthClientCredentials,
			State:    MCPAuthStateRequired,
			Resource: httpConfig.URL,
		},
		sensitiveValues: append(
			[]string(nil),
			output.SensitiveValues...,
		),
		tokenStore: m.oauthTokenStore,
	}
	return nil
}

func defaultAuthStatus(
	config server.RuntimeConfig,
) MCPAuthStatus {
	status := MCPAuthStatus{
		Server:   config.Server,
		AuthMode: server.MCPHTTPAuthNone,
		State:    MCPAuthStateNotRequired,
	}
	if config.StreamableHTTP == nil {
		return status
	}

	status.AuthMode = normalizeHTTPAuthMode(
		config.StreamableHTTP.AuthMode,
	)
	status.Resource = config.StreamableHTTP.URL
	switch status.AuthMode {
	case server.MCPHTTPAuthNone:
		status.State = MCPAuthStateNotRequired
	case server.MCPHTTPAuthAPIKey,
		server.MCPHTTPAuthOAuth,
		server.MCPHTTPAuthClientCredentials:
		status.State = MCPAuthStateRequired
	default:
		status.State = MCPAuthStateError
		status.LastError = "unsupported MCP HTTP authentication mode"
	}
	return status
}

func mergeArtifactAuthStatus(
	current MCPAuthStatus,
	defaults MCPAuthStatus,
) MCPAuthStatus {
	if current.Server != defaults.Server ||
		current.AuthMode != defaults.AuthMode {
		return defaults
	}
	if current.Resource != "" &&
		defaults.Resource != "" &&
		current.Resource != defaults.Resource {
		return defaults
	}
	if current.Resource == "" {
		current.Resource = defaults.Resource
	}
	if current.State == "" {
		current.State = defaults.State
	}
	return current
}

func authConfigured(
	config server.RuntimeConfig,
	status MCPAuthStatus,
	manager *AuthManager,
) bool {
	switch status.AuthMode {
	case server.MCPHTTPAuthNone:
		return true
	case server.MCPHTTPAuthAPIKey:
		return config.StreamableHTTP != nil &&
			len(config.StreamableHTTP.Headers) != 0
	case server.MCPHTTPAuthClientCredentials:
		return config.StreamableHTTP != nil &&
			config.StreamableHTTP.ClientCredentialRef != ""
	case server.MCPHTTPAuthOAuth:
		return status.State == MCPAuthStateAuthorized ||
			(manager != nil &&
				manager.oauthBroker != nil &&
				strings.TrimSpace(manager.oauthRedirectURL) != "")
	default:
		return false
	}
}

func authHealthState(
	state MCPAuthState,
) MCPAuthHealthState {
	switch state {
	case MCPAuthStateNotRequired:
		return MCPAuthHealthStateNotRequired
	case MCPAuthStateAuthorized:
		return MCPAuthHealthStateAuthorized
	case MCPAuthStateExpired:
		return MCPAuthHealthStateExpired
	case MCPAuthStateInsufficientScope:
		return MCPAuthHealthStateInsufficientScope
	case MCPAuthStateError:
		return MCPAuthHealthStateError
	default:
		return MCPAuthHealthStateAuthorizationNeeded
	}
}

func validateArtifactAuthInput(
	ctx context.Context,
	config server.RuntimeConfig,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: MCP auth context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := config.Server.Validate(); err != nil {
		return err
	}
	return nil
}

func normalizeHTTPAuthMode(
	mode server.MCPHTTPAuthMode,
) server.MCPHTTPAuthMode {
	mode = server.MCPHTTPAuthMode(
		strings.TrimSpace(string(mode)),
	)
	if mode == "" {
		return server.MCPHTTPAuthNone
	}
	return mode
}

type redactedAuthError struct {
	message string
	cause   error
}

func (e redactedAuthError) Error() string {
	return e.message
}

func (e redactedAuthError) Unwrap() error {
	return e.cause
}

// RedactError preserves an error chain while replacing configured secret
// values in the externally observable message.
func RedactError(err error, sensitiveValues []string) error {
	return redactAuthError(err, sensitiveValues)
}

func redactAuthError(
	err error,
	sensitiveValues []string,
) error {
	if err == nil {
		return nil
	}
	redacted := redactSensitive(err.Error(), sensitiveValues)
	if redacted == err.Error() {
		return err
	}
	return redactedAuthError{
		message: redacted,
		cause:   err,
	}
}

func parseOAuthClientCredentialsSecret(
	raw string,
	requireClientSecret bool,
) (*oauthex.ClientCredentials, []string, error) {
	var wire struct {
		ClientID     string `json:"clientID"`
		ClientSecret string `json:"clientSecret"`
	}

	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return nil, nil, fmt.Errorf(
			"%w: OAuth client credentials must be one JSON object",
			spec.ErrMCPInvalidRequest,
		)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, nil, fmt.Errorf(
			"%w: OAuth client credentials contain trailing JSON",
			spec.ErrMCPInvalidRequest,
		)
	}
	if strings.TrimSpace(wire.ClientID) == "" ||
		strings.TrimSpace(wire.ClientID) != wire.ClientID {
		return nil, nil, fmt.Errorf(
			"%w: OAuth client credentials require a trimmed clientID",
			spec.ErrMCPInvalidRequest,
		)
	}
	if requireClientSecret &&
		strings.TrimSpace(wire.ClientSecret) == "" {
		return nil, nil, fmt.Errorf(
			"%w: OAuth client credentials require clientSecret",
			spec.ErrMCPInvalidRequest,
		)
	}

	credentials := &oauthex.ClientCredentials{
		ClientID: wire.ClientID,
	}
	sensitive := []string{raw}
	if wire.ClientSecret != "" {
		credentials.ClientSecretAuth = &oauthex.ClientSecretAuth{
			ClientSecret: wire.ClientSecret,
		}
		sensitive = append(sensitive, wire.ClientSecret)
	}
	if err := credentials.Validate(); err != nil {
		return nil, nil, fmt.Errorf(
			"%w: %w",
			spec.ErrMCPInvalidRequest,
			err,
		)
	}
	return credentials, sensitive, nil
}

func cloneArtifactAuthStatus(
	input MCPAuthStatus,
) MCPAuthStatus {
	output := input
	output.Scopes = append([]string(nil), input.Scopes...)
	output.ExpiresAt = cloneTime(input.ExpiresAt)
	return output
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
