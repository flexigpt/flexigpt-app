package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"

	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

type metadataEndpoint struct {
	URL      string
	Resource string
}

func discoverProtectedResourceMetadata(
	ctx context.Context,
	resourceURL string,
	metadataURL string,
	c *http.Client,
) (*oauthex.ProtectedResourceMetadata, error) {
	if c == nil {
		c = http.DefaultClient
	}

	var lastErr error
	for _, candidate := range protectedResourceMetadataURLs(metadataURL, resourceURL) {
		prm, err := oauthex.GetProtectedResourceMetadata(ctx, candidate.URL, candidate.Resource, c)
		if err != nil {
			lastErr = err
			continue
		}
		if prm != nil {
			return prm, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrMCPAuthRequired, lastErr)
	}
	return nil, fmt.Errorf("%w: protected resource metadata not found", spec.ErrMCPAuthRequired)
}

func discoverAuthorizationServerMetadataNoPKCE(
	ctx context.Context,
	issuerURL string,
	c *http.Client,
) (*oauthex.AuthServerMeta, error) {
	if c == nil {
		c = http.DefaultClient
	}

	var lastErr error
	for _, candidate := range authorizationServerMetadataURLs(issuerURL) {
		asm, err := fetchAuthorizationServerMetadataNoPKCE(ctx, candidate, issuerURL, c)
		if err != nil {
			lastErr = err
			continue
		}
		if asm != nil {
			return asm, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrMCPAuthRequired, lastErr)
	}
	//nolint:nilnil // Ok.
	return nil, nil
}

func fetchAuthorizationServerMetadataNoPKCE(
	ctx context.Context,
	metadataURL string,
	issuer string,
	c *http.Client,
) (*oauthex.AuthServerMeta, error) {
	if err := checkHTTPSOrLoopback(metadataURL); err != nil {
		return nil, fmt.Errorf("metadataURL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			//nolint:nilnil // Ok.
			return nil, nil
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("bad status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	ct := resp.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil || mediaType != "application/json" {
		return nil, fmt.Errorf("bad content type %q", ct)
	}

	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	var asm oauthex.AuthServerMeta
	if err := dec.Decode(&asm); err != nil {
		return nil, err
	}

	if strings.TrimRight(asm.Issuer, "/") != strings.TrimRight(issuer, "/") {
		return nil, fmt.Errorf("metadata issuer %q does not match issuer URL %q", asm.Issuer, issuer)
	}
	if asm.Issuer == "" {
		return nil, errors.New("authorization server metadata is missing issuer")
	}
	if asm.TokenEndpoint == "" {
		return nil, errors.New("authorization server metadata is missing token_endpoint")
	}
	if err := validateAuthServerMetaURLs(&asm); err != nil {
		return nil, err
	}

	return &asm, nil
}

func authorizationServerMetadataURLs(issuerURL string) []string {
	var urls []string

	baseURL, err := url.Parse(issuerURL)
	if err != nil {
		return nil
	}

	if baseURL.Path == "" {
		baseURL.Path = "/.well-known/oauth-authorization-server"
		urls = append(urls, baseURL.String())

		baseURL.Path = "/.well-known/openid-configuration"
		urls = append(urls, baseURL.String())
		return urls
	}

	originalPath := baseURL.Path

	baseURL.Path = "/.well-known/oauth-authorization-server/" + strings.TrimLeft(originalPath, "/")
	urls = append(urls, baseURL.String())

	baseURL.Path = "/.well-known/openid-configuration/" + strings.TrimLeft(originalPath, "/")
	urls = append(urls, baseURL.String())

	baseURL.Path = "/" + strings.Trim(originalPath, "/") + "/.well-known/openid-configuration"
	urls = append(urls, baseURL.String())

	return urls
}

func protectedResourceMetadataURLs(metadataURL, resourceURL string) []metadataEndpoint {
	var urls []metadataEndpoint
	if metadataURL != "" {
		urls = append(urls, metadataEndpoint{
			URL:      metadataURL,
			Resource: resourceURL,
		})
	}

	ru, err := url.Parse(resourceURL)
	if err != nil {
		return urls
	}

	mu := *ru

	mu.Path = "/.well-known/oauth-protected-resource/" + strings.TrimLeft(ru.Path, "/")
	urls = append(urls, metadataEndpoint{
		URL:      mu.String(),
		Resource: resourceURL,
	})

	mu.Path = "/.well-known/oauth-protected-resource"
	ru.Path = ""
	urls = append(urls, metadataEndpoint{
		URL:      mu.String(),
		Resource: ru.String(),
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

func validateAuthServerMetaURLs(asm *oauthex.AuthServerMeta) error {
	if asm == nil {
		return errors.New("authorization server metadata is nil")
	}

	urls := []struct {
		name  string
		value string
	}{
		{"issuer", asm.Issuer},
		{"authorization_endpoint", asm.AuthorizationEndpoint},
		{"token_endpoint", asm.TokenEndpoint},
		{"jwks_uri", asm.JWKSURI},
		{"registration_endpoint", asm.RegistrationEndpoint},
		{"service_documentation", asm.ServiceDocumentation},
		{"op_policy_uri", asm.OpPolicyURI},
		{"op_tos_uri", asm.OpTOSURI},
		{"revocation_endpoint", asm.RevocationEndpoint},
		{"introspection_endpoint", asm.IntrospectionEndpoint},
	}

	for _, u := range urls {
		if err := checkURLScheme(u.value); err != nil {
			return fmt.Errorf("%s: %w", u.name, err)
		}
	}

	urls = []struct {
		name  string
		value string
	}{
		{"issuer", asm.Issuer},
		{"authorization_endpoint", asm.AuthorizationEndpoint},
		{"token_endpoint", asm.TokenEndpoint},
		{"registration_endpoint", asm.RegistrationEndpoint},
		{"introspection_endpoint", asm.IntrospectionEndpoint},
	}

	for _, u := range urls {
		if err := checkHTTPSOrLoopback(u.value); err != nil {
			return fmt.Errorf("%s: %w", u.name, err)
		}
	}

	return nil
}

func checkURLScheme(u string) error {
	if u == "" {
		return nil
	}
	uu, err := url.Parse(u)
	if err != nil {
		return err
	}
	scheme := strings.ToLower(uu.Scheme)
	if scheme == "javascript" || scheme == "data" || scheme == "vbscript" {
		return fmt.Errorf("URL has disallowed scheme %q", scheme)
	}
	return nil
}

func checkHTTPSOrLoopback(addr string) error {
	if addr == "" {
		return nil
	}
	u, err := url.Parse(addr)
	if err != nil {
		return err
	}
	if u.Scheme == "https" {
		return nil
	}
	if isLoopbackHost(u.Hostname()) {
		return nil
	}
	return fmt.Errorf("URL %q does not use HTTPS or is not a loopback address", addr)
}

func isLoopbackHost(host string) bool {
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
