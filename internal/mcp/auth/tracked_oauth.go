package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	mcpAuth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"
)

type authStatusProvider interface {
	AuthStatus() (spec.MCPAuthStatus, bool)
}

type trackingTokenSource struct {
	source oauth2.TokenSource
	sink   AuthStatusSink
	status spec.MCPAuthStatus
}

func (s *trackingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := s.source.Token()
	if err != nil {
		if s.sink != nil {
			_ = s.sink.SaveAuthStatus(context.Background(), authStatusFromTokenError(s.status, err))
		}
		return nil, err
	}
	if tok == nil {
		err := errors.New("oauth token source returned nil token")
		if s.sink != nil {
			_ = s.sink.SaveAuthStatus(context.Background(), authStatusFromTokenError(s.status, err))
		}
		return nil, err
	}
	if s.sink != nil {
		_ = s.sink.SaveAuthStatus(context.Background(), authStatusFromToken(s.status, tok))
	}

	return tok, nil
}

type trackedOAuthHandler struct {
	inner  mcpAuth.OAuthHandler
	sink   AuthStatusSink
	status spec.MCPAuthStatus
}

func (h *trackedOAuthHandler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	ts, err := h.inner.TokenSource(ctx)
	if err != nil {
		h.publish(ctx, authStatusFromTokenError(h.status, err))
		return nil, err
	}
	if ts == nil {
		//nolint:nilnil // Ok to return nik when token source is nil.
		return nil, nil
	}

	return &trackingTokenSource{
		source: ts,
		sink:   h.sink,
		status: h.status,
	}, nil
}

func (h *trackedOAuthHandler) Authorize(
	ctx context.Context,
	req *http.Request,
	resp *http.Response,
) error {
	defer drainAndClose(resp)

	resourceURL := ""
	if req != nil && req.URL != nil {
		resourceURL = req.URL.String()
	}
	err := h.inner.Authorize(ctx, req, resp)
	if err != nil {
		h.publish(ctx, authStatusFromHTTPFailure(h.status, resp, err))
		return err
	}

	if provider, ok := h.inner.(authStatusProvider); ok {
		if st, ok := provider.AuthStatus(); ok {
			if st.ServerID == "" {
				st.ServerID = h.status.ServerID
			}
			if st.AuthMode == "" {
				st.AuthMode = h.status.AuthMode
			}
			if st.State == "" {
				st.State = spec.MCPAuthStateAuthorized
			}
			if st.Resource == "" && resourceURL != "" {
				st.Resource = resourceURL
			}
			h.publish(ctx, st)
			return nil
		}
	}

	st := h.status
	st.State = spec.MCPAuthStateAuthorized
	st.LastError = ""
	if st.Resource == "" && resourceURL != "" {
		st.Resource = resourceURL
	}
	if tokenStatus, ok := h.currentTokenStatus(ctx); ok {
		if tokenStatus.Resource == "" && resourceURL != "" {
			tokenStatus.Resource = resourceURL
		}
		st = tokenStatus
	}
	h.publish(ctx, st)
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
	return authStatusFromToken(h.status, tok), true
}

func (h *trackedOAuthHandler) publish(ctx context.Context, st spec.MCPAuthStatus) {
	if h == nil || h.sink == nil {
		return
	}

	_ = h.sink.SaveAuthStatus(context.WithoutCancel(ctx), st)
}

func authStatusFromToken(base spec.MCPAuthStatus, tok *oauth2.Token) spec.MCPAuthStatus {
	st := base
	st.State = spec.MCPAuthStateAuthorized
	st.LastError = ""

	if tok == nil {
		return st
	}

	if !tok.Expiry.IsZero() {
		expiresAt := tok.Expiry.UTC()
		st.ExpiresAt = &expiresAt
	}
	if scopes := scopesFromOAuthToken(tok); len(scopes) > 0 {
		st.Scopes = scopes
	}

	return st
}

func authStatusFromTokenError(base spec.MCPAuthStatus, err error) spec.MCPAuthStatus {
	st := base
	st.State = spec.MCPAuthStateError
	if err != nil {
		st.LastError = err.Error()
	}

	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) && retrieveErr.ErrorCode == "invalid_grant" {
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
	switch {
	case challengeErr == "insufficient_scope":
		st.State = spec.MCPAuthStateInsufficientScope
	case challengeErr == "invalid_token":
		st.State = spec.MCPAuthStateExpired
	case resp.StatusCode == http.StatusUnauthorized:
		st.State = spec.MCPAuthStateRequired
	case resp.StatusCode == http.StatusForbidden:
		st.State = spec.MCPAuthStateError
	}
	return st
}

func drainAndClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
}

func bearerChallengeValues(headers []string) (challengeErr string, scopes []string) {
	challenges, err := oauthex.ParseWWWAuthenticate(headers)
	if err != nil {
		return "", nil
	}
	for _, c := range challenges {
		if c.Scheme != "bearer" {
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
