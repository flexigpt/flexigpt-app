package auth

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	mcpAuth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"
)

// trackingTokenSource wraps an oauth2.TokenSource and pushes status updates to
// the configured sink on every Token() call. The underlying source already
// handles caching and refresh; this wrapper only observes results.
type trackingTokenSource struct {
	source          oauth2.TokenSource
	sink            AuthStatusSink
	status          spec.MCPAuthStatus
	sensitiveValues []string
}

func (s *trackingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := s.source.Token()
	if err != nil {
		if s.sink != nil {
			_ = s.sink.SaveAuthStatus(
				context.Background(),
				redactAuthStatus(authStatusFromTokenError(s.status, err), s.sensitiveValues),
			)
		}
		return nil, err
	}
	if tok == nil {
		nilErr := errors.New("oauth token source returned nil token")
		if s.sink != nil {
			_ = s.sink.SaveAuthStatus(
				context.Background(),
				redactAuthStatus(authStatusFromTokenError(s.status, nilErr), s.sensitiveValues),
			)
		}
		return nil, nilErr
	}
	if s.sink != nil {
		_ = s.sink.SaveAuthStatus(
			context.Background(),
			redactAuthStatus(authStatusFromToken(s.status, tok), s.sensitiveValues),
		)
	}
	return tok, nil
}

// trackedOAuthHandler decorates a SDK OAuthHandler with auth-status tracking.
// It never touches the HTTP request/response body itself; body lifecycle is
// owned by the wrapped SDK handler.
type trackedOAuthHandler struct {
	inner           mcpAuth.OAuthHandler
	sink            AuthStatusSink
	status          spec.MCPAuthStatus
	sensitiveValues []string
}

func (h *trackedOAuthHandler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	ts, err := h.inner.TokenSource(ctx)
	if err != nil {
		h.publish(ctx, authStatusFromTokenError(h.status, err))
		return nil, err
	}
	if ts == nil {
		//nolint:nilnil // The SDK contract allows a nil token source before authorization.
		return nil, nil
	}
	return &trackingTokenSource{
		source:          ts,
		sink:            h.sink,
		status:          h.status,
		sensitiveValues: h.sensitiveValues,
	}, nil
}

func (h *trackedOAuthHandler) Authorize(
	ctx context.Context,
	req *http.Request,
	resp *http.Response,
) error {
	// Body lifecycle is owned by the inner SDK handler; both
	// AuthorizationCodeHandler and extauth.ClientCredentialsHandler drain and
	// close it via their own defers. We must not touch it here.
	err := h.inner.Authorize(ctx, req, resp)
	if err != nil {
		h.publish(ctx, authStatusFromHTTPFailure(h.status, resp, err))
		return err
	}

	// The SDK returns nil from Authorize in two cases:
	//   1. Authorization actually happened and a token source was produced.
	//   2. The SDK chose to retry the call without performing authorization
	//      (e.g. 403 without an insufficient_scope challenge). In that case
	//      no new token exists.
	// Only publish "authorized" in case 1.
	tokenStatus, ok := h.currentTokenStatus(ctx)
	if !ok {
		return nil
	}
	h.publish(ctx, tokenStatus)
	return nil
}

func (h *trackedOAuthHandler) currentTokenStatus(ctx context.Context) (spec.MCPAuthStatus, bool) {
	ts, err := h.inner.TokenSource(ctx)
	if err != nil {
		return authStatusFromTokenError(h.status, err), true
	}
	if ts == nil {
		return spec.MCPAuthStatus{}, false
	}
	tok, err := ts.Token()
	if err != nil {
		return authStatusFromTokenError(h.status, err), true
	}
	if tok == nil {
		nilErr := errors.New("oauth token source returned nil token")
		return authStatusFromTokenError(h.status, nilErr), true
	}
	return authStatusFromToken(h.status, tok), true
}

func (h *trackedOAuthHandler) publish(ctx context.Context, st spec.MCPAuthStatus) {
	if h == nil || h.sink == nil {
		return
	}
	_ = h.sink.SaveAuthStatus(context.WithoutCancel(ctx), redactAuthStatus(st, h.sensitiveValues))
}

func authStatusFromToken(base spec.MCPAuthStatus, tok *oauth2.Token) spec.MCPAuthStatus {
	st := base
	st.LastError = ""

	if tok == nil {
		return authStatusFromTokenError(base, errors.New("oauth token source returned nil token"))
	}
	if scopes := scopesFromOAuthToken(tok); len(scopes) > 0 {
		st.Scopes = scopes
	}
	if !tok.Expiry.IsZero() {
		expiresAt := tok.Expiry.UTC()
		st.ExpiresAt = &expiresAt
	}
	if strings.TrimSpace(tok.AccessToken) == "" {
		st.State = spec.MCPAuthStateError
		st.LastError = "OAuth token is missing an access token"
		return st
	}
	if st.ExpiresAt != nil && !time.Now().UTC().Before(st.ExpiresAt.UTC()) {
		st.State = spec.MCPAuthStateExpired
		st.LastError = "OAuth token is expired"
		return st
	}
	st.State = spec.MCPAuthStateAuthorized
	return st
}

func authStatusFromTokenError(base spec.MCPAuthStatus, err error) spec.MCPAuthStatus {
	st := base
	st.State = spec.MCPAuthStateError
	st.Scopes = nil
	st.ExpiresAt = nil
	if err != nil {
		st.LastError = err.Error()
	}
	if isExpiredOAuthTokenError(err) {
		st.State = spec.MCPAuthStateExpired
	}
	return st
}

func authStatusFromHTTPFailure(
	base spec.MCPAuthStatus,
	resp *http.Response,
	err error,
) spec.MCPAuthStatus {
	st := base
	st.State = spec.MCPAuthStateError
	if err != nil {
		st.LastError = err.Error()
	}
	if resp == nil {
		return st
	}
	challengeErr, scopes := bearerChallengeValues(resp.Header.Values("WWW-Authenticate"))
	if len(scopes) > 0 {
		st.Scopes = scopes
	}
	retrieveErrCode := oauthRetrieveErrorCode(err)
	switch {
	case challengeErr == "insufficient_scope":
		st.State = spec.MCPAuthStateInsufficientScope
	case challengeErr == "invalid_token" || isExpiredOAuthTokenError(err):
		st.State = spec.MCPAuthStateExpired
	case retrieveErrCode != "":
		st.State = spec.MCPAuthStateError
	case resp.StatusCode == http.StatusUnauthorized:
		st.State = spec.MCPAuthStateRequired
	case resp.StatusCode == http.StatusForbidden:
		st.State = spec.MCPAuthStateError
	}
	return st
}

func bearerChallengeValues(headers []string) (challengeErr string, scopes []string) {
	challenges, err := oauthex.ParseWWWAuthenticate(headers)
	if err != nil {
		return "", nil
	}
	for _, c := range challenges {
		if !strings.EqualFold(c.Scheme, "bearer") {
			continue
		}
		if challengeErr == "" {
			challengeErr = c.Params["error"]
		}
		if scope := c.Params["scope"]; scope != "" {
			scopes = strings.Fields(scope)
		}
	}
	return challengeErr, scopes
}

// scopesFromOAuthToken extracts a "scope" claim from an oauth2.Token's extras.
// Some authorization servers return it as a space-separated string, others as
// an array; tolerate both. Returns nil when no scope information is present.
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

func isExpiredOAuthTokenError(err error) bool {
	if err == nil {
		return false
	}
	if oauthRetrieveErrorCode(err) == errStrInvalidGrant {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, errStrInvalidGrant) ||
		(strings.Contains(msg, "expired") && strings.Contains(msg, "refresh token"))
}

func oauthRetrieveErrorCode(err error) string {
	if retrieveErr, ok := errors.AsType[*oauth2.RetrieveError](err); ok {
		return retrieveErr.ErrorCode
	}
	return ""
}

func redactAuthStatus(st spec.MCPAuthStatus, sensitiveValues []string) spec.MCPAuthStatus {
	st.LastError = redactSensitive(st.LastError, sensitiveValues)
	return st
}

func redactSensitive(in string, sensitiveValues []string) string {
	if in == "" || len(sensitiveValues) == 0 {
		return in
	}
	out := in
	for _, v := range sensitiveValues {
		if v == "" || strings.TrimSpace(v) == "" {
			continue
		}
		out = strings.ReplaceAll(out, v, "[REDACTED]")
	}
	return out
}
