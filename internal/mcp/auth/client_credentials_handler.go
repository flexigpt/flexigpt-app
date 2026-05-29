package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"

	mcpAuth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

type clientCredentialsOAuthHandler struct {
	mu            sync.RWMutex
	httpClient    *http.Client
	creds         *oauthex.ClientCredentials
	tokenSource   oauth2.TokenSource
	grantedScopes map[string][]string
	lastStatus    spec.MCPAuthStatus
}

var _ mcpAuth.OAuthHandler = (*clientCredentialsOAuthHandler)(nil)

func newClientCredentialsOAuthHandler(
	creds *oauthex.ClientCredentials,
	httpClient *http.Client,
) (*clientCredentialsOAuthHandler, error) {
	if creds == nil {
		return nil, fmt.Errorf("%w: client credentials are required", spec.ErrMCPAuthRequired)
	}
	if err := creds.Validate(); err != nil {
		return nil, fmt.Errorf("%w: invalid client credentials: %w", spec.ErrMCPInvalidRequest, err)
	}
	if creds.ClientSecretAuth == nil || strings.TrimSpace(creds.ClientSecretAuth.ClientSecret) == "" {
		return nil, fmt.Errorf("%w: clientSecretAuth is required for client credentials grant", spec.ErrMCPAuthRequired)
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &clientCredentialsOAuthHandler{
		httpClient:    httpClient,
		creds:         creds,
		grantedScopes: map[string][]string{},
	}, nil
}

func (h *clientCredentialsOAuthHandler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	if h == nil {
		//nolint:nilnil // Ok if token source is nil.
		return nil, nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.tokenSource, nil
}

func (h *clientCredentialsOAuthHandler) AuthStatus() (spec.MCPAuthStatus, bool) {
	if h == nil {
		return spec.MCPAuthStatus{}, false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.tokenSource == nil {
		return spec.MCPAuthStatus{}, false
	}
	return h.lastStatus, true
}

func (h *clientCredentialsOAuthHandler) Authorize(
	ctx context.Context,
	req *http.Request,
	resp *http.Response,
) error {
	if h == nil {
		return fmt.Errorf("%w: nil client credentials handler", spec.ErrMCPRuntimeNotReady)
	}
	if req == nil || req.URL == nil {
		return fmt.Errorf("%w: missing request URL", spec.ErrMCPAuthRequired)
	}

	defer drainAndClose(resp)

	challenges, err := oauthex.ParseWWWAuthenticate(resp.Header.Values("WWW-Authenticate"))
	if err != nil {
		return fmt.Errorf("failed to parse WWW-Authenticate header: %w", err)
	}

	resourceURL := req.URL.String()
	prm, err := discoverProtectedResourceMetadata(
		ctx,
		resourceURL,
		resourceMetadataURLFromChallenges(challenges),
		h.httpClient,
	)
	if err != nil {
		return err
	}
	if len(prm.AuthorizationServers) == 0 {
		return fmt.Errorf(
			"%w: protected resource metadata has no authorization servers specified",
			spec.ErrMCPAuthRequired,
		)
	}

	asm, err := discoverAuthorizationServerMetadataNoPKCE(ctx, prm.AuthorizationServers[0], h.httpClient)
	if err != nil {
		return err
	}
	if asm == nil {
		return fmt.Errorf("%w: authorization server metadata not found", spec.ErrMCPAuthRequired)
	}

	requestedScopes := scopesFromChallenges(challenges)
	if len(requestedScopes) == 0 && len(prm.ScopesSupported) > 0 {
		requestedScopes = slices.Clone(prm.ScopesSupported)
	}

	h.mu.RLock()
	prevScopes := slices.Clone(h.grantedScopes[asm.Issuer])
	h.mu.RUnlock()

	requestedScopes = unionScopes(prevScopes, requestedScopes)

	cfg := &clientcredentials.Config{
		ClientID:       h.creds.ClientID,
		ClientSecret:   h.creds.ClientSecretAuth.ClientSecret,
		TokenURL:       asm.TokenEndpoint,
		Scopes:         requestedScopes,
		AuthStyle:      selectTokenAuthMethod(asm.TokenEndpointAuthMethodsSupported),
		EndpointParams: url.Values{},
	}
	if prm.Resource != "" {
		cfg.EndpointParams.Set("resource", prm.Resource)
	}

	ctxWithClient := context.WithValue(ctx, oauth2.HTTPClient, h.httpClient)
	ts := cfg.TokenSource(ctxWithClient)

	tok, err := ts.Token()
	if err != nil {
		h.mu.Lock()
		h.tokenSource = nil
		h.lastStatus = spec.MCPAuthStatus{
			AuthMode: spec.MCPHTTPAuthClientCredentials,
			State:    spec.MCPAuthStateRequired,
			Resource: prm.Resource,
		}
		h.mu.Unlock()
		return fmt.Errorf("client credentials token request failed: %w", err)
	}

	h.mu.Lock()
	h.tokenSource = ts
	h.lastStatus = spec.MCPAuthStatus{
		AuthMode:            spec.MCPHTTPAuthClientCredentials,
		State:               spec.MCPAuthStateAuthorized,
		Resource:            prm.Resource,
		AuthorizationServer: asm.Issuer,
	}
	if !tok.Expiry.IsZero() {
		expiresAt := tok.Expiry.UTC()
		h.lastStatus.ExpiresAt = &expiresAt
	}
	if scopes := scopesFromOAuthToken(tok); len(scopes) > 0 {
		h.lastStatus.Scopes = slices.Clone(scopes)
	} else if len(requestedScopes) > 0 {
		h.lastStatus.Scopes = slices.Clone(requestedScopes)
	}
	h.grantedScopes[asm.Issuer] = slices.Clone(h.lastStatus.Scopes)
	h.mu.Unlock()

	return nil
}

func selectTokenAuthMethod(supported []string) oauth2.AuthStyle {
	prefOrder := []string{
		"client_secret_post",
		"client_secret_basic",
	}
	for _, method := range prefOrder {
		if slices.Contains(supported, method) {
			switch method {
			case "client_secret_post":
				return oauth2.AuthStyleInParams
			case "client_secret_basic":
				return oauth2.AuthStyleInHeader
			}
		}
	}
	return oauth2.AuthStyleAutoDetect
}

func unionScopes(base, add []string) []string {
	seen := make(map[string]struct{}, len(base)+len(add))
	out := make([]string, 0, len(base)+len(add))

	appendScope := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	for _, s := range base {
		appendScope(s)
	}
	for _, s := range add {
		appendScope(s)
	}
	return out
}
