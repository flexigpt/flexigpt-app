package auth

import (
	"context"
	"errors"
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
	credentials *oauthex.ClientCredentials
	httpClient  *http.Client

	mu           sync.RWMutex
	tokenSource  oauth2.TokenSource
	grantedScope map[string][]string
}

var _ mcpAuth.OAuthHandler = (*clientCredentialsOAuthHandler)(nil)

func newClientCredentialsOAuthHandler(
	credentials *oauthex.ClientCredentials,
	httpClient *http.Client,
) (mcpAuth.OAuthHandler, error) {
	if credentials == nil {
		return nil, fmt.Errorf("%w: credentials are required", spec.ErrMCPInvalidRequest)
	}
	if err := credentials.Validate(); err != nil {
		return nil, fmt.Errorf("%w: invalid credentials: %w", spec.ErrMCPInvalidRequest, err)
	}
	if credentials.ClientSecretAuth == nil {
		return nil, fmt.Errorf(
			"%w: clientSecretAuth is required for client credentials grant",
			spec.ErrMCPInvalidRequest,
		)
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &clientCredentialsOAuthHandler{
		credentials:  credentials,
		httpClient:   httpClient,
		grantedScope: map[string][]string{},
	}, nil
}

func (h *clientCredentialsOAuthHandler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	h.mu.RLock()
	ts := h.tokenSource
	h.mu.RUnlock()
	return ts, nil
}

func (h *clientCredentialsOAuthHandler) Authorize(
	ctx context.Context,
	req *http.Request,
	resp *http.Response,
) error {
	wwwChallenges, err := oauthex.ParseWWWAuthenticate(resp.Header[http.CanonicalHeaderKey("WWW-Authenticate")])
	if err != nil {
		return fmt.Errorf("failed to parse WWW-Authenticate header: %w", err)
	}

	httpClient := h.httpClient

	prm, err := getProtectedResourceMetadata(ctx, wwwChallenges, req.URL.String(), httpClient)
	if err != nil {
		return err
	}
	if len(prm.AuthorizationServers) == 0 {
		return errors.New("protected resource metadata has no authorization servers specified")
	}

	asm, err := mcpAuth.GetAuthServerMetadata(ctx, prm.AuthorizationServers[0], httpClient)
	if err != nil {
		return fmt.Errorf("failed to get authorization server metadata: %w", err)
	}
	if asm == nil {
		authServerURL := prm.AuthorizationServers[0]
		asm = &oauthex.AuthServerMeta{
			Issuer:        authServerURL,
			TokenEndpoint: authServerURL + "/token",
		}
	}

	requestedScopes := scopesFromChallenges(wwwChallenges)
	if len(requestedScopes) == 0 && len(prm.ScopesSupported) > 0 {
		requestedScopes = slices.Clone(prm.ScopesSupported)
	}
	requestedScopes = unionScopes(
		h.previousGrantedScopes(asm.Issuer),
		requestedScopes,
	)

	cfg := &clientcredentials.Config{
		ClientID:     h.credentials.ClientID,
		ClientSecret: h.credentials.ClientSecretAuth.ClientSecret,
		TokenURL:     asm.TokenEndpoint,
		Scopes:       requestedScopes,
		AuthStyle:    selectTokenAuthMethod(asm.TokenEndpointAuthMethodsSupported),
	}

	ctxWithClient := context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	ts := cfg.TokenSource(ctxWithClient)

	tok, err := ts.Token()
	if err != nil {
		h.mu.Lock()
		h.tokenSource = nil
		h.mu.Unlock()
		return fmt.Errorf("client credentials token request failed: %w", err)
	}

	scopes := scopesFromOAuthToken(tok)
	if len(scopes) == 0 {
		scopes = slices.Clone(requestedScopes)
	}

	h.mu.Lock()
	h.tokenSource = ts
	if h.grantedScope == nil {
		h.grantedScope = map[string][]string{}
	}
	h.grantedScope[asm.Issuer] = scopes
	h.mu.Unlock()

	return nil
}

func (h *clientCredentialsOAuthHandler) previousGrantedScopes(issuer string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return slices.Clone(h.grantedScope[issuer])
}

func getProtectedResourceMetadata(
	ctx context.Context,
	wwwChallenges []oauthex.Challenge,
	mcpServerURL string,
	httpClient *http.Client,
) (*oauthex.ProtectedResourceMetadata, error) {
	for _, u := range protectedResourceMetadataURLs(resourceMetadataURLFromChallenges(wwwChallenges), mcpServerURL) {
		prm, err := oauthex.GetProtectedResourceMetadata(ctx, u.url, u.resource, httpClient)
		if err != nil {
			continue
		}
		if prm == nil {
			continue
		}
		if len(prm.AuthorizationServers) == 0 {
			return nil, errors.New("protected resource metadata has no authorization servers specified")
		}
		return prm, nil
	}

	u, err := url.Parse(mcpServerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MCP server URL: %w", err)
	}
	u.Path = ""
	return &oauthex.ProtectedResourceMetadata{
		AuthorizationServers: []string{u.String()},
		Resource:             mcpServerURL,
	}, nil
}

type prmURL struct {
	url      string
	resource string
}

func protectedResourceMetadataURLs(metadataURL, resourceURL string) []prmURL {
	var urls []prmURL
	if metadataURL != "" {
		urls = append(urls, prmURL{
			url:      metadataURL,
			resource: resourceURL,
		})
	}
	ru, err := url.Parse(resourceURL)
	if err != nil {
		return urls
	}
	mu := *ru
	mu.Path = "/.well-known/oauth-protected-resource/" + strings.TrimLeft(ru.Path, "/")
	urls = append(urls, prmURL{
		url:      mu.String(),
		resource: resourceURL,
	})
	mu.Path = "/.well-known/oauth-protected-resource"
	ru.Path = ""
	urls = append(urls, prmURL{
		url:      mu.String(),
		resource: ru.String(),
	})
	return urls
}

func resourceMetadataURLFromChallenges(cs []oauthex.Challenge) string {
	for _, c := range cs {
		if u := c.Params["resource_metadata"]; u != "" {
			return u
		}
	}
	return ""
}

func scopesFromChallenges(cs []oauthex.Challenge) []string {
	for _, c := range cs {
		if c.Scheme == "bearer" && c.Params["scope"] != "" {
			return strings.Fields(c.Params["scope"])
		}
	}
	return nil
}

func scopesFromOAuthToken(tok *oauth2.Token) []string {
	if tok == nil {
		return nil
	}

	switch v := tok.Extra("scope").(type) {
	case string:
		return strings.Fields(v)
	case []string:
		return slices.Clone(v)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func unionScopes(sets ...[]string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)

	for _, scopes := range sets {
		for _, scope := range scopes {
			scope = strings.TrimSpace(scope)
			if scope == "" {
				continue
			}
			if _, ok := seen[scope]; ok {
				continue
			}
			seen[scope] = struct{}{}
			out = append(out, scope)
		}
	}

	return out
}

func selectTokenAuthMethod(supported []string) oauth2.AuthStyle {
	prefOrder := []string{
		"client_secret_post",
		"client_secret_basic",
	}
	for _, method := range prefOrder {
		if slices.Contains(supported, method) {
			return authMethodToStyle(method)
		}
	}
	return oauth2.AuthStyleAutoDetect
}

func authMethodToStyle(method string) oauth2.AuthStyle {
	switch method {
	case "client_secret_post":
		return oauth2.AuthStyleInParams
	case "client_secret_basic":
		return oauth2.AuthStyleInHeader
	case "none":
		return oauth2.AuthStyleInParams
	default:
		return oauth2.AuthStyleAutoDetect
	}
}
